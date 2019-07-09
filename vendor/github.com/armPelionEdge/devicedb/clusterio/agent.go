package clusterio

import (
    "context"
    "fmt"
    "github.com/prometheus/client_golang/prometheus"
    "sync"
    "time"

    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/logging"
    . "github.com/armPelionEdge/devicedb/routes"
)

var (
    prometheusRequestCounts = prometheus.NewCounterVec(prometheus.CounterOpts{
        Namespace: "sites",
        Subsystem: "devicedb_internal",
        Name: "request_counts",
        Help: "The number of requests",
    }, []string{
        "type",
        "source_node",
        "endpoint_node",
    })

    prometheusRequestFailures = prometheus.NewCounterVec(prometheus.CounterOpts{
        Namespace: "sites",
        Subsystem: "devicedb_internal",
        Name: "request_failures",
        Help: "The number of request failures",
    }, []string{
        "type",
        "source_node",        
        "endpoint_node",
    })

    prometheusReachabilityStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "devicedb_peer_reachability",
        Help: "A binary guage indicating the reachability status of peer nodes",
    }, []string{
        "node",
	})
)

func init() {
    prometheus.MustRegister(prometheusRequestCounts, prometheusRequestFailures, prometheusReachabilityStatus)
}

var DefaultTimeout time.Duration = time.Second * 20

type getResult struct {
    nodeID uint64
    siblingSets []*SiblingSet
}

type getMatchesResult struct {
    nodeID uint64
    siblingSetIterator SiblingSetIterator
}

type Agent struct {
    PartitionResolver PartitionResolver
    NodeClient NodeClient
    NodeReadRepairer NodeReadRepairer
    Timeout time.Duration
    mu sync.Mutex
    nextOperationID uint64
    operationCancellers map[uint64]func()
}

func NewAgent(nodeClient NodeClient, partitionResolver PartitionResolver) *Agent {
    readRepairer := NewReadRepairer(nodeClient)
    readRepairer.Timeout = DefaultTimeout

    return &Agent{
        Timeout: DefaultTimeout,
        PartitionResolver: partitionResolver,
        NodeClient: nodeClient,
        NodeReadRepairer: readRepairer,
        operationCancellers: make(map[uint64]func(), 0),
    }
}

func (agent *Agent) recordRequestMetrics(requestType string, destinationNode uint64, err error) {
    var labels = prometheus.Labels{
        "type": requestType,
        "source_node": fmt.Sprintf("%d", agent.NodeClient.LocalNodeID()),
        "endpoint_node": fmt.Sprintf("%d", destinationNode),
    }

    prometheusRequestCounts.With(labels).Inc()

    var connectivityStatus float64 = 1

    if err != nil {
        prometheusRequestFailures.With(labels).Inc()


        // The connectivity status should only be set to 0 if the error was connectivity based, not a request related error produced
        // by the destination node.
        if _, ok := err.(DBerror); !ok {
            connectivityStatus = 0
        }
    }
        
    prometheusReachabilityStatus.With(prometheus.Labels{ "node": labels["endpoint_node"] }).Set(connectivityStatus)
}

func (agent *Agent) Merge(ctx context.Context, siteID string, bucket string, patch map[string]*SiblingSet) (int, int, error) {
    var partitionNumber uint64 = agent.PartitionResolver.Partition(siteID)
    var replicaNodes []uint64 = agent.PartitionResolver.ReplicaNodes(partitionNumber)
    var resultError error = ENoQuorum

    opID, ctxDeadline := agent.newOperation(ctx)

    var remainingNodes map[uint64]bool = make(map[uint64]bool, len(replicaNodes))
    
    for _, nodeID := range replicaNodes {
        remainingNodes[nodeID] = true
    }

    nTotal := len(remainingNodes)
    nMerged, err := agent.merge(ctxDeadline, opID, remainingNodes, agent.NQuorum(nTotal), partitionNumber, siteID, bucket, patch, false)

    if err == ENoQuorum {
        // If a specific error occurred before this overrides ENoQuorum
        err = resultError
    }

    return nTotal, nMerged, err
}

