package metrics

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	once sync.Once

	ActiveClients = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tunnel_active_clients",
		Help: "Number of active connected clients",
	})

	ActiveStreams = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tunnel_active_streams",
		Help: "Number of active multiplexed streams",
	})

	BytesTransferred = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tunnel_bytes_transferred_total",
		Help: "Total bytes transferred over tunnel",
	}, []string{"direction"})

	AuthFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tunnel_auth_failures_total",
		Help: "Total number of failed authentication attempts",
	})

	RateLimitTriggers = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tunnel_rate_limit_triggers_total",
		Help: "Total number of connections dropped due to rate limiting",
	})

	ConnectionDrops = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tunnel_connection_drops_total",
		Help: "Total number of unexpected connection drops during stream transfer",
	})
)

func Register() {
	once.Do(func() {
		prometheus.MustRegister(ActiveClients)
		prometheus.MustRegister(ActiveStreams)
		prometheus.MustRegister(BytesTransferred)
		prometheus.MustRegister(AuthFailures)
		prometheus.MustRegister(RateLimitTriggers)
		prometheus.MustRegister(ConnectionDrops)
	})
}

func Handler() http.Handler {
	return promhttp.Handler()
}
