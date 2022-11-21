package observability

import (
	"github.com/celerway/metamorphosis/log"
	"github.com/prometheus/client_golang/prometheus"
)

type Channel chan StatusMessage

type StatusMessage int

const (
	MattReceived StatusMessage = iota
	MqttError
	KafkaSent
	KafkaError
)

func (d StatusMessage) String() string {
	return [...]string{"MattReceived", "MqttError", "KafkaSent", "KafkaError"}[d]
}

type Params struct {
	Channel    Channel
	HealthPort int
}

type observability struct {
	channel      Channel
	mqttReceived prometheus.Counter
	mqttErrors   prometheus.Counter
	kafkaSent    prometheus.Counter
	kafkaErrors  prometheus.Counter
	kafkaState   prometheus.Gauge
	logger       *log.Logger
	ready        bool
	healthPort   int
	promReg      *prometheus.Registry
}
