package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/celerway/metamorphosis/bridge/observability"
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

func (client kafkaClient) handleMessageWrite(ctx context.Context, msg KafkaChannelMessage) {
	log.Debug("Issuing write to kafka")
	msgJson, err := json.Marshal(msg) // Todo: handle error.
	if err != nil {
		log.Fatalf("Could not marshal message %v: %s", msg, err)
	}
	log.Tracef("Kafka(%s): %s", client.topic, string(msgJson))
	kMsg := gokafka.Message{
		Value: msgJson}
	err = client.writer.WriteMessages(ctx, kMsg)
	if err != nil {
		client.obsChannel <- observability.KafkaError
		log.Fatalf("Kafka: Error while writing: %s", err) // Todo: Improve error handling. Do something smart.
	} else {
		client.obsChannel <- observability.KafkaSent
	}
	log.Trace("Message written.")
}

func Run(ctx context.Context, params KafkaParams) {

	client := kafkaClient{
		broker:     params.Broker,
		port:       params.Port,
		ch:         params.Channel,
		waitGroup:  params.WaitGroup,
		topic:      params.Topic,
		writer:     getWriter(),
		obsChannel: params.ObsChannel,
	}

	go client.mainloop(ctx)
	log.Debugf("Kafka writer setup against %s:%d", client.broker, client.port)

}

func (client kafkaClient) mainloop(ctx context.Context) {
	client.waitGroup.Add(1)
	keepRunning := true

	for keepRunning {
		select {
		case <-ctx.Done():
			log.Info("Kafka writer shutting down")
			keepRunning = false
		case msg := <-client.ch:
			client.handleMessageWrite(ctx, msg)
		}
	}
	client.waitGroup.Done()
}
