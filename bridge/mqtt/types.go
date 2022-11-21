package mqtt

import (
	"crypto/tls"
	"github.com/celerway/metamorphosis/bridge/observability"
	paho "github.com/eclipse/paho.mqtt.golang"
	logrus "github.com/sirupsen/logrus"
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
	logger     *logrus.Entry
}
