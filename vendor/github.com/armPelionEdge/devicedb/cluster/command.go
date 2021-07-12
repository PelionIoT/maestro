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
    "encoding/json"
)

type ClusterCommandType int

// Scenarios
// Adding a node to a cluster
// Decommission a node (graceful)
// Removing a node (forced)
// Repairing a node in a cluster
// Increasing a node's capacity
// Decreasing a node's capacity
//
// Adding a node to a cluster
//   ***all changes should be temporary until committed by the release. We should be able to abort the process at any point with no effect
//   (phase i synchronized)
//   Start ring transaction
//   Propose update node capacity
//   Update node capacity committed
//   Take tokens committed
//   (phase ii async)
//   whenever token allocation is changed check to see if this node needs to transfer over any partitions
//   when i have all the partitions i need propose update to partition holder state until it gets committed
//
// Partition owner vs partition holder
// Partition owner is derived from the token distribution and describes which nodes are responsible for which partitions
// Partition holder is the actual map of which nodes hold the authoritative data for which partitions. Since the token distribution might change many times
// between background partition transfers, you need to keep track of these things independently
//
// After token change I need a partition I don't currently have
// I know who has the partition I need by looking at the holders map
// I ask the node that holds this partition to give me a copy
// When the transfer is complete I propose an entry that makes me the holder of this partition
// I can only accept writes to the partition after holder assignment has been committed
//
// What do we even put in the log?
// For now lets assume only token state changes go into the log
//   1) Some nodes may have an old view of the token distribution at any given time
//   2) If a node has log entry 8 they also have log entries 1-7
//   3) How do we prevent a partition from being transferred to a node that no longer owns it if that node hasn't yet gotten the update that it no longer owns it?
//   4) How do we reliably keep track of who has a copy of which partition
//   5) Latest log entry index can be used as an indication of the latest configuration version that a node is aware of (a logical clock)
//   6) A node may receive a configuration change and then query other nodes to initiate a partition transfer. It should include its configuration version.
//      the other nodes should inform it if their config is newer and do not agree that the requester is an owner of this partition so it can cancel its transfer
//   7) A node may receive a new configuration change while it's in the middle of a transfer that makes that partition no longer owned by it. In that case it should
//      cancel the transfer
//   8) Accepting updates to a partition though?? When to lock or unlock it for writes. Maybe we just don't care. Maybe we just merge conflicting partition histories?
// a) Propose conf change for partition ownership change
// b) Propose conf change for partition holder change
//
// A node locks updates to a partition when they no longer owns it. In other words when it receives a committed (a)
// A node keeps requesting transfer of a partition from its current holder until the transfer is complete or it receives an (a) in its commit log transferring ownership away from it
//    the current holder will continually refuse the transfer as long as the partition is not write lock. In other words as long as the current holder also believes themselves to be
//    the owner they will not transfer it.
// A node only proposes (b) if they have fully transferred the partition from what they believe is the current holder
//
// Node receiving (a) only commits those changes to the config if it doesn't conflict with the allocation constraints
// Node receiving (b):
//    rule 1) if the node receiving the holder transfer thinks that the proposer of that holder transfer is not the owner then the holder transfer is not commmited to the config
//    rule 2) if the proposer == the owner then make that node the holder as well
//    rule 3) if the node receiving (b) is the node that proposed it and it is still the owner then unlock the partition
// 
// Safety guarantees
//   1) As long as a primary doesn't disappear forever there can be no lost updates
//   2) A partition is unlocked at at most one node at a time
//
// 
// 

const (
    ClusterUpdateNode ClusterCommandType = iota
    ClusterAddNode ClusterCommandType = iota
    ClusterRemoveNode ClusterCommandType = iota
    ClusterTakePartitionReplica ClusterCommandType = iota
    ClusterSetReplicationFactor ClusterCommandType = iota
    ClusterSetPartitionCount ClusterCommandType = iota

    ClusterAddSite ClusterCommandType = iota
    ClusterRemoveSite ClusterCommandType = iota
    ClusterAddRelay ClusterCommandType = iota
    ClusterRemoveRelay ClusterCommandType = iota
    ClusterMoveRelay ClusterCommandType = iota
    ClusterSnapshot ClusterCommandType = iota
)

