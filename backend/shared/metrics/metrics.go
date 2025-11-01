package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns an http.Handler for Prometheus scraping
func Handler() http.Handler {
	return promhttp.Handler()
}
