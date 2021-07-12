package cluster
//
 // Copyright (c) 2019 ARM Limited.
 //
 // SPDX-License-Identifier: MIT
 //
 // Permission is hereby granted, free of charge, to any person obtaining a copy
 // of this software and associated documentation files (the "Software"), to
 // deal in the Software without restriction, including without limitation the
 // rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
 // sell copies of the Software, and to permit persons to whom the Software is
 // furnished to do so, subject to the following conditions:
 //
 // The above copyright notice and this permission notice shall be included in all
 // copies or substantial portions of the Software.
 //
 // THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 // IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 // FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 // AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 // LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 // OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 // SOFTWARE.
 //


import (
    ddbRaft "github.com/armPelionEdge/devicedb/raft"
)

type ClusterStateDeltaType int

const (
    DeltaNodeAdd ClusterStateDeltaType = iota
    DeltaNodeRemove ClusterStateDeltaType = iota
    DeltaNodeLoseToken ClusterStateDeltaType = iota
    DeltaNodeGainToken ClusterStateDeltaType = iota
    DeltaNodeGainPartitionReplicaOwnership ClusterStateDeltaType = iota
    DeltaNodeLosePartitionReplicaOwnership ClusterStateDeltaType = iota
    DeltaNodeLosePartitionReplica ClusterStateDeltaType = iota
    DeltaNodeGainPartitionReplica ClusterStateDeltaType = iota
    DeltaSiteAdded ClusterStateDeltaType = iota
    DeltaSiteRemoved ClusterStateDeltaType = iota
    DeltaRelayAdded ClusterStateDeltaType = iota
    DeltaRelayRemoved ClusterStateDeltaType = iota
    DeltaRelayMoved ClusterStateDeltaType = iota
)

type ClusterStateDeltaRange []ClusterStateDelta

func (r ClusterStateDeltaRange) Len() int {
    return len(r)
}

func (r ClusterStateDeltaRange) Swap(i, j int) {
    r[i], r[j] = r[j], r[i]
}

