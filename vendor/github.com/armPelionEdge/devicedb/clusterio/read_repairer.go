package clusterio

import (
    "context"
    "sync"
    "time"

    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/logging"
)

type ReadRepairer struct {
    NodeClient NodeClient
    Timeout time.Duration
    mu sync.Mutex
    nextOperationID uint64
    operationCancellers map[uint64]func()
    stopped bool
}

func NewReadRepairer(nodeClient NodeClient) *ReadRepairer {
    return &ReadRepairer{
        NodeClient: nodeClient,
        operationCancellers: make(map[uint64]func()),
    }
}

func (readRepairer *ReadRepairer) BeginRepair(partition uint64, siteID string, bucket string, readMerger NodeReadMerger) {
    readRepairer.mu.Lock()
    defer readRepairer.mu.Unlock()

    if readRepairer.stopped {
        return
    }

    var wg sync.WaitGroup
    opID, ctxDeadline := readRepairer.newOperation(context.Background())

    for nodeID, _ := range readMerger.Nodes() {
        patch := readMerger.Patch(nodeID)

        for key, siblingSet := range patch {
            if siblingSet.Size() == 0 {
                // Filter out keys that don't need patch
                delete(patch, key)

                continue
            }
            
            Log.Infof("Repairing key %s in bucket %s at site %s at node %d", key, bucket, siteID, nodeID)
        }

        if len(patch) == 0 {
            continue
        }

        wg.Add(1)

        go func(nodeID uint64, patch map[string]*SiblingSet) {
            defer wg.Done()

            if err := readRepairer.NodeClient.Merge(ctxDeadline, nodeID, partition, siteID, bucket, patch, true); err != nil {
                Log.Errorf("Unable to perform read repair on bucket %s at site %s at node %d: %v", bucket, siteID, nodeID, err.Error())
            }
        }(nodeID, patch)
    }

    go func() {
        wg.Wait()
        readRepairer.mu.Lock()
        defer readRepairer.mu.Unlock()

        readRepairer.cancelOperation(opID)
    }()
}

func (readRepairer *ReadRepairer) newOperation(ctx context.Context) (uint64, context.Context) {
    var id uint64 = readRepairer.nextOperationID
    readRepairer.nextOperationID++

    ctxDeadline, cancel := context.WithTimeout(ctx, readRepairer.Timeout)

    readRepairer.operationCancellers[id] = cancel

    return id, ctxDeadline
}

func (readRepairer *ReadRepairer) cancelOperation(id uint64) {
    if cancel, ok := readRepairer.operationCancellers[id]; ok {
        cancel()
        delete(readRepairer.operationCancellers, id)
    }
}

func (readRepairer *ReadRepairer) StopRepairs() {
    readRepairer.mu.Lock()
    defer readRepairer.mu.Unlock()

    if readRepairer.stopped {
        return
    }

    for opID, cancel := range readRepairer.operationCancellers {
        cancel()
        delete(readRepairer.operationCancellers, opID)
    }

    readRepairer.stopped = true
}