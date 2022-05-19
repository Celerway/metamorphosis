package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	is2 "github.com/matryer/is"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type mockWriter struct {
	mu         sync.Mutex
	storage    []kafka.Message
	failed     bool
	msgs       uint64
	writes     uint64
	deadlock   bool
	batchDelay time.Duration
	msgDelay   time.Duration
}

func (m *mockWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// if deadlock, block until context is cancelled
	if m.deadlock {
		<-ctx.Done()
	}
	time.Sleep(m.batchDelay + m.msgDelay*time.Duration(len(msgs)))
	if m.failed {
		return errors.New("storage is in a failed state")
	}
	if m.storage == nil {
		m.storage = make([]kafka.Message, 0)
	}
	l := uint64(len(msgs))
	log.Debugf("Writing %d messages to pretend kafka", l)
	m.storage = append(m.storage, msgs...)
	atomic.AddUint64(&m.msgs, l)
	atomic.AddUint64(&m.writes, 1)
	return nil
}

func (m *mockWriter) setDelay(batchDelay, msgDelay time.Duration) {
	log.Infof("Setting storage delay to %v for batch / %v for msg", batchDelay, msgDelay)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.batchDelay = batchDelay
	m.msgDelay = msgDelay
}
func (m *mockWriter) setState(failed bool) {
	log.Info("Setting storage failed state to ", failed)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failed = failed
}
func (m *mockWriter) setDeadlock(deadlock bool) {
	log.Info("Setting storage deadlock to ", deadlock)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deadlock = deadlock
}

func (m *mockWriter) getMessage(id int) (Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id >= len(m.storage) {
		return Message{}, errors.New("message not found")
	}
	var Msg Message
	err := json.Unmarshal(m.storage[id].Value, &Msg)
	if err != nil {
		return Message{}, err
	}
	return Msg, nil
}

func (m *mockWriter) getDecodedMessage(id int) (Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id >= len(m.storage) {
		return Message{}, errors.New("message not found")
	}
	mess := m.storage[id]
	var Msg Message
	err := json.Unmarshal(mess.Value, &Msg)
	if err != nil {
		return Message{}, err
	}
	return Msg, nil

}

func waitForAtomic(a *uint64, v uint64, timeout, sleeptime time.Duration) error {
	start := time.Now()
	for time.Since(start) < timeout {
		if atomic.LoadUint64(a) >= v {
			return nil
		}
		time.Sleep(sleeptime)
	}
	return fmt.Errorf("waitForAtomic (waiting for %d, is %d) timed out after %v", v, atomic.LoadUint64(a), timeout)
}

func TestMain(m *testing.M) {
	log.SetLevel(log.TraceLevel)
	log.Debug("Running test suite")
	ret := m.Run()
	log.Debug("Test suite complete")
	os.Exit(ret)
}

// Test that we can start and stop a buffer.
func TestBuffer_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	storage := &mockWriter{}
	buffer := makeTestBuffer(storage)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := buffer.Run(ctx)
		if err != nil {
			log.Errorf("Error %s", err)
		}
		log.Info("buffer run complete")
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	log.Debug("Cancel issued. Waiting.")
	wg.Wait()
	log.Debug("Done")
}

func makeTestBuffer(writer *mockWriter) buffer {
	return buffer{
		interval:             1 * time.Millisecond,
		failureRetryInterval: 80 * time.Millisecond,
		buffer:               make([]kafka.Message, 0, 10),
		topic:                "unittest",
		writer:               writer,
		C:                    make(chan Message, 0),
		batchSize:            5,
		maxBatchSize:         20,
		kafkaTimeout:         25 * time.Millisecond,
	}
}

