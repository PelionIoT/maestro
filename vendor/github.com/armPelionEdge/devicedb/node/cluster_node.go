package node

import (
    "context"
    "crypto/tls"
    "encoding/binary"
    "errors"
    "fmt"
    "io"
    "math/rand"
    "net/http"
    "sync"
    "time"

    . "github.com/armPelionEdge/devicedb/bucket"
    "github.com/armPelionEdge/devicedb/client"
    . "github.com/armPelionEdge/devicedb/cluster"
    "github.com/armPelionEdge/devicedb/clusterio"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/logging"
    . "github.com/armPelionEdge/devicedb/merkle"
    . "github.com/armPelionEdge/devicedb/partition"
    . "github.com/armPelionEdge/devicedb/raft"
    . "github.com/armPelionEdge/devicedb/routes"
    . "github.com/armPelionEdge/devicedb/server"
    . "github.com/armPelionEdge/devicedb/site"
    . "github.com/armPelionEdge/devicedb/storage"
    ddbSync "github.com/armPelionEdge/devicedb/sync"
    . "github.com/armPelionEdge/devicedb/transfer"
    . "github.com/armPelionEdge/devicedb/util"

    "github.com/gorilla/websocket"
    "github.com/coreos/etcd/raft"
    "github.com/coreos/etcd/raft/raftpb"
)

const (
    RaftStoreStoragePrefix = iota
    SiteStoreStoragePrefix = iota
    SnapshotMetadataPrefix = iota
)

const SnapshotUUIDKey string = "UUID"

const ClusterJoinRetryTimeout = 5

type ClusterNodeConfig struct {
    StorageDriver StorageDriver
    CloudServer *CloudServer
    MerkleDepth uint8
    Capacity uint64
    NoValidate bool
}

type ClusterNode struct {
    interClusterClient *client.Client
    configController ClusterConfigController
    configControllerBuilder ClusterConfigControllerBuilder
    cloudServer *CloudServer
    raftTransport *TransportHub
    raftStore RaftNodeStorage
    transferAgent PartitionTransferAgent
    clusterioAgent clusterio.ClusterIOAgent
    storageDriver StorageDriver
    partitionFactory PartitionFactory
    partitionPool PartitionPool
    joinedCluster chan int
    leftCluster chan int
    leftClusterResult chan error
    isRunning bool
    shutdown chan int
    empty chan int
    initializedCB func()
    merkleDepth uint8
    capacity uint64
    shutdownDecommissioner func()
    lock sync.Mutex
    emptyMu sync.Mutex
    relayConnectionsMu sync.Mutex
    hub *Hub
    noValidate bool
    snapshotsDirectory string
    snapshotter *Snapshotter
}

func New(config ClusterNodeConfig) *ClusterNode {
    if config.MerkleDepth < MerkleMinDepth {
        config.MerkleDepth = MerkleDefaultDepth
    }

    clusterNode := &ClusterNode{
        storageDriver: config.StorageDriver,
        cloudServer: config.CloudServer,
        raftStore: NewRaftStorage(NewPrefixedStorageDriver([]byte{ RaftStoreStoragePrefix }, config.StorageDriver)),
        raftTransport: NewTransportHub(0),
        configControllerBuilder: &ConfigControllerBuilder{ },
        interClusterClient: client.NewClient(client.ClientConfig{ }),
        merkleDepth: config.MerkleDepth,
        capacity: config.Capacity,
        partitionFactory: NewDefaultPartitionFactory(),
        partitionPool: NewDefaultPartitionPool(),
        noValidate: config.NoValidate,
    }

    if clusterNode.noValidate {
        Log.Criticalf("!!! Starting node with NoValidate set to true. This option should not be used in production as it allows connecting relays to determine their own ID based on an HTTP header. This is for use in testing only and should not be active in a production cluster !!!")
    }

    return clusterNode
}

func (node *ClusterNode) UseRaftStore(raftStore RaftNodeStorage) {
    node.raftStore = raftStore
}

func (node *ClusterNode) getNodeID() (uint64, error) {
    if err := node.raftStore.Open(); err != nil {
        Log.Criticalf("Local node unable to open raft store: %v", err.Error())

        return 0, err
    }

    nodeID, err := node.raftStore.NodeID()

    if err != nil {
        Log.Criticalf("Local node unable to obtain node ID from raft store: %v", err.Error())

        return 0, err
    }

    if nodeID == 0 {
        nodeID = UUID64()

        Log.Infof("Local node initializing with ID %d", nodeID)

        if err := node.raftStore.SetNodeID(nodeID); err != nil {
            Log.Criticalf("Local node unable to store new node ID: %v", err.Error())

            return 0, err
        }
    }

    return nodeID, nil
}

