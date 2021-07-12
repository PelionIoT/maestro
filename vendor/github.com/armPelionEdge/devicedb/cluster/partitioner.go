package cluster
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


import (
    "errors"
    "sort"
    "sync"

    . "github.com/armPelionEdge/devicedb/data"
)

type NodeTokenCount struct {
    NodeID uint64
    TokenCount int
}

type NodeTokenCountHeap []NodeTokenCount

func (nodeTokenCountHeap NodeTokenCountHeap) Len() int {
    return len(nodeTokenCountHeap)
}

func (nodeTokenCountHeap NodeTokenCountHeap) Swap(i, j int) {
    nodeTokenCountHeap[i], nodeTokenCountHeap[j] = nodeTokenCountHeap[j], nodeTokenCountHeap[i]
}

func (nodeTokenCountHeap NodeTokenCountHeap) Less(i, j int) bool {
    if nodeTokenCountHeap[i].TokenCount == nodeTokenCountHeap[j].TokenCount {
        return nodeTokenCountHeap[i].NodeID < nodeTokenCountHeap[j].NodeID
    }

    return nodeTokenCountHeap[i].TokenCount > nodeTokenCountHeap[j].TokenCount
}

func (nodeTokenCountHeap *NodeTokenCountHeap) Push(x interface{}) {
    *nodeTokenCountHeap = append(*nodeTokenCountHeap, x.(NodeTokenCount))
}

func (nodeTokenCountHeap *NodeTokenCountHeap) Pop() interface{} {
    old := *nodeTokenCountHeap
    n := len(old)
    x := old[n - 1]
    *nodeTokenCountHeap = old[0 : n - 1]

    return x
}

const MaxPartitionCount uint64 = 65536
const DefaultPartitionCount uint64 = 1024
const MinPartitionCount uint64 = 64

var EPreconditionFailed = errors.New("Unable to validate precondition")
var ENoNodesAvailable = errors.New("Unable to assign tokens because there are no available nodes in the cluster")

type PartitioningStrategy interface {
    AssignTokens(nodes []NodeConfig, currentTokenAssignment []uint64, partitions uint64) ([]uint64, error)
    AssignPartitions(nodes []NodeConfig, currentPartitionAssignment [][]uint64)
    Owners(tokenAssignment []uint64, partition uint64, replicationFactor uint64) []uint64
    Partition(key string, partitionCount uint64) uint64
}

// Simple replication strategy that does not account for capacity other than finding nodes
// that are marked as having 0 capacity to account for decomissioned nodes. Other than that
// It just tries to assign as close to an even amount of tokens to each node as possible
type SimplePartitioningStrategy struct {
    // cached partition count
    partitionCount uint64
    // cached shift amount so it doesnt have to be recalculated every time
    shiftAmount int
    lock sync.Mutex
}

func (ps *SimplePartitioningStrategy) countAvailableNodes(nodes []NodeConfig) int {
    availableNodes := 0

    for _, node := range nodes {
        if node.Capacity == 0 {
            continue
        }

        availableNodes++
    }

    return availableNodes
}

func (ps *SimplePartitioningStrategy) countTokens(nodes []NodeConfig) []uint64 {
    tokens := make([]uint64, len(nodes))

    for i, node := range nodes {
        tokens[i] = uint64(len(node.Tokens))
    }

    return tokens
}

func (ps *SimplePartitioningStrategy) checkPreconditions(nodes []NodeConfig, currentAssignments []uint64, partitions uint64) error {
    // Precondition 1: nodes must be non-nil
    if nodes == nil {
        return EPreconditionFailed
    }

    // Precondition 2: nodes must be sorted in order of ascending node id and all node ids are unique
    if !ps.nodesAreSortedAndUnique(nodes) {
        return EPreconditionFailed
    }

    // Precondition 3: The length of currentAssignments must be equal to partitions
    if uint64(len(currentAssignments)) != partitions {
        return EPreconditionFailed
    }

    // Precondition 4: partitions must be non-zero
    if partitions == 0 {
        return EPreconditionFailed
    }

    // Precondition 5: For all assignments in currentAssignments, the node a token is assigned to must exist in nodes[] unless it is set to zero
    // which indicates that the node does not exist
    for _, owner := range currentAssignments {
        if owner == 0 {
            continue
        }

        ownerExists := false

        for _, node := range nodes {
            if node.Address.NodeID == owner {
                ownerExists = true

                break
            }
        }

        if !ownerExists {
            return EPreconditionFailed
        }
    }

    return nil
}

