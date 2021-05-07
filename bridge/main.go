package bridge

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/celerway/metamorphosis/bridge/kafka"
	"github.com/celerway/metamorphosis/bridge/mqtt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
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
	rootCtx := context.Background()
	ctx, cancel := context.WithCancel(rootCtx)

	br := bridge{
		mqttCh:    make(mqtt.MessageChannel),
		kafkaCh:   make(kafka.MessageChannel),
		waitGroup: &wg,
	}

	mqttParams := mqtt.MqttParams{
		TlsConfig: tlsConfig,
		Broker:    params.MqttBroker,
		Port:      params.MqttPort,
		Topic:     params.MqttTopic,
		Tls:       params.Tls,
		Channel:   br.mqttCh,
		WaitGroup: &wg,
	}
	kafkaParams := kafka.KafkaParams{
		Broker:    params.KafkaBroker,
		Port:      params.KafkaPort,
		Channel:   br.kafkaCh,
		WaitGroup: &wg,
		Topic:     params.KafkaTopic,
	}
	// Start the goroutines that do the work.
	mqtt.Run(ctx, mqttParams)
	kafka.Run(ctx, kafkaParams)
	br.run(ctx)

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
			cancel()
			log.Warn("Cancel sent to workers. Waiting for workers to exit cleanly")
			wg.Done()
			return
		}
	}()
	log.Trace("Main goroutine waiting for bridge shutdown.")
	wg.Wait()
	log.Infof("Program exiting. There are currently %d goroutines: ", runtime.NumGoroutine())
	cancel()
}