// Simple test. Send 10 messages and check that they are all received.
func TestBuffer_Process_ok(t *testing.T) {
	is := is2.New(t)
	storage := &mockWriter{}
	ctx, cancel := context.WithCancel(context.Background())
	buffer := makeTestBuffer(storage)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := buffer.Run(ctx)
		if err != nil {
			log.Errorf("Error %s", err)
		}
		log.Info("buffer run complete")
	}()

	for i := 0; i < 10; i++ {
		buffer.C <- makeMessage("test", i)
	}
	cancel()
	wg.Wait()
	for i := 1; i <= 10; i++ {
		m, err := storage.getDecodedMessage(i)
		is.NoErr(err)
		is.Equal([]byte(fmt.Sprintf("Test message %d", i-1)), m.Content)
	}
	log.Debug("Done")
}

// Somewhat more advanced. Induce a failure and check that the buffer recovers.
func TestBuffer_Process_fail(t *testing.T) {
	storage := &mockWriter{}
	ctx, cancel := context.WithCancel(context.Background())
	buffer := makeTestBuffer(storage)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := buffer.Run(ctx)
		if err != nil {
			log.Errorf("Error %s", err)
		}
		log.Info("buffer run complete")
	}()
	log.Info("Sending msgs 0 -> 5 ")
	for i := 0; i < 5; i++ {
		buffer.C <- makeMessage("test", i)
	}
	storage.setState(true)
	log.Info("Sending msgs 5 -> 10")
	for i := 5; i < 10; i++ {
		buffer.C <- makeMessage("test", i)
	}
	log.Info("Done with msgs")
	time.Sleep(100 * time.Millisecond)
	storage.setState(false)
	time.Sleep(1 * time.Second)
	cancel()
	wg.Wait()
	for i := 0; i < 10; i++ {
		m, err := storage.getMessage(i)
		if err != nil {
			t.Errorf("Error getting message %d: %s", i, err)
		}
		fmt.Printf("Message: %s\n", string(m.Content))
	}
	log.Debug("Done")
}

// Test with the buffer in a failed state at startup.
func TestBuffer_Process_initial_fail(t *testing.T) {
	is := is2.New(t)
	storage := &mockWriter{}
	storage.setState(true)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	buffer := makeTestBuffer(storage)
	err := buffer.Run(ctx)
	is.True(err != nil) // should be error
}

// Induces slowness into the writer. Not sure what it should test, really.
func TestBuffer_Process_slow(t *testing.T) {
	storage := &mockWriter{}
	storage.setDelay(2*time.Millisecond, time.Microsecond*10)
	ctx, cancel := context.WithCancel(context.Background())
	buffer := makeTestBuffer(storage)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := buffer.Run(ctx)
		if err != nil {
			log.Errorf("Error %s", err)
		}
		log.Info("buffer run complete")
	}()
	for i := 0; i < 50; i++ {
		buffer.C <- makeMessage("test", i)
	}
	err := waitForAtomic(&storage.msgs, 51, time.Millisecond*5000, time.Millisecond)
	if err != nil {
		t.Errorf("Error %s", err)
	}
	cancel()
	wg.Wait()
	for i := 0; i < 50; i++ {
		m, err := storage.getMessage(i)
		if err != nil {
			t.Errorf("Error getting message %d: %s", i, err)
		}
		topic := m.Topic
		body := m.Content
		fmt.Printf("Topic: %s Message: %s\n", topic, body)
	}
	log.Debug("Done")
}

func TestBuffer_Batching(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	is := is2.New(t)
	storage := &mockWriter{}
	buffer := makeTestBuffer(storage)
	buffer.batchSize = 100
	buffer.maxBatchSize = 1000
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := buffer.Run(ctx)
		if err != nil {
			log.Errorf("Error %s", err)
		}
		log.Info("buffer run complete")
	}()
	storage.setState(false)
	for i := 0; i < 10000; i++ {
		buffer.C <- makeMessage("test", i)
	}
	storage.setState(false)
	err := waitForAtomic(&storage.msgs, 10001, time.Millisecond*500, time.Millisecond)
	is.NoErr(err)
	is.Equal(atomic.LoadUint64(&storage.writes), uint64(101))
	is.Equal(atomic.LoadUint64(&storage.msgs), uint64(10001))
	cancel()
	wg.Wait()
	log.Debug("Done")

}