func (node *ClusterNode) Start(options NodeInitializationOptions) error {
    node.isRunning = true
    node.shutdown = make(chan int)
    node.joinedCluster = make(chan int, 1)
    node.leftCluster = make(chan int, 1)
    node.snapshotsDirectory = options.SnapshotDirectory

    if err := node.openStorageDriver(); err != nil {
        return err
    }

    nodeID, err := node.getNodeID()

    if err != nil {
        return err
    }

    node.snapshotter = &Snapshotter{
        nodeID: nodeID,
        snapshotsDirectory: node.snapshotsDirectory,
        storageDriver: node.storageDriver,
    }

    Log.Infof("Local node (id = %d) starting up...", nodeID)

    node.raftTransport.SetLocalPeerID(nodeID)

    clusterHost, clusterPort := options.ClusterAddress()
    node.configControllerBuilder.SetLocalNodeAddress(PeerAddress{ NodeID: nodeID, Host: clusterHost, Port: clusterPort })
    node.configControllerBuilder.SetRaftNodeStorage(node.raftStore)
    node.configControllerBuilder.SetRaftNodeTransport(node.raftTransport)
    node.configControllerBuilder.SetCreateNewCluster(options.ShouldStartCluster())
    node.configController = node.configControllerBuilder.Create()

    stateCoordinator := NewClusterNodeStateCoordinator(&NodeCoordinatorFacade{ node: node }, nil)
    node.configController.OnLocalUpdates(func(deltas []ClusterStateDelta) {
        stateCoordinator.ProcessClusterUpdates(deltas)
    })

    node.configController.OnClusterSnapshot(func(snapshotIndex uint64, snapshotId string) {
        node.localSnapshot(snapshotIndex, snapshotId)
    })

    node.configController.Start()
    defer node.Stop()

    if node.configController.ClusterController().LocalNodeWasRemovedFromCluster() {
        Log.Errorf("Local node (id = %d) unable to start because it was removed from the cluster", nodeID)

        return ERemoved
    }

    // It is important to initialize node before networking starts
    // to ensure no cluster config state changes occur while initialize is being called.
    // Initialize needs to set up transfers and partitions with the node's last known
    // state before changes to its partitions ownership and partition transfers
    // occur
    node.transferAgent = NewDefaultHTTPTransferAgent(node.configController, node.partitionPool)
    node.clusterioAgent = clusterio.NewAgent(NewNodeClient(node, node.configController), NewPartitionResolver(node.configController))

    if options.SyncPeriod < 1000 {
        options.SyncPeriod = 1000
    }

    bucketProxyFactory := &ddbSync.CloudBucketProxyFactory{
        Client: *node.interClusterClient,
        ClusterController: node.configController.ClusterController(),
        PartitionPool: node.partitionPool,
        ClusterIOAgent: node.clusterioAgent,
    }
    syncController := NewSyncController(options.SyncMaxSessions, bucketProxyFactory, ddbSync.NewMultiSyncScheduler(time.Millisecond * time.Duration(options.SyncPeriod)), options.SyncPathLimit)
    node.hub = NewHub("", syncController, nil)

    stateCoordinator.InitializeNodeState()

    node.hub.SyncController().Start()
    serverStopResult := node.startNetworking()
    decommission, err := node.raftStore.IsDecommissioning()

    if err != nil {
        Log.Criticalf("Local node (id = %d) unable to start up since it could not check the decomissioning flag: %v", nodeID, err.Error())

        return err
    }

    if decommission {
        Log.Infof("Local node (id = %d) will resume decommissioning process", nodeID)

        err, result := node.LeaveCluster()

        if err != nil {
            Log.Criticalf("Local node (id = %d) unable to resume decommissioning process: %v", nodeID, err.Error())

            return err
        }

        return <-result
    }
    
    if !node.configController.ClusterController().LocalNodeIsInCluster() || !node.configController.ClusterController().State.ClusterSettings.AreInitialized() {
        if options.ShouldJoinCluster() {
            seedHost, seedPort := options.SeedNode()

            Log.Infof("Local node (id = %d) joining existing cluster. Seed node at %s:%d", nodeID, seedHost, seedPort)

            if err := node.joinCluster(seedHost, seedPort); err != nil {
                Log.Criticalf("Local node (id = %d) unable to join cluster: %v", nodeID, err.Error())

                return err
            }
        } else {
            Log.Infof("Local node (id = %d) creating new cluster...", nodeID)

            if err := node.initializeCluster(options.ClusterSettings); err != nil {
                Log.Criticalf("Local node (id = %d) unable to create new cluster: %v", nodeID, err.Error())

                return err
            }
        }
    }

    node.notifyInitialized()

    select {
    case <-node.leftCluster:
        Log.Infof("Local node (id = %d) shutting down...", nodeID)
        return ERemoved
    case err := <-serverStopResult:
        Log.Errorf("Local node (id = %d) stopped with error: %v", nodeID, err.Error())
        return err
    case <-node.shutdown:
        return nil
    }
}

func (node *ClusterNode) notifyInitialized() {
    if node.initializedCB != nil {
        node.initializedCB()
    }
}

func (node *ClusterNode) OnInitialized(cb func()) {
    node.initializedCB = cb
}

func (node *ClusterNode) ClusterConfigController() ClusterConfigController {
    return node.configController
}

func (node *ClusterNode) openStorageDriver() error {
    if err := node.storageDriver.Open(); err != nil {
        if err != ECorrupted {
            Log.Criticalf("Error opening storage driver: %v", err.Error())
            
            return EStorage
        }

        Log.Error("Database is corrupted. Attempting automatic recovery now...")

        recoverError := node.recover()

        if recoverError != nil {
            Log.Criticalf("Unable to recover corrupted database. Reason: %v", recoverError.Error())
            Log.Critical("Database daemon will now exit")

            return EStorage
        }
    }

    return nil
}

