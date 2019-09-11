package server

import (
    "encoding/json"

    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/logging"
    ddbSync "github.com/armPelionEdge/devicedb/sync"
)

const (
    START = iota
    HANDSHAKE = iota
    ROOT_HASH_COMPARE = iota
    LEFT_HASH_COMPARE = iota
    RIGHT_HASH_COMPARE = iota
    HASH_COMPARE = iota
    DB_OBJECT_PUSH = iota
    END = iota
)

func StateName(s int) string {
    names := map[int]string{
        START: "START",
        HANDSHAKE: "HANDSHAKE",
        ROOT_HASH_COMPARE: "ROOT_HASH_COMPARE",
        LEFT_HASH_COMPARE: "LEFT_HASH_COMPARE",
        RIGHT_HASH_COMPARE: "RIGHT_HASH_COMPARE",
        HASH_COMPARE: "HASH_COMPARE",
        DB_OBJECT_PUSH: "DB_OBJECT_PUSH",
        END: "END",
    }
    
    return names[s]
}

const PROTOCOL_VERSION uint = 2

// the state machine
type InitiatorSyncSession struct {
    sessionID uint
    currentState int
    maxDepth uint8
    theirDepth uint8
    explorationQueue []uint32
    explorationPathLimit uint32
    bucketProxy ddbSync.BucketProxy
    replicatesOutgoing bool
    currentNodeKeys map[string]bool
}

func NewInitiatorSyncSession(id uint, bucketProxy ddbSync.BucketProxy, explorationPathLimit uint32, replicatesOutgoing bool) *InitiatorSyncSession {
    return &InitiatorSyncSession{
        sessionID: id,
        currentState: START,
        maxDepth: bucketProxy.MerkleTree().Depth(),
        explorationQueue: make([]uint32, 0),
        explorationPathLimit: explorationPathLimit,
        bucketProxy: bucketProxy,
        replicatesOutgoing: replicatesOutgoing,
    }
}

func (syncSession *InitiatorSyncSession) State() int {
    return syncSession.currentState
}

func (syncSession *InitiatorSyncSession) SetState(state int) {
    syncSession.currentState = state
}

func (syncSession *InitiatorSyncSession) SetResponderDepth(d uint8) {
    if d < syncSession.maxDepth {
        syncSession.maxDepth = d
    }
    
    syncSession.theirDepth = d
}

func (syncSession *InitiatorSyncSession) ResponderDepth() uint8 {
    return syncSession.theirDepth
}

func (syncSession *InitiatorSyncSession) PushExplorationQueue(n uint32) {
    syncSession.explorationQueue = append(syncSession.explorationQueue, n)
}

func (syncSession *InitiatorSyncSession) PopExplorationQueue() uint32 {
    var n uint32

    if len(syncSession.explorationQueue) != 0 {
        n = syncSession.explorationQueue[0]

        syncSession.explorationQueue = syncSession.explorationQueue[1:]
    }

    return n
}

func (syncSession *InitiatorSyncSession) PeekExplorationQueue() uint32 {
    var n uint32

    if len(syncSession.explorationQueue) != 0 {
        n = syncSession.explorationQueue[0]
    }

    return n
}

func (syncSession *InitiatorSyncSession) ExplorationQueueSize() uint32 {
    return uint32(len(syncSession.explorationQueue))
}

func (syncSession *InitiatorSyncSession) SetExplorationPathLimit(limit uint32) {
    syncSession.explorationPathLimit = limit
}

func (syncSession *InitiatorSyncSession) ExplorationPathLimit() uint32 {
    return syncSession.explorationPathLimit
}

func (syncSession *InitiatorSyncSession) getNodeKeys() error {
    if syncSession.replicatesOutgoing {
        return nil
    }
    
    nodeKeys := make(map[string]bool)
    iter, err := syncSession.bucketProxy.GetSyncChildren(syncSession.PeekExplorationQueue())
    
    if err != nil {
        return err
    }
    
    for iter.Next() {
        nodeKeys[string(iter.Key())] = true
    }

    iter.Release()

    if iter.Error() != nil {
        return iter.Error()
    }

    syncSession.currentNodeKeys = nodeKeys

    return nil
}

