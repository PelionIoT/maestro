package routes_test

import (
    "context"
    "github.com/gorilla/websocket"
    "io"
    "net/http"

    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/raft"
    . "github.com/armPelionEdge/devicedb/routes"
)

type MockClusterFacade struct {
    defaultAddNodeResponse error
    defaultRemoveNodeResponse error
    defaultReplaceNodeResponse error
    defaultDecommissionPeerResponse error
    defaultDecommissionResponse error
    localNodeID uint64
    defaultPeerAddress PeerAddress
    defaultAddRelayResponse error
    defaultRemoveRelayResponse error
    defaultMoveRelayResponse error
    defaultAddSiteResponse error
    defaultRemoveSiteResponse error
    defaultBatchResponse BatchResult
    defaultBatchError error
    defaultLocalBatchPatch map[string]*SiblingSet
    defaultLocalBatchError error
    defaultLocalMergeResponse error
    defaultGetResponse []*SiblingSet
    defaultGetResponseError error
    defaultLocalGetResponse []*SiblingSet
    defaultLocalGetResponseError error
    defaultGetMatchesResponse SiblingSetIterator
    defaultGetMatchesResponseError error
    defaultLocalGetMatchesResponse SiblingSetIterator
    defaultLocalGetMatchesResponseError error
    defaultLocalLogDumpResponse LogDump
    defaultLocalLogDumpError error
    defaultLocalSnapshotResponse Snapshot
    defaultLocalSnapshotError error
    addNodeCB func(ctx context.Context, nodeConfig NodeConfig)
    replaceNodeCB func(ctx context.Context, nodeID uint64, replacementNodeID uint64)
    removeNodeCB func(ctx context.Context, nodeID uint64)
    decommisionCB func()
    decommisionPeerCB func(nodeID uint64)
    batchCB func(siteID string, bucket string, updateBatch *UpdateBatch)
    getCB func(siteID string, bucket string, keys [][]byte)
    getMatchesCB func(siteID string, bucket string, keys [][]byte)
    localBatchCB func(partition uint64, siteID string, bucket string, updateBatch *UpdateBatch)
    localMergeCB func(partition uint64, siteID string, bucket string, patch map[string]*SiblingSet, broadcastToRelays bool)
    localGetCB func(partition uint64, siteID string, bucket string, keys [][]byte)
    localGetMatchesCB func(partition uint64, siteID string, bucket string, keys [][]byte)
    addRelayCB func(ctx context.Context, relayID string)
    removeRelayCB func(ctx context.Context, relayID string)
    moveRelayCB func(ctx context.Context, relayID string, siteID string)
    addSiteCB func(ctx context.Context, siteID string)
    removeSiteCB func(ctx context.Context, siteID string)
    acceptRelayConnectionCB func(conn *websocket.Conn)
}

func (clusterFacade *MockClusterFacade) AddNode(ctx context.Context, nodeConfig NodeConfig) error {
    if clusterFacade.addNodeCB != nil {
        clusterFacade.addNodeCB(ctx, nodeConfig)
    }

    return clusterFacade.defaultAddNodeResponse
}

func (clusterFacade *MockClusterFacade) RemoveNode(ctx context.Context, nodeID uint64) error {
    if clusterFacade.removeNodeCB != nil {
        clusterFacade.removeNodeCB(ctx, nodeID)
    }

    return clusterFacade.defaultRemoveNodeResponse
}

func (clusterFacade *MockClusterFacade) ReplaceNode(ctx context.Context, nodeID uint64, replacementNodeID uint64) error {
    if clusterFacade.replaceNodeCB != nil {
        clusterFacade.replaceNodeCB(ctx, nodeID, replacementNodeID)
    }

    return clusterFacade.defaultReplaceNodeResponse
}

func (clusterFacade *MockClusterFacade) Decommission() error {
    if clusterFacade.decommisionCB != nil {
        clusterFacade.decommisionCB()
    }

    return clusterFacade.defaultDecommissionResponse
}

