package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/celerway/metamorphosis/bridge"
	proxy "github.com/celerway/metamorphosis/integration/proxy"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	gokafka "github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

const noOfMessages = 500

type Message struct {
	Id           int
	RandomString string
	Checksum     [32]byte
}

type KMessage struct {
	Topic   string
	Content []byte
}

const startServices = false

func TestMain(m *testing.M) {
	// Main setup goroutine
	var (
		rootCtx, mqttCtx, kafkaCtx context.Context
		mqttCancel, kafkaCancel    context.CancelFunc
	)
	log.SetLevel(log.DebugLevel)
	log.Debug("Set log level")
	if startServices {
		rootCtx = context.Background()
		fmt.Println("Starting MQTT")
		mqttCtx, mqttCancel = context.WithCancel(rootCtx)
		startMqtt(mqttCtx)
		fmt.Println("Starting Kafka")
		kafkaCtx, kafkaCancel = context.WithCancel(rootCtx)
		startKafka(kafkaCtx)
	}
	rand.Seed(time.Now().Unix())
	ret := m.Run() // Run the tests.
	if startServices {
		fmt.Println("Stopping MQTT")
		mqttCancel()
		fmt.Println("Stopping Kafka")
		kafkaCancel()
		os.Exit(ret)
	}
}

func TestDummy(t *testing.T) {
	fmt.Println("Dummy test ok")
}

func TestBasic(t *testing.T) {
	logger := log.WithFields(log.Fields{"test": "TestBasic"})
	fmt.Println("Testing basic stuff")
	rootCtx := context.Background()
	proxyCtx, proxyCancel := context.WithCancel(rootCtx)
	mqttPort, kafkaPort, healthPort := getRandomPorts()
	rTopic := getRandomString(12)
	kafkaTopic(t, "create", rTopic)
	go proxy.StartProxy(proxyCtx, mqttPort, kafkaPort, logger)
	runConfig := bridge.BridgeParams{
		MqttBroker:   "localhost",
		MqttPort:     mqttPort,
		MqttTopic:    rTopic,
		MqttTls:      false,
		KafkaBroker:  "localhost",
		KafkaPort:    kafkaPort,
		KafkaTopic:   rTopic,
		KafkaWorkers: 1,
		HealthPort:   healthPort,
	}
	bridgeCtx, bridgeCancel := context.WithCancel(rootCtx)
	logger.Debug("Starting bridge")
	go bridge.Run(bridgeCtx, runConfig)

	waitForBridge(t, healthPort)
	log.Debug("Publishing messages on MQTT")
	publishMqttMessages(t, rTopic, noOfMessages, mqttPort) // Publish 100 messages
	log.Debug("Verifying messages")
	verifyKafkaMessages(t, rTopic, noOfMessages, kafkaPort) // Verify the messages.
	bridgeCancel()
	proxyCancel()
	kafkaTopic(t, "delete", rTopic)
	/*
	   	plan:
	         - create a topic in Kafka (random name)
	         - spin up the proxy (with two random ports)
	         - spin up bridge (remember to connect to the ports)
	         - push 100 messages through the MQTT
	         - see that they arrive in Kafka
	      	  - verify obs data
	*/
}

