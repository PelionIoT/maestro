package sync

import (
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/logging"
    . "github.com/armPelionEdge/devicedb/rest"
    . "github.com/armPelionEdge/devicedb/partition"

    "io"
    "net/http"
    "strconv"
    "encoding/json"
    "github.com/gorilla/mux"
)

type BucketSyncHTTP struct {
    PartitionPool PartitionPool
    ClusterConfigController ClusterConfigController
}

func (bucketSync *BucketSyncHTTP) Attach(router *mux.Router) {
    router.HandleFunc("/sites/{siteID}/buckets/{bucket}/merkle", func(w http.ResponseWriter, r *http.Request) {
        siteID := mux.Vars(r)["siteID"]
        bucketName := mux.Vars(r)["bucket"]
        partitionNumber := bucketSync.ClusterConfigController.ClusterController().Partition(siteID)
        partition := bucketSync.PartitionPool.Get(partitionNumber)

        if partition == nil {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/merkle: Site does not exist at this node", siteID, bucketName)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(ESiteDoesNotExist.JSON()) + "\n")
            
            return
        }

        site := partition.Sites().Acquire(siteID)
        defer partition.Sites().Release(siteID)

        if site == nil {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/merkle: Site does not exist at this node", siteID, bucketName)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(ESiteDoesNotExist.JSON()) + "\n")
            
            return
        }

        if site.Buckets().Get(bucketName) == nil {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/merkle: Bucket does not exist at this site", siteID, bucketName)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(EBucketDoesNotExist.JSON()) + "\n")
            
            return
        }

        responseMerkleDepth := MerkleTree{
            Depth: site.Buckets().Get(bucketName).MerkleTree().Depth(),
        }

        body, _ := json.Marshal(&responseMerkleDepth)

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, string(body))
    }).Methods("GET")

    router.HandleFunc("/sites/{siteID}/buckets/{bucket}/merkle/nodes/{nodeID}/keys", func(w http.ResponseWriter, r *http.Request) {
        siteID := mux.Vars(r)["siteID"]
        bucketName := mux.Vars(r)["bucket"]
        partitionNumber := bucketSync.ClusterConfigController.ClusterController().Partition(siteID)
        partition := bucketSync.PartitionPool.Get(partitionNumber)

        if partition == nil {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/merkle: Site does not exist at this node", siteID, bucketName)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(ESiteDoesNotExist.JSON()) + "\n")
            
            return
        }

        site := partition.Sites().Acquire(siteID)
        defer partition.Sites().Release(siteID)

        if site == nil {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/merkle: Site does not exist at this node", siteID, bucketName)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(ESiteDoesNotExist.JSON()) + "\n")
            
            return
        }

        if site.Buckets().Get(bucketName) == nil {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/merkle: Bucket does not exist at this site", siteID, bucketName)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(EBucketDoesNotExist.JSON()) + "\n")
            
            return
        }

        nodeID, err := strconv.ParseUint(mux.Vars(r)["nodeID"], 10, 32)

        if err != nil {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/merkle/nodes/{nodeID}/keys: nodeID was not properly formatted", siteID, bucketName, mux.Vars(r)["nodeID"])
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EMerkleRange.JSON()) + "\n")
            
            return
        }

        siblingSetIter, err := site.Buckets().Get(bucketName).GetSyncChildren(uint32(nodeID))

        if err != nil {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/merkle/nodes/{nodeID}/keys: %v", siteID, bucketName, mux.Vars(r)["nodeID"], err.Error())

            var code int
            var body string

            if err == EMerkleRange {
                code = http.StatusBadRequest
                body = string(EMerkleRange.JSON())
            } else if err == EStorage {
                code = http.StatusInternalServerError
                body = string(EStorage.JSON())
            } else {
                code = http.StatusInternalServerError
                body = string(EStorage.JSON())
            }
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(code)
            io.WriteString(w, body + "\n")
            
            return
        }

        responseMerkleKeys := MerkleKeys{
            Keys: make([]Key, 0),
        }

        defer siblingSetIter.Release()

        for siblingSetIter.Next() {
            responseMerkleKeys.Keys = append(responseMerkleKeys.Keys, Key{
                Key: string(siblingSetIter.Key()),
                Value: siblingSetIter.Value(),
            })
        }

        if siblingSetIter.Error() != nil {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/merkle/nodes/{nodeID}/keys: Sibling set iterator error: %v", siteID, bucketName, mux.Vars(r)["nodeID"], siblingSetIter.Error())
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, string(EStorage.JSON()) + "\n")
            
            return
        }

        body, _ := json.Marshal(&responseMerkleKeys)

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, string(body))
    }).Methods("GET")

    router.HandleFunc("/sites/{siteID}/buckets/{bucket}/merkle/nodes/{nodeID}", func(w http.ResponseWriter, r *http.Request) {
        // Get the hash of a node
        siteID := mux.Vars(r)["siteID"]
        bucketName := mux.Vars(r)["bucket"]
        partitionNumber := bucketSync.ClusterConfigController.ClusterController().Partition(siteID)
        partition := bucketSync.PartitionPool.Get(partitionNumber)

        if partition == nil {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/merkle: Site does not exist at this node", siteID, bucketName)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(ESiteDoesNotExist.JSON()) + "\n")
            
            return
        }

        site := partition.Sites().Acquire(siteID)
        defer partition.Sites().Release(siteID)

        if site == nil {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/merkle: Site does not exist at this node", siteID, bucketName)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(ESiteDoesNotExist.JSON()) + "\n")
            
            return
        }

        if site.Buckets().Get(bucketName) == nil {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/merkle: Bucket does not exist at this site", siteID, bucketName)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(EBucketDoesNotExist.JSON()) + "\n")
            
            return
        }

        nodeID, err := strconv.ParseUint(mux.Vars(r)["nodeID"], 10, 32)

        if err != nil {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/merkle/nodes/{nodeID}/keys: nodeID was not properly formatted", siteID, bucketName, mux.Vars(r)["nodeID"])
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EMerkleRange.JSON()) + "\n")
            
            return
        }

        if nodeID >= uint64(site.Buckets().Get(bucketName).MerkleTree().NodeLimit()) {
            Log.Warningf("GET /sites/{siteID}/buckets/{bucket}/merkle/nodes/{nodeID}/keys: nodeID out of range", siteID, bucketName, mux.Vars(r)["nodeID"])
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EMerkleRange.JSON()) + "\n")
            
            return
        }

        nodeHash := site.Buckets().Get(bucketName).MerkleTree().NodeHash(uint32(nodeID))

        responseMerkleNodeHash := MerkleNode{
            Hash: nodeHash,
        }

        body, _ := json.Marshal(&responseMerkleNodeHash)

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, string(body))
    }).Methods("GET")
}