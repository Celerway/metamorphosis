package kafka

import (
	"context"
	"fmt"
	gokafka "github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
)

func getInt(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		log.Errorf("getInt failed on %s: %s", s, err)
	}
	return val
}
func getWriter() *gokafka.Writer {

	broker := fmt.Sprintf("%s:%d",
		os.Getenv("KAFKA_BROKER"), getInt(os.Getenv("KAFKA_PORT")))

	w := &gokafka.Writer{
		Addr:     gokafka.TCP(broker),
		Topic:    os.Getenv("KAFKA_TOPIC"),
		Balancer: &gokafka.LeastBytes{},
	}

	return w
}

func Run(ctx context.Context, params KafkaParams) {

	client := kafkaClient{
		broker:    params.Broker,
		port:      params.Port,
		ch:        params.Channel,
		waitGroup: params.WaitGroup,
		topic:     params.Topic,
		writer:    getWriter(),
	}
	go client.mainloop(ctx)
	log.Debugf("Kafka writer setup against %s:%d", client.broker, client.port)

}

func (client kafkaClient) mainloop(ctx context.Context) {
	client.waitGroup.Add(1)

	select {
	case <-ctx.Done():
		log.Info("Kafka writer shutting down")
		break
	}
	client.waitGroup.Done()
}
