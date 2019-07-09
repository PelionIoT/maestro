package transfer_test

import (
    "context"
    "errors"
    "fmt"
    "io"
    "net"
    "net/http"

    . "github.com/armPelionEdge/devicedb/transfer"
    . "github.com/armPelionEdge/devicedb/site"
    . "github.com/armPelionEdge/devicedb/partition"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/merkle"
    . "github.com/armPelionEdge/devicedb/bucket"

    "github.com/coreos/etcd/raft/raftpb"
)

type MockPartitionTransferFactory struct {
    createIncomingTransferResponses []PartitionTransfer
    createOutgoingTransferResponses []PartitionTransfer
    createIncomingTransferCB func(reader io.Reader)
    createOutgoingTransferCB func(partition Partition)
    createIncomingTransferCalls int
    createOutgoingTransferCalls int
    defaultIncomingTransferResponse PartitionTransfer
}

func NewMockPartitionTransferFactory() *MockPartitionTransferFactory {
    return &MockPartitionTransferFactory{
        createIncomingTransferResponses: make([]PartitionTransfer, 0),
        createOutgoingTransferResponses: make([]PartitionTransfer, 0),
    }
}

func (transferFactory *MockPartitionTransferFactory) onCreateIncomingTransfer(cb func(reader io.Reader)) {
    transferFactory.createIncomingTransferCB = cb
}

func (transferFactory *MockPartitionTransferFactory) notifyCreateIncomingTransfer(reader io.Reader) {
    if transferFactory.createIncomingTransferCB == nil {
        return
    }

    transferFactory.createIncomingTransferCB(reader)
}

func (transferFactory *MockPartitionTransferFactory) CreateIncomingTransfer(reader io.Reader) PartitionTransfer {
    transferFactory.createIncomingTransferCalls++
    transferFactory.notifyCreateIncomingTransfer(reader)

    if len(transferFactory.createIncomingTransferResponses) == 0 {
        return transferFactory.defaultIncomingTransferResponse
    }

    response := transferFactory.createIncomingTransferResponses[0]
    transferFactory.createIncomingTransferResponses = transferFactory.createIncomingTransferResponses[1:]
    
    return response
}

func (transferFactory *MockPartitionTransferFactory) SetDefaultIncomingTransfer(incomingTransfer PartitionTransfer) {
    transferFactory.defaultIncomingTransferResponse = incomingTransfer
}

func (transferFactory *MockPartitionTransferFactory) AppendNextIncomingTransfer(partitionTransfer PartitionTransfer) {
    transferFactory.createIncomingTransferResponses = append(transferFactory.createIncomingTransferResponses, partitionTransfer)
}

func (transferFactory *MockPartitionTransferFactory) onCreateOutgoingTransfer(cb func(partition Partition)) {
    transferFactory.createOutgoingTransferCB = cb
}

func (transferFactory *MockPartitionTransferFactory) notifyCreateOutgoingTransfer(partition Partition) {
    if transferFactory.createOutgoingTransferCB == nil {
        return
    }

    transferFactory.createOutgoingTransferCB(partition)
}

func (transferFactory *MockPartitionTransferFactory) CreateOutgoingTransfer(partition Partition) (PartitionTransfer, error) {
    transferFactory.createOutgoingTransferCalls++
    transferFactory.notifyCreateOutgoingTransfer(partition)

    if len(transferFactory.createOutgoingTransferResponses) == 0 {
        return nil, nil
    }

    response := transferFactory.createOutgoingTransferResponses[0]
    transferFactory.createOutgoingTransferResponses = transferFactory.createOutgoingTransferResponses[1:]
    
    return response, nil
}

func (transferFactory *MockPartitionTransferFactory) AppendNextOutgoingTransfer(partitionTransfer PartitionTransfer) {
    transferFactory.createOutgoingTransferResponses = append(transferFactory.createOutgoingTransferResponses, partitionTransfer)
}

type mockNextChunkResponse struct {
    chunk PartitionChunk
    err error
}

