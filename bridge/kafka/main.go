package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	gokafka "github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"strconv"
	"time"
)

const (
	maxBatchSize = 8000
)

func Initialize(p Params) *buffer {
	logger := log.WithFields(log.Fields{"module": "kafka"})
	brokerAddr := gokafka.TCP(p.Broker + ":" + strconv.FormatInt(int64(p.Port), 10))
	writer := &gokafka.Writer{
		Addr:         brokerAddr,
		Topic:        p.Topic,
		MaxAttempts:  10,
		BatchSize:    1,
		BatchTimeout: time.Millisecond * 20,
		RequiredAcks: gokafka.RequireAll,
		Async:        false,
		Compression:  0,
		Logger:       nil,
		ErrorLogger:  logger,
	}
	return &buffer{
		batchSize:            p.BatchSize,
		interval:             p.Interval,
		failureState:         false,
		failureRetryInterval: p.RetryInterval,
		C:                    p.Channel,
		buffer:               make([]gokafka.Message, 0, 1000),
		writer:               writer,
		maxBatchSize:         maxBatchSize,
		kafkaTimeout:         time.Second,
		logger:               logger,
	}
}

// Run starts monitoring the channel and sends messages to the broker.
func (k *buffer) Run(ctx context.Context) error {
	err := k.sendTestMessage()
	if err != nil {
		return fmt.Errorf("failed to send initial test message: %w", err)
	}
	ticker := time.NewTicker(k.interval)
	k.logger.Infof("Kafka buffer started with interval %v", k.interval)
loop:
	for {
		select {
		case <-ctx.Done():
			k.logger.Info("Kafka buffer: context cancelled")
			break loop
		case <-ticker.C:
			if time.Since(k.lastSendAttempt) > k.interval {
				k.logger.Trace("Tick: Send it!")
				k.Send(false)
			}
		case m := <-k.C:
			k.logger.Trace("Kafka buffer: Message received")
			k.Enqueue(m)
		}
	}
	ticker.Stop()
	k.logger.Info("Final flush of the buffer")
	k.Send(true)
	return nil
}

// Enqueue adds a message to the buffer
// It'll transform it from the Message type (used by MQTT) to what Kafka expects.
// if the number of enqueued messages is greater than the batch size, it'll send them.
func (k *buffer) Enqueue(msg Message) {
	msgJson, err := json.Marshal(msg)
	if err != nil {
		// todo: Log the error. There is nothing else we can do here.
		return
	}
	m := gokafka.Message{
		Value: msgJson,
	}
	k.buffer = append(k.buffer, m)
	if len(k.buffer) > k.batchSize {
		if k.failureState {
			// Not triggering flush if we're failing.
			return
		}
		k.logger.Debugf("Triggering flush (buffer is %d, batchSize is %d)", len(k.buffer), k.batchSize)
		k.Send(false)
		return
	}
}

// Send will send all messages in the buffer to the gokafka broker
func (k *buffer) Send(force bool) {

	if len(k.buffer) == 0 {
		k.logger.Trace("buffer empty")
		return
	}
	if k.failureState && time.Since(k.lastSendAttempt) < k.failureRetryInterval {
		if force {
			k.logger.Trace("Forced send")
		} else {
			k.logger.Tracef("In a failed state. Not time to retry yet. Time since last check: %v (%v)", time.Since(k.lastSendAttempt), k.failureRetryInterval)
			return
		}
	}
	k.logger.Debug("Attempting to send messages")
	defer k.updateLastSendAttempt() // update the attempt time, even if we fail.
	var err error
	start := time.Now()
	if len(k.buffer) < k.maxBatchSize {
		err = k.sendAll()
	} else {
		err = k.sendBatched()
	}
	if err != nil {
		k.logger.Warnf("kafkabuffer/Send: %s (time taken: %v, failures: %d)", err, time.Since(start), k.failures)
		k.failureState = true
		k.failures++
		return
	}
	log.Debugf("kafkabuffer/Send: Wrote %d messages in %v", len(k.buffer), time.Since(start))
	k.failureState = false
	k.buffer = k.buffer[:0] // clear the buffer
	k.updateLastSendAttempt()
}

// sendAll sends all messages in the buffer.
func (k *buffer) sendAll() error {
	k.logger.Debugf("Sending all messages (%d) in the buffer", len(k.buffer))
	ctx, cancel := context.WithTimeout(context.Background(), k.kafkaTimeout)
	defer cancel()
	return k.writer.WriteMessages(ctx, k.buffer...)
}

// sendBatched sends messages in batches of maxBatchSize.
func (k *buffer) sendBatched() error {
	l := len(k.buffer)
	k.logger.Debugf("Sending all (%d) messages in batches", l)
	batches := l / k.maxBatchSize
	timeout := k.kafkaTimeout * time.Duration(batches)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	batch := 0
	for {
		batch++
		k.logger.Debug("attempting to send batch:", batch)
		l := len(k.buffer)
		if l == 0 {
			break
		}
		if l < k.maxBatchSize {
			err := k.writer.WriteMessages(ctx, k.buffer...)
			if err != nil {
				return fmt.Errorf("error batch %d", batch)
			}
			k.buffer = k.buffer[:0] // done. clear the buffer.
			break
		} else {
			err := k.writer.WriteMessages(ctx, k.buffer[:k.maxBatchSize]...)
			if err != nil {
				return err
			}
			k.buffer = k.buffer[k.maxBatchSize:] // remove the first k.maxBatchSize messages from the buffer.
		}
	}
	return nil
}

// sendTestMessage sends a test message with the mqtt topic "test".
// You wanna ignore these messages in the Kafka consumers.
func (k *buffer) sendTestMessage() error {
	ctx, cancel := context.WithTimeout(context.Background(), k.kafkaTimeout)
	defer cancel()
	err := k.writer.WriteMessages(ctx, generateTestMessage())
	return err
}

func (k *buffer) updateLastSendAttempt() {
	k.lastSendAttempt = time.Now()
}

func generateTestMessage() gokafka.Message {
	msg := Message{
		Topic:   "test",
		Content: []byte("Internal test to see if gokafka is alive at startup"),
	}
	msgJson, err := json.Marshal(msg)
	if err != nil {
		// something is very wrong. bail out.
		log.Fatalf("mashalling test message: %s", err)
	}
	testMsg := gokafka.Message{
		Value: msgJson,
	}
	return testMsg
}
