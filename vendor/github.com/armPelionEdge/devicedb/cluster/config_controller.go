// This module bridges the gap between the cluster configuration controller
// and the raft library
package cluster

import (
    "github.com/armPelionEdge/devicedb/raft"
    . "github.com/armPelionEdge/devicedb/logging"
    . "github.com/armPelionEdge/devicedb/util"

    raftEtc "github.com/coreos/etcd/raft"
    "github.com/coreos/etcd/raft/raftpb"

    "context"
    "errors"
    "sync"
    "time"
)

const ProposalRetryPeriodSeconds = 15

type ClusterConfigController interface {
    LogDump() (raftpb.Snapshot, []raftpb.Entry, error)
    AddNode(ctx context.Context, nodeConfig NodeConfig) error
    ReplaceNode(ctx context.Context, replacedNodeID uint64, replacementNodeID uint64) error
    RemoveNode(ctx context.Context, nodeID uint64) error
    ClusterCommand(ctx context.Context, commandBody interface{}) error
    OnLocalUpdates(cb func(deltas []ClusterStateDelta))
    OnClusterSnapshot(cb func(snapshotIndex uint64, snapshotId string))
    ClusterController() *ClusterController
    Start() error
    Stop()
    CancelProposals()
}

var EBadContext = errors.New("The node addition or removal had an invalid context")
var ERaftNodeStartup = errors.New("Encountered an error while starting up raft controller")
var ERaftProtocolError = errors.New("Raft controller encountered a protocol error")
var ECancelled = errors.New("The request was cancelled")
var EStopped = errors.New("The server was stopped")

type proposalResponse struct {
    err error
}

type ConfigController struct {
    raftNode *raft.RaftNode
    raftTransport *raft.TransportHub
    clusterController *ClusterController
    requestMap *RequestMap
    stop chan int
    restartLock sync.Mutex
    lock sync.Mutex
    pendingProposals map[uint64]func()
    proposalsCancelled bool
    onLocalUpdatesCB func([]ClusterStateDelta)
    onClusterSnapshotCB func(uint64, string)
    // entryLog serves as an
    // easily accsessible record of what happened
    // at this node to bring its state to what it
    // is now. These fields are used by the log_dump
    // and allow a developer to debug cluster state
    // inconsistencies.
    entryLog []raftpb.Entry
}

func NewConfigController(raftNode *raft.RaftNode, raftTransport *raft.TransportHub, clusterController *ClusterController) *ConfigController {
    return &ConfigController{
        raftNode: raftNode,
        raftTransport: raftTransport,
        clusterController: clusterController,
        requestMap: NewRequestMap(),
        pendingProposals: make(map[uint64]func()),
        entryLog: make([]raftpb.Entry, 0),
    }
}

func (cc *ConfigController) LogDump() (raftpb.Snapshot, []raftpb.Entry, error) {
    var baseSnapshot raftpb.Snapshot

    baseSnapshot, err := cc.raftNode.LastSnapshot()

    if err != nil {
        return raftpb.Snapshot{}, []raftpb.Entry{ }, err
    }

    entries, err := cc.raftNode.CommittedEntries()

    if err != nil {
        return raftpb.Snapshot{}, []raftpb.Entry{ }, err
    }

    return baseSnapshot, entries, nil
}

func (cc *ConfigController) CancelProposals() {
    // Cancels all current proposals and prevents any future ones from being made, causing them all to return ECancelled
    cc.lock.Lock()
    defer cc.lock.Unlock()

    cc.proposalsCancelled = true

    for _, cancel := range cc.pendingProposals {
        cancel()
    }
}

func (cc *ConfigController) unregisterProposal(id uint64) {
    cc.lock.Lock()
    defer cc.lock.Unlock()

    delete(cc.pendingProposals, id)
}

func (cc *ConfigController) AddNode(ctx context.Context, nodeConfig NodeConfig) error {
    encodedAddCommandBody, _ := EncodeClusterCommandBody(ClusterAddNodeBody{ NodeID: nodeConfig.Address.NodeID, NodeConfig: nodeConfig })
    addCommand := ClusterCommand{ Type: ClusterAddNode, Data: encodedAddCommandBody, SubmitterID: cc.clusterController.LocalNodeID, CommandID: cc.nextCommandID() }
    addContext, _ := EncodeClusterCommand(addCommand)

    cc.lock.Lock()
    if cc.proposalsCancelled {
        cc.lock.Unlock()
        return ECancelled
    }

    ctx, cancel := context.WithCancel(ctx)
    cc.pendingProposals[addCommand.CommandID] = cancel
    defer cc.unregisterProposal(addCommand.CommandID)
    cc.lock.Unlock()

    respCh := cc.requestMap.MakeRequest(addCommand.CommandID)

    if err := cc.raftNode.AddNode(ctx, nodeConfig.Address.NodeID, addContext); err != nil {
        cc.requestMap.Respond(addCommand.CommandID, nil)
        return err
    }

    select {
    case resp := <-respCh:
        return resp.(proposalResponse).err
    case <-ctx.Done():
        cc.requestMap.Respond(addCommand.CommandID, nil)
        return ECancelled
    case <-cc.stop:
        cc.requestMap.Respond(addCommand.CommandID, nil)
        return EStopped
    }
}