func (node *ClusterNode) recover() error {
    recoverError := node.storageDriver.Recover()

    if recoverError != nil {
        Log.Criticalf("Unable to recover corrupted database. Reason: %v", recoverError.Error())

        return EStorage
    }

    return nil
}

func (node *ClusterNode) startNetworking() <-chan error {
    router := node.cloudServer.Router()
    clusterEndpoint := &ClusterEndpoint{ ClusterFacade: &ClusterNodeFacade{ node: node } }
    partitionsEndpoint := &PartitionsEndpoint{ ClusterFacade: &ClusterNodeFacade{ node: node } }
    relaysEndpoint := &RelaysEndpoint{ ClusterFacade: &ClusterNodeFacade{ node: node } }
    sitesEndpoint := &SitesEndpoint{ ClusterFacade: &ClusterNodeFacade{ node: node } }
    syncEndpoint := &SyncEndpoint{ ClusterFacade: &ClusterNodeFacade{ node: node }, Upgrader: websocket.Upgrader{ ReadBufferSize: 1024, WriteBufferSize: 1024 } }
    logDumEndpoint := &LogDumpEndpoint{ ClusterFacade: &ClusterNodeFacade{ node: node } }
    snapshotEndpoint := &SnapshotEndpoint{ ClusterFacade: &ClusterNodeFacade{ node: node } }
    profileEndpoint := &ProfilerEndpoint{ }
    prometheusEndpoint := &PrometheusEndpoint{ }
    merkleSyncEndpoint := &ddbSync.BucketSyncHTTP{ PartitionPool: node.partitionPool, ClusterConfigController: node.configController }
    kubernetesEndpoint := &KubernetesEndpoint{ }

    node.raftTransport.Attach(router)
    node.transferAgent.(*HTTPTransferAgent).Attach(router)
    clusterEndpoint.Attach(router)
    partitionsEndpoint.Attach(router)
    relaysEndpoint.Attach(router)
    // Note: Need to have merkleSyncEndpoint before sitesEndpoint
    // since sitesEndpoint sets up a PrefixPath route for /sites/
    // which is a prefix the merkleSyncEndpoints share.
    merkleSyncEndpoint.Attach(router)    
    sitesEndpoint.Attach(router)
    syncEndpoint.Attach(router)
    logDumEndpoint.Attach(router)
    snapshotEndpoint.Attach(router)
    profileEndpoint.Attach(router)
    prometheusEndpoint.Attach(router)
    kubernetesEndpoint.Attach(router)

    startResult := make(chan error)

    go func() {
        startResult <- node.cloudServer.Start()
    }()

    return startResult
}

func (node *ClusterNode) sitePool(partitionNumber uint64) SitePool {
    storageDriver := NewPrefixedStorageDriver(node.sitePoolStorePrefix(partitionNumber), node.storageDriver)
    siteFactory := &CloudSiteFactory{ NodeID: node.Name(), MerkleDepth: node.merkleDepth, StorageDriver: storageDriver }

    return &CloudNodeSitePool{ SiteFactory: siteFactory }
}

func (node *ClusterNode) sitePoolStorePrefix(partitionNumber uint64) []byte {
    prefix := make([]byte, 9)

    prefix[0] = SiteStoreStoragePrefix
    binary.BigEndian.PutUint64(prefix[1:], partitionNumber)

    return prefix
}

func (node *ClusterNode) Stop() {
    node.lock.Lock()
    defer node.lock.Unlock()

    node.stop()
}

func (node *ClusterNode) stop() {
    node.storageDriver.Close()
    node.configController.Stop()
    node.cloudServer.Stop()

    if node.shutdownDecommissioner != nil {
        node.shutdownDecommissioner()
    }

    if node.isRunning {
        node.isRunning = false
        close(node.shutdown)
    }
}

func (node *ClusterNode) ID() uint64 {
    return node.configController.ClusterController().LocalNodeID
}

func (node *ClusterNode) Name() string {
    return "cloud-" + fmt.Sprintf("%d", node.ID())
}

func (node *ClusterNode) initializeCluster(settings ClusterSettings) error {
    ctx, cancel := context.WithCancel(context.Background())

    go func() {
        select {
        case <-ctx.Done():
            return
        case <-node.shutdown:
            cancel()
            return
        }
    }()

    Log.Infof("Local node (id = %d) initializing cluster settings (replication_factor = %d, partitions = %d)", node.ID(), settings.ReplicationFactor, settings.Partitions)

    if err := node.configController.ClusterCommand(ctx, ClusterSetReplicationFactorBody{ ReplicationFactor: settings.ReplicationFactor }); err != nil {
        Log.Criticalf("Local node (id = %d) was unable to initialize the replication factor of the new cluster: %v", node.ID(), err.Error())

        return err
    }

    if err := node.configController.ClusterCommand(ctx, ClusterSetPartitionCountBody{ Partitions: settings.Partitions }); err != nil {
        Log.Criticalf("Local node (id = %d) was unable to initialize the partition count factor of the new cluster: %v", node.ID(), err.Error())

        return err
    }

    Log.Infof("Cluster initialization complete!")

    return nil
}

