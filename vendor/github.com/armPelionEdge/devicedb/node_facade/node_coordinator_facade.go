package node_facade

type ClusterNodeCoordinatorFacade interface {
    ID() uint64
    // Create, initialize, and add a partition to the node's
    // partition pool if it isn't already there. Initializes the partition as both
    // read and write locked
    AddPartition(partitionNumber uint64)
    // Remove a partition from the node's partition pool.
    RemovePartition(partitionNumber uint64)
    // Allow other nodes to request a copy of this partition's
    // data
    EnableOutgoingTransfers(partitionNumber uint64)
    // Disallow other nodes from requesting a copy of this partition's
    // data. Cancel any current outgoing transfers for this partition
    DisableOutgoingTransfers(partitionNumber uint64)
    // Obtain a copy of this partition's data if necessary from another
    // node and then transfer holdership of this partition replica
    // to this node
    StartIncomingTransfer(partitionNumber uint64, replicaNumber uint64)
    // Stop any pending or ongoing transfers of this partition replica to
    // this node. Also cancel any downloads for this partition from 
    // any other replicas
    StopIncomingTransfer(partitionNumber uint64, replicaNumber uint64)
    // Ensure that no updates can occur to the local copy of this partition
    LockPartitionWrites(partitionNumber uint64)
    // Allow updates to the local copy of this partition
    UnlockPartitionWrites(partitionNumber uint64)
    // Ensure that the local copy of this partition cannot be read for
    // transfers or any other purpose
    LockPartitionReads(partitionNumber uint64)
    // Allow the local copy of this partition to serve reads
    UnlockPartitionReads(partitionNumber uint64)
    // Add site to the partition that it belongs to
    // if this node owns that partition
    AddSite(siteID string)
    // Remove site from the partition that it belongs to
    // if this node owns that partition. Disconnect any
    // relays that are in that site
    RemoveSite(siteID string)
    AddRelay(relayID string)
    RemoveRelay(relayID string)
    MoveRelay(relayID string, siteID string)
    DisconnectRelays(partitionNumber uint64)
    // Return a count of cluster members that have non-zero capacity
    NeighborsWithCapacity() int
    // Obtain a two dimensional map indicating which partition replicas are
    // currently owned by this node. map[partitionNumber][replicaNumber]
    OwnedPartitionReplicas() map[uint64]map[uint64]bool
    // Obtain a two dimensional map indicating which partition replicas are
    // currently held by this node. map[partitionNumber][replicaNumber]
    HeldPartitionReplicas() map[uint64]map[uint64]bool
    // Notify a node that it has been added to a cluster
    NotifyJoinedCluster()
    // Notify a node that it has been removed from a cluster
    NotifyLeftCluster()
    // Notify a node that it no longer owns or holds any partition replicas
    NotifyEmpty()
}