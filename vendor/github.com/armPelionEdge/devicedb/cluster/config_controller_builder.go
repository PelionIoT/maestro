package cluster

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