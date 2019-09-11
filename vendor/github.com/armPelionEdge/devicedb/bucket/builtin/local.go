package builtin

import (
    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/storage"
    . "github.com/armPelionEdge/devicedb/resolver/strategies"
)

type LocalBucket struct {
    Store
}

func NewLocalBucket(nodeID string, storageDriver StorageDriver, merkleDepth uint8) (*LocalBucket, error) {
    localBucket := &LocalBucket{}

    err := localBucket.Initialize(nodeID, storageDriver, merkleDepth, &MultiValue{})

    if err != nil {
        return nil, err
    }

    return localBucket, nil
}

func (localBucket *LocalBucket) Name() string {
    return "local"
}

func (localBucket *LocalBucket) ShouldReplicateOutgoing(peerID string) bool {
    return false
}

func (localBucket *LocalBucket) ShouldReplicateIncoming(peerID string) bool {
    return false 
}

func (localBucket *LocalBucket) ShouldAcceptWrites(clientID string) bool {
    return true
}

func (localBucket *LocalBucket) ShouldAcceptReads(clientID string) bool {
    return true
}