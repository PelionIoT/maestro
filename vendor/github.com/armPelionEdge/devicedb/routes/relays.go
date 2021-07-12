package routes
//
 // Copyright (c) 2019 ARM Limited.
 //
 // SPDX-License-Identifier: MIT
 //
 // Permission is hereby granted, free of charge, to any person obtaining a copy
 // of this software and associated documentation files (the "Software"), to
 // deal in the Software without restriction, including without limitation the
 // rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
 // sell copies of the Software, and to permit persons to whom the Software is
 // furnished to do so, subject to the following conditions:
 //
 // The above copyright notice and this permission notice shall be included in all
 // copies or substantial portions of the Software.
 //
 // THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 // IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 // FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 // AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 // LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 // OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 // SOFTWARE.
 //


import (
    "github.com/gorilla/mux"
    "encoding/json"
    "io"
    "io/ioutil"
    "net/http"

    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/logging"
)

type RelaysEndpoint struct {
    ClusterFacade ClusterFacade
}

func (relaysEndpoint *RelaysEndpoint) Attach(router *mux.Router) {
    router.HandleFunc("/relays/{relayID}", func(w http.ResponseWriter, r *http.Request) {
        body, err := ioutil.ReadAll(r.Body)

        if err != nil {
            Log.Warningf("PATCH /relays/{relayID}: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EReadBody.JSON()) + "\n")
            
            return
        }

        var relayPatch RelaySettingsPatch

        if err := json.Unmarshal(body, &relayPatch); err != nil {
            Log.Warningf("PATCH /relays/{relayID}: Unable to parse relay settings body")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EReadBody.JSON()) + "\n")
            
            return
        }

        err = relaysEndpoint.ClusterFacade.MoveRelay(r.Context(), mux.Vars(r)["relayID"], relayPatch.Site)

        if err == ENoSuchRelay {
            Log.Warningf("PATCH /relays/{relayID}: Relay does not exist")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(ERelayDoesNotExist.JSON()) + "\n")
            
            return
        }

        if err == ENoSuchSite {
            Log.Warningf("PATCH /relays/{relayID}: Site does not exist")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(ESiteDoesNotExist.JSON()) + "\n")
            
            return
        }

        if err != nil {
            Log.Warningf("PATCH /relays/{relayID}: %v", err.Error())
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")
            
            return
        }

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, "\n")
    }).Methods("PATCH")

    // Add relay or move it to a site
    router.HandleFunc("/relays/{relayID}", func(w http.ResponseWriter, r *http.Request) {
        err := relaysEndpoint.ClusterFacade.AddRelay(r.Context(), mux.Vars(r)["relayID"])

        if err != nil {
            Log.Warningf("PUT /relays/{relayID}: %v", err.Error())
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")
            
            return
        }

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, "\n")
    }).Methods("PUT")

    // Remove a relay and disassociate it from a site
    router.HandleFunc("/relays/{relayID}", func(w http.ResponseWriter, r *http.Request) {
        err := relaysEndpoint.ClusterFacade.RemoveRelay(r.Context(), mux.Vars(r)["relayID"])

        if err != nil {
            Log.Warningf("DELETE /relays/{relayID}: %v", err.Error())
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")
            
            return
        }
        
        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, "\n")
    }).Methods("DELETE")

    // Get the status of a relay
    router.HandleFunc("/relays/{relayID}", func(w http.ResponseWriter, r *http.Request) {
        query := r.URL.Query()
        _, local := query["local"]

        var relayStatus RelayStatus
        var err error

        if local {
            relayStatus, err = relaysEndpoint.ClusterFacade.LocalGetRelayStatus(mux.Vars(r)["relayID"])
        } else {
            relayStatus, err = relaysEndpoint.ClusterFacade.GetRelayStatus(r.Context(), mux.Vars(r)["relayID"])
        }

        if err == ERelayDoesNotExist {
            Log.Warningf("GET /relays/{relayID}: %v", err.Error())
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(ERelayDoesNotExist.JSON()) + "\n")
            
            return
        }

        if err != nil {
            Log.Warningf("GET /relays/{relayID}: %v", err.Error())
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")
            
            return
        }
        
        encodedStatus, err := json.Marshal(relayStatus)

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, string(encodedStatus) + "\n")
    }).Methods("GET")
}