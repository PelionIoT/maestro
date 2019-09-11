package node

type NodeCoordinatorFacade struct {
    node *ClusterNode
}

func (nodeFacade *NodeCoordinatorFacade) ID() uint64 {
    return nodeFacade.node.ID()
}

func (nodeFacade *NodeCoordinatorFacade) AddPartition(partitionNumber uint64) {
    if nodeFacade.node.partitionPool.Get(partitionNumber) == nil {
        partition := nodeFacade.node.partitionFactory.CreatePartition(partitionNumber, nodeFacade.node.sitePool(partitionNumber))

        for siteID, _ := range nodeFacade.node.ClusterConfigController().ClusterController().State.Sites {
            if nodeFacade.node.configController.ClusterController().Partition(siteID) == partitionNumber {
                partition.Sites().Add(siteID)
            }
        }

        partition.LockReads()
        partition.LockWrites()
        nodeFacade.node.partitionPool.Add(partition)
    }
}

func (nodeFacade *NodeCoordinatorFacade) RemovePartition(partitionNumber uint64) {
    nodeFacade.node.partitionPool.Remove(partitionNumber)
}

func (nodeFacade *NodeCoordinatorFacade) EnableOutgoingTransfers(partitionNumber uint64) {
    nodeFacade.node.transferAgent.EnableOutgoingTransfers(partitionNumber)
}

func (nodeFacade *NodeCoordinatorFacade) DisableOutgoingTransfers(partitionNumber uint64) {
    nodeFacade.node.transferAgent.DisableOutgoingTransfers(partitionNumber)
}

func (nodeFacade *NodeCoordinatorFacade) StartIncomingTransfer(partitionNumber uint64, replicaNumber uint64) {
    nodeFacade.node.transferAgent.StartTransfer(partitionNumber, replicaNumber)
}

func (nodeFacade *NodeCoordinatorFacade) StopIncomingTransfer(partitionNumber uint64, replicaNumber uint64) {
    nodeFacade.node.transferAgent.StopTransfer(partitionNumber, replicaNumber)
}

func (nodeFacade *NodeCoordinatorFacade) LockPartitionWrites(partitionNumber uint64) {
    partition := nodeFacade.node.partitionPool.Get(partitionNumber)

    if partition != nil {
        partition.LockWrites()
    }
}

func (nodeFacade *NodeCoordinatorFacade) UnlockPartitionWrites(partitionNumber uint64) {
    partition := nodeFacade.node.partitionPool.Get(partitionNumber)

    if partition != nil {
        partition.UnlockWrites()
    }
}

func (nodeFacade *NodeCoordinatorFacade) LockPartitionReads(partitionNumber uint64) {
    partition := nodeFacade.node.partitionPool.Get(partitionNumber)

    if partition != nil {
        partition.LockReads()
    }
}

func (nodeFacade *NodeCoordinatorFacade) UnlockPartitionReads(partitionNumber uint64) {
    partition := nodeFacade.node.partitionPool.Get(partitionNumber)

    if partition != nil {
        partition.UnlockReads()
    }
}

func (nodeFacade *NodeCoordinatorFacade) AddSite(siteID string) {
    partitionNumber := nodeFacade.node.configController.ClusterController().Partition(siteID)
    partition := nodeFacade.node.partitionPool.Get(partitionNumber)

    if partition == nil {
        return
    }

    partition.Sites().Add(siteID)
}

func (nodeFacade *NodeCoordinatorFacade) RemoveSite(siteID string) {
    partitionNumber := nodeFacade.node.configController.ClusterController().Partition(siteID)
    partition := nodeFacade.node.partitionPool.Get(partitionNumber)

    if partition == nil {
        return
    }

    partition.Sites().Remove(siteID)
    nodeFacade.node.DisconnectRelayBySite(siteID)
}

func (nodeFacade *NodeCoordinatorFacade) AddRelay(relayID string) {
}

func (nodeFacade *NodeCoordinatorFacade) RemoveRelay(relayID string) {
    nodeFacade.node.DisconnectRelay(relayID)
}

func (nodeFacade *NodeCoordinatorFacade) MoveRelay(relayID string, siteID string) {
    nodeFacade.node.DisconnectRelay(relayID)
}

func (nodeFacade *NodeCoordinatorFacade) DisconnectRelays(partitionNumber uint64) {
    nodeFacade.node.DisconnectRelayByPartition(partitionNumber)
}

func (nodeFacade *NodeCoordinatorFacade) OwnedPartitionReplicas() map[uint64]map[uint64]bool {
    var ownedPartitionReplicas map[uint64]map[uint64]bool = make(map[uint64]map[uint64]bool, 0)

    for _, partitionReplica := range nodeFacade.node.configController.ClusterController().LocalNodeOwnedPartitionReplicas() {
        if _, ok := ownedPartitionReplicas[partitionReplica.Partition]; !ok {
            ownedPartitionReplicas[partitionReplica.Partition] = make(map[uint64]bool)
        }

        ownedPartitionReplicas[partitionReplica.Partition][partitionReplica.Replica] = true
    }

    return ownedPartitionReplicas
}

func (nodeFacade *NodeCoordinatorFacade) HeldPartitionReplicas() map[uint64]map[uint64]bool {
    var heldPartitionReplicas map[uint64]map[uint64]bool = make(map[uint64]map[uint64]bool, 0)

    for _, partitionReplica := range nodeFacade.node.configController.ClusterController().LocalNodeHeldPartitionReplicas() {
        if _, ok := heldPartitionReplicas[partitionReplica.Partition]; !ok {
            heldPartitionReplicas[partitionReplica.Partition] = make(map[uint64]bool)
        }

        heldPartitionReplicas[partitionReplica.Partition][partitionReplica.Replica] = true
    }

    return heldPartitionReplicas
}

func (nodeFacade *NodeCoordinatorFacade) NeighborsWithCapacity() int {
    var n int

    for _, node := range nodeFacade.node.configController.ClusterController().State.Nodes {
        if node.Capacity > 0 {
            n++
        }
    }

    return n
}

func (nodeFacade *NodeCoordinatorFacade) NotifyJoinedCluster() {
    nodeFacade.node.joinedCluster <- 1
}

func (nodeFacade *NodeCoordinatorFacade) NotifyLeftCluster() {
    nodeFacade.node.leftCluster <- 1
}
    
func (nodeFacade *NodeCoordinatorFacade) NotifyEmpty() {
    nodeFacade.node.notifyEmpty()
}