func (r ClusterStateDeltaRange) Less(i, j int) bool {
    if r[i].Type != r[j].Type {
        return r[i].Type < r[j].Type
    }

    switch r[i].Type {
    case DeltaNodeAdd:
        return r[i].Delta.(NodeAdd).NodeID < r[j].Delta.(NodeAdd).NodeID
    case DeltaNodeRemove:
        return r[i].Delta.(NodeRemove).NodeID < r[j].Delta.(NodeRemove).NodeID
    case DeltaNodeLoseToken:
        if r[i].Delta.(NodeLoseToken).NodeID != r[j].Delta.(NodeLoseToken).NodeID {
            return r[i].Delta.(NodeLoseToken).NodeID < r[j].Delta.(NodeLoseToken).NodeID
        }

        return r[i].Delta.(NodeLoseToken).Token < r[j].Delta.(NodeLoseToken).Token
    case DeltaNodeGainToken:
        if r[i].Delta.(NodeGainToken).NodeID != r[j].Delta.(NodeGainToken).NodeID {
            return r[i].Delta.(NodeGainToken).NodeID < r[j].Delta.(NodeGainToken).NodeID
        }

        return r[i].Delta.(NodeGainToken).Token < r[j].Delta.(NodeGainToken).Token
    case DeltaNodeGainPartitionReplicaOwnership:
        if r[i].Delta.(NodeGainPartitionReplicaOwnership).NodeID != r[j].Delta.(NodeGainPartitionReplicaOwnership).NodeID {
            return r[i].Delta.(NodeGainPartitionReplicaOwnership).NodeID < r[j].Delta.(NodeGainPartitionReplicaOwnership).NodeID
        }

        if r[i].Delta.(NodeGainPartitionReplicaOwnership).Partition != r[j].Delta.(NodeGainPartitionReplicaOwnership).Partition {
            return r[i].Delta.(NodeGainPartitionReplicaOwnership).Partition < r[j].Delta.(NodeGainPartitionReplicaOwnership).Partition
        }

        return r[i].Delta.(NodeGainPartitionReplicaOwnership).Replica < r[j].Delta.(NodeGainPartitionReplicaOwnership).Replica
    case DeltaNodeLosePartitionReplicaOwnership:
        if r[i].Delta.(NodeLosePartitionReplicaOwnership).NodeID != r[j].Delta.(NodeLosePartitionReplicaOwnership).NodeID {
            return r[i].Delta.(NodeLosePartitionReplicaOwnership).NodeID < r[j].Delta.(NodeLosePartitionReplicaOwnership).NodeID
        }

        if r[i].Delta.(NodeLosePartitionReplicaOwnership).Partition != r[j].Delta.(NodeLosePartitionReplicaOwnership).Partition {
            return r[i].Delta.(NodeLosePartitionReplicaOwnership).Partition < r[j].Delta.(NodeLosePartitionReplicaOwnership).Partition
        }

        return r[i].Delta.(NodeLosePartitionReplicaOwnership).Replica < r[j].Delta.(NodeLosePartitionReplicaOwnership).Replica
    case DeltaNodeGainPartitionReplica:
        if r[i].Delta.(NodeGainPartitionReplica).NodeID != r[j].Delta.(NodeGainPartitionReplica).NodeID {
            return r[i].Delta.(NodeGainPartitionReplica).NodeID < r[j].Delta.(NodeGainPartitionReplica).NodeID
        }

        if r[i].Delta.(NodeGainPartitionReplica).Partition != r[j].Delta.(NodeGainPartitionReplica).Partition {
            return r[i].Delta.(NodeGainPartitionReplica).Partition < r[j].Delta.(NodeGainPartitionReplica).Partition
        }

        return r[i].Delta.(NodeGainPartitionReplica).Replica < r[j].Delta.(NodeGainPartitionReplica).Replica
    case DeltaNodeLosePartitionReplica:
        if r[i].Delta.(NodeLosePartitionReplica).NodeID != r[j].Delta.(NodeLosePartitionReplica).NodeID {
            return r[i].Delta.(NodeLosePartitionReplica).NodeID < r[j].Delta.(NodeLosePartitionReplica).NodeID
        }

        if r[i].Delta.(NodeLosePartitionReplica).Partition != r[j].Delta.(NodeLosePartitionReplica).Partition {
            return r[i].Delta.(NodeLosePartitionReplica).Partition < r[j].Delta.(NodeLosePartitionReplica).Partition
        }

        return r[i].Delta.(NodeLosePartitionReplica).Replica < r[j].Delta.(NodeLosePartitionReplica).Replica
    case DeltaSiteAdded:
        return r[i].Delta.(SiteAdded).SiteID < r[j].Delta.(SiteAdded).SiteID
    case DeltaSiteRemoved:
        return r[i].Delta.(SiteRemoved).SiteID < r[j].Delta.(SiteRemoved).SiteID
    case DeltaRelayAdded:
        return r[i].Delta.(RelayAdded).RelayID < r[j].Delta.(RelayAdded).RelayID
    case DeltaRelayRemoved:
        return r[i].Delta.(RelayRemoved).RelayID < r[j].Delta.(RelayRemoved).RelayID
    case DeltaRelayMoved:
        if r[i].Delta.(RelayMoved).RelayID != r[j].Delta.(RelayMoved).RelayID {
            return r[i].Delta.(RelayMoved).RelayID < r[j].Delta.(RelayMoved).RelayID
        }

        return r[i].Delta.(RelayMoved).SiteID < r[j].Delta.(RelayMoved).SiteID
    }

    return false
}

type ClusterStateDelta struct {
    Type ClusterStateDeltaType
    Delta interface{}
}

type NodeAdd struct {
    NodeID uint64
    NodeConfig NodeConfig
}

type NodeRemove struct {
    NodeID uint64
}

type NodeAddress struct {
    NodeID uint64
    Address ddbRaft.PeerAddress
}

type NodeGainToken struct {
    NodeID uint64
    Token uint64
}

type NodeLoseToken struct {
    NodeID uint64
    Token uint64
}

type NodeGainPartitionReplicaOwnership struct {
    NodeID uint64
    Partition uint64
    Replica uint64
}

type NodeLosePartitionReplicaOwnership struct {
    NodeID uint64
    Partition uint64
    Replica uint64
}

type NodeGainPartitionReplica struct {
    NodeID uint64
    Partition uint64
    Replica uint64
}

type NodeLosePartitionReplica struct {
    NodeID uint64
    Partition uint64
    Replica uint64
}

type SiteAdded struct {
    SiteID string
}

type SiteRemoved struct {
    SiteID string
}

type RelayAdded struct {
    RelayID string
}

type RelayRemoved struct {
    RelayID string
}

type RelayMoved struct {
    RelayID string
    SiteID string
}