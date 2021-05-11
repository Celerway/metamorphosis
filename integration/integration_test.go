package main

import (
	"context"
	"fmt"
	toxiproxy "github.com/Shopify/toxiproxy/client"
	"github.com/celerway/metamorphosis/bridge"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"testing"
	"time"
)

const noOfMessages = 5
const originMqttPort = 1883
const originKafkaPort = 9093
const defaultHealthPort = 8080
const toxiPort = 8474
const startServices = false

func TestMain(m *testing.M) {
	// Main setup goroutine
	var (
		rootCtx, mqttCtx, kafkaCtx context.Context
		mqttCancel, kafkaCancel    context.CancelFunc
	)
	log.SetLevel(log.InfoLevel)
	f := log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	}

	log.SetFormatter(&f)
	log.Debug("Set log level")
	if startServices {
		rootCtx = context.Background()
		fmt.Println("Starting MQTT")
		mqttCtx, mqttCancel = context.WithCancel(rootCtx)
		startMqtt(mqttCtx)
		fmt.Println("Starting Kafka")
		kafkaCtx, kafkaCancel = context.WithCancel(rootCtx)
		startKafka(kafkaCtx)
	}
	rand.Seed(time.Now().Unix())
	ret := m.Run() // Run the tests.
	if startServices {
		fmt.Println("Stopping MQTT")
		mqttCancel()
		fmt.Println("Stopping Kafka")
		kafkaCancel()
		os.Exit(ret)
	}
}

func TestDummy(t *testing.T) {
	fmt.Println("Dummy test ok")
}

/*
   	plan:
         - create a topic in Kafka (random name)
         - spin up the proxy
         - spin up bridge (remember to connect to the ports)
         - push X messages through the MQTT
         - see that they arrive in Kafka
         - verify obs data
*/
func TestBasic(t *testing.T) {
	// Todo: Remove the proxy mess from this test. It isn't needed.
	logger := log.WithFields(log.Fields{"test": "TestBasic"})
	fmt.Println("Testing basic stuff")
	rootCtx := context.Background()
	rTopic := getRandomString(12)
	kafkaTopic(t, "create", rTopic)
	bridgeCtx, bridgeCancel := context.WithCancel(rootCtx)
	go bridge.Run(bridgeCtx, makeConfig(originMqttPort, originKafkaPort, defaultHealthPort, rTopic))
	waitForBridge(t, logger, defaultHealthPort)
	publishMqttMessages(t, rTopic, noOfMessages, 0, originMqttPort) // Publish messages
	verifyKafkaMessages(t, rTopic, noOfMessages, originKafkaPort)   // Verify the messages.
	// +1 for the kafka messages. (There is a test message, you know)
	verifyObsdata(t, defaultHealthPort, noOfMessages, noOfMessages+1, 0, 0)
	bridgeCancel()
	kafkaTopic(t, "delete", rTopic)
}

/*
   	plan:
         - create a topic in Kafka (random name)
         - spin up the proxy (with two random ports)
         - spin up bridge (remember to connect to the ports)
         - push 100 messages through the MQTT
         - cancel the proxy
         - push 100 messages through the MQTT broker
         - wait a while (11 seconds)
         - start the proxy again
         - wait another while (10 seconds)
         - see that they have all arrive in Kafka
         - verify obs data
*/

func TestBasicWithProxy(t *testing.T) {
	// Todo: Remove the proxy mess from this test. It isn't needed.
	logger := log.WithFields(log.Fields{"test": "TestBasic"})

	kafkaPort := getRandomPort()
	toxi := toxiproxy.NewClient(fmt.Sprintf("localhost:%d", toxiPort))
	deleteProxies(toxi) // Todo: This should be removed when tests are solid.
	proxy, err := toxi.CreateProxy("kafka",
		fmt.Sprintf("localhost:%d", kafkaPort),
		fmt.Sprintf("localhost:%d", originKafkaPort))
	defer proxy.Delete()
	if err != nil {
		panic(err)
	}
	fmt.Println("Testing basic stuff")
	rootCtx := context.Background()
	rTopic := getRandomString(12)
	kafkaTopic(t, "create", rTopic)
	bridgeCtx, bridgeCancel := context.WithCancel(rootCtx)
	go bridge.Run(bridgeCtx, makeConfig(originMqttPort, kafkaPort, defaultHealthPort, rTopic))
	waitForBridge(t, logger, defaultHealthPort)
	publishMqttMessages(t, rTopic, noOfMessages, 0, originMqttPort) // Publish messages
	verifyKafkaMessages(t, rTopic, noOfMessages, originKafkaPort)   // Verify the messages.
	// +1 for the kafka messages. (There is a test message, you know)
	verifyObsdata(t, defaultHealthPort, noOfMessages, noOfMessages+1, 0, 0)
	bridgeCancel()
	kafkaTopic(t, "delete", rTopic)
}

func TestKafkaFailure(t *testing.T) {
	logger := log.WithFields(log.Fields{"test": "TestBasic"})

	// kafkaPort := getRandomPort()
	kafkaPort := 9092
	toxi := toxiproxy.NewClient(fmt.Sprintf("localhost:%d", toxiPort))
	deleteProxies(toxi)
	// proxy, err := toxi.CreateProxy("kafka",fmt.Sprintf("localhost:%d", kafkaPort),fmt.Sprintf("localhost:%d", originKafkaPort))
	proxy, err := toxi.CreateProxy("kafka", fmt.Sprintf("localhost:%d", kafkaPort), fmt.Sprintf("localhost:%d", originKafkaPort))
	defer proxy.Delete()
	if err != nil {
		panic(err)
	}
	fmt.Println("Testing basic stuff")
	rootCtx := context.Background()
	rTopic := getRandomString(12)
	kafkaTopic(t, "create", rTopic)
	bridgeCtx, bridgeCancel := context.WithCancel(rootCtx)
	go bridge.Run(bridgeCtx, makeConfig(originMqttPort, kafkaPort, defaultHealthPort, rTopic))
	waitForBridge(t, logger, defaultHealthPort)
	publishMqttMessages(t, rTopic, noOfMessages, 0, originMqttPort) // Publish messages
	time.Sleep(1 * time.Second)                                     // Give kafka time to write stuff.
	proxy.Disable()
	fmt.Println("==== Kafka DISABLED === ")
	time.Sleep(1 * time.Second)
	// New batch of messages. Now kafka should be dead. note the offset.
	publishMqttMessages(t, rTopic, noOfMessages, noOfMessages, originMqttPort) // Publish messages
	time.Sleep(3 * time.Second)
	proxy.Enable()
	fmt.Println("==== Kafka ENABLED === ")

	time.Sleep(3 * time.Second)                                     // Give Kafka time to recover.
	verifyKafkaMessages(t, rTopic, noOfMessages*2, originKafkaPort) // Verify the messages.
	fmt.Println("Done verifying kafka data. Checking obs data.")
	verifyObsdata(t, defaultHealthPort, noOfMessages*2, noOfMessages*2+2, 0, 2)
	bridgeCancel()
	kafkaTopic(t, "delete", rTopic)

}

func TestMqttFailure(t *testing.T) {
	fmt.Println("Testing kafka failure")
	/*
		plan:
			 - create a topic in Kafka (random name)
			 - spin up the proxy (with two random ports)
			 - spin up bridge (remember to connect to the ports)
			 - push 100 messages through the MQTT
			 - SIGINT the proxy so it shuts down
			 - spin up another proxy on the same ports
			 - push 100 messages through the MQTT broker
			 - see that they have all arrive in Kafka
			 - verify obs data
	*/
}
