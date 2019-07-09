package transfer_test

import (
    "errors"
    "io"
    "time"

    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/raft"
    . "github.com/armPelionEdge/devicedb/transfer"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("Downloader", func() {
    Describe("#Download", func() {
        Context("When this node is already a holder for this partition", func() {
            var configController *ConfigController
            var downloader *Downloader

            BeforeEach(func() {
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
                clusterController.State.AddNode(NodeConfig{ Address: PeerAddress{ NodeID: 1 }, Capacity: 1, PartitionReplicas: map[uint64]map[uint64]bool{ } })
                clusterController.State.AssignPartitionReplica(0, 0, 1)
                configController = NewConfigController(nil, nil, clusterController)

                downloader = NewDownloader(configController, nil, nil, nil, nil)
            })

            It("Should return a closed done channel and make no download attempts", func() {
                done := downloader.Download(0)

                _, ok := <-done

                Expect(ok).Should(BeFalse())
                Expect(downloader.IsDownloading(0)).Should(BeFalse())
            })
        })

        Context("When there are no nodes which hold replicas for the specified partition", func() {
            var configController *ConfigController
            var downloader *Downloader

            BeforeEach(func() {
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
                clusterController.State.AddNode(NodeConfig{ Address: PeerAddress{ NodeID: 1 }, Capacity: 1, PartitionReplicas: map[uint64]map[uint64]bool{ } })
                configController = NewConfigController(nil, nil, clusterController)

                // initialize it with an empty mock transfer partner strategy so it will return 0
                // when ChooseTransferPartner() is called indicating there are no holders for this
                // partition
                downloader = NewDownloader(configController, nil, NewMockTransferPartnerStrategy(), nil, nil)
            })

            It("Should return a closed done channel and make no download attempts", func() {
                done := downloader.Download(0)

                _, ok := <-done

                Expect(ok).Should(BeFalse())
                Expect(downloader.IsDownloading(0)).Should(BeFalse())
            })
        })

        Context("When there is at least one node which holds a replica for the specified partition", func() {
            var configController *ConfigController

            BeforeEach(func() {
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
                clusterController.State.AddNode(NodeConfig{ Address: PeerAddress{ NodeID: 1 }, Capacity: 1, PartitionReplicas: map[uint64]map[uint64]bool{ } })
                clusterController.State.AddNode(NodeConfig{ Address: PeerAddress{ NodeID: 2 }, Capacity: 1, PartitionReplicas: map[uint64]map[uint64]bool{ } })
                clusterController.State.AddNode(NodeConfig{ Address: PeerAddress{ NodeID: 3 }, Capacity: 1, PartitionReplicas: map[uint64]map[uint64]bool{ } })
                // Set up cluster so nodes 2 and 3 hold replicas of partition 0 but node 1 doesn't hold any replicas
                clusterController.State.AssignPartitionReplica(0, 0, 2) // assign patition 0, replica 0 to node 2
                clusterController.State.AssignPartitionReplica(0, 1, 3) // assign patition 0, replica 1 to node 3
                configController = NewConfigController(nil, nil, clusterController)
            })

            Context("And there is no download currently in progress for this partition", func() {
                It("Should return a new done channel", func() {
                    // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                    downloader := NewDownloader(configController, NewMockTransferTransport(), NewMockTransferPartnerStrategy().AppendNextTransferPartner(2), nil, nil)
                    done := downloader.Download(0)

                    select {
                    case <-done:
                        Fail("Done channel should be open")
                    default:
                    }

                    downloader.CancelDownload(0)
                })

                It("Should choose a transfer partner from the list of nodes which hold a replica of this partition", func() {
                    chooseTransferPartnerCalled := make(chan int)
                    partnerStrategy := NewMockTransferPartnerStrategy()
                    partnerStrategy.AppendNextTransferPartner(2)
                    partnerStrategy.onChooseTransferPartner(func(partition uint64) {
                        defer GinkgoRecover()
                        Expect(partition).Should(Equal(uint64(0)))
                        chooseTransferPartnerCalled <- 1
                    })

                    // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                    downloader := NewDownloader(configController, NewMockTransferTransport(), partnerStrategy, nil, nil)
                    downloader.Download(0)

                    select {
                    case <-chooseTransferPartnerCalled:
                    case <-time.After(time.Second):
                        Fail("Test timed out")
                    }

                    downloader.CancelDownload(0)
                })

                It("Should then initiate a download from its chosen transfer partner", func() {
                    getCalled := make(chan int)
                    partnerStrategy := NewMockTransferPartnerStrategy()
                    partnerStrategy.AppendNextTransferPartner(2)
                    transferTransport := NewMockTransferTransport()
                    transferTransport.onGet(func(node uint64, partition uint64) {
                        defer GinkgoRecover()
                        Expect(node).Should(Equal(uint64(2)))
                        Expect(partition).Should(Equal(uint64(0)))
                        getCalled <- 1
                    })

                    // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                    downloader := NewDownloader(configController, transferTransport, partnerStrategy, nil, nil)
                    downloader.Download(0)

                    select {
                    case <-getCalled:
                    case <-time.After(time.Second):
                        Fail("Test timed out")
                    }

                    downloader.CancelDownload(0)
                })

                Context("And if there is a problem initiating a download from its chosen transfer partner", func() {
                    It("Should wait for some period of time before again choosing a transfer partner and attempting to restart the download", func() {
                        chooseTransferPartnerCalled := make(chan int)
                        getCalled := make(chan int)
                        partnerStrategy := NewMockTransferPartnerStrategy()
                        partnerStrategy.AppendNextTransferPartner(2)
                        partnerStrategy.AppendNextTransferPartner(2)
                        partnerStrategy.onChooseTransferPartner(func(partition uint64) {
                            defer GinkgoRecover()
                            Expect(partition).Should(Equal(uint64(0)))
                            chooseTransferPartnerCalled <- 1
                        })
                        // An empty mock transfer transport will always return an error when Get is called so no need to initialize with responses
                        transferTransport := NewMockTransferTransport()
                        transferTransport.onGet(func(node uint64, partition uint64) {
                            defer GinkgoRecover()
                            Expect(node).Should(Equal(uint64(2)))
                            Expect(partition).Should(Equal(uint64(0)))
                            getCalled <- 1
                        })

                        // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                        downloader := NewDownloader(configController, transferTransport, partnerStrategy, nil, nil)
                        downloader.Download(0)

                        select {
                        case <-chooseTransferPartnerCalled:
                        case <-time.After(time.Second):
                            Fail("Test timed out")
                        }

                        select {
                        case <-getCalled:
                        case <-time.After(time.Second):
                            Fail("Test timed out")
                        }

                        select {
                        case <-chooseTransferPartnerCalled:
                        // Wait period should be no more than 2 seconds
                        case <-time.After(time.Second * 2):
                            Fail("Test timed out")
                        }

                        select {
                        case <-getCalled:
                        case <-time.After(time.Second):
                            Fail("Test timed out")
                        }

                        downloader.CancelDownload(0)
                    })
                })

                Context("And if the download is started successfully", func() {
                    It("Should decode the read stream as an incoming transfer", func() {
                        createIncomingTransferCalled := make(chan int)
                        partnerStrategy := NewMockTransferPartnerStrategy()
                        partnerStrategy.AppendNextTransferPartner(2)
                        transferTransport := NewMockTransferTransport()
                        infiniteReader := NewInfiniteReader()
                        transferTransport.AppendNextGetResponse(infiniteReader, func() { }, nil)
                        incomingTransfer := NewMockPartitionTransfer()
                        incomingTransfer.AppendNextChunkResponse(PartitionChunk{ }, io.EOF)
                        transferFactory := NewMockPartitionTransferFactory()
                        transferFactory.AppendNextIncomingTransfer(incomingTransfer)
                        transferFactory.onCreateIncomingTransfer(func(reader io.Reader) {
                            defer GinkgoRecover()
                            Expect(reader.(*InfiniteReader)).Should(Equal(infiniteReader))
                            createIncomingTransferCalled <- 1
                        })

                        // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                        downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, nil)
                        downloader.Download(0)

                        select {
                        case <-createIncomingTransferCalled:
                        case <-time.After(time.Second):
                            Fail("Test timed out")
                        }                       

                        downloader.CancelDownload(0)
                    })

                    Specify("For each chunk sent, all entries in that chunk should be written to the partition", func() {
                        siteAMergeCalled := make(chan int)
                        siteBMergeCalled := make(chan int)

                        siteADefaultBucket := NewMockBucket("default")
                        siteADefaultBucket.AppendNextMergeResponse(nil)
                        siteADefaultBucket.AppendNextMergeResponse(nil)
                        siteADefaultBucket.onMerge(func(siblingSets map[string]*SiblingSet) {
                            defer GinkgoRecover()

                            Expect(len(siblingSets)).Should(Equal(1))
                            var key string

                            if siteADefaultBucket.mergeCalls == 1 {
                                key = "aa"
                            } else {
                                key = "bb"
                            }

                            _, ok := siblingSets[key]

                            Expect(ok).Should(BeTrue())

                            siteAMergeCalled <- 1
                        })
                        siteBDefaultBucket := NewMockBucket("default")
                        siteBDefaultBucket.AppendNextMergeResponse(nil)
                        siteBDefaultBucket.AppendNextMergeResponse(nil)
                        siteBDefaultBucket.onMerge(func(siblingSets map[string]*SiblingSet) {
                            defer GinkgoRecover()

                            Expect(len(siblingSets)).Should(Equal(1))
                            var key string

                            if siteBDefaultBucket.mergeCalls == 1 {
                                key = "aa"
                            } else {
                                key = "bb"
                            }

                            _, ok := siblingSets[key]

                            Expect(ok).Should(BeTrue())

                            siteBMergeCalled <- 1
                        })
                        siteABuckets := NewBucketList()
                        siteABuckets.AddBucket(siteADefaultBucket)
                        siteBBuckets := NewBucketList()
                        siteBBuckets.AddBucket(siteBDefaultBucket)
                        siteA := NewMockSite(siteABuckets)
                        siteB := NewMockSite(siteBBuckets)
                        sitePool := NewMockSitePool()
                        sitePool.AppendNextAcquireResponse(siteA)
                        sitePool.AppendNextAcquireResponse(siteA)
                        sitePool.AppendNextAcquireResponse(siteB)
                        sitePool.AppendNextAcquireResponse(siteB)
                        partition := NewMockPartition(0, 0)
                        partition.SetSites(sitePool)
                        partitionPool := NewMockPartitionPool()
                        partitionPool.Add(partition)
                        partnerStrategy := NewMockTransferPartnerStrategy()
                        partnerStrategy.AppendNextTransferPartner(2)
                        transferTransport := NewMockTransferTransport()
                        infiniteReader := NewInfiniteReader()
                        transferTransport.AppendNextGetResponse(infiniteReader, func() { }, nil)
                        incomingTransfer := NewMockPartitionTransfer()
                        incomingTransfer.AppendNextChunkResponse(PartitionChunk{ Index: 1, Checksum: Hash{ }, Entries: []Entry{ 
                            Entry{ 
                                Site: "siteA",
                                Bucket: "default",
                                Key: "aa",
                                Value: nil,
                            },
                            Entry{ 
                                Site: "siteA",
                                Bucket: "default",
                                Key: "bb",
                                Value: nil,
                            },
                        } }, nil)
                        incomingTransfer.AppendNextChunkResponse(PartitionChunk{ Index: 2, Checksum: Hash{ }, Entries: []Entry{ 
                            Entry{ 
                                Site: "siteB",
                                Bucket: "default",
                                Key: "aa",
                                Value: nil,
                            },
                            Entry{ 
                                Site: "siteB",
                                Bucket: "default",
                                Key: "bb",
                                Value: nil,
                            },
                        } }, nil)
                        incomingTransfer.AppendNextChunkResponse(PartitionChunk{ }, io.EOF)
                        transferFactory := NewMockPartitionTransferFactory()
                        transferFactory.AppendNextIncomingTransfer(incomingTransfer)

                        // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                        downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, partitionPool)
                        downloader.Download(0)

                        select {
                        case <-siteAMergeCalled:
                        case <-time.After(time.Second):
                            Fail("Test timed out")
                        }

                        select {
                        case <-siteAMergeCalled:
                        case <-time.After(time.Second):
                            Fail("Test timed out")
                        }

                        select {
                        case <-siteBMergeCalled:
                        case <-time.After(time.Second):
                            Fail("Test timed out")
                        }

                        select {
                        case <-siteBMergeCalled:
                        case <-time.After(time.Second):
                            Fail("Test timed out")
                        }

                        downloader.CancelDownload(0)
                    })

                    Context("And if that partition is not in the partition pool", func() {
                        It("Should cause a panic", func() {
                            partitionPool := NewMockPartitionPool()
                            partnerStrategy := NewMockTransferPartnerStrategy()
                            partnerStrategy.AppendNextTransferPartner(2)
                            transferTransport := NewMockTransferTransport()
                            infiniteReader := NewInfiniteReader()
                            transferTransport.AppendNextGetResponse(infiniteReader, func() { }, nil)
                            incomingTransfer := NewMockPartitionTransfer()
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ Index: 1, Checksum: Hash{ }, Entries: []Entry{ 
                                Entry{ 
                                    Site: "siteA",
                                    Bucket: "default",
                                    Key: "aa",
                                    Value: nil,
                                },
                            } }, nil)
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ }, io.EOF)
                            transferFactory := NewMockPartitionTransferFactory()
                            transferFactory.AppendNextIncomingTransfer(incomingTransfer)
    
                            // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                            panicHappened := make(chan int)
                            downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, partitionPool)
                            downloader.OnPanic(func(p interface{}) {
                                panicHappened <- 1
                            })
                            downloader.Download(0)

                            select {
                            case <-panicHappened:
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }
                            
                            downloader.CancelDownload(0)                           
                        })
                    })

                    Context("And if that bucket is not in the partition", func() {
                        It("Should cause a panic", func() {
                            siteABuckets := NewBucketList()
                            siteBBuckets := NewBucketList()
                            siteA := NewMockSite(siteABuckets)
                            siteB := NewMockSite(siteBBuckets)
                            sitePool := NewMockSitePool()
                            sitePool.AppendNextAcquireResponse(siteA)
                            sitePool.AppendNextAcquireResponse(siteA)
                            sitePool.AppendNextAcquireResponse(siteB)
                            sitePool.AppendNextAcquireResponse(siteB)
                            partition := NewMockPartition(0, 0)
                            partition.SetSites(sitePool)
                            partitionPool := NewMockPartitionPool()
                            partitionPool.Add(partition)
                            partnerStrategy := NewMockTransferPartnerStrategy()
                            partnerStrategy.AppendNextTransferPartner(2)
                            transferTransport := NewMockTransferTransport()
                            infiniteReader := NewInfiniteReader()
                            transferTransport.AppendNextGetResponse(infiniteReader, func() { }, nil)
                            incomingTransfer := NewMockPartitionTransfer()
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ Index: 1, Checksum: Hash{ }, Entries: []Entry{ 
                                Entry{
                                    Site: "siteB",
                                    Bucket: "default",
                                    Key: "bb",
                                    Value: nil,
                                },
                            } }, nil)
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ }, io.EOF)
                            transferFactory := NewMockPartitionTransferFactory()
                            transferFactory.AppendNextIncomingTransfer(incomingTransfer)

                            // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                            panicHappened := make(chan int)
                            downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, partitionPool)
                            downloader.OnPanic(func(p interface{}) {
                                panicHappened <- 1
                            })
                            downloader.Download(0)

                            select {
                            case <-panicHappened:
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            downloader.CancelDownload(0)
                        })
                    })

                    Context("And if that site is not in the partition at the local node", func() {
                        It("Should ensure the transport cleans up by calling its closer function", func() {
                            transferCancelHappened := make(chan int)
                            transportCancelHappened := make(chan int)
                            sitePool := NewMockSitePool()
                            partition := NewMockPartition(0, 0)
                            partition.SetSites(sitePool)
                            partitionPool := NewMockPartitionPool()
                            partitionPool.Add(partition)
                            partnerStrategy := NewMockTransferPartnerStrategy()
                            partnerStrategy.AppendNextTransferPartner(2)
                            transferTransport := NewMockTransferTransport()
                            infiniteReader := NewInfiniteReader()
                            transportCancel := func() {
                                transportCancelHappened <- 1
                            }
                            transferTransport.AppendNextGetResponse(infiniteReader, transportCancel, nil)
                            incomingTransfer := NewMockPartitionTransfer()
                            incomingTransfer.onCancel(func() {
                                transferCancelHappened <- 1
                            })
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ Index: 1, Checksum: Hash{ }, Entries: []Entry{ 
                                Entry{ 
                                    Site: "siteA",
                                    Bucket: "default",
                                    Key: "aa",
                                    Value: nil,
                                },
                            } }, nil)
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ }, io.EOF)
                            transferFactory := NewMockPartitionTransferFactory()
                            transferFactory.AppendNextIncomingTransfer(incomingTransfer)

                            // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                            downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, partitionPool)
                            downloader.Download(0)

                            select {
                            case <-transportCancelHappened:
                                select {
                                case <-transferCancelHappened:
                                case <-time.After(time.Second):
                                    Fail("Test timed out")
                                }
                            case <-transferCancelHappened:
                                select {
                                case <-transportCancelHappened:
                                case <-time.After(time.Second):
                                    Fail("Test timed out")
                                }
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            downloader.CancelDownload(0)
                        })

                        It("Should wait for some period of time before again choosing a transfer partner and attempting to restart the download", func() {
                            chooseTransferPartnerCalled := make(chan int)
                            getCalled := make(chan int)
                            sitePool := NewMockSitePool()
                            partition := NewMockPartition(0, 0)
                            partition.SetSites(sitePool)
                            partitionPool := NewMockPartitionPool()
                            partitionPool.Add(partition)
                            partnerStrategy := NewMockTransferPartnerStrategy()
                            partnerStrategy.AppendNextTransferPartner(2)
                            partnerStrategy.AppendNextTransferPartner(2)
                            partnerStrategy.onChooseTransferPartner(func(partition uint64) {
                                defer GinkgoRecover()
                                Expect(partition).Should(Equal(uint64(0)))
                                chooseTransferPartnerCalled <- 1
                            })
                            transferTransport := NewMockTransferTransport()
                            transferTransport.onGet(func(node uint64, partition uint64) {
                                defer GinkgoRecover()
                                Expect(node).Should(Equal(uint64(2)))
                                Expect(partition).Should(Equal(uint64(0)))
                                getCalled <- 1
                            })
                            infiniteReader := NewInfiniteReader()
                            transferTransport.AppendNextGetResponse(infiniteReader, func() { }, nil)
                            transferTransport.AppendNextGetResponse(infiniteReader, func() { }, nil)
                            incomingTransfer := NewMockPartitionTransfer()
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ Index: 1, Checksum: Hash{ }, Entries: []Entry{ 
                                Entry{ 
                                    Site: "siteA",
                                    Bucket: "default",
                                    Key: "aa",
                                    Value: nil,
                                },
                            } }, nil)
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ Index: 1, Checksum: Hash{ }, Entries: []Entry{ 
                                Entry{ 
                                    Site: "siteA",
                                    Bucket: "default",
                                    Key: "aa",
                                    Value: nil,
                                },
                            } }, nil)
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ }, io.EOF)
                            transferFactory := NewMockPartitionTransferFactory()
                            transferFactory.AppendNextIncomingTransfer(incomingTransfer)
                            transferFactory.AppendNextIncomingTransfer(incomingTransfer)

                            // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                            downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, partitionPool)
                            downloader.Download(0)

                            select {
                            case <-chooseTransferPartnerCalled:
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            select {
                            case <-getCalled:
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            select {
                            case <-chooseTransferPartnerCalled:
                            // Wait period should be no more than 2 seconds
                            case <-time.After(time.Second * 2):
                                Fail("Test timed out")
                            }

                            select {
                            case <-getCalled:
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            downloader.CancelDownload(0)
                        })
                    })

                    Context("And if an error occurs while writing to the partition", func() {
                        It("Should ensure the transport cleans up by calling its closer function", func() {
                            transferCancelHappened := make(chan int)
                            transportCancelHappened := make(chan int)
                            siteADefaultBucket := NewMockBucket("default")
                            siteADefaultBucket.AppendNextMergeResponse(errors.New("Something bad happened"))
                            siteABuckets := NewBucketList()
                            siteABuckets.AddBucket(siteADefaultBucket)
                            siteA := NewMockSite(siteABuckets)
                            sitePool := NewMockSitePool()
                            sitePool.AppendNextAcquireResponse(siteA)
                            partition := NewMockPartition(0, 0)
                            partition.SetSites(sitePool)
                            partitionPool := NewMockPartitionPool()
                            partitionPool.Add(partition)
                            partnerStrategy := NewMockTransferPartnerStrategy()
                            partnerStrategy.AppendNextTransferPartner(2)
                            transferTransport := NewMockTransferTransport()
                            infiniteReader := NewInfiniteReader()
                            transportCancel := func() {
                                transportCancelHappened <- 1
                            }
                            transferTransport.AppendNextGetResponse(infiniteReader, transportCancel, nil)
                            incomingTransfer := NewMockPartitionTransfer()
                            incomingTransfer.onCancel(func() {
                                transferCancelHappened <- 1
                            })
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ Index: 1, Checksum: Hash{ }, Entries: []Entry{ 
                                Entry{ 
                                    Site: "siteA",
                                    Bucket: "default",
                                    Key: "aa",
                                    Value: nil,
                                },
                            } }, nil)
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ }, io.EOF)
                            transferFactory := NewMockPartitionTransferFactory()
                            transferFactory.AppendNextIncomingTransfer(incomingTransfer)

                            // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                            downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, partitionPool)
                            downloader.Download(0)

                            select {
                            case <-transportCancelHappened:
                                select {
                                case <-transferCancelHappened:
                                case <-time.After(time.Second):
                                    Fail("Test timed out")
                                }
                            case <-transferCancelHappened:
                                select {
                                case <-transportCancelHappened:
                                case <-time.After(time.Second):
                                    Fail("Test timed out")
                                }
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            downloader.CancelDownload(0)
                        })

                        It("Should wait for some period of time before again choosing a transfer partner and attempting to restart the download", func() {
                            chooseTransferPartnerCalled := make(chan int)
                            getCalled := make(chan int)
                            siteADefaultBucket := NewMockBucket("default")
                            siteADefaultBucket.AppendNextMergeResponse(errors.New("Something bad happened"))
                            siteADefaultBucket.AppendNextMergeResponse(errors.New("Something bad happened"))
                            siteABuckets := NewBucketList()
                            siteABuckets.AddBucket(siteADefaultBucket)
                            siteA := NewMockSite(siteABuckets)
                            sitePool := NewMockSitePool()
                            sitePool.AppendNextAcquireResponse(siteA)
                            sitePool.AppendNextAcquireResponse(siteA)
                            partition := NewMockPartition(0, 0)
                            partition.SetSites(sitePool)
                            partitionPool := NewMockPartitionPool()
                            partitionPool.Add(partition)
                            partnerStrategy := NewMockTransferPartnerStrategy()
                            partnerStrategy.onChooseTransferPartner(func(partition uint64) {
                                defer GinkgoRecover()
                                Expect(partition).Should(Equal(uint64(0)))
                                chooseTransferPartnerCalled <- 1
                            })
                            partnerStrategy.AppendNextTransferPartner(2)
                            partnerStrategy.AppendNextTransferPartner(2)
                            transferTransport := NewMockTransferTransport()
                            transferTransport.onGet(func(node uint64, partition uint64) {
                                defer GinkgoRecover()
                                Expect(node).Should(Equal(uint64(2)))
                                Expect(partition).Should(Equal(uint64(0)))
                                getCalled <- 1
                            })
                            infiniteReader := NewInfiniteReader()
                            transferTransport.AppendNextGetResponse(infiniteReader, func() { }, nil)
                            transferTransport.AppendNextGetResponse(infiniteReader, func() { }, nil)
                            incomingTransfer := NewMockPartitionTransfer()
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ Index: 1, Checksum: Hash{ }, Entries: []Entry{ 
                                Entry{ 
                                    Site: "siteA",
                                    Bucket: "default",
                                    Key: "aa",
                                    Value: nil,
                                },
                            } }, nil)
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ Index: 2, Checksum: Hash{ }, Entries: []Entry{ 
                                Entry{ 
                                    Site: "siteA",
                                    Bucket: "default",
                                    Key: "aa",
                                    Value: nil,
                                },
                            } }, nil)
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ }, io.EOF)
                            transferFactory := NewMockPartitionTransferFactory()
                            transferFactory.AppendNextIncomingTransfer(incomingTransfer)
                            transferFactory.AppendNextIncomingTransfer(incomingTransfer)

                            // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                            downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, partitionPool)
                            downloader.Download(0)

                            select {
                            case <-chooseTransferPartnerCalled:
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            select {
                            case <-getCalled:
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            select {
                            case <-chooseTransferPartnerCalled:
                            // Wait period should be no more than 2 seconds
                            case <-time.After(time.Second * 2):
                                Fail("Test timed out")
                            }

                            select {
                            case <-getCalled:
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            downloader.CancelDownload(0)
                        })
                    })

                    Context("And if there is a problem encountered while decoding the incoming transfer", func() {
                        It("Should ensure the transport cleans up by calling its closer function", func() {
                            transferCancelHappened := make(chan int)
                            transportCancelHappened := make(chan int)
                            siteADefaultBucket := NewMockBucket("default")
                            siteADefaultBucket.AppendNextMergeResponse(nil)
                            siteABuckets := NewBucketList()
                            siteABuckets.AddBucket(siteADefaultBucket)
                            siteA := NewMockSite(siteABuckets)
                            sitePool := NewMockSitePool()
                            sitePool.AppendNextAcquireResponse(siteA)
                            partition := NewMockPartition(0, 0)
                            partition.SetSites(sitePool)
                            partitionPool := NewMockPartitionPool()
                            partitionPool.Add(partition)
                            partnerStrategy := NewMockTransferPartnerStrategy()
                            partnerStrategy.AppendNextTransferPartner(2)
                            transferTransport := NewMockTransferTransport()
                            infiniteReader := NewInfiniteReader()
                            transportCancel := func() {
                                transportCancelHappened <- 1
                            }
                            transferTransport.AppendNextGetResponse(infiniteReader, transportCancel, nil)
                            incomingTransfer := NewMockPartitionTransfer()
                            incomingTransfer.onCancel(func() {
                                transferCancelHappened <- 1
                            })
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ }, errors.New("Something terrible happened"))
                            transferFactory := NewMockPartitionTransferFactory()
                            transferFactory.AppendNextIncomingTransfer(incomingTransfer)

                            // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                            downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, partitionPool)
                            downloader.Download(0)

                            select {
                            case <-transportCancelHappened:
                                select {
                                case <-transferCancelHappened:
                                case <-time.After(time.Second):
                                    Fail("Test timed out")
                                }
                            case <-transferCancelHappened:
                                select {
                                case <-transportCancelHappened:
                                case <-time.After(time.Second):
                                    Fail("Test timed out")
                                }
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            downloader.CancelDownload(0)
                        })

                        It("Should wait for some period of time before again choosing a transfer partner and attempting to restart the download", func() {
                            chooseTransferPartnerCalled := make(chan int)
                            getCalled := make(chan int)
                            siteADefaultBucket := NewMockBucket("default")
                            siteADefaultBucket.AppendNextMergeResponse(nil)
                            siteABuckets := NewBucketList()
                            siteABuckets.AddBucket(siteADefaultBucket)
                            siteA := NewMockSite(siteABuckets)
                            sitePool := NewMockSitePool()
                            sitePool.AppendNextAcquireResponse(siteA)
                            sitePool.AppendNextAcquireResponse(siteA)
                            partition := NewMockPartition(0, 0)
                            partition.SetSites(sitePool)
                            partitionPool := NewMockPartitionPool()
                            partitionPool.Add(partition)
                            partnerStrategy := NewMockTransferPartnerStrategy()
                            partnerStrategy.onChooseTransferPartner(func(partition uint64) {
                                defer GinkgoRecover()
                                Expect(partition).Should(Equal(uint64(0)))
                                chooseTransferPartnerCalled <- 1
                            })
                            partnerStrategy.AppendNextTransferPartner(2)
                            partnerStrategy.AppendNextTransferPartner(2)
                            transferTransport := NewMockTransferTransport()
                            transferTransport.onGet(func(node uint64, partition uint64) {
                                defer GinkgoRecover()
                                Expect(node).Should(Equal(uint64(2)))
                                Expect(partition).Should(Equal(uint64(0)))
                                getCalled <- 1
                            })
                            infiniteReader := NewInfiniteReader()
                            transferTransport.AppendNextGetResponse(infiniteReader, func() { }, nil)
                            transferTransport.AppendNextGetResponse(infiniteReader, func() { }, nil)
                            incomingTransfer := NewMockPartitionTransfer()
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ Index: 1, Checksum: Hash{ }, Entries: []Entry{ 
                                Entry{ 
                                    Site: "siteA",
                                    Bucket: "default",
                                    Key: "aa",
                                    Value: nil,
                                },
                            } }, errors.New("Something terrible happened"))
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ Index: 2, Checksum: Hash{ }, Entries: []Entry{ 
                                Entry{ 
                                    Site: "siteA",
                                    Bucket: "default",
                                    Key: "aa",
                                    Value: nil,
                                },
                            } }, errors.New("Something terrible happened"))
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ }, io.EOF)
                            transferFactory := NewMockPartitionTransferFactory()
                            transferFactory.AppendNextIncomingTransfer(incomingTransfer)
                            transferFactory.AppendNextIncomingTransfer(incomingTransfer)

                            // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                            downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, partitionPool)
                            downloader.Download(0)

                            select {
                            case <-chooseTransferPartnerCalled:
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            select {
                            case <-getCalled:
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            select {
                            case <-chooseTransferPartnerCalled:
                            // Wait period should be no more than 2 seconds
                            case <-time.After(time.Second * 2):
                                Fail("Test timed out")
                            }

                            select {
                            case <-getCalled:
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            downloader.CancelDownload(0)
                        })
                    })

                    Context("And when the download is finished successfully", func() {
                        It("Should ensure the transport cleans up by calling its closer function", func() {
                            transferCancelHappened := make(chan int)
                            transportCancelHappened := make(chan int)
                            partnerStrategy := NewMockTransferPartnerStrategy()
                            partnerStrategy.AppendNextTransferPartner(2)
                            transferTransport := NewMockTransferTransport()
                            infiniteReader := NewInfiniteReader()
                            transportCancel := func() {
                                transportCancelHappened <- 1
                            }
                            transferTransport.AppendNextGetResponse(infiniteReader, transportCancel, nil)
                            incomingTransfer := NewMockPartitionTransfer()
                            incomingTransfer.onCancel(func() {
                                transferCancelHappened <- 1
                            })
                            // io.EOF indicates the end of the transfer
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ }, io.EOF)
                            transferFactory := NewMockPartitionTransferFactory()
                            transferFactory.AppendNextIncomingTransfer(incomingTransfer)

                            // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                            downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, nil)
                            downloader.Download(0)

                            select {
                            case <-transportCancelHappened:
                                select {
                                case <-transferCancelHappened:
                                case <-time.After(time.Second):
                                    Fail("Test timed out")
                                }
                            case <-transferCancelHappened:
                                select {
                                case <-transportCancelHappened:
                                case <-time.After(time.Second):
                                    Fail("Test timed out")
                                }
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            downloader.CancelDownload(0)
                        })

                        It("Should close the done channel for this partition download", func() {
                            transferCancelHappened := make(chan int)
                            transportCancelHappened := make(chan int)
                            partnerStrategy := NewMockTransferPartnerStrategy()
                            partnerStrategy.AppendNextTransferPartner(2)
                            transferTransport := NewMockTransferTransport()
                            infiniteReader := NewInfiniteReader()
                            transportCancel := func() {
                                transportCancelHappened <- 1
                            }
                            transferTransport.AppendNextGetResponse(infiniteReader, transportCancel, nil)
                            incomingTransfer := NewMockPartitionTransfer()
                            incomingTransfer.onCancel(func() {
                                transferCancelHappened <- 1
                            })
                            // io.EOF indicates the end of the transfer
                            incomingTransfer.AppendNextChunkResponse(PartitionChunk{ }, io.EOF)
                            transferFactory := NewMockPartitionTransferFactory()
                            transferFactory.AppendNextIncomingTransfer(incomingTransfer)

                            // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                            downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, nil)
                            done := downloader.Download(0)

                            select {
                            case <-transportCancelHappened:
                                select {
                                case <-transferCancelHappened:
                                case <-time.After(time.Second):
                                    Fail("Test timed out")
                                }
                            case <-transferCancelHappened:
                                select {
                                case <-transportCancelHappened:
                                case <-time.After(time.Second):
                                    Fail("Test timed out")
                                }
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            select {
                            case <-done:
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }
                        })
                    })

                    Context("And if Cancel is called for that partition while the download is in progress", func() {
                        It("Should call Cancel on the incoming transfer", func() {
                            mergeCalled := make(chan int)
                            siteADefaultBucket := NewMockBucket("default")
                            siteADefaultBucket.onMerge(func(siblingSets map[string]*SiblingSet) {
                                // This condition ensures that future calls to merge do not block, allowing
                                // cancel to work
                                if siteADefaultBucket.MergeCallCount() <= 2 {
                                    mergeCalled <- 1
                                }
                            })
                            siteADefaultBucket.SetDefaultMergeResponse(nil)
                            siteABuckets := NewBucketList()
                            siteABuckets.AddBucket(siteADefaultBucket)
                            siteA := NewMockSite(siteABuckets)
                            sitePool := NewMockSitePool()
                            sitePool.SetDefaultAcquireResponse(siteA)
                            partition := NewMockPartition(0, 0)
                            partition.SetSites(sitePool)
                            partitionPool := NewMockPartitionPool()
                            partitionPool.Add(partition)

                            transferCancelHappened := make(chan int)
                            transportCancelHappened := make(chan int)
                            partnerStrategy := NewMockTransferPartnerStrategy()
                            partnerStrategy.SetDefaultChooseTransferPartnerResponse(2)
                            transferTransport := NewMockTransferTransport()
                            infiniteReader := NewInfiniteReader()
                            transportCancel := func() {
                                transportCancelHappened <- 1
                            }
                            transferTransport.SetDefaultGetResponse(infiniteReader, transportCancel, nil)
                            incomingTransfer := NewMockPartitionTransfer()
                            incomingTransfer.onCancel(func() {
                                transferCancelHappened <- 1
                            })
                            // Default response ensures the transfer never ends on its own. We will have to call cancel in order for things to wrap up
                            incomingTransfer.SetDefaultNextChunkResponse(PartitionChunk{ Index: 1, Entries: []Entry{ Entry{ 
                                Site: "siteA",
                                Bucket: "default",
                                Key: "aa",
                                Value: nil,
                            } } }, nil)
                            transferFactory := NewMockPartitionTransferFactory()
                            transferFactory.SetDefaultIncomingTransfer(incomingTransfer)

                            // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                            downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, partitionPool)
                            downloader.Download(0)

                            // Wait for merge to be called two times to know it's working
                            select {
                            case <-mergeCalled:
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            select {
                            case <-mergeCalled:
                                siteADefaultBucket.onMerge(nil)
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            // Cancel and wait for the shutdown prodedure
                            downloader.CancelDownload(0)

                            select {
                            case <-transportCancelHappened:
                                select {
                                case <-transferCancelHappened:
                                case <-time.After(time.Second):
                                    Fail("Test timed out")
                                }
                            case <-transferCancelHappened:
                                select {
                                case <-transportCancelHappened:
                                case <-time.After(time.Second * 5):
                                    Fail("Test timed out")
                                }
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }
                        })

                        It("Should leave the done channel open", func() {
                            mergeCalled := make(chan int)
                            siteADefaultBucket := NewMockBucket("default")
                            siteADefaultBucket.onMerge(func(siblingSets map[string]*SiblingSet) {
                                // This condition ensures that future calls to merge do not block, allowing
                                // cancel to work
                                if siteADefaultBucket.MergeCallCount() <= 2 {
                                    mergeCalled <- 1
                                }
                            })
                            siteADefaultBucket.SetDefaultMergeResponse(nil)
                            siteABuckets := NewBucketList()
                            siteABuckets.AddBucket(siteADefaultBucket)
                            siteA := NewMockSite(siteABuckets)
                            sitePool := NewMockSitePool()
                            sitePool.SetDefaultAcquireResponse(siteA)
                            partition := NewMockPartition(0, 0)
                            partition.SetSites(sitePool)
                            partitionPool := NewMockPartitionPool()
                            partitionPool.Add(partition)

                            partnerStrategy := NewMockTransferPartnerStrategy()
                            partnerStrategy.SetDefaultChooseTransferPartnerResponse(2)
                            transferTransport := NewMockTransferTransport()
                            infiniteReader := NewInfiniteReader()
                            transferTransport.SetDefaultGetResponse(infiniteReader, func() { }, nil)
                            incomingTransfer := NewMockPartitionTransfer()
                            // Default response ensures the transfer never ends on its own. We will have to call cancel in order for things to wrap up
                            incomingTransfer.SetDefaultNextChunkResponse(PartitionChunk{ Index: 1, Entries: []Entry{ Entry{ 
                                Site: "siteA",
                                Bucket: "default",
                                Key: "aa",
                                Value: nil,
                            } } }, nil)
                            transferFactory := NewMockPartitionTransferFactory()
                            transferFactory.SetDefaultIncomingTransfer(incomingTransfer)

                            // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                            downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, partitionPool)
                            done := downloader.Download(0)

                            // Wait for merge to be called two times to know it's working
                            select {
                            case <-mergeCalled:
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            select {
                            case <-mergeCalled:
                                siteADefaultBucket.onMerge(nil)
                            case <-time.After(time.Second):
                                Fail("Test timed out")
                            }

                            downloader.CancelDownload(0)

                            // After canceling the download ensure that the returned done channel is not closed
                            select {
                            case <-done:
                                Fail("Done should not have closed")
                            case <-time.After(time.Second):
                            }
                        })
                    })
                })
            })

            Context("And there is already a download in progress for this partition", func() {
                It("Should return the done channel for that partition", func() {
                    mergeCalled := make(chan int)
                    siteADefaultBucket := NewMockBucket("default")
                    siteADefaultBucket.onMerge(func(siblingSets map[string]*SiblingSet) {
                        // This condition ensures that future calls to merge do not block, allowing
                        // cancel to work
                        if siteADefaultBucket.MergeCallCount() <= 2 {
                            mergeCalled <- 1
                        }
                    })
                    siteADefaultBucket.SetDefaultMergeResponse(nil)
                    siteABuckets := NewBucketList()
                    siteABuckets.AddBucket(siteADefaultBucket)
                    siteA := NewMockSite(siteABuckets)
                    sitePool := NewMockSitePool()
                    sitePool.SetDefaultAcquireResponse(siteA)
                    partition := NewMockPartition(0, 0)
                    partition.SetSites(sitePool)
                    partitionPool := NewMockPartitionPool()
                    partitionPool.Add(partition)

                    partnerStrategy := NewMockTransferPartnerStrategy()
                    partnerStrategy.SetDefaultChooseTransferPartnerResponse(2)
                    transferTransport := NewMockTransferTransport()
                    infiniteReader := NewInfiniteReader()
                    transferTransport.SetDefaultGetResponse(infiniteReader, func() { }, nil)
                    incomingTransfer := NewMockPartitionTransfer()
                    // Default response ensures the transfer never ends on its own. We will have to call cancel in order for things to wrap up
                    incomingTransfer.SetDefaultNextChunkResponse(PartitionChunk{ Index: 1, Entries: []Entry{ Entry{ 
                        Site: "siteA",
                        Bucket: "default",
                        Key: "aa",
                        Value: nil,
                    } } }, nil)
                    transferFactory := NewMockPartitionTransferFactory()
                    transferFactory.SetDefaultIncomingTransfer(incomingTransfer)

                    // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                    downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, partitionPool)
                    done := downloader.Download(0)

                    // Wait for merge to be called two times to know it's working
                    select {
                    case <-mergeCalled:
                    case <-time.After(time.Second):
                        Fail("Test timed out")
                    }

                    select {
                    case <-mergeCalled:
                        siteADefaultBucket.onMerge(nil)
                    case <-time.After(time.Second):
                        Fail("Test timed out")
                    }

                    Expect(downloader.Download(0)).Should(Equal(done))
                    Expect(downloader.Download(0)).Should(Equal(done))
                    Expect(downloader.Download(0)).Should(Equal(done))

                    downloader.CancelDownload(0)

                    // If the download is restarted it should return a new done channel
                    Expect(downloader.Download(0)).Should(Not(Equal(done)))

                    downloader.CancelDownload(0)
                })
            })
            
            Context("And a download already successfully completed for this partition", func() {
                Context("And Reset() has not yet been called for that partition", func() {
                    It("Should return the done channel for that partition which should be closed", func() {
                        partnerStrategy := NewMockTransferPartnerStrategy()
                        partnerStrategy.SetDefaultChooseTransferPartnerResponse(2)
                        transferTransport := NewMockTransferTransport()
                        infiniteReader := NewInfiniteReader()
                        transferTransport.SetDefaultGetResponse(infiniteReader, func() { }, nil)
                        incomingTransfer := NewMockPartitionTransfer()
                        // Default response to ensure the transfer completes right away
                        incomingTransfer.SetDefaultNextChunkResponse(PartitionChunk{ }, io.EOF)
                        transferFactory := NewMockPartitionTransferFactory()
                        transferFactory.SetDefaultIncomingTransfer(incomingTransfer)

                        // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                        downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, nil)
                        done := downloader.Download(0)

                        // Wait for the download to finish
                        select {
                        case <-done:
                        case <-time.After(time.Second):
                            Fail("Test timed out")
                        }

                        Expect(downloader.Download(0)).Should(Equal(done))
                        Expect(downloader.Download(0)).Should(Equal(done))
                        Expect(downloader.Download(0)).Should(Equal(done))
                    })
                })

                Context("And Reset() has been called for that partition", func() {
                    It("Should start a new download for that partition", func() {
                        partnerStrategy := NewMockTransferPartnerStrategy()
                        partnerStrategy.SetDefaultChooseTransferPartnerResponse(2)
                        transferTransport := NewMockTransferTransport()
                        infiniteReader := NewInfiniteReader()
                        transferTransport.SetDefaultGetResponse(infiniteReader, func() { }, nil)
                        incomingTransfer := NewMockPartitionTransfer()
                        // Default response to ensure the transfer completes right away
                        incomingTransfer.SetDefaultNextChunkResponse(PartitionChunk{ }, io.EOF)
                        transferFactory := NewMockPartitionTransferFactory()
                        transferFactory.SetDefaultIncomingTransfer(incomingTransfer)

                        // download won't be able to make progress since the transfer strategy will return an error. That way the done channel won't close
                        downloader := NewDownloader(configController, transferTransport, partnerStrategy, transferFactory, nil)
                        done := downloader.Download(0)

                        // Wait for the download to finish
                        select {
                        case <-done:
                        case <-time.After(time.Second):
                            Fail("Test timed out")
                        }

                        Expect(downloader.Download(0)).Should(Equal(done))
                        Expect(downloader.Download(0)).Should(Equal(done))
                        Expect(downloader.Download(0)).Should(Equal(done))

                        downloader.Reset(0)
                        done2 := downloader.Download(0)

                        Expect(done2).Should(Not(Equal(done)))
                        Expect(downloader.Download(0)).Should(Equal(done2))
                        Expect(downloader.Download(0)).Should(Equal(done2))
                        Expect(downloader.Download(0)).Should(Equal(done2))
                    })
                })
            })
        })
    })
})
