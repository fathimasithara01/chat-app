package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	Connections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ws_active_connections",
		Help: "Active websocket connections",
	})
)

func Init() {
	prometheus.MustRegister(Connections)
}