type MockPartitionTransfer struct {
    nextChunkCalls int
    cancelCalls int
    defaultResponse mockNextChunkResponse
    expectedResponses []mockNextChunkResponse
    cancelCB func()
}

func NewMockPartitionTransfer() *MockPartitionTransfer {
    return &MockPartitionTransfer{
        nextChunkCalls: 0,
        cancelCalls: 0,
        expectedResponses: make([]mockNextChunkResponse, 0),
    }
}

func (mockPartitionTransfer *MockPartitionTransfer) AppendNextChunkResponse(chunk PartitionChunk, err error) *MockPartitionTransfer {
    mockPartitionTransfer.expectedResponses = append(mockPartitionTransfer.expectedResponses, mockNextChunkResponse{ chunk: chunk, err: err })

    return mockPartitionTransfer
}

func (mockPartitionTransfer *MockPartitionTransfer) SetDefaultNextChunkResponse(chunk PartitionChunk, err error) *MockPartitionTransfer {
    mockPartitionTransfer.defaultResponse = mockNextChunkResponse{ chunk: chunk, err: err }

    return mockPartitionTransfer
}

func (mockPartitionTransfer *MockPartitionTransfer) NextChunkCallCount() int {
    return mockPartitionTransfer.nextChunkCalls
}

func (mockPartitionTransfer *MockPartitionTransfer) CancelCallCount() int {
    return mockPartitionTransfer.cancelCalls
}

func (mockPartitionTransfer *MockPartitionTransfer) UseFilter(entryFilter EntryFilter) {
}

func (mockPartitionTransfer *MockPartitionTransfer) NextChunk() (PartitionChunk, error) {
    mockPartitionTransfer.nextChunkCalls++

    if len(mockPartitionTransfer.expectedResponses) == 0 {
        return mockPartitionTransfer.defaultResponse.chunk, mockPartitionTransfer.defaultResponse.err
    }

    nextResponse := mockPartitionTransfer.expectedResponses[0]
    mockPartitionTransfer.expectedResponses = mockPartitionTransfer.expectedResponses[1:]

    return nextResponse.chunk, nextResponse.err
}

func (mockPartitionTransfer *MockPartitionTransfer) onCancel(cb func()) {
    mockPartitionTransfer.cancelCB = cb
}

func (mockPartitionTransfer *MockPartitionTransfer) notifyCancel() {
    if mockPartitionTransfer.cancelCB == nil {
        return
    }

    mockPartitionTransfer.cancelCB()
}

func (mockPartitionTransfer *MockPartitionTransfer) Cancel() {
    mockPartitionTransfer.cancelCalls++
    mockPartitionTransfer.notifyCancel()
    mockPartitionTransfer.defaultResponse.chunk = PartitionChunk{ }
    mockPartitionTransfer.defaultResponse.err = ETransferCancelled
}

type MockBucket struct {
    name string
    mergeResponses []error
    mergeCalls int
    mergeCB func(siblingSets map[string]*SiblingSet)
    defaultMergeResponse error
}

func NewMockBucket(name string) *MockBucket {
    return &MockBucket{
        mergeResponses: make([]error, 0),
        name: name,
        defaultMergeResponse: errors.New("No responses"),
    }
}

func (bucket *MockBucket) Name() string {
    return bucket.name
}

func (bucket *MockBucket) ShouldReplicateOutgoing(peerID string) bool {
    return false
}

func (bucket *MockBucket) ShouldReplicateIncoming(peerID string) bool {
    return false
}

func (bucket *MockBucket) ShouldAcceptWrites(clientID string) bool {
    return false
}

func (bucket *MockBucket) ShouldAcceptReads(clientID string) bool {
    return false
}

func (bucket *MockBucket) RecordMetadata() error {
    return nil
}

func (bucket *MockBucket) RebuildMerkleLeafs() error {
    return nil
}

func (bucket *MockBucket) MerkleTree() *MerkleTree {
    return nil
}

func (bucket *MockBucket) GarbageCollect(tombstonePurgeAge uint64) error {
    return nil
}

