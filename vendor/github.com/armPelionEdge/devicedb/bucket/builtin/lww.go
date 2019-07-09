package builtin

import (
    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/storage"
    . "github.com/armPelionEdge/devicedb/resolver/strategies"
)

type LWWBucket struct {
    Store
}

func NewLWWBucket(nodeID string, storageDriver StorageDriver, merkleDepth uint8) (*LWWBucket, error) {
    lwwBucket := &LWWBucket{}

    err := lwwBucket.Initialize(nodeID, storageDriver, merkleDepth, &LastWriterWins{})

    if err != nil {
        return nil, err
    }

    return lwwBucket, nil
}

func (lwwBucket *LWWBucket) Name() string {
    return "lww"
}

func (lwwBucket *LWWBucket) ShouldReplicateOutgoing(peerID string) bool {
    return true 
}

func (lwwBucket *LWWBucket) ShouldReplicateIncoming(peerID string) bool {
    return true
}

func (lwwBucket *LWWBucket) ShouldAcceptWrites(clientID string) bool {
    return true
}

func (lwwBucket *LWWBucket) ShouldAcceptReads(clientID string) bool {
    return true
}