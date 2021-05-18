package kafka

import (
	"context"
	"fmt"
	"github.com/celerway/metamorphosis/bridge/observability"
	gokafka "github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type KafkaMessage struct {
	Topic   string
	Content []byte
}

func (msg KafkaMessage) String() string {
	return fmt.Sprintf("Topic: %s, payload %s", msg.Topic, string(msg.Content))
}

type MessageChannel chan KafkaMessage

// Startup parameters for the Kafka subsystem
type KafkaParams struct {
	Broker        string
	Port          int
	Channel       MessageChannel
	WaitGroup     *sync.WaitGroup
	Topic         string
	ObsChannel    observability.ObservabilityChannel
	RetryInterval time.Duration
}

// Internal state struct for the Kafka subsystem
type kafkaClient struct {
	broker        string
	port          int
	ch            MessageChannel
	waitGroup     *sync.WaitGroup
	topic         string
	writer        *gokafka.Writer
	obsChannel    observability.ObservabilityChannel
	writeHandler  func(ctx context.Context, client kafkaClient, msg KafkaMessage) bool
	logger        *log.Entry
	retryInterval time.Duration
}
