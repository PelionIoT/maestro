package builtin

import (
    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/storage"
    . "github.com/armPelionEdge/devicedb/resolver/strategies"
)

const (
    CloudMode = iota
    RelayMode = iota
    CloudPeerID = "cloud"
)

type CloudBucket struct {
    Store
    mode int
}

func NewCloudBucket(nodeID string, storageDriver StorageDriver, merkleDepth uint8, mode int) (*CloudBucket, error) {
    cloudBucket := &CloudBucket{
        mode: mode,
    }

    err := cloudBucket.Initialize(nodeID, storageDriver, merkleDepth, &MultiValue{})

    if err != nil {
        return nil, err
    }

    return cloudBucket, nil
}

func (cloudBucket *CloudBucket) Name() string {
    return "cloud"
}

func (cloudBucket *CloudBucket) ShouldReplicateOutgoing(peerID string) bool {
    return cloudBucket.mode == CloudMode
}

func (cloudBucket *CloudBucket) ShouldReplicateIncoming(peerID string) bool {
    return cloudBucket.mode == RelayMode && peerID == CloudPeerID
}

func (cloudBucket *CloudBucket) ShouldAcceptWrites(clientID string) bool {
    return cloudBucket.mode == CloudMode
}

func (cloudBucket *CloudBucket) ShouldAcceptReads(clientID string) bool {
    return true
}