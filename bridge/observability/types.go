package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
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
	WaitGroup  *sync.WaitGroup
}

type observability struct {
	channel      Channel
	mqttReceived prometheus.Counter
	mqttErrors   prometheus.Counter
	kafkaSent    prometheus.Counter
	kafkaErrors  prometheus.Counter
	kafkaState   prometheus.Gauge
	logger       *log.Entry
	ready        bool
	healthPort   int
	waitGroup    *sync.WaitGroup
	srv          *http.Server
	promReg      *prometheus.Registry
}
