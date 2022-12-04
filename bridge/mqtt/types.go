package mqtt

import (
	"crypto/tls"
	"github.com/celerway/metamorphosis/bridge/observability"
	"github.com/celerway/metamorphosis/log"
	paho "github.com/eclipse/paho.mqtt.golang"
)

type Params struct {
	Broker     string
	Port       int
	Clientid   string
	Tls        bool
	TlsConfig  *tls.Config
	Channel    MessageChannel
	Topic      string
	ObsChannel observability.Channel
	LogLevel   log.LogLevel
}

type ChannelMessage struct {
	Topic   string
	Content []byte
}

type MessageChannel chan ChannelMessage

type client struct {
	paho       paho.Client
	tlsConfig  *tls.Config
	broker     string
	port       int
	clientId   string
	tls        bool
	ch         MessageChannel
	topic      string
	obsChannel observability.Channel
	logger     *log.Logger
}
