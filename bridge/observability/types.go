package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"net/http"
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
	Channel    ObservabilityChannel
	HealthPort int
	WaitGroup  *sync.WaitGroup
}

type observability struct {
	channel      ObservabilityChannel
	mqttReceived prometheus.Counter
	mqttErrors   prometheus.Counter
	kafkaSent    prometheus.Counter
	kafkaErrors  prometheus.Counter
	logger       *log.Entry
	ready        bool
	healthPort   int
	waitGroup    *sync.WaitGroup
	srv          *http.Server
	promReg      *prometheus.Registry
}
