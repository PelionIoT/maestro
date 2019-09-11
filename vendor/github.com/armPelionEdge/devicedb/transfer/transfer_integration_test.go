package transfer_test

import (
    "context"
    "io"
    "net/http"
    "time"

    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/raft"
    . "github.com/armPelionEdge/devicedb/transfer"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "github.com/gorilla/mux"
)

var _ = Describe("TransferIntegrationTest", func() {
    // Test the full pipeline starting out with a partition iterator, transforming it into a read stream and finally
    // re-encoding it into a series of partition chunks
    Describe("Partition Chunks -> Outgoing transfer -> Transfer Encoder -> Incoming Transfer -> Partition Chunks", func() {
        Context("Without a network layer in between", func() {
            Specify("Should encode entries from the partition into a byte stream and decode it on the other end into the same series of entries", func () {
                partition := NewMockPartition(0, 0)
                iterator := partition.MockIterator()
                dummySibling := NewSibling(NewDVV(NewDot("A", 0), map[string]uint64{ }), []byte{ }, 0)
                dummySiblingSet := NewSiblingSet(map[*Sibling]bool{ dummySibling: true })
                iterator.AppendNextState(true, "site1", "default", "a", dummySiblingSet, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "b", dummySiblingSet, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "c", dummySiblingSet, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "d", dummySiblingSet, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "e", dummySiblingSet, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "f", dummySiblingSet, Hash{}, nil)
                iterator.AppendNextState(false, "", "", "", nil, Hash{}, nil)

                outgoingTransfer := NewOutgoingTransfer(partition, 2)
                transferEncoder := NewTransferEncoder(outgoingTransfer)
                r, _ := transferEncoder.Encode()
                transferDecoder := NewTransferDecoder(r)
                incomingTransfer, _ := transferDecoder.Decode()

                nextChunk, err := incomingTransfer.NextChunk()
                Expect(len(nextChunk.Entries)).Should(Equal(2))
                Expect(nextChunk.Index).Should(Equal(uint64(1)))
                Expect(nextChunk.Entries[0].Site).Should(Equal("site1"))
                Expect(nextChunk.Entries[0].Bucket).Should(Equal("default"))
                Expect(nextChunk.Entries[0].Key).Should(Equal("a"))
                Expect(nextChunk.Entries[1].Site).Should(Equal("site1"))
                Expect(nextChunk.Entries[1].Bucket).Should(Equal("default"))
                Expect(nextChunk.Entries[1].Key).Should(Equal("b"))
                Expect(err).Should(BeNil())
                nextChunk, err = incomingTransfer.NextChunk()
                Expect(len(nextChunk.Entries)).Should(Equal(2))
                Expect(nextChunk.Index).Should(Equal(uint64(2)))
                Expect(nextChunk.Entries[0].Site).Should(Equal("site1"))
                Expect(nextChunk.Entries[0].Bucket).Should(Equal("default"))
                Expect(nextChunk.Entries[0].Key).Should(Equal("c"))
                Expect(nextChunk.Entries[1].Site).Should(Equal("site1"))
                Expect(nextChunk.Entries[1].Bucket).Should(Equal("default"))
                Expect(nextChunk.Entries[1].Key).Should(Equal("d"))
                Expect(err).Should(BeNil())
                nextChunk, err = incomingTransfer.NextChunk()
                Expect(len(nextChunk.Entries)).Should(Equal(2))
                Expect(nextChunk.Index).Should(Equal(uint64(3)))
                Expect(nextChunk.Entries[0].Site).Should(Equal("site1"))
                Expect(nextChunk.Entries[0].Bucket).Should(Equal("default"))
                Expect(nextChunk.Entries[0].Key).Should(Equal("e"))
                Expect(nextChunk.Entries[1].Site).Should(Equal("site1"))
                Expect(nextChunk.Entries[1].Bucket).Should(Equal("default"))
                Expect(nextChunk.Entries[1].Key).Should(Equal("f"))
                Expect(err).Should(BeNil())
                nextChunk, err = incomingTransfer.NextChunk()
                Expect(nextChunk.IsEmpty()).Should(BeTrue())
                Expect(err).Should(Equal(io.EOF))
            })
        })

        Context("With a network layer in between", func() {
            Specify("Should encode entries from the partition into a byte stream and decode it on the other end into the same series of entries", func () {
                partition := NewMockPartition(0, 0)
                iterator := partition.MockIterator()
                iterator.AppendNextState(true, "site1", "default", "a", nil, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "b", nil, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "c", nil, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "d", nil, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "e", nil, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "f", nil, Hash{}, nil)
                iterator.AppendNextState(false, "", "", "", nil, Hash{}, nil)

                outgoingTransfer := NewOutgoingTransfer(partition, 2)
                transferEncoder := NewTransferEncoder(outgoingTransfer)
                encodedStream, _ := transferEncoder.Encode()

                testServer := NewHTTPTestServer(9090, &StringResponseHandler{ str: encodedStream })
                clusterController := &ClusterController{
                    LocalNodeID: 1,
                    State: ClusterState{
                        Nodes: map[uint64]*NodeConfig{
                            1: &NodeConfig{
                                Address: PeerAddress{
                                    NodeID: 1,
                                    Host: "localhost",
                                    Port: 9090,
                                },
                            },
                        },
                    },
                }
                configController := NewConfigController(nil, nil, clusterController)
                httpClient := &http.Client{}
                transferTransport := NewHTTPTransferTransport(configController, httpClient)
                testServer.Start()
                // give it enough time to fully start
                <-time.After(time.Second)
                r, cancel, err := transferTransport.Get(1, 0)

                transferDecoder := NewTransferDecoder(r)
                incomingTransfer, _ := transferDecoder.Decode()

                nextChunk, err := incomingTransfer.NextChunk()
                Expect(nextChunk).Should(Equal(PartitionChunk{
                    Index: 1,
                    Checksum: Hash{ },
                    Entries: []Entry{
                        Entry{
                            Site: "site1",
                            Bucket: "default",
                            Key: "a",
                            Value: nil,
                        },
                        Entry{
                            Site: "site1",
                            Bucket: "default",
                            Key: "b",
                            Value: nil,
                        },
                    },
                }))
                Expect(err).Should(BeNil())
                nextChunk, err = incomingTransfer.NextChunk()
                Expect(nextChunk).Should(Equal(PartitionChunk{
                    Index: 2,
                    Checksum: Hash{ },
                    Entries: []Entry{
                        Entry{
                            Site: "site1",
                            Bucket: "default",
                            Key: "c",
                            Value: nil,
                        },
                        Entry{
                            Site: "site1",
                            Bucket: "default",
                            Key: "d",
                            Value: nil,
                        },
                    },
                }))
                Expect(err).Should(BeNil())
                nextChunk, err = incomingTransfer.NextChunk()
                Expect(nextChunk).Should(Equal(PartitionChunk{
                    Index: 3,
                    Checksum: Hash{ },
                    Entries: []Entry{
                        Entry{
                            Site: "site1",
                            Bucket: "default",
                            Key: "e",
                            Value: nil,
                        },
                        Entry{
                            Site: "site1",
                            Bucket: "default",
                            Key: "f",
                            Value: nil,
                        },
                    },
                }))
                Expect(err).Should(BeNil())
                nextChunk, err = incomingTransfer.NextChunk()
                Expect(nextChunk.IsEmpty()).Should(BeTrue())
                Expect(err).Should(Equal(io.EOF))

                cancel()
                testServer.Stop()
            })
        })
    })

    // Test a partition transfer from download to proposal
    // Ensuring correct ordering and interaction between downloader and
    // Transfer proposer 
    Describe("Downloader Process", func() {
        Specify("The downloader should request a partition transfer from the current holder of the partition replica and write it do a partition", func() {
            partition := NewMockPartition(0, 0)
            iterator := partition.MockIterator()
            iterator.AppendNextState(true, "site1", "default", "a", nil, Hash{}, nil)
            iterator.AppendNextState(true, "site1", "default", "b", nil, Hash{}, nil)
            iterator.AppendNextState(true, "site1", "default", "c", nil, Hash{}, nil)
            iterator.AppendNextState(true, "site1", "default", "d", nil, Hash{}, nil)
            iterator.AppendNextState(true, "site1", "default", "e", nil, Hash{}, nil)
            iterator.AppendNextState(true, "site1", "default", "f", nil, Hash{}, nil)
            iterator.AppendNextState(false, "", "", "", nil, Hash{}, nil)

            outgoingTransfer := NewOutgoingTransfer(partition, 2)
            transferEncoder := NewTransferEncoder(outgoingTransfer)
            encodedStream, _ := transferEncoder.Encode()

            testServer := NewHTTPTestServer(7070, &StringResponseHandler{ str: encodedStream })
            clusterController := &ClusterController{
                LocalNodeID: 1,
                State: ClusterState{
                    ClusterSettings: ClusterSettings{
                        Partitions: 1024,
                        ReplicationFactor: 3,
                    },
                },
            }
            clusterController.State.Initialize()
            clusterController.State.AddNode(NodeConfig{ Address: PeerAddress{ NodeID: 1, Host: "localhost", Port: 6060 }, Capacity: 1, PartitionReplicas: map[uint64]map[uint64]bool{ } })
            clusterController.State.AddNode(NodeConfig{ Address: PeerAddress{ NodeID: 2, Host: "localhost", Port: 7070 }, Capacity: 1, PartitionReplicas: map[uint64]map[uint64]bool{ } })
            clusterController.State.AssignPartitionReplica(0, 0, 2)
            configController := NewConfigController(nil, nil, clusterController)
            httpClient := &http.Client{}
            transferTransport := NewHTTPTransferTransport(configController, httpClient)
            partnerStrategy := NewRandomTransferPartnerStrategy(configController)
            transferFactory := &TransferFactory{ }
            localDefaultBucket := NewMockBucket("default")
            localDefaultBucket.SetDefaultMergeResponse(nil)
            bucketList := NewBucketList()
            bucketList.AddBucket(localDefaultBucket)
            localSite1 := NewMockSite(bucketList)
            localSitePool := NewMockSitePool()
            localSitePool.AppendNextAcquireResponse(localSite1)
            localSitePool.AppendNextAcquireResponse(localSite1)
            localSitePool.AppendNextAcquireResponse(localSite1)
            localSitePool.AppendNextAcquireResponse(localSite1)
            localSitePool.AppendNextAcquireResponse(localSite1)
            localSitePool.AppendNextAcquireResponse(localSite1)
            localPartition := NewMockPartition(0, 0)
            localPartition.SetSites(localSitePool)
            partitionPool := NewMockPartitionPool()
            partitionPool.Add(localPartition)
            nextMerge := make(chan map[string]*SiblingSet)
            downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, partitionPool)
            localDefaultBucket.onMerge(func(siblingSets map[string]*SiblingSet) {
                nextMerge <- siblingSets
            })
            testServer.Start()
            defer testServer.Stop()

            // give it enough time to fully start
            <-time.After(time.Second)

            keys := []string{ "a", "b", "c", "d", "e", "f" }
            done := downloader.Download(0)
           
            for _, key := range keys {
                select {
                case n := <-nextMerge:
                    Expect(len(n)).Should(Equal(1))
                    _, ok := n[key]
                    Expect(ok).Should(BeTrue())
                case <-time.After(time.Second):
                    Fail("Test timed out")
                }
            }

            select {
            case <-done:
            case <-time.After(time.Second):
                Fail("Test timed out")
            }
        })
    })

    // Test using an HTTP server with the handler defined in HTTPTransferAgent and sending
    // a partition across the network to a downloader. Ensure that keys from sites are filtered
    // out of the result if the site does not exist at the current holder of the partition
    Describe("Full Transfer Process", func() {
        Specify("", func() {
            requesterClusterController := &ClusterController{
                LocalNodeID: 1,
                State: ClusterState{
                    ClusterSettings: ClusterSettings{
                        Partitions: 1024,
                        ReplicationFactor: 3,
                    },
                },
            }
            requesterClusterController.State.Initialize()
            requesterClusterController.State.AddNode(NodeConfig{ Address: PeerAddress{ NodeID: 1, Host: "localhost", Port: 5050 }, Capacity: 1, PartitionReplicas: map[uint64]map[uint64]bool{ } })
            requesterClusterController.State.AddNode(NodeConfig{ Address: PeerAddress{ NodeID: 2, Host: "localhost", Port: 6060 }, Capacity: 1, PartitionReplicas: map[uint64]map[uint64]bool{ } })
            requesterClusterController.State.AssignPartitionReplica(0, 0, 2)
            requesterClusterController.State.AddSite("site1")
            requesterClusterController.State.AddSite("site2")

            holderClusterController := &ClusterController{
                LocalNodeID: 2,
                State: ClusterState{
                    ClusterSettings: ClusterSettings{
                        Partitions: 1024,
                        ReplicationFactor: 3,
                    },
                },
            }
            holderClusterController.State.Initialize()
            holderClusterController.State.AddNode(NodeConfig{ Address: PeerAddress{ NodeID: 1, Host: "localhost", Port: 5050 }, Capacity: 1, PartitionReplicas: map[uint64]map[uint64]bool{ } })
            holderClusterController.State.AddNode(NodeConfig{ Address: PeerAddress{ NodeID: 2, Host: "localhost", Port: 6060 }, Capacity: 1, PartitionReplicas: map[uint64]map[uint64]bool{ } })
            holderClusterController.State.AssignPartitionReplica(0, 0, 2)
            holderClusterController.State.AddSite("site1")
            // ***Holder knows site2 was deleted so its entries should be excluded from the transfer

            requesterConfigController := NewMockConfigController(requesterClusterController)
            holderConfigController := NewConfigController(nil, nil, holderClusterController)

            clusterProposalMade := make(chan int)
            requesterConfigController.onClusterCommand(func(ctx context.Context, commandBody interface{}) {
                c, ok := commandBody.(ClusterTakePartitionReplicaBody)

                Expect(ok).Should(BeTrue())
                Expect(c.NodeID).Should(Equal(uint64(1)))
                Expect(c.Partition).Should(Equal(uint64(0)))
                Expect(c.Replica).Should(Equal(uint64(0)))

                clusterProposalMade <- 1
            })

            transferProposer := NewTransferProposer(requesterConfigController)
            httpClient := &http.Client{}
            transferTransport := NewHTTPTransferTransport(requesterConfigController, httpClient)
            partnerStrategy := NewRandomTransferPartnerStrategy(requesterConfigController)
            transferFactory := &TransferFactory{ }
            localDefaultBucket := NewMockBucket("default")
            nextMerge := make(chan map[string]*SiblingSet)
            localDefaultBucket.onMerge(func(siblingSets map[string]*SiblingSet) {
                nextMerge <- siblingSets
            })
            localDefaultBucket.SetDefaultMergeResponse(nil)
            bucketList := NewBucketList()
            bucketList.AddBucket(localDefaultBucket)
            localSite1 := NewMockSite(bucketList)
            localSitePool := NewMockSitePool()
            localSitePool.AppendNextAcquireResponse(localSite1)
            localSitePool.AppendNextAcquireResponse(localSite1)
            localSitePool.AppendNextAcquireResponse(localSite1)
            localSitePool.AppendNextAcquireResponse(localSite1)
            localSitePool.AppendNextAcquireResponse(localSite1)
            localSitePool.AppendNextAcquireResponse(localSite1)
            localPartition := NewMockPartition(0, 0)
            localPartition.SetSites(localSitePool)
            partitionPool := NewMockPartitionPool()
            partitionPool.Add(localPartition)
            downloader := NewDownloader(requesterConfigController, transferTransport, partnerStrategy, transferFactory, partitionPool)

            holderPartition := NewMockPartition(0, 0)
            iterator := holderPartition.MockIterator()
            iterator.AppendNextState(true, "site2", "default", "h", nil, Hash{}, nil)
            iterator.AppendNextState(true, "site1", "default", "a", nil, Hash{}, nil)
            iterator.AppendNextState(true, "site1", "default", "b", nil, Hash{}, nil)
            iterator.AppendNextState(true, "site1", "default", "c", nil, Hash{}, nil)
            iterator.AppendNextState(true, "site1", "default", "d", nil, Hash{}, nil)
            iterator.AppendNextState(true, "site1", "default", "e", nil, Hash{}, nil)
            iterator.AppendNextState(true, "site1", "default", "f", nil, Hash{}, nil)
            iterator.AppendNextState(true, "site2", "default", "g", nil, Hash{}, nil)
            iterator.AppendNextState(false, "", "", "", nil, Hash{}, nil)
            holderPartitionPool := NewMockPartitionPool()
            holderPartitionPool.Add(holderPartition)

            transferAgentRequester := NewHTTPTransferAgent(requesterConfigController, transferProposer, downloader, transferFactory, partitionPool)
            transferAgentHolder := NewHTTPTransferAgent(holderConfigController, nil, nil, transferFactory, holderPartitionPool)
            transferAgentHolder.EnableOutgoingTransfers(0)

            router := mux.NewRouter()
            transferAgentHolder.Attach(router)
            testServer := NewHTTPTestServer(6060, router)
            
            testServer.Start()
            defer testServer.Stop()

            // give it enough time to fully start
            <-time.After(time.Second)

            transferAgentRequester.StartTransfer(0, 0)

            // Make sure all unfiltered keys are written in the correct order
            keys := []string{ "a", "b", "c", "d", "e", "f" }
           
            for _, key := range keys {
                select {
                case n := <-nextMerge:
                    Expect(len(n)).Should(Equal(1))
                    _, ok := n[key]
                    Expect(ok).Should(BeTrue())
                case <-time.After(time.Second):
                    Fail("Test timed out")
                }
            }

            // Make sure after download is complete the transfer proposal is made
            select {
            case <-clusterProposalMade:
            case <-time.After(time.Second):
                Fail("Test timed out")
            }
        })
    })
})
