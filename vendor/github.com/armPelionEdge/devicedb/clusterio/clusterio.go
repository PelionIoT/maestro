package clusterio

import (
    "context"

    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/routes"
)

type ClusterIOAgent interface {
    Merge(ctx context.Context, siteID string, bucket string, patch map[string]*SiblingSet) (replicas int, nApplied int, err error)
    Batch(ctx context.Context, siteID string, bucket string, updateBatch *UpdateBatch) (replicas int, nApplied int, err error)
    Get(ctx context.Context, siteID string, bucket string, keys [][]byte) ([]*SiblingSet, error)
    GetMatches(ctx context.Context, siteID string, bucket string, keys [][]byte) (SiblingSetIterator, error)
    RelayStatus(ctx context.Context, siteID string, relayID string) (RelayStatus, error)
    CancelAll()
}

type PartitionResolver interface {
    Partition(partitioningKey string) uint64
    ReplicaNodes(partition uint64) []uint64
}

type NodeClient interface {
    Merge(ctx context.Context, nodeID uint64, partition uint64, siteID string, bucket string, patch map[string]*SiblingSet, broadcastToRelays bool) error
    Batch(ctx context.Context, nodeID uint64, partition uint64, siteID string, bucket string, updateBatch *UpdateBatch) (map[string]*SiblingSet, error)
    Get(ctx context.Context, nodeID uint64, partition uint64, siteID string, bucket string, keys [][]byte) ([]*SiblingSet, error)
    GetMatches(ctx context.Context, nodeID uint64, partition uint64, siteID string, bucket string, keys [][]byte) (SiblingSetIterator, error)
    RelayStatus(ctx context.Context, nodeID uint64, siteID string, relayID string) (RelayStatus, error)
    LocalNodeID() uint64
}

type NodeReadMerger interface {
    // Add to the pool of replicas for this key
    InsertKeyReplica(nodeID uint64, key string, siblingSet *SiblingSet)
    // Get the merged set for this key
    Get(key string) *SiblingSet
    // Obtain a patch that needs to be merged into the specified node to bring it up to date
    // for any keys for which there are updates that it has not received
    Patch(nodeID uint64) map[string]*SiblingSet
    // Get a set of nodes involved in the read merger
    Nodes() map[uint64]bool
}

type NodeReadRepairer interface {
    BeginRepair(partition uint64, siteID string, bucket string, readMerger NodeReadMerger)
    StopRepairs()
}