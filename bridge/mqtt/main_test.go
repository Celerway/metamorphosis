package mqtt

import (
	"bytes"
	"context"
	"fmt"
	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/celerway/metamorphosis/bridge/observability"
	is2 "github.com/matryer/is"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

var toxiClient *toxiproxy.Client

func TestMain(m *testing.M) {
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{})
	// Setup test environment, mosquitto server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		runMosquitto(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		runToxy(ctx)
	}()
	time.Sleep(time.Second)
	// setup toxiproxy

	upstreamService := "localhost:1883"
	listen := "localhost:1884"

	toxiClient = toxiproxy.NewClient("http://localhost:8474")
	proxy, err := toxiClient.CreateProxy("mqtt", listen, upstreamService)
	if err != nil {
		log.Fatal("creating Toxi Proxy: ", err)
	}
	defer proxy.Delete()

	ret := m.Run()
	cancel()
	wg.Wait()
	os.Exit(ret)
}

func Test_Simple(t *testing.T) {
	is := is2.New(t)
	is.True(true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := sync.WaitGroup{}
	ch := make(MessageChannel, 100)
	params := getTestParams(ch)
	wg.Add(1)
	go func() {
		defer wg.Done()
		Run(ctx, params)
	}()
	time.Sleep(time.Second * 1) // give mosquitto time to start and the client time to connect.
	err := injectMessage("testTopic", "testMessage")
	is.NoErr(err)
	select {
	case msg := <-ch:
		is.Equal(msg.Topic, "testTopic")
	case <-time.After(time.Second):
		is.Fail() // Didn't get a message
	}
	cancel()
	time.Sleep(time.Second * 1) // give the client time to disconnect
	wg.Wait()
}

func Test_Reset(t *testing.T) {
	is := is2.New(t)
	is.True(true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := sync.WaitGroup{}
	ch := make(MessageChannel, 100)
	params := getTestParams(ch)
	wg.Add(1)
	go func() {
		defer wg.Done()
		Run(ctx, params)
	}()
	time.Sleep(time.Second * 1) // give mosquitto time to start and the client time to connect.
	err := injectMessage("testTopic", "testMessage")
	is.NoErr(err)
	select {
	case msg := <-ch:
		is.Equal(msg.Topic, "testTopic")
	case <-time.After(time.Second):
		is.Fail() // Didn't get a message
	}
	proxy, err := toxiClient.Proxy("mqtt")
	is.NoErr(err)
	// Add a reset_peer toxic. This will reset the connection to the broker.
	_, err = proxy.AddToxic("reset", "reset_peer", "", 1, toxiproxy.Attributes{})
	// Send a message so toxiproxy can reset the connection
	err = injectMessage("testTopic", "testMessage")
	//is.NoErr(err)
	time.Sleep(time.Second)
	err = proxy.RemoveToxic("reset")
	is.NoErr(err)
	time.Sleep(time.Second)
	err = injectMessage("testTopic", "testMessage")
	is.NoErr(err)
	select {
	case msg := <-ch:
		is.Equal(msg.Topic, "testTopic")
	case <-time.After(time.Second):
		is.Fail() // Didn't get a message
	}

	cancel()
	time.Sleep(time.Second * 1) // give the client time to disconnect
	wg.Wait()
}

// runMosquitto runs the mosquitto server. Blocks until the context is cancelled
func runMosquitto(ctx context.Context) {
	cmd := exec.CommandContext(ctx, "mosquitto")
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	buffer := bytes.NewBuffer(make([]byte, 0, 10000))
	cmd.Stdout = buffer
	cmd.Stderr = buffer
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Error running mosquitto: %v", err)
	}
	<-ctx.Done()
	// context cancelled. Kill the process
	err = cmd.Process.Signal(os.Interrupt)
	if err != nil {
		log.Errorf("Error killing mosquitto: %v", err)
	}
	_ = cmd.Wait()
	_ = cmd.Process.Kill()
	fmt.Println("==== mosquitto stopped ====")
	fmt.Println(buffer.String())
}

func runToxy(ctx context.Context) {
	cmd := exec.CommandContext(ctx, "toxiproxy-server")
	buffer := bytes.NewBuffer(make([]byte, 0, 10000))
	cmd.Stdout = buffer
	cmd.Stderr = buffer
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Error running toxiproxy: %v", err)
	}
	<-ctx.Done()
	// context cancelled. Kill the process
	err = cmd.Process.Signal(os.Interrupt)
	if err != nil {
		log.Errorf("Error killing mosquitto: %v", err)
	}
	_ = cmd.Wait()
	_ = cmd.Process.Kill()
	fmt.Println("==== toxiproxy stopped ====")
	fmt.Println(buffer.String())

}

func injectMessage(topic, message string) error {
	cmd := exec.Command("mosquitto_pub", "-h", "localhost", "-p", "1883", "-t", topic, "-m", message)
	err := cmd.Run()
	return err
}

func getTestParams(ch MessageChannel) Params {
	p := Params{
		Broker:     "localhost",
		Port:       1884,
		Clientid:   "testClient",
		Tls:        false,
		TlsConfig:  nil,
		Channel:    ch,
		Topic:      "testTopic",
		ObsChannel: make(observability.Channel, 100),
	}
	return p
}
