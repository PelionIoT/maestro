package routes
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
    "time"

    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/transport"
)

const (
    SnapshotProcessing string = "processing"
    SnapshotFailed string = "failed"
    SnapshotComplete string = "completed"
    SnapshotMissing string = "missing"
)

type RelayStatus struct {
    Connected bool
    ConnectedTo uint64
    Ping time.Duration
    Site string
}

type ClusterOverview struct {
    Nodes []NodeConfig
    ClusterSettings ClusterSettings
    PartitionDistribution [][]uint64
    TokenAssignments []uint64
}

type InternalEntry struct {
    Prefix string
    Key string
    Siblings *SiblingSet
}

func (entry *InternalEntry) ToAPIEntry() *APIEntry {
    var transportSiblingSet TransportSiblingSet

    transportSiblingSet.FromSiblingSet(entry.Siblings)

    return &APIEntry{
        Prefix: entry.Prefix,
        Key: entry.Key,
        Context: transportSiblingSet.Context,
        Siblings: transportSiblingSet.Siblings,
    }
}

type APIEntry struct {
    Prefix string `json:"prefix"`
    Key string `json:"key"`
    Context string `json:"context"`
    Siblings []string `json:"siblings"`
}

type BatchResult struct {
    // Number of replicas that the batch was successfully applied to
    NApplied uint64 `json:"nApplied"`
    // Number of replicas in the replica set for this site
    Replicas uint64 `json:"replicas"`
    // Was write quorum achieved
    Quorum bool
    Patch map[string]*SiblingSet `json:"patch"`
}

type RelaySettingsPatch struct {
    Site string `json:"site"`
}

type LogSnapshot struct {
    Index uint64
    State ClusterState
}

type LogEntry struct {
    Index uint64
    Command ClusterCommand
}

type LogDump struct {
    BaseSnapshot LogSnapshot
    Entries []LogEntry
    CurrentSnapshot LogSnapshot
}

type Snapshot struct {
    UUID string `json:"uuid"`
    Status string `json:"status"`
}