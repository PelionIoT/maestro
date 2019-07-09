package raft

import (
    "math"
    "time"
    "errors"
    "context"

    . "github.com/armPelionEdge/devicedb/logging"
    
    "github.com/coreos/etcd/raft"
    "github.com/coreos/etcd/raft/raftpb"
    //"golang.org/x/net/context"
)

var ECancelConfChange = errors.New("Conf change cancelled")

// Limit on the number of entries that can accumulate before a snapshot and compaction occurs
var LogCompactionSize int = 1000

type RaftNodeConfig struct {
    ID uint64
    CreateClusterIfNotExist bool
    Storage RaftNodeStorage
    GetSnapshot func() ([]byte, error)
    Context []byte
}

type RaftNode struct {
    config *RaftNodeConfig
    node raft.Node
    isRunning bool
    stop chan int
    lastReplayIndex uint64
    lastCommittedIndex uint64
    onMessagesCB func([]raftpb.Message) error
    onSnapshotCB func(raftpb.Snapshot) error
    onEntryCB func(raftpb.Entry) error
    onErrorCB func(error) error
    onReplayDoneCB func() error
    currentRaftConfState raftpb.ConfState
}

func NewRaftNode(config *RaftNodeConfig) *RaftNode {
    raftNode := &RaftNode{
        config: config,
        node: nil,
    }

    return raftNode
}

func (raftNode *RaftNode) ID() uint64 {
    return raftNode.config.ID
}

func (raftNode *RaftNode) AddNode(ctx context.Context, nodeID uint64, context []byte) error {
    Log.Infof("Node %d proposing addition of node %d to its cluster", raftNode.config.ID, nodeID)

    err := raftNode.node.ProposeConfChange(ctx, raftpb.ConfChange{
        ID: nodeID,
        Type: raftpb.ConfChangeAddNode,
        NodeID: nodeID,
        Context: context,
    })

    if err != nil {
        Log.Errorf("Node %d was unable to propose addition of node %d to its cluster: %s", raftNode.config.ID, nodeID, err.Error())
    }

    return err
}

func (raftNode *RaftNode) RemoveNode(ctx context.Context, nodeID uint64, context []byte) error {
    Log.Infof("Node %d proposing removal of node %d from its cluster", raftNode.config.ID, nodeID)

    err := raftNode.node.ProposeConfChange(ctx, raftpb.ConfChange{
        ID: nodeID,
        Type: raftpb.ConfChangeRemoveNode,
        NodeID: nodeID,
        Context: context,
    })

    if err != nil {
        Log.Errorf("Node %d was unable to propose removal of node %d from its cluster: %s", raftNode.config.ID, nodeID, err.Error())
    }

    return err
}

func (raftNode *RaftNode) Propose(ctx context.Context, proposition []byte) error {
    return raftNode.node.Propose(ctx, proposition)
}

func (raftNode *RaftNode) LastSnapshot() (raftpb.Snapshot, error) {
    return raftNode.config.Storage.Snapshot()
}

func (raftNode *RaftNode) CommittedEntries() ([]raftpb.Entry, error) {
    firstIndex, err := raftNode.config.Storage.FirstIndex()

    if err != nil {
        return nil, err
    }

    if firstIndex == raftNode.lastCommittedIndex + 1 {
        return []raftpb.Entry{}, nil
    }

    return raftNode.config.Storage.Entries(firstIndex, raftNode.lastCommittedIndex + 1, math.MaxUint64)
}

