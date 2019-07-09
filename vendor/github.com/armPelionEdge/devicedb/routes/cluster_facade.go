package routes

import (
    "context"
    "github.com/gorilla/websocket"
    "io"
    "net/http"

    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/raft"
)

type ClusterFacade interface {
    AddNode(ctx context.Context, nodeConfig NodeConfig) error
    RemoveNode(ctx context.Context, nodeID uint64) error
    ReplaceNode(ctx context.Context, nodeID uint64, replacementNodeID uint64) error
    DecommissionPeer(nodeID uint64) error
    Decommission() error
    LocalNodeID() uint64
    PeerAddress(nodeID uint64) PeerAddress
    AddRelay(ctx context.Context, relayID string) error
    RemoveRelay(ctx context.Context, relayID string) error
    MoveRelay(ctx context.Context, relayID string, siteID string) error
    AddSite(ctx context.Context, siteID string) error
    RemoveSite(ctx context.Context, siteID string) error
    Batch(siteID string, bucket string, updateBatch *UpdateBatch) (BatchResult, error)
    LocalBatch(partition uint64, siteID string, bucket string, updateBatch *UpdateBatch) (map[string]*SiblingSet, error)
    LocalMerge(partition uint64, siteID string, bucket string, patch map[string]*SiblingSet, broadcastToRelays bool) error
    Get(siteID string, bucket string, keys [][]byte) ([]*SiblingSet, error)
    LocalGet(partition uint64, siteID string, bucket string, keys [][]byte) ([]*SiblingSet, error)
    GetMatches(siteID string, bucket string, keys [][]byte) (SiblingSetIterator, error)
    LocalGetMatches(partition uint64, siteID string, bucket string, keys [][]byte) (SiblingSetIterator, error)
    AcceptRelayConnection(conn *websocket.Conn, header http.Header)
    ClusterNodes() []NodeConfig
    ClusterSettings() ClusterSettings
    PartitionDistribution() [][]uint64
    TokenAssignments() []uint64
    GetRelayStatus(ctx context.Context, relayID string) (RelayStatus, error)
    LocalGetRelayStatus(relayID string) (RelayStatus, error)
    LocalLogDump() (LogDump, error)
    ClusterSnapshot(ctx context.Context) (Snapshot, error)
    CheckLocalSnapshotStatus(snapshotId string) error
    WriteLocalSnapshot(snapshotId string, w io.Writer) error
}