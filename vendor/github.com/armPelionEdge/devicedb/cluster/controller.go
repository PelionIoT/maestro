package cluster

import (
    "errors"
    "sync"
    "sort"
    "github.com/armPelionEdge/devicedb/raft"

    . "github.com/armPelionEdge/devicedb/logging"
)

var ENoSuchCommand = errors.New("The cluster command type is not supported")
var ENoSuchNode = errors.New("The node specified in the update does not exist")
var ENoSuchSite = errors.New("The specified site does not exist")
var ENoSuchRelay = errors.New("The specified relay does not exist")
var ENodeDoesNotOwnReplica = errors.New("A node tried to transfer a partition replica to itself but it no longer owns that replica")
var ECouldNotParseCommand = errors.New("The cluster command data was not properly formatted. Unable to parse it.")
var EReplicaNumberInvalid = errors.New("The command specified an invalid replica number for a partition.")

type ClusterController struct {
    LocalNodeID uint64
    State ClusterState
    PartitioningStrategy PartitioningStrategy
    LocalUpdates chan []ClusterStateDelta
    notificationsEnabled bool
    notificationsEnabledLock sync.Mutex
    stateUpdateLock sync.Mutex
    nextDeltaSet []ClusterStateDelta
    localNodeOwnedPartitionReplicaCache map[uint64]map[uint64]bool
    partitionOwnersCache map[uint64][]uint64
}

func (clusterController *ClusterController) Partition(key string) uint64 {
    if clusterController.State.ClusterSettings.AreInitialized() {
        return clusterController.PartitioningStrategy.Partition(key, clusterController.State.ClusterSettings.Partitions)
    }

    return 0
}

func (clusterController *ClusterController) EnableNotifications() {
    clusterController.notificationsEnabledLock.Lock()
    defer clusterController.notificationsEnabledLock.Unlock()
    clusterController.notificationsEnabled = true
}

func (clusterController *ClusterController) DisableNotifications() {
    clusterController.notificationsEnabledLock.Lock()
    defer clusterController.notificationsEnabledLock.Unlock()
    clusterController.notificationsEnabled = false
}

func (clusterController *ClusterController) Step(clusterCommand ClusterCommand) ([]ClusterStateDelta, error) {
    body, err := DecodeClusterCommandBody(clusterCommand)

    if err != nil {
        return nil, ECouldNotParseCommand
    }


    clusterController.stateUpdateLock.Lock()
    clusterController.nextDeltaSet = []ClusterStateDelta{ }

    switch clusterCommand.Type {
    case ClusterUpdateNode:
        err = clusterController.UpdateNodeConfig(body.(ClusterUpdateNodeBody))
    case ClusterAddNode:
        err = clusterController.AddNode(body.(ClusterAddNodeBody))
    case ClusterRemoveNode:
        err = clusterController.RemoveNode(body.(ClusterRemoveNodeBody))
    case ClusterTakePartitionReplica:
        err = clusterController.TakePartitionReplica(body.(ClusterTakePartitionReplicaBody))
    case ClusterSetReplicationFactor:
        err = clusterController.SetReplicationFactor(body.(ClusterSetReplicationFactorBody))
    case ClusterSetPartitionCount:
        err = clusterController.SetPartitionCount(body.(ClusterSetPartitionCountBody))
    case ClusterAddSite:
        err = clusterController.AddSite(body.(ClusterAddSiteBody))
    case ClusterRemoveSite:
        err = clusterController.RemoveSite(body.(ClusterRemoveSiteBody))
    case ClusterAddRelay:
        err = clusterController.AddRelay(body.(ClusterAddRelayBody))
    case ClusterRemoveRelay:
        err = clusterController.RemoveRelay(body.(ClusterRemoveRelayBody))
    case ClusterMoveRelay:
        err = clusterController.MoveRelay(body.(ClusterMoveRelayBody))
    case ClusterSnapshot:
        // Do nothing
        err = nil
    default:
        return nil, ENoSuchCommand
    }

    clusterController.stateUpdateLock.Unlock()
    clusterController.notificationsEnabledLock.Lock()

    if clusterController.notificationsEnabled {
        clusterController.LocalUpdates <- clusterController.nextDeltaSet
    }

    clusterController.notificationsEnabledLock.Unlock()

    return clusterController.nextDeltaSet, err
}

