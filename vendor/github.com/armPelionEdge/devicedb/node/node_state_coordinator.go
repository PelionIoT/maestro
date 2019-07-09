package node

import (
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/node_facade"
    . "github.com/armPelionEdge/devicedb/logging"
)

type ClusterNodeStateCoordinator struct {
    nodeFacade ClusterNodeCoordinatorFacade
    partitionUpdater ClusterNodePartitionUpdater
}

func NewClusterNodeStateCoordinator(nodeFacade ClusterNodeCoordinatorFacade, partitionUpdater ClusterNodePartitionUpdater) *ClusterNodeStateCoordinator {
    if partitionUpdater == nil {
        // allow dependency injection for unit testing but by default use default implementation
        partitionUpdater = NewNodePartitionUpdater(nodeFacade)
    }

    return &ClusterNodeStateCoordinator{
        nodeFacade: nodeFacade,
        partitionUpdater: partitionUpdater,
    }
}

func (coordinator *ClusterNodeStateCoordinator) InitializeNodeState() {
    ownedPartitionReplicas := coordinator.nodeFacade.OwnedPartitionReplicas()
    heldPartitionReplicas := coordinator.nodeFacade.HeldPartitionReplicas()

    for partitionNumber, _ := range heldPartitionReplicas {
        coordinator.partitionUpdater.UpdatePartition(partitionNumber)
    }

    for partitionNumber, _ := range ownedPartitionReplicas {
        coordinator.partitionUpdater.UpdatePartition(partitionNumber)
    }

    coordinator.startTransfers()
}

func (coordinator *ClusterNodeStateCoordinator) startTransfers() {
    ownedPartitionReplicas := coordinator.nodeFacade.OwnedPartitionReplicas()
    heldPartitionReplicas := coordinator.nodeFacade.HeldPartitionReplicas()

    for partitionNumber, replicas := range ownedPartitionReplicas {
        for replicaNumber, _ := range replicas {
            if heldPartitionReplicas[partitionNumber] != nil && heldPartitionReplicas[partitionNumber][replicaNumber] {
                // Since the node already holds and owns this partition replica there is nothing to do
                continue
            }

            if heldPartitionReplicas[partitionNumber] == nil || !heldPartitionReplicas[partitionNumber][replicaNumber] {
                // This indicates that the node has not yet fully transferred the partition form its old holder
                // start transfer initiates the transfer from the old owner and starts downloading data
                // if another download is not already in progress for that partition
                coordinator.nodeFacade.StartIncomingTransfer(partitionNumber, replicaNumber)
            }
        }
    }
}

