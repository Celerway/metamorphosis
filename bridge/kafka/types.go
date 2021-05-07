package kafka

import (
	gokafka "github.com/segmentio/kafka-go"
	"sync"
)

type KafkaChannelMessage struct {
	Topic   string
	Content []byte
}

type MessageChannel chan KafkaChannelMessage

type KafkaParams struct {
	Broker    string
	Port      int
	Channel   MessageChannel
	WaitGroup *sync.WaitGroup
	Topic     string
}

type kafkaClient struct {
	broker    string
	port      int
	ch        MessageChannel
	waitGroup *sync.WaitGroup
	topic     string
	writer    *gokafka.Writer
}