func (clusterFacade *MockClusterFacade) DecommissionPeer(nodeID uint64) error {
    if clusterFacade.decommisionPeerCB != nil {
        clusterFacade.decommisionPeerCB(nodeID)
    }

    return clusterFacade.defaultDecommissionPeerResponse
}

func (clusterFacade *MockClusterFacade) LocalNodeID() uint64 {
    return clusterFacade.localNodeID
}

func (clusterFacade *MockClusterFacade) PeerAddress(nodeID uint64) PeerAddress {
    return clusterFacade.defaultPeerAddress
}

func (clusterFacade *MockClusterFacade) AddRelay(ctx context.Context, relayID string) error {
    if clusterFacade.addRelayCB != nil {
        clusterFacade.addRelayCB(ctx, relayID)
    }

    return clusterFacade.defaultAddRelayResponse
}

func (clusterFacade *MockClusterFacade) RemoveRelay(ctx context.Context, relayID string) error {
    if clusterFacade.removeRelayCB != nil {
        clusterFacade.removeRelayCB(ctx, relayID)
    }

    return clusterFacade.defaultRemoveRelayResponse
}

func (clusterFacade *MockClusterFacade) MoveRelay(ctx context.Context, relayID string, siteID string) error {
    if clusterFacade.moveRelayCB != nil {
        clusterFacade.moveRelayCB(ctx, relayID, siteID)
    }

    return clusterFacade.defaultMoveRelayResponse
}

func (clusterFacade *MockClusterFacade) AddSite(ctx context.Context, siteID string) error {
    if clusterFacade.addSiteCB != nil {
        clusterFacade.addSiteCB(ctx, siteID)
    }

    return clusterFacade.defaultAddSiteResponse
}

func (clusterFacade *MockClusterFacade) RemoveSite(ctx context.Context, siteID string) error {
    if clusterFacade.removeSiteCB != nil {
        clusterFacade.removeSiteCB(ctx, siteID)
    }

    return clusterFacade.defaultRemoveSiteResponse
}

func (clusterFacade *MockClusterFacade) Batch(siteID string, bucket string, updateBatch *UpdateBatch) (BatchResult, error) {
    if clusterFacade.batchCB != nil {
        clusterFacade.batchCB(siteID, bucket, updateBatch)
    }

    return clusterFacade.defaultBatchResponse, clusterFacade.defaultBatchError
}

func (clusterFacade *MockClusterFacade) LocalBatch(partition uint64, siteID string, bucket string, updateBatch *UpdateBatch) (map[string]*SiblingSet, error) {
    if clusterFacade.localBatchCB != nil {
        clusterFacade.localBatchCB(partition, siteID, bucket, updateBatch)
    }

    return clusterFacade.defaultLocalBatchPatch, clusterFacade.defaultLocalBatchError
}

func (clusterFacade *MockClusterFacade) LocalMerge(partition uint64, siteID string, bucket string, patch map[string]*SiblingSet, broadcastToRelays bool) error {
    if clusterFacade.localMergeCB != nil {
        clusterFacade.localMergeCB(partition, siteID, bucket, patch, broadcastToRelays)
    }

    return clusterFacade.defaultLocalMergeResponse
}

func (clusterFacade *MockClusterFacade) Get(siteID string, bucket string, keys [][]byte) ([]*SiblingSet, error) {
    if clusterFacade.getCB != nil {
        clusterFacade.getCB(siteID, bucket, keys)
    }

    return clusterFacade.defaultGetResponse, clusterFacade.defaultGetResponseError
}

func (clusterFacade *MockClusterFacade) LocalGet(partition uint64, siteID string, bucket string, keys [][]byte) ([]*SiblingSet, error) {
    if clusterFacade.localGetCB != nil {
        clusterFacade.localGetCB(partition, siteID, bucket, keys)
    }

    return clusterFacade.defaultLocalGetResponse, clusterFacade.defaultLocalGetResponseError
}

func (clusterFacade *MockClusterFacade) GetMatches(siteID string, bucket string, keys [][]byte) (SiblingSetIterator, error) {
    if clusterFacade.getMatchesCB != nil {
        clusterFacade.getMatchesCB(siteID, bucket, keys)
    }

    return clusterFacade.defaultGetMatchesResponse, clusterFacade.defaultGetMatchesResponseError
}

