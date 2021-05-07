package bridge

import (
	"context"
	"github.com/celerway/metamorphosis/bridge/kafka"
	"github.com/celerway/metamorphosis/bridge/mqtt"
	"sync"
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
	mqttCh    mqtt.MessageChannel
	kafkaCh   kafka.MessageChannel
	ctx       context.Context
	waitGroup *sync.WaitGroup
}
