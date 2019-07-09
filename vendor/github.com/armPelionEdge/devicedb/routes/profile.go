package routes

import (
    "github.com/gorilla/mux"
    "net/http/pprof"
)

type ProfilerEndpoint struct {
}

func (profiler *ProfilerEndpoint) Attach(router *mux.Router) {
    router.HandleFunc("/debug/pprof/", pprof.Index)
    router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
    router.HandleFunc("/debug/pprof/profile", pprof.Profile)
    router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
}