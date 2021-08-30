package observability

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
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

func Run(params ObservabilityParams) *observability {
	reg := prometheus.NewRegistry()

	obs := observability{
		channel:    params.Channel,
		logger:     log.WithFields(log.Fields{"module": "observability"}),
		healthPort: params.HealthPort,
		waitGroup:  params.WaitGroup,
		promReg:    reg,
	}

	obs.mqttReceived = promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "mqtt_received",
		Help: "Number of received MQTT messages",
	})
	obs.mqttErrors = promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "mqtt_errors",
		Help: "Number of erroneous MQTT messages",
	})
	obs.kafkaSent = promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "kafka_sent",
		Help: "Number of sent Kafka messages",
	})
	obs.kafkaErrors = promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "kafka_errors",
		Help: "No of errors encountered with Kafka",
	})
	log.Debug("Obs entering mainloop")
	go obs.mainloop()
	log.Debug("Obs starting http server")
	obs.runHttpServer()
	return &obs // Return the struct so the bridge can adjust the health status.
}

func (obs *observability) runHttpServer() {
	// We don't care about waitGroups and stuff here. We can be aborted at any time.
	router := mux.NewRouter().StrictSlash(true)
	router.Handle("/metrics", promhttp.HandlerFor(obs.promReg, promhttp.HandlerOpts{}))
	router.HandleFunc("/healthz", obs.HealthzHandler)
	listenPort := fmt.Sprintf(":%d", obs.healthPort)
	obs.logger.Infof("Observability service attempting to listen to port %s", listenPort)
	go func() {
		obs.waitGroup.Add(1)
		defer obs.waitGroup.Done()
		srv := &http.Server{ // http.Server has some mutexes, so we use a pointer.
			Addr:    fmt.Sprintf("%s", listenPort),
			Handler: router,
		}
		obs.srv = srv
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			obs.logger.Fatal(err)
		}
	}()
}

func (obs *observability) Shutdown() {
	obs.logger.Info("De-registering prometheus counters")
	obs.promReg.Unregister(obs.mqttReceived) // During testing we run multiple bridges in the same binary.
	obs.promReg.Unregister(obs.mqttErrors)   // So we must make sure that these don't collide.
	obs.promReg.Unregister(obs.kafkaSent)
	obs.promReg.Unregister(obs.kafkaSent)

	obs.logger.Info("Shutting down obs HTTP server.")
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := obs.srv.Shutdown(timeoutCtx); err != nil {
		panic(err) // failure/timeout shutting down the server gracefully
	}
}

func (obs observability) handleChannelMessage(msg StatusMessage) {
	obs.logger.Tracef("Observability received %s", msg)

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
		obs.logger.Errorf("Observability: Unknown message recived")
	}
}

func GetChannel(size int) ObservabilityChannel {
	return make(ObservabilityChannel, size) //
}

func (obs *observability) HealthzHandler(w http.ResponseWriter, _ *http.Request) {
	if obs.ready {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	} else {
		w.WriteHeader(423)
		_, _ = w.Write([]byte("not ready"))
	}
}

func (obs *observability) Ready() {
	obs.ready = true
}
