package builtin

import (
    . "github.com/PelionIoT/devicedb/bucket"
    . "github.com/PelionIoT/devicedb/storage"
    . "github.com/PelionIoT/devicedb/resolver/strategies"
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