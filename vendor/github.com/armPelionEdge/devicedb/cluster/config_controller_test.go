package cluster_test

import (
    "fmt"
    "encoding/binary"
    "crypto/rand"
    "context"
    "time"
    
    . "github.com/armPelionEdge/devicedb/storage"
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/raft"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

func randomString() string {
    randomBytes := make([]byte, 16)
    rand.Read(randomBytes)
    
    high := binary.BigEndian.Uint64(randomBytes[:8])
    low := binary.BigEndian.Uint64(randomBytes[8:])
    
    return fmt.Sprintf("%05x%05x", high, low)
}

var _ = Describe("ConfigController", func() {
    Describe("Restarting a node", func() {
        Specify("A node should not start until it has replayed all previous log entries and has restored its config state to what it was before it was stopped", func() {
            // Create Cluster Controller
            clusterController := &ClusterController{
                LocalNodeID: 0x1,
                State: ClusterState{ 
                    Nodes: make(map[uint64]*NodeConfig),
                },
                PartitioningStrategy: &SimplePartitioningStrategy{ },
                LocalUpdates: make(chan []ClusterStateDelta),
            }

            addNodeBody, _ := EncodeClusterCommandBody(ClusterAddNodeBody{ NodeID: 0x1, NodeConfig: NodeConfig{ Address: PeerAddress{ NodeID: 0x1 }, Capacity: 1 } })
            addNodeContext, _ := EncodeClusterCommand(ClusterCommand{ Type: ClusterAddNode, Data: addNodeBody })

            // Create New Raft Node
            raftNode := NewRaftNode(&RaftNodeConfig{
                ID: 0x1,
                CreateClusterIfNotExist: true,
                Context: addNodeContext,
                Storage: NewRaftStorage(NewLevelDBStorageDriver("/tmp/testraftstore-" + randomString(), nil)),
                GetSnapshot: func() ([]byte, error) {
                    return clusterController.State.Snapshot()
                },
            })

            transport := NewTransportHub(0x1)
            // Pass raft node and cluster controller into new config controller
            configController := NewConfigController(raftNode, transport, clusterController)

            receivedDeltas := make(chan int)
            configController.OnLocalUpdates(func(deltas []ClusterStateDelta) {
                Expect(len(deltas)).Should(Equal(1))
                Expect(deltas[0].Type).Should(Equal(DeltaNodeAdd))
                Expect(deltas[0].Delta.(NodeAdd).NodeID).Should(Equal(uint64(0x1)))
                receivedDeltas <- 1
            })

            // Start Config Controller
            fmt.Println("Call Start() on config controller")
            Expect(configController.Start()).Should(BeNil())
            fmt.Println("Called Start() on config controller")
            // Wait for node to be added to single-node cluster
            <-receivedDeltas
            fmt.Println("Received deltas")

            ownedTokens := make(map[uint64]bool)

            configController.OnLocalUpdates(func(deltas []ClusterStateDelta) {
                for _, delta := range deltas {
                    if delta.Type == DeltaNodeGainToken {
                        Expect(delta.Delta.(NodeGainToken).NodeID).Should(Equal(uint64(0x1)))
                        ownedTokens[delta.Delta.(NodeGainToken).Token] = true
                    }
                }

                Expect(len(ownedTokens)).Should(Equal(1024))

                receivedDeltas <- 1
            })

            // Propose A Few Cluster Config Changes
            Expect(configController.ClusterCommand(context.TODO(), ClusterSetPartitionCountBody{ Partitions: 1024 })).Should(BeNil())
            Expect(configController.ClusterCommand(context.TODO(), ClusterSetReplicationFactorBody{ ReplicationFactor: 2 })).Should(BeNil())

            fmt.Println("Wait for  deltas 2")
            <-receivedDeltas
            fmt.Println("Received deltas 2")

            // Verify changes
            Expect(len(ownedTokens)).Should(Equal(1024))
            Expect(ownedTokens).Should(Equal(clusterController.State.Nodes[1].Tokens))

            // Stop old config controller
            configController.Stop()

            // Give it time to shut down
            <-time.After(time.Second)

            // Create a new cluster and config controller to ensure fresh state. Keep the same raft node
            // to ensure logs are restored from the same persistent storage file
            newClusterController := &ClusterController{
                LocalNodeID: 0x1,
                State: ClusterState{ 
                    Nodes: make(map[uint64]*NodeConfig),
                },
                PartitioningStrategy: &SimplePartitioningStrategy{ },
                LocalUpdates: nil,//make(chan ClusterStateDelta),
            }

            newTransport := NewTransportHub(0x1)
            newConfigController := NewConfigController(raftNode, newTransport, newClusterController)
            Expect(newConfigController.Start()).Should(BeNil())

            // Make Sure Current Configuration == Configuration Before Controller Was Stopped
            Expect(newClusterController.State).Should(Equal(clusterController.State))
        })

        Context("A node is restarting after being disconnected from the cluster for some time.", func() {
            Specify("If the state was compacted at other nodes then this node should receive a snapshot from them first", func() {
            })

            Specify("If the state was not yet compacted at other nodes then this node should receive a series of cluster commands  allowing its state to catch up", func() {
            })
        })
    })
})

// Node ip change?
// should generate node id based on a name?
