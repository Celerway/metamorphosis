package bridge

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/celerway/metamorphosis/bridge/mqtt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
)

// import "github.com/celerway/metamorphosis/bridge/mqtt"

type BridgeParams struct {
	MqttBroker     string
	Tls            bool
	MqttPort       int
	TlsRootCrtFile string
	ClientCertFile string
	CLientKeyFile  string
}

type Bridge struct {
	MqttCh mqtt.MessageChannel
}

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
	tlsConfig := NewTlsConfig(params.TlsRootCrtFile, params.ClientCertFile, params.CLientKeyFile)

	log.WithFields(log.Fields{
		"module": "bridge",
	})

	rootCtx := context.Background()
	ctx, cancel := context.WithCancel(rootCtx)

	mqttParams := mqtt.MqttParams{
		Ctx:       ctx,
		TlsConfig: tlsConfig,
		Broker:    params.MqttBroker,
		Port:      params.MqttPort,
		Tls:       params.Tls,
	}

	mqtt.Run(mqttParams)
	cancel()
}