func (coordinator *ClusterNodeStateCoordinator) ProcessClusterUpdates(deltas []ClusterStateDelta) {
    for _, delta := range deltas {
        switch delta.Type {
        case DeltaNodeAdd:
            Log.Infof("Local node (id = %d) was added to a cluster.", coordinator.nodeFacade.ID())
            coordinator.nodeFacade.NotifyJoinedCluster()
        case DeltaNodeRemove:
            Log.Infof("Local node (id = %d) was removed from its cluster. It will now shut down...", coordinator.nodeFacade.ID())
            coordinator.nodeFacade.NotifyLeftCluster()
        case DeltaNodeGainPartitionReplica:
            partition := delta.Delta.(NodeGainPartitionReplica).Partition
            replica := delta.Delta.(NodeGainPartitionReplica).Replica

            Log.Infof("Local node (id = %d) gained a partition replica (%d, %d)", coordinator.nodeFacade.ID(), partition, replica)

            coordinator.partitionUpdater.UpdatePartition(partition)
        case DeltaNodeLosePartitionReplica:
            partition := delta.Delta.(NodeLosePartitionReplica).Partition
            replica := delta.Delta.(NodeLosePartitionReplica).Replica

            Log.Infof("Local node (id = %d) lost a partition replica (%d, %d)", coordinator.nodeFacade.ID(), partition, replica)

            coordinator.partitionUpdater.UpdatePartition(partition)
        case DeltaNodeGainPartitionReplicaOwnership:
            partition := delta.Delta.(NodeGainPartitionReplicaOwnership).Partition
            replica := delta.Delta.(NodeGainPartitionReplicaOwnership).Replica

            Log.Infof("Local node (id = %d) gained ownership over a partition replica (%d, %d)", coordinator.nodeFacade.ID(), partition, replica)

            coordinator.partitionUpdater.UpdatePartition(partition)
            coordinator.nodeFacade.StartIncomingTransfer(partition, replica)
        case DeltaNodeLosePartitionReplicaOwnership:
            partition := delta.Delta.(NodeLosePartitionReplicaOwnership).Partition
            replica := delta.Delta.(NodeLosePartitionReplicaOwnership).Replica

            Log.Infof("Local node (id = %d) lost ownership over a partition replica (%d, %d)", coordinator.nodeFacade.ID(), partition, replica)

            coordinator.nodeFacade.StopIncomingTransfer(partition, replica)
            coordinator.partitionUpdater.UpdatePartition(partition)
        case DeltaSiteAdded:
            site := delta.Delta.(SiteAdded).SiteID
            coordinator.nodeFacade.AddSite(site)
        case DeltaSiteRemoved:
            site := delta.Delta.(SiteRemoved).SiteID
            coordinator.nodeFacade.RemoveSite(site)
        case DeltaRelayAdded:
            relay := delta.Delta.(RelayAdded).RelayID
            coordinator.nodeFacade.AddRelay(relay)
        case DeltaRelayRemoved:
            relay := delta.Delta.(RelayRemoved).RelayID
            coordinator.nodeFacade.RemoveRelay(relay)
        case DeltaRelayMoved:
            relay := delta.Delta.(RelayMoved).RelayID
            site := delta.Delta.(RelayMoved).SiteID
            coordinator.nodeFacade.MoveRelay(relay, site)
        }
    }

    if len(coordinator.nodeFacade.OwnedPartitionReplicas()) == 0 && (len(coordinator.nodeFacade.HeldPartitionReplicas()) == 0 || coordinator.nodeFacade.NeighborsWithCapacity() == 0) {
        coordinator.nodeFacade.NotifyEmpty()
    }
}

type ClusterNodePartitionUpdater interface {
    UpdatePartition(partitionNumber uint64)
}

type NodePartitionUpdater struct {
    nodeFacade ClusterNodeCoordinatorFacade
}

func NewNodePartitionUpdater(nodeFacade ClusterNodeCoordinatorFacade) *NodePartitionUpdater {
    return &NodePartitionUpdater{
        nodeFacade: nodeFacade,
    }
}

func (partitionUpdater *NodePartitionUpdater) UpdatePartition(partitionNumber uint64) {
    // does this node hold some replica of this partition?
    nodeHoldsPartition := len(partitionUpdater.nodeFacade.HeldPartitionReplicas()[partitionNumber]) > 0
    // does this node own some replica of this partition?
    nodeOwnsPartition := len(partitionUpdater.nodeFacade.OwnedPartitionReplicas()[partitionNumber]) > 0

    if !nodeOwnsPartition && !nodeHoldsPartition {
        Log.Infof("Local node (id = %d) no longer owns or holds any replica of partition %d. It will remove this partition from its local store", partitionUpdater.nodeFacade.ID(), partitionNumber)

        partitionUpdater.nodeFacade.DisconnectRelays(partitionNumber)
        partitionUpdater.nodeFacade.DisableOutgoingTransfers(partitionNumber)
        partitionUpdater.nodeFacade.LockPartitionReads(partitionNumber)
        partitionUpdater.nodeFacade.LockPartitionWrites(partitionNumber)
        partitionUpdater.nodeFacade.RemovePartition(partitionNumber)

        return
    }

    partitionUpdater.nodeFacade.AddPartition(partitionNumber)

    if nodeOwnsPartition {
        partitionUpdater.nodeFacade.UnlockPartitionWrites(partitionNumber)
    } else {
        partitionUpdater.nodeFacade.DisconnectRelays(partitionNumber)
        partitionUpdater.nodeFacade.LockPartitionWrites(partitionNumber)
    }

    if nodeHoldsPartition {
        // allow reads so this partition data can be transferred to another node
        // or this node can serve reads for this partition
        partitionUpdater.nodeFacade.UnlockPartitionReads(partitionNumber)
        partitionUpdater.nodeFacade.EnableOutgoingTransfers(partitionNumber)
    } else {
        // lock reads until this node has finalized a partition transfer
        partitionUpdater.nodeFacade.DisableOutgoingTransfers(partitionNumber)
        partitionUpdater.nodeFacade.LockPartitionReads(partitionNumber)
    }
}