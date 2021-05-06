package mqtt

import (
	"context"
	"crypto/tls"
	"fmt"
	paho "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

type MqttParams struct {
	Broker       string
	Port         int
	Clientid     string
	Tls          bool
	LostCallback func()
	TlsConfig    *tls.Config
	Ctx          context.Context
}

type MqttClient struct {
	client       paho.Client
	tlsConfig    *tls.Config
	broker       string
	port         int
	clientId     string
	lostCallBack func()
	tls          bool
	ch           MessageChannel
	ctx          context.Context
}

type MqttMessage struct {
	topic   string
	content []byte
}

type MessageChannel chan MqttMessage

// handleConnect
// Connects to the broker. Blocks until the connection is established.
func (c *MqttClient) handleConnect() {
	log.Debug("Worker starting connect to broker.")
	token := c.client.Connect()
	if token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	log.Tracef("Router %v connected", c.clientId)
}

func Run(p MqttParams) {
	log.Debugf("Starting MQTT Worker.")
	log.Debugf("Broker: %s:%d (tls: %v)", p.Broker, p.Port, p.Tls)
	c := MqttClient{
		client:       nil,
		broker:       p.Broker,
		port:         p.Port,
		clientId:     "metamorphosis", // Todo: Figure out a proper ID
		lostCallBack: nil,
		tls:          p.Tls,
		ch:           make(MessageChannel), // Todo: Consider buffered channel here.
	}
	opts := paho.NewClientOptions()
	if p.Tls {
		opts.SetTLSConfig(p.TlsConfig)
		opts.AddBroker(fmt.Sprintf("ssl://%s:%d", p.Broker, p.Port))
		c.tls = true
	} else {
		opts.AddBroker(fmt.Sprintf("mqtt://%s:%d", p.Broker, p.Port))
	}
	opts.SetClientID(c.clientId)
	client := paho.NewClient(opts)
	c.client = client

}