func (node *ClusterNode) joinCluster(seedHost string, seedPort int) error {
    node.raftTransport.SetDefaultRoute(seedHost, seedPort)

    memberAddress := PeerAddress{
        Host: seedHost,
        Port: seedPort,
    }

    newMemberConfig := NodeConfig{
        Capacity: node.capacity,
        Address: PeerAddress{
            NodeID: node.ID(),
            Host: node.cloudServer.InternalHost(),
            Port: node.cloudServer.InternalPort(),
        },
    }

    for {
        ctx, cancel := context.WithCancel(context.Background())
        wasAdded := false
        stopped := make(chan int)

        // run a goroutine in the background to
        // cancel running add node request when
        // this node is shut down
        go func() {
            defer func() { stopped <- 1 }()

            for {
                select {
                case <-node.joinedCluster:
                    wasAdded = true
                    cancel()
                    return
                case <-ctx.Done():
                    return
                case <-node.shutdown:
                    cancel()
                    return
                }
            }
        }()

        Log.Infof("Local node (id = %d) is trying to join a cluster through an existing cluster member at %s:%d", node.ID(), seedHost, seedPort)
        err := node.interClusterClient.AddNode(ctx, memberAddress, newMemberConfig)

        // Cancel to ensure the goroutine gets cleaned up
        cancel()

        // Ensure that the above goroutine has exited and there are no new updates to consume
        <-stopped

        if wasAdded {
            return nil
        }

        if _, ok := err.(DBerror); ok {
            if err.(DBerror) == EDuplicateNodeID {
                Log.Criticalf("Local node (id = %d) request to join the cluster failed because its ID is not unique. This may indicate that the node is trying to use a duplicate ID or it may indicate that a previous proposal that this node made was already accepted and it just hasn't heard about it yet.", node.ID())
                Log.Criticalf("Local node (id = %d) will now wait one minute to see if it is part of the cluster. If it receives no messages it will shut down", node.ID())

                select {
                case <-node.joinedCluster:
                    return nil
                case <-node.shutdown:
                    return EStopped
                case <-time.After(time.Minute):
                    return EDuplicateNodeID
                }
            }
        }

        if err != nil {
            Log.Errorf("Local node (id = %d) encountered an error while trying to join cluster: %v", node.ID(), err.Error())
            Log.Infof("Local node (id = %d) will try to join the cluster again in %d seconds", node.ID(), ClusterJoinRetryTimeout)

            select {
            case <-node.joinedCluster:
                // The node has been added to the cluster. The AddNode() request may
                // have been successfully submitted but the response just didn't make
                // it to this node, but it worked. No need to retry joining
                return nil
            case <-node.shutdown:
                return EStopped
            case <-time.After(time.Second * ClusterJoinRetryTimeout):
                continue
            }
        }

        select {
        case <-node.joinedCluster:
            return nil
        case <-node.shutdown:
            return EStopped
        }
    }
}

func (node *ClusterNode) LeaveCluster() (error, <-chan error) {
    node.lock.Lock()
    defer node.lock.Unlock()
    
    node.waitForEmpty()

    // allow at mot one decommissioner
    if node.shutdownDecommissioner != nil {
        return nil, node.leftClusterResult
    }

    Log.Infof("Local node (id = %d) is being put into decommissioning mode", node.ID())

    if err := node.raftStore.SetDecommissioningFlag(); err != nil {
        Log.Errorf("Local node (id = %d) was unable to be put into decommissioning mode: %v", node.ID(), err.Error())

        return err, nil
    }

    ctx, cancel := context.WithCancel(context.Background())
    node.shutdownDecommissioner = cancel
    node.leftClusterResult = make(chan error, 1)

    go func() {
        node.leftClusterResult <- node.decommission(ctx)
    }()

    return nil, node.leftClusterResult
}

func (node *ClusterNode) waitForEmpty() {
    node.emptyMu.Lock()
    defer node.emptyMu.Unlock()

    node.empty = make(chan int, 1)
}

func (node *ClusterNode) notifyEmpty() {
    node.emptyMu.Lock()
    defer node.emptyMu.Unlock()

    if node.empty != nil {
        node.empty <- 1
    }
}