type ClusterCommand struct {
    Type ClusterCommandType
	SubmitterID uint64
	CommandID uint64
    Data []byte
}

type ClusterUpdateNodeBody struct {
    NodeID uint64
    NodeConfig NodeConfig
}

type ClusterAddNodeBody struct {
    NodeID uint64
    NodeConfig NodeConfig
}

type ClusterRemoveNodeBody struct {
	NodeID uint64
	ReplacementNodeID uint64
}

type ClusterTakePartitionReplicaBody struct {
    Partition uint64
    Replica uint64
    NodeID uint64
}

type ClusterSetReplicationFactorBody struct {
    ReplicationFactor uint64
}

type ClusterSetPartitionCountBody struct {
    Partitions uint64
}

type ClusterAddSiteBody struct {
    SiteID string
}

type ClusterRemoveSiteBody struct {
    SiteID string
}

type ClusterAddRelayBody struct {
    RelayID string
}

type ClusterRemoveRelayBody struct {
    RelayID string
}

type ClusterMoveRelayBody struct {
    RelayID string
    SiteID string
}

type ClusterSnapshotBody struct {
    UUID string
}

func EncodeClusterCommand(command ClusterCommand) ([]byte, error) {
    encodedCommand, err := json.Marshal(command)

    if err != nil {
        return nil, err
    }

    return encodedCommand, nil
}

func DecodeClusterCommand(encodedCommand []byte) (ClusterCommand, error) {
    var command ClusterCommand

    err := json.Unmarshal(encodedCommand, &command)

    if err != nil {
        return ClusterCommand{}, err
    }

    return command, nil
}

func CreateClusterCommand(commandType ClusterCommandType, body interface{}) (ClusterCommand, error) {
    encodedBody, err := json.Marshal(body)

    if err != nil {
        return ClusterCommand{}, ECouldNotParseCommand
    }

    switch commandType {
    case ClusterUpdateNode:
        if _, ok := body.(ClusterUpdateNodeBody); !ok {
            return ClusterCommand{}, ECouldNotParseCommand
        }
    case ClusterAddNode:
        if _, ok := body.(ClusterAddNodeBody); !ok {
            return ClusterCommand{}, ECouldNotParseCommand
        }
    case ClusterRemoveNode:
        if _, ok := body.(ClusterRemoveNodeBody); !ok {
            return ClusterCommand{}, ECouldNotParseCommand
        }
    case ClusterTakePartitionReplica:
        if _, ok := body.(ClusterTakePartitionReplicaBody); !ok {
            return ClusterCommand{}, ECouldNotParseCommand
        }
    case ClusterSetReplicationFactor:
        if _, ok := body.(ClusterSetReplicationFactorBody); !ok {
            return ClusterCommand{}, ECouldNotParseCommand
        }
    case ClusterSetPartitionCount:
        if _, ok := body.(ClusterSetPartitionCountBody); !ok {
            return ClusterCommand{}, ECouldNotParseCommand
        }
    case ClusterAddSite:
        if _, ok := body.(ClusterAddSiteBody); !ok {
            return ClusterCommand{}, ECouldNotParseCommand
        }
    case ClusterRemoveSite:
        if _, ok := body.(ClusterRemoveSiteBody); !ok {
            return ClusterCommand{}, ECouldNotParseCommand
        }
    case ClusterAddRelay:
        if _, ok := body.(ClusterAddRelayBody); !ok {
            return ClusterCommand{}, ECouldNotParseCommand
        }
    case ClusterRemoveRelay:
        if _, ok := body.(ClusterRemoveRelayBody); !ok {
            return ClusterCommand{}, ECouldNotParseCommand
        }
    case ClusterMoveRelay:
        if _, ok := body.(ClusterMoveRelayBody); !ok {
            return ClusterCommand{}, ECouldNotParseCommand
        }
    case ClusterSnapshot:
        if _, ok := body.(ClusterSnapshotBody); !ok {
            return ClusterCommand{}, ECouldNotParseCommand
        }
    default:
        return ClusterCommand{ }, ENoSuchCommand
    }

    return ClusterCommand{ Type: commandType, Data: encodedBody }, nil
}

