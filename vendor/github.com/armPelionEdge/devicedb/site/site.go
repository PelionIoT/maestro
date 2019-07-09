package site

import (
    . "github.com/armPelionEdge/devicedb/bucket"
)

type Site interface {
    Buckets() *BucketList
    Iterator() SiteIterator
    ID() string
    LockWrites()
    UnlockWrites()
    LockReads()
    UnlockReads()
}

type RelaySiteReplica struct {
    bucketList *BucketList
    id string
}

func NewRelaySiteReplica(id string, buckets *BucketList) *RelaySiteReplica {
    return &RelaySiteReplica{
        id: id,
        bucketList: buckets,
    }
}

func (relaySiteReplica *RelaySiteReplica) Buckets() *BucketList {
    if relaySiteReplica == nil {
        return NewBucketList()
    }

    return relaySiteReplica.bucketList
}

func (relaySiteReplica *RelaySiteReplica) ID() string {
    return relaySiteReplica.id
}

func (relaySiteReplica *RelaySiteReplica) Iterator() SiteIterator {
    return &RelaySiteIterator{ }
}

func (relaySiteReplica *RelaySiteReplica) LockWrites() {
}

func (relaySiteReplica *RelaySiteReplica) UnlockWrites() {
}

func (relaySiteReplica *RelaySiteReplica) LockReads() {
}

func (relaySiteReplica *RelaySiteReplica) UnlockReads() {
}

type CloudSiteReplica struct {
    bucketList *BucketList
    id string
}

func (cloudSiteReplica *CloudSiteReplica) Buckets() *BucketList {
    if cloudSiteReplica == nil {
        return NewBucketList()
    }

    return cloudSiteReplica.bucketList
}

func (cloudSiteReplica *CloudSiteReplica) ID() string {
    return cloudSiteReplica.id
}

func (cloudSiteReplica *CloudSiteReplica) Iterator() SiteIterator {
    return &CloudSiteIterator{ buckets: cloudSiteReplica.bucketList.All() }
}

func (cloudSiteReplica *CloudSiteReplica) LockWrites() {
    for _, bucket := range cloudSiteReplica.bucketList.All() {
        bucket.LockWrites()
    }
}

func (cloudSiteReplica *CloudSiteReplica) UnlockWrites() {
    for _, bucket := range cloudSiteReplica.bucketList.All() {
        bucket.UnlockWrites()
    }
}

func (cloudSiteReplica *CloudSiteReplica) LockReads() {
    for _, bucket := range cloudSiteReplica.bucketList.All() {
        bucket.LockReads()
    }
}

func (cloudSiteReplica *CloudSiteReplica) UnlockReads() {
    for _, bucket := range cloudSiteReplica.bucketList.All() {
        bucket.UnlockReads()
    }
}