func (cc *ConfigController) ReplaceNode(ctx context.Context, replacedNodeID uint64, replacementNodeID uint64) error {
    encodedRemoveCommandBody, _ := EncodeClusterCommandBody(ClusterRemoveNodeBody{ NodeID: replacedNodeID, ReplacementNodeID: replacementNodeID })
    replaceCommand := ClusterCommand{ Type: ClusterRemoveNode, Data: encodedRemoveCommandBody, SubmitterID: cc.clusterController.LocalNodeID, CommandID: cc.nextCommandID() }
    replaceContext, _ := EncodeClusterCommand(replaceCommand)

    cc.lock.Lock()
    if cc.proposalsCancelled {
        cc.lock.Unlock()
        return ECancelled
    }

    ctx, cancel := context.WithCancel(ctx)
    cc.pendingProposals[replaceCommand.CommandID] = cancel
    defer cc.unregisterProposal(replaceCommand.CommandID)
    cc.lock.Unlock()

    respCh := cc.requestMap.MakeRequest(replaceCommand.CommandID)

    if err := cc.raftNode.RemoveNode(ctx, replacedNodeID, replaceContext); err != nil {
        cc.requestMap.Respond(replaceCommand.CommandID, nil)
        return err
    }

    for {
        select {
        case resp := <-respCh:
            return resp.(proposalResponse).err
        case <-time.After(time.Second * ProposalRetryPeriodSeconds):
        case <-ctx.Done():
            cc.requestMap.Respond(replaceCommand.CommandID, nil)
            return ECancelled
        case <-cc.stop:
            cc.requestMap.Respond(replaceCommand.CommandID, nil)
            return EStopped
        }
    }
}

func (cc *ConfigController) RemoveNode(ctx context.Context, nodeID uint64) error {
    encodedRemoveCommandBody, _ := EncodeClusterCommandBody(ClusterRemoveNodeBody{ NodeID: nodeID, ReplacementNodeID: 0 })
    removeCommand := ClusterCommand{ Type: ClusterRemoveNode, Data: encodedRemoveCommandBody, SubmitterID: cc.clusterController.LocalNodeID, CommandID: cc.nextCommandID() }
    removeContext, _ := EncodeClusterCommand(removeCommand)

    cc.lock.Lock()
    if cc.proposalsCancelled {
        cc.lock.Unlock()
        return ECancelled
    }

    ctx, cancel := context.WithCancel(ctx)
    cc.pendingProposals[removeCommand.CommandID] = cancel
    defer cc.unregisterProposal(removeCommand.CommandID)
    cc.lock.Unlock()

    respCh := cc.requestMap.MakeRequest(removeCommand.CommandID)

    if err := cc.raftNode.RemoveNode(ctx, nodeID, removeContext); err != nil {
        cc.requestMap.Respond(removeCommand.CommandID, nil)
        return err
    }

    for {
        select {
        case resp := <-respCh:
            return resp.(proposalResponse).err
        case <-time.After(time.Second * ProposalRetryPeriodSeconds):
        case <-ctx.Done():
            cc.requestMap.Respond(removeCommand.CommandID, nil)
            return ECancelled
        case <-cc.stop:
            cc.requestMap.Respond(removeCommand.CommandID, nil)
            return EStopped
        }
    }
}

