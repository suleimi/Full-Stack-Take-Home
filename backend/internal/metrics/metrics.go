package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var requestCounter = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "http_request_total",
	Help: "Total Number of requests",
})

// newAppCollectors s where the non system metrics for the application are registered
func newAppCollectors() []prometheus.Collector {
	return []prometheus.Collector{requestCounter}
}

func NewAppMetricHandler() http.Handler {
	c := newAppCollectors()
	c = append(c, collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	registry := prometheus.NewRegistry()
	registry.MustRegister(c...)

	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

func IncreaseRequestCounter() {
	requestCounter.Inc()
}
