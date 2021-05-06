package mqtt

import (
	"context"
	"crypto/tls"
	"fmt"
	paho "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"sync"
)

type MqttParams struct {
	Broker       string
	Port         int
	Clientid     string
	Tls          bool
	LostCallback func()
	TlsConfig    *tls.Config
	Ctx          context.Context
	Channel      MessageChannel
	WaitGroup    *sync.WaitGroup
}

type mqttClient struct {
	client       paho.Client
	tlsConfig    *tls.Config
	broker       string
	port         int
	clientId     string
	lostCallBack func()
	tls          bool
	ch           MessageChannel
	ctx          context.Context
	waitGroup    *sync.WaitGroup
}

type MqttMessage struct {
	topic   string
	content []byte
}

type MessageChannel chan MqttMessage

// handleConnect
// Connects to the broker. Blocks until the connection is established.
func (c *mqttClient) handleConnect() {
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
	c := mqttClient{
		broker:    p.Broker,
		port:      p.Port,
		clientId:  "metamorphosis", // Todo: Figure out a proper ID
		tls:       p.Tls,
		ch:        p.Channel,
		waitGroup: p.WaitGroup,
		ctx:       p.Ctx,
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
	log.Debug("Spinning off the MQTT worker")
	c.waitGroup.Add(1)
	go mainloop(c)
}

func mainloop(client mqttClient) {
	log.Debug("In mqtt main loop")
	keepRunning := true
	for keepRunning {
		select {
		case <-client.ctx.Done():
			log.Debug("MQTT bridge shutting down.")
			keepRunning = false
		}
	}
	log.Debug("Finishing")
	client.waitGroup.Done()
}