// Get the buffer up and running. Fails it and then proceeed to rewrite
// 10000 messages to it. See if it recovers and clears all the messages.
func TestBuffer_Batching_Recovery(t *testing.T) {
	log.SetLevel(log.ErrorLevel)
	is := is2.New(t)
	storage := &mockWriter{}
	buffer := makeTestBuffer(storage)
	buffer.batchSize = 100
	buffer.maxBatchSize = 1000
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := buffer.Run(ctx)
		if err != nil {
			log.Errorf("Error %s", err)
		}
		log.Info("buffer run complete")
	}()
	storage.setState(false)
	buffer.C <- makeMessage("test", 0)
	storage.setState(true)
	for i := 0; i < 10000; i++ {
		buffer.C <- makeMessage("test", i)
	}
	storage.setState(false)
	log.SetLevel(log.DebugLevel)
	err := waitForAtomic(&storage.msgs, 10002, time.Millisecond*2000, time.Millisecond)
	log.Infof("Writes: %d", atomic.LoadUint64(&storage.writes))
	log.Infof("Msgs: %d", atomic.LoadUint64(&storage.msgs))
	log.Infof("Failures: %d", buffer.failures)
	is.NoErr(err)
	is.Equal(atomic.LoadUint64(&storage.msgs), uint64(10002))
	is.Equal(atomic.LoadUint64(&storage.writes), uint64(12))
	is.Equal(buffer.failures, 1)

	cancel()
	wg.Wait()
	log.Debug("Done")

}

// Pump a 1000 messages into the buffer when storage is failed.
// Have the storage recover.
// Interrrupt the recovery by failing the storage in the middle of the recovery.
// Then have the storage recover again
// Finally check that all messages have been written to the storage correctly in the right order.
func TestBuffer_Batching_RecoveryInterrupted(t *testing.T) {
	const count = 1000
	log.SetLevel(log.DebugLevel)
	is := is2.New(t)
	storage := &mockWriter{}
	buffer := makeTestBuffer(storage)
	buffer.batchSize = 10
	buffer.maxBatchSize = 100
	storage.batchDelay = time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := buffer.Run(ctx)
		if err != nil {
			log.Errorf("Error %s", err)
		}
		log.Info("buffer run complete")
	}()
	time.Sleep(10 * time.Millisecond)
	storage.setState(true)
	for i := 1; i <= count; i++ {
		buffer.C <- makeMessage("test", i)
	}
	log.Info("Pumped 1000 messages into buffer")
	storage.setState(false)
	err := waitForAtomic(&storage.msgs, 500, time.Second*3, time.Nanosecond*100)
	is.NoErr(err)
	storage.setState(true)
	time.Sleep(100 * time.Millisecond)
	storage.setState(false)
	err = waitForAtomic(&storage.msgs, 1000, time.Second*3, time.Nanosecond*100)
	is.NoErr(err)
	log.Infof("Writes: %d", atomic.LoadUint64(&storage.writes))
	log.Infof("Msgs: %d", atomic.LoadUint64(&storage.msgs))
	log.Infof("Failures: %d", buffer.failures)
	cancel()
	wg.Wait()
	for i := 1; i <= count; i++ {
		msg := storage.storage[i]
		val := msg.Value
		jmsg := Message{}
		err := json.Unmarshal(val, &jmsg)
		is.NoErr(err)
		is.Equal(fmt.Sprintf("Test message %d", i), string(jmsg.Content))
	}

	log.Debug("Done")

}

func makeMessage(topic string, id int) Message {
	return Message{
		Topic:   topic,
		Content: []byte(fmt.Sprintf("Test message %d", id)),
	}
}
