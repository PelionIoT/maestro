package routes

import (
    "encoding/json"
    "github.com/gorilla/mux"
    "io"
    "io/ioutil"
    "net/http"
    "strconv"

    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/logging"
    . "github.com/armPelionEdge/devicedb/raft"
)

type ClusterEndpoint struct {
    ClusterFacade ClusterFacade
}

func (clusterEndpoint *ClusterEndpoint) Attach(router *mux.Router) {
    // Get an overview of the cluster
    router.HandleFunc("/cluster", func(w http.ResponseWriter, r *http.Request) {
        var clusterOverview ClusterOverview

        clusterOverview.Nodes = clusterEndpoint.ClusterFacade.ClusterNodes()
        clusterOverview.ClusterSettings = clusterEndpoint.ClusterFacade.ClusterSettings()
        clusterOverview.PartitionDistribution = clusterEndpoint.ClusterFacade.PartitionDistribution()
        clusterOverview.TokenAssignments = clusterEndpoint.ClusterFacade.TokenAssignments()

        encodedOverview, err := json.Marshal(clusterOverview)

        if err != nil {
            Log.Warningf("GET /cluster: %v", err)

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")
            
            return
        }

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, string(encodedOverview) + "\n")
    }).Methods("GET")

    router.HandleFunc("/cluster/nodes", func(w http.ResponseWriter, r *http.Request) {
        // Add a node to the cluster
        body, err := ioutil.ReadAll(r.Body)

        if err != nil {
            Log.Warningf("POST /cluster/nodes: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EReadBody.JSON()) + "\n")
            
            return
        }

        var nodeConfig NodeConfig

        if err := json.Unmarshal(body, &nodeConfig); err != nil {
            Log.Warningf("POST /cluster/nodes: Unable to parse node config body")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(ENodeConfigBody.JSON()) + "\n")
            
            return
        }

        if err := clusterEndpoint.ClusterFacade.AddNode(r.Context(), nodeConfig); err != nil {
            Log.Warningf("POST /cluster/nodes: Unable to add node to cluster: %v", err.Error())

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)

            if err == ECancelConfChange {
                io.WriteString(w, string(EDuplicateNodeID.JSON()) + "\n")
            } else {
                io.WriteString(w, string(EProposalError.JSON()) + "\n")
            }
            
            return
        }

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, "\n")
    }).Methods("POST")

    router.HandleFunc("/cluster/nodes/{nodeID}", func(w http.ResponseWriter, r *http.Request) {
        // Remove, replace, or deccommission a node
        query := r.URL.Query()
        _, wasForwarded := query["forwarded"]
        _, replace := query["replace"]
        _, decommission := query["decommission"]

        if replace && decommission {
            Log.Warningf("DELETE /cluster/nodes/{nodeID}: Both the replace and decommission query parameters are set. This is not allowed")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, "\n")
            
            return
        }

        nodeID, err := strconv.ParseUint(mux.Vars(r)["nodeID"], 10, 64)

        if err != nil {
            Log.Warningf("DELETE /cluster/nodes/{nodeID}: Invalid node ID")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, "\n")
            
            return
        }

        if nodeID == 0 {
            nodeID = clusterEndpoint.ClusterFacade.LocalNodeID()
        }

        if decommission {
            if nodeID == clusterEndpoint.ClusterFacade.LocalNodeID() {
                if err := clusterEndpoint.ClusterFacade.Decommission(); err != nil {
                    Log.Warningf("DELETE /cluster/nodes/%d: Encountered an error while putting the node into decomissioning mode: %v", clusterEndpoint.ClusterFacade.LocalNodeID(), err.Error())
                    
                    w.Header().Set("Content-Type", "application/json; charset=utf8")
                    w.WriteHeader(http.StatusInternalServerError)
                    io.WriteString(w, "\n")
                    
                    return
                }

                Log.Infof("DELETE /cluster/nodes/%d: Local node is in decommissioning mode", clusterEndpoint.ClusterFacade.LocalNodeID())

                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusOK)
                io.WriteString(w, "\n")

                return
            }

            if wasForwarded {
                Log.Warningf("DELETE /cluster/nodes/{nodeID}: Received a forwarded decommission request but we're not the correct node", clusterEndpoint.ClusterFacade.LocalNodeID())
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusForbidden)
                io.WriteString(w, "\n")
                
                return
            } 
            
            // forward the request to another node
            peerAddress := clusterEndpoint.ClusterFacade.PeerAddress(nodeID)

            if peerAddress.IsEmpty() {
                Log.Warningf("DELETE /cluster/nodes/{nodeID}: Unable to forward decommission request since this node doesn't know how to contact the decommissioned node", clusterEndpoint.ClusterFacade.LocalNodeID())
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusBadGateway)
                io.WriteString(w, "\n")
                
                return
            }

            err := clusterEndpoint.ClusterFacade.DecommissionPeer(nodeID)

            if err != nil {
                Log.Warningf("DELETE /cluster/nodes/{nodeID}: Error forwarding decommission request: %v", clusterEndpoint.ClusterFacade.LocalNodeID(), err.Error())
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusBadGateway)
                io.WriteString(w, "\n")
                
                return
            }

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusOK)
            io.WriteString(w, "\n")

            return
        }

        var replacementNodeID uint64

        if replace {
            replacementNodeID, err = strconv.ParseUint(query["replace"][0], 10, 64)

            if err != nil {
                Log.Warningf("DELETE /cluster/nodes/{nodeID}: Invalid replacement node ID")
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusBadRequest)
                io.WriteString(w, "\n")
                
                return
            }
        }

        if replacementNodeID != 0 {
            err = clusterEndpoint.ClusterFacade.ReplaceNode(r.Context(), nodeID, replacementNodeID)
        } else {
            err = clusterEndpoint.ClusterFacade.RemoveNode(r.Context(), nodeID)
        }

        if err != nil {
            Log.Warningf("DELETE /cluster/nodes/{nodeID}: Unable to remove node from the cluster: %v", err.Error())
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")
            
            return
        }

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, "\n")
    }).Methods("DELETE")
}