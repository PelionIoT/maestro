package routes

import (
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "net/http"

    . "github.com/armPelionEdge/devicedb/logging"
)

type SyncEndpoint struct {
    ClusterFacade ClusterFacade
    Upgrader websocket.Upgrader
}

func (syncEndpoint *SyncEndpoint) Attach(router *mux.Router) {
    router.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
        conn, err := syncEndpoint.Upgrader.Upgrade(w, r, nil)
        
        if err != nil {
            Log.Errorf("Unable to upgrade connection: %v", err.Error())

            return
        }
        
        syncEndpoint.ClusterFacade.AcceptRelayConnection(conn, r.Header)
    }).Methods("GET")
}