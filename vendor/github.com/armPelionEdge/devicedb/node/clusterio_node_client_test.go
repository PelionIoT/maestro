package node_test

import (
    "context"
    "encoding/json"
    "errors"
    "net/http"
    "net/url"
    "strings"
    "strconv"
    "time"

    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/node"
    . "github.com/armPelionEdge/devicedb/raft"
    . "github.com/armPelionEdge/devicedb/routes"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
    "github.com/onsi/gomega/ghttp"
)

func parseURL(urlString string) (string, int) {
    u, _ := url.Parse(urlString)
    parts := strings.Split(u.Host, ":")
    host := parts[0]
    port, _ := strconv.Atoi(parts[1])

    return host, port
}

var _ = Describe("ClusterioNodeClient", func() {
    var server *ghttp.Server
    var client *NodeClient
    var configController *MockConfigController
    var localNode *MockNode
    var localNodeID uint64 = 0x1
    var remoteNodeID uint64 = 0x2
    var unknownNodeID uint64 = 0x1000

    BeforeEach(func() {
        localNode = NewMockNode(localNodeID)
        server = ghttp.NewServer()
        serverHost, serverPort := parseURL(server.URL())
        configController = NewMockConfigController(&ClusterController{
            LocalNodeID: localNodeID,
            PartitioningStrategy: &SimplePartitioningStrategy{ },
        })
        // Add the local node
        configController.ClusterController().AddNode(ClusterAddNodeBody{ NodeID: localNodeID, NodeConfig: NodeConfig{ Address: PeerAddress{ NodeID: localNodeID }, Capacity: 1 } })
        // Add some remote node whose address matches that of the mock server
        configController.ClusterController().AddNode(ClusterAddNodeBody{ NodeID: remoteNodeID, NodeConfig: NodeConfig{ Address: PeerAddress{ NodeID: remoteNodeID, Host: serverHost, Port: serverPort }, Capacity: 1 } })
        configController.ClusterController().SetPartitionCount(ClusterSetPartitionCountBody{ Partitions: 64 })
        configController.ClusterController().SetReplicationFactor(ClusterSetReplicationFactorBody{ ReplicationFactor: 3 })
        client = NewNodeClient(localNode, configController)
    })

    AfterEach(func() {
        server.Close()
    })

    Describe("#Merge", func() {
        Context("When the specified nodeID does not refer to a known node", func() {
            It("Should return an error", func() {
                Expect(client.Merge(context.TODO(), unknownNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(Not(BeNil()))
            })
        })

        Context("When the specified ndoeID refers to the local node", func() {
            It("Should invoke Merge() on the local node with the same parameters that were passed to it", func() {
                var patch map[string]*SiblingSet = map[string]*SiblingSet{ }

                mergeCalled := make(chan int, 1)
                localNode.mergeCB = func(ctx context.Context, partition uint64, siteID string, bucket string, p map[string]*SiblingSet, broadcastToRelays bool) {
                    Expect(partition).Should(Equal(uint64(50)))
                    Expect(siteID).Should(Equal("site1"))
                    Expect(bucket).Should(Equal("default"))
                    Expect(patch).Should(Equal(p))
                    Expect(broadcastToRelays).Should(BeTrue())

                    mergeCalled <- 1
                }

                client.Merge(context.TODO(), localNodeID, 50, "site1", "default", patch, true)

                select {
                case <-mergeCalled:
                default:
                    Fail("Did not invoke Merge() on local node")
                }
            })

            Context("And if the call to Merge() on the local node returns ENoQuorum", func() {
                It("Should return an error", func() {
                    localNode.defaultMergeError = ENoQuorum

                    Expect(client.Merge(context.TODO(), localNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(Equal(ENoQuorum))
                })
            })

            Context("And if the call to Merge() on the local node returns ENoSuchPartition", func() {
                It("Should return an error", func() {
                    localNode.defaultMergeError = ENoSuchPartition

                    Expect(client.Merge(context.TODO(), localNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(Not(BeNil()))
                })
            })

            Context("And if the call to Merge() on the local node returns an ENoSuchBucket error", func() {
                It("Should return an EBucketDoesNotExist error", func() {
                    localNode.defaultMergeError = ENoSuchBucket

                    Expect(client.Merge(context.TODO(), localNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(Equal(EBucketDoesNotExist))
                })
            })

            Context("And if the call to Merge() on the local node returns an ENoSuchSite error", func() {
                It("Should return an ESiteDoesNotExist error", func() {
                    localNode.defaultMergeError = ENoSuchSite

                    Expect(client.Merge(context.TODO(), localNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(Equal(ESiteDoesNotExist))
                })
            })

            Context("And if the call to Batch() on the local node returns any other error", func() {
                It("Should return an error", func() {
                    localNode.defaultMergeError = errors.New("Some error")

                    Expect(client.Merge(context.TODO(), localNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(Not(BeNil()))
                })
            })

            Context("And if the call to Batch() returns nil", func() {
                It("Should return nil", func() {
                    Expect(client.Merge(context.TODO(), localNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(BeNil())
                })
            })
        })

        Context("When the specified nodeID refers to a known node that is not the local node", func() {
            It("Should send a POST request to /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/merges at that node", func() {
                server.AppendHandlers(ghttp.VerifyRequest("POST", "/partitions/50/sites/site1/buckets/default/merges"))
                client.Merge(context.TODO(), remoteNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)
                Expect(server.ReceivedRequests()).Should(HaveLen(1))
            })

            It("Should send a POST request whose body is the encoded patch", func() {
                var patch map[string]*SiblingSet = map[string]*SiblingSet{ }
                var encodedPatch []byte
                encodedPatch, err := json.Marshal(patch)

                Expect(err).Should(BeNil())

                server.AppendHandlers(ghttp.VerifyBody(encodedPatch))
                client.Merge(context.TODO(), remoteNodeID, 50, "site1", "default", patch, true)
                Expect(server.ReceivedRequests()).Should(HaveLen(1))
            })

            Context("And ctx.Done() is closed before the http request responds", func() {
                It("Should return an error", func() {
                    var ctx context.Context
                    var cancel func()

                    ctx, cancel = context.WithCancel(context.Background())

                    server.AppendHandlers(func(w http.ResponseWriter, req *http.Request) {
                        cancel()
                        // Wait some time before returning to ensure that the call to Merge cancels due to the cancelled context
                        // before the server handler responds
                        <-time.After(time.Millisecond * 100)
                    })

                    mergeResult := make(chan error)

                    go func() {
                        mergeResult <- client.Merge(ctx, remoteNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)
                    }()

                    select {
                    case err := <-mergeResult:
                        Expect(err).Should(HaveOccurred())
                    case <-time.After(time.Second):
                        Fail("Merge did not return in time")
                    }

                    Expect(server.ReceivedRequests()).Should(HaveLen(1))
                })
            })

            Context("And the http client request call return an error", func() {
                It("Should return an error", func() {
                    // close the server to force an http connection error
                    server.Close()
                    Expect(client.Merge(context.TODO(), remoteNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(HaveOccurred())
                    Expect(server.ReceivedRequests()).Should(HaveLen(0))
                })
            })

            Context("And the http request responds with a 404 status code", func() {
                Context("And the body is a JSON-encoded EBucketDoesNotExist error", func() {
                    It("Should return EBucketDoesNotExist", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, EBucketDoesNotExist.JSON()))
                        Expect(client.Merge(context.TODO(), remoteNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(Equal(EBucketDoesNotExist))
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the body is a JSON-encoded ESiteDoesNotExist error", func() {
                    It("Should return ESiteDoesNotExist", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, ESiteDoesNotExist.JSON()))
                        Expect(client.Merge(context.TODO(), remoteNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(Equal(ESiteDoesNotExist))
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the body is not a JSON-encoded database error", func() {
                    It("Should return an error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, "asdf"))
                        Expect(client.Merge(context.TODO(), remoteNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(HaveOccurred())
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the body is a JSON-encoded database error but is neither EBucketDoesNotExist nor ESiteDoesNotExist", func() {
                    It("Should return an error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, EStorage.JSON()))
                        Expect(client.Merge(context.TODO(), remoteNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(HaveOccurred())
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })
            })

            Context("And the http request responds with a 500 status code", func() {
                It("Should return an error", func() {
                    server.AppendHandlers(ghttp.RespondWith(http.StatusInternalServerError, ""))
                    Expect(client.Merge(context.TODO(), remoteNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(HaveOccurred())
                    Expect(server.ReceivedRequests()).Should(HaveLen(1))
                })
            })

            Context("And the http request responds with a 200 status code", func() {
                Context("And the response body cannot be parsed as a BatchResult as defined in the routes module", func() {
                    It("Should return an error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "asdf"))
                        Expect(client.Merge(context.TODO(), remoteNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(HaveOccurred())
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the response body can be parsed as a BatchResult", func() {
                    var batchResult BatchResult

                    Context("And the NApplied field is 0", func() {
                        BeforeEach(func() {
                            batchResult.NApplied = 0
                        })

                        It("Should return ENoQuorum", func() {
                            encodedBatchResult, err := json.Marshal(batchResult)

                            Expect(err).Should(Not(HaveOccurred()))

                            server.AppendHandlers(ghttp.RespondWith(http.StatusOK, encodedBatchResult))
                            Expect(client.Merge(context.TODO(), remoteNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(Equal(ENoQuorum))
                            Expect(server.ReceivedRequests()).Should(HaveLen(1))
                        })
                    })

                    Context("And the NApplied field is not 0", func() {
                        BeforeEach(func() {
                            batchResult.NApplied = 1
                        })

                        It("Should return nil", func() {
                            encodedBatchResult, err := json.Marshal(batchResult)

                            Expect(err).Should(Not(HaveOccurred()))

                            server.AppendHandlers(ghttp.RespondWith(http.StatusOK, encodedBatchResult))
                            Expect(client.Merge(context.TODO(), remoteNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(Not(HaveOccurred()))
                            Expect(server.ReceivedRequests()).Should(HaveLen(1))
                        })
                    })
                })
            })

            Context("And the http request responds with a status code other than 200, 404, or 500", func() {
                It("Should return an error", func() {
                    server.AppendHandlers(ghttp.RespondWith(http.StatusForbidden, ""))
                    Expect(client.Merge(context.TODO(), remoteNodeID, 50, "site1", "default", map[string]*SiblingSet{ }, true)).Should(HaveOccurred())
                    Expect(server.ReceivedRequests()).Should(HaveLen(1))
                })
            })
        })
    })

    Describe("#Batch", func() {
        Context("When the specified nodeID does not refer to a known node", func() {
            It("Should return an error", func() {
                patch, err := client.Batch(context.TODO(), unknownNodeID, 50, "site1", "default", NewUpdateBatch())

                Expect(patch).Should(BeNil())
                Expect(err).Should(Not(BeNil()))
            })
        })

        Context("When the specified nodeID refers to the local node", func() {
            It("Should invoke Batch() on the local node with the same parameters that were passed to it", func() {
                var updateBatch *UpdateBatch = NewUpdateBatch()

                batchCalled := make(chan int, 1)
                localNode.batchCB = func(ctx context.Context, partition uint64, siteID string, bucket string, updateBatch *UpdateBatch) {
                    Expect(partition).Should(Equal(uint64(50)))
                    Expect(siteID).Should(Equal("site1"))
                    Expect(bucket).Should(Equal("default"))
                    Expect(updateBatch).Should(Equal(updateBatch))

                    batchCalled <- 1
                }

                client.Batch(context.TODO(), localNodeID, 50, "site1", "default", updateBatch)

                select {
                case <-batchCalled:
                default:
                    Fail("Did not invoke Batch() on local node")
                }
            })

            Context("And if the call to Batch() on the local node returns ENoQuorum", func() {
                It("Should return an error", func() {
                    localNode.defaultBatchError = ENoQuorum

                    patch, err := client.Batch(context.TODO(), localNodeID, 50, "site1", "default", NewUpdateBatch())

                    Expect(patch).Should(BeNil())
                    Expect(err).Should(Equal(ENoQuorum))
                })
            })

            Context("And if the call to Batch() on the local node returns ENoSuchPartition", func() {
                It("Should return an error", func() {
                    localNode.defaultBatchError = ENoSuchPartition

                    patch, err := client.Batch(context.TODO(), localNodeID, 50, "site1", "default", NewUpdateBatch())

                    Expect(patch).Should(BeNil())
                    Expect(err).Should(Not(BeNil()))
                })
            })

            Context("And if the call to Batch() on the local node returns an ENoSuchBucket error", func() {
                It("Should return an EBucketDoesNotExist error", func() {
                    localNode.defaultBatchError = ENoSuchBucket

                    patch, err := client.Batch(context.TODO(), localNodeID, 50, "site1", "default", NewUpdateBatch())

                    Expect(patch).Should(BeNil())
                    Expect(err).Should(Equal(EBucketDoesNotExist))
                })
            })

            Context("And if the call to Batch() on the local node returns an ENoSuchSite error", func() {
                It("Should return an ESiteDoesNotExist error", func() {
                    localNode.defaultBatchError = ENoSuchSite

                    patch, err := client.Batch(context.TODO(), localNodeID, 50, "site1", "default", NewUpdateBatch())

                    Expect(patch).Should(BeNil())
                    Expect(err).Should(Equal(ESiteDoesNotExist))
                })
            })

            Context("And if the call to Batch() on the local node returns any other error", func() {
                It("Should return an error", func() {
                    localNode.defaultBatchError = errors.New("Some error")

                    patch, err := client.Batch(context.TODO(), localNodeID, 50, "site1", "default", NewUpdateBatch())

                    Expect(patch).Should(BeNil())
                    Expect(err).Should(Not(BeNil()))
                })
            })

            Context("And if the call to Batch() returns nil", func() {
                It("Should return nil", func() {
                    localNode.defaultBatchPatch = map[string]*SiblingSet{ }

                    patch, err := client.Batch(context.TODO(), localNodeID, 50, "site1", "default", NewUpdateBatch())

                    Expect(patch).Should(Equal(localNode.defaultBatchPatch))
                    Expect(err).Should(BeNil())
                })
            })
        })

        Context("When the specified nodeID refers to a known node that is not the local node", func() {
            It("Should send a POST request to /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/batches at that node", func() {
                server.AppendHandlers(ghttp.VerifyRequest("POST", "/partitions/50/sites/site1/buckets/default/batches"))
                client.Batch(context.TODO(), remoteNodeID, 50, "site1", "default", NewUpdateBatch())
                Expect(server.ReceivedRequests()).Should(HaveLen(1))
            })

            It("Should send a POST request whose body is the encoded update batch", func() {
                var updateBatch *UpdateBatch = NewUpdateBatch()
                var encodedUpdateBatch []byte
                encodedUpdateBatch, err := updateBatch.ToJSON()

                Expect(err).Should(BeNil())

                server.AppendHandlers(ghttp.VerifyBody(encodedUpdateBatch))
                client.Batch(context.TODO(), remoteNodeID, 50, "site1", "default", updateBatch)
                Expect(server.ReceivedRequests()).Should(HaveLen(1))
            })

            Context("And ctx.Done() is closed before the http request responds", func() {
                It("Should return an error", func() {
                    var ctx context.Context
                    var cancel func()

                    ctx, cancel = context.WithCancel(context.Background())

                    server.AppendHandlers(func(w http.ResponseWriter, req *http.Request) {
                        cancel()
                        // Wait some time before returning to ensure that the call to Batch cancels due to the cancelled context
                        // before the server handler responds
                        <-time.After(time.Millisecond * 100)
                    })

                    batchResult := make(chan error)

                    go func() {
                        _, err := client.Batch(ctx, remoteNodeID, 50, "site1", "default", NewUpdateBatch())
                        batchResult <- err
                    }()

                    select {
                    case err := <-batchResult:
                        Expect(err).Should(HaveOccurred())
                    case <-time.After(time.Second):
                        Fail("Batch did not return in time")
                    }

                    Expect(server.ReceivedRequests()).Should(HaveLen(1))
                })
            })

            Context("And the http client request call return an error", func() {
                It("Should return an error", func() {
                    // close the server to force an http connection error
                    server.Close()
                    patch, err := client.Batch(context.TODO(), remoteNodeID, 50, "site1", "default", NewUpdateBatch())

                    Expect(patch).Should(BeNil())
                    Expect(err).Should(HaveOccurred())
                    Expect(server.ReceivedRequests()).Should(HaveLen(0))
                })
            })

            Context("And the http request responds with a 404 status code", func() {
                Context("And the body is a JSON-encoded EBucketDoesNotExist error", func() {
                    It("Should return EBucketDoesNotExist", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, EBucketDoesNotExist.JSON()))
                        patch, err := client.Batch(context.TODO(), remoteNodeID, 50, "site1", "default", NewUpdateBatch())
                        
                        Expect(patch).Should(BeNil())
                        Expect(err).Should(Equal(EBucketDoesNotExist))
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the body is a JSON-encoded ESiteDoesNotExist error", func() {
                    It("Should return ESiteDoesNotExist", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, ESiteDoesNotExist.JSON()))
                        patch, err := client.Batch(context.TODO(), remoteNodeID, 50, "site1", "default", NewUpdateBatch())
                       
                        Expect(patch).Should(BeNil())
                        Expect(err).Should(Equal(ESiteDoesNotExist))
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the body is not a JSON-encoded database error", func() {
                    It("Should return an error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, "asdf"))
                        patch, err := client.Batch(context.TODO(), remoteNodeID, 50, "site1", "default", NewUpdateBatch())
                       
                        Expect(patch).Should(BeNil())
                        Expect(err).Should(HaveOccurred())
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the body is a JSON-encoded database error but is neither EBucketDoesNotExist nor ESiteDoesNotExist", func() {
                    It("Should return an error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, EStorage.JSON()))
                        patch, err := client.Batch(context.TODO(), remoteNodeID, 50, "site1", "default", NewUpdateBatch())
                        
                        Expect(patch).Should(BeNil())
                        Expect(err).Should(HaveOccurred())
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })
            })

            Context("And the http request responds with a 500 status code", func() {
                It("Should return an error", func() {
                    server.AppendHandlers(ghttp.RespondWith(http.StatusInternalServerError, ""))
                    patch, err := client.Batch(context.TODO(), remoteNodeID, 50, "site1", "default", NewUpdateBatch())
                    
                    Expect(patch).Should(BeNil())
                    Expect(err).Should(HaveOccurred())
                    Expect(server.ReceivedRequests()).Should(HaveLen(1))
                })
            })

            Context("And the http request responds with a 200 status code", func() {
                Context("And the response body cannot be parsed as a BatchResult as defined in the routes module", func() {
                    It("Should return an error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "asdf"))
                        patch, err := client.Batch(context.TODO(), remoteNodeID, 50, "site1", "default", NewUpdateBatch())

                        Expect(patch).Should(BeNil())
                        Expect(err).Should(HaveOccurred())
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the response body can be parsed as a BatchResult", func() {
                    var batchResult BatchResult

                    Context("And the NApplied field is 0", func() {
                        BeforeEach(func() {
                            batchResult.NApplied = 0
                        })

                        It("Should return ENoQuorum", func() {
                            encodedBatchResult, err := json.Marshal(batchResult)

                            Expect(err).Should(Not(HaveOccurred()))

                            server.AppendHandlers(ghttp.RespondWith(http.StatusOK, encodedBatchResult))
                            patch, err := client.Batch(context.TODO(), remoteNodeID, 50, "site1", "default", NewUpdateBatch())
                           
                            Expect(patch).Should(BeNil())
                            Expect(err).Should(Equal(ENoQuorum))
                            Expect(server.ReceivedRequests()).Should(HaveLen(1))
                        })
                    })

                    Context("And the NApplied field is not 0", func() {
                        BeforeEach(func() {
                            batchResult.NApplied = 1
                        })

                        It("Should return nil", func() {
                            encodedBatchResult, err := json.Marshal(batchResult)

                            Expect(err).Should(Not(HaveOccurred()))

                            server.AppendHandlers(ghttp.RespondWith(http.StatusOK, encodedBatchResult))
                            patch, err := client.Batch(context.TODO(), remoteNodeID, 50, "site1", "default", NewUpdateBatch())
                           
                            Expect(patch).Should(BeNil())
                            Expect(err).Should(Not(HaveOccurred()))
                            Expect(server.ReceivedRequests()).Should(HaveLen(1))
                        })
                    })
                })
            })

            Context("And the http request responds with a status code other than 200, 404, or 500", func() {
                It("Should return an error", func() {
                    server.AppendHandlers(ghttp.RespondWith(http.StatusForbidden, ""))
                    patch, err := client.Batch(context.TODO(), remoteNodeID, 50, "site1", "default", NewUpdateBatch())
                   
                    Expect(patch).Should(BeNil())
                    Expect(err).Should(HaveOccurred())
                    Expect(server.ReceivedRequests()).Should(HaveLen(1))
                })
            })
        })
    })

    Describe("#Get", func() {
        Context("When the specified nodeID does not refer to a known node", func() {
            It("Should return a nil sibling set array an error", func() {
                siblingSets, err := client.Get(context.TODO(), unknownNodeID, 50, "site1", "default", [][]byte{ })

                Expect(siblingSets).Should(BeNil())
                Expect(err).Should(HaveOccurred())
            })
        })

        Context("When the specified nodeID refers to the local node", func() {
            It("Should invoke Get() on the local node with the same parameters that were passed to it", func() {
                getCalled := make(chan int, 1)
                localNode.getCB = func(ctx context.Context, partition uint64, siteID string, bucket string, keys [][]byte) {
                    Expect(partition).Should(Equal(uint64(50)))
                    Expect(siteID).Should(Equal("site1"))
                    Expect(bucket).Should(Equal("default"))
                    Expect(keys).Should(Equal([][]byte{ []byte("a"), []byte("b"), []byte("c") }))

                    getCalled <- 1
                }

                client.Get(context.TODO(), localNodeID, 50, "site1", "default", [][]byte{ []byte("a"), []byte("b"), []byte("c") })

                select {
                case <-getCalled:
                default:
                    Fail("Did not invoke Get() on local node")
                }
            })

            Context("And if the call to Get() on the local node returns ENoSuchPartition", func() {
                It("Should return a nil sibling set array and an error", func() {
                    localNode.defaultGetError = ENoSuchPartition
                    siblingSets, err := client.Get(context.TODO(), localNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(siblingSets).Should(BeNil())
                    Expect(err).Should(HaveOccurred())
                })
            })

            Context("And if the call to Get() on the local node returns an ENoSuchBucket error", func() {
                It("Should return a nil sibling set array and an EBucketDoesNotExist error", func() {
                    localNode.defaultGetError = ENoSuchBucket
                    siblingSets, err := client.Get(context.TODO(), localNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(siblingSets).Should(BeNil())
                    Expect(err).Should(Equal(EBucketDoesNotExist))
                })
            })

            Context("And if the call to Get() on the local node returns an ENoSuchSite error", func() {
                It("Should return a nil sibling set array and an ESiteDoesNotExist error", func() {
                    localNode.defaultGetError = ENoSuchSite
                    siblingSets, err := client.Get(context.TODO(), localNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(siblingSets).Should(BeNil())
                    Expect(err).Should(Equal(ESiteDoesNotExist))
                })
            })

            Context("And if the call to Get() on the local node returns any other error", func() {
                It("Should return a nil sibling set array and an error", func() {
                    localNode.defaultGetError = errors.New("Some error")
                    siblingSets, err := client.Get(context.TODO(), localNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(siblingSets).Should(BeNil())
                    Expect(err).Should(HaveOccurred())
                })
            })

            Context("And if the call to Get() returns a nil error", func() {
                It("Should return the sibling set array returned by Get() and a nil error", func() {
                    localNode.defaultGetError = nil
                    localNode.defaultGetSiblingSetArray = []*SiblingSet{ nil, nil, nil }
                    siblingSets, err := client.Get(context.TODO(), localNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(siblingSets).Should(Equal(localNode.defaultGetSiblingSetArray))
                    Expect(err).Should(BeNil())
                })
            })
        })

        Context("When the specified nodeID refers to a known node that is not the local node", func() {
            It("Should send a GET request to /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/keys at that node", func() {
                server.AppendHandlers(ghttp.VerifyRequest("GET", "/partitions/50/sites/site1/buckets/default/keys"))
                client.Get(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })
                Expect(server.ReceivedRequests()).Should(HaveLen(1))
            })

            It("Should send a GET request with one 'key' query parameter per entry in the keys array", func() {
                server.AppendHandlers(ghttp.VerifyRequest("GET", "/partitions/50/sites/site1/buckets/default/keys", "key=a&key=b&key=c"))
                client.Get(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ []byte("a"), []byte("b"), []byte("c") })
                Expect(server.ReceivedRequests()).Should(HaveLen(1))
            })

            Context("And one of the keys has a space in it", func() {
                // This test case acts as a regression test to simulates a bug where the keys were not being URL encoded so
                // there was a 400 being returned automatically by the framework for those calls
                It("Should not return an error", func() {
                    var entries []InternalEntry = []InternalEntry{
                        InternalEntry{ Prefix: "", Key: "devicejs.core.resourceGroups.root/Study Room", Siblings: nil },
                    }

                    encodedEntries, err := json.Marshal(&entries)

                    Expect(err).Should(Not(HaveOccurred()))

                    server.AppendHandlers(ghttp.RespondWith(http.StatusOK, encodedEntries))
                    siblingSets, err := client.Get(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ []byte("devicejs.core.resourceGroups.root/Study Room") })

                    Expect(siblingSets).Should(Equal([]*SiblingSet{ nil }))
                    Expect(err).Should(Not(HaveOccurred()))
                    Expect(server.ReceivedRequests()).Should(HaveLen(1))
                })
            })

            Context("And ctx.Done() is closed before the http request responds", func() {
                It("Should return a nil sibling set array and an error", func() {
                    var ctx context.Context
                    var cancel func()

                    ctx, cancel = context.WithCancel(context.Background())

                    server.AppendHandlers(func(w http.ResponseWriter, req *http.Request) {
                        cancel()
                        // Wait some time before returning to ensure that the call to Batch cancels due to the cancelled context
                        // before the server handler responds
                        <-time.After(time.Millisecond * 100)
                    })

                    getReturned := make(chan int)

                    go func() {
                        siblingSets, err := client.Get(ctx, remoteNodeID, 50, "site1", "default", [][]byte{ })

                        Expect(siblingSets).Should(BeNil())
                        Expect(err).Should(HaveOccurred())

                        getReturned <- 1
                    }()

                    select {
                    case <-getReturned:
                    case <-time.After(time.Second):
                        Fail("Get did not return in time")
                    }

                    Expect(server.ReceivedRequests()).Should(HaveLen(1))
                })
            })

            Context("And the http client request call return an error", func() {
                It("Should return a nil sibling set array an error", func() {
                    // close the server to force an http connection error
                    server.Close()
                    siblingSets, err := client.Get(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(siblingSets).Should(BeNil())
                    Expect(err).Should(HaveOccurred())
                    Expect(server.ReceivedRequests()).Should(HaveLen(0))
                })
            })

            Context("And the http request responds with a 404 status code", func() {
                Context("And the body is a JSON-encoded EBucketDoesNotExist error", func() {
                    It("Should return a nil sibling set array and an EBucketDoesNotExist error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, EBucketDoesNotExist.JSON()))
                        siblingSets, err := client.Get(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })

                        Expect(siblingSets).Should(BeNil())
                        Expect(err).Should(Equal(EBucketDoesNotExist))
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the body is a JSON-encoded ESiteDoesNotExist error", func() {
                    It("Should return a nil sibling set array and an ESiteDoesNotExist error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, ESiteDoesNotExist.JSON()))
                        siblingSets, err := client.Get(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })

                        Expect(siblingSets).Should(BeNil())
                        Expect(err).Should(Equal(ESiteDoesNotExist))
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the body is not a JSON-encoded database error", func() {
                    It("Should return a nil sibling set array and an error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, "asdf"))
                        siblingSets, err := client.Get(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })

                        Expect(siblingSets).Should(BeNil())
                        Expect(err).Should(HaveOccurred())
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the body is a JSON-encoded database error but is neither EBucketDoesNotExist nor ESiteDoesNotExist", func() {
                    It("Should return a nil sibling set array and an error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, EStorage.JSON()))
                        siblingSets, err := client.Get(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })

                        Expect(siblingSets).Should(BeNil())
                        Expect(err).Should(HaveOccurred())
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })
            })

            Context("And the http request responds with a 500 status code", func() {
                It("Should return a nil sibling set array and an error", func() {
                    server.AppendHandlers(ghttp.RespondWith(http.StatusInternalServerError, ""))
                    siblingSets, err := client.Get(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(siblingSets).Should(BeNil())
                    Expect(err).Should(HaveOccurred())
                    Expect(server.ReceivedRequests()).Should(HaveLen(1))
                })
            })

            Context("And the http request responds with a 200 status code", func() {
                Context("And the body is a JSON-encoded InternalEntry array as defined in the routes package", func() {
                    It("Should return an array of sibling sets which is a mapping between elements of the InternalEntry array and their contained sibling sets and a nil error", func() {
                        var entries []InternalEntry = []InternalEntry{
                            InternalEntry{ Prefix: "", Key: "a", Siblings: nil },
                            InternalEntry{ Prefix: "", Key: "b", Siblings: nil },
                            InternalEntry{ Prefix: "", Key: "c", Siblings: nil },
                        }

                        encodedEntries, err := json.Marshal(&entries)

                        Expect(err).Should(Not(HaveOccurred()))

                        server.AppendHandlers(ghttp.RespondWith(http.StatusOK, encodedEntries))
                        siblingSets, err := client.Get(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ []byte("a"), []byte("b"), []byte("c") })

                        Expect(siblingSets).Should(Equal([]*SiblingSet{ nil, nil, nil }))
                        Expect(err).Should(Not(HaveOccurred()))
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the body is not a JSON-encoded InternalEntry array", func() {
                    It("Should return a nil sibling set array and an error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusOK, ""))
                        siblingSets, err := client.Get(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ []byte("a"), []byte("b"), []byte("c") })

                        Expect(siblingSets).Should(BeNil())
                        Expect(err).Should(HaveOccurred())
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })
            })

            Context("And the http request responds with a status code other than 200, 404, or 500", func() {
                It("Should return a nil sibling set array and an error", func() {
                    server.AppendHandlers(ghttp.RespondWith(http.StatusForbidden, ""))
                    siblingSets, err := client.Get(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(siblingSets).Should(BeNil())
                    Expect(err).Should(HaveOccurred())
                    Expect(server.ReceivedRequests()).Should(HaveLen(1))
                })
            })
        })
    })

    Describe("#GetMatches", func() {
        Context("When the specified nodeID does not refer to a known node", func() {
            It("Should return a nil sibling set iterator an error", func() {
                iter, err := client.GetMatches(context.TODO(), unknownNodeID, 50, "site1", "default", [][]byte{ })

                Expect(iter).Should(BeNil())
                Expect(err).Should(HaveOccurred())
            })
        })

        Context("When the specified nodeID refers to the local node", func() {
            It("Should invoke GetMatches() on the local node with the same parameters that were passed to it", func() {
                getMatchesCalled := make(chan int, 1)
                localNode.getMatchesCB = func(ctx context.Context, partition uint64, siteID string, bucket string, keys [][]byte) {
                    Expect(partition).Should(Equal(uint64(50)))
                    Expect(siteID).Should(Equal("site1"))
                    Expect(bucket).Should(Equal("default"))
                    Expect(keys).Should(Equal([][]byte{ []byte("a"), []byte("b"), []byte("c") }))

                    getMatchesCalled <- 1
                }

                client.GetMatches(context.TODO(), localNodeID, 50, "site1", "default", [][]byte{ []byte("a"), []byte("b"), []byte("c") })

                select {
                case <-getMatchesCalled:
                default:
                    Fail("Did not invoke GetMatches() on local node")
                }
            })

            Context("And if the call to GetMatches() on the local node returns ENoSuchPartition", func() {
                It("Should return a nil sibling set iterator and an error", func() {
                    localNode.defaultGetMatchesError = ENoSuchPartition
                    iter, err := client.GetMatches(context.TODO(), localNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(iter).Should(BeNil())
                    Expect(err).Should(HaveOccurred())
                })
            })

            Context("And if the call to GetMatches() on the local node returns an ENoSuchBucket error", func() {
                It("Should return a nil sibling set iterator and an EBucketDoesNotExist error", func() {
                    localNode.defaultGetMatchesError = ENoSuchBucket
                    iter, err := client.GetMatches(context.TODO(), localNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(iter).Should(BeNil())
                    Expect(err).Should(Equal(EBucketDoesNotExist))
                })
            })

            Context("And if the call to GetMatches() on the local node returns an ENoSuchSite error", func() {
                It("Should return a nil sibling set iterator and an ESiteDoesNotExist error", func() {
                    localNode.defaultGetMatchesError = ENoSuchSite
                    iter, err := client.GetMatches(context.TODO(), localNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(iter).Should(BeNil())
                    Expect(err).Should(Equal(ESiteDoesNotExist))
                })
            })

            Context("And if the call to GetMatches() on the local node returns any other error", func() {
                It("Should return a nil sibling set iterator and an error", func() {
                    localNode.defaultGetMatchesError = errors.New("Some error")
                    iter, err := client.GetMatches(context.TODO(), localNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(iter).Should(BeNil())
                    Expect(err).Should(HaveOccurred())
                })
            })

            Context("And if the call to GetMatches() returns a nil error", func() {
                It("Should return the sibling set iterator returned by GetMatches() and a nil error", func() {
                    localNode.defaultGetMatchesError = nil
                    localNode.defaultGetMatchesSiblingSetIterator = NewMemorySiblingSetIterator()
                    iter, err := client.GetMatches(context.TODO(), localNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(iter).Should(Equal(localNode.defaultGetMatchesSiblingSetIterator))
                    Expect(err).Should(Not(HaveOccurred()))
                })
            })
        })

        Context("When the specified nodeID refers to a known node that is not the local node", func() {
            It("Should send a GET request to /partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/keys", func() {
                server.AppendHandlers(ghttp.VerifyRequest("GET", "/partitions/50/sites/site1/buckets/default/keys"))
                client.GetMatches(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })
                Expect(server.ReceivedRequests()).Should(HaveLen(1))
            })

            It("Should send a GET request with one 'prefix' query parameter per entry in the keys array", func() {
                server.AppendHandlers(ghttp.VerifyRequest("GET", "/partitions/50/sites/site1/buckets/default/keys", "prefix=a&prefix=b&prefix=c"))
                client.GetMatches(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ []byte("a"), []byte("b"), []byte("c") })
                Expect(server.ReceivedRequests()).Should(HaveLen(1))
            })

            Context("And one of the keys has a space in it", func() {
                // This test case acts as a regression test to simulates a bug where the keys were not being URL encoded so
                // there was a 400 being returned automatically by the framework for those calls
                It("Should not return an error", func() {
                    var entries []InternalEntry = []InternalEntry{
                        InternalEntry{ Prefix: "devicejs.core.resourceGroups.root/Study Room", Key: "devicejs.core.resourceGroups.root/Study Room", Siblings: nil },
                    }

                    encodedEntries, err := json.Marshal(&entries)

                    Expect(err).Should(Not(HaveOccurred()))

                    server.AppendHandlers(ghttp.RespondWith(http.StatusOK, encodedEntries))
                    iter, err := client.GetMatches(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ []byte("devicejs.core.resourceGroups.root/Study Room") })

                    Expect(iter).Should(Not(BeNil()))
                    Expect(err).Should(Not(HaveOccurred()))
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Prefix()).Should(Equal([]byte("devicejs.core.resourceGroups.root/Study Room")))
                    Expect(iter.Key()).Should(Equal([]byte("devicejs.core.resourceGroups.root/Study Room")))
                    Expect(iter.Value()).Should(BeNil())

                    Expect(iter.Next()).Should(BeFalse())
                    Expect(server.ReceivedRequests()).Should(HaveLen(1))
                })
            })

            Context("And ctx.Done() is closed before the http request responds", func() {
                It("Should return a nil sibling set iterator and an error", func() {
                    var ctx context.Context
                    var cancel func()

                    ctx, cancel = context.WithCancel(context.Background())

                    server.AppendHandlers(func(w http.ResponseWriter, req *http.Request) {
                        cancel()
                        // Wait some time before returning to ensure that the call to Batch cancels due to the cancelled context
                        // before the server handler responds
                        <-time.After(time.Millisecond * 100)
                    })

                    getMatchesReturned := make(chan int)

                    go func() {
                        iter, err := client.Get(ctx, remoteNodeID, 50, "site1", "default", [][]byte{ })

                        Expect(iter).Should(BeNil())
                        Expect(err).Should(HaveOccurred())

                        getMatchesReturned <- 1
                    }()

                    select {
                    case <-getMatchesReturned:
                    case <-time.After(time.Second):
                        Fail("GetMatches did not return in time")
                    }

                    Expect(server.ReceivedRequests()).Should(HaveLen(1))
                })
            })

            Context("And the http client request call returns an error", func() {
                It("Should return a nil sibling set iterator an error", func() {
                    // close the server to force an http connection error
                    server.Close()
                    iter, err := client.GetMatches(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(iter).Should(BeNil())
                    Expect(err).Should(HaveOccurred())
                    Expect(server.ReceivedRequests()).Should(HaveLen(0))
                })
            })

            Context("And the http request responds with a 404 status code", func() {
                Context("And the body is a JSON-encoded EBucketDoesNotExist error", func() {
                    It("Should return a nil sibling set iterator and an EBucketDoesNotExist error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, EBucketDoesNotExist.JSON()))
                        iter, err := client.GetMatches(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })

                        Expect(iter).Should(BeNil())
                        Expect(err).Should(Equal(EBucketDoesNotExist))
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the body is a JSON-encoded ESiteDoesNotExist error", func() {
                    It("Should return a nil sibling set iterator and an ESiteDoesNotExist error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, ESiteDoesNotExist.JSON()))
                        iter, err := client.GetMatches(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })

                        Expect(iter).Should(BeNil())
                        Expect(err).Should(Equal(ESiteDoesNotExist))
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the body is not a JSON-encoded database error", func() {
                    It("Should return a nil sibling set iterator and an error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, "asdf"))
                        iter, err := client.GetMatches(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })

                        Expect(iter).Should(BeNil())
                        Expect(err).Should(HaveOccurred())
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the body is a JSON-encoded database error but is neither EBucketDoesNotExist nor ESiteDoesNotExist", func() {
                    It("Should return a nil sibling set iterator and an error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusNotFound, EStorage.JSON()))
                        iter, err := client.GetMatches(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })

                        Expect(iter).Should(BeNil())
                        Expect(err).Should(HaveOccurred())
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })
            })

            Context("And the http request responds with a 500 status code", func() {
                It("Should return a nil sibling set iterator and an error", func() {
                    server.AppendHandlers(ghttp.RespondWith(http.StatusInternalServerError, ""))
                    iter, err := client.GetMatches(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(iter).Should(BeNil())
                    Expect(err).Should(HaveOccurred())
                    Expect(server.ReceivedRequests()).Should(HaveLen(1))
                })
            })

            Context("And the http request responds with a 200 status code", func() {
                Context("And the body is a JSON-encoded InternalEntry array as defined in the routes package", func() {
                    It("Should return a sibling set iterator that allows iteration over all the elements in the InternalEntry array in the order that they appear in the array and a nil error", func() {
                        var entries []InternalEntry = []InternalEntry{
                            InternalEntry{ Prefix: "a", Key: "aa", Siblings: nil },
                            InternalEntry{ Prefix: "a", Key: "ab", Siblings: nil },
                            InternalEntry{ Prefix: "b", Key: "ba", Siblings: nil },
                            InternalEntry{ Prefix: "b", Key: "bb", Siblings: nil },
                            InternalEntry{ Prefix: "c", Key: "ca", Siblings: nil },
                            InternalEntry{ Prefix: "c", Key: "cb", Siblings: nil },
                        }

                        encodedEntries, err := json.Marshal(&entries)

                        Expect(err).Should(Not(HaveOccurred()))

                        server.AppendHandlers(ghttp.RespondWith(http.StatusOK, encodedEntries))
                        iter, err := client.GetMatches(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ []byte("a"), []byte("b"), []byte("c") })

                        Expect(iter).Should(Not(BeNil()))
                        Expect(err).Should(Not(HaveOccurred()))
                        Expect(iter.Next()).Should(BeTrue())
                        Expect(iter.Prefix()).Should(Equal([]byte("a")))
                        Expect(iter.Key()).Should(Equal([]byte("aa")))
                        Expect(iter.Value()).Should(BeNil())

                        Expect(iter.Next()).Should(BeTrue())
                        Expect(iter.Prefix()).Should(Equal([]byte("a")))
                        Expect(iter.Key()).Should(Equal([]byte("ab")))
                        Expect(iter.Value()).Should(BeNil())

                        Expect(iter.Next()).Should(BeTrue())
                        Expect(iter.Prefix()).Should(Equal([]byte("b")))
                        Expect(iter.Key()).Should(Equal([]byte("ba")))
                        Expect(iter.Value()).Should(BeNil())

                        Expect(iter.Next()).Should(BeTrue())
                        Expect(iter.Prefix()).Should(Equal([]byte("b")))
                        Expect(iter.Key()).Should(Equal([]byte("bb")))
                        Expect(iter.Value()).Should(BeNil())

                        Expect(iter.Next()).Should(BeTrue())
                        Expect(iter.Prefix()).Should(Equal([]byte("c")))
                        Expect(iter.Key()).Should(Equal([]byte("ca")))
                        Expect(iter.Value()).Should(BeNil())

                        Expect(iter.Next()).Should(BeTrue())
                        Expect(iter.Prefix()).Should(Equal([]byte("c")))
                        Expect(iter.Key()).Should(Equal([]byte("cb")))
                        Expect(iter.Value()).Should(BeNil())

                        Expect(iter.Next()).Should(BeFalse())
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })

                Context("And the body is not a JSON-encoded InternalEntry array", func() {
                    It("Should return a nil sibling set iterator and an error", func() {
                        server.AppendHandlers(ghttp.RespondWith(http.StatusOK, ""))
                        siblingSets, err := client.GetMatches(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ []byte("a"), []byte("b"), []byte("c") })

                        Expect(siblingSets).Should(BeNil())
                        Expect(err).Should(HaveOccurred())
                        Expect(server.ReceivedRequests()).Should(HaveLen(1))
                    })
                })
            })

            Context("And the http request responds with a status code other than 200, 404, or 500", func() {
                It("Should return a nil sibling set iterator and an error", func() {
                    server.AppendHandlers(ghttp.RespondWith(http.StatusForbidden, ""))
                    iter, err := client.GetMatches(context.TODO(), remoteNodeID, 50, "site1", "default", [][]byte{ })

                    Expect(iter).Should(BeNil())
                    Expect(err).Should(HaveOccurred())
                    Expect(server.ReceivedRequests()).Should(HaveLen(1))
                })
            })
        })
    })
})