func (node *ClusterNode) decommission(ctx context.Context) error {
    Log.Infof("Local node (id = %d) starting decommissioning process...", node.ID())

    localNodeConfig := node.configController.ClusterController().LocalNodeConfig()

    if localNodeConfig == nil {
        Log.Criticalf("Local node (id = %d) unable to continue decommissioning process since its node config is not in the cluster config", node.ID())

        return ERemoved
    }

    if localNodeConfig.Capacity != 0 {
        Log.Infof("Local node (id = %d) decommissioning (1/4): Giving up tokens...", node.ID())

        if err := node.configController.ClusterCommand(ctx, ClusterUpdateNodeBody{ NodeID: node.ID(), NodeConfig: NodeConfig{ Capacity: 0, Address: localNodeConfig.Address } }); err != nil {
            Log.Criticalf("Local node (id = %d) was unable to give up its tokens: %v", node.ID(), err.Error())

            return err
        }
    }

    // Transfers should be stopped anyway once the capacity is set to zero and this node no longer owns
    // any tokens but call it here to make sure all have stopped by this point.
    node.transferAgent.StopAllTransfers()
    heldPartitionReplicas := node.configController.ClusterController().LocalNodeHeldPartitionReplicas()

    if len(heldPartitionReplicas) > 0 {
        Log.Infof("Local node (id = %d) decommissioning (2/4): Locking partitions...", node.ID())

        // Write lock partitions that are still held. This should occur anyway since
        // The node no longer owns these partitions but calling it here ensures this
        // invariant holds for the next steps of the decommissioning process
        for _, partitionReplica := range heldPartitionReplicas {
            partition := node.partitionPool.Get(partitionReplica.Partition)

            if partition != nil {
                Log.Debugf("Local node (id = %d) decommissioning (2/4): Write locking partition %d", node.ID(), partition.Partition())

                node.transferAgent.EnableOutgoingTransfers(partition.Partition())
                partition.LockWrites()
            }
        }

        Log.Infof("Local node (id = %d) decommissioning (3/4): Transferring partition data...", node.ID())

        // Wait for all partition data to be transferred away from this node. This ensures that
        // the data that this node held is replicated elsewhere before it removes itself from the
        // cluster permanently.
        select {
        case <-node.leftCluster:
            return ERemoved
        case <-node.empty:
        case <-ctx.Done():
            return ECancelled
        }
    }

    Log.Infof("Local node (id = %d) decommissioning (4/4): Leaving cluster...", node.ID())

    if err := node.configController.RemoveNode(ctx, node.ID()); err != nil {
        Log.Criticalf("Local node (id = %d) was unable to leave cluster: %v", node.ID(), err.Error())

        return err
    }

    return EDecommissioned
}

func (node *ClusterNode) Batch(ctx context.Context, partitionNumber uint64, siteID string, bucketName string, updateBatch *UpdateBatch) (map[string]*SiblingSet, error) {
    partition := node.partitionPool.Get(partitionNumber)

    if partition == nil {
        return nil, ENoSuchPartition
    }

    site := partition.Sites().Acquire(siteID)

    if site == nil {
        return nil, ENoSuchSite
    }

    bucket := site.Buckets().Get(bucketName)

    if bucket == nil {
        return nil, ENoSuchBucket
    }

    if !node.configController.ClusterController().LocalNodeHoldsPartition(partitionNumber) {
        return nil, ENoQuorum
    }

    patch, err := bucket.Batch(updateBatch)

    if err != nil {
        return nil, err
    }

    node.hub.BroadcastUpdate(siteID, bucketName, patch, 10)

    return patch, nil
}

func (node *ClusterNode) Merge(ctx context.Context, partitionNumber uint64, siteID string, bucketName string, patch map[string]*SiblingSet, broadcastToRelays bool) error {
    partition := node.partitionPool.Get(partitionNumber)

    if partition == nil {
        return ENoSuchPartition
    }

    site := partition.Sites().Acquire(siteID)

    if site == nil {
        return ENoSuchSite
    }

    bucket := site.Buckets().Get(bucketName)

    if bucket == nil {
        return ENoSuchBucket
    }

    err := bucket.Merge(patch)

    if err != nil {
        return err
    }

    if !node.configController.ClusterController().LocalNodeHoldsPartition(partitionNumber) {
        return ENoQuorum
    }

    if broadcastToRelays {
        node.hub.BroadcastUpdate(siteID, bucketName, patch, 10)
    }

    return nil
}

func (node *ClusterNode) Get(ctx context.Context, partitionNumber uint64, siteID string, bucketName string, keys [][]byte) ([]*SiblingSet, error) {
    partition := node.partitionPool.Get(partitionNumber)

    if partition == nil {
        return nil, ENoSuchPartition
    }

    site := partition.Sites().Acquire(siteID)

    if site == nil {
        return nil, ENoSuchSite
    }

    bucket := site.Buckets().Get(bucketName)

    if bucket == nil {
        return nil, ENoSuchBucket
    }

    return bucket.Get(keys)
}

func (node *ClusterNode) GetMatches(ctx context.Context, partitionNumber uint64, siteID string, bucketName string, keys [][]byte) (SiblingSetIterator, error) {
    partition := node.partitionPool.Get(partitionNumber)

    if partition == nil {
        return nil, ENoSuchPartition
    }

    site := partition.Sites().Acquire(siteID)

    if site == nil {
        return nil, ENoSuchSite
    }

    bucket := site.Buckets().Get(bucketName)

    if bucket == nil {
        return nil, ENoSuchBucket
    }

    return bucket.GetMatches(keys)
}