func (raftNode *RaftNode) Start() error {
    raftNode.stop = make(chan int)
    raftNode.isRunning = true

    if err := raftNode.config.Storage.Open(); err != nil {
        return err
    }

    nodeContext := raftNode.config.Context

    if nodeContext == nil {
        nodeContext = []byte{ }
    }

    config := &raft.Config{
        ID: raftNode.config.ID,
        ElectionTick: 10,
        HeartbeatTick: 1,
        Storage: raftNode.config.Storage,
        MaxSizePerMsg: math.MaxUint16,
        MaxInflightMsgs: 256,
    }

    if !raftNode.config.Storage.IsEmpty() {
        // indicates that this node has already been run before
        raftNode.node = raft.RestartNode(config)
    } else {
        // by default create a new cluster with one member (this node)
        peers := []raft.Peer{ raft.Peer{ ID: raftNode.config.ID, Context: raftNode.config.Context } }

        if !raftNode.config.CreateClusterIfNotExist {
            // indicates that this node should join an existing cluster
            peers = nil
        } else {
            Log.Infof("Creating a new single node cluster")
        }

        raftNode.node = raft.StartNode(config, peers)
    }

    hardState, _, _ := raftNode.config.Storage.InitialState()

    // this keeps track of the highest index that is knowns
    // to have been committed to the cluster
    if !raft.IsEmptyHardState(hardState) {
        raftNode.lastReplayIndex = hardState.Commit
    }

    Log.Debugf("Starting up raft node. Last known commit index was %v", raftNode.lastReplayIndex)

    go raftNode.run()

    return nil
}

func (raftNode *RaftNode) Receive(ctx context.Context, msg raftpb.Message) error {
    return raftNode.node.Step(ctx, msg)
}

func (raftNode *RaftNode) notifyIfReplayDone() {
    if raftNode.lastCommittedIndex == raftNode.lastReplayIndex {
        raftNode.onReplayDoneCB()
    }
}

func (raftNode *RaftNode) run() {
    ticker := time.Tick(time.Second)

    defer func() {
        // makes sure cleanup happens when the loop exits
        raftNode.lastCommittedIndex = 0
        raftNode.lastReplayIndex = 0
        raftNode.node.Stop()
        raftNode.config.Storage.Close()
        raftNode.currentRaftConfState = raftpb.ConfState{ }
    }()

    lastSnapshot, _ := raftNode.LastSnapshot()

    if !raft.IsEmptySnap(lastSnapshot) {
        // It is essential that this happen so that when a new snapshot occurs
        // it does not write an empty conf with no nodes in it. This will corrupt
        // the cluster state. This caused an error where a node forgot that it belonged
        // to its own cluster and couldn't become a leader or start a campaign.
        raftNode.currentRaftConfState = lastSnapshot.Metadata.ConfState
        // call onSnapshot callback to give initial state to system config
        raftNode.onSnapshotCB(lastSnapshot)

        if lastSnapshot.Metadata.Index == raftNode.lastReplayIndex {
            raftNode.lastCommittedIndex = raftNode.lastReplayIndex
        }
    }

    raftNode.notifyIfReplayDone()

    for {
        select {
        case <-ticker:
            raftNode.node.Tick()
        case rd := <-raftNode.node.Ready():
            // Saves raft state to persistent storage first. If the process dies or fails after this point
            // This ensures that there is a checkpoint to resume from on restart
            if err := raftNode.saveToStorage(rd.HardState, rd.Entries, rd.Snapshot); err != nil {
                raftNode.onErrorCB(err)

                return
            }

            // Messages must be sent after entries and hard state are saved to stable storage
            if len(rd.Messages) != 0 {
                raftNode.onMessagesCB(rd.Messages)
            }

            if !raft.IsEmptySnap(rd.Snapshot) {
                // snapshots received from other nodes. 
                // Used to allow this node to catch up
                raftNode.onSnapshotCB(rd.Snapshot)
                raftNode.lastCommittedIndex = rd.Snapshot.Metadata.Index
                raftNode.notifyIfReplayDone()
            }

            for _, entry := range rd.CommittedEntries {
                // Skip an entry if it has already been applied.
                // I don't know why I would receive a committed entry here
                // twice but the raft example does this so I'm doing it.
                if entry.Index < raftNode.lastCommittedIndex {
                    continue
                }

                cancelConfChange := raftNode.onEntryCB(entry) == ECancelConfChange

                raftNode.lastCommittedIndex = entry.Index

                if entry.Type == raftpb.EntryConfChange {
                    if err := raftNode.applyConfigurationChange(entry, cancelConfChange); err != nil {
                        raftNode.onErrorCB(err)

                        return
                    }
                }
                
                raftNode.notifyIfReplayDone()
            }

            // Snapshot current state and perform a compaction of entries
            // if the number of entries exceeds a certain theshold
            if err := raftNode.takeSnapshotIfEnoughEntries(); err != nil {
                raftNode.onErrorCB(err)
                
                return
            }

            raftNode.node.Advance()
        case <-raftNode.stop:
            return
        }
    }
}

