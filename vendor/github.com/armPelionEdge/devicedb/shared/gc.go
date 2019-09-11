package shared

import (
    "time"

    . "github.com/armPelionEdge/devicedb/logging"
    . "github.com/armPelionEdge/devicedb/bucket"
)

type GarbageCollector struct {
    buckets *BucketList
    gcInterval time.Duration
    gcPurgeAge uint64
    done chan bool
}

func NewGarbageCollector(buckets *BucketList, gcInterval uint64, gcPurgeAge uint64) *GarbageCollector {
    return &GarbageCollector{
        buckets: buckets,
        gcInterval: time.Millisecond * time.Duration(gcInterval),
        gcPurgeAge: gcPurgeAge,
        done: make(chan bool),
    }
}

func (garbageCollector *GarbageCollector) Start() {
    go func() {
        for {
            select {
            case <-garbageCollector.done:
                garbageCollector.done = make(chan bool)
                return
            case <-time.After(garbageCollector.gcInterval):
                for _, bucket := range garbageCollector.buckets.All() {
                    Log.Infof("Performing garbage collection sweep on %s bucket", bucket.Name())
                    bucket.GarbageCollect(garbageCollector.gcPurgeAge)
                }
            }
        }
    }()
}

func (garbageCollector *GarbageCollector) Stop() {
    close(garbageCollector.done)
}