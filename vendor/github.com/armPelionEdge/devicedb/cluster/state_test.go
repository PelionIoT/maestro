package cluster_test

import (
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/raft"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("State", func() {
    Describe("ClusterState", func() {
        Describe("#AddNode", func() {
            It("should add the node to the nodes map if the node ID is non-zero and has not yet been added", func() {
                clusterState := &ClusterState{ }
                node := NodeConfig{ Capacity: 1, Address: PeerAddress{ NodeID: 1 } }

                Expect(len(clusterState.Nodes)).Should(Equal(0))
                clusterState.AddNode(node)
                Expect(clusterState.Nodes).Should(Equal(map[uint64]*NodeConfig{ 1: &node }))
            })

            It("should add no nodes to the nodes map if the node ID is zero", func() {
                clusterState := &ClusterState{ }
                node := NodeConfig{ Capacity: 1, Address: PeerAddress{ NodeID: 0 } }

                Expect(len(clusterState.Nodes)).Should(Equal(0))
                clusterState.AddNode(node)
                Expect(len(clusterState.Nodes)).Should(Equal(0))
            })

            It("should have no effect if the node has already been added", func() {
                clusterState := &ClusterState{ }
                nodeV1 := NodeConfig{ Capacity: 1, Address: PeerAddress{ NodeID: 1 } }
                nodeV2 := NodeConfig{ Capacity: 10, Address: PeerAddress{ NodeID: 1 } }

                Expect(len(clusterState.Nodes)).Should(Equal(0))
                clusterState.AddNode(nodeV1)
                Expect(clusterState.Nodes).Should(Equal(map[uint64]*NodeConfig{ 1: &nodeV1 }))
                clusterState.AddNode(nodeV2)
                Expect(clusterState.Nodes).Should(Equal(map[uint64]*NodeConfig{ 1: &nodeV1 }))
            })
        })

        Describe("#RemoveNode", func() {
            It("should have no effect if the node being removed does not exist in the map", func() {
                clusterState := &ClusterState{ }
                node := NodeConfig{ Capacity: 1, Address: PeerAddress{ NodeID: 1 } }

                // first try removing from an uninitialized map
                Expect(len(clusterState.Nodes)).Should(Equal(0))
                clusterState.RemoveNode(node.Address.NodeID)
                Expect(len(clusterState.Nodes)).Should(Equal(0))
                clusterState.AddNode(node)
                Expect(clusterState.Nodes).Should(Equal(map[uint64]*NodeConfig{ 1: &node }))
                // first try removing node 2 from an initialized map with one node in it whose ID = 1
                clusterState.RemoveNode(2)
                Expect(clusterState.Nodes).Should(Equal(map[uint64]*NodeConfig{ 1: &node }))
            })

            It("should release any partition replicas that were held by this node", func() {
                node := NodeConfig{ 
                    Capacity: 1, 
                    Address: PeerAddress{ NodeID: 1 },
                    PartitionReplicas: map[uint64]map[uint64]bool{
                        1: {
                            0: true,
                        },
                    },
                }

                clusterState := &ClusterState{ 
                    Nodes: map[uint64]*NodeConfig{ 
                        1: &node,
                    },
                    Partitions: [][]*PartitionReplica {
                        []*PartitionReplica{ },
                        []*PartitionReplica{ &PartitionReplica{ Partition: 1, Replica: 0, Holder: 1 } },
                    },
                }

                Expect(clusterState.Nodes).Should(Equal(map[uint64]*NodeConfig{ 1: &node }))
                clusterState.RemoveNode(node.Address.NodeID)
                Expect(len(clusterState.Nodes)).Should(Equal(0))
                Expect(*clusterState.Partitions[1][0]).Should(Equal(PartitionReplica{ Partition: 1, Replica: 0, Holder: 0 }))
            })

            It("should release any tokens that were owned by this node", func() {
                node := NodeConfig{ 
                    Capacity: 1, 
                    Address: PeerAddress{ NodeID: 1 },
                    Tokens: map[uint64]bool{
                        1: true,
                    },
                }

                clusterState := &ClusterState{ 
                    Nodes: map[uint64]*NodeConfig{
                        1: &node,
                    },
                    Tokens: []uint64{
                        0, // token 0 is not owned by anybody
                        1, // token 1 is owned by node 1
                    },
                }

                Expect(clusterState.Nodes).Should(Equal(map[uint64]*NodeConfig{ 1: &node }))
                clusterState.RemoveNode(node.Address.NodeID)
                Expect(len(clusterState.Nodes)).Should(Equal(0))
                Expect(clusterState.Tokens[1]).Should(Equal(uint64(0)))
            })

            It("should remove this node from the nodes map", func() {
                clusterState := &ClusterState{ }
                node := NodeConfig{ Capacity: 1, Address: PeerAddress{ NodeID: 1 } }

                Expect(len(clusterState.Nodes)).Should(Equal(0))
                clusterState.AddNode(node)
                Expect(clusterState.Nodes).Should(Equal(map[uint64]*NodeConfig{ 1: &node }))
                clusterState.RemoveNode(node.Address.NodeID)
            })
        })

        Describe("#AssignToken", func() {
            It("should return ENoSuchToken if the caller attempts to assign a node to a token that does not exist", func() {
                clusterState := &ClusterState{ }
                node := NodeConfig{ Capacity: 1, Address: PeerAddress{ NodeID: 1 } }

                Expect(len(clusterState.Nodes)).Should(Equal(0))
                clusterState.AddNode(node)
                // attempt to assign token 0 to node 1
                Expect(clusterState.AssignToken(1, 0)).Should(Equal(ENoSuchToken))
            })

            It("should return ENoSuchNode if the caller attempts to assign a node to a token and the node has not been added to the cluster", func() {
                clusterState := &ClusterState{ 
                    Tokens: []uint64{
                        0, // Make sure token 0 exists
                    },
                }

                Expect(len(clusterState.Nodes)).Should(Equal(0))
                // attempt to assign token 0 to node 1
                Expect(clusterState.AssignToken(1, 0)).Should(Equal(ENoSuchNode))
            })

            It("should assign the token to the node after releasing it from its current owner if it is owned by another node", func() {
                node1 := NodeConfig{ 
                    Capacity: 1, 
                    Address: PeerAddress{ NodeID: 1 },
                    Tokens: map[uint64]bool{
                        1: true,
                    },
                }
                node2 := NodeConfig{
                    Capacity: 1, 
                    Address: PeerAddress{ NodeID: 2 },
                    Tokens: map[uint64]bool{ },
                }

                clusterState := &ClusterState{ 
                    Nodes: map[uint64]*NodeConfig{
                        1: &node1,
                        2: &node2,
                    },
                    Tokens: []uint64{
                        0,
                        1, // Token 1 owned by node 1
                    },
                }

                // reassign token 1 to node 2
                Expect(clusterState.AssignToken(2, 1)).Should(BeNil())
                Expect(clusterState.Nodes[1].Tokens).Should(Equal(map[uint64]bool{ }))
                Expect(clusterState.Nodes[2].Tokens).Should(Equal(map[uint64]bool{ 1: true }))
                Expect(clusterState.Tokens[1]).Should(Equal(uint64(2)))
            })

            It("should assign the token to the node if no other node currently owns the token", func() {
                node1 := NodeConfig{
                    Capacity: 1, 
                    Address: PeerAddress{ NodeID: 1 },
                    Tokens: map[uint64]bool{ },
                }

                clusterState := &ClusterState{ 
                    Nodes: map[uint64]*NodeConfig{
                        1: &node1,
                    },
                    Tokens: []uint64{
                        0,
                        0, // Token 1 owned by nobody
                    },
                }

                // assign token 1 to node 1
                Expect(clusterState.Nodes[1].Tokens).Should(Equal(map[uint64]bool{ }))
                Expect(clusterState.Tokens[1]).Should(Equal(uint64(0)))
                Expect(clusterState.AssignToken(1, 1)).Should(BeNil())
                Expect(clusterState.Nodes[1].Tokens).Should(Equal(map[uint64]bool{ 1: true }))
                Expect(clusterState.Tokens[1]).Should(Equal(uint64(1)))
            })
        })

        Describe("#AssignPartitionReplica", func() {
            It("should return ENoSuchPartition if the specififed partition does not exist", func() {
                node := NodeConfig{ 
                    Capacity: 1, 
                    Address: PeerAddress{ NodeID: 1 },
                    PartitionReplicas: map[uint64]map[uint64]bool{
                        1: {
                            0: true,
                        },
                    },
                }

                clusterState := &ClusterState{ 
                    Nodes: map[uint64]*NodeConfig{ 
                        1: &node,
                    },
                    Partitions: [][]*PartitionReplica {
                        []*PartitionReplica{ },
                        []*PartitionReplica{ &PartitionReplica{ Partition: 1, Replica: 0, Holder: 1 } },
                    },
                }

                Expect(clusterState.AssignPartitionReplica(2, 0, 1)).Should(Equal(ENoSuchPartition))
            })

            It("should return ENoSuchReplica if the specified replica does not exist", func() {
                node := NodeConfig{ 
                    Capacity: 1, 
                    Address: PeerAddress{ NodeID: 1 },
                    PartitionReplicas: map[uint64]map[uint64]bool{
                        1: {
                            0: true,
                        },
                    },
                }

                clusterState := &ClusterState{ 
                    Nodes: map[uint64]*NodeConfig{ 
                        1: &node,
                    },
                    Partitions: [][]*PartitionReplica {
                        []*PartitionReplica{ },
                        []*PartitionReplica{ &PartitionReplica{ Partition: 1, Replica: 0, Holder: 1 } },
                    },
                }

                Expect(clusterState.AssignPartitionReplica(1, 1, 1)).Should(Equal(ENoSuchReplica))
            })

            It("should return ENoSuchNode if the specified node does not exist", func() {
                node := NodeConfig{ 
                    Capacity: 1, 
                    Address: PeerAddress{ NodeID: 1 },
                    PartitionReplicas: map[uint64]map[uint64]bool{
                        1: {
                            0: true,
                        },
                    },
                }

                clusterState := &ClusterState{ 
                    Nodes: map[uint64]*NodeConfig{ 
                        1: &node,
                    },
                    Partitions: [][]*PartitionReplica {
                        []*PartitionReplica{ },
                        []*PartitionReplica{ &PartitionReplica{ Partition: 1, Replica: 0, Holder: 1 } },
                    },
                }

                Expect(clusterState.AssignPartitionReplica(1, 0, 2)).Should(Equal(ENoSuchNode))
            })

            It("should assign the partition replica to the specified node if it is not yet assigned", func() {
                node := NodeConfig{ 
                    Capacity: 1, 
                    Address: PeerAddress{ NodeID: 1 },
                    PartitionReplicas: map[uint64]map[uint64]bool{ },
                }

                clusterState := &ClusterState{ 
                    Nodes: map[uint64]*NodeConfig{ 
                        1: &node,
                    },
                    Partitions: [][]*PartitionReplica {
                        []*PartitionReplica{ },
                        []*PartitionReplica{ &PartitionReplica{ Partition: 1, Replica: 0, Holder: 0 } },
                    },
                }

                Expect(clusterState.AssignPartitionReplica(1, 0, 1)).Should(BeNil())
                Expect(clusterState.Nodes[1].PartitionReplicas).Should(Equal(map[uint64]map[uint64]bool{ 1: { 0: true } }))
                Expect(clusterState.Partitions[1][0].Holder).Should(Equal(uint64(1)))
            })

            It("should assign the partition replica to the specified node after removing it from its current node if it is already assigned", func() {
                node1 := NodeConfig{ 
                    Capacity: 1, 
                    Address: PeerAddress{ NodeID: 1 },
                    PartitionReplicas: map[uint64]map[uint64]bool{ 1: { 0: true } },
                }
                node2 := NodeConfig{ 
                    Capacity: 1, 
                    Address: PeerAddress{ NodeID: 2 },
                    PartitionReplicas: map[uint64]map[uint64]bool{ },
                }

                clusterState := &ClusterState{ 
                    Nodes: map[uint64]*NodeConfig{ 
                        1: &node1,
                        2: &node2,
                    },
                    Partitions: [][]*PartitionReplica {
                        []*PartitionReplica{ },
                        []*PartitionReplica{ &PartitionReplica{ Partition: 1, Replica: 0, Holder: 1 } },
                    },
                }

                Expect(clusterState.AssignPartitionReplica(1, 0, 2)).Should(BeNil())
                Expect(clusterState.Nodes[1].PartitionReplicas).Should(Equal(map[uint64]map[uint64]bool{ }))
                Expect(clusterState.Nodes[2].PartitionReplicas).Should(Equal(map[uint64]map[uint64]bool{ 1: { 0: true } }))
                Expect(clusterState.Partitions[1][0].Holder).Should(Equal(uint64(2)))
            })
        })

        Describe("#Initialize", func() {
            It("should do nothing if cluster settings are not initialized", func() {
                clusterState := &ClusterState{ 
                    ClusterSettings: ClusterSettings{
                        Partitions: 0,
                        ReplicationFactor: 0,
                    },
                }

                Expect(clusterState.Tokens).Should(BeNil())
                Expect(clusterState.Partitions).Should(BeNil())
                clusterState.Initialize()
                Expect(clusterState.Tokens).Should(BeNil())
                Expect(clusterState.Partitions).Should(BeNil())
            })

            It("should initialize tokens to an array whose length is equal to the number of partitions specified in the cluster settings and that is initialized to zeroes", func() {
                clusterState := &ClusterState{ 
                    ClusterSettings: ClusterSettings{
                        Partitions: 10,
                        ReplicationFactor: 3,
                    },
                }

                Expect(clusterState.Tokens).Should(BeNil())
                clusterState.Initialize()
                Expect(len(clusterState.Tokens)).Should(Equal(10))

                for _, owner := range clusterState.Tokens {
                    Expect(owner).Should(Equal(uint64(0)))
                }
            })

            It("should initialize partitions to a PxR array of PartitionReplica objects where P = the number of partitions and R = the replication factor", func() {
                clusterState := &ClusterState{
                    ClusterSettings: ClusterSettings{
                        Partitions: 10,
                        ReplicationFactor: 3,
                    },
                }

                Expect(clusterState.Partitions).Should(BeNil())
                clusterState.Initialize()
                Expect(len(clusterState.Partitions)).Should(Equal(10))

                for partition, replicas := range clusterState.Partitions {
                    Expect(len(replicas)).Should(Equal(3))

                    for replica, partitionReplica := range replicas {
                        Expect(*partitionReplica).Should(Equal(PartitionReplica{ Partition: uint64(partition), Replica: uint64(replica) }))
                    }
                }
            })
        })

        Describe("#Snapshot + #Recover", func() {
            It("Snapshot should produce a byte array that when parsed by Recover produces a copy of the cluster state", func() {
                node1 := NodeConfig{ 
                    Capacity: 1, 
                    Address: PeerAddress{ NodeID: 1 },
                    PartitionReplicas: map[uint64]map[uint64]bool{ 1: { 0: true } },
                }
                node2 := NodeConfig{ 
                    Capacity: 1, 
                    Address: PeerAddress{ NodeID: 2 },
                    PartitionReplicas: map[uint64]map[uint64]bool{ },
                }
                clusterState := &ClusterState{ 
                    Nodes: map[uint64]*NodeConfig{ 
                        1: &node1,
                        2: &node2,
                    },
                    Partitions: [][]*PartitionReplica {
                        []*PartitionReplica{ },
                        []*PartitionReplica{ &PartitionReplica{ Partition: 1, Replica: 0, Holder: 1 } },
                    },
                }

                snap, err := clusterState.Snapshot()

                Expect(err).Should(BeNil())

                var clusterStateCopy ClusterState

                Expect(clusterStateCopy.Recover(snap)).Should(BeNil())
                Expect(clusterStateCopy).Should(Equal(*clusterState))
            })
        })
    })

    Describe("ClusterSettings", func() {
        Describe("#AreInitialized", func() {
            It("should return false if ReplicationFactor is zero", func() {
                clusterSettings := &ClusterSettings{ ReplicationFactor: 0, Partitions: 1 }
                Expect(clusterSettings.AreInitialized()).Should(BeFalse())
                clusterSettings = &ClusterSettings{ ReplicationFactor: 0, Partitions: 0 }
                Expect(clusterSettings.AreInitialized()).Should(BeFalse())
            })

            It("should return false if Partitions is zero", func() {
                clusterSettings := &ClusterSettings{ ReplicationFactor: 1, Partitions: 0 }
                Expect(clusterSettings.AreInitialized()).Should(BeFalse())
                clusterSettings = &ClusterSettings{ ReplicationFactor: 0, Partitions: 0 }
                Expect(clusterSettings.AreInitialized()).Should(BeFalse())
            })

            It("should return true if ReplicationFactor and Partitions is non-zero", func() {
                clusterSettings := &ClusterSettings{ ReplicationFactor: 1, Partitions: 1 }
                Expect(clusterSettings.AreInitialized()).Should(BeTrue())
            })
        })
    })
})
