package client

import (
    "github.com/armPelionEdge/devicedb/transport"
)

// Contains a database update operation
type Batch struct {
    ops map[string]transport.TransportUpdateOp
}

// Create a new batch update
func NewBatch() *Batch {
    return &Batch{
        ops: make(map[string]transport.TransportUpdateOp),
    }
}

// Adds a key put operation to this update. Key and value are the
// key that is being modified and the value that it should be set to.
// context is the causal context for the modification. It can be
// left blank if 
func (batch *Batch) Put(key string, value string, context string) *Batch {
    batch.ops[key] = transport.TransportUpdateOp{
        Type: "put",
        Key: key,
        Value: value,
        Context: context,
    }

    return batch
}

func (batch *Batch) Delete(key string, context string) *Batch {
    batch.ops[key] = transport.TransportUpdateOp{
        Type: "delete",
        Key: key,
        Context: context,
    }

    return batch
}

func (batch *Batch) ToTransportUpdateBatch() transport.TransportUpdateBatch {
    var updateBatch []transport.TransportUpdateOp = make([]transport.TransportUpdateOp, 0, len(batch.ops))

    for _, op := range batch.ops {
        updateBatch = append(updateBatch, op)
    }

    return transport.TransportUpdateBatch(updateBatch)
}