func (clusterController *ClusterController) Deltas() []ClusterStateDelta {
    return clusterController.nextDeltaSet
}

// Apply a snapshot to the state and notify on the local updates channel of any relevant
// changes
func (clusterController *ClusterController) ApplySnapshot(snap []byte) error {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()
    clusterController.clearPartitionOwnersCache()
    clusterController.localNodeOwnedPartitionReplicaCache = nil
    localNodeOwnedPartitionReplica := clusterController.localNodeOwnedPartitionReplicas()
    localNodeTokenSnapshot := clusterController.localNodeTokenSnapshot()
    localNodePartitionReplicaSnapshot := clusterController.localNodePartitionReplicaSnapshot()
    relaysSnapshot := clusterController.relaysSnapshot()
    sitesSnapshot := clusterController.sitesSnapshot()
    _, localNodeWasPresentBefore := clusterController.State.Nodes[clusterController.LocalNodeID]

    if err := clusterController.State.Recover(snap); err != nil {
        return err
    }

    nodeConfig, localNodeIsPresentNow := clusterController.State.Nodes[clusterController.LocalNodeID]

    if !localNodeWasPresentBefore && localNodeIsPresentNow {
        // This node was added. Provide an add node delta
        clusterController.notifyLocalNode(DeltaNodeAdd, NodeAdd{ NodeID: clusterController.LocalNodeID, NodeConfig: *nodeConfig })
    }

    clusterController.localDiffTokensAndNotify(localNodeTokenSnapshot)
    clusterController.localDiffOwnedPartitionReplicasAndNotify(localNodeOwnedPartitionReplica)
    clusterController.localDiffPartitionReplicasAndNotify(localNodePartitionReplicaSnapshot)
    clusterController.diffRelaysAndNotify(relaysSnapshot)
    clusterController.diffSitesAndNotify(sitesSnapshot)

    if localNodeWasPresentBefore && !localNodeIsPresentNow {
        // This node was removed. Provide a remove node delta
        clusterController.notifyLocalNode(DeltaNodeRemove, NodeRemove{ NodeID: clusterController.LocalNodeID })
    }

    return nil
}

func (clusterController *ClusterController) relaysSnapshot() map[string]string {
    var relays map[string]string = make(map[string]string)

    for relay, site := range clusterController.State.Relays {
        relays[relay] = site
    }

    return relays
}

func (clusterController *ClusterController) sitesSnapshot() map[string]bool {
    var sites map[string]bool = make(map[string]bool)

    for site, _ := range clusterController.State.Sites {
        sites[site] = true
    }

    return sites
}

func (clusterController *ClusterController) diffRelaysAndNotify(relaysSnapshot map[string]string) {
    for relay, site := range relaysSnapshot {
        if _, ok := clusterController.State.Relays[relay]; !ok {
            clusterController.notifyLocalNode(DeltaRelayRemoved, RelayRemoved{ RelayID: relay })
        } else if clusterController.State.Relays[relay] != site {
            clusterController.notifyLocalNode(DeltaRelayMoved, RelayMoved{ RelayID: relay, SiteID: clusterController.State.Relays[relay] })
        }
    }

    for relay, _ := range clusterController.State.Relays {
        if _, ok := relaysSnapshot[relay]; !ok {
            clusterController.notifyLocalNode(DeltaRelayAdded, RelayAdded{ RelayID: relay })
        }
    }
}

func (clusterController *ClusterController) diffSitesAndNotify(sitesSnapshot map[string]bool) {
    for site, _ := range sitesSnapshot {
        if _, ok := clusterController.State.Sites[site]; !ok {
            clusterController.notifyLocalNode(DeltaSiteRemoved, SiteRemoved{ SiteID: site })
        }
    }

    for site, _ := range clusterController.State.Sites {
        if _, ok := sitesSnapshot[site]; !ok {
            clusterController.notifyLocalNode(DeltaSiteAdded, SiteAdded{ SiteID: site })
        }
    }
}

