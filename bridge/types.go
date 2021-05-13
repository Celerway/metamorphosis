package bridge

import (
	"github.com/celerway/metamorphosis/bridge/kafka"
	"github.com/celerway/metamorphosis/bridge/mqtt"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
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
	KafkaRetryInterval time.Duration
	MainWaitGroup      *sync.WaitGroup
}

type bridge struct {
	mqttCh  mqtt.MessageChannel
	kafkaCh kafka.MessageChannel
	logger  *log.Entry
}
