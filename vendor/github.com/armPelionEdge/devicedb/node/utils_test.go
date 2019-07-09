package node_test

import (
    "context"

    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/node"
    . "github.com/armPelionEdge/devicedb/raft"
    . "github.com/armPelionEdge/devicedb/routes"

    "github.com/coreos/etcd/raft/raftpb"
)

type MockConfigController struct {
    clusterController *ClusterController
    defaultClusterCommandResponse error
    clusterCommandCB func(ctx context.Context, commandBody interface{})
}

func NewMockConfigController(clusterController *ClusterController) *MockConfigController {
    return &MockConfigController{
        clusterController: clusterController,
    }
}

func (configController *MockConfigController) LogDump() (raftpb.Snapshot, []raftpb.Entry, error) {
    return raftpb.Snapshot{}, nil, nil
}

func (configController *MockConfigController) AddNode(ctx context.Context, nodeConfig NodeConfig) error {
    return nil
}

func (configController *MockConfigController) ReplaceNode(ctx context.Context, replacedNodeID uint64, replacementNodeID uint64) error {
    return nil
}

func (configController *MockConfigController) RemoveNode(ctx context.Context, nodeID uint64) error {
    return nil
}

func (configController *MockConfigController) ClusterCommand(ctx context.Context, commandBody interface{}) error {
    configController.notifyClusterCommand(ctx, commandBody)
    return configController.defaultClusterCommandResponse
}

func (configController *MockConfigController) SetDefaultClusterCommandResponse(err error) {
    configController.defaultClusterCommandResponse = err
}

func (configController *MockConfigController) onClusterCommand(cb func(ctx context.Context, commandBody interface{})) {
    configController.clusterCommandCB = cb
}

func (configController *MockConfigController) notifyClusterCommand(ctx context.Context, commandBody interface{}) {
    configController.clusterCommandCB(ctx, commandBody)
}

func (configController *MockConfigController) OnLocalUpdates(cb func(deltas []ClusterStateDelta)) {
}

func (configController *MockConfigController) OnClusterSnapshot(cb func(snapshotIndex uint64, snapshotId string)) {
}

func (configController *MockConfigController) ClusterController() *ClusterController {
    return configController.clusterController
}

func (configController *MockConfigController) SetClusterController(clusterController *ClusterController) {
    configController.clusterController = clusterController
}

func (configController *MockConfigController) Start() error {
    return nil
}

func (configController *MockConfigController) Stop() {
}

func (configController *MockConfigController) CancelProposals() {
}

type MockClusterConfigControllerBuilder struct {
    defaultClusterConfigController ClusterConfigController
}

func NewMockClusterConfigControllerBuilder() *MockClusterConfigControllerBuilder {
    return &MockClusterConfigControllerBuilder{ }
}

func (builder *MockClusterConfigControllerBuilder) SetCreateNewCluster(b bool) ClusterConfigControllerBuilder {
    return builder
}

func (builder *MockClusterConfigControllerBuilder) SetLocalNodeAddress(peerAddress PeerAddress) ClusterConfigControllerBuilder {
    return builder
}

func (builder *MockClusterConfigControllerBuilder) SetRaftNodeStorage(raftStorage RaftNodeStorage) ClusterConfigControllerBuilder {
    return builder
}

func (builder *MockClusterConfigControllerBuilder) SetRaftNodeTransport(transport *TransportHub) ClusterConfigControllerBuilder {
    return builder
}

func (builder *MockClusterConfigControllerBuilder) Create() ClusterConfigController {
    return builder.defaultClusterConfigController
}

func (builder *MockClusterConfigControllerBuilder) SetDefaultConfigController(configController ClusterConfigController) {
    builder.defaultClusterConfigController = configController
}

type MockNode struct {
    id uint64
    batchCB func(ctx context.Context, partition uint64, siteID string, bucket string, updateBatch *UpdateBatch)
    mergeCB func(ctx context.Context, partition uint64, siteID string, bucket string, patch map[string]*SiblingSet, broadcastToRelays bool)
    defaultBatchPatch map[string]*SiblingSet
    defaultBatchError error
    defaultMergeError error
    getCB func(ctx context.Context, partition uint64, siteID string, bucket string, keys [][]byte)
    defaultGetSiblingSetArray []*SiblingSet
    defaultGetError error
    getMatchesCB func(ctx context.Context, partition uint64, siteID string, bucket string, keys [][]byte)
    defaultGetMatchesSiblingSetIterator SiblingSetIterator
    defaultGetMatchesError error
}

func NewMockNode(id uint64) *MockNode {
    return &MockNode{
        id: id,
    }
}

func (node *MockNode) ID() uint64 {
    return node.id
}

func (node *MockNode) Start(options NodeInitializationOptions) error {
    return nil
}

func (node *MockNode) Stop() {
}

func (node *MockNode) Batch(ctx context.Context, partition uint64, siteID string, bucket string, updateBatch *UpdateBatch) (map[string]*SiblingSet, error) {
    if node.batchCB != nil {
        node.batchCB(ctx, partition, siteID, bucket, updateBatch)
    }

    return node.defaultBatchPatch, node.defaultBatchError
}

func (node *MockNode) Merge(ctx context.Context, partition uint64, siteID string, bucket string, patch map[string]*SiblingSet, broadcastToRelays bool) error {
    if node.mergeCB != nil {
        node.mergeCB(ctx, partition, siteID, bucket, patch, broadcastToRelays)
    }

    return node.defaultMergeError
}

func (node *MockNode) Get(ctx context.Context, partition uint64, siteID string, bucket string, keys [][]byte) ([]*SiblingSet, error) {
    if node.getCB != nil {
        node.getCB(ctx, partition, siteID, bucket, keys)
    }

    return node.defaultGetSiblingSetArray, node.defaultGetError
}

func (node *MockNode) GetMatches(ctx context.Context, partition uint64, siteID string, bucket string, keys [][]byte) (SiblingSetIterator, error) {
    if node.getMatchesCB != nil {
        node.getMatchesCB(ctx, partition, siteID, bucket, keys)
    }

    return node.defaultGetMatchesSiblingSetIterator, node.defaultGetMatchesError
}
    
func (node *MockNode) RelayStatus(relayID string) (RelayStatus, error) {
    return RelayStatus{}, nil
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