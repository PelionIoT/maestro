package partition

import (
    . "github.com/armPelionEdge/devicedb/site"
)

type PartitionFactory interface {
    CreatePartition(partitionNumber uint64, sitePool SitePool) Partition
}

type DefaultPartitionFactory struct {
}

func NewDefaultPartitionFactory() *DefaultPartitionFactory {
    return &DefaultPartitionFactory{ }
}

func (partitionFactory *DefaultPartitionFactory) CreatePartition(partitionNumber uint64, sitePool SitePool) Partition {
    return NewDefaultPartition(partitionNumber, sitePool)
}
