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
    . "github.com/armPelionEdge/devicedb/raft"
)

type ClusterConfigControllerBuilder interface {
    SetCreateNewCluster(b bool) ClusterConfigControllerBuilder
    SetLocalNodeAddress(peerAddress PeerAddress) ClusterConfigControllerBuilder
    SetRaftNodeStorage(raftStorage RaftNodeStorage) ClusterConfigControllerBuilder
    SetRaftNodeTransport(transport *TransportHub) ClusterConfigControllerBuilder
    Create() ClusterConfigController
}

type ConfigControllerBuilder struct {
    createNewCluster bool
    localNodeAddress PeerAddress
    raftStorage RaftNodeStorage
    raftTransport *TransportHub
}

func (builder *ConfigControllerBuilder) SetCreateNewCluster(b bool) ClusterConfigControllerBuilder {
    builder.createNewCluster = b

    return builder
}

func (builder *ConfigControllerBuilder) SetLocalNodeAddress(address PeerAddress) ClusterConfigControllerBuilder {
    builder.localNodeAddress = address

    return builder
}

func (builder *ConfigControllerBuilder) SetRaftNodeStorage(raftStorage RaftNodeStorage) ClusterConfigControllerBuilder {
    builder.raftStorage = raftStorage

    return builder
}

func (builder *ConfigControllerBuilder) SetRaftNodeTransport(transport *TransportHub) ClusterConfigControllerBuilder {
    builder.raftTransport = transport

    return builder
}

func (builder *ConfigControllerBuilder) Create() ClusterConfigController {
    addNodeBody, _ := EncodeClusterCommandBody(ClusterAddNodeBody{
        NodeID: builder.localNodeAddress.NodeID,
        NodeConfig: NodeConfig{
            Address: PeerAddress{
                NodeID: builder.localNodeAddress.NodeID,
                Host: builder.localNodeAddress.Host,
                Port: builder.localNodeAddress.Port,
            },
            Capacity: 1,
        },
    })
    addNodeContext, _ := EncodeClusterCommand(ClusterCommand{ Type: ClusterAddNode, Data: addNodeBody })
    clusterController := &ClusterController{
        LocalNodeID: builder.localNodeAddress.NodeID,
        State: ClusterState{ },
        PartitioningStrategy: &SimplePartitioningStrategy{ },
        LocalUpdates: make(chan []ClusterStateDelta),
    }
    raftNode := NewRaftNode(&RaftNodeConfig{
        ID: builder.localNodeAddress.NodeID,
        CreateClusterIfNotExist: builder.createNewCluster,
        Context: addNodeContext,
        Storage: builder.raftStorage,
        GetSnapshot: func() ([]byte, error) {
            return clusterController.State.Snapshot()
        },
    })

    return NewConfigController(raftNode, builder.raftTransport, clusterController)
}