func (agent *Agent) Batch(ctx context.Context, siteID string, bucket string, updateBatch *UpdateBatch) (int, int, error) {
    var partitionNumber uint64 = agent.PartitionResolver.Partition(siteID)
    var replicaNodes []uint64 = agent.PartitionResolver.ReplicaNodes(partitionNumber)
    var resultError error = ENoQuorum
    var nFailed int

    opID, ctxDeadline := agent.newOperation(ctx)

    var remainingNodes map[uint64]bool = make(map[uint64]bool, len(replicaNodes))
    
    for _, nodeID := range replicaNodes {
        remainingNodes[nodeID] = true
    }

    var nTotal int = len(remainingNodes)

    // If local node is included in remainingNodes it should be attempted first
    if remainingNodes[agent.NodeClient.LocalNodeID()] {
        nodeID := agent.NodeClient.LocalNodeID()
        patch, err := agent.NodeClient.Batch(ctxDeadline, nodeID, partitionNumber, siteID, bucket, updateBatch)

        delete(remainingNodes, nodeID)

        agent.recordRequestMetrics("batch", nodeID, err)

        if err != nil {
            Log.Errorf("Unable to execute batch update to bucket %s at site %s at node %d: %v", bucket, siteID, nodeID, err.Error())

            if err == EBucketDoesNotExist || err == ESiteDoesNotExist {
                resultError = err
            }

            nFailed++
        } else {
            // passing in NQuorum() - 1 since one node was already successful in applying the update so
            // we require one less node to achieve write quorum
            nMerged, err := agent.merge(ctxDeadline, opID, remainingNodes, agent.NQuorum(nTotal) - 1, partitionNumber, siteID, bucket, patch, true)

            if err == ENoQuorum {
                // If a specific error occurred before this overrides ENoQuorum
                err = resultError
            }

            return nTotal, nMerged + 1, err
        }
    }

    for nodeID, _ := range remainingNodes {
        patch, err := agent.NodeClient.Batch(ctxDeadline, nodeID, partitionNumber, siteID, bucket, updateBatch)

        delete(remainingNodes, nodeID)

        agent.recordRequestMetrics("batch", nodeID, err)

        if err != nil {
            Log.Errorf("Unable to execute batch update to bucket %s at site %s at node %d: %v", bucket, siteID, nodeID, err.Error())

            if err == EBucketDoesNotExist || err == ESiteDoesNotExist {
                resultError = err
            }

            nFailed++

            continue
        }

        // passing in NQuorum() - 1 since one node was already successful in applying the update so
        // we require one less node to achieve write quorum
        nMerged, err := agent.merge(ctxDeadline, opID, remainingNodes, agent.NQuorum(nTotal) - 1, partitionNumber, siteID, bucket, patch, true)

        if err == ENoQuorum {
            // If a specific error occurred before this overrides ENoQuorum
            err = resultError
        }

        return nTotal, nMerged + 1, err
    }

    return nTotal, 0, resultError
}

func (agent *Agent) merge(ctx context.Context, opID uint64, nodes map[uint64]bool, nQuorum int, partitionNumber uint64, siteID string, bucket string, patch map[string]*SiblingSet, broadcastToRelays bool) (int, error) {
    var nApplied int = 0
    var nFailed int = 0
    var applied chan int = make(chan int, len(nodes))
    var failed chan error = make(chan error, len(nodes))
    var resultError error = ENoQuorum

    for nodeID, _ := range nodes {
        go func(nodeID uint64) {
            err := agent.NodeClient.Merge(ctx, nodeID, partitionNumber, siteID, bucket, patch, broadcastToRelays)

            agent.recordRequestMetrics("merge", nodeID, err)

            if err != nil {
                Log.Errorf("Unable to merge patch into bucket %s at site %s at node %d: %v", bucket, siteID, nodeID, err.Error())

                failed <- err

                return
            }

            applied <- 1
        }(nodeID)
    }

    var quorumReached chan int = make(chan int)
    var allAttemptsMade chan int = make(chan int, 1)

    go func() {
        // All calls to NodeClient.Merge() will succeed, receive a failure reponse from the specified peer, or time out eventually
        for nApplied + nFailed < len(nodes) {
            select {
            case err := <-failed:
                // indicates that a call to NodeClient.Merge() either received an error response from the specified node or timed out
                nFailed++

                if err == EBucketDoesNotExist || err == ESiteDoesNotExist {
                    resultError = err
                }
            case <-applied:
                // indicates that a call to NodeClient.Merge() received a success response from the specified node
                nApplied++

                if nApplied == nQuorum {
                    // if a quorum of nodes have successfully been written to then the Merge() function should return successfully
                    // without waiting for the rest of the responses to come in. However, this function should continue to run
                    // until the deadline is reached or a response (whether it be failure or success) is received for each call
                    // to Merge() to allow this update to propogate to all replicas
                    quorumReached <- nApplied
                }
            }
        }

        // Once the deadline is reached or all calls to Merge() have received a response this channel should be written
        // to so that if Merge() is still waiting to return (since quorum was never reached) it will return an error response
        // indicating that quorum was not establish and the update was not successfully applied
        allAttemptsMade <- nApplied
        // Do this to remove the canceller from the map even though the operation is already done
        agent.cancelOperation(opID)
    }()

    if len(nodes) == 0 {
        if nQuorum == 0 {
            return 0, nil
        }

        return 0, ENoQuorum
    }

    select {
    case n := <-allAttemptsMade:
        return n, resultError
    case n := <-quorumReached:
        return n, nil
    }
}

