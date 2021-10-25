package integration

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/celerway/metamorphosis/bridge"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/common/expfmt"
	gokafka "github.com/segmentio/kafka-go"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

func promVerify(t *testing.T, name string, value float64, exptected int) {
	if int(value) != exptected {
		t.Errorf("Observed counter %s mismatch, expected %d, got %d (%f)",
			name, exptected, int(value), value)
	}
}

func verifyObsdata(t *testing.T, port, mqttMessages, kafkaMessages, mqttErrors, kafkaErrors int) {
	url := fmt.Sprintf("http://localhost:%d/metrics", port)
	fmt.Printf("Quering metrics on %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("Could not get metrics (%s): %s", url, err)
	}
	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		t.Errorf("TextToMetricFamilies failed: %s", err)
	}
	promVerify(t, "mqtt_received", *mf["mqtt_received"].Metric[0].Counter.Value, mqttMessages)
	promVerify(t, "kafka_sent", *mf["kafka_sent"].Metric[0].Counter.Value, kafkaMessages)
	promVerify(t, "mqtt_errors", *mf["mqtt_errors"].Metric[0].Counter.Value, mqttErrors)
	promVerify(t, "kafka_errors", *mf["kafka_errors"].Metric[0].Counter.Value, kafkaErrors)
}

func verifyKafkaDown(t *testing.T, port int) {
	url := fmt.Sprintf("http://localhost:%d/metrics", port)
	fmt.Printf("Quering metrics on %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("Could not get metrics (%s): %s", url, err)
	}
	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		t.Errorf("TextToMetricFamilies failed: %s", err)
	}

	promVerify(t, "kafka_state", *mf["kafka_state"].Metric[0].Gauge.Value, 1)
}

func mkBrigeParam(wg *sync.WaitGroup, mqttPort, kafkaPort, healthPort int, topic string) bridge.BridgeParams {
	return bridge.BridgeParams{
		MqttBroker:         "localhost",
		MqttPort:           mqttPort,
		MqttTopic:          topic,
		MqttTls:            false,
		KafkaBroker:        "127.0.0.1",
		KafkaPort:          kafkaPort,
		KafkaTopic:         topic,
		KafkaWorkers:       1,
		HealthPort:         healthPort,
		KafkaRetryInterval: 3 * time.Second,
		MainWaitGroup:      wg,
	}

}
func waitForBridge(port int) {
	url := fmt.Sprintf("http://localhost:%d/healthz", port)
	fmt.Printf("Waiting for bridge to come up on %s\n", url)

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
	fmt.Printf("Bridge OK\n")
}

func getRandomString(length int) string {
	var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
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
		return false, fmt.Errorf("mqtt message json unmarshal: %s (message: %s)", err, kmsg.Content)
	}
	fmt.Printf("Actual message being verified: %v", actualMessage)
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
		fmt.Printf("Sending message with ID %d on MQTT\n", i)
		if err != nil {
			t.Errorf("While making message: %s", err)
		}
		token := client.Publish(topic, 1, false, msg)
		token.Wait()
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Printf("Published %d messages on MQTT\n", noMessages)
	client.Disconnect(0)
}

func verifyKafkaMessages(t *testing.T, topic string, noOfMessages, port int) {
	fmt.Println("============== verifyKafkaMessages =======")
	client := getKafkaReader(port, topic)
	err := client.SetOffset(0)
	if err != nil {
		t.Errorf("Setting offset: %s", err)
	}
	noOfValidMessages := 0

	for noOfValidMessages < noOfMessages {
		fmt.Printf("kafka: reading message (expected id %d)\n", noOfValidMessages)
		msg, err := client.ReadMessage(context.Background())
		if err != nil {
			t.Errorf("Reading kafka message: %s", err)

		}
		valid, err := verifyMessage(noOfValidMessages, msg, topic)
		if err != nil {
			t.Errorf("Verifying kafka message: %s", err)
		} else {
			if valid == false {
				fmt.Println("Test message observed (verifymessage is false, no error)")
			} else {
				fmt.Println("Verified message OK")
			}
		}
		if valid {
			fmt.Printf("Message %d is valid\n", noOfValidMessages)
			noOfValidMessages++
		}
	}
	fmt.Printf("Verified %d messages from Kafka\n", noOfMessages)
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
		fmt.Printf("Setting offset: %s\n", err)
		panic(err)
	}
	return r
}