func (clusterController *ClusterController) UpdateNodeConfig(clusterCommand ClusterUpdateNodeBody) error {
    currentNodeConfig, ok := clusterController.State.Nodes[clusterCommand.NodeID]

    if !ok {
        // No such node
        return nil
    }

    currentNodeConfig.Address.Host = clusterCommand.NodeConfig.Address.Host
    currentNodeConfig.Address.Port = clusterCommand.NodeConfig.Address.Port

    if clusterCommand.NodeConfig.Capacity != currentNodeConfig.Capacity {
        currentNodeConfig.Capacity = clusterCommand.NodeConfig.Capacity

        // a capacity change with any node means tokens need to be redistributed to account for different
        // relative capacity of the nodes. This has no effect with the simple partitioning strategy unless
        // a node has been assigned capacity 0 indicating that it is leaving the cluster soon

        if clusterController.State.ClusterSettings.AreInitialized() {
            clusterController.assignTokens()
        }
    }

    return nil
}

func (clusterController *ClusterController) AddNode(clusterCommand ClusterAddNodeBody) error {
    // If a node already exists with this node ID then this request should be ignored
    if _, ok := clusterController.State.Nodes[clusterCommand.NodeID]; ok {
        Log.Warningf("Ignoring request to add a node whose id = %d because a node with that ID already exists in the cluster", clusterCommand.NodeID)

        return raft.ECancelConfChange
    }

    // Ensure that a node ID is not reused if it was used by a node that used to belong to the cluster
    if clusterController.State.RemovedNodes != nil {
        if _, ok := clusterController.State.RemovedNodes[clusterCommand.NodeID]; ok {
            Log.Warningf("Ignoring request to add a node whose id = %d because a node with that ID used to exist in the cluster", clusterCommand.NodeID)

            return raft.ECancelConfChange
        }
    }

    // add the node if it isn't already added
    clusterCommand.NodeConfig.Tokens = make(map[uint64]bool)
    clusterCommand.NodeConfig.PartitionReplicas = make(map[uint64]map[uint64]bool)
    clusterCommand.NodeConfig.OwnedPartitionReplicas = make(map[uint64]map[uint64]bool)

    clusterController.State.AddNode(clusterCommand.NodeConfig)

    if clusterCommand.NodeID == clusterController.LocalNodeID {
        // notify the local node that it has been added to the cluster
        clusterController.notifyLocalNode(DeltaNodeAdd, NodeAdd{ NodeID: clusterController.LocalNodeID, NodeConfig: clusterCommand.NodeConfig })
    }

    // redistribute tokens in the cluster. tokens will be reassigned from other nodes to this node to distribute the load
    if clusterController.State.ClusterSettings.AreInitialized() {
        clusterController.assignTokens()
    }

    return nil
}

func (clusterController *ClusterController) RemoveNode(clusterCommand ClusterRemoveNodeBody) error {
    replacementNode, ok := clusterController.State.Nodes[clusterCommand.ReplacementNodeID]

    if (!ok && clusterCommand.ReplacementNodeID != 0) || (ok && len(replacementNode.Tokens) != 0) || clusterCommand.ReplacementNodeID == clusterCommand.NodeID {
        // configuration change should be cancelled if the replacement node does not exist, the node already has a token assignment or it is the node being removed
        return raft.ECancelConfChange
    }

    if _, ok := clusterController.State.Nodes[clusterCommand.NodeID]; ok {
        if ok && clusterCommand.ReplacementNodeID != 0 {
            // assign tokens that this node owned to another token
            clusterController.reassignTokens(clusterCommand.NodeID, clusterCommand.ReplacementNodeID)
        }

        // remove the node if it isn't already removed
        clusterController.State.RemoveNode(clusterCommand.NodeID)

        if (!ok || clusterCommand.ReplacementNodeID == 0) && clusterController.State.ClusterSettings.AreInitialized() {
            // redistribute tokens in the cluster, making sure to distribute tokens that were owned by this node to other nodes
            clusterController.assignTokens()
        }

        if clusterCommand.NodeID == clusterController.LocalNodeID {
            // notify the local node that it has been removed from the cluster
            clusterController.notifyLocalNode(DeltaNodeRemove, NodeRemove{ NodeID: clusterController.LocalNodeID })
        }
    }

    return nil
}

