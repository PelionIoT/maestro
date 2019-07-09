package routes

import (
	"github.com/gorilla/mux"
	"net/http"	
)

type KubernetesEndpoint struct {
}

func (kubernetesEndpoint *KubernetesEndpoint) Attach(router *mux.Router) {
	// For liveness and readiness probes
	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")
}