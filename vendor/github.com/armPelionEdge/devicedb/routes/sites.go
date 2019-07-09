package routes

import (
    "encoding/json"
    "io"
    "io/ioutil"
    "github.com/gorilla/mux"
    "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"

    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/logging"
    . "github.com/armPelionEdge/devicedb/transport"
)

var (
    prometheusRequestDurations = prometheus.NewHistogramVec(prometheus.HistogramOpts{
        Namespace: "sites",
        Subsystem: "devicedb",
        Name: "request_durations_seconds",
        Help: "The duration of each request",
        Buckets: []float64{ 0.05, 0.25, 0.45, 0.65, 0.85, 1.05, 1.25, 1.45, 1.65, 1.85, 5, 10 },
    }, []string{
        "handler",
        "code",
    })

    prometheusRequestCounts = prometheus.NewCounterVec(prometheus.CounterOpts{
        Namespace: "sites",
        Subsystem: "devicedb",
        Name: "request_counts",
        Help: "The number of requests",
    }, []string{
        "handler",
        "code",
    })
)

func init() {
    prometheus.MustRegister(prometheusRequestDurations, prometheusRequestCounts)
}

type SitesEndpoint struct {
    ClusterFacade ClusterFacade
}

func (sitesEndpoint *SitesEndpoint) Attach(outerRouter *mux.Router) {
    var router *mux.Router = mux.NewRouter()

    outerRouter.PathPrefix("/sites/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var routeMatch mux.RouteMatch

        if router.Match(r, &routeMatch) {
            var labels prometheus.Labels = prometheus.Labels{
                "handler": routeMatch.Route.GetName(),
            }

            promhttp.InstrumentHandlerDuration(prometheusRequestDurations.MustCurryWith(labels),
                promhttp.InstrumentHandlerCounter(prometheusRequestCounts.MustCurryWith(labels), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { 
                    router.ServeHTTP(w, r)
                })),
            )(w, r)

            return
        }

        router.ServeHTTP(w, r)
    })

    // Add a site
    router.HandleFunc("/sites/{siteID}", func(w http.ResponseWriter, r *http.Request) {
        err := sitesEndpoint.ClusterFacade.AddSite(r.Context(), mux.Vars(r)["siteID"])

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
    }).Methods("PUT").Name("add_site")

    // Remove a site
    router.HandleFunc("/sites/{siteID}", func(w http.ResponseWriter, r *http.Request) {
        err := sitesEndpoint.ClusterFacade.RemoveSite(r.Context(), mux.Vars(r)["siteID"])

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
    }).Methods("DELETE").Name("remove_site")

    // Submit an update to a bucket
    router.HandleFunc("/sites/{siteID}/buckets/{bucket}/batches", func(w http.ResponseWriter, r *http.Request) {
        body, err := ioutil.ReadAll(r.Body)

        if err != nil {
            Log.Warningf("POST /sites/{siteID}/buckets/{bucket}/batches: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EReadBody.JSON()) + "\n")
            
            return
        }

        var transportBatch TransportUpdateBatch

        if err := json.Unmarshal(body, &transportBatch); err != nil {
            Log.Warningf("POST /sites/{siteID}/buckets/{bucket}/batches: Unable to parse update batch")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EReadBody.JSON()) + "\n")
            
            return
        }

        var updateBatch UpdateBatch

        err = transportBatch.ToUpdateBatch(&updateBatch)

        if err != nil {
            Log.Warningf("POST /sites/{siteID}/buckets/{bucket}/batches: Invalid update batch")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EReadBody.JSON()) + "\n")
            
            return
        }

        batchResult, err := sitesEndpoint.ClusterFacade.Batch(mux.Vars(r)["siteID"], mux.Vars(r)["bucket"], &updateBatch)

        if err == ENoSuchSite {
            Log.Warningf("POST /sites/{siteID}/buckets/{bucket}/batches: Site does not exist")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(ESiteDoesNotExist.JSON()) + "\n")
            
            return
        }

        if err == ENoSuchBucket {
            Log.Warningf("POST /sites/{siteID}/buckets/{bucket}/batches: Site does not exist")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(EBucketDoesNotExist.JSON()) + "\n")
            
            return
        }

        batchResult.Quorum = true
        
        if err == ENoQuorum {
            Log.Warningf("POST /sites/{siteID}/buckets/{bucket}/batches: Write failed at some replicas")
            batchResult.Quorum = false
        } else if err != nil {
            Log.Warningf("POST /sites/{siteID}/buckets/{bucket}/batches: Site does not exist")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, string(EStorage.JSON()) + "\n")
            
            return
        }

        encodedBatchResult, _ := json.Marshal(batchResult)

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, string(encodedBatchResult) + "\n")
    }).Methods("POST").Name("update_bucket")

    // Query keys in bucket
    router.HandleFunc("/sites/{siteID}/buckets/{bucket}/keys", func(w http.ResponseWriter, r *http.Request) {
        //sitesEndpoint.ClusterFacade.Get(siteID, bucket, keys)
        //sitesEndpoint.ClusterFacade.GetMatches(siteID, bucket, keys)
        //Returns APIEntry
        query := r.URL.Query()
        keys := query["key"]
        prefixes := query["prefix"]

        if len(keys) != 0 && len(prefixes) != 0 {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucketID}/keys: Client specified both prefixes and keys in the same request")

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, "\n")
            
            return
        }

        if len(keys) == 0 && len(prefixes) == 0 {
            var entries []APIEntry = []APIEntry{ }
            encodedEntries, _ := json.Marshal(entries)

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusOK)
            io.WriteString(w, string(encodedEntries) + "\n")
            
            return
        }

        var siteID string = mux.Vars(r)["siteID"]
        var bucket string = mux.Vars(r)["bucket"]

        if len(keys) > 0 {
            var byteKeys [][]byte = make([][]byte, len(keys))

            for i, key := range keys {
                byteKeys[i] = []byte(key)
            }

            siblingSets, err := sitesEndpoint.ClusterFacade.Get(siteID, bucket, byteKeys)

            if err == ENoSuchSite {
                Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/keys: Site does not exist")
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusNotFound)
                io.WriteString(w, string(ESiteDoesNotExist.JSON()) + "\n")
                
                return
            }

            if err == ENoSuchBucket {
                Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/keys: Bucket does not exist")
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusNotFound)
                io.WriteString(w, string(EBucketDoesNotExist.JSON()) + "\n")
                
                return
            }

            if err == ENoQuorum {
                Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/keys: Read quorum could not be established")
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusInternalServerError)
                io.WriteString(w, string(ENoQuorum.JSON()) + "\n")
                
                return
            }

            if err != nil {
                Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/keys: %v", err.Error())
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusInternalServerError)
                io.WriteString(w, string(EStorage.JSON()) + "\n")
                
                return
            }

            var entries []APIEntry = make([]APIEntry, len(siblingSets))

            for i, key := range keys {
                internalEntry := InternalEntry{
                    Prefix: "",
                    Key: key,
                    Siblings: siblingSets[i],
                }

                entries[i] = *internalEntry.ToAPIEntry()
            }

            encodedEntries, _ := json.Marshal(entries)

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusOK)
            io.WriteString(w, string(encodedEntries) + "\n")

            return
        }

        if len(prefixes) > 0 {
            var byteKeys [][]byte = make([][]byte, len(prefixes))

            for i, key := range prefixes {
                byteKeys[i] = []byte(key)
            }

            ssIterator, err := sitesEndpoint.ClusterFacade.GetMatches(siteID, bucket, byteKeys)

            if err == ENoSuchSite {
                Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/keys: Site does not exist")
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusNotFound)
                io.WriteString(w, string(ESiteDoesNotExist.JSON()) + "\n")
                
                return
            }

            if err == ENoSuchBucket {
                Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/keys: Bucket does not exist")
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusNotFound)
                io.WriteString(w, string(EBucketDoesNotExist.JSON()) + "\n")
                
                return
            }

            if err == ENoQuorum {
                Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/keys: Read quorum could not be established")
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusInternalServerError)
                io.WriteString(w, string(ENoQuorum.JSON()) + "\n")
                
                return
            }

            if err != nil {
                Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/keys: %v", err.Error())
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusInternalServerError)
                io.WriteString(w, string(EStorage.JSON()) + "\n")
                
                return
            }

            var entries []APIEntry = make([]APIEntry, 0)

            for ssIterator.Next() {
                if ssIterator.Value().IsTombstoneSet() {
                    continue
                }

                internalEntry := InternalEntry{
                    Prefix: string(ssIterator.Prefix()),
                    Key: string(ssIterator.Key()),
                    Siblings: ssIterator.Value(),
                }

                entries = append(entries, *internalEntry.ToAPIEntry())
            }

            if ssIterator.Error() != nil {
                Log.Warningf("GET /sites/{siteID}/buckets/{bucketID}/keys: %v", ssIterator.Error().Error())

                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusInternalServerError)
                io.WriteString(w, string(EStorage.JSON()) + "\n")
                
                return
            }

            encodedEntries, _ := json.Marshal(entries)

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusOK)
            io.WriteString(w, string(encodedEntries) + "\n")

            return
        }
    }).Methods("GET").Name("read_bucket")
}