func (clusterController *ClusterController) TakePartitionReplica(clusterCommand ClusterTakePartitionReplicaBody) error {
    localNodePartitionReplicaSnapshot := clusterController.localNodePartitionReplicaSnapshot()

    partitionOwners := clusterController.partitionOwners(clusterCommand.Partition)

    if clusterCommand.Replica >= uint64(len(partitionOwners)) || len(partitionOwners) == 0 {
        // May be best to return an error
        return EReplicaNumberInvalid
    }

    if partitionOwners[int(clusterCommand.Replica)] != clusterCommand.NodeID {
        // If a node does not own a partition replica it cannot become the holder.
        // It is ok for a node to lose ownership and remain the holder but a node
        // must be the owner to hold it initially.
        return ENodeDoesNotOwnReplica
    }

    if err := clusterController.State.AssignPartitionReplica(clusterCommand.Partition, clusterCommand.Replica, clusterCommand.NodeID); err != nil {
        // Log Error
        return err
    }

    clusterController.localDiffPartitionReplicasAndNotify(localNodePartitionReplicaSnapshot)

    return nil
}

func (clusterController *ClusterController) localNodePartitionReplicaSnapshot() map[uint64]map[uint64]bool {
    nodeConfig, ok := clusterController.State.Nodes[clusterController.LocalNodeID]

    if !ok {
        return map[uint64]map[uint64]bool{ }
    }

    partitionReplicaSnapshot := make(map[uint64]map[uint64]bool, len(nodeConfig.PartitionReplicas))

    for partition, replicas := range nodeConfig.PartitionReplicas {
        partitionReplicaSnapshot[partition] = make(map[uint64]bool, len(replicas))

        for replica, _ := range replicas {
            partitionReplicaSnapshot[partition][replica] = true
        }
    }

    return partitionReplicaSnapshot
}

func (clusterController *ClusterController) localDiffPartitionReplicasAndNotify(partitionReplicaSnapshot map[uint64]map[uint64]bool) {
    nodeConfig, ok := clusterController.State.Nodes[clusterController.LocalNodeID]

    if !ok {
        return
    }

    // find out which partition replicas have been lost
    for partition, replicas := range partitionReplicaSnapshot {
        for replica, _ := range replicas {
            if _, ok := nodeConfig.PartitionReplicas[partition]; !ok {
                clusterController.notifyLocalNode(DeltaNodeLosePartitionReplica, NodeLosePartitionReplica{ NodeID: clusterController.LocalNodeID, Partition: partition, Replica: replica })

                continue
            }

            if _, ok := nodeConfig.PartitionReplicas[partition][replica]; !ok {
                clusterController.notifyLocalNode(DeltaNodeLosePartitionReplica, NodeLosePartitionReplica{ NodeID: clusterController.LocalNodeID, Partition: partition, Replica: replica })
            }
        }
    }

    // find out which partition replicas have been gained
    for partition, replicas := range nodeConfig.PartitionReplicas {
        for replica, _ := range replicas {
            if _, ok := partitionReplicaSnapshot[partition]; !ok {
                clusterController.notifyLocalNode(DeltaNodeGainPartitionReplica, NodeGainPartitionReplica{ NodeID: clusterController.LocalNodeID, Partition: partition, Replica: replica })

                continue
            }

            if _, ok := partitionReplicaSnapshot[partition][replica]; !ok {
                clusterController.notifyLocalNode(DeltaNodeGainPartitionReplica, NodeGainPartitionReplica{ NodeID: clusterController.LocalNodeID, Partition: partition, Replica: replica })
            }
        }
    }
}

func (clusterController *ClusterController) Lock() {
    clusterController.stateUpdateLock.Lock()
}

func (clusterController *ClusterController) Unlock() {
    clusterController.stateUpdateLock.Unlock()
}