func (node *ClusterNode) AcceptRelayConnection(conn *websocket.Conn, header http.Header) {
    node.relayConnectionsMu.Lock()
    defer node.relayConnectionsMu.Unlock()

    var relayID string

    if _, ok := conn.UnderlyingConn().(*tls.Conn); !ok {
        if header.Get("X-WigWag-RelayID") == "" {
            Log.Warningf("Cannot accept non-secure relay connections. Must use TLS")

            conn.Close()

            return
        }

        relayID = header.Get("X-WigWag-RelayID")
    } else {
        var err error
        relayID, err = node.hub.ExtractPeerID(conn.UnderlyingConn().(*tls.Conn))

        if err != nil && !node.noValidate {
            Log.Warningf("Cannot accept connection from relay because it provided an invalid client cert.")

            conn.Close()

            return
        }

        if node.noValidate && header.Get("X-WigWag-RelayID") != "" {
            relayID = header.Get("X-WigWag-RelayID")
        }
    }

    siteID := node.configController.ClusterController().RelaySite(relayID)

    if siteID == "" {
        Log.Warningf("Unable to accept connection from relay %s because it has either not been added to the devicedb relay database or it does not belong to a site", relayID)

        conn.Close()

        return
    }

    partitionNumber := node.configController.ClusterController().Partition(siteID)
    owners := node.configController.ClusterController().PartitionOwners(partitionNumber)

    if len(owners) == 0 {
        Log.Warningf("Unable to accept connection from relay %s because no node owns the partition for its site, site %s", relayID, siteID)

        conn.Close()

        return
    }

    for _, nodeID := range owners {
        if nodeID == node.configController.ClusterController().LocalNodeID {
            Log.Infof("Local node (id = %d) accepting connection from relay %s which belongs to site %s", nodeID, relayID, siteID)

            // The local node owns this site database. It can accept the connection for this relay
            node.hub.Accept(conn, partitionNumber, relayID, siteID, node.noValidate)

            return
        }
    }

    // Can only proxy wss -> ws
    //if _, ok := conn.UnderlyingConn().(*tls.Conn); !ok {
    //    Log.Warningf("Local node (id = %d) cannot accept proxied connection from relay %s because it does not own the partition to which its site, site %s, belongs", node.configController.ClusterController().LocalNodeID, relayID, siteID)

    //    conn.Close()

    //    return
    //}

    // The local node does not own the site database for this site. It should proxy the connection to one of the owners
    nodeID := owners[int(rand.Uint32() % uint32(len(owners)))]

    Log.Infof("Local node (id = %d) proxying connection from relay %s which belongs to site %s to node %d", node.configController.ClusterController().LocalNodeID, relayID, siteID, nodeID)
    node.proxyRelayConnection(nodeID, relayID, conn)
}

func (node *ClusterNode) proxyRelayConnection(nodeID uint64, relayID string, conn *websocket.Conn) {
    var dialer *websocket.Dialer = websocket.DefaultDialer

    header := http.Header{}
    header.Set("X-WigWag-RelayID", relayID)
    nodeAddress := node.ClusterConfigController().ClusterController().ClusterMemberAddress(nodeID)
    connBackend, _, err := dialer.Dial(fmt.Sprintf("ws://%s:%d/sync", nodeAddress.Host, nodeAddress.Port), header)

    if err != nil {
        Log.Warningf("Unable to proxy connection to node %d: %v", nodeID, err)

        conn.Close()

        return
    }

    errors := make(chan error, 2)
    cp := func(dest io.Writer, src io.Reader) {
        _, err := io.Copy(dest, src)
        errors <- err
    }

    go cp(connBackend.UnderlyingConn(), conn.UnderlyingConn())
    go cp(conn.UnderlyingConn(), connBackend.UnderlyingConn())

    go func() {
        err := <-errors
        Log.Infof("Closing proxied connection: %v", err)

        conn.Close()
        connBackend.Close()
    }()
}

func (node *ClusterNode) DisconnectRelay(relayID string) {
    node.relayConnectionsMu.Lock()
    defer node.relayConnectionsMu.Unlock()

    node.hub.ReconnectPeer(relayID)
}

func (node *ClusterNode) DisconnectRelayBySite(siteID string) {
    node.relayConnectionsMu.Lock()
    defer node.relayConnectionsMu.Unlock()

    node.hub.ReconnectPeerBySite(siteID)
}

func (node *ClusterNode) DisconnectRelayByPartition(partitionNumber uint64) {
    node.relayConnectionsMu.Lock()
    defer node.relayConnectionsMu.Unlock()

    node.hub.ReconnectPeerByPartition(partitionNumber)
}

func (node *ClusterNode) ClusterIO() clusterio.ClusterIOAgent {
    return node.clusterioAgent
}

func (node *ClusterNode) RelayStatus(relayID string) (RelayStatus, error) {
    var status RelayStatus

    _, relayAdded := node.configController.ClusterController().State.Relays[relayID]

    if !relayAdded {
        return RelayStatus{}, ERelayDoesNotExist
    }

    connected, ping := node.hub.PeerStatus(relayID)

    status.Connected = connected
    status.Ping = ping
    status.ConnectedTo = node.ID()
    status.Site = node.configController.ClusterController().RelaySite(relayID)

    return status, nil
}

func (node *ClusterNode) localSnapshot(snapshotIndex uint64, snapshotId string) error {
    return node.snapshotter.Snapshot(snapshotIndex, snapshotId)
}

type ClusterNodeFacade struct {
    node *ClusterNode
}

