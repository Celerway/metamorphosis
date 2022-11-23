package main

import (
	"context"
	_ "embed"
	"flag"
	"github.com/celerway/metamorphosis/bridge"
	"github.com/celerway/metamorphosis/log"
	"github.com/joho/godotenv"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed .version
var embeddedVersion string

func main() {
	var ( // default settings:
		logLevelStr          string
		mqttBroker           string
		mqttPort             int = 8883
		mqttTopic            string
		mqttTls              bool   = true
		mqttClientId         string = "metamorphosis"
		caRootCertFile       string
		mqttCaClientCertFile string
		mqttCaClientKeyFile  string
		kafkaBroker          string
		kafkaPort            int = 9092
		kafkaTopic           string
		healthPort           int    = 8080
		kafkaRetryInterval   int    = 3
		kafkaInterval        int    = 5
		kafkaBatchSize       int    = 1000
		kafkaMaxBatchSize    int    = 8000
		kafkaTestTopic       string = ""
	)

	err := godotenv.Load()
	log.Infof("Metamorphosis %s starting up.", embeddedVersion)
	if err != nil {
		log.Infof("Error loading .env file, assuming production: %s", err.Error())
	}

	flag.StringVar(&logLevelStr, "log-level",
		LookupEnvOrString("LOG_LEVEL", logLevelStr), "Log level (trace|debug|info|warn|error")
	flag.StringVar(&caRootCertFile, "root-ca",
		LookupEnvOrString("ROOT_CA", caRootCertFile), "Path to root CA certificate (pubkey)")
	flag.StringVar(&mqttCaClientCertFile, "mqtt-client-cert",
		LookupEnvOrString("MQTT_CLIENT_CERT", mqttCaClientCertFile), "Path to client cert (pubkey)")
	flag.StringVar(&mqttCaClientKeyFile, "mqtt-client-key",
		LookupEnvOrString("MQTT_CLIENT_KEY", mqttCaClientKeyFile), "Path to client key (privkey)")
	flag.BoolVar(&mqttTls, "mqtt-tls",
		LookupEnvOrBool("MQTT_TLS", mqttTls), "Tls (true|false)")
	flag.StringVar(&mqttBroker, "mqtt-broker",
		LookupEnvOrString("MQTT_BROKER", mqttBroker), "MQTT broker hostname")
	flag.IntVar(&mqttPort, "mqtt-port",
		LookupEnvOrInt("MQTT_PORT", mqttPort), "Mqtt broker port.")
	flag.StringVar(&mqttTopic, "mqtt-topic",
		LookupEnvOrString("MQTT_TOPIC", mqttTopic), "MQTT topic to listen to (wildcards ok)")
	flag.StringVar(&mqttClientId, "mqtt-client-id",
		LookupEnvOrString("MQTT_CLIENT_ID", mqttClientId), "MQTT client id")
	flag.StringVar(&kafkaBroker, "kafka-broker",
		LookupEnvOrString("KAFKA_BROKER", kafkaBroker), "Kafka broker hostname")
	flag.IntVar(&kafkaPort, "kakfa-port",
		LookupEnvOrInt("KAFKA_PORT", kafkaPort), "Kafka broker port")
	flag.StringVar(&kafkaTopic, "kafka-topic",
		LookupEnvOrString("KAFKA_TOPIC", kafkaTopic), "Kafka topic to write to")
	flag.IntVar(&kafkaRetryInterval, "kafka-retry-interval",
		LookupEnvOrInt("KAFKA_RETRY_INTERVAL", kafkaRetryInterval), "Kafka retry interval in case of failure (seconds)")
	flag.IntVar(&healthPort, "health-port",
		LookupEnvOrInt("HEALTH_PORT", healthPort), "HTTP port for healthz and prometheus")
	flag.IntVar(&kafkaBatchSize, "kafka-batch-size",
		LookupEnvOrInt("KAFKA_BATCH_SIZE", kafkaBatchSize), "Kafka batch size")
	flag.IntVar(&kafkaMaxBatchSize, "kafka-max-batch-size",
		LookupEnvOrInt("KAFKA_MAX_BATCH_SIZE", kafkaMaxBatchSize), "Kafka MAX batch size (used when un-spooling after failure)")
	flag.IntVar(&kafkaInterval, "kafka-interval",
		LookupEnvOrInt("KAFKA_INTERVAL", kafkaInterval), "Kafka interval. How often a write is triggered (seconds)")
	flag.StringVar(&kafkaTestTopic, "kafka-test-topic",
		LookupEnvOrString("KAFKA_TEST_TOPIC", kafkaTestTopic), "Initial test message will be sent to this kafka topic. No initial test will be done if this isn't set.")
	flag.Parse()
	var logLevel log.LogLevel
	if logLevelStr != "" {
		logLevel, err = log.ParseLogLevel(logLevelStr)
		if err != nil {
			log.Fatalf("Invalid log level: %s", logLevelStr)
		}
		log.Infof("Setting log level to %s", logLevel.String())
		log.SetLevel(logLevel)
	}

	if mqttTls {
		CheckSet(caRootCertFile, "ROOT_CA", "tls is enabled")
		CheckSet(mqttCaClientCertFile, "MQTT_CLIENT_CERT", "tls is enabled")
		CheckSet(mqttCaClientKeyFile, "MQTT_CLIENT_KEY", "tls is enabled")
	}

	runConfig := bridge.Params{
		MqttBroker:         mqttBroker,
		MqttPort:           mqttPort,
		MqttTopic:          mqttTopic,
		MqttTls:            mqttTls,
		MqttClientId:       mqttClientId,
		TlsRootCrtFile:     caRootCertFile,
		MqttClientCertFile: mqttCaClientCertFile,
		MqttClientKeyFile:  mqttCaClientKeyFile,
		KafkaBroker:        kafkaBroker,
		KafkaPort:          kafkaPort,
		KafkaTopic:         kafkaTopic,
		KafkaRetryInterval: time.Duration(kafkaRetryInterval) * time.Second,
		KafkaInterval:      time.Duration(kafkaInterval) * time.Second,
		KafkaBatchSize:     kafkaBatchSize,
		KafkaMaxBatchSize:  kafkaMaxBatchSize,
		HealthPort:         healthPort,
		KafkaTestTopic:     kafkaTestTopic,
		LogLevel:           logLevel,
	}
	log.Infof("Startup options: %v", runConfig)
	log.Debug("Starting bridge")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		bridge.Run(ctx, runConfig)
	}()
	wg.Wait()
	log.Debug("Waiting over. Exiting.")

}

func CheckSet(s, name, reason string) {
	if s == "" {
		log.Fatalf("%s can't be empty when %s", name, reason)
	}
}

func LookupEnvOrString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func LookupEnvOrInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			log.Fatalf("LookupEnvOrInt[%s]: %v", key, err)
		}
		return v
	}
	return defaultVal
}
func LookupEnvOrBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		return strings.ToUpper(val) == "TRUE"
	}
	return defaultVal
}
