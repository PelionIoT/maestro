package transfer_test

import (
    "context"
    "errors"
    "time"

    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/raft"
    . "github.com/armPelionEdge/devicedb/transfer"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("TransferProposer", func() {
    var defaultClusterController *ClusterController

    BeforeSuite(func() {
        defaultClusterController = &ClusterController{
            LocalNodeID: 1,
            State: ClusterState{
                ClusterSettings: ClusterSettings{
                    Partitions: 1024,
                    ReplicationFactor: 3,
                },
            },
        }
        defaultClusterController.State.Initialize()
        defaultClusterController.State.AddNode(NodeConfig{ Address: PeerAddress{ NodeID: 1 }, Capacity: 1, PartitionReplicas: map[uint64]map[uint64]bool{ } })
    })

    Describe("#QueueTransferProposal", func() {
        Context("When the queued transfer propsal is cancelled before the after channel is closed", func() {
            It("Should not attempt to make a proposal", func() {
                configController := NewMockConfigController(defaultClusterController)
                transferProposer := NewTransferProposer(configController)
                clusterCommandCalled := make(chan int)
                after := make(chan int)

                configController.onClusterCommand(func(ctx context.Context, commandBody interface{}) {
                    clusterCommandCalled <- 1
                })

                transferProposer.QueueTransferProposal(0, 0, after)
                Expect(transferProposer.PendingProposals(0)).Should(Equal(1))
                transferProposer.CancelTransferProposal(0, 0)
                Expect(transferProposer.PendingProposals(0)).Should(Equal(0))

                // Wait a second and make sure cluster command isn't ever called
                select {
                case <-clusterCommandCalled:
                    Fail("Cluster command should not have been invoked")
                case <-time.After(time.Second):
                }
            })

            It("Should not write anything to the returned result channel", func() {
                configController := NewMockConfigController(defaultClusterController)
                transferProposer := NewTransferProposer(configController)
                after := make(chan int)

                result := transferProposer.QueueTransferProposal(0, 0, after)
                Expect(transferProposer.PendingProposals(0)).Should(Equal(1))
                transferProposer.CancelTransferProposal(0, 0)
                Expect(transferProposer.PendingProposals(0)).Should(Equal(0))

                // Wait a second and make sure cluster command isn't ever called
                select {
                case <-result:
                    Fail("Cluster command should not have been invoked")
                case <-time.After(time.Second):
                }
            })
        })

        Context("When the after channel is closed", func() {
            It("Should make proposal to take the specified partition replica by making a call to ClusterCommand()", func() {
                configController := NewMockConfigController(defaultClusterController)
                transferProposer := NewTransferProposer(configController)
                clusterCommandCalled := make(chan int)
                after := make(chan int)

                configController.onClusterCommand(func(ctx context.Context, commandBody interface{}) {
                    defer GinkgoRecover()

                    c, ok := commandBody.(ClusterTakePartitionReplicaBody)

                    Expect(ok).Should(BeTrue())
                    Expect(c.NodeID).Should(Equal(uint64(1)))
                    Expect(c.Partition).Should(Equal(uint64(2)))
                    Expect(c.Replica).Should(Equal(uint64(3)))

                    clusterCommandCalled <- 1
                })

                transferProposer.QueueTransferProposal(2, 3, after)
                Expect(transferProposer.PendingProposals(2)).Should(Equal(1))
                close(after)

                // Wait a second and make sure cluster command is
                select {
                case <-clusterCommandCalled:
                case <-time.After(time.Second):
                    Fail("Test timed out")
                }
            })

            Context("And after the call to ClusterCommand() returns", func() {
                // This case may occur if, for example, this node gains ownership over partition 0 replica 1
                // And queues a transfer proposal for that partition replica. However, while waiting for the
                // proposal to occur this node loses ownership, triggering a call to CancelTransferProposal()
                // for that partition replica. In the meantime it regains ownership and another call to QueueTransferProposal()
                // occurs for the same partition replica.
                Context("If that partition replica transfer was cancelled with a corresponding call to CancelTransferProposal() while ClusterCommand() was running and then another identical transfer proposal was made", func() {
                    Specify("The cleanup for the first call to QueueTransferProposal should have no effect on the reported number of running transfers for this partition", func() {
                        configController := NewMockConfigController(defaultClusterController)
                        transferProposer := NewTransferProposer(configController)
                        clusterCommandCalled := make(chan int)
                        after := make(chan int)
                        after2 := make(chan int)

                        configController.onClusterCommand(func(ctx context.Context, commandBody interface{}) {
                            // This happens during the first Queued proposal's call to ClusterCommand
                            // It should cancel the currently running request without it knowing
                            // then queue another request for the same partition replica
                            Expect(transferProposer.PendingProposals(0)).Should(Equal(1))
                            transferProposer.CancelTransferProposal(0, 0)
                            Expect(transferProposer.PendingProposals(0)).Should(Equal(0))
                            transferProposer.QueueTransferProposal(0, 0, after2)
                            Expect(transferProposer.PendingProposals(0)).Should(Equal(1))
                            clusterCommandCalled <- 1
                        })

                        result := transferProposer.QueueTransferProposal(0, 0, after)
                        Expect(transferProposer.PendingProposals(0)).Should(Equal(1))

                        // Close after to trigger call to ClusterCommand
                        close(after)

                        // Wait a second and make sure cluster command isn't ever called
                        select {
                        case <-clusterCommandCalled:
                        case <-time.After(time.Second):
                            Fail("Test timed out")
                        }

                        // Wait for the first transfer proposal to return a result
                        <-result

                        // Ensure that after the first request finished the new proposal didn't get removed
                        Expect(transferProposer.PendingProposals(0)).Should(Equal(1))
                    })
                })

                Specify("It should write the return value of ClusterCommand() to the result channel", func() {
                    configController := NewMockConfigController(defaultClusterController)
                    transferProposer := NewTransferProposer(configController)
                    clusterCommandCalled := make(chan int)
                    after := make(chan int)

                    configController.SetDefaultClusterCommandResponse(errors.New("Some error"))
                    configController.onClusterCommand(func(ctx context.Context, commandBody interface{}) {
                        clusterCommandCalled <- 1
                    })

                    result := transferProposer.QueueTransferProposal(0, 0, after)
                    Expect(transferProposer.PendingProposals(0)).Should(Equal(1))

                    // Close after to trigger call to ClusterCommand
                    close(after)

                    // Wait a second and make sure cluster command isn't ever called
                    select {
                    case <-clusterCommandCalled:
                    case <-time.After(time.Second):
                        Fail("Test timed out")
                    }

                    // Wait for the first transfer proposal to return a result
                    Expect(<-result).Should(Equal(errors.New("Some error")))
                })

                Specify("It should remove itself from the pending proposals", func() {
                    configController := NewMockConfigController(defaultClusterController)
                    transferProposer := NewTransferProposer(configController)
                    clusterCommandCalled := make(chan int)
                    after := make(chan int)

                    configController.onClusterCommand(func(ctx context.Context, commandBody interface{}) {
                        clusterCommandCalled <- 1
                    })

                    result := transferProposer.QueueTransferProposal(0, 0, after)
                    Expect(transferProposer.PendingProposals(0)).Should(Equal(1))

                    // Close after to trigger call to ClusterCommand
                    close(after)

                    // Wait a second and make sure cluster command isn't ever called
                    select {
                    case <-clusterCommandCalled:
                    case <-time.After(time.Second):
                        Fail("Test timed out")
                    }

                    // Wait for the first transfer proposal to return a result
                    <-result

                    // Ensure that after the first request finished the new proposal didn't get removed
                    Expect(transferProposer.PendingProposals(0)).Should(Equal(0))
                })
            })
        })
    })

    Describe("#CancelTransferProposal", func() {
        Specify("It should cancel any currently queued partition replica transfer for the specified partition replica causing the reported number of queued transfers for a partition to decrease", func() {
            configController := NewMockConfigController(defaultClusterController)
            transferProposer := NewTransferProposer(configController)
            after := make(chan int)

            transferProposer.QueueTransferProposal(0, 0, after)
            transferProposer.QueueTransferProposal(0, 1, after)
            transferProposer.QueueTransferProposal(0, 2, after)
            Expect(transferProposer.PendingProposals(0)).Should(Equal(3))
            transferProposer.CancelTransferProposal(0, 0)
            Expect(transferProposer.PendingProposals(0)).Should(Equal(2))
        })
    })

    Describe("#CancelTransferProposals", func() {
        Specify("It should cancel all queued partition replica transfers for that partition causing the reported number of queued transfers for a partition to fall to zero", func() {
            configController := NewMockConfigController(defaultClusterController)
            transferProposer := NewTransferProposer(configController)
            after := make(chan int)

            transferProposer.QueueTransferProposal(0, 0, after)
            transferProposer.QueueTransferProposal(0, 1, after)
            transferProposer.QueueTransferProposal(0, 2, after)
            Expect(transferProposer.PendingProposals(0)).Should(Equal(3))
            transferProposer.CancelTransferProposals(0)
            Expect(transferProposer.PendingProposals(0)).Should(Equal(0))
        })
    })
})
