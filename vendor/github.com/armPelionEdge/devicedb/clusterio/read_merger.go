package clusterio

import (
    "sync"

    . "github.com/armPelionEdge/devicedb/data"
    "github.com/armPelionEdge/devicedb/resolver"
    "github.com/armPelionEdge/devicedb/resolver/strategies"
)

type ReadMerger struct {
    keyVersions map[string]map[uint64]*SiblingSet
    mergedKeys map[string]*SiblingSet
    conflictResolver resolver.ConflictResolver
    mu sync.Mutex
}

func NewReadMerger(bucket string) *ReadMerger {
    var conflictResolver resolver.ConflictResolver

    switch bucket {
    case "lww":
        conflictResolver = &strategies.LastWriterWins{}
    default:
        conflictResolver = &strategies.MultiValue{}
    }

    return &ReadMerger{
        keyVersions: make(map[string]map[uint64]*SiblingSet),
        mergedKeys: make(map[string]*SiblingSet),
        conflictResolver: conflictResolver,
    }
}

func (readMerger *ReadMerger) InsertKeyReplica(nodeID uint64, key string, siblingSet *SiblingSet) {
    readMerger.mu.Lock()
    defer readMerger.mu.Unlock()

    if _, ok := readMerger.keyVersions[key]; !ok {
        readMerger.keyVersions[key] = make(map[uint64]*SiblingSet)
    }

    if siblingSet == nil {
        siblingSet = NewSiblingSet(map[*Sibling]bool{ })
    }

    readMerger.keyVersions[key][nodeID] = siblingSet

    if readMerger.mergedKeys[key] == nil {
        readMerger.mergedKeys[key] = siblingSet
    } else {
        readMerger.mergedKeys[key] = readMerger.mergedKeys[key].Sync(siblingSet)
    }
}

func (readMerger *ReadMerger) Get(key string) *SiblingSet {
    readMerger.mu.Lock()
    defer readMerger.mu.Unlock()

    if readMerger.mergedKeys[key] == nil {
        return nil
    }

    if readMerger.mergedKeys[key].Size() == 0 {
        return nil
    }
    
    return readMerger.conflictResolver.ResolveConflicts(readMerger.mergedKeys[key])
}

func (readMerger *ReadMerger) Patch(nodeID uint64) map[string]*SiblingSet {
    readMerger.mu.Lock()
    defer readMerger.mu.Unlock()

    var patch map[string]*SiblingSet = make(map[string]*SiblingSet, len(readMerger.keyVersions))

    for key, versions := range readMerger.keyVersions {
        if version, ok := versions[nodeID]; ok {
            patch[key] = version.Diff(readMerger.mergedKeys[key])
        } else {
            patch[key] = readMerger.mergedKeys[key]
        }
    }

    return patch
}


func (readMerger *ReadMerger) Nodes() map[uint64]bool {
    readMerger.mu.Lock()
    defer readMerger.mu.Unlock()
    
    var nodes map[uint64]bool = make(map[uint64]bool)

    for _, keyHolders  := range readMerger.keyVersions {
        for nodeID, _ := range keyHolders {
            nodes[nodeID] = true
        }
    }

    return nodes
}