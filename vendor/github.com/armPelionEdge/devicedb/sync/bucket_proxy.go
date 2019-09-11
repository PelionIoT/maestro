package sync

import (
    "context"
    "errors"
    "math/rand"

    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/client"
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/clusterio"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/partition"
    . "github.com/armPelionEdge/devicedb/site"
    . "github.com/armPelionEdge/devicedb/raft"
    rest "github.com/armPelionEdge/devicedb/rest"
    . "github.com/armPelionEdge/devicedb/merkle"
)

var ENoLocalBucket = errors.New("No such bucket exists locally")

type BucketProxyFactory interface {
    // Return a set of buckets for which updates can be
    // pushed from the given node to this node/cluster
    IncomingBuckets(peerID string) map[string]bool
    // Return a set of buckets for which updates can be
    // pushed from this node/cluster to the given node
    OutgoingBuckets(peerID string) map[string]bool
    // Create a bucket proxy to the bucket specified in the site
    // that the peer belongs to
    CreateBucketProxy(peerID string, bucket string) (BucketProxy, error)
}

type RelayBucketProxyFactory struct {
    // The site pool for this node
    SitePool SitePool
}

func (relayBucketProxyFactory *RelayBucketProxyFactory) CreateBucketProxy(peerID string, bucketName string) (BucketProxy, error) {
    site := relayBucketProxyFactory.SitePool.Acquire("")

    if site.Buckets().Get(bucketName) == nil {
        return nil, ENoLocalBucket
    }

    return &RelayBucketProxy{
        Bucket: site.Buckets().Get(bucketName),
        SitePool: relayBucketProxyFactory.SitePool,
        SiteID: "",
    }, nil
}

func (relayBucketProxyFactory *RelayBucketProxyFactory) IncomingBuckets(peerID string) map[string]bool {
    var buckets map[string]bool = make(map[string]bool)

    site := relayBucketProxyFactory.SitePool.Acquire("")

    for _, bucket := range site.Buckets().Incoming(peerID) {
        buckets[bucket.Name()] = true
    }

    return buckets
}

func (relayBucketProxyFactory *RelayBucketProxyFactory) OutgoingBuckets(peerID string) map[string]bool {
    var buckets map[string]bool = make(map[string]bool)

    site := relayBucketProxyFactory.SitePool.Acquire("")

    for _, bucket := range site.Buckets().Outgoing(peerID) {
        buckets[bucket.Name()] = true
    }

    return buckets
}

type CloudBucketProxyFactory struct {
    // An intra-cluster client
    Client Client
    // The cluster controller for this node
    ClusterController *ClusterController
    // The partition pool for this node
    PartitionPool PartitionPool
    // The cluster io agent for this node
    ClusterIOAgent ClusterIOAgent
}

func (cloudBucketProxyFactory *CloudBucketProxyFactory) CreateBucketProxy(peerID string, bucketName string) (BucketProxy, error) {
    siteID := cloudBucketProxyFactory.ClusterController.RelaySite(peerID)
    partitionNumber := cloudBucketProxyFactory.ClusterController.Partition(siteID)
    nodeIDs := cloudBucketProxyFactory.ClusterController.PartitionOwners(partitionNumber)

    if len(nodeIDs) == 0 {
        return nil, errors.New("No node owns this partition")
    }

    // Choose a node at random from the nodes that own this site database
    nodeID := nodeIDs[int(rand.Uint32() % uint32(len(nodeIDs)))]

    if cloudBucketProxyFactory.ClusterController.LocalNodeID == nodeID {
        partition := cloudBucketProxyFactory.PartitionPool.Get(partitionNumber)

        if partition == nil {
            return nil, ENoLocalBucket
        }

        site := partition.Sites().Acquire(siteID)

        if site == nil || site.Buckets().Get(bucketName) == nil {
            return nil, ENoLocalBucket
        }

        localBucket := &CloudLocalBucketProxy{
            Bucket: site.Buckets().Get(bucketName),
            SitePool: partition.Sites(),
            SiteID: siteID,
            ClusterIOAgent: cloudBucketProxyFactory.ClusterIOAgent,
        }

        return localBucket, nil
    }

    return &CloudRemoteBucketProxy{
        Client: cloudBucketProxyFactory.Client,
        PeerAddress: cloudBucketProxyFactory.ClusterController.ClusterMemberAddress(nodeID),
        SiteID: siteID,
        BucketName: bucketName,
        ClusterIOAgent: cloudBucketProxyFactory.ClusterIOAgent,
    }, nil
}

