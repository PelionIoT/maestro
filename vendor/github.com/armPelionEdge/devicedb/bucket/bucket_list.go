package bucket
//
 // Copyright (c) 2019 ARM Limited.
 //
 // SPDX-License-Identifier: MIT
 //
 // Permission is hereby granted, free of charge, to any person obtaining a copy
 // of this software and associated documentation files (the "Software"), to
 // deal in the Software without restriction, including without limitation the
 // rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
 // sell copies of the Software, and to permit persons to whom the Software is
 // furnished to do so, subject to the following conditions:
 //
 // The above copyright notice and this permission notice shall be included in all
 // copies or substantial portions of the Software.
 //
 // THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 // IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 // FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 // AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 // LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 // OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 // SOFTWARE.
 //


// each namespace in the database has two main factors that differentiate it from other namespaces or buckets
// 1) Replication strategy
//        . some buckets are incoming only, we never send data from the bucket out to another node
//           - but are only incoming from a specific other node, i.e. some master node
//        . some buckets are both incoming and outgoing. Every database node shares with every other node
//        . some buckets can be neither, storing data local to that node only
// 2) Conflict resolution strategy
//        . whether or not to merge conflicting siblings
//        . the way in which conflicting sibilngs are merged into one sibling

type BucketList struct {
    buckets map[string]Bucket
}

func NewBucketList() *BucketList {
    return &BucketList{ make(map[string]Bucket) }
}

func (bucketList *BucketList) AddBucket(bucket Bucket) *BucketList {
    bucketList.buckets[bucket.Name()] = bucket
    
    return bucketList
}

func (bucketList *BucketList) Outgoing(peerID string) []Bucket {
    buckets := make([]Bucket, 0, len(bucketList.buckets))
    
    for _, bucket := range bucketList.buckets {
        if bucket.ShouldReplicateOutgoing(peerID) {
            buckets = append(buckets, bucket)
        }
    }
    
    return buckets
}

func (bucketList *BucketList) Incoming(peerID string) []Bucket {
    buckets := make([]Bucket, 0, len(bucketList.buckets))
    
    for _, bucket := range bucketList.buckets {
        if bucket.ShouldReplicateIncoming(peerID) {
            buckets = append(buckets, bucket)
        }
    }
    
    return buckets
}

func (bucketList *BucketList) All() []Bucket {
    buckets := make([]Bucket, 0, len(bucketList.buckets))
    
    for _, bucket := range bucketList.buckets {
        buckets = append(buckets, bucket)
    }
    
    return buckets
}

func (bucketList *BucketList) HasBucket(bucketName string) bool {
    _, ok := bucketList.buckets[bucketName]
    
    return ok
}

func (bucketList *BucketList) Get(bucketName string) Bucket {
    return bucketList.buckets[bucketName]
}