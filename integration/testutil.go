package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	toxiproxy "github.com/Shopify/toxiproxy/client"
	"github.com/celerway/metamorphosis/bridge"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/common/expfmt"
	gokafka "github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

func deleteProxies(client *toxiproxy.Client) {
	proxies, _ := client.Proxies()
	for _, proxy := range proxies {
		err := proxy.Delete()
		if err != nil {
			panic(err)
		}
	}

}

func verifyCounter(t *testing.T, name string, value float64, exptected int) {
	if int(value) != exptected {
		t.Errorf("Observed counter %s mismatch, expected %d, got %d (%f)",
			name, exptected, int(value), value)
	}
}

func verifyObsdata(t *testing.T, port, mqttMessages, kafkaMessages, mqttErrors, kafkaErrors int) {
	url := fmt.Sprintf("http://localhost:%d/metrics", port)
	log.Debugf("Quering metrics on %s", url)
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("Could not get metrics (%s): %s", url, err)
	}
	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		t.Errorf("TextToMetricFamilies failed: %s", err)
	}
	verifyCounter(t, "mqtt_received", *mf["mqtt_received"].Metric[0].Counter.Value, mqttMessages)
	verifyCounter(t, "kafka_sent", *mf["kafka_sent"].Metric[0].Counter.Value, kafkaMessages)

	verifyCounter(t, "mqtt_errors", *mf["mqtt_errors"].Metric[0].Counter.Value, mqttErrors)
	verifyCounter(t, "kafka_errors", *mf["kafka_errors"].Metric[0].Counter.Value, kafkaErrors)
}

func makeConfig(mqttPort, kafkaPort, healthPort int, topic string) bridge.BridgeParams {
	return bridge.BridgeParams{
		MqttBroker:         "localhost",
		MqttPort:           mqttPort,
		MqttTopic:          topic,
		MqttTls:            false,
		KafkaBroker:        "localhost",
		KafkaPort:          kafkaPort,
		KafkaTopic:         topic,
		KafkaWorkers:       1,
		HealthPort:         healthPort,
		KafkaRetryInterval: 3 * time.Second,
	}

}
func waitForBridge(t *testing.T, logger *log.Entry, port int) {
	url := fmt.Sprintf("http://localhost:%d/healthz", port)
	logger.Debugf("Waiting for bridge to come up on %s", url)

	bridgeOk := false
	for !bridgeOk {
		resp, err := http.Get(url)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if resp.StatusCode == 200 {
			bridgeOk = true
		}
		time.Sleep(100 * time.Millisecond)
	}
	logger.Debug("Bridge OK.")
}
func startMqtt(ctx context.Context) {
	go func() {
		err := exec.CommandContext(ctx, "/usr/sbin/mosquitto").Run()
		if err != nil {
			fmt.Println("Error starting MQTT", err)
		} else {
			fmt.Println("MQTT started")
		}
	}()
}

func startKafka(ctx context.Context) {
	go func() {
		err := exec.CommandContext(ctx, "/usr/bin/rpk", "redpanda", "start")
		if err != nil {
			fmt.Println("Error starting MQTT", err)
		} else {
			fmt.Println("Kafka started")
		}
	}()
}

// Get a set of random ports for the proxy
func getRandomPort() int {
	return rand.Intn(10000) + 50000
}

func getRandomString(length int) string {
	var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func kafkaTopic(t *testing.T, action, topic string) {
	cmd := exec.Command("rpk", "topic", action, topic)
	var stdout, stderr bytes.Buffer
	log.Debugf("kafka topic(%s): %s", action, topic)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Errorf("Kafka topic (%s) stdout: %s stderr: %s err: %s", action, stdout.String(), stderr.String(), err)
	}
}

func getMqttClient(port int) mqtt.Client {
	var broker = "localhost"
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetClientID("go_mqtt_client-" + fmt.Sprint(os.Getpid()))
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	return client
}

func makeMessage(id int) ([]byte, error) {
	rString := getRandomString(200)
	checksum := sha256.Sum256([]byte(rString))
	msg := Message{
		Id:           id,
		RandomString: rString,
		Checksum:     checksum,
	}
	return json.Marshal(msg)
}

// takes a raw message we got from Kafka and
// verify the content.
// fuck this. this is a bit on the complex side.
func verifyMessage(i int, m gokafka.Message, topic string) (bool, error) {
	var kmsg KMessage // Kafka message.
	var actualMessage Message
	if m.Topic != topic {
		panic("Invalid topic")
	}
	// unmarshal the message from Kafka.
	err := json.Unmarshal(m.Value, &kmsg)
	if err != nil {
		return false, fmt.Errorf("kafka message json unmarshal: %s", err)
	}
	if kmsg.Topic == "test" {
		// test message. Ignore.
		fmt.Println("Verify saw a test message. Marking as invalid without errors.")
		return false, nil
	}

	// get the mqtt message. It is now in kmsg.Content
	err = json.Unmarshal(kmsg.Content, &actualMessage)
	if err != nil {
		return false, fmt.Errorf("mqtt message json unmarshal: %s", err)
	}

	// Now check the inner message for content.
	if actualMessage.Id != i {
		return false, fmt.Errorf("kafka message has wrong ID, got %d, expected %d", actualMessage.Id, i)
	}
	msgChecksum := sha256.Sum256([]byte(actualMessage.RandomString))

	if msgChecksum != actualMessage.Checksum {
		return false, fmt.Errorf("Message checksum mismatch, got %s expected %s", msgChecksum, actualMessage.Checksum)
	}
	return true, nil
}

// connects to the MQTT broker, publishes a batch of messages (blocking), then disconnects.
func publishMqttMessages(t *testing.T, topic string, noMessages, offset, port int) {
	client := getMqttClient(port)
	for i := offset; i < noMessages+offset; i++ {
		msg, err := makeMessage(i)
		if err != nil {
			t.Errorf("While making message: %s", err)
		}
		token := client.Publish(topic, 1, false, msg)
		token.Wait()
		log.Debugf("Published message %d on MQTT", i)
		time.Sleep(100 * time.Millisecond)
	}
	client.Disconnect(0)
}

func verifyKafkaMessages(t *testing.T, topic string, noOfMessages, port int) {
	client := getKafkaReader(port, topic)
	noOfValidMessages := 0

	for noOfValidMessages < noOfMessages {
		log.Debugf("kafka: reading message %d", noOfValidMessages)
		msg, err := client.ReadMessage(context.Background())
		if err != nil {
			t.Errorf("Reading kafka message: %s", err)
		}
		valid, err := verifyMessage(noOfValidMessages, msg, topic)
		if err != nil {
			t.Errorf("Verifying kafka message: %s", err)
		}
		log.Debugf("Verify valid: %v", valid)
		if valid {
			noOfValidMessages++
		}
	}
}

func getKafkaReader(port int, topic string) *gokafka.Reader {
	broker := fmt.Sprintf("localhost:%d", port)
	r := gokafka.NewReader(gokafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   topic,
	})
	// Skip the two first messages. The first is a null message. The next is the one the bridge issues.
	err := r.SetOffset(2)
	if err != nil {
		log.Fatalf("Setting offset: %s", err)
	}
	return r
}