func (cloudBucketProxyFactory *CloudBucketProxyFactory) IncomingBuckets(peerID string) map[string]bool {
    return map[string]bool{ "default": true, "lww": true }
}

func (cloudBucketProxyFactory *CloudBucketProxyFactory) OutgoingBuckets(peerID string) map[string]bool {
    return map[string]bool{ "default": true, "lww": true, "cloud": true }
}

type BucketProxy interface {
    Name() string
    MerkleTree() MerkleTreeProxy
    GetSyncChildren(nodeID uint32) (SiblingSetIterator, error)
    Merge(mergedKeys map[string]*SiblingSet) error
    Forget(keys [][]byte) error
    Close()
}

type RelayBucketProxy struct {
    Bucket Bucket
    SiteID string
    SitePool SitePool
}

func (relayBucketProxy *RelayBucketProxy) Name() string {
    return relayBucketProxy.Bucket.Name()
}

func (relayBucketProxy *RelayBucketProxy) MerkleTree() MerkleTreeProxy {
    return &DirectMerkleTreeProxy{
        merkleTree: relayBucketProxy.Bucket.MerkleTree(),
    }
}

func (relayBucketProxy *RelayBucketProxy) GetSyncChildren(nodeID uint32) (SiblingSetIterator, error) {
    return relayBucketProxy.Bucket.GetSyncChildren(nodeID)
}

func (relayBucketProxy *RelayBucketProxy) Close() {
    relayBucketProxy.SitePool.Release(relayBucketProxy.SiteID)
}

func (relayBucketProxy *RelayBucketProxy) Merge(mergedKeys map[string]*SiblingSet) error {
    return relayBucketProxy.Bucket.Merge(mergedKeys)
}

func (relayBucketProxy *RelayBucketProxy) Forget(keys [][]byte) error {
    return relayBucketProxy.Bucket.Forget(keys)
}

type CloudResponderMerkleNodeIterator struct {
    MerkleKeys rest.MerkleKeys
    CurrentIndex int
}

func (iter *CloudResponderMerkleNodeIterator) Next() bool {
    if iter.CurrentIndex >= len(iter.MerkleKeys.Keys) - 1 {
        iter.CurrentIndex = len(iter.MerkleKeys.Keys)

        return false
    }

    iter.CurrentIndex++

    return true
}

func (iter *CloudResponderMerkleNodeIterator) Prefix() []byte {
    return nil
}

func (iter *CloudResponderMerkleNodeIterator) Key() []byte {
    if iter.CurrentIndex < 0 || len(iter.MerkleKeys.Keys) == 0 || iter.CurrentIndex >= len(iter.MerkleKeys.Keys) {
        return nil
    }

    return []byte(iter.MerkleKeys.Keys[iter.CurrentIndex].Key)
}

func (iter *CloudResponderMerkleNodeIterator) Value() *SiblingSet {
    if iter.CurrentIndex < 0 || len(iter.MerkleKeys.Keys) == 0 || iter.CurrentIndex >= len(iter.MerkleKeys.Keys) {
        return nil
    }

    return iter.MerkleKeys.Keys[iter.CurrentIndex].Value
}

func (iter *CloudResponderMerkleNodeIterator) LocalVersion() uint64 {
    return 0
}

func (iter *CloudResponderMerkleNodeIterator) Release() {
}

func (iter *CloudResponderMerkleNodeIterator) Error() error {
    return nil
}

