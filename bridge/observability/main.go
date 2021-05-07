package observability

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func (obs observability) mainloop() {
	log.Debug("Observability worker is running")
	for true {
		select {
		case msg := <-obs.channel:
			obs.handleChannelMessage(msg)
		}

	}
}

func Run(params ObservabilityParams) {
	obs := observability{
		channel: params.Channel,
	}

	obs.mqttReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mqtt_received",
		Help: "Number of received MQTT messages",
	})
	obs.mqttErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mqtt_errors",
		Help: "Number of erroneous MQTT messages",
	})
	obs.kafkaSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_sent",
		Help: "Number of sent Kafka messages",
	})
	obs.kafkaErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_errors",
		Help: "No of errors encountered with Kafka",
	})
	go obs.mainloop()
	go obs.httpStuff()
}

func (obs observability) httpStuff() {
	// We don't care about waitgroups and stuff here. We can be aborted at any time.
	router := mux.NewRouter().StrictSlash(true)
	router.Handle("/metrics", promhttp.Handler())
	router.HandleFunc("/healthz", SillyHealthzHandler)
	listenPort := ":8080"
	log.Infof("Observability service attempting to listen to port %s", listenPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s", listenPort), router))
}

func (obs observability) handleChannelMessage(msg StatusMessage) {
	log.Debugf("Observability received %s", msg)

	switch msg {
	case MqttRecieved:
		obs.mqttReceived.Inc()
	case MqttError:
		obs.mqttErrors.Inc()
	case KafkaSent:
		obs.kafkaSent.Inc()
	case KafkaError:
		obs.kafkaErrors.Inc()
	default:
		log.Errorf("Observability: Unknown message recived")
	}
}

func GetChannel() ObservabilityChannel {
	return make(ObservabilityChannel, 0)
}

func SillyHealthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(200)
	_, _ = w.Write([]byte("ok"))
}
