package site

import (
    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/bucket/builtin"
    . "github.com/armPelionEdge/devicedb/merkle"
    . "github.com/armPelionEdge/devicedb/storage"
)

var keyStorePrefix = []byte{ 0 }

const (
    defaultNodePrefix = iota
    cloudNodePrefix = iota
    lwwNodePrefix = iota
    localNodePrefix = iota
    historianPrefix = iota
    alertsLogPrefix = iota
)

type SiteFactory interface {
    CreateSite(siteID string) Site
}

type RelaySiteFactory struct {
    MerkleDepth uint8
    StorageDriver StorageDriver
    RelayID string
}

func (relaySiteFactory *RelaySiteFactory) CreateSite(siteID string) Site {
    bucketList := NewBucketList()

    defaultBucket, _ := NewDefaultBucket(relaySiteFactory.RelayID, NewPrefixedStorageDriver([]byte{ defaultNodePrefix }, relaySiteFactory.StorageDriver), relaySiteFactory.MerkleDepth)
    cloudBucket, _ := NewCloudBucket(relaySiteFactory.RelayID, NewPrefixedStorageDriver([]byte{ cloudNodePrefix }, relaySiteFactory.StorageDriver), relaySiteFactory.MerkleDepth, RelayMode)
    lwwBucket, _ := NewLWWBucket(relaySiteFactory.RelayID, NewPrefixedStorageDriver([]byte{ lwwNodePrefix }, relaySiteFactory.StorageDriver), relaySiteFactory.MerkleDepth)
    localBucket, _ := NewLocalBucket(relaySiteFactory.RelayID, NewPrefixedStorageDriver([]byte{ localNodePrefix }, relaySiteFactory.StorageDriver), MerkleMinDepth)
    
    bucketList.AddBucket(defaultBucket)
    bucketList.AddBucket(lwwBucket)
    bucketList.AddBucket(cloudBucket)
    bucketList.AddBucket(localBucket)

    return &RelaySiteReplica{
        bucketList: bucketList,
        id: siteID,
    }
}

type CloudSiteFactory struct {
    NodeID string
    MerkleDepth uint8
    StorageDriver StorageDriver
}

func (cloudSiteFactory *CloudSiteFactory) siteBucketStorageDriver(siteID string, bucketPrefix []byte) StorageDriver {
    return NewPrefixedStorageDriver(cloudSiteFactory.siteBucketPrefix(siteID, bucketPrefix), cloudSiteFactory.StorageDriver)
}

func (cloudSiteFactory *CloudSiteFactory) siteBucketPrefix(siteID string, bucketPrefix []byte) []byte {
    prefix := make([]byte, 0, len(keyStorePrefix) + len([]byte(siteID)) + len([]byte(".")) + len(bucketPrefix) + len([]byte(".")))

    prefix = append(prefix, keyStorePrefix...)
    prefix = append(prefix, []byte(siteID)...)
    prefix = append(prefix, []byte(".")...)
    prefix = append(prefix, bucketPrefix...)
    prefix = append(prefix, []byte(".")...)

    return prefix
}

func (cloudSiteFactory *CloudSiteFactory) CreateSite(siteID string) Site {
    bucketList := NewBucketList()

    defaultBucket, _ := NewDefaultBucket(cloudSiteFactory.NodeID, cloudSiteFactory.siteBucketStorageDriver(siteID, []byte{ defaultNodePrefix }), cloudSiteFactory.MerkleDepth)
    cloudBucket, _ := NewCloudBucket(cloudSiteFactory.NodeID, cloudSiteFactory.siteBucketStorageDriver(siteID, []byte{ cloudNodePrefix }), cloudSiteFactory.MerkleDepth, CloudMode)
    lwwBucket, _ := NewLWWBucket(cloudSiteFactory.NodeID, cloudSiteFactory.siteBucketStorageDriver(siteID, []byte{ lwwNodePrefix }), cloudSiteFactory.MerkleDepth)
    localBucket, _ := NewLocalBucket(cloudSiteFactory.NodeID, cloudSiteFactory.siteBucketStorageDriver(siteID, []byte{ localNodePrefix }), MerkleMinDepth)
    
    bucketList.AddBucket(defaultBucket)
    bucketList.AddBucket(lwwBucket)
    bucketList.AddBucket(cloudBucket)
    bucketList.AddBucket(localBucket)

    return &CloudSiteReplica{
        bucketList: bucketList,
        id: siteID,
    }
}
