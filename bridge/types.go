package bridge

import "github.com/celerway/metamorphosis/bridge/mqtt"

type BridgeParams struct {
	MqttBroker     string
	Tls            bool
	MqttPort       int
	TlsRootCrtFile string
	ClientCertFile string
	CLientKeyFile  string
	MqttTopic      string
}

type Bridge struct {
	MqttCh mqtt.MessageChannel
}
