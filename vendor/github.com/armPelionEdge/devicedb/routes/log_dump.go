package routes

import (
    "encoding/json"
    "github.com/gorilla/mux"
    "io"
    "net/http"

    . "github.com/armPelionEdge/devicedb/logging"
)

type LogDumpEndpoint struct {
    ClusterFacade ClusterFacade
}

func (logDumpEndpoint *LogDumpEndpoint) Attach(router *mux.Router) {
    router.HandleFunc("/log_dump", func(w http.ResponseWriter, r *http.Request) {
        logDump, err := logDumpEndpoint.ClusterFacade.LocalLogDump()

        if err != nil {
            Log.Warningf("GET /log_dump: %v", err)

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")
            
            return
        }

        encodedLogDump, err := json.Marshal(logDump)

        if err != nil {
            Log.Warningf("GET /log_dump: %v", err)

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")
            
            return
        }

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, string(encodedLogDump) + "\n")
    }).Methods("GET")
}