func (clusterFacade *MockClusterFacade) LocalGetMatches(partition uint64, siteID string, bucket string, keys [][]byte) (SiblingSetIterator, error) {
    if clusterFacade.localGetMatchesCB != nil {
        clusterFacade.localGetMatchesCB(partition, siteID, bucket, keys)
    }

    return clusterFacade.defaultLocalGetMatchesResponse, clusterFacade.defaultLocalGetMatchesResponseError
}

func (clusterFacade *MockClusterFacade) AcceptRelayConnection(conn *websocket.Conn, header http.Header) {
    if clusterFacade.acceptRelayConnectionCB != nil {
        clusterFacade.acceptRelayConnectionCB(conn)
    }
}

func (clusterFacade *MockClusterFacade) ClusterNodes() []NodeConfig {
    return nil
}

func (clusterFacade *MockClusterFacade) ClusterSettings() ClusterSettings {
    return ClusterSettings{}
}

func (clusterFacade *MockClusterFacade) PartitionDistribution() [][]uint64 {
    return nil
}

func (clusterFacade *MockClusterFacade) TokenAssignments() []uint64 {
    return nil
}

func (clusterFacade *MockClusterFacade) GetRelayStatus(ctx context.Context, relayID string) (RelayStatus, error) {
    return RelayStatus{}, nil
}

func (clusterFacade *MockClusterFacade) LocalGetRelayStatus(relayID string) (RelayStatus, error) {
    return RelayStatus{}, nil
}

func (clusterFacade *MockClusterFacade) LocalLogDump() (LogDump, error) {
    return clusterFacade.defaultLocalLogDumpResponse, clusterFacade.defaultLocalLogDumpError
}

func (clusterFacade *MockClusterFacade) ClusterSnapshot(ctx context.Context) (Snapshot, error) {
    return clusterFacade.defaultLocalSnapshotResponse, clusterFacade.defaultLocalSnapshotError
}

func (clusterFacade *MockClusterFacade) CheckLocalSnapshotStatus(snapshotId string) error {
    return nil
}

func (clusterFacade *MockClusterFacade) WriteLocalSnapshot(snapshotId string, w io.Writer) error {
    return nil
}

type siblingSetIteratorEntry struct {
    Prefix []byte
    Key []byte
    Value *SiblingSet
    Error error
}

type MemorySiblingSetIterator struct {
    entries []*siblingSetIteratorEntry
    nextEntry *siblingSetIteratorEntry
}

func NewMemorySiblingSetIterator() *MemorySiblingSetIterator {
    return &MemorySiblingSetIterator{
        entries: make([]*siblingSetIteratorEntry, 0),
    }
}

func (iter *MemorySiblingSetIterator) AppendNext(prefix []byte, key []byte, value *SiblingSet, err error) {
    iter.entries = append(iter.entries, &siblingSetIteratorEntry{
        Prefix: prefix,
        Key: key,
        Value: value,
        Error: err,
    })
}

func (iter *MemorySiblingSetIterator) Next() bool {
    iter.nextEntry = nil

    if len(iter.entries) == 0 {
        return false
    }

    iter.nextEntry = iter.entries[0]
    iter.entries = iter.entries[1:]

    if iter.nextEntry.Error != nil {
        return false
    }

    return true
}

func (iter *MemorySiblingSetIterator) Prefix() []byte {
    return iter.nextEntry.Prefix
}

func (iter *MemorySiblingSetIterator) Key() []byte {
    return iter.nextEntry.Key
}

func (iter *MemorySiblingSetIterator) Value() *SiblingSet {
    return iter.nextEntry.Value
}

func (iter *MemorySiblingSetIterator) LocalVersion() uint64 {
    return 0
}

func (iter *MemorySiblingSetIterator) Release() {
}

func (iter *MemorySiblingSetIterator) Error() error {
    if iter.nextEntry == nil {
        return nil
    }

    return iter.nextEntry.Error
}