func (clusterController *ClusterController) SetReplicationFactor(clusterCommand ClusterSetReplicationFactorBody) error {
    if clusterController.State.ClusterSettings.ReplicationFactor != 0 {
        // The replication factor has already been set and cannot be changed
        return nil
    }

    clusterController.State.ClusterSettings.ReplicationFactor = clusterCommand.ReplicationFactor
    clusterController.initializeClusterIfReady()

    return nil
}

func (clusterController *ClusterController) SetPartitionCount(clusterCommand ClusterSetPartitionCountBody) error {
    if clusterController.State.ClusterSettings.Partitions != 0 {
        // The partition count has already been set and cannot be changed
        return nil
    }

    clusterController.State.ClusterSettings.Partitions = clusterCommand.Partitions
    clusterController.initializeClusterIfReady()

    return nil
}

func (clusterController *ClusterController) AddSite(clusterCommand ClusterAddSiteBody) error {
    if clusterController.State.SiteExists(clusterCommand.SiteID) {
        return nil
    }

    clusterController.State.AddSite(clusterCommand.SiteID)
    clusterController.notifyLocalNode(DeltaSiteAdded, SiteAdded{ SiteID: clusterCommand.SiteID })

    return nil
}

func (clusterController *ClusterController) RemoveSite(clusterCommand ClusterRemoveSiteBody) error {
    if !clusterController.State.SiteExists(clusterCommand.SiteID) {
        return nil
    }

    clusterController.State.RemoveSite(clusterCommand.SiteID)
    clusterController.notifyLocalNode(DeltaSiteRemoved, SiteRemoved{ SiteID: clusterCommand.SiteID })

    return nil
}

func (clusterController *ClusterController) AddRelay(clusterCommand ClusterAddRelayBody) error {
    if _, ok := clusterController.State.Relays[clusterCommand.RelayID]; ok {
        return nil
    }

    clusterController.State.AddRelay(clusterCommand.RelayID)
    clusterController.notifyLocalNode(DeltaRelayAdded, RelayAdded{ RelayID: clusterCommand.RelayID })

    return nil
}

func (clusterController *ClusterController) RemoveRelay(clusterCommand ClusterRemoveRelayBody) error {
    if _, ok := clusterController.State.Relays[clusterCommand.RelayID]; !ok {
        return nil
    }

    clusterController.State.RemoveRelay(clusterCommand.RelayID)
    clusterController.notifyLocalNode(DeltaRelayRemoved, RelayRemoved{ RelayID: clusterCommand.RelayID })

    return nil
}

func (clusterController *ClusterController) MoveRelay(clusterCommand ClusterMoveRelayBody) error {
    if !clusterController.State.SiteExists(clusterCommand.SiteID) && clusterCommand.SiteID != "" {
        return ENoSuchSite
    }

    if _, ok := clusterController.State.Relays[clusterCommand.RelayID]; !ok {
        return ENoSuchRelay
    }

    if clusterController.State.Relays[clusterCommand.RelayID] == clusterCommand.SiteID {
        return nil
    }

    clusterController.State.MoveRelay(clusterCommand.RelayID, clusterCommand.SiteID)
    clusterController.notifyLocalNode(DeltaRelayMoved, RelayMoved{ RelayID: clusterCommand.RelayID, SiteID: clusterCommand.SiteID })

    return nil
}

func (clusterController *ClusterController) RelaySite(relayID string) string {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()

    return clusterController.State.Relays[relayID]
}

func (clusterController *ClusterController) ClusterIsInitialized() bool {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()

    return clusterController.State.ClusterSettings.AreInitialized()
}

func (clusterController *ClusterController) ClusterMemberAddress(nodeID uint64) raft.PeerAddress {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()

    nodeConfig, ok := clusterController.State.Nodes[nodeID]

    if !ok {
        return raft.PeerAddress{}
    }

    return nodeConfig.Address
}

func (clusterController *ClusterController) PartitionOwners(partition uint64) []uint64 {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()

    if partition >= uint64(len(clusterController.State.Partitions)) || !clusterController.State.ClusterSettings.AreInitialized() {
        return []uint64{ }
    }

    return clusterController.partitionOwners(partition)
}

func (clusterController *ClusterController) LocalNodeHoldsPartition(partition uint64) bool {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()

    return len(clusterController.State.Nodes[clusterController.LocalNodeID].PartitionReplicas[partition]) > 0
}

