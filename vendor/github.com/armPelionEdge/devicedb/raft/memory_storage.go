package raft
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
    "github.com/coreos/etcd/raft"
    "github.com/coreos/etcd/raft/raftpb"
)

type RaftMemoryStorage struct {
    memoryStorage *raft.MemoryStorage
    isEmpty bool
    isDecomissioning bool
    nodeID uint64
}

func NewRaftMemoryStorage() *RaftMemoryStorage {
    return &RaftMemoryStorage{
        isEmpty: true,
        memoryStorage: raft.NewMemoryStorage(),
    }
}

func (raftStorage *RaftMemoryStorage) InitialState() (raftpb.HardState, raftpb.ConfState, error) {
    return raftStorage.memoryStorage.InitialState()
}

func (raftStorage *RaftMemoryStorage) Entries(lo, hi, maxSize uint64) ([]raftpb.Entry, error) {
    return raftStorage.memoryStorage.Entries(lo, hi, maxSize)
}

func (raftStorage *RaftMemoryStorage) Term(i uint64) (uint64, error) {
    return raftStorage.memoryStorage.Term(i)
}

func (raftStorage *RaftMemoryStorage) LastIndex() (uint64, error) {
    return raftStorage.memoryStorage.LastIndex()
}

func (raftStorage *RaftMemoryStorage) FirstIndex() (uint64, error) {
    return raftStorage.memoryStorage.FirstIndex()
}

func (raftStorage *RaftMemoryStorage) Snapshot() (raftpb.Snapshot, error) {
    return raftStorage.memoryStorage.Snapshot()
}

func (raftStorage *RaftMemoryStorage) Open() error {
    return nil
}

func (raftStorage *RaftMemoryStorage) Close() error {
    return nil
}

func (raftStorage *RaftMemoryStorage) IsEmpty() bool {
    return raftStorage.isEmpty
}

func (raftStorage *RaftMemoryStorage) SetIsEmpty(b bool) {
    raftStorage.isEmpty = b
}

func (raftStorage *RaftMemoryStorage) SetDecommissioningFlag() error {
    raftStorage.isDecomissioning = true

    return nil
}

func (raftStorage *RaftMemoryStorage) IsDecommissioning() (bool, error) {
    return raftStorage.isDecomissioning, nil
}

func (raftStorage *RaftMemoryStorage) SetNodeID(id uint64) error {
    raftStorage.nodeID = id

    return nil
}

func (raftStorage *RaftMemoryStorage) NodeID() (uint64, error) {
    return raftStorage.nodeID, nil
}

func (raftStorage *RaftMemoryStorage) Append(entries []raftpb.Entry) error {
    return raftStorage.memoryStorage.Append(entries)
}

func (raftStorage *RaftMemoryStorage) SetHardState(st raftpb.HardState) error {
    return raftStorage.memoryStorage.SetHardState(st)
}

func (raftStorage *RaftMemoryStorage) ApplySnapshot(snap raftpb.Snapshot) error {
    return raftStorage.memoryStorage.ApplySnapshot(snap)
}

func (raftStorage *RaftMemoryStorage) CreateSnapshot(i uint64, cs *raftpb.ConfState, data []byte) (raftpb.Snapshot, error) {
    return raftStorage.memoryStorage.CreateSnapshot(i, cs, data)
}

func (raftStorage *RaftMemoryStorage) ApplyAll(hs raftpb.HardState, ents []raftpb.Entry, snap raftpb.Snapshot) error {
    raftStorage.Append(ents)

    // update hard state if set
    if !raft.IsEmptyHardState(hs) {
        raftStorage.SetHardState(hs)
    }

    // apply snapshot
    if !raft.IsEmptySnap(snap) {
        raftStorage.ApplySnapshot(snap)
    }

    return nil
}