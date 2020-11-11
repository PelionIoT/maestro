package builtin

import (
    . "github.com/PelionIoT/devicedb/bucket"
    . "github.com/PelionIoT/devicedb/storage"
    . "github.com/PelionIoT/devicedb/resolver/strategies"
)

type DefaultBucket struct {
    Store
}

func NewDefaultBucket(nodeID string, storageDriver StorageDriver, merkleDepth uint8) (*DefaultBucket, error) {
    defaultBucket := &DefaultBucket{}

    err := defaultBucket.Initialize(nodeID, storageDriver, merkleDepth, &MultiValue{})

    if err != nil {
        return nil, err
    }

    return defaultBucket, nil
}

func (defaultBucket *DefaultBucket) Name() string {
    return "default"
}

func (defaultBucket *DefaultBucket) ShouldReplicateOutgoing(peerID string) bool {
    return true
}

func (defaultBucket *DefaultBucket) ShouldReplicateIncoming(peerID string) bool {
    return true
}

func (defaultBucket *DefaultBucket) ShouldAcceptWrites(clientID string) bool {
    return true
}

func (defaultBucket *DefaultBucket) ShouldAcceptReads(clientID string) bool {
    return true
}