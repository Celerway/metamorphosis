package integration

import (
	"context"
	"fmt"
	"github.com/celerway/metamorphosis/bridge"
	"github.com/pingcap/failpoint"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"
)

const noOfMessages = 20
const originMqttPort = 1883
const originKafkaPort = 9092
const defaultHealthPort = 8080
const startServices = false

func TestMain(m *testing.M) {
	// Main setup goroutine
	var (
		rootCtx, serviceCtx context.Context
		cancel              context.CancelFunc
	)
	f := log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	}
	log.SetLevel(log.InfoLevel)

	log.SetFormatter(&f)
	log.Debug("Log level set")
	if startServices {
		stopAllServices() // Attempt to stop all the services so we have a known state.
		rootCtx = context.Background()
		serviceCtx, cancel = context.WithCancel(rootCtx)
		startMqtt(serviceCtx)
		startKafka(serviceCtx)
		startToxi(serviceCtx)
	}
	rand.Seed(time.Now().Unix())
	ret := m.Run() // Run the tests.
	if startServices {
		fmt.Println("Stopping Services")
		cancel()
	}
	os.Exit(ret)

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

	logger := log.WithFields(log.Fields{"test": "TestBasic"})
	fmt.Println("Testing basic stuff")
	rootCtx := context.Background()
	rTopic := getRandomString(12)
	kafkaTopic(t, "create", rTopic)
	bridgeCtx, bridgeCancel := context.WithCancel(rootCtx)
	wg := sync.WaitGroup{}
	go bridge.Run(bridgeCtx, mkBrigeParam(&wg, originMqttPort, originKafkaPort, defaultHealthPort, rTopic))
	waitForBridge(t, logger, defaultHealthPort)
	publishMqttMessages(t, rTopic, noOfMessages, 0, originMqttPort) // Publish messages
	verifyKafkaMessages(t, rTopic, noOfMessages, originKafkaPort)   // Verify the messages.
	// +1 for the kafka messages. (There is a test message, you know)
	verifyObsdata(t, defaultHealthPort, noOfMessages, noOfMessages+1, 0, 0)
	bridgeCancel()
	kafkaTopic(t, "delete", rTopic)
	wg.Wait()
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

func TestKafkaFailure(t *testing.T) {
	logger := log.WithFields(log.Fields{"test": "TestKafkaFailure"})

	fmt.Println("TestKafkaFailure")
	rootCtx := context.Background()
	rTopic := getRandomString(12)
	kafkaTopic(t, "create", rTopic)
	bridgeCtx, bridgeCancel := context.WithCancel(rootCtx)
	wg := sync.WaitGroup{}
	go bridge.Run(bridgeCtx, mkBrigeParam(&wg, originMqttPort, originKafkaPort, defaultHealthPort, rTopic))
	waitForBridge(t, logger, defaultHealthPort)
	publishMqttMessages(t, rTopic, noOfMessages, 0, originMqttPort) // Publish messages
	time.Sleep(1 * time.Second)                                     // Give kafka time to write stuff.
	fmt.Println("==== Kafka DISABLED === ")
	failpoint.Enable("github.com/celerway/metamorphosis/bridge/kafka/writePoint", "return(true)")
	// New batch of messages. Now kafka should be dead. note the offset.
	publishMqttMessages(t, rTopic, noOfMessages, noOfMessages, originMqttPort) // Publish messages
	time.Sleep(3 * time.Second)
	fmt.Println("==== Kafka ENABLED === ")
	failpoint.Disable("github.com/celerway/metamorphosis/bridge/kafka/writePoint")

	time.Sleep(3 * time.Second)                                     // Give Kafka time to recover.
	verifyKafkaMessages(t, rTopic, noOfMessages*2, originKafkaPort) // Verify the messages.
	fmt.Println("Done verifying kafka data. Checking obs data.")
	verifyObsdata(t, defaultHealthPort, noOfMessages*2, noOfMessages*2+2, 0, 2)
	bridgeCancel()
	kafkaTopic(t, "delete", rTopic)
	wg.Wait()
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
