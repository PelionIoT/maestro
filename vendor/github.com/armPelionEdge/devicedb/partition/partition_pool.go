package partition

import (
    "sync"
)

type PartitionPool interface {
    Add(partition Partition)
    Remove(partitionNumber uint64)
    Get(partitionNumber uint64) Partition
}

type DefaultPartitionPool struct {
    lock sync.Mutex
    partitions map[uint64]Partition
}

func NewDefaultPartitionPool() *DefaultPartitionPool {
    return &DefaultPartitionPool{
        partitions: make(map[uint64]Partition, 0),
    }
}

func (partitionPool *DefaultPartitionPool) Add(partition Partition) {
    partitionPool.lock.Lock()
    defer partitionPool.lock.Unlock()

    partitionPool.partitions[partition.Partition()] = partition
}

func (partitionPool *DefaultPartitionPool) Remove(partitionNumber uint64) {
    partitionPool.lock.Lock()
    defer partitionPool.lock.Unlock()

    delete(partitionPool.partitions, partitionNumber)
}

func (partitionPool *DefaultPartitionPool) Get(partitionNumber uint64) Partition {
    partitionPool.lock.Lock()
    defer partitionPool.lock.Unlock()

    return partitionPool.partitions[partitionNumber]
}