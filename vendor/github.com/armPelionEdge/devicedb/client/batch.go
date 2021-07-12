package client
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