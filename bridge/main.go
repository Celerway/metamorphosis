package bridge

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
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

func Run(params BridgeParams) {
	var wg sync.WaitGroup

	// In order to avoid hanging when we shut down we shutdown things in a certain order. So we use two contexts
	// to do this.
	rootCtx := context.Background()
	mqttCtx, mqttCancel := context.WithCancel(rootCtx)   // Mqtt client. Shutdown first.
	kafkaCtx, kafkaCancel := context.WithCancel(rootCtx) // Kafka, shutdown after mqtt.

	obsChan := observability.GetChannel()

	br := bridge{
		mqttCh:  make(mqtt.MessageChannel, 0),  // For pure performance these should be buffered
		kafkaCh: make(kafka.MessageChannel, 0), // However, this could hide potential dead locks.
		logger:  log.WithFields(log.Fields{"module": "bridge"}),
	}
	tlsConfig := NewTlsConfig(params.TlsRootCrtFile, params.MqttClientCertFile, params.MqttClientKeyFile, br.logger)

	mqttParams := mqtt.MqttParams{
		TlsConfig:  tlsConfig,
		Broker:     params.MqttBroker,
		Port:       params.MqttPort,
		Topic:      params.MqttTopic,
		Tls:        params.MqttTls,
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
		Channel: obsChan,
	}
	// Start the goroutines that do the work.
	mqtt.Run(mqttCtx, mqttParams)
	kafka.Run(kafkaCtx, kafkaParams)
	br.run()
	observability.Run(obsParams)

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// Spin off a goroutine that will wait for SIGNALs and cancel the context.
	// If we wanna do something on a regular basis (log stats or whatnot)
	// this is a good place.
	go func() {
		wg.Add(1)
		br.logger.Debug("Signal listening goroutine is running.")
		select {
		case <-sigChan:
			br.logger.Warn("Cancelled context. Initiating shutdown.")
			mqttCancel()
			time.Sleep(5 * time.Second) // This should be enough to make sure Kafka is flushed out.
			kafkaCancel()
			wg.Done()
			return
		}
	}()
	br.logger.Trace("Main goroutine waiting for bridge shutdown.")
	wg.Wait()
	br.logger.Infof("Program exiting. There are currently %d goroutines: ", runtime.NumGoroutine())
}

func NewTlsConfig(caFile, clientCertFile, clientKeyFile string, logger *log.Entry) *tls.Config {
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(caFile)
	if err != nil {
		log.Fatalln(err.Error())
	}
	certPool.AppendCertsFromPEM(ca)
	// Import client certificate/key pair
	clientKeyPair, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	if err != nil {
		logger.Fatalf("tls.LoadX509KeyPair(%s,%s): %s", clientCertFile, clientKeyFile, err)
		panic(err)
	}
	logger.Debugf("Initialized TLS Client config with CA (%s) Client cert/key (%s/%s)",
		caFile, clientCertFile, clientKeyFile)
	return &tls.Config{
		RootCAs:            certPool,
		ClientAuth:         tls.NoClientCert,
		ClientCAs:          nil,
		InsecureSkipVerify: false,
		Certificates:       []tls.Certificate{clientKeyPair},
	}
}

func (br BridgeParams) String() string {
	jsonBytes, _ := json.MarshalIndent(br, "", "  ")
	return string(jsonBytes)
}