func (bucket *MockBucket) Get(keys [][]byte) ([]*SiblingSet, error) {
    return nil, nil
}

func (bucket *MockBucket) GetAll() (SiblingSetIterator, error) {
    return nil, nil
}

func (bucket *MockBucket) Watch(ctx context.Context, keys [][]byte, prefixes [][]byte, localVersion uint64, ch chan Row) {

}

func (bucket *MockBucket) LockReads() {
}

func (bucket *MockBucket) UnlockReads() {
}

func (bucket *MockBucket) LockWrites() {
}

func (bucket *MockBucket) UnlockWrites() {
}

func (bucket *MockBucket) GetMatches(keys [][]byte) (SiblingSetIterator, error) {
    return nil, nil
}

func (bucket *MockBucket) GetSyncChildren(nodeID uint32) (SiblingSetIterator, error) {
    return nil, nil
}

func (bucket *MockBucket) Forget(keys [][]byte) error {
    return nil
}

func (bucket *MockBucket) Batch(batch *UpdateBatch) (map[string]*SiblingSet, error) {
    return nil, nil
}

func (bucket *MockBucket) Merge(siblingSets map[string]*SiblingSet) error {
    bucket.mergeCalls++
    bucket.notifyMerge(siblingSets)

    if len(bucket.mergeResponses) == 0 {
        return bucket.defaultMergeResponse
    }

    result := bucket.mergeResponses[0]
    bucket.mergeResponses = bucket.mergeResponses[1:]

    return result
}

func (bucket *MockBucket) MergeCallCount() int {
    return bucket.mergeCalls
}

func (bucket *MockBucket) onMerge(cb func(siblingSets map[string]*SiblingSet)) {
    bucket.mergeCB = cb
}

func (bucket *MockBucket) notifyMerge(siblingSets map[string]*SiblingSet) {
    if bucket.mergeCB == nil {
        return
    }

    bucket.mergeCB(siblingSets)
}

func (bucket *MockBucket) SetDefaultMergeResponse(err error) {
    bucket.defaultMergeResponse = err
}

func (bucket *MockBucket) AppendNextMergeResponse(err error) {
    bucket.mergeResponses = append(bucket.mergeResponses, err)
}

type MockSite struct {
    buckets *BucketList
}

func NewMockSite(buckets *BucketList) *MockSite {
    return &MockSite{
        buckets: buckets,
    }
}

func (site *MockSite) ID() string {
    return ""
}

func (site *MockSite) LockWrites() {
}

func (site *MockSite) UnlockWrites() {
}

func (site *MockSite) LockReads() {
}

func (site *MockSite) UnlockReads() {
}

func (site *MockSite) Iterator() SiteIterator {
    return nil
}

func (site *MockSite) Buckets() *BucketList {
    return site.buckets
}

type MockSitePool struct {
    acquireResponses []Site
    acquireCB func(siteID string)
    acquireCalls int
    defaultAcquireResponse Site
}

func NewMockSitePool() *MockSitePool {
    return &MockSitePool{
        acquireResponses: make([]Site, 0),
    }
}

func (sitePool *MockSitePool) Iterator() SitePoolIterator {
    return nil
}

func (sitePool *MockSitePool) LockWrites() {
}

func (sitePool *MockSitePool) UnlockWrites() {
}

func (sitePool *MockSitePool) LockReads() {
}

func (sitePool *MockSitePool) UnlockReads() {
}

func (sitePool *MockSitePool) Acquire(siteID string) Site {
    sitePool.acquireCalls++
    sitePool.notifyAcquire(siteID)

    if len(sitePool.acquireResponses) == 0 {
        return sitePool.defaultAcquireResponse
    }

    result := sitePool.acquireResponses[0]
    sitePool.acquireResponses = sitePool.acquireResponses[1:]

    return result
}

func (sitePool *MockSitePool) AcquireCallCount() int {
    return sitePool.acquireCalls
}

func (sitePool *MockSitePool) onAcquire(cb func(siteID string)) {
    sitePool.acquireCB = cb
}

