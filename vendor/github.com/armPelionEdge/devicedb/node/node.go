package node

import (
    "context"
    
    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/routes"
)

// A Node coordinates interactions between
// internal node components
type Node interface {
    // Start up the node. 
    // Case 1) This node is not yet part of a cluster 
    //   It will use the initialization options to figure out whether it should start a new cluster or join an existing one.
    // Case 2) This node is part of a cluster and the decomissioning flag is not set
    //   It should start up and resume its operations as a member of its cluster. Start will run until Stop is called,
    //   in which case it will return nil, or until the node is removed from the cluster in which case it returns ERemoved
    //   or EDecommissioned
    // Case 3) This node is part of a cluster and the decomissioning flag is set
    //   It should start up in decomissioning mode, allowing only operations
    //   which transfer its partitions to new owners. After it has been removed from the cluster
    //   Start returns EDecomissioned or ERemoved
    // EDecomissioned is returned when the node was removed from the cluster after successfully transferring away all its
    // data to other nodes in the cluster
    // ERemoved is returned when the node was removed from the cluster before successfully transferring away all its data
    // to other nodes in the cluster
    ID() uint64
    Start(options NodeInitializationOptions) error
    // Shut down the node
    Stop()
    Batch(ctx context.Context, partition uint64, siteID string, bucket string, updateBatch *UpdateBatch) (map[string]*SiblingSet, error)
    Merge(ctx context.Context, partition uint64, siteID string, bucket string, patch map[string]*SiblingSet, broadcastToRelays bool) error
    Get(ctx context.Context, partition uint64, siteID string, bucket string, keys [][]byte) ([]*SiblingSet, error)
    GetMatches(ctx context.Context, partition uint64, siteID string, bucket string, keys [][]byte) (SiblingSetIterator, error)
    RelayStatus(relayID string) (RelayStatus, error)
}
