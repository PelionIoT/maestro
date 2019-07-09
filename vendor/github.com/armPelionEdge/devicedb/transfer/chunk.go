package transfer

import (
    . "github.com/armPelionEdge/devicedb/data"
)

const DefaultChunkSize = 100

type Entry struct {
    Site string
    Bucket string
    Key string
    Value *SiblingSet
}

type PartitionChunk struct {
    Index uint64
    Entries []Entry
    Checksum Hash
}

func (partitionChunk *PartitionChunk) IsEmpty() bool {
    return len(partitionChunk.Entries) == 0
}