func (syncSession *InitiatorSyncSession) forgetNonAuthoritativeKeys() error {
    if syncSession.replicatesOutgoing {
        return nil
    }

    nodeKeys := make([][]byte, 0, len(syncSession.currentNodeKeys))

    for key, _ := range syncSession.currentNodeKeys {
        nodeKeys = append(nodeKeys, []byte(key))
    }

    return syncSession.bucketProxy.Forget(nodeKeys)
}

func (syncSession *InitiatorSyncSession) NextState(syncMessageWrapper *SyncMessageWrapper) *SyncMessageWrapper {
    // Once an error occurs in the MerkleTree() the merkle tree will remain in the error state
    // should abort this sync session
    var messageWrapper *SyncMessageWrapper

    switch syncSession.currentState {
    case START:
        syncSession.currentState = HANDSHAKE
    
        messageWrapper = &SyncMessageWrapper{
            SessionID: syncSession.sessionID,
            MessageType: SYNC_START,
            MessageBody: Start{
                ProtocolVersion: PROTOCOL_VERSION,
                MerkleDepth: syncSession.bucketProxy.MerkleTree().Depth(),
                Bucket: syncSession.bucketProxy.Name(),
            },
        }

        break
    case HANDSHAKE:
        if syncMessageWrapper == nil || syncMessageWrapper.MessageType != SYNC_START {
            syncSession.currentState = END
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: &Abort{ },
            }

            break
        }

        if syncMessageWrapper.MessageBody.(Start).ProtocolVersion != PROTOCOL_VERSION {
            Log.Warningf("Initiator Session %d: responder protocol version is at %d which is unsupported by this database peer. Aborting...", syncSession.sessionID, syncMessageWrapper.MessageBody.(Start).ProtocolVersion)
            
            syncSession.currentState = END
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: &Abort{ },
            }

            break
        }
        
        if syncSession.maxDepth > syncMessageWrapper.MessageBody.(Start).MerkleDepth {
            syncSession.maxDepth = syncMessageWrapper.MessageBody.(Start).MerkleDepth
        }
        
        syncSession.theirDepth = syncMessageWrapper.MessageBody.(Start).MerkleDepth
        syncSession.currentState = ROOT_HASH_COMPARE
        syncSession.PushExplorationQueue(syncSession.bucketProxy.MerkleTree().RootNode())
    
        messageWrapper = &SyncMessageWrapper{
            SessionID: syncSession.sessionID,
            MessageType: SYNC_NODE_HASH,
            MessageBody: MerkleNodeHash{
                NodeID: syncSession.bucketProxy.MerkleTree().TranslateNode(syncSession.PeekExplorationQueue(), syncSession.theirDepth),
                HashHigh: syncSession.bucketProxy.MerkleTree().NodeHash(syncSession.PeekExplorationQueue()).High(),
                HashLow: syncSession.bucketProxy.MerkleTree().NodeHash(syncSession.PeekExplorationQueue()).Low(),
            },
        }

        break
    case ROOT_HASH_COMPARE:
        myHash := syncSession.bucketProxy.MerkleTree().NodeHash(syncSession.PeekExplorationQueue())
        
        if syncMessageWrapper == nil || syncMessageWrapper.MessageType != SYNC_NODE_HASH {
            syncSession.currentState = END
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        } else if syncMessageWrapper.MessageBody.(MerkleNodeHash).HashHigh == myHash.High() && syncMessageWrapper.MessageBody.(MerkleNodeHash).HashLow == myHash.Low() {
            syncSession.currentState = END
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        } else if syncSession.bucketProxy.MerkleTree().Level(syncSession.PeekExplorationQueue()) != syncSession.maxDepth {
            syncSession.currentState = LEFT_HASH_COMPARE
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_NODE_HASH,
                MessageBody: MerkleNodeHash{
                    NodeID: syncSession.bucketProxy.MerkleTree().TranslateNode(syncSession.bucketProxy.MerkleTree().LeftChild(syncSession.PeekExplorationQueue()), syncSession.theirDepth),
                    HashHigh: 0,
                    HashLow: 0,
                },
            }

            break
        } else {
            syncSession.currentState = DB_OBJECT_PUSH
                
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_OBJECT_NEXT,
                MessageBody: ObjectNext{
                    NodeID: syncSession.bucketProxy.MerkleTree().TranslateNode(syncSession.PeekExplorationQueue(), syncSession.theirDepth),
                },
            }

            break
        }
    case LEFT_HASH_COMPARE:
        myLeftChildHash := syncSession.bucketProxy.MerkleTree().NodeHash(syncSession.bucketProxy.MerkleTree().LeftChild(syncSession.PeekExplorationQueue()))
        
        if syncMessageWrapper == nil || syncMessageWrapper.MessageType != SYNC_NODE_HASH {
            syncSession.currentState = END
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        }
        
        if syncMessageWrapper.MessageBody.(MerkleNodeHash).HashHigh != myLeftChildHash.High() || syncMessageWrapper.MessageBody.(MerkleNodeHash).HashLow != myLeftChildHash.Low() {
            syncSession.PushExplorationQueue(syncSession.bucketProxy.MerkleTree().LeftChild(syncSession.PeekExplorationQueue()))
        }

        syncSession.currentState = RIGHT_HASH_COMPARE
        
        messageWrapper = &SyncMessageWrapper{
            SessionID: syncSession.sessionID,
            MessageType: SYNC_NODE_HASH,
            MessageBody: MerkleNodeHash{
                NodeID: syncSession.bucketProxy.MerkleTree().TranslateNode(syncSession.bucketProxy.MerkleTree().RightChild(syncSession.PeekExplorationQueue()), syncSession.theirDepth),
                HashHigh: 0,
                HashLow: 0,
            },
        }

        break
    case RIGHT_HASH_COMPARE:
        myRightChildHash := syncSession.bucketProxy.MerkleTree().NodeHash(syncSession.bucketProxy.MerkleTree().RightChild(syncSession.PeekExplorationQueue()))
        
        if syncMessageWrapper == nil || syncMessageWrapper.MessageType != SYNC_NODE_HASH {
            syncSession.currentState = END
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        }

        if syncMessageWrapper.MessageBody.(MerkleNodeHash).HashHigh != myRightChildHash.High() || syncMessageWrapper.MessageBody.(MerkleNodeHash).HashLow != myRightChildHash.Low() {
            if syncSession.ExplorationQueueSize() <= syncSession.ExplorationPathLimit() {
                syncSession.PushExplorationQueue(syncSession.bucketProxy.MerkleTree().RightChild(syncSession.PeekExplorationQueue()))
            }
        }

        syncSession.PopExplorationQueue()

        if syncSession.ExplorationQueueSize() == 0 {
            // no more nodes to explore. abort
            syncSession.currentState = END
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        } else if syncSession.bucketProxy.MerkleTree().Level(syncSession.PeekExplorationQueue()) == syncSession.maxDepth {
            // cannot dig any deeper in any paths. need to move to DB_OBJECT_PUSH
            err := syncSession.getNodeKeys()

            if err != nil {
                syncSession.currentState = END
                
                messageWrapper = &SyncMessageWrapper{
                    SessionID: syncSession.sessionID,
                    MessageType: SYNC_ABORT,
                    MessageBody: Abort{ },
                }

                break
            }

            syncSession.currentState = DB_OBJECT_PUSH

            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_OBJECT_NEXT,
                MessageBody: ObjectNext{
                    NodeID: syncSession.bucketProxy.MerkleTree().TranslateNode(syncSession.PeekExplorationQueue(), syncSession.theirDepth),
                },
            }

            break
        }

        syncSession.currentState = LEFT_HASH_COMPARE

        messageWrapper = &SyncMessageWrapper{
            SessionID: syncSession.sessionID,
            MessageType: SYNC_NODE_HASH,
            MessageBody: MerkleNodeHash{
                NodeID: syncSession.bucketProxy.MerkleTree().TranslateNode(syncSession.bucketProxy.MerkleTree().LeftChild(syncSession.PeekExplorationQueue()), syncSession.theirDepth),
                HashHigh: 0,
                HashLow: 0,
            },
        }

        break
    case DB_OBJECT_PUSH:
        if syncMessageWrapper == nil || syncMessageWrapper.MessageType != SYNC_PUSH_MESSAGE && syncMessageWrapper.MessageType != SYNC_PUSH_DONE {
            syncSession.currentState = END
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        }

        if syncMessageWrapper.MessageType == SYNC_PUSH_DONE {
            syncSession.PopExplorationQueue()

            err := syncSession.forgetNonAuthoritativeKeys()

            if err != nil || syncSession.ExplorationQueueSize() == 0 {
                syncSession.currentState = END
                
                messageWrapper = &SyncMessageWrapper{
                    SessionID: syncSession.sessionID,
                    MessageType: SYNC_ABORT,
                    MessageBody: Abort{ },
                }

                break
            }

            err = syncSession.getNodeKeys()

            if err != nil {
                syncSession.currentState = END
                
                messageWrapper = &SyncMessageWrapper{
                    SessionID: syncSession.sessionID,
                    MessageType: SYNC_ABORT,
                    MessageBody: Abort{ },
                }

                break
            }

            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_OBJECT_NEXT,
                MessageBody: ObjectNext{
                    NodeID: syncSession.bucketProxy.MerkleTree().TranslateNode(syncSession.PeekExplorationQueue(), syncSession.theirDepth),
                },
            }
            
            break
        }

        var key string = syncMessageWrapper.MessageBody.(PushMessage).Key
        var siblingSet *SiblingSet = syncMessageWrapper.MessageBody.(PushMessage).Value

        delete(syncSession.currentNodeKeys, key)

        err := syncSession.bucketProxy.Merge(map[string]*SiblingSet{ key: siblingSet })
        
        if err != nil {
            syncSession.currentState = END
        
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        }
        
        messageWrapper = &SyncMessageWrapper{
            SessionID: syncSession.sessionID,
            MessageType: SYNC_OBJECT_NEXT,
            MessageBody: ObjectNext{
                NodeID: syncSession.bucketProxy.MerkleTree().TranslateNode(syncSession.PeekExplorationQueue(), syncSession.theirDepth),
            },
        }

        break
    case END:
        return nil
    }
    
    if syncSession.bucketProxy.MerkleTree().Error() != nil {
        Log.Errorf("Initiator sync session %d encountered a merkle tree error: %v", syncSession.sessionID, syncSession.bucketProxy.MerkleTree().Error())

        // Encountered a proxy error with the merkle tree
        // need to abort
        syncSession.currentState = END

        messageWrapper = &SyncMessageWrapper{
            SessionID: syncSession.sessionID,
            MessageType: SYNC_ABORT,
            MessageBody: Abort{ },
        }
    }

    return messageWrapper
}

    // the state machine
