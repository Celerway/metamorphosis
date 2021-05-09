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

// Run Constructor. Sort of.
func Run(ctx context.Context, params KafkaParams, id int) {
	// This should be fairly easy to test in case we wanna mock Kafka.
	client := kafkaClient{
		broker:       params.Broker,
		port:         params.Port,
		ch:           params.Channel,
		waitGroup:    params.WaitGroup,
		topic:        params.Topic,
		obsChannel:   params.ObsChannel,
		writeHandler: handleMessageWrite,
		logger: log.WithFields(log.Fields{
			"module": "kafka",
			"worker": fmt.Sprint(id),
		}),
	}
	client.writer = getWriter(client.logger) // Give it the context aware logger.

	// Sends a test message to Kafka. This will block Run so when Run returns we
	// know we're OK.
	if !sendTestMessage(ctx, client) {
		client.logger.Fatalf("Can't send test message on startup. Aborting.")
	}
	go mainloop(ctx, client)

}

func mainloop(ctx context.Context, client kafkaClient) {
	client.waitGroup.Add(1)
	keepRunning := true
	msgBuffer := make([]KafkaMessage, 0) // Buffer to store things if Kafka is causing issues.
	var failed bool
	var lastAttempt time.Time
	client.logger.Infof("Kafka writer running %s:%d", client.broker, client.port)
	for keepRunning {
		select {
		case <-ctx.Done():
			client.logger.Info("Kafka writer shutting down")
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
					// Todo: Should we limit the number of messages we can spool?
					// Now the heap will just grow and grow until it ooms.
					client.logger.Infof("Message spooled. Currently %d messages in the spool.", len(msgBuffer))
				} else {
					// more than 10s passed. Lets try a test message.
					success := sendTestMessage(ctx, client)
					if success {
						client.logger.Info("Kafka has recovered")
						failed = false
						lastAttempt = time.Now()
						msgBuffer, failed = despool(ctx, msgBuffer, client) // Actual de-spool here.
					} else {
						client.logger.Warn("Kafka is still down. Will retry in 10s")
						msgBuffer = append(msgBuffer, msg)
					}
				}
			}
		}
	}
	client.logger.Info("Kafka done.")
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
func getInt(s string, logger *log.Entry) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		logger.Errorf("getInt failed on %s: %s", s, err)
	}
	return val
}

// This creates a write struct. Used when initializing.
func getWriter(logger *log.Entry) *gokafka.Writer {
	broker := fmt.Sprintf("%s:%d",
		os.Getenv("KAFKA_BROKER"), getInt(os.Getenv("KAFKA_PORT"), logger))
	w := &gokafka.Writer{
		Addr:         gokafka.TCP(broker),
		Topic:        os.Getenv("KAFKA_TOPIC"),
		Balancer:     &gokafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond, // Only give the client 10ms to batch up writes.
	}
	return w
}

// The handler that gets called when we get a message.
func handleMessageWrite(ctx context.Context, msg KafkaMessage, client kafkaClient) bool {
	startWriteTime := time.Now()
	client.logger.Trace("Issuing write to kafka (mqtt topic: %s)", msg.Topic)
	msgJson, err := json.Marshal(msg)
	if err != nil {
		client.logger.Error("Could not marshal message %v: %s", msg, err)
		// Guess there isn't much we can do at this point but to move on.
		client.obsChannel <- observability.KafkaError
		return true // We ignore these errors. No sense in re-trying.
	}
	client.logger.Tracef("Kafka(%s): %s", client.topic, string(msgJson))
	kMsg := gokafka.Message{
		Value: msgJson}
	err = client.writer.WriteMessages(ctx, kMsg)
	// Todo: Handle this error specifically Kafka: Error while writing: context canceled
	// If we're shutting down there is no point in doing anything else.
	if err != nil {
		client.obsChannel <- observability.KafkaError
		client.logger.Errorf("Kafka: Error while writing: %s", err)
		return false
	} else {
		client.obsChannel <- observability.KafkaSent
	}
	client.logger.Debugf("Write done. Took %v", time.Since(startWriteTime))
	return true
}

// sendTestMessage sends a test message with the mqtt topic "test".
// You wanna ignore these messages in the Kafka consumers.
func sendTestMessage(ctx context.Context, client kafkaClient) bool {
	testMsg := KafkaMessage{
		Topic:   "test",
		Content: []byte("Just a test"),
	}
	return handleMessageWrite(ctx, testMsg, client)
}