func (cc *ConfigController) ClusterCommand(ctx context.Context, commandBody interface{}) error {
    var command ClusterCommand = ClusterCommand{
        SubmitterID: cc.clusterController.LocalNodeID,
    }

    switch commandBody.(type) {
    case ClusterAddNodeBody:
        command.Type = ClusterAddNode
    case ClusterRemoveNodeBody:
        command.Type = ClusterRemoveNode
    case ClusterUpdateNodeBody:
        command.Type = ClusterUpdateNode
    case ClusterTakePartitionReplicaBody:
        command.Type = ClusterTakePartitionReplica
    case ClusterSetReplicationFactorBody:
        command.Type = ClusterSetReplicationFactor
    case ClusterSetPartitionCountBody:
        command.Type = ClusterSetPartitionCount
    case ClusterAddSiteBody:
        command.Type = ClusterAddSite
    case ClusterRemoveSiteBody:
        command.Type = ClusterRemoveSite
    case ClusterAddRelayBody:
        command.Type = ClusterAddRelay
    case ClusterRemoveRelayBody:
        command.Type = ClusterRemoveRelay
    case ClusterMoveRelayBody:
        command.Type = ClusterMoveRelay
    case ClusterSnapshotBody:
        command.Type = ClusterSnapshot
    default:
        return ENoSuchCommand
    }

    encodedCommandBody, _ := EncodeClusterCommandBody(commandBody)
    command.Data = encodedCommandBody
    command.SubmitterID = cc.clusterController.LocalNodeID
    command.CommandID = cc.nextCommandID()
    encodedCommand, _ := EncodeClusterCommand(command)

    cc.lock.Lock()
    if cc.proposalsCancelled {
        cc.lock.Unlock()
        return ECancelled
    }

    ctx, cancel := context.WithCancel(ctx)
    cc.pendingProposals[command.CommandID] = cancel
    defer cc.unregisterProposal(command.CommandID)
    cc.lock.Unlock()

    respCh := cc.requestMap.MakeRequest(command.CommandID)

    // Note: It is possible that the proposal is lost due to message loss. Proposals are not
    // re-sent if being forwarded from follower to leader. It is possible for the call to
    // Propose() to accept a proposal and queue it for forwarding to the leader but there is no mechanism
    // to re-send the proposal message if it is lost due to network congestion or if 
    // the leader is down. Since the intention of this method is to block until the given command
    // has been committed to the log, it needs to retry proposals after a timeout period.
    // 
    // This issue was discovered when the downloader attempted to submit a cluster command
    // that would transfer holdership of a partition replica to the requesting node but the TCP
    // connection reached the "file open" limit and some messages were dropped. A way to mitigate
    // this as well is to send the message serially instead of at the same time with multiple goroutines
    for {
        Log.Debugf("Making raft proposal for command %d", command.CommandID)

        if err := cc.raftNode.Propose(ctx, encodedCommand); err != nil {
            cc.requestMap.Respond(command.CommandID, nil)
            return err
        }

        select {
        case resp := <-respCh:
            Log.Debugf("Command %d accepted", command.CommandID)
            return resp.(proposalResponse).err
        case <-time.After(time.Second * ProposalRetryPeriodSeconds):
            // Time to retry the proposal
            Log.Debugf("Re-attempting proposal for command %d", command.CommandID)
        case <-ctx.Done():
            cc.requestMap.Respond(command.CommandID, nil)
            return ECancelled
        case <-cc.stop:
            cc.requestMap.Respond(command.CommandID, nil)
            return EStopped
        }
    }
}

func (cc *ConfigController) OnLocalUpdates(cb func(deltas []ClusterStateDelta)) {
    cc.onLocalUpdatesCB = cb
}

func (cc *ConfigController) OnClusterSnapshot(cb func(snapshotIndex uint64, snapshotId string)) {
    cc.onClusterSnapshotCB = cb
}

