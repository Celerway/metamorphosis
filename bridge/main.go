package bridge

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/celerway/metamorphosis/bridge/kafka"
	"github.com/celerway/metamorphosis/bridge/mqtt"
	"github.com/celerway/metamorphosis/bridge/observability"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const channelSize = 100

func Run(ctx context.Context, params BridgeParams) {
	params.MainWaitGroup.Add(1) // allows the caller to wait for clean exit.
	var wg sync.WaitGroup       // wg for children.
	var tlsConfig *tls.Config
	// In order to avoid hanging when we shut down we shutdown things in a certain order. So we use two contexts
	// to do this.
	mqttCtx, mqttCancel := context.WithCancel(ctx)   // Mqtt client. Shutdown first.
	kafkaCtx, kafkaCancel := context.WithCancel(ctx) // Kafka, shutdown after mqtt.

	obsChan := observability.GetChannel(channelSize)

	br := bridge{
		mqttCh:  make(mqtt.MessageChannel, channelSize),
		kafkaCh: make(kafka.MessageChannel, channelSize),
		logger:  log.WithFields(log.Fields{"module": "bridge"}),
	}
	if params.MqttTls {
		tlsConfig = NewTlsConfig(params.TlsRootCrtFile, params.MqttClientCertFile, params.MqttClientKeyFile, br.logger)
	}
	mqttParams := mqtt.MqttParams{
		TlsConfig:  tlsConfig,
		Broker:     params.MqttBroker,
		Port:       params.MqttPort,
		Topic:      params.MqttTopic,
		Tls:        params.MqttTls,
		Clientid:   getMqttClientId(),
		Channel:    br.mqttCh,
		WaitGroup:  &wg,
		ObsChannel: obsChan,
	}
	kafkaParams := kafka.KafkaParams{
		Broker:        params.KafkaBroker,
		Port:          params.KafkaPort,
		Channel:       br.kafkaCh,
		WaitGroup:     &wg,
		Topic:         params.KafkaTopic,
		ObsChannel:    obsChan,
		RetryInterval: params.KafkaRetryInterval,
	}
	obsParams := observability.ObservabilityParams{
		Channel:    obsChan,
		HealthPort: params.HealthPort,
		WaitGroup:  &wg,
	}
	// Start the goroutines that do the work.
	obs := observability.Run(obsParams) // Fire up obs.
	br.run()                            // Start the bridge so MQTT can send messages to Kafka.
	for i := 1; i < params.KafkaWorkers+1; i++ {
		kafka.Run(kafkaCtx, kafkaParams, i) // start the writer(s).
	}
	mqtt.Run(mqttCtx, mqttParams) // Then connect to MQTT
	obs.Ready()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// Spin off a goroutine that will wait for SIGNALs and cancel the context.
	// If we wanna do something on a regular basis (log stats or whatnot)
	// this is a good place.
	go func() {
		wg.Add(1)
		br.logger.Debug("Signal listening goroutine is running.")
		select {
		case <-ctx.Done():
			br.logger.Warn("Context cancelled. Initiating shutdown.")
			mqttCancel()
			time.Sleep(1 * time.Second) // This should be enough to make sure Kafka is flushed out.
			kafkaCancel()
			obs.Shutdown()
			wg.Done()
			return
		case <-sigChan:
			br.logger.Warn("Signal caught. Initiating shutdown.")
			mqttCancel()
			time.Sleep(1 * time.Second) // This should be enough to make sure Kafka is flushed out.
			kafkaCancel()
			obs.Shutdown()
			wg.Done()
			return
		}
	}()
	br.logger.Trace("Main goroutine waiting for bridge shutdown.")
	wg.Wait()
	br.logger.Infof("Bridge exiting")
	params.MainWaitGroup.Done() // main group
}

func getMqttClientId() string {
	return fmt.Sprintf("metamorphosis-%d", os.Getpid())
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