func (raftNode *RaftNode) saveToStorage(hs raftpb.HardState, ents []raftpb.Entry, snap raftpb.Snapshot) error {
    // Ensures that all updates get applied atomically to persistent storage: HardState, Entries, Snapshot.
    // If any part of the update fails then no change is applied. This is important so that the persistent state
    // remains consistent.
    if err := raftNode.config.Storage.ApplyAll(hs, ents, snap); err != nil {
        return err
    }

    return nil
}

func (raftNode *RaftNode) applyConfigurationChange(entry raftpb.Entry, cancelConfChange bool) error {
    var confChange raftpb.ConfChange

    if err := confChange.Unmarshal(entry.Data); err != nil {
        return err
    }

    if cancelConfChange {
        // From the etcd/raft docs: "The configuration change may be cancelled at this point by setting the NodeID field to zero before calling ApplyConfChange"
        switch confChange.Type {
        case raftpb.ConfChangeAddNode:
            Log.Debugf("Ignoring proposed addition of node %d", confChange.NodeID)
        case raftpb.ConfChangeRemoveNode:
            Log.Debugf("Ignoring proposed removal of node %d", confChange.NodeID)
        }

        confChange.NodeID = 0
    }

    raftNode.currentRaftConfState = *raftNode.node.ApplyConfChange(confChange)

    return nil
}

func (raftNode *RaftNode) takeSnapshotIfEnoughEntries() error {
    lastSnapshot, err := raftNode.config.Storage.Snapshot()

    if err != nil {
        return err
    }

    if raftNode.lastCommittedIndex < lastSnapshot.Metadata.Index {
        return nil
    }

    if raftNode.lastCommittedIndex - lastSnapshot.Metadata.Index >= uint64(LogCompactionSize) {
        // data is my config state snapshot
        data, err := raftNode.config.GetSnapshot()

        if err != nil {
            return err
        }

        _, err = raftNode.config.Storage.CreateSnapshot(raftNode.lastCommittedIndex, &raftNode.currentRaftConfState, data)

        if err != nil {
            return err
        }
    }

    return nil
}

func (raftNode *RaftNode) Stop() {
    if raftNode.isRunning {
        raftNode.isRunning = false
        close(raftNode.stop)
    }
}

func (raftNode *RaftNode) OnReplayDone(cb func() error) {
    raftNode.onReplayDoneCB = cb
}

func (raftNode *RaftNode) OnMessages(cb func([]raftpb.Message) error) {
    raftNode.onMessagesCB = cb
}

func (raftNode *RaftNode) OnSnapshot(cb func(raftpb.Snapshot) error) {
    raftNode.onSnapshotCB = cb
}

func (raftNode *RaftNode) OnCommittedEntry(cb func(raftpb.Entry) error) {
    raftNode.onEntryCB = cb
}

func (raftNode *RaftNode) OnError(cb func(error) error) {
    raftNode.onErrorCB = cb
}

func (raftNode *RaftNode) ReportUnreachable(id uint64) {
    raftNode.node.ReportUnreachable(id)
}

func (raftNode *RaftNode) ReportSnapshot(id uint64, status raft.SnapshotStatus) {
    raftNode.node.ReportSnapshot(id, status)
}