func (cc *ConfigController) Start() error {
    cc.restartLock.Lock()
    restored := make(chan int, 1)
    replayDone := false
    cc.stop = make(chan int)
    cc.restartLock.Unlock()
    cc.clusterController.DisableNotifications()

    cc.raftTransport.OnReceive(func(ctx context.Context, msg raftpb.Message) error {
        return cc.raftNode.Receive(ctx, msg)
    })

    cc.raftNode.OnMessages(func(messages []raftpb.Message) error {
        // This used to send messages in parallel using one goroutine per
        // message but this overwhelms the TCP connection and results
        // in more lost messages. Should send serially
        for _, msg := range messages {
            err := cc.raftTransport.Send(context.TODO(), msg, false)

            if err != nil {
                cc.raftNode.ReportUnreachable(msg.To)

                if msg.Type == raftpb.MsgSnap {
                    cc.raftNode.ReportSnapshot(msg.To, raftEtc.SnapshotFailure)
                }
            } else if msg.Type == raftpb.MsgSnap {
                cc.raftNode.ReportSnapshot(msg.To, raftEtc.SnapshotFinish)
            }
        }

        return nil
    })

    cc.raftNode.OnSnapshot(func(snap raftpb.Snapshot) error {
        err := cc.clusterController.ApplySnapshot(snap.Data)

        if err != nil {
            return err
        }

        for _, nodeConfig := range cc.clusterController.State.Nodes {
            cc.raftTransport.AddPeer(nodeConfig.Address)
        }

        if replayDone {
            if cc.onLocalUpdatesCB != nil && len(cc.clusterController.Deltas()) > 0 {
                cc.onLocalUpdatesCB(cc.clusterController.Deltas())
            }
        }

        return nil
    })

    cc.raftNode.OnCommittedEntry(func(entry raftpb.Entry) error {
        Log.Debugf("New entry at node %d [%d]: %v", cc.clusterController.LocalNodeID, entry.Index, entry)

        var encodedClusterCommand []byte
        var clusterCommand ClusterCommand
        var clusterCommandBody interface{}

        switch entry.Type {
        case raftpb.EntryConfChange:
            var confChange raftpb.ConfChange
            var err error

            if err := confChange.Unmarshal(entry.Data); err != nil {
                return err
            }

            clusterCommand, err = DecodeClusterCommand(confChange.Context)

            if err != nil {
                return err
            }

            clusterCommandBody, err = DecodeClusterCommandBody(clusterCommand)

            if err != nil {
                return err
            }

            switch clusterCommand.Type {
            case ClusterAddNode:
                if clusterCommandBody.(ClusterAddNodeBody).NodeID != clusterCommandBody.(ClusterAddNodeBody).NodeConfig.Address.NodeID {
                    return EBadContext
                }
            case ClusterRemoveNode:
            default:
                return EBadContext
            }

            encodedClusterCommand = confChange.Context
        case raftpb.EntryNormal:
            var err error
            encodedClusterCommand = entry.Data
            clusterCommand, err = DecodeClusterCommand(encodedClusterCommand)

            if err != nil {
                return err
            }
        }
        
        localUpdates, err := cc.clusterController.Step(clusterCommand)

        if err != nil {
            if clusterCommand.SubmitterID == cc.clusterController.LocalNodeID {
                cc.requestMap.Respond(clusterCommand.CommandID, proposalResponse{
                    err: err,
                })
            }

            return err
        }

        // Only update transport if the cluster config was updated
        if entry.Type == raftpb.EntryConfChange {
            switch clusterCommand.Type {
            case ClusterAddNode:
                cc.raftTransport.AddPeer(clusterCommandBody.(ClusterAddNodeBody).NodeConfig.Address)
            case ClusterRemoveNode:
                cc.raftTransport.RemovePeer(raft.PeerAddress{ NodeID: clusterCommandBody.(ClusterRemoveNodeBody).NodeID })
            }
        }

        if clusterCommand.Type == ClusterSnapshot {
            body, _ := DecodeClusterCommandBody(clusterCommand)
            snapshotMeta := body.(ClusterSnapshotBody)

            if replayDone {            
                if cc.onClusterSnapshotCB != nil {
                    if cc.clusterController.LocalNodeIsInCluster() {
                        cc.onClusterSnapshotCB(entry.Index, snapshotMeta.UUID)
                    }
                }
            }
        }

        if clusterCommand.SubmitterID == cc.clusterController.LocalNodeID {
            cc.requestMap.Respond(clusterCommand.CommandID, proposalResponse{
                err: nil,
            })
        }

        if replayDone {
            if cc.onLocalUpdatesCB != nil && len(localUpdates) > 0 {
                cc.onLocalUpdatesCB(localUpdates)
            }
        }

        return nil
    })

    cc.raftNode.OnError(func(err error) error {
        // indicates that raft node is shutting down
        Log.Criticalf("Raft node encountered an unrecoverable error and will now shut down: %v", err)

        return nil
    })

    cc.raftNode.OnReplayDone(func() error {
        Log.Debug("OnReplayDone() called")
        // cc.clusterController.EnableNotifications()
        replayDone = true
        restored <- 1

        return nil
    })

    if err := cc.raftNode.Start(); err != nil {
        Log.Criticalf("Unable to start the config controller due to an error while starting up raft node: %v", err.Error())

        return ERaftNodeStartup
    }

    Log.Info("Config controller started up raft node. It is now waiting for log replay...")

    <-restored

    Log.Info("Config controller log replay complete")

    return nil
}

func (cc *ConfigController) Stop() {
    cc.restartLock.Lock()
    defer cc.restartLock.Unlock()

    if cc.stop == nil {
        return
    }

    cc.raftNode.Stop()
    close(cc.stop)
    cc.stop = nil
}

func (cc *ConfigController) ClusterController() *ClusterController {
    return cc.clusterController
}

func (cc *ConfigController) nextCommandID() uint64 {
    return UUID64()
}