func (clusterController *ClusterController) PartitionHolders(partition uint64) []uint64 {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()

    if partition >= uint64(len(clusterController.State.Partitions)) {
        return []uint64{ }
    }

    replicas := clusterController.State.Partitions[partition]
    holders := make([]uint64, 0, len(replicas))
    holdersMap := make(map[uint64]bool, len(replicas))

    for _, partitionReplica := range replicas {
        if partitionReplica.Holder != 0 {
            if _, ok := holdersMap[partitionReplica.Holder]; !ok {
                holdersMap[partitionReplica.Holder] = true
                holders = append(holders, partitionReplica.Holder)
            }
        }
    }

    return holders
}

func (clusterController *ClusterController) SiteExists(siteID string) bool {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()

    _, ok := clusterController.State.Sites[siteID]

    return ok
}

func (clusterController *ClusterController) LocalNodeHeldPartitionReplicas() []PartitionReplica {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()

    partitionReplicas := make([]PartitionReplica, 0)

    if !clusterController.State.ClusterSettings.AreInitialized() {
        return partitionReplicas
    }

    if clusterController.State.Nodes[clusterController.LocalNodeID] == nil {
        return partitionReplicas
    }

    for partition, replicas := range clusterController.State.Nodes[clusterController.LocalNodeID].PartitionReplicas {
        for replica, _ := range replicas {
            partitionReplicas = append(partitionReplicas, PartitionReplica{ Partition: partition, Replica: replica })
        }
    }

    return partitionReplicas
}

func (clusterController *ClusterController) LocalNodeOwnedPartitionReplicas() []PartitionReplica {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()

    partitionReplicas := make([]PartitionReplica, 0)

    if !clusterController.State.ClusterSettings.AreInitialized() {
        return partitionReplicas
    }

    if clusterController.State.Nodes[clusterController.LocalNodeID] == nil {
        return partitionReplicas
    }

    for partition, replicas := range clusterController.localNodeOwnedPartitionReplicas() {
        for replica, _ := range replicas {
            partitionReplicas = append(partitionReplicas, PartitionReplica{ Partition: partition, Replica: replica })
        }
    }

    return partitionReplicas
}

func (clusterController *ClusterController) localNodeOwnedPartitionReplicas() map[uint64]map[uint64]bool {
    if !clusterController.State.ClusterSettings.AreInitialized() {
        return map[uint64]map[uint64]bool{ }
    }

    if clusterController.localNodeOwnedPartitionReplicaCache != nil {
        return clusterController.localNodeOwnedPartitionReplicaCache
    }

    partitionReplicas := make(map[uint64]map[uint64]bool)

    for i := 0; i < len(clusterController.State.Tokens); i++ {
        partitionOwners := clusterController.partitionOwners(uint64(i))

        for replica, nodeID := range partitionOwners {
            if nodeID == clusterController.LocalNodeID {
                if _, ok := partitionReplicas[uint64(i)]; !ok {
                    partitionReplicas[uint64(i)] = make(map[uint64]bool)
                }

                partitionReplicas[uint64(i)][uint64(replica)] = true
            }
        }
    }

    clusterController.localNodeOwnedPartitionReplicaCache = partitionReplicas

    //Log.Criticalf("Node %d owns partitions %v", clusterController.LocalNodeID, partitionReplicas)

    return partitionReplicas
}

func (clusterController *ClusterController) partitionOwners(partition uint64) []uint64 {
    if clusterController.partitionOwnersCache == nil {
        clusterController.partitionOwnersCache = make(map[uint64][]uint64)
    }

    if owners, ok := clusterController.partitionOwnersCache[partition]; ok {
        return owners
    }

    owners := clusterController.PartitioningStrategy.Owners(clusterController.State.Tokens, partition, clusterController.State.ClusterSettings.ReplicationFactor)

    clusterController.partitionOwnersCache[partition] = owners

    return owners
}

func (clusterController *ClusterController) clearPartitionOwnersCache() {
    clusterController.partitionOwnersCache = nil
}

