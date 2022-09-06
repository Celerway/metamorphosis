package kafka

import (
	"github.com/celerway/metamorphosis/bridge/observability"
	gokafka "github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"time"
)

type buffer struct {
	C                    MessageChan       // channel for new messages to be written
	buffer               []gokafka.Message // This is where we store the messages
	lastSendAttempt      time.Time
	failureState         bool
	failureRetryInterval time.Duration
	interval             time.Duration
	writer               KafkaWriter
	maxBatchSize         int
	batchSize            int
	topic                string
	kafkaTimeout         time.Duration
	failures             int
	logger               *log.Entry
	obsChannel           observability.Channel
	testMessageTopic     string
}

type Message struct {
	Topic   string `json:"topic"`
	Content []byte `json:"content"`
}

type MessageChan chan Message

type Params struct {
	Broker           string
	Port             int
	Channel          MessageChan
	BatchSize        int
	MaxBatchSize     int
	Interval         time.Duration
	Topic            string
	ObsChannel       observability.Channel
	RetryInterval    time.Duration
	TestMessageTopic string
}