func (sitePool *MockSitePool) notifyAcquire(siteID string) {
    if sitePool.acquireCB == nil {
        return
    }

    sitePool.acquireCB(siteID)
}

func (sitePool *MockSitePool) SetDefaultAcquireResponse(site Site) {
    sitePool.defaultAcquireResponse = site
}

func (sitePool *MockSitePool) AppendNextAcquireResponse(site Site) {
    sitePool.acquireResponses = append(sitePool.acquireResponses, site)
}

func (sitePool *MockSitePool) Release(siteID string) {
}

func (sitePool *MockSitePool) Add(siteID string) {
}

func (sitePool *MockSitePool) Remove(siteID string) {
}

type MockPartition struct {
    partition uint64
    replica uint64
    sitePool SitePool
    iterator *MockPartitionIterator
}

func NewMockPartition(partition uint64, replica uint64) *MockPartition {
    return &MockPartition{
        partition: partition,
        replica: replica,
        iterator: NewMockPartitionIterator(),
    }
}

func (partition *MockPartition) Partition() uint64 {
    return partition.partition
}

func (partition *MockPartition) Replica() uint64 {
    return partition.replica
}

func (partition *MockPartition) Sites() SitePool {
    return partition.sitePool
}

func (partition *MockPartition) SetSites(sites SitePool) {
    partition.sitePool = sites
}

func (partition *MockPartition) Iterator() PartitionIterator {
    return partition.iterator
}

func (partition *MockPartition) MockIterator() *MockPartitionIterator {
    return partition.iterator
}

func (partition *MockPartition) LockWrites() {
}

func (partition *MockPartition) UnlockWrites() {
}

func (partition *MockPartition) LockReads() {
}

func (partition *MockPartition) UnlockReads() {
}

type MockPartitionPool struct {
    partitions map[uint64]Partition
    onGetCB func(partitionNumber uint64)
    getCalls int
}

func NewMockPartitionPool() *MockPartitionPool {
    return &MockPartitionPool{
        partitions: make(map[uint64]Partition),
    }
}

func (partitionPool *MockPartitionPool) Add(partition Partition) {
    partitionPool.partitions[partition.Partition()] = partition
}

func (partitionPool *MockPartitionPool) Remove(partitionNumber uint64) {
    delete(partitionPool.partitions, partitionNumber)
}

func (partitionPool *MockPartitionPool) Get(partitionNumber uint64) Partition {
    partitionPool.getCalls++

    return partitionPool.partitions[partitionNumber]
}

func (partitionPool *MockPartitionPool) GetCallCount() int {
    return partitionPool.getCalls
}

func (partitionPool *MockPartitionPool) onGet(cb func(partitionNumber uint64)) {
    partitionPool.onGetCB = cb
}

func (partitionPool *MockPartitionPool) notifyGet(partitionNumber uint64) {
    if partitionPool.onGetCB == nil {
        return
    }

    partitionPool.onGetCB(partitionNumber)
}

type mockIteratorState struct {
    next bool
    site string
    bucket string
    key string
    value *SiblingSet
    err error
}

type MockPartitionIterator struct {
    nextCalls int
    siteCalls int
    bucketCalls int
    keyCalls int
    valueCalls int
    checksumCalls int
    releaseCalls int
    errorCalls int
    currentState int
    states []mockIteratorState
}

func NewMockPartitionIterator() *MockPartitionIterator {
    return &MockPartitionIterator{
        currentState: -1,
        states: make([]mockIteratorState, 0),
    }
}

func (partitionIterator *MockPartitionIterator) AppendNextState(next bool, site string, bucket string, key string, value *SiblingSet, checksum Hash, err error) *MockPartitionIterator {
    partitionIterator.states = append(partitionIterator.states, mockIteratorState{
        next: next,
        site: site,
        bucket: bucket,
        key: key,
        value: value,
        err: err,
    })

    return partitionIterator
}