func (clusterFacade *ClusterNodeFacade) AddNode(ctx context.Context, nodeConfig NodeConfig) error {
    return clusterFacade.node.configController.AddNode(ctx, nodeConfig)
}

func (clusterFacade *ClusterNodeFacade) RemoveNode(ctx context.Context, nodeID uint64) error {
    return clusterFacade.node.configController.RemoveNode(ctx, nodeID)
}

func (clusterFacade *ClusterNodeFacade) ReplaceNode(ctx context.Context, nodeID uint64, replacementNodeID uint64) error {
    return clusterFacade.node.configController.ReplaceNode(ctx, nodeID, replacementNodeID)
}

func (clusterFacade *ClusterNodeFacade) ClusterClient() *client.Client {
    return clusterFacade.node.interClusterClient
}

func (clusterFacade *ClusterNodeFacade) Decommission() error {
    err, _ := clusterFacade.node.LeaveCluster()

    return err
}

func (clusterFacade *ClusterNodeFacade) DecommissionPeer(nodeID uint64) error {
    peerAddress := clusterFacade.PeerAddress(nodeID)

    if peerAddress.IsEmpty() {
        return errors.New("No address for peer")
    }

    return clusterFacade.node.interClusterClient.RemoveNode(context.TODO(), peerAddress, nodeID, 0, true, true)
}

func (clusterFacade *ClusterNodeFacade) LocalNodeID() uint64 {
    return clusterFacade.node.ID()
}

func (clusterFacade *ClusterNodeFacade) PeerAddress(nodeID uint64) PeerAddress {
    return clusterFacade.node.configController.ClusterController().ClusterMemberAddress(nodeID)
}

func (clusterFacade *ClusterNodeFacade) Batch(siteID string, bucket string, updateBatch *UpdateBatch) (BatchResult, error) {
    replicas, nApplied, err := clusterFacade.node.clusterioAgent.Batch(context.TODO(), siteID, bucket, updateBatch)

    if err == ESiteDoesNotExist {
        return BatchResult{}, ENoSuchSite
    }

    if err == EBucketDoesNotExist {
        return BatchResult{}, ENoSuchBucket
    }

    return BatchResult{
        Replicas: uint64(replicas),
        NApplied: uint64(nApplied),
    }, err
}

func (clusterFacade *ClusterNodeFacade) Get(siteID string, bucket string, keys [][]byte) ([]*SiblingSet, error) {
    siblingSets, err := clusterFacade.node.clusterioAgent.Get(context.TODO(), siteID, bucket, keys)

    if err == ESiteDoesNotExist {
        return nil, ENoSuchSite
    }

    if err == EBucketDoesNotExist {
        return nil, ENoSuchBucket
    }

    if err != nil {
        return nil, err
    }

    return siblingSets, nil
}

func (clusterFacade *ClusterNodeFacade) GetMatches(siteID string, bucket string, keys [][]byte) (SiblingSetIterator, error) {
    iter, err := clusterFacade.node.clusterioAgent.GetMatches(context.TODO(), siteID, bucket, keys)

    if err == ESiteDoesNotExist {
        return nil, ENoSuchSite
    }

    if err == EBucketDoesNotExist {
        return nil, ENoSuchBucket
    }

    if err != nil {
        return nil, err
    }

    return iter, nil
}

func (clusterFacade *ClusterNodeFacade) LocalGetMatches(partitionNumber uint64, siteID string, bucketName string, keys [][]byte) (SiblingSetIterator, error) {
    return clusterFacade.node.GetMatches(context.TODO(), partitionNumber, siteID, bucketName, keys)
}

func (clusterFacade *ClusterNodeFacade) LocalGet(partitionNumber uint64, siteID string, bucketName string, keys [][]byte) ([]*SiblingSet, error) {
    return clusterFacade.node.Get(context.TODO(), partitionNumber, siteID, bucketName, keys)
}

func (clusterFacade *ClusterNodeFacade) LocalBatch(partitionNumber uint64, siteID string, bucketName string, updateBatch *UpdateBatch) (map[string]*SiblingSet, error) {
    return clusterFacade.node.Batch(context.TODO(), partitionNumber, siteID, bucketName, updateBatch)
}

func (clusterFacade *ClusterNodeFacade) LocalMerge(partitionNumber uint64, siteID string, bucketName string, patch map[string]*SiblingSet, broadcastToRelays bool) error {
    return clusterFacade.node.Merge(context.TODO(), partitionNumber, siteID, bucketName, patch, broadcastToRelays)
}

func (clusterFacade *ClusterNodeFacade) AddRelay(ctx context.Context, relayID string) error {
    return clusterFacade.node.configController.ClusterCommand(ctx, ClusterAddRelayBody{ RelayID: relayID })
}

func (clusterFacade *ClusterNodeFacade) RemoveRelay(ctx context.Context, relayID string) error {
    return clusterFacade.node.configController.ClusterCommand(ctx, ClusterRemoveRelayBody{ RelayID: relayID })
}

func (clusterFacade *ClusterNodeFacade) MoveRelay(ctx context.Context, relayID string, siteID string) error {
    return clusterFacade.node.configController.ClusterCommand(ctx, ClusterMoveRelayBody{ RelayID: relayID, SiteID: siteID })
}