func waitForBridge(t *testing.T, port int) {
	url := fmt.Sprintf("http://localhost:%d/healthz", port)
	log.Debugf("Waiting for bridge to come up on %s", url)

	bridgeOk := false
	for !bridgeOk {
		resp, err := http.Get(url)
		log.Tracef("Checked %s --> %s", url, resp.Status)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if resp.StatusCode == 200 {
			bridgeOk = true
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Debug("Bridge OK.")
}

func TestKafkaFailure(t *testing.T) {
	fmt.Println("Testing kafka failure")
	/*
	   	plan:
	         - create a topic in Kafka (random name)
	         - spin up the proxy (with two random ports)
	         - spin up bridge (remember to connect to the ports)
	         - push 100 messages through the MQTT
	         - SIGSTOP the proxy
	         - push 100 messages through the MQTT broker
	         - wait a while (11 seconds)
	         - SIGCONT the proxy
	         - wait another while (10 seconds)
	         - see that they have all arrive in Kafka
	         - verify obs data

	*/

}

func TestMqttFailure(t *testing.T) {
	fmt.Println("Testing kafka failure")
	/*
		plan:
			 - create a topic in Kafka (random name)
			 - spin up the proxy (with two random ports)
			 - spin up bridge (remember to connect to the ports)
			 - push 100 messages through the MQTT
			 - SIGINT the proxy so it shuts down
			 - spin up another proxy on the same ports
			 - push 100 messages through the MQTT broker
			 - see that they have all arrive in Kafka
			 - verify obs data
	*/
}

func startMqtt(ctx context.Context) {
	go func() {
		err := exec.CommandContext(ctx, "/usr/sbin/mosquitto").Run()
		if err != nil {
			fmt.Println("Error starting MQTT", err)
		} else {
			fmt.Println("MQTT started")
		}
	}()
}

func startKafka(ctx context.Context) {
	go func() {
		err := exec.CommandContext(ctx, "/usr/bin/rpk", "redpanda", "start")
		if err != nil {
			fmt.Println("Error starting MQTT", err)
		} else {
			fmt.Println("Kafka started")
		}
	}()
}

// Get a set of random ports for the proxy
func getRandomPorts() (int, int, int) {
	return rand.Intn(63000) + 1024, rand.Intn(63000) + 1000, rand.Intn(63000) + 1000
}

func getRandomString(length int) string {
	var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func kafkaTopic(t *testing.T, action, topic string) {
	cmd := exec.Command("rpk", "topic", action, topic)
	var stdout, stderr bytes.Buffer
	log.Debugf("kafka topic(%s): %s", action, topic)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Errorf("Kafka topic (%s) stdout: %s stderr: %s err: %s", action, stdout.String(), stderr.String(), err)
	}
}

func getMqttClient(port int) mqtt.Client {
	var broker = "localhost"
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetClientID("go_mqtt_client")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	return client
}

func makeMessage(id int) ([]byte, error) {
	rString := getRandomString(200)
	checksum := sha256.Sum256([]byte(rString))
	msg := Message{
		Id:           id,
		RandomString: rString,
		Checksum:     checksum,
	}
	return json.Marshal(msg)
}

// takes a raw message we got from Kafka and
// verify the content.
// fuck this. this is a bit on the complex side.
func verifyMessage(i int, m gokafka.Message, topic string) error {
	var kmsg KMessage // Kafka message.
	var actualMessage Message
	if m.Topic != topic {
		panic("Invalid topic")
	}
	// unmarshal the message from Kafka.
	err := json.Unmarshal(m.Value, &kmsg)
	if err != nil {
		return fmt.Errorf("kafka message json unmarshal: %s", err)
	}
	// get the mqtt message. It is now in kmsg.Content
	err = json.Unmarshal(kmsg.Content, &actualMessage)
	if err != nil {
		return fmt.Errorf("mqtt message json unmarshal: %s", err)
	}

	// Now check the inner message for content.
	if actualMessage.Id != i {
		return fmt.Errorf("kafka message has wrong ID, got %d, expected %d", actualMessage.Id, i)
	}
	msgChecksum := sha256.Sum256([]byte(actualMessage.RandomString))

	if msgChecksum != actualMessage.Checksum {
		return fmt.Errorf("Message checksum mismatch, got %s expected %s", msgChecksum, actualMessage.Checksum)
	}
	return nil
}

func publishMqttMessages(t *testing.T, topic string, noMessages, port int) {
	client := getMqttClient(port)
	for i := 0; i < noMessages; i++ {
		msg, err := makeMessage(i)
		if err != nil {
			t.Errorf("While making message: %s", err)
		}
		client.Publish(topic, 1, false, msg)
		log.Tracef("Published message %d on MQTT", i)
	}
}

func verifyKafkaMessages(t *testing.T, topic string, noMessages, port int) {
	client := getKafkaReader(port, topic)
	for i := 0; i < noMessages; i++ {
		log.Debugf("kafka: reading message %d", i)
		msg, err := client.ReadMessage(context.Background())
		if err != nil {
			t.Errorf("Reading kafka message: %s", err)
		}
		err = verifyMessage(i, msg, topic)
		if err != nil {
			t.Errorf("Verifying kafka message: %s", err)
		}
	}
}

func getKafkaReader(port int, topic string) *gokafka.Reader {
	broker := fmt.Sprintf("localhost:%d", port)
	r := gokafka.NewReader(gokafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   topic,
	})
	// Skip the two first messages. The first is a null message. The next is the one the bridge issues.
	err := r.SetOffset(2)
	if err != nil {
		log.Fatalf("Setting offset: %s", err)
	}
	return r
}