func (ps *SimplePartitioningStrategy) nodesAreSortedAndUnique(nodes []NodeConfig) bool {
    var lastNodeID uint64 = 0

    for _, node := range nodes {
        if lastNodeID >= node.Address.NodeID {
            return false
        }

        lastNodeID = node.Address.NodeID
    }

    return true
}

func (ps *SimplePartitioningStrategy) AssignTokens(nodes []NodeConfig, currentAssignments []uint64, partitions uint64) ([]uint64, error) {
    if err := ps.checkPreconditions(nodes, currentAssignments, partitions); err != nil {
        return nil, err
    }

    // Precondition 6: The number of nodes must be <= partitions
    if uint64(len(nodes)) > partitions {
        // in this case return a valid assignment using just a subset of nodes. limit the number of nodes
        // that are assigned tokens to be <= partitions
        nodes = nodes[:partitions]
    }

    assignments := make([]uint64, partitions)
    tokenCounts := ps.countTokens(nodes)
    availableNodes := ps.countAvailableNodes(nodes)

    if availableNodes == 0 {
        return assignments, nil
    }

    tokenCountFloor := partitions / uint64(availableNodes)
    tokenCountCeil := tokenCountFloor

    if partitions % uint64(availableNodes) != 0 {
        tokenCountCeil += 1
    }

    copy(assignments, currentAssignments)

    // unassign any token owned by a decommissioned node
    for i, node := range nodes {
        if node.Capacity != 0 {
            continue
        }

        // release tokens owned by this decommissioned node
        for token, _ := range node.Tokens {
            assignments[token] = 0
            tokenCounts[i]--
        }
    }

    var nextNode int

    // find an owner for unplaced tokens. Tokens may be unplaced due to an uninitialized cluster,
    // removed nodes, or decommissioned nodes
    for token, owner := range assignments {
        if owner != 0 {
            continue
        }

        // Token is unassigned. Need to find a home for it
        for i := 0; i < len(nodes); i++ {
            nodeIndex := (nextNode + i) % len(nodes)
            node := nodes[nodeIndex]

            if node.Capacity == 0 {
                // This node is decommissioning. It is effectively removed from the cluster
                continue
            }

            if tokenCounts[nodeIndex] < tokenCountCeil {
                assignments[token] = node.Address.NodeID
                tokenCounts[nodeIndex]++
                nextNode = (nodeIndex + 1) % len(nodes)

                break
            }
        }
    }

    // invariant: all tokens should be placed at some non-decomissioned node

    for i, _ := range nodes {
        if nodes[i].Capacity == 0 {
            // The ith node is decommissioning. It should receive none of the tokens
            continue
        }

        // Pass 1: Each time after givine a token to this node,
        // skip some tokens in an attempt to avoid adjacencies
        // Space out the placement evenly throughout the range using this skip amount
        skipAmount := len(assignments) / int(tokenCountFloor)

        if skipAmount > 1 {
            skipAmount -= 1
        }

        for token := 0; token < len(assignments) && tokenCounts[i] < tokenCountFloor; token++ {
            owner := assignments[token]
            ownerIndex := 0

            for j := 0; j < len(nodes); j++ {
                if nodes[j].Address.NodeID == owner {
                    ownerIndex = j
                    break
                }
            }

            // take this token
            if tokenCounts[ownerIndex] > tokenCountFloor {
                assignments[token] = nodes[i].Address.NodeID
                tokenCounts[i]++
                tokenCounts[ownerIndex]--

                // Increment so a space is skipped
                token += skipAmount
            }
        }

        // Pass 2: Don't skip any spaces. Just make sure the node
        // gets enough tokens assigned to it
        // Should evenly space tokens throughout the ring for this node for even
        // partition distributions
        for j := 0; tokenCounts[i] < tokenCountFloor && j < len(tokenCounts); j++ {
            if j == i || tokenCounts[j] <= tokenCountFloor {
                // a node can't steal a token from itself and it can't steal a token
                // from a node that doesn't have surplus tokens
                continue
            }

            // steal a token from the jth node
            for token, owner := range assignments {
                if owner == nodes[j].Address.NodeID {
                    assignments[token] = nodes[i].Address.NodeID
                    tokenCounts[i]++
                    tokenCounts[j]--
                  
                    // We have taken all the tokens that we can take from this node. need to move on
                    if tokenCounts[j] == tokenCountFloor {
                        break
                    }
                }
            }
        }

        // loop invariant: all nodes in nodes[:i+1] that have positive capacity have been assigned at least tokenCountFloor tokens and at most tokenCountCeil tokens
    }

    return assignments, nil
}