func (agent *Agent) Get(ctx context.Context, siteID string, bucket string, keys [][]byte) ([]*SiblingSet, error) {
    var partitionNumber uint64 = agent.PartitionResolver.Partition(siteID)
    var replicaNodes []uint64 = agent.PartitionResolver.ReplicaNodes(partitionNumber)
    var readMerger *ReadMerger = NewReadMerger(bucket)
    var readResults chan getResult = make(chan getResult, len(replicaNodes))
    var failed chan error = make(chan error, len(replicaNodes))
    var nRead int = 0
    var nFailed int = 0
    var resultError error = ENoQuorum

    opID, ctxDeadline := agent.newOperation(ctx)

    var appliedNodes map[uint64]bool = make(map[uint64]bool, len(replicaNodes))
    
    for _, nodeID := range replicaNodes {
        if appliedNodes[nodeID] {
            continue
        }

        appliedNodes[nodeID] = true

        go func(nodeID uint64) {
            siblingSets, err := agent.NodeClient.Get(ctxDeadline, nodeID, partitionNumber, siteID, bucket, keys)

            agent.recordRequestMetrics("get", nodeID, err)

            if err != nil {
                Log.Errorf("Unable to get keys from bucket %s at site %s at node %d: %v", bucket, siteID, nodeID, err.Error())

                failed <- err

                return
            }

            readResults <- getResult{ nodeID: nodeID, siblingSets: siblingSets }
        }(nodeID)
    }

    var mergedResult chan []*SiblingSet = make(chan []*SiblingSet)
    var allAttemptsMade chan int = make(chan int, 1)

    go func() {
        for nRead + nFailed < len(appliedNodes) {
            select {
            case err := <-failed:
                nFailed++

                if err == EBucketDoesNotExist || err == ESiteDoesNotExist {
                    resultError = err
                }
            case r := <-readResults:
                nRead++

                for i, key := range keys {
                    readMerger.InsertKeyReplica(r.nodeID, string(key), r.siblingSets[i])
                }

                if nRead == agent.NQuorum(len(appliedNodes)) {
                    // calculate result set
                    var resultSet []*SiblingSet = make([]*SiblingSet, len(keys))

                    for i, key := range keys {
                        resultSet[i] = readMerger.Get(string(key))
                    }

                    mergedResult <- resultSet
                }
            }
        }

        agent.NodeReadRepairer.BeginRepair(partitionNumber, siteID, bucket, readMerger)
        allAttemptsMade <- 1
        agent.cancelOperation(opID)
    }()

    select {
    case result := <-mergedResult:
        return result, nil
    case <-allAttemptsMade:
        return nil, resultError
    }
}

func (agent *Agent) GetMatches(ctx context.Context, siteID string, bucket string, keys [][]byte) (SiblingSetIterator, error) {
    var partitionNumber uint64 = agent.PartitionResolver.Partition(siteID)
    var replicaNodes []uint64 = agent.PartitionResolver.ReplicaNodes(partitionNumber)
    var readMerger *ReadMerger = NewReadMerger(bucket)
    var mergeIterator *SiblingSetMergeIterator = NewSiblingSetMergeIterator(readMerger)
    var readResults chan getMatchesResult = make(chan getMatchesResult, len(replicaNodes))
    var failed chan error = make(chan error, len(replicaNodes))
    var nRead int = 0
    var nFailed int = 0
    var resultError error = ENoQuorum

    opID, ctxDeadline := agent.newOperation(ctx)

    var appliedNodes map[uint64]bool = make(map[uint64]bool, len(replicaNodes))

    for _, nodeID := range replicaNodes {
        if appliedNodes[nodeID] {
            continue
        }

        appliedNodes[nodeID] = true

        go func(nodeID uint64) {
            ssIterator, err := agent.NodeClient.GetMatches(ctxDeadline, nodeID, partitionNumber, siteID, bucket, keys)

            agent.recordRequestMetrics("get_matches", nodeID, err)

            if err != nil {
                Log.Errorf("Unable to get matches from bucket %s at site %s at node %d: %v", bucket, siteID, nodeID, err.Error())

                failed <- err

                return
            }

            readResults <- getMatchesResult{ nodeID: nodeID, siblingSetIterator: ssIterator }
        }(nodeID)
    }

    var quorumReached chan int = make(chan int)
    var allAttemptsMade chan int = make(chan int, 1)

    go func() {
        for nFailed + nRead < len(appliedNodes) {
            select {
            case err := <-failed:
                nFailed++
                
                if err == EBucketDoesNotExist || err == ESiteDoesNotExist {
                    resultError = err
                }
            case result := <-readResults:
                for result.siblingSetIterator.Next() {
                    readMerger.InsertKeyReplica(result.nodeID, string(result.siblingSetIterator.Key()), result.siblingSetIterator.Value())
                    mergeIterator.AddKey(string(result.siblingSetIterator.Prefix()), string(result.siblingSetIterator.Key()))
                }

                if result.siblingSetIterator.Error() != nil {
                    nFailed++
                } else {
                    nRead++
                }

                if nRead == agent.NQuorum(len(appliedNodes)) {
                    quorumReached <- 1
                }
            }
        }

        agent.NodeReadRepairer.BeginRepair(partitionNumber, siteID, bucket, readMerger)
        allAttemptsMade <- 1
        agent.cancelOperation(opID)
    }()

    select {
    case <-allAttemptsMade:
        return nil, resultError
    case <-quorumReached:
        mergeIterator.SortKeys()
        return mergeIterator, nil
    }
}

