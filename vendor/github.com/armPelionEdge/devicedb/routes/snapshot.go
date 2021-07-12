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
    "encoding/json"
    "github.com/gorilla/mux"
    "io"
    "net/http"

	. "github.com/armPelionEdge/devicedb/error"
	. "github.com/armPelionEdge/devicedb/logging"
)

type SnapshotEndpoint struct {
    ClusterFacade ClusterFacade
}

func (snapshotEndpoint *SnapshotEndpoint) Attach(router *mux.Router) {
    router.HandleFunc("/snapshot", func(w http.ResponseWriter, r *http.Request) {
        snapshot, err := snapshotEndpoint.ClusterFacade.ClusterSnapshot(r.Context())

        if err != nil {
            Log.Warningf("POST /snapshot: %v", err)

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, err.Error())
            
            return
		}

		snapshot.Status = SnapshotProcessing
		
        encodedSnapshot, err := json.Marshal(snapshot)

        if err != nil {
            Log.Warningf("POST /snapshot: %v", err)

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")
            
            return
        }

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, string(encodedSnapshot))
	}).Methods("POST")
	
	// This endpoint definition must come before the one for /snapshot/{snapshotId}.tar or it will be overridden
	router.HandleFunc("/snapshot/{snapshotId}.tar", func(w http.ResponseWriter, r *http.Request) {
		var snapshotId string = mux.Vars(r)["snapshotId"]

		err := snapshotEndpoint.ClusterFacade.CheckLocalSnapshotStatus(snapshotId)

		if err == ESnapshotInProgress {
            Log.Warningf("GET /snapshot/{snapshotId}: %v", err)
			
			w.Header().Set("Content-Type", "application/json; charset=utf8")
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, string(ESnapshotInProgress.JSON()))

			return
		} else if err == ESnapshotOpenFailed {
            Log.Warningf("GET /snapshot/{snapshotId}: %v", err)
			
			w.Header().Set("Content-Type", "application/json; charset=utf8")
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, string(ESnapshotOpenFailed.JSON()))

			return
		} else if err == ESnapshotReadFailed {
            Log.Warningf("GET /snapshot/{snapshotId}: %v", err)
			
			w.Header().Set("Content-Type", "application/json; charset=utf8")
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, string(ESnapshotReadFailed.JSON()))

			return
		} else if err != nil {
			Log.Warningf("GET /snapshot/{snapshotId}: %v", err)
			
			w.Header().Set("Content-Type", "application/json; charset=utf8")
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, string(EStorage.JSON()))

			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		err = snapshotEndpoint.ClusterFacade.WriteLocalSnapshot(snapshotId, w)

		if err != nil {
			Log.Errorf("GET /snapshot/{snapshotId}: %v", err)
		}
	}).Methods("GET")

	router.HandleFunc("/snapshot/{snapshotId}", func(w http.ResponseWriter, r *http.Request) {
		var snapshotId string = mux.Vars(r)["snapshotId"]
		var snapshot Snapshot = Snapshot{ UUID: snapshotId }
		
		err := snapshotEndpoint.ClusterFacade.CheckLocalSnapshotStatus(snapshotId)		

		if err == ESnapshotInProgress {
			snapshot.Status = SnapshotProcessing
		} else if err == ESnapshotOpenFailed {
			snapshot.Status = SnapshotMissing
		} else if err == ESnapshotReadFailed {
			snapshot.Status = SnapshotFailed
		} else if err != nil {
            Log.Warningf("GET /snapshot/{snapshotId}: %v", err)

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, err.Error())
            
            return
        } else {
			snapshot.Status = SnapshotComplete
		}

        encodedSnapshot, err := json.Marshal(snapshot)

        if err != nil {
            Log.Warningf("GET /snapshot/{snapshotId}: %v", err)

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")
            
            return
        }

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, string(encodedSnapshot))
	}).Methods("GET")
}