func (clusterFacade *ClusterNodeFacade) AddSite(ctx context.Context, siteID string) error {
    return clusterFacade.node.configController.ClusterCommand(ctx, ClusterAddSiteBody{ SiteID: siteID })
}

func (clusterFacade *ClusterNodeFacade) RemoveSite(ctx context.Context, siteID string) error {
    return clusterFacade.node.configController.ClusterCommand(ctx, ClusterRemoveSiteBody{ SiteID: siteID })
}

func (clusterFacade *ClusterNodeFacade) AcceptRelayConnection(conn *websocket.Conn, header http.Header) {
    clusterFacade.node.AcceptRelayConnection(conn, header)
}

func (clusterFacade *ClusterNodeFacade) ClusterNodes() []NodeConfig {
    var nodeConfigs []NodeConfig = clusterFacade.node.configController.ClusterController().ClusterNodeConfigs()

    for i, nodeConfig := range nodeConfigs {
        nodeConfigs[i] = NodeConfig{
            Address: nodeConfig.Address,
            Capacity: nodeConfig.Capacity,
        }
    }

    return nodeConfigs
}

func (clusterFacade *ClusterNodeFacade) ClusterSettings() ClusterSettings {
    return clusterFacade.node.configController.ClusterController().State.ClusterSettings
}

func (clusterFacade *ClusterNodeFacade) PartitionDistribution() [][]uint64 {
    var partitionDistribution [][]uint64 = make([][]uint64, clusterFacade.node.configController.ClusterController().State.ClusterSettings.Partitions)

    for partition, _ := range partitionDistribution {
        partitionDistribution[partition] = clusterFacade.node.configController.ClusterController().PartitionOwners(uint64(partition))
    }

    return partitionDistribution
}

func (clusterFacade *ClusterNodeFacade) TokenAssignments() []uint64 {
    return clusterFacade.node.configController.ClusterController().State.Tokens
}

func (clusterFacade *ClusterNodeFacade) GetRelayStatus(ctx context.Context, relayID string) (RelayStatus, error) {
    var siteID string = clusterFacade.node.configController.ClusterController().RelaySite(relayID)

    return clusterFacade.node.clusterioAgent.RelayStatus(ctx, siteID, relayID)
}

func (clusterFacade *ClusterNodeFacade) LocalGetRelayStatus(relayID string) (RelayStatus, error) {
    return clusterFacade.node.RelayStatus(relayID)
}

func (clusterFacade *ClusterNodeFacade) LocalLogDump() (LogDump, error) {
    var logDump LogDump

    baseSnapshot, entries, err := clusterFacade.node.configController.LogDump()

    if err != nil {
        Log.Errorf("Error while retrieving log dump: %v", err.Error())

        return LogDump{}, err
    }

    if !raft.IsEmptySnap(baseSnapshot) {
        var clusterState ClusterState

        err = clusterState.Recover(baseSnapshot.Data)

        if err != nil {
            Log.Errorf("Error while parsing base cluster snapshot: %v", err.Error())

            return LogDump{}, err
        }

        logDump.BaseSnapshot.Index = baseSnapshot.Metadata.Index
        logDump.BaseSnapshot.State = clusterState
    }

    logDump.Entries = make([]LogEntry, 0, len(entries))

    for _, entry := range entries {
        var logEntry LogEntry

        logEntry.Index = entry.Index

        switch entry.Type {
        case raftpb.EntryConfChange:
            var confChange raftpb.ConfChange
            var err error

            if err := confChange.Unmarshal(entry.Data); err != nil {
                Log.Errorf("Error while parsing committed entry: %v", err.Error())

                return LogDump{}, err
            }

            clusterCommand, err := DecodeClusterCommand(confChange.Context)

            if err != nil {
                Log.Errorf("Error while parsing committed entry: %v", err.Error())

                return LogDump{}, err
            }

            logEntry.Command = clusterCommand
        case raftpb.EntryNormal:
            if len(entry.Data) == 0 {
                continue
            }

            clusterCommand, err := DecodeClusterCommand(entry.Data)

            if err != nil {
                Log.Errorf("Error while parsing committed entry: %v", err.Error())

                return LogDump{}, err
            }

            logEntry.Command = clusterCommand
        }

        logDump.Entries = append(logDump.Entries, logEntry)
    }

    return logDump, nil
}

func (clusterFacade *ClusterNodeFacade) ClusterSnapshot(ctx context.Context) (Snapshot, error) {
    snapshotId, err := UUID()

    if err != nil {
        return Snapshot{}, err
    }

    if err := clusterFacade.node.configController.ClusterCommand(ctx, ClusterSnapshotBody{ UUID: snapshotId }); err != nil {
        return Snapshot{}, err
    }

    return Snapshot{UUID: snapshotId}, nil
}

func (clusterFacade *ClusterNodeFacade) CheckLocalSnapshotStatus(snapshotId string) error {
    return clusterFacade.node.snapshotter.CheckSnapshotStatus(snapshotId)
}

func (clusterFacade *ClusterNodeFacade) WriteLocalSnapshot(snapshotId string, w io.Writer) error {
    return clusterFacade.node.snapshotter.WriteSnapshot(snapshotId, w)
}