func (clusterController *ClusterController) localDiffOwnedPartitionReplicasAndNotify(partitionReplicaSnapshot map[uint64]map[uint64]bool) {
    currentPartitionReplicas := clusterController.localNodeOwnedPartitionReplicas()

    // find out which partition replicas have been lost
    for partition, replicas := range partitionReplicaSnapshot {
        for replica, _ := range replicas {
            if _, ok := currentPartitionReplicas[partition]; !ok {
                clusterController.notifyLocalNode(DeltaNodeLosePartitionReplicaOwnership, NodeLosePartitionReplicaOwnership{ NodeID: clusterController.LocalNodeID, Partition: partition, Replica: replica })

                continue
            }

            if _, ok := currentPartitionReplicas[partition][replica]; !ok {
                clusterController.notifyLocalNode(DeltaNodeLosePartitionReplicaOwnership, NodeLosePartitionReplicaOwnership{ NodeID: clusterController.LocalNodeID, Partition: partition, Replica: replica })
            }
        }
    }

    // find out which partition replicas have been gained
    for partition, replicas := range currentPartitionReplicas {
        for replica, _ := range replicas {
            if _, ok := partitionReplicaSnapshot[partition]; !ok {
                clusterController.notifyLocalNode(DeltaNodeGainPartitionReplicaOwnership, NodeGainPartitionReplicaOwnership{ NodeID: clusterController.LocalNodeID, Partition: partition, Replica: replica })

                continue
            }

            if _, ok := partitionReplicaSnapshot[partition][replica]; !ok {
                clusterController.notifyLocalNode(DeltaNodeGainPartitionReplicaOwnership, NodeGainPartitionReplicaOwnership{ NodeID: clusterController.LocalNodeID, Partition: partition, Replica: replica })
            }
        }
    }
}

func (clusterController *ClusterController) LocalNodeConfig() *NodeConfig {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()

    return clusterController.State.Nodes[clusterController.LocalNodeID]
}

func (clusterController *ClusterController) LocalPartitionReplicasCount() int {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()

    replicaCount := 0
    localNodeConfig := clusterController.State.Nodes[clusterController.LocalNodeID]

    if localNodeConfig == nil {
        return replicaCount
    }

    for _, partitions := range localNodeConfig.PartitionReplicas {
        for _, _ = range partitions {
            replicaCount += 1
        }
    }

    return replicaCount
}

func (clusterController *ClusterController) LocalNodeIsInCluster() bool {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()
    
    _, ok := clusterController.State.Nodes[clusterController.LocalNodeID]

    return ok
}

func (clusterController *ClusterController) NodeIsInCluster(nodeID uint64) bool {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()
   
    _, ok := clusterController.State.Nodes[nodeID]

    return ok
}

func (clusterController *ClusterController) LocalNodeWasRemovedFromCluster() bool {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()

    _, ok := clusterController.State.RemovedNodes[clusterController.LocalNodeID]

    return ok
}

func (clusterController *ClusterController) ClusterNodes() map[uint64]bool {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()

    nodeMap := make(map[uint64]bool)

    for node, _ := range clusterController.State.Nodes {
        nodeMap[node] = true
    }

    return nodeMap
}

func (clusterController *ClusterController) ClusterNodeConfigs() []NodeConfig {
    clusterController.stateUpdateLock.Lock()
    defer clusterController.stateUpdateLock.Unlock()

    nodeConfigs := make([]NodeConfig, 0, len(clusterController.State.Nodes))

    for _, config := range clusterController.State.Nodes {
        nodeConfigs = append(nodeConfigs, *config)
    }

    return nodeConfigs
}

func (clusterController *ClusterController) initializeClusterIfReady() {
    if !clusterController.State.ClusterSettings.AreInitialized() {
        // the cluster settings have not been finalized so the cluster cannot yet be initialized
        return
    }

    clusterController.State.Initialize()
    clusterController.assignTokens()
}

