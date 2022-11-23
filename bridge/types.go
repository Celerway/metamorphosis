package bridge

import (
	"github.com/celerway/metamorphosis/bridge/kafka"
	"github.com/celerway/metamorphosis/bridge/mqtt"
	"github.com/celerway/metamorphosis/log"
	"time"
)

type Params struct {
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
	KafkaRetryInterval time.Duration
	MqttClientId       string
	KafkaBatchSize     int
	KafkaMaxBatchSize  int
	KafkaInterval      time.Duration
	TestMessageTopic   string
	LogLevel           log.LogLevel
}

type bridge struct {
	mqttCh  mqtt.MessageChannel
	kafkaCh kafka.MessageChan
	logger  *log.Logger
}
