package partition

import (
    . "github.com/armPelionEdge/devicedb/site"
)

type Partition interface {
    Partition() uint64
    Sites() SitePool
    Iterator() PartitionIterator
    LockWrites()
    UnlockWrites()
    LockReads()
    UnlockReads()
}

type DefaultPartition struct {
    partition uint64
    sitePool SitePool
}

func NewDefaultPartition(partition uint64, sitePool SitePool) *DefaultPartition {
    return &DefaultPartition{
        partition: partition,
        sitePool: sitePool,
    }
}

func (partition *DefaultPartition) Partition() uint64 {
    return partition.partition
}

func (partition *DefaultPartition) Sites() SitePool {
    return partition.sitePool
}

func (partition *DefaultPartition) Iterator() PartitionIterator {
    return partition.sitePool.Iterator()
}

func (partition *DefaultPartition) LockWrites() {
}

func (partition *DefaultPartition) UnlockWrites() {
}

func (partition *DefaultPartition) LockReads() {
}

func (partition *DefaultPartition) UnlockReads() {
}