package bridge

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/celerway/metamorphosis/bridge/kafka"
	"github.com/celerway/metamorphosis/bridge/mqtt"
	"github.com/celerway/metamorphosis/bridge/observability"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// import "github.com/celerway/metamorphosis/bridge/mqtt"

func NewTlsConfig(caFile, clientCertFile, clientKeyFile string) *tls.Config {
	certpool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(caFile)
	if err != nil {
		log.Fatalln(err.Error())
	}
	certpool.AppendCertsFromPEM(ca)
	// Import client certificate/key pair
	clientKeyPair, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	if err != nil {
		log.Fatalf("tls.LoadX509KeyPair(%s,%s): %s", clientCertFile, clientKeyFile, err)
		panic(err)
	}
	log.Debugf("Initialized TLS Client config with CA (%s) Client cert/key (%s/%s)",
		caFile, clientCertFile, clientKeyFile)
	return &tls.Config{
		RootCAs:            certpool,
		ClientAuth:         tls.NoClientCert,
		ClientCAs:          nil,
		InsecureSkipVerify: false,
		Certificates:       []tls.Certificate{clientKeyPair},
	}
}

func Run(params BridgeParams) {
	var wg sync.WaitGroup

	tlsConfig := NewTlsConfig(params.TlsRootCrtFile, params.ClientCertFile, params.ClientKeyFile)

	// In order to avoid hanging when we shut down we shutdown things in a certain order. So we use multiple contexts
	// to do this.
	rootCtx := context.Background()
	mqttCtx, mqttCancel := context.WithCancel(rootCtx)   // Mqtt client. Shutdown first.
	kafkaCtx, kafkaCancel := context.WithCancel(rootCtx) // Kafka, shutdown after mqtt.
	mainCtx, mainCancel := context.WithCancel(rootCtx)   // Bridge and obs. Shutdown last.

	obsChan := observability.GetChannel()

	br := bridge{
		mqttCh:    make(mqtt.MessageChannel, 0),
		kafkaCh:   make(kafka.MessageChannel, 0),
		waitGroup: &wg,
	}

	mqttParams := mqtt.MqttParams{
		TlsConfig:  tlsConfig,
		Broker:     params.MqttBroker,
		Port:       params.MqttPort,
		Topic:      params.MqttTopic,
		Tls:        params.Tls,
		Channel:    br.mqttCh,
		WaitGroup:  &wg,
		ObsChannel: obsChan,
	}
	kafkaParams := kafka.KafkaParams{
		Broker:     params.KafkaBroker,
		Port:       params.KafkaPort,
		Channel:    br.kafkaCh,
		WaitGroup:  &wg,
		Topic:      params.KafkaTopic,
		ObsChannel: obsChan,
	}
	obsParams := observability.ObservabilityParams{
		Channel:   obsChan,
		Waitgroup: &wg,
	}
	// Start the goroutines that do the work.
	mqtt.Run(mqttCtx, mqttParams)
	kafka.Run(kafkaCtx, kafkaParams)
	br.run(mainCtx)
	observability.Run(mainCtx, obsParams)

	log.Debug("MQTT receiver running")

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// Spin off a goroutine that will wait for SIGNALs and cancel the context.
	// If we wanna do something on a regular basis (log stats or whatnot)
	// this is a good place.
	go func() {
		wg.Add(1)
		log.Debug("Signal listening goroutine is running.")
		select {
		case <-sigChan:
			log.Warn("Cancelled context. Initiating shutdown.")
			mqttCancel()
			time.Sleep(1 * time.Second)
			kafkaCancel()
			time.Sleep(1 * time.Second)
			mainCancel()
			wg.Done()
			return
		}
	}()
	log.Trace("Main goroutine waiting for bridge shutdown.")
	wg.Wait()
	log.Infof("Program exiting. There are currently %d goroutines: ", runtime.NumGoroutine())
	mainCancel()
}