func (partitionIterator *MockPartitionIterator) Next() bool {
    partitionIterator.nextCalls++
    partitionIterator.currentState++

    return partitionIterator.states[partitionIterator.currentState].next
}

func (partitionIterator *MockPartitionIterator) NextCallCount() int {
    return partitionIterator.nextCalls
}

func (partitionIterator *MockPartitionIterator) Site() string {
    partitionIterator.siteCalls++

    return partitionIterator.states[partitionIterator.currentState].site
}

func (partitionIterator *MockPartitionIterator) SiteCallCount() int {
    return partitionIterator.siteCalls
}

func (partitionIterator *MockPartitionIterator) Bucket() string {
    partitionIterator.bucketCalls++

    return partitionIterator.states[partitionIterator.currentState].bucket
}

func (partitionIterator *MockPartitionIterator) BucketCallCount() int {
    return partitionIterator.bucketCalls
}

func (partitionIterator *MockPartitionIterator) Key() string {
    partitionIterator.keyCalls++

    return partitionIterator.states[partitionIterator.currentState].key
}

func (partitionIterator *MockPartitionIterator) KeyCallCount() int {
    return partitionIterator.keyCalls
}

func (partitionIterator *MockPartitionIterator) Value() *SiblingSet {
    partitionIterator.valueCalls++

    return partitionIterator.states[partitionIterator.currentState].value
}

func (partitionIterator *MockPartitionIterator) ValueCallCount() int {
    return partitionIterator.valueCalls
}

func (partitionIterator *MockPartitionIterator) ChecksumCallCount() int {
    return partitionIterator.checksumCalls
}

func (partitionIterator *MockPartitionIterator) Release() {
    partitionIterator.releaseCalls++
}

func (partitionIterator *MockPartitionIterator) ReleaseCallCount() int {
    return partitionIterator.releaseCalls
}

func (partitionIterator *MockPartitionIterator) Error() error {
    partitionIterator.errorCalls++

    return partitionIterator.states[partitionIterator.currentState].err
}

func (partitionIterator *MockPartitionIterator) ErrorCallCount() int {
    return partitionIterator.errorCalls
}

type MockErrorReader struct {
    errors []error
}

func NewMockErrorReader(errors []error) *MockErrorReader {
    return &MockErrorReader{
        errors: errors,
    }
}

func (errorReader *MockErrorReader) Read(p []byte) (n int, err error) {
    nextError := errorReader.errors[0]
    errorReader.errors = errorReader.errors[1:]

    return 0, nextError
}

type HTTPTestServer struct {
    server *http.Server
    listener net.Listener
    port int
    done chan int
}

func NewHTTPTestServer(port int, handler http.Handler) *HTTPTestServer {
    return &HTTPTestServer{
        port: port,
        server: &http.Server{
            Addr: fmt.Sprintf(":%d", port),
            Handler: handler,
        },
        done: make(chan int),
    }
}

func (testServer *HTTPTestServer) Start() {
    go func() {
        listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", testServer.port))
        testServer.listener = listener
        testServer.server.Serve(listener)
        close(testServer.done)
    }()
}

func (testServer *HTTPTestServer) Stop() {
    testServer.listener.Close()
    <-testServer.done
}

type HandlerFuncHandler struct {
    f http.HandlerFunc
}

func (handlerFuncHandler *HandlerFuncHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    handlerFuncHandler.f(w, req)
}

type StringResponseHandler struct {
    str io.Reader
    status int
    written int64
    err error
    after chan int
}

func (stringResponseHandler *StringResponseHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    if stringResponseHandler.status == 0 {
        stringResponseHandler.status = http.StatusOK
    }

    w.Header().Set("Content-Type", "application/json; charset=utf8")
    w.WriteHeader(stringResponseHandler.status)
    written, err := io.Copy(w, stringResponseHandler.str)

    stringResponseHandler.written = written
    stringResponseHandler.err = err

    if stringResponseHandler.after != nil {
        stringResponseHandler.after <- 1
    }
}

type InfiniteReader struct {
}

func NewInfiniteReader() *InfiniteReader {
    return &InfiniteReader{ }
}

