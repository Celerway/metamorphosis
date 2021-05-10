package bridge

import (
	"github.com/celerway/metamorphosis/bridge/kafka"
	"github.com/celerway/metamorphosis/bridge/mqtt"
	log "github.com/sirupsen/logrus"
)

type BridgeParams struct {
	MqttBroker         string
	MqttTls            bool
	MqttPort           int
	TlsRootCrtFile     string
	MqttClientCertFile string
	MqttClientKeyFile  string
	MqttTopic          string
	KafkaBroker        string
	KafkaPort          int
	KafkaTopic         string
	KafkaWorkers       int
	HealthPort         int
}

type bridge struct {
	mqttCh  mqtt.MessageChannel
	kafkaCh kafka.MessageChannel
	logger  *log.Entry
}
