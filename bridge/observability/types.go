package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"sync"
)

type ObservabilityChannel chan StatusMessage

type StatusMessage int

const (
	MqttRecieved StatusMessage = iota
	MqttError
	KafkaSent
	KafkaError
)

func (d StatusMessage) String() string {
	return [...]string{"MqttRecieved", "MqttError", "KafkaSent", "KafkaError"}[d]
}

type ObservabilityParams struct {
	Channel   ObservabilityChannel
	Waitgroup *sync.WaitGroup
}

type observability struct {
	waitGroup    *sync.WaitGroup
	channel      ObservabilityChannel
	mqttReceived prometheus.Counter
	mqttErrors   prometheus.Counter
	kafkaSent    prometheus.Counter
	kafkaErrors  prometheus.Counter
}