func (infiniteReader *InfiniteReader) Read(p []byte) (n int, err error) {
    for i := 0; i < len(p); i++ {
        p[i] = byte(0)
    }

    return len(p), nil
}

type mockTransferTransportResponse struct {
    reader io.Reader
    cancel func()
    err error
}

type MockTransferTransport struct {
    responses []mockTransferTransportResponse
    getCalls int
    getCB func(nodeID uint64, partition uint64)
    defaultResponse mockTransferTransportResponse
}

func NewMockTransferTransport() *MockTransferTransport {
    return &MockTransferTransport{
        responses: make([]mockTransferTransportResponse, 0),
    }
}

func (transferTransport *MockTransferTransport) onGet(cb func(node uint64, partition uint64)) {
    transferTransport.getCB = cb
}

func (transferTransport *MockTransferTransport) notifyGet(node uint64, partition uint64) {
    if transferTransport.getCB != nil {
        transferTransport.getCB(node, partition)
    }
}

func (transferTransport *MockTransferTransport) Get(nodeID uint64, partition uint64) (io.Reader, func(), error) {
    transferTransport.getCalls++
    defer transferTransport.notifyGet(nodeID, partition)

    if len(transferTransport.responses) == 0 {
        if transferTransport.defaultResponse.reader == nil && transferTransport.defaultResponse.cancel == nil &&  transferTransport.defaultResponse.err == nil {
            return nil, nil, errors.New("No responses")
        }

        return transferTransport.defaultResponse.reader, transferTransport.defaultResponse.cancel, transferTransport.defaultResponse.err
    }

    response := transferTransport.responses[0]
    transferTransport.responses = transferTransport.responses[1:]

    return response.reader, response.cancel, response.err
}

func (transferTransport *MockTransferTransport) GetCallCount() int {
    return transferTransport.getCalls
}

func (transferTransport *MockTransferTransport) SetDefaultGetResponse(reader io.Reader, cancel func(), err error) *MockTransferTransport {
    transferTransport.defaultResponse = mockTransferTransportResponse{
        reader: reader,
        cancel: cancel,
        err: err,
    }

    return transferTransport
}

func (transferTransport *MockTransferTransport) AppendNextGetResponse(reader io.Reader, cancel func(), err error) *MockTransferTransport {
    transferTransport.responses = append(transferTransport.responses, mockTransferTransportResponse{
        reader: reader,
        cancel: cancel,
        err: err,
    })

    return transferTransport
}

type MockTransferPartnerStrategy struct {
    responses []uint64
    chooseTransferPartnerCalls int
    chooseTransferPartnerCB func(partition uint64)
    defaultResponse uint64
}

func NewMockTransferPartnerStrategy() *MockTransferPartnerStrategy {
    return &MockTransferPartnerStrategy{
        responses: make([]uint64, 0),
    }
}

// Callback adds tooling so the flow of execution in downloader can be tracked
func (partnerStrategy *MockTransferPartnerStrategy) onChooseTransferPartner(cb func(partition uint64)) {
    partnerStrategy.chooseTransferPartnerCB = cb
}

func (partnerStrategy *MockTransferPartnerStrategy) notifyChooseTransferPartner(partition uint64) {
    if partnerStrategy.chooseTransferPartnerCB != nil {
        partnerStrategy.chooseTransferPartnerCB(partition)
    }
}

func (partnerStrategy *MockTransferPartnerStrategy) ChooseTransferPartner(partition uint64) uint64 {
    partnerStrategy.chooseTransferPartnerCalls++
    defer partnerStrategy.notifyChooseTransferPartner(partition)

    if len(partnerStrategy.responses) == 0 {
        return partnerStrategy.defaultResponse
    }

    response := partnerStrategy.responses[0]
    partnerStrategy.responses = partnerStrategy.responses[1:]

    return response
}

func (partnerStrategy *MockTransferPartnerStrategy) ChooseTransferPartnerCalls() int {
    return partnerStrategy.chooseTransferPartnerCalls
}

