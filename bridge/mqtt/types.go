package mqtt

import (
	"context"
	"crypto/tls"
	paho "github.com/eclipse/paho.mqtt.golang"
	"sync"
)

type MqttParams struct {
	Broker    string
	Port      int
	Clientid  string
	Tls       bool
	TlsConfig *tls.Config
	Ctx       context.Context
	Channel   MessageChannel
	WaitGroup *sync.WaitGroup
	Topic     string
}

type mqttClient struct {
	client    paho.Client
	tlsConfig *tls.Config
	broker    string
	port      int
	clientId  string
	tls       bool
	ch        MessageChannel
	ctx       context.Context
	waitGroup *sync.WaitGroup
	topic     string
}

type MqttMessage struct {
	topic   string
	content []byte
}

type MessageChannel chan MqttMessage
