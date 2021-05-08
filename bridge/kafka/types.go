package kafka

import (
	"context"
	"github.com/celerway/metamorphosis/bridge/observability"
	gokafka "github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"sync"
)

type KafkaMessage struct {
	Topic   string
	Content []byte
}

type MessageChannel chan KafkaMessage

// Startup parameters for the Kafka subsystem
type KafkaParams struct {
	Broker     string
	Port       int
	Channel    MessageChannel
	WaitGroup  *sync.WaitGroup
	Topic      string
	ObsChannel observability.ObservabilityChannel
}

// Internal state struct for the Kafka subsystem
type kafkaClient struct {
	broker       string
	port         int
	ch           MessageChannel
	waitGroup    *sync.WaitGroup
	topic        string
	writer       *gokafka.Writer
	obsChannel   observability.ObservabilityChannel
	writeHandler func(ctx context.Context, msg KafkaMessage, client kafkaClient) bool
	logger       *log.Entry
}
