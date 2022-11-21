package bridge

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"github.com/celerway/metamorphosis/bridge/kafka"
	"github.com/celerway/metamorphosis/bridge/mqtt"
	"github.com/celerway/metamorphosis/bridge/observability"
	"github.com/celerway/metamorphosis/log"
	"os"
	"sync"
	"time"
)

const channelSize = 100

func Run(ctx context.Context, params Params) {
	// params.MainWaitGroup.Add(1) // allows the caller to wait for clean exit.
	var wg sync.WaitGroup // wg for children.
	var tlsConfig *tls.Config
	// In order to avoid hanging when we shut down we shutdown things in a certain order. So we use two contexts
	// to do this.
	mqttCtx, mqttCancel := context.WithCancel(context.Background())   // Mqtt client. Cleanup first.
	kafkaCtx, kafkaCancel := context.WithCancel(context.Background()) // Kafka, shutdown after mqtt.
	obsCtx, obsCancel := context.WithCancel(context.Background())     // obs, needs to be shutdown last to avoid deadlocks.
	obsChan := observability.GetChannel(channelSize)
	br := bridge{
		mqttCh:  make(mqtt.MessageChannel, channelSize),
		kafkaCh: make(kafka.MessageChan, channelSize),
		logger:  log.NewWithPrefix(os.Stdout, os.Stderr, "bridge"),
	}
	if params.MqttTls {
		tlsConfig = NewTlsConfig(params.TlsRootCrtFile, params.MqttClientCertFile, params.MqttClientKeyFile, br.logger)
	}
	mqttParams := mqtt.Params{
		TlsConfig:  tlsConfig,
		Broker:     params.MqttBroker,
		Port:       params.MqttPort,
		Topic:      params.MqttTopic,
		Tls:        params.MqttTls,
		Clientid:   params.MqttClientId,
		Channel:    br.mqttCh,
		ObsChannel: obsChan,
	}
	kafkaParams := kafka.Params{
		Broker:           params.KafkaBroker,
		Port:             params.KafkaPort,
		Channel:          br.kafkaCh,
		Topic:            params.KafkaTopic,
		ObsChannel:       obsChan,
		RetryInterval:    params.KafkaRetryInterval,
		Interval:         params.KafkaInterval,
		BatchSize:        params.KafkaBatchSize,
		MaxBatchSize:     params.KafkaMaxBatchSize,
		TestMessageTopic: params.TestMessageTopic,
	}
	obsParams := observability.Params{
		Channel:    obsChan,
		HealthPort: params.HealthPort,
	}
	// Start the goroutines that do the work.
	obs := observability.Initialize(obsParams) // Fire up obs.
	wg.Add(1)
	go func() {
		defer wg.Done()
		obs.Run(obsCtx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		br.mainloop()
	}()
	kafkaWorker := kafka.Initialize(kafkaParams)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := kafkaWorker.Run(kafkaCtx)
		if err != nil {
			log.Fatalf("Could not initialize kafka worker: %s", err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		mqtt.Run(mqttCtx, mqttParams) // Then connect to MQTT
	}()
	obs.Ready()

	// Spin off a goroutine that will wait for SIGNALs and cancel the context.
	// If we wanna do something on a regular basis (log stats or whatnot)
	// this is a good place.

	<-ctx.Done()
	br.logger.Warn("Context cancelled. Initiating shutdown.")
	mqttCancel()
	time.Sleep(time.Second)
	close(br.mqttCh)            // Closing the channel will cause the mainloop to exit.
	time.Sleep(3 * time.Second) // This should be enough to make sure Kafka is flushed out.
	kafkaCancel()
	obsCancel() // shuts down the HTTP server for obs.
	obs.Cleanup()
	wg.Wait() // Block and for us to shut down completely.
	br.logger.Warn("Bridge exiting")
}

func NewTlsConfig(caFile, clientCertFile, clientKeyFile string, logger *log.Logger) *tls.Config {
	certPool := x509.NewCertPool()
	ca, err := os.ReadFile(caFile)
	if err != nil {
		log.Fatal(err.Error())
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

func (br Params) String() string {
	jsonBytes, _ := json.MarshalIndent(br, "", "  ")
	return string(jsonBytes)
}
