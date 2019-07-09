// How to gossip sites and relays??
// set<site>
// map<relay, site>
// Using gossip protocol for this association could result in inconsistencies for the client:
//
// Asks node 1 to add site so node 1 adds site to its local map. node 1 responds with a success code
// Before node 1 can gossip to other nodes the client asks node 2 something about site 1 and since node 2 has not yet heard about it it returns an error
// 
// 

package gossiper

type ClusterConfigurationListener interface {
    // Invoked after the node has joined the cluster
    OnJoined(func())
    // Invoked after the node has left or been forcibly removed from the cluster
    OnLeft(func())
    OnGainPartitionReplica(func(partition, replica uint64))
    OnLostPartitionReplica(func(partition, replica uint64))
    OnGainedPartitionReplicaOwnership(func(partition, replica uint64))
    OnLostPartitionReplicaOwnership(func(partition, replica uint64))
    OnSiteAdded(func(siteID string))
    OnSiteRemoved(func(siteID string))
    OnRelayAdded(func(relayID string))
    OnRelayRemoved(func(relayID string))
    OnRelayMoved(func(relayID string, siteID string))
}

type ClusterConfiguration interface {
    AddNode()
    RemoveNode()
    ReplaceNode()
}

type ClusterState interface {
    // Get the NodeState associated with this node
    Get(nodeID uint64) NodeState
    // Add a new node state for this node
    Add(nodeID uint64)
    Digest() map[uint64]NodeStateVersion
}

type NodeState interface {
    Tick()
    Heartbeat() NodeStateVersion
    Get(key string) NodeStateEntry
    Put(key string, value string)
    Version() NodeStateVersion
    LatestEntries(minVersion NodeStateVersion) []NodeStateEntry
}

type NodeStateEntry interface {
    Key() string
    Value() string
    Version() NodeStateVersion
}

type NodeStateVersion interface {
    Version() uint64
}