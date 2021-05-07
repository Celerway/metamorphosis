package mqtt

import (
	"crypto/tls"
	"github.com/celerway/metamorphosis/bridge/observability"
	paho "github.com/eclipse/paho.mqtt.golang"
	"sync"
)

type MqttParams struct {
	Broker     string
	Port       int
	Clientid   string
	Tls        bool
	TlsConfig  *tls.Config
	Channel    MessageChannel
	WaitGroup  *sync.WaitGroup
	Topic      string
	ObsChannel observability.ObservabilityChannel
}

type MqttChannelMessage struct {
	Topic   string
	Content []byte
}

type MessageChannel chan MqttChannelMessage

type mqttClient struct {
	paho       paho.Client
	tlsConfig  *tls.Config
	broker     string
	port       int
	clientId   string
	tls        bool
	ch         MessageChannel
	waitGroup  *sync.WaitGroup
	topic      string
	obsChannel observability.ObservabilityChannel
}
