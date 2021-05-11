package main

import (
	"context"
	"flag"
	"github.com/celerway/metamorphosis/bridge"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	var (
		logLevel             string
		mqttBroker           string
		mqttPort             int = 8883
		mqttTopic            string
		mqttTls              bool = true
		caRootCertFile       string
		mqttCaClientCertFile string
		mqttCaClientKeyFile  string
		kafkaBroker          string
		kafkaPort            int = 9092
		kafkaTopic           string
		healthPort           int = 8080
		kafkaWorkers         int = 1
	)

	err := godotenv.Load()
	log.Info("Metamorphosis starting up.")
	if err != nil {
		log.Infof("Error loading .env file, assuming production: %s", err.Error())
	}

	flag.StringVar(&logLevel, "log-level",
		LookupEnvOrString("LOG_LEVEL", logLevel), "Log level (trace|debug|info|warn|error")
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
	flag.StringVar(&kafkaBroker, "kafka-broker",
		LookupEnvOrString("KAFKA_BROKER", kafkaBroker), "Kafka broker hostname")
	flag.IntVar(&kafkaPort, "kakfa-port",
		LookupEnvOrInt("KAFKA_PORT", kafkaPort), "Kafka broker port")
	flag.StringVar(&kafkaTopic, "kafka-topic",
		LookupEnvOrString("KAFKA_TOPIC", kafkaTopic), "Kafka topic to write to")
	flag.IntVar(&kafkaWorkers, "kafka-workers",
		LookupEnvOrInt("KAFKA_WORKERS", kafkaWorkers), "Kafka workers")
	flag.IntVar(&healthPort, "health-port",
		LookupEnvOrInt("HEALTH_PORT", healthPort), "HTTP port for healthz and prometheus")
	flag.Parse()

	setLoglevel(logLevel)

	if mqttTls {
		CheckSet(caRootCertFile, "ROOT_CA", "tls is enabled")
		CheckSet(mqttCaClientCertFile, "MQTT_CLIENT_CERT", "tls is enabled")
		CheckSet(mqttCaClientKeyFile, "MQTT_CLIENT_KEY", "tls is enabled")
	}

	runConfig := bridge.BridgeParams{
		MqttBroker:         mqttBroker,
		MqttPort:           mqttPort,
		MqttTopic:          mqttTopic,
		MqttTls:            mqttTls,
		TlsRootCrtFile:     caRootCertFile,
		MqttClientCertFile: mqttCaClientCertFile,
		MqttClientKeyFile:  mqttCaClientKeyFile,
		KafkaBroker:        kafkaBroker,
		KafkaPort:          kafkaPort,
		KafkaTopic:         kafkaTopic,
		KafkaWorkers:       kafkaWorkers,
		KafkaRetryInterval: 3 * time.Second,
		HealthPort:         healthPort,
	}
	log.Infof("Startup options: %v", runConfig)
	log.Debug("Starting bridge")
	bridge.Run(context.Background(), runConfig)

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

func setLoglevel(level string) {
	switch level {
	case "": // Default choice.
		log.SetLevel(log.InfoLevel)
	case "trace":
		log.SetLevel(log.TraceLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.Errorf("Unknown loglevel: %s", level)
		os.Exit(1)
	}
	log.Debugf("Log level set to %s", level)
}