type CloudLocalBucketProxy struct {
    Bucket Bucket
    SiteID string
    SitePool SitePool
    ClusterIOAgent ClusterIOAgent
}

func (bucketProxy *CloudLocalBucketProxy) Name() string {
    return bucketProxy.Bucket.Name()
}

func (bucketProxy *CloudLocalBucketProxy) MerkleTree() MerkleTreeProxy {
    return &DirectMerkleTreeProxy{
        merkleTree: bucketProxy.Bucket.MerkleTree(),
    }
}

func (bucketProxy *CloudLocalBucketProxy) GetSyncChildren(nodeID uint32) (SiblingSetIterator, error) {
    return bucketProxy.Bucket.GetSyncChildren(nodeID)
}

func (bucketProxy *CloudLocalBucketProxy) Merge(mergedKeys map[string]*SiblingSet) error {
    _, _, err := bucketProxy.ClusterIOAgent.Merge(context.TODO(), bucketProxy.SiteID, bucketProxy.Bucket.Name(), mergedKeys)

    return err
}

func (bucketProxy *CloudLocalBucketProxy) Forget(keys [][]byte) error {
    return nil
}

func (bucketProxy *CloudLocalBucketProxy) Close() {
    bucketProxy.SitePool.Release(bucketProxy.SiteID)
}

type CloudRemoteBucketProxy struct {
    Client Client
    PeerAddress PeerAddress
    SiteID string
    BucketName string
    ClusterIOAgent ClusterIOAgent
    merkleTreeProxy MerkleTreeProxy
}

func (bucketProxy *CloudRemoteBucketProxy) Name() string {
    return bucketProxy.BucketName
}

func (bucketProxy *CloudRemoteBucketProxy) MerkleTree() MerkleTreeProxy {
    if bucketProxy.merkleTreeProxy != nil {
        return bucketProxy.merkleTreeProxy
    }

    merkleTreeStats, err := bucketProxy.Client.MerkleTreeStats(context.TODO(), bucketProxy.PeerAddress, bucketProxy.SiteID, bucketProxy.BucketName)

    if err != nil {
        bucketProxy.merkleTreeProxy = &CloudResponderMerkleTreeProxy{
            err: err,
        }

        return bucketProxy.merkleTreeProxy
    }

    dummyMerkleTree, err := NewDummyMerkleTree(merkleTreeStats.Depth)

    if err != nil {
        bucketProxy.merkleTreeProxy = &CloudResponderMerkleTreeProxy{
            err: err,
        }

        return bucketProxy.merkleTreeProxy
    }

    bucketProxy.merkleTreeProxy = &CloudResponderMerkleTreeProxy{
        err: nil,
        client: bucketProxy.Client,
        peerAddress: bucketProxy.PeerAddress,
        siteID: bucketProxy.SiteID,
        bucketName: bucketProxy.BucketName,
        merkleTree: dummyMerkleTree,
    }

    return bucketProxy.merkleTreeProxy
}

func (bucketProxy *CloudRemoteBucketProxy) GetSyncChildren(nodeID uint32) (SiblingSetIterator, error) {
    merkleKeys, err := bucketProxy.Client.MerkleTreeNodeKeys(context.TODO(), bucketProxy.PeerAddress, bucketProxy.SiteID, bucketProxy.BucketName, nodeID)

    if err != nil {
        return nil, err
    }

    return &CloudResponderMerkleNodeIterator{
        MerkleKeys: merkleKeys,
        CurrentIndex: -1,
    }, nil
}

func (bucketProxy *CloudRemoteBucketProxy) Merge(mergedKeys map[string]*SiblingSet) error {
    _, _, err := bucketProxy.ClusterIOAgent.Merge(context.TODO(), bucketProxy.SiteID, bucketProxy.BucketName, mergedKeys)

    return err
}

func (bucketProxy *CloudRemoteBucketProxy) Forget(keys [][]byte) error {
    return nil
}

func (bucketProxy *CloudRemoteBucketProxy) Close() {
}