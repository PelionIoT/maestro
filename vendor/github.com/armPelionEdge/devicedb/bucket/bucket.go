package bucket
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
    "context"
    "errors"

    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/merkle"
)

var ENoSuchBucket = errors.New("No such bucket")

type Bucket interface {
    Name() string
    ShouldReplicateOutgoing(peerID string) bool
    ShouldReplicateIncoming(peerID string) bool
    ShouldAcceptWrites(clientID string) bool
    ShouldAcceptReads(clientID string) bool
    RecordMetadata() error
    RebuildMerkleLeafs() error
    MerkleTree() *MerkleTree
    GarbageCollect(tombstonePurgeAge uint64) error
    Get(keys [][]byte) ([]*SiblingSet, error)
    GetMatches(keys [][]byte) (SiblingSetIterator, error)
    GetSyncChildren(nodeID uint32) (SiblingSetIterator, error)
    GetAll() (SiblingSetIterator, error)
    Forget(keys [][]byte) error
    Batch(batch *UpdateBatch) (map[string]*SiblingSet, error)
    Merge(siblingSets map[string]*SiblingSet) error
    Watch(ctx context.Context, keys [][]byte, prefixes [][]byte, localVersion uint64, ch chan Row)
    LockWrites()
    UnlockWrites()
    LockReads()
    UnlockReads()
}