func (ps *SimplePartitioningStrategy) AssignPartitions(nodes []NodeConfig, currentPartitionAssignment [][]uint64) {
    sort.Sort(NodeConfigList(nodes))

    var nodeOwnershipHistogram map[uint64]int = make(map[uint64]int)

    for _, node := range nodes {
        if node.Capacity > 0 {
            nodeOwnershipHistogram[node.Address.NodeID] = 0
        }
    }

    // build histogram and mark any partition replicas in the assignment that were assigned to nodes
    // that are no longer in the cluster
    for partition := 0; partition < len(currentPartitionAssignment); partition++ {
        for replica := 0; replica < len(currentPartitionAssignment[partition]); replica++ {
            ownerID := currentPartitionAssignment[partition][replica]

            if _, ok := nodeOwnershipHistogram[ownerID]; !ok {
                currentPartitionAssignment[partition][replica] = 0
                continue
            }

            nodeOwnershipHistogram[ownerID]++
        }
    }

    if len(nodeOwnershipHistogram) == 0 {
        return
    }

    var minimumPartitionReplicas int = (len(currentPartitionAssignment) * len(currentPartitionAssignment[0])) / len(nodeOwnershipHistogram)

    for _, node := range nodes {
        var nodeID uint64 = node.Address.NodeID

        if _, ok := nodeOwnershipHistogram[nodeID]; !ok {
            continue
        }

        if nodeOwnershipHistogram[nodeID] < minimumPartitionReplicas {
            for partition := 0; partition < len(currentPartitionAssignment); partition++ {
                for replica := 0; replica < len(currentPartitionAssignment[partition]); replica++ {
                    ownerID := currentPartitionAssignment[partition][replica]

                    if ownerID == 0 || nodeOwnershipHistogram[ownerID] > minimumPartitionReplicas {
                        currentPartitionAssignment[partition][replica] = nodeID
                        nodeOwnershipHistogram[nodeID]++

                        if ownerID != 0 {
                            nodeOwnershipHistogram[ownerID]--
                        }
                    }
                }
            }
        }
    }
}

func (ps *SimplePartitioningStrategy) Owners(tokenAssignment []uint64, partition uint64, replicationFactor uint64) []uint64 {
    if tokenAssignment == nil {
        return []uint64{}
    }

    if partition >= uint64(len(tokenAssignment)) {
        return []uint64{}
    }

    ownersSet := make(map[uint64]bool, int(replicationFactor))
    owners := make([]uint64, 0, int(replicationFactor))

    for i := 0; i < len(tokenAssignment) && len(ownersSet) < int(replicationFactor); i++ {
        realIndex := (i + int(partition)) % len(tokenAssignment)

        if _, ok := ownersSet[tokenAssignment[realIndex]]; !ok {
            ownersSet[tokenAssignment[realIndex]] = true
            owners = append(owners, tokenAssignment[realIndex])
        }
    }

    if uint64(len(owners)) > 0 && replicationFactor > uint64(len(owners)) {
        originalOwnersList := owners

        for i := 0; uint64(i) < replicationFactor - uint64(len(originalOwnersList)); i++ {
            owners = append(owners, originalOwnersList[i % len(originalOwnersList)])
        }
    }

    return owners
}

func (ps *SimplePartitioningStrategy) Partition(key string, partitionCount uint64) uint64 {
    hash := NewHash([]byte(key)).High()
    
    if ps.shiftAmount == 0 {
        ps.CalculateShiftAmount(partitionCount)
    }

    return hash >> uint(ps.shiftAmount)
}

func (ps *SimplePartitioningStrategy) CalculateShiftAmount(partitionCount uint64) int {
    ps.lock.Lock()
    defer ps.lock.Unlock()

    if ps.shiftAmount != 0 {
        return ps.shiftAmount
    }

    ps.shiftAmount = 65

    for partitionCount > 0 {
        ps.shiftAmount--
        partitionCount = partitionCount >> 1
    }

    return ps.shiftAmount
}