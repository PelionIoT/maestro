package routes

import (
    "encoding/json"
    "github.com/gorilla/mux"
    "io"
    "net/http"
    "strconv"

    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/logging"
)

type PartitionsEndpoint struct {
    ClusterFacade ClusterFacade
}

func (partitionsEndpoint *PartitionsEndpoint) Attach(router *mux.Router) {
    // Merge values into a bucket
    router.HandleFunc("/partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/merges", func(w http.ResponseWriter, r *http.Request) {
        var patch map[string]*SiblingSet
        var err error
        var decoder *json.Decoder = json.NewDecoder(r.Body)
        var broadcast bool = r.URL.Query().Get("broadcast") != ""

        err = decoder.Decode(&patch)

        if err != nil {
            Log.Warningf("POST /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/merges: Unable to parse request body: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, "\n")
            
            return
        }

        partitionID, err := strconv.ParseUint(mux.Vars(r)["partitionID"], 10, 64)

        if err != nil {
            Log.Warningf("POST /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/merges: Unable to parse partition ID as uint64: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, "\n")
            
            return
        }

        var siteID string = mux.Vars(r)["siteID"]
        var bucket string = mux.Vars(r)["bucketID"]

        err = partitionsEndpoint.ClusterFacade.LocalMerge(partitionID, siteID, bucket, patch, broadcast)

        if err == ENoSuchPartition || err == ENoSuchSite || err == ENoSuchBucket {
            var responseBody string

            switch err {
            case ENoSuchSite:
                responseBody = string(ESiteDoesNotExist.JSON())
            case ENoSuchBucket:
                responseBody = string(EBucketDoesNotExist.JSON())
            }

            Log.Warningf("POST /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/merges: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, responseBody + "\n")
            
            return
        }

        if err != nil && err != ENoQuorum {
            Log.Warningf("POST /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/merges: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")
            
            return
        }

        var batchResult BatchResult
        batchResult.NApplied = 1
        
        if err == ENoQuorum {
            batchResult.NApplied = 0
        }

        encodedBatchResult, _ := json.Marshal(batchResult)

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, string(encodedBatchResult) + "\n")
    }).Methods("POST")

    // Submit an update to a bucket
    router.HandleFunc("/partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/batches", func(w http.ResponseWriter, r *http.Request) {
        var updateBatch UpdateBatch
        var err error

        err = updateBatch.FromJSON(r.Body)

        if err != nil {
            Log.Warningf("POST /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/batches: Unable to parse request body: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, "\n")
            
            return
        }

        partitionID, err := strconv.ParseUint(mux.Vars(r)["partitionID"], 10, 64)

        if err != nil {
            Log.Warningf("POST /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/batches: Unable to parse partition ID as uint64: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, "\n")
            
            return
        }

        var siteID string = mux.Vars(r)["siteID"]
        var bucket string = mux.Vars(r)["bucketID"]

        patch, err := partitionsEndpoint.ClusterFacade.LocalBatch(partitionID, siteID, bucket, &updateBatch)

        if err == ENoSuchPartition || err == ENoSuchSite || err == ENoSuchBucket {
            var responseBody string

            switch err {
            case ENoSuchSite:
                responseBody = string(ESiteDoesNotExist.JSON())
            case ENoSuchBucket:
                responseBody = string(EBucketDoesNotExist.JSON())
            }

            Log.Warningf("POST /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/batches: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, responseBody + "\n")
            
            return
        }

        if err != nil && err != ENoQuorum {
            Log.Warningf("POST /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/batches: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")
            
            return
        }

        var batchResult BatchResult
        batchResult.NApplied = 1
        
        if err == ENoQuorum {
            batchResult.NApplied = 0
        } else {
            batchResult.Patch = patch
        }

        encodedBatchResult, _ := json.Marshal(batchResult)

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, string(encodedBatchResult) + "\n")
    }).Methods("POST")

    // Query keys in bucket
    router.HandleFunc("/partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/keys", func(w http.ResponseWriter, r *http.Request) {
        query := r.URL.Query()
        keys := query["key"]
        prefixes := query["prefix"]
        partitionID, err := strconv.ParseUint(mux.Vars(r)["partitionID"], 10, 64)

        if err != nil {
            Log.Warningf("GET /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/keys: Unable to parse partition ID as uint64: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, "\n")
            
            return
        }

        if len(keys) != 0 && len(prefixes) != 0 {
            Log.Warningf("GET /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/keys: Client specified both prefixes and keys in the same request")

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, "\n")
            
            return
        }

        if len(keys) == 0 && len(prefixes) == 0 {
            var entries []InternalEntry = []InternalEntry{ }
            encodedEntries, _ := json.Marshal(entries)

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusOK)
            io.WriteString(w, string(encodedEntries) + "\n")
            
            return
        }

        var siteID string = mux.Vars(r)["siteID"]
        var bucket string = mux.Vars(r)["bucketID"]

        if len(keys) > 0 {
            var byteKeys [][]byte = make([][]byte, len(keys))

            for i, key := range keys {
                byteKeys[i] = []byte(key)
            }

            siblingSets, err := partitionsEndpoint.ClusterFacade.LocalGet(partitionID, siteID, bucket, byteKeys)

            if err == ENoSuchPartition || err == ENoSuchBucket || err == ENoSuchSite {
                var responseBody string

                switch err {
                case ENoSuchBucket:
                    responseBody = string(EBucketDoesNotExist.JSON())
                case ENoSuchSite:
                    responseBody = string(ESiteDoesNotExist.JSON())
                }

                Log.Warningf("POST /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/keys: %v", err)
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusNotFound)
                io.WriteString(w, responseBody + "\n")

                return
            }

            if err != nil {
                Log.Warningf("GET /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/keys: %v", err.Error())

                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusInternalServerError)
                io.WriteString(w, "\n")
                
                return
            }

            var entries []InternalEntry = make([]InternalEntry, len(siblingSets))

            for i, key := range keys {
                entries[i] = InternalEntry{
                    Prefix: "",
                    Key: key,
                    Siblings: siblingSets[i],
                }
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

            ssIterator, err := partitionsEndpoint.ClusterFacade.LocalGetMatches(partitionID, siteID, bucket, byteKeys)

            if err == ENoSuchPartition || err == ENoSuchBucket || err == ENoSuchSite {
                var responseBody string

                switch err {
                case ENoSuchBucket:
                    responseBody = string(EBucketDoesNotExist.JSON())
                case ENoSuchSite:
                    responseBody = string(ESiteDoesNotExist.JSON())
                }

                Log.Warningf("POST /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/keys: %v", err)
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusNotFound)
                io.WriteString(w, responseBody + "\n")

                return
            }

            if err != nil {
                Log.Warningf("GET /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/keys: %v", err.Error())

                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusInternalServerError)
                io.WriteString(w, "\n")
                
                return
            }

            var entries []InternalEntry = make([]InternalEntry, 0)

            for ssIterator.Next() {
                var nextEntry InternalEntry = InternalEntry{
                    Prefix: string(ssIterator.Prefix()),
                    Key: string(ssIterator.Key()),
                    Siblings: ssIterator.Value(),
                }

                entries = append(entries, nextEntry)
            }

            if ssIterator.Error() != nil {
                Log.Warningf("GET /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/keys: %v", ssIterator.Error().Error())

                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusInternalServerError)
                io.WriteString(w, "\n")
                
                return
            }

            encodedEntries, _ := json.Marshal(entries)

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusOK)
            io.WriteString(w, string(encodedEntries) + "\n")

            return
        }
    }).Methods("GET")
}