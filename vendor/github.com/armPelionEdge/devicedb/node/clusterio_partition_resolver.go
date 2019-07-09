package node

import (
    . "github.com/armPelionEdge/devicedb/cluster"
)

type PartitionResolver struct {
    configController ClusterConfigController
}

func NewPartitionResolver(configController ClusterConfigController) *PartitionResolver {
    return &PartitionResolver{
        configController: configController,
    }
}

func (partitionResolver *PartitionResolver) Partition(partitioningKey string) uint64 {
    return partitionResolver.configController.ClusterController().Partition(partitioningKey)
}

func (partitionResolver *PartitionResolver) ReplicaNodes(partition uint64) []uint64 {
    return partitionResolver.configController.ClusterController().PartitionOwners(partition)
}