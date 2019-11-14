package telemetry

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	histogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method_name", "service_name"},
	)
)

// PromTransaction ...
type PromTransaction struct {
	Obs   prometheus.ObserverVec
	Start time.Time
}

// StartTransaction ...
func StartTransaction(transactionName, serviceName string) (txn PromTransaction) {
	txn = PromTransaction{GetHistogram().MustCurryWith(prometheus.Labels{"method_name": transactionName, "service_name": serviceName}), time.Now()}
	return
}

// EndTransaction ...
func EndTransaction(txn PromTransaction) {
	txn.Obs.With(prometheus.Labels{}).Observe(time.Since(txn.Start).Seconds())
}

// RegisterPromtheus handle /metrics
func RegisterPromtheus() {
	http.Handle("/metrics", promhttp.Handler())
}

// GetHistogram ...
func GetHistogram() (responseSize *prometheus.HistogramVec) {
	return histogram
}
