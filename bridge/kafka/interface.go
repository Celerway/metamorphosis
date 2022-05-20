package kafka

import (
	"context"
	gokafka "github.com/segmentio/kafka-go"
)

type KafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...gokafka.Message) error
}
