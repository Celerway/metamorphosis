package bridge

import (
	"github.com/celerway/metamorphosis/bridge/kafka"
	"github.com/celerway/metamorphosis/bridge/mqtt"
)

type BridgeParams struct {
	MqttBroker     string
	Tls            bool
	MqttPort       int
	TlsRootCrtFile string
	ClientCertFile string
	ClientKeyFile  string
	MqttTopic      string
	KafkaBroker    string
	KafkaPort      int
	KafkaTopic     string
}

type bridge struct {
	mqttCh  mqtt.MessageChannel
	kafkaCh kafka.MessageChannel
}