func EncodeClusterCommandBody(body interface{}) ([]byte, error) {
    encodedBody, err := json.Marshal(body)

    if err != nil {
        return nil, ECouldNotParseCommand
    }

    return encodedBody, nil
}

func DecodeClusterCommandBody(command ClusterCommand) (interface{}, error) {
    switch command.Type {
    case ClusterUpdateNode:
        var body ClusterUpdateNodeBody

        if err := json.Unmarshal(command.Data, &body); err != nil {
            break
        }

        return body, nil
    case ClusterAddNode:
        var body ClusterAddNodeBody

        if err := json.Unmarshal(command.Data, &body); err != nil {
            break
        }

        return body, nil
    case ClusterRemoveNode:
        var body ClusterRemoveNodeBody

        if err := json.Unmarshal(command.Data, &body); err != nil {
            break
        }

        return body, nil
    case ClusterTakePartitionReplica:
        var body ClusterTakePartitionReplicaBody

        if err := json.Unmarshal(command.Data, &body); err != nil {
            break
        }

        return body, nil
    case ClusterSetReplicationFactor:
        var body ClusterSetReplicationFactorBody

        if err := json.Unmarshal(command.Data, &body); err != nil {
            break
        }

        return body, nil
    case ClusterSetPartitionCount:
        var body ClusterSetPartitionCountBody

        if err := json.Unmarshal(command.Data, &body); err != nil {
            break
        }

        return body, nil
    case ClusterAddSite:
        var body ClusterAddSiteBody

        if err := json.Unmarshal(command.Data, &body); err != nil {
            break
        }

        return body, nil
    case ClusterRemoveSite:
        var body ClusterRemoveSiteBody

        if err := json.Unmarshal(command.Data, &body); err != nil {
            break
        }

        return body, nil
    case ClusterAddRelay:
        var body ClusterAddRelayBody

        if err := json.Unmarshal(command.Data, &body); err != nil {
            break
        }

        return body, nil
    case ClusterRemoveRelay:
        var body ClusterRemoveRelayBody

        if err := json.Unmarshal(command.Data, &body); err != nil {
            break
        }

        return body, nil
    case ClusterMoveRelay:
        var body ClusterMoveRelayBody

        if err := json.Unmarshal(command.Data, &body); err != nil {
            break
        }

        return body, nil
    case ClusterSnapshot:
        var body ClusterSnapshotBody

        if err := json.Unmarshal(command.Data, &body); err != nil {
            break
        }

        return body, nil
    default:
        return nil, ENoSuchCommand
    }

    return nil, ECouldNotParseCommand
}

// assign tokens 
//    maybe the token assignment entry doesn't contain any tokens? maybe it just determines a deterministic order for assigning tokens?
// Scenarios
// Adding a node to a cluster
//     LOG: ... [ ADD NODE i ] ... [ UPDATE TOKEN ASSIGNMENTS FOR NODE i (capacity = 40 GiB) ] ...
// Decommission a node (graceful)
//     LOG: ... [ UPDATE TOKEN ASSIGNMENTS FOR NODE i (capacity = 0 GiB) ] ... [ REMOVE NODE i ] ...
// Removing a node (forced)
//     LOG: ... [ REMOVE NODE i ] ...
// Repairing a node in a cluster
//
// Updating a node's capacity
//     LOG: ... [ UPDATE TOKEN ASSIGNMENTS FOR NODE i (capacity = 80 GiB) ] ... increase
//     LOG: ... [ UPDATE TOKEN ASSIGNMENTS FOR NODE i (capacity = 20 GiB) ] ... decrease
