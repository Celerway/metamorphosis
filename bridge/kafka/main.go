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
	"time"
)

func getInt(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		log.Errorf("getInt failed on %s: %s", s, err)
	}
	return val
}

// This creates a write struct. Used when initializing.
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

// The handler that gets called when we get a message.
func handleMessageWrite(ctx context.Context, msg KafkaMessage, client kafkaClient) bool {
	log.Debug("Issuing write to kafka")
	msgJson, err := json.Marshal(msg)
	if err != nil {
		log.Error("Could not marshal message %v: %s", msg, err)
		// Guess there isn't much we can do at this point but to move on.
		client.obsChannel <- observability.KafkaError
	}
	log.Tracef("Kafka(%s): %s", client.topic, string(msgJson))
	kMsg := gokafka.Message{
		Value: msgJson}
	err = client.writer.WriteMessages(ctx, kMsg)
	// Todo: Handle this error specifically.
	// Kafka: Error while writing: context canceled
	// If we're shutting down there is no point in doing anything else.
	if err != nil {
		client.obsChannel <- observability.KafkaError
		log.Errorf("Kafka: Error while writing: %s", err)
		return false
	} else {
		client.obsChannel <- observability.KafkaSent
	}
	log.Trace("Message written.")
	return true
}

func sendTestMessage(ctx context.Context, client kafkaClient) bool {
	testMsg := KafkaMessage{
		Topic:   "test",
		Content: []byte("Just a test"),
	}
	return handleMessageWrite(ctx, testMsg, client)
}

// Run Constructor. Sort of.
func Run(ctx context.Context, params KafkaParams) {
	// This should be fairly easy to test in case we wanna mock Kafka.
	client := kafkaClient{
		broker:       params.Broker,
		port:         params.Port,
		ch:           params.Channel,
		waitGroup:    params.WaitGroup,
		topic:        params.Topic,
		writer:       getWriter(),
		obsChannel:   params.ObsChannel,
		writeHandler: handleMessageWrite,
	}
	go mainloop(ctx, client)
	log.Debugf("Kafka writer setup against %s:%d", client.broker, client.port)

}

func mainloop(ctx context.Context, client kafkaClient) {
	client.waitGroup.Add(1)
	if !sendTestMessage(ctx, client) {
		log.Fatalf("Can't send test message on startup. Aborting.")
	}
	keepRunning := true
	msgBuffer := make([]KafkaMessage, 0) // Buffer to store things if Kafka is causing issues.
	var failed bool
	var lastAttempt time.Time
	for keepRunning {
		select {
		case <-ctx.Done():
			log.Info("Kafka writer shutting down")
			keepRunning = false
		case msg := <-client.ch:
			if !failed { // We assume Kafka is up.
				success := client.writeHandler(ctx, msg, client) // Send msg. Get back status.
				if !success {
					msgBuffer = append(msgBuffer, msg) // spool the message.
					failed = true
					lastAttempt = time.Now() // Time of last failure.
				}
			} else {
				// Here we're in trouble. We assume Kafka is down. We retest every 10s until we get it working again.
				if time.Since(lastAttempt) < 10*time.Second { // Less than 10s since last try. Just spool the message.
					msgBuffer = append(msgBuffer, msg)
					log.Infof("Message spooled. Currently %d messages in the spool.", len(msgBuffer))
				} else {
					// more than 10s passed. Lets try a test message.
					success := sendTestMessage(ctx, client)
					if success {
						log.Info("Kafka has recovered")
						failed = false
						lastAttempt = time.Now()
						msgBuffer, failed = despool(ctx, msgBuffer, client) // Actual de-spool here.
					} else {
						log.Warn("Kafka is still down. Will retry in 10s")
						msgBuffer = append(msgBuffer, msg)
					}
				}
			}
		}
	}
	log.Info("Kafka done.")
	client.waitGroup.Done()
}

func despool(ctx context.Context, buffer []KafkaMessage, client kafkaClient) ([]KafkaMessage, bool) {
	failed := false
	for i, msg := range buffer {
		success := client.writeHandler(ctx, msg, client)
		if success {
			continue
		}
		// Gosh darn it! Kafka is down again.
		// i should point at the last successful message we sent.
		// If we didn't send any it'll be 0.
		buffer = buffer[i:]
		failed = true
	}

	return buffer, failed
}