func (partnerStrategy *MockTransferPartnerStrategy) SetDefaultChooseTransferPartnerResponse(node uint64) *MockTransferPartnerStrategy {
    partnerStrategy.defaultResponse = node

    return partnerStrategy
}

func (partnerStrategy *MockTransferPartnerStrategy) AppendNextTransferPartner(node uint64) *MockTransferPartnerStrategy {
    partnerStrategy.responses = append(partnerStrategy.responses, node)

    return partnerStrategy
}

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

type MockPartitionDownloader struct {
    downloads map[uint64]chan int
    downloadCB func(partition uint64)
    cancelDownloadCB func(partition uint64)
}

func NewMockPartitionDownloader() *MockPartitionDownloader {
    return &MockPartitionDownloader{
        downloads: make(map[uint64]chan int, 0),
    }
}

func (downloader *MockPartitionDownloader) Reset(partition uint64) {
}

func (downloader *MockPartitionDownloader) Download(partition uint64) <-chan int {
    downloader.notifyDownload(partition)

    if _, ok := downloader.downloads[partition]; !ok {
        downloader.downloads[partition] = make(chan int)
    }

    return downloader.downloads[partition]
}

func (downloader *MockPartitionDownloader) notifyDownload(partition uint64) {
    if downloader.downloadCB == nil {
        return
    }

    downloader.downloadCB(partition)
}

func (downloader *MockPartitionDownloader) onDownload(cb func(partition uint64)) {
    downloader.downloadCB = cb
}

func (downloader *MockPartitionDownloader) IsDownloading(partition uint64) bool {
    _, ok := downloader.downloads[partition]

    return ok
}

func (downloader *MockPartitionDownloader) CancelDownload(partition uint64) {
    downloader.notifyCancelDownload(partition)

    delete(downloader.downloads, partition)
}

func (downloader *MockPartitionDownloader) notifyCancelDownload(partition uint64) {
    if downloader.cancelDownloadCB == nil {
        return
    }

    downloader.cancelDownloadCB(partition)
}

func (downloader *MockPartitionDownloader) onCancelDownload(cb func(partition uint64)) {
    downloader.cancelDownloadCB = cb
}

type MockPartitionTransferProposer struct {
    proposals map[uint64]map[uint64]bool
    queueTransferProposalCB func(uint64, uint64, <-chan int)
}

func NewMockPartitionTransferProposer() *MockPartitionTransferProposer {
    return &MockPartitionTransferProposer{
        proposals: make(map[uint64]map[uint64]bool, 0),
    }
}

func (transferProposer *MockPartitionTransferProposer) QueueTransferProposal(partition uint64, replica uint64, after <-chan int) <-chan error {
    transferProposer.notifyQueueTransferProposal(partition, replica, after)

    if _, ok := transferProposer.proposals[partition]; !ok {
        transferProposer.proposals[partition] = make(map[uint64]bool, 0)
    }

    transferProposer.proposals[partition][replica] = true

    return nil
}

func (transferProposer *MockPartitionTransferProposer) notifyQueueTransferProposal(partition uint64, replica uint64, after <-chan int) {
    if transferProposer.queueTransferProposalCB == nil {
        return
    }

    transferProposer.queueTransferProposalCB(partition, replica, after)
}

func (transferProposer *MockPartitionTransferProposer) onQueueTransferProposal(cb func(partition uint64, replica uint64, after <-chan int)) {
    transferProposer.queueTransferProposalCB = cb
}

func (transferProposer *MockPartitionTransferProposer) CancelTransferProposal(partition uint64, replica uint64) {
    delete(transferProposer.proposals[partition], replica)
}

func (transferProposer *MockPartitionTransferProposer) CancelTransferProposals(partition uint64) {
}

func (transferProposer *MockPartitionTransferProposer) PendingProposals(partition uint64) int {
    return len(transferProposer.proposals[partition])
}

func (transferProposer *MockPartitionTransferProposer) QueuedProposals() map[uint64]map[uint64]bool {
    return transferProposer.proposals
}