type ResponderSyncSession struct {
    sessionID uint
    currentState int
    node uint32
    maxDepth uint8
    theirDepth uint8
    bucketProxy ddbSync.BucketProxy
    iter SiblingSetIterator
    currentIterationNode uint32
}

func NewResponderSyncSession(bucketProxy ddbSync.BucketProxy) *ResponderSyncSession {
    return &ResponderSyncSession{
        sessionID: 0,
        currentState: START,
        node: bucketProxy.MerkleTree().RootNode(),
        maxDepth: bucketProxy.MerkleTree().Depth(),
        bucketProxy: bucketProxy,
        iter: nil,
    }
}

func (syncSession *ResponderSyncSession) State() int {
    return syncSession.currentState
}

func (syncSession *ResponderSyncSession) SetState(state int) {
    syncSession.currentState = state
}

func (syncSession *ResponderSyncSession) SetInitiatorDepth(d uint8) {
    syncSession.theirDepth = d
}

func (syncSession *ResponderSyncSession) InitiatorDepth() uint8 {
    return syncSession.theirDepth
}

func (syncSession *ResponderSyncSession) NextState(syncMessageWrapper *SyncMessageWrapper) *SyncMessageWrapper {
    var messageWrapper *SyncMessageWrapper

    switch syncSession.currentState {
    case START:
        if syncMessageWrapper != nil {
            syncSession.sessionID = syncMessageWrapper.SessionID
        }
        
        if syncMessageWrapper == nil || syncMessageWrapper.MessageType != SYNC_START {
            syncSession.currentState = END
        
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        }

        if syncMessageWrapper.MessageBody.(Start).ProtocolVersion != PROTOCOL_VERSION {
            Log.Warningf("Responder Session %d: responder protocol version is at %d which is unsupported by this database peer. Aborting...", syncSession.sessionID, syncMessageWrapper.MessageBody.(Start).ProtocolVersion)
            
            syncSession.currentState = END
        
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        }
    
        syncSession.theirDepth = syncMessageWrapper.MessageBody.(Start).MerkleDepth
        syncSession.currentState = HASH_COMPARE
    
        messageWrapper = &SyncMessageWrapper{
            SessionID: syncSession.sessionID,
            MessageType: SYNC_START,
            MessageBody: Start{
                ProtocolVersion: PROTOCOL_VERSION,
                MerkleDepth: syncSession.bucketProxy.MerkleTree().Depth(),
                Bucket: syncSession.bucketProxy.Name(),
            },
        }

        break
    case HASH_COMPARE:
        if syncMessageWrapper == nil || syncMessageWrapper.MessageType != SYNC_NODE_HASH && syncMessageWrapper.MessageType != SYNC_OBJECT_NEXT {
            syncSession.currentState = END
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        }
        
        if syncMessageWrapper.MessageType == SYNC_NODE_HASH {
            nodeID := syncMessageWrapper.MessageBody.(MerkleNodeHash).NodeID
            
            if nodeID >= syncSession.bucketProxy.MerkleTree().NodeLimit() || nodeID == 0 {
                syncSession.currentState = END
                
                messageWrapper = &SyncMessageWrapper{
                    SessionID: syncSession.sessionID,
                    MessageType: SYNC_ABORT,
                    MessageBody: Abort{ },
                }

                break
            }
            
            nodeHash := syncSession.bucketProxy.MerkleTree().NodeHash(nodeID)
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_NODE_HASH,
                MessageBody: MerkleNodeHash{
                    NodeID: syncSession.bucketProxy.MerkleTree().TranslateNode(nodeID, syncSession.theirDepth),
                    HashHigh: nodeHash.High(),
                    HashLow: nodeHash.Low(),
                },
            }

            break
        }

        // if items to iterate over, send first
        nodeID := syncMessageWrapper.MessageBody.(ObjectNext).NodeID
        syncSession.currentIterationNode = nodeID
        iter, err := syncSession.bucketProxy.GetSyncChildren(nodeID)
        
        if err != nil {
            syncSession.currentState = END
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        }
        
        if !iter.Next() {
            err := iter.Error()
            iter.Release()

            if err == nil {
                syncSession.currentState = DB_OBJECT_PUSH

                messageWrapper = &SyncMessageWrapper{
                    SessionID: syncSession.sessionID,
                    MessageType: SYNC_PUSH_DONE,
                    MessageBody: PushDone{ },
                }

                break
            }
            
            syncSession.currentState = END
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        }
    
        syncSession.iter = iter
        syncSession.currentState = DB_OBJECT_PUSH
        
        messageWrapper = &SyncMessageWrapper{
            SessionID: syncSession.sessionID,
            MessageType: SYNC_PUSH_MESSAGE,
            MessageBody: PushMessage{
                Key: string(iter.Key()),
                Value: iter.Value(),
            },
        }

        break
    case DB_OBJECT_PUSH:
        if syncMessageWrapper == nil || syncMessageWrapper.MessageType != SYNC_OBJECT_NEXT {
            if syncSession.iter != nil {
                syncSession.iter.Release()
            }
                
            syncSession.currentState = END
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        }

        if syncSession.iter == nil {
            syncSession.currentState = END
                
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        }

        if syncSession.currentIterationNode != syncMessageWrapper.MessageBody.(ObjectNext).NodeID {
            if syncSession.iter != nil {
                syncSession.iter.Release()
            }

            syncSession.currentIterationNode = syncMessageWrapper.MessageBody.(ObjectNext).NodeID
            iter, err := syncSession.bucketProxy.GetSyncChildren(syncSession.currentIterationNode)

            if err != nil {
                syncSession.currentState = END
                
                messageWrapper = &SyncMessageWrapper{
                    SessionID: syncSession.sessionID,
                    MessageType: SYNC_ABORT,
                    MessageBody: Abort{ },
                }

                break
            }

            syncSession.iter = iter
        }
        
        if !syncSession.iter.Next() {
            if syncSession.iter != nil {
                err := syncSession.iter.Error()

                syncSession.iter.Release()

                if err == nil {
                    messageWrapper = &SyncMessageWrapper{
                        SessionID: syncSession.sessionID,
                        MessageType: SYNC_PUSH_DONE,
                        MessageBody: PushDone{ },
                    }

                    break
                }
            }
            
            syncSession.currentState = END
            
            messageWrapper = &SyncMessageWrapper{
                SessionID: syncSession.sessionID,
                MessageType: SYNC_ABORT,
                MessageBody: Abort{ },
            }

            break
        }
        
        messageWrapper = &SyncMessageWrapper{
            SessionID: syncSession.sessionID,
            MessageType: SYNC_PUSH_MESSAGE,
            MessageBody: PushMessage{
                Key: string(syncSession.iter.Key()),
                Value: syncSession.iter.Value(),
            },
        }

        break
    case END:
        return nil
    }

    if syncSession.bucketProxy.MerkleTree().Error() != nil {
        // Encountered a proxy error with the merkle tree
        // need to abort
        Log.Errorf("Initiator sync session %d encountered a merkle tree error: %v", syncSession.sessionID, syncSession.bucketProxy.MerkleTree().Error())
        
        syncSession.currentState = END

        messageWrapper = &SyncMessageWrapper{
            SessionID: syncSession.sessionID,
            MessageType: SYNC_ABORT,
            MessageBody: Abort{ },
        }
    }
    
    return messageWrapper
}

