package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ActiveClients = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tunnel_active_clients",
		Help: "Current number of active connected clients",
	})

	ActiveStreams = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tunnel_active_streams",
		Help: "Current number of active multiplexed streams",
	})

	BytesTransferred = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tunnel_bytes_transferred_total",
		Help: "Total bytes transferred over tunnel",
	}, []string{"direction"})

	AuthFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tunnel_auth_failures_total",
		Help: "Total number of failed authentication attempts",
	})
)

func Register() {
	prometheus.MustRegister(ActiveClients)
	prometheus.MustRegister(ActiveStreams)
	prometheus.MustRegister(BytesTransferred)
	prometheus.MustRegister(AuthFailures)
}

func Handler() http.Handler {
	return promhttp.Handler()
}