func (clusterController *ClusterController) reassignTokens(oldOwnerID, newOwnerID uint64) {
    clusterController.localNodeOwnedPartitionReplicaCache = nil
    clusterController.clearPartitionOwnersCache()
    localNodeOwnedPartitionReplicas := clusterController.localNodeOwnedPartitionReplicas()
    localNodeTokenSnapshot := clusterController.localNodeTokenSnapshot()
    clusterController.localNodeOwnedPartitionReplicaCache = nil
    clusterController.clearPartitionOwnersCache()

    // make new owner match old owners capacity
    clusterController.State.Nodes[newOwnerID].Capacity = clusterController.State.Nodes[oldOwnerID].Capacity

    // move tokens from old owner to new owner
    for token, _ := range clusterController.State.Nodes[oldOwnerID].Tokens {
        clusterController.State.AssignToken(newOwnerID, token)
    }

    // perform diff between original token assignment and new token assignment to build deltas to place into update channel
    clusterController.localDiffTokensAndNotify(localNodeTokenSnapshot)
    clusterController.localDiffOwnedPartitionReplicasAndNotify(localNodeOwnedPartitionReplicas)
}

func (clusterController *ClusterController) assignTokens() {
    nodes := make([]NodeConfig, 0, len(clusterController.State.Nodes))

    for _, nodeConfig := range clusterController.State.Nodes {
        nodes = append(nodes, *nodeConfig)
    }

    sort.Sort(NodeConfigList(nodes))
    newTokenAssignment, _ := clusterController.PartitioningStrategy.AssignTokens(nodes, clusterController.State.Tokens, clusterController.State.ClusterSettings.Partitions)

    clusterController.localNodeOwnedPartitionReplicaCache = nil
    clusterController.clearPartitionOwnersCache()
    localNodeOwnedPartitionReplicas := clusterController.localNodeOwnedPartitionReplicas()
    localNodeTokenSnapshot := clusterController.localNodeTokenSnapshot()
    clusterController.localNodeOwnedPartitionReplicaCache = nil
    clusterController.clearPartitionOwnersCache()

    for token, owner := range newTokenAssignment {
        clusterController.State.AssignToken(owner, uint64(token))
    }

    // perform diff between original token assignment and new token assignment to build deltas to place into update channel
    clusterController.localDiffTokensAndNotify(localNodeTokenSnapshot)
    clusterController.localDiffOwnedPartitionReplicasAndNotify(localNodeOwnedPartitionReplicas)
}

func (clusterController *ClusterController) localNodeTokenSnapshot() map[uint64]bool {
    nodeConfig, ok := clusterController.State.Nodes[clusterController.LocalNodeID]

    if !ok {
        return map[uint64]bool{ }
    }

    tokenSnapshot := make(map[uint64]bool, len(nodeConfig.Tokens))

    for token, _ := range nodeConfig.Tokens {
        tokenSnapshot[token] = true
    }

    return tokenSnapshot
}

func (clusterController *ClusterController) localDiffTokensAndNotify(tokenSnapshot map[uint64]bool) {
    nodeConfig, ok := clusterController.State.Nodes[clusterController.LocalNodeID]

    if !ok {
        return
    }

    // find out which tokens have been lost
    for token, _ := range tokenSnapshot {
        if _, ok := nodeConfig.Tokens[token]; !ok {
            // this token was present in the original snapshot but is not there now
            clusterController.notifyLocalNode(DeltaNodeLoseToken, NodeLoseToken{ NodeID: clusterController.LocalNodeID, Token: token })
        }
    }

    // find out which tokens have been gained
    for token, _ := range nodeConfig.Tokens {
        if _, ok := tokenSnapshot[token]; !ok {
            // this token wasn't present in the original snapshot but is there now
            clusterController.notifyLocalNode(DeltaNodeGainToken, NodeGainToken{ NodeID: clusterController.LocalNodeID, Token: token })
        }
    }
}

// A channel that provides notifications for updates to configuration affecting the local node
// This includes gaining or losing ownership of tokens, gaining or losing ownership of partition
// replicas, becoming part of a cluster or being removed from a cluster
func (clusterController *ClusterController) notifyLocalNode(deltaType ClusterStateDeltaType, delta interface{ }) {
    clusterController.nextDeltaSet = append(clusterController.nextDeltaSet, ClusterStateDelta{ Type: deltaType, Delta: delta })
}