const (
    SYNC_START = iota
    SYNC_ABORT = iota
    SYNC_NODE_HASH = iota
    SYNC_OBJECT_NEXT = iota
    SYNC_PUSH_MESSAGE = iota
    REQUEST = iota
    RESPONSE = iota
    PUSH = iota
    SYNC_PUSH_DONE = iota
)

func MessageTypeName(m int) string {
    names := map[int]string{
        SYNC_START: "SYNC_START",
        SYNC_ABORT: "SYNC_ABORT",
        SYNC_NODE_HASH: "SYNC_NODE_HASH",
        SYNC_OBJECT_NEXT: "SYNC_OBJECT_NEXT",
        SYNC_PUSH_MESSAGE: "SYNC_PUSH_MESSAGE",
        SYNC_PUSH_DONE: "SYNC_PUSH_DONE",
    }
    
    return names[m]
}

type rawSyncMessageWrapper struct {
    SessionID uint `json:"sessionID"`
    MessageType int `json:"type"`
    MessageBody json.RawMessage `json:"body"`
    Direction uint `json:"dir"`
    nodeID string
}

type SyncMessageWrapper struct {
    SessionID uint `json:"sessionID"`
    MessageType int `json:"type"`
    MessageBody interface{ } `json:"body"`
    Direction uint `json:"dir"`
    nodeID string
}

type Start struct {
    ProtocolVersion uint
    MerkleDepth uint8
    Bucket string
}

type Abort struct {
}

type MerkleNodeHash struct {
    NodeID uint32
    HashHigh uint64
    HashLow uint64
}

type ObjectNext struct {
    NodeID uint32
}

type PushMessage struct {
    Bucket string
    Key string
    Value *SiblingSet
}

type PushDone struct {
}