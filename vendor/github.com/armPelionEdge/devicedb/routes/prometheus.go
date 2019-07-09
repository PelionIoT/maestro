package routes

import (
    "github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusEndpoint struct {
}

func (prometheusEndpoint *PrometheusEndpoint) Attach(router *mux.Router) {
	router.Handle("/metrics", promhttp.Handler())
}