func (agent *Agent) RelayStatus(ctx context.Context, siteID string, relayID string) (RelayStatus, error) {
    var partitionNumber uint64 = agent.PartitionResolver.Partition(siteID)
    var replicaNodes []uint64 = agent.PartitionResolver.ReplicaNodes(partitionNumber)
    var results chan RelayStatus = make(chan RelayStatus, len(replicaNodes))
    var failed chan error = make(chan error, len(replicaNodes))
    var nRead int = 0
    var nFailed int = 0
    var resultError error

    opID, ctxDeadline := agent.newOperation(ctx)

    var appliedNodes map[uint64]bool = make(map[uint64]bool, len(replicaNodes))
    
    for _, nodeID := range replicaNodes {
        if appliedNodes[nodeID] {
            continue
        }

        appliedNodes[nodeID] = true

        go func(nodeID uint64) {
            relayStatus, err := agent.NodeClient.RelayStatus(ctxDeadline, nodeID, siteID, relayID)

            agent.recordRequestMetrics("relay_status", nodeID, err)

            if err != nil {
                Log.Errorf("Unable to get relay status for relay %s at node %d: %v", relayID, nodeID, err.Error())

                failed <- err

                return
            }

            results <- relayStatus
        }(nodeID)
    }

    var allAttemptsMade chan int = make(chan int, 1)
    var mergedResult chan RelayStatus = make(chan RelayStatus)

    go func() {
        for nRead + nFailed < len(appliedNodes) {
            select {
            case err := <-failed:
                nFailed++
                resultError = err
            case r := <-results:
                nRead++

                // If it's connected to this node return right away and cancel any
                // ongoing requests to get the status
                if r.Connected {
                    mergedResult <- r
                    agent.cancelOperation(opID)
                }
            }
        }

        if resultError != nil {
            allAttemptsMade <- 1
        } else {
            mergedResult <- RelayStatus{
                Connected: false,
                ConnectedTo: 0,
                Ping: 0,
                Site: "",
            }
        }

        agent.cancelOperation(opID)
    }()

    select {
    case result := <-mergedResult:
        return result, nil
    case <-allAttemptsMade:
        return RelayStatus{}, resultError
    }
}

func (agent *Agent) newOperation(ctx context.Context) (uint64, context.Context) {
    agent.mu.Lock()
    defer agent.mu.Unlock()

    var id uint64 = agent.nextOperationID
    agent.nextOperationID++

    ctxDeadline, cancel := context.WithTimeout(ctx, agent.Timeout)

    agent.operationCancellers[id] = cancel

    return id, ctxDeadline
}

func (agent *Agent) cancelOperation(id uint64) {
    agent.mu.Lock()
    defer agent.mu.Unlock()

    if cancel, ok := agent.operationCancellers[id]; ok {
        cancel()
        delete(agent.operationCancellers, id)
    }
}

func (agent *Agent) NQuorum(replicas int) int {
    return (replicas / 2) + 1
}

func (agent *Agent) CancelAll() {
    agent.mu.Lock()
    defer agent.mu.Unlock()

    agent.NodeReadRepairer.StopRepairs()
    
    for id, cancel := range agent.operationCancellers {
        cancel()
        delete(agent.operationCancellers, id)
    }
}