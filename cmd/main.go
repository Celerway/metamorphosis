package main

import (
	"flag"
	"github.com/celerway/metamorphosis/bridge"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
)

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

func setOptionStr(paramPtr *string, defaultValue, name, env string) string {
	var ret string
	if *paramPtr == "" {
		ret = os.Getenv(env)
	} else {
		ret = *paramPtr
	}
	if ret == "" {
		ret = defaultValue
	}
	if ret == "" {
		log.Errorf("Mandatory option %s not given in ENV{%s} or by flag", name, env)
	}
	log.Tracef("Option '%s' set to '%s'", name, ret)
	return ret
}
func setOptionInt(paramPtr *int, defaultValue int, name, env string) int {
	var ret int
	var err error
	if *paramPtr == 0 {
		val, ok := os.LookupEnv(env)
		if ok {
			ret, err = strconv.Atoi(val)
			if err != nil {
				log.Fatalf("Could not make sense of ENV{%s}: %s", env, os.Getenv(env))
			}
		}
	} else {
		ret = *paramPtr
	}
	if ret == 0 {
		ret = defaultValue
	}
	if ret == 0 {
		log.Fatalf("Mandatory option %s not given in ENV{%s} or by flag", name, env)
	}
	log.Tracef("Option '%s' set to %d", name, ret)
	return ret
}
func setOptionBool(paramPtr *bool, defaultValue bool, name, env string) bool {
	var ret bool
	if *paramPtr == false {
		val, ok := os.LookupEnv(env)

		if ok && strings.ToUpper(val) == "TRUE" {
			ret = true
		}
	} else {
		ret = *paramPtr
	}
	if ret == false {
		ret = defaultValue
	}
	log.Tracef("Option '%s' is set to '%v'", name, ret)
	return ret
}

func main() {
	err := godotenv.Load()
	log.Info("Metamorphosis starting up.")
	if err != nil {
		log.Infof("Error loading .env file, assuming production: %s", err.Error())
	}

	logLevelPtr := flag.String("loglevel", "", "Log level (trace|debug|info|warn|error")
	caRootCertFilePtr := flag.String("ca", "", "Path to root CA certificate (pubkey)")
	caClientCertFilePtr := flag.String("client-cert", "", "Path to client cert (pubkey)")
	caClientKeyFilePtr := flag.String("client-key", "", "Path to client key (privkey)")
	noTlsPtr := flag.Bool("mqtt-no-tls", false, "Disable TLS for MQTT")
	mqttBrokerPtr := flag.String("mqtt-broker", "", "MQTT broker hostname")
	mqttPortPtr := flag.Int("mqtt-port", 0, "Mqtt broker port.")
	mqttTopicPtr := flag.String("mqtt-topic", "", "MQTT topic to listen to")
	kafkaBrokerPtr := flag.String("kafka-broker", "", "Kafka broker hostname")
	kafkaPortPtr := flag.Int("kakfa-port", 0, "Kafka broker port")
	kafkaTopicPtr := flag.String("kafka-topic", "", "Kafka topic to write to")

	flag.Parse()
	logLevel := setOptionStr(logLevelPtr, "info", "log level", "LOG_LEVEL")
	setLoglevel(logLevel)

	Tls := !setOptionBool(noTlsPtr, false, "no TLS", "MQTT_NO_TLS") // Notice the logical flip.
	var (
		caRootCertFile, caClientCertFile, caClientKeyFile string
	)
	if Tls {
		caRootCertFile = setOptionStr(caRootCertFilePtr, "", "Root CA Cert", "ROOT_CA")
		caClientCertFile = setOptionStr(caClientCertFilePtr, "", "Client TLS Cert", "CLIENT_CERT")
		caClientKeyFile = setOptionStr(caClientKeyFilePtr, "", "Client TLS key", "CLIENT_KEY")
	}
	mqttBroker := setOptionStr(mqttBrokerPtr, "", "mqtt broker", "MQTT_BROKER")
	mqttPort := setOptionInt(mqttPortPtr, 8883, "mqtt port", "MQTT_PORT")
	mqttTopic := setOptionStr(mqttTopicPtr, "", "mqtt topic", "MQTT_TOPIC")

	kafkaBroker := setOptionStr(kafkaBrokerPtr, "", "kafka broker", "KAFKA_BROKER")
	kafkaPort := setOptionInt(kafkaPortPtr, 9092, "kafka port", "KAFKA_PORT")
	kafkaTopic := setOptionStr(kafkaTopicPtr, "", "kafka topic", "KAFKA_TOPIC")

	runConfig := bridge.BridgeParams{
		MqttBroker:     mqttBroker,
		MqttPort:       mqttPort,
		MqttTopic:      mqttTopic,
		Tls:            Tls,
		TlsRootCrtFile: caRootCertFile,
		ClientCertFile: caClientCertFile,
		ClientKeyFile:  caClientKeyFile,
		KafkaBroker:    kafkaBroker,
		KafkaPort:      kafkaPort,
		KafkaTopic:     kafkaTopic,
	}
	log.Debug("Starting bridge")
	bridge.Run(runConfig)

}
