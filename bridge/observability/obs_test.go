package observability

import (
	"context"
	"fmt"
	is2 "github.com/matryer/is"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

const obsPort = 2000

type metricsMap map[string]float64

func Test_observability_Run(t *testing.T) {
	is := is2.New(t)
	ch := make(Channel)
	params := Params{
		Channel:    ch,
		HealthPort: obsPort,
	}
	obs := Initialize(params)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		obs.Run(ctx)
	}()
	time.Sleep(10 * time.Millisecond)
	metrics, err := getMetrics(obsPort)
	is.NoErr(err)
	metricNames := []string{"mqtt_received", "mqtt_errors", "kafka_sent", "kafka_errors", "kafka_state"}
	// all
	for _, name := range metricNames {
		is.Equal(metrics[name], float64(0))
	}
	// generate an error and see that kafka_state goes to 1
	ch <- KafkaError
	metrics, err = getMetrics(obsPort)
	is.NoErr(err)
	is.Equal(metrics["kafka_state"], float64(1))
	is.Equal(metrics["kafka_errors"], float64(1))
	// generate a message and see that kafka_state goes to 0
	ch <- KafkaSent
	metrics, err = getMetrics(obsPort)
	is.NoErr(err)
	is.Equal(metrics["kafka_state"], float64(0))
	is.Equal(metrics["kafka_sent"], float64(1))
	ch <- MattReceived
	metrics, err = getMetrics(obsPort)
	is.NoErr(err)
	is.Equal(metrics["mqtt_received"], float64(1))
	ch <- MqttError
	metrics, err = getMetrics(obsPort)
	is.NoErr(err)
	is.Equal(metrics["mqtt_errors"], float64(1))
	cancel()
	wg.Wait()
}

// getMetrics fetches the metrics from the /metrics endpoint and returns a map of metric name to value.
func getMetrics(port int) (metricsMap, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/metrics", port), nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	promMetrics, err := parseMF(resp.Body)
	if err != nil {
		return nil, err
	}
	metrics := make(metricsMap)
	for k, v := range promMetrics {
		if k == "kafka_state" {
			metrics[k] = float64(v.Metric[0].Gauge.GetValue())
		} else {
			metrics[k] = *v.Metric[0].Counter.Value
		}
	}
	return metrics, nil
}

func parseMF(reader io.Reader) (map[string]*dto.MetricFamily, error) {
	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		return nil, err
	}
	return mf, nil
}
