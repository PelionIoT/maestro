package sync_test

import (
    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/merkle"
    rest "github.com/armPelionEdge/devicedb/rest"
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/client"
    . "github.com/armPelionEdge/devicedb/raft"
    . "github.com/armPelionEdge/devicedb/site"
    . "github.com/armPelionEdge/devicedb/partition"
    . "github.com/armPelionEdge/devicedb/sync"

    "github.com/coreos/etcd/raft/raftpb"
    "github.com/gorilla/mux"
    "strconv"
    "time"
    "net"
    "net/http"
    "context"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

type DummySitePool struct {
    sites map[string]Site
    released map[string]int
}

func (dummySitePool *DummySitePool) Acquire(siteID string) Site {
    return dummySitePool.sites[siteID]
}

func (dummySitePool *DummySitePool) Release(siteID string) {
    if dummySitePool.released == nil {
        dummySitePool.released = make(map[string]int)
    }

    dummySitePool.released[siteID] = dummySitePool.released[siteID] + 1
}

func (dummySitePool *DummySitePool) Add(siteID string) {
}

func (dummySitePool *DummySitePool) Remove(siteID string) {
}

func (dummySitePool *DummySitePool) Iterator() SitePoolIterator {
    return nil
}

func (dummySitePool *DummySitePool) LockWrites() {
}

func (dummySitePool *DummySitePool) UnlockWrites() {
}

func (dummySitePool *DummySitePool) LockReads() {
}

func (dummySitePool *DummySitePool) UnlockReads() {
}

type DummySite struct {
    bucketList *BucketList
}

func (dummySite *DummySite) Buckets() *BucketList {
    if dummySite == nil {
        return NewBucketList()
    }

    return dummySite.bucketList
}

func (dummySite *DummySite) Iterator() SiteIterator {
    return nil
}

func (dummySite *DummySite) ID() string {
    return ""
}

func (dummySite *DummySite) LockWrites() {
}

func (dummySite *DummySite) UnlockWrites() {
}

func (dummySite *DummySite) LockReads() {
}

func (dummySite *DummySite) UnlockReads() {
}

type DummyBucket struct {
    name string
    mergeCalls int
    forgetCalls int
    merkleTree *MerkleTree
    syncChildren map[uint32]SiblingSetIterator
}

func (dummyBucket *DummyBucket) Name() string {
    return dummyBucket.name
}

func (dummyBucket *DummyBucket) ShouldReplicateOutgoing(peerID string) bool {
    return false
}

func (dummyBucket *DummyBucket) ShouldReplicateIncoming(peerID string) bool {
    return false
}

func (dummyBucket *DummyBucket) ShouldAcceptWrites(clientID string) bool {
    return false
}

func (dummyBucket *DummyBucket) ShouldAcceptReads(clientID string) bool {
    return false
}

func (dummyBucket *DummyBucket) RecordMetadata() error {
    return nil
}

func (dummyBucket *DummyBucket) RebuildMerkleLeafs() error {
    return nil
}

func (dummyBucket *DummyBucket) MerkleTree() *MerkleTree {
    return dummyBucket.merkleTree
}

func (dummyBucket *DummyBucket) GarbageCollect(tombstonePurgeAge uint64) error {
    return nil
}

func (dummyBucket *DummyBucket) Get(keys [][]byte) ([]*SiblingSet, error) {
    return nil, nil
}

func (dummyBucket *DummyBucket) GetMatches(keys [][]byte) (SiblingSetIterator, error) {
    return nil, nil
}

func (dummyBucket *DummyBucket) GetSyncChildren(nodeID uint32) (SiblingSetIterator, error) {
    return dummyBucket.syncChildren[nodeID], nil
}

func (dummyBucket *DummyBucket) GetAll() (SiblingSetIterator, error) {
    return nil, nil
}

func (dummyBucket *DummyBucket) Watch(ctx context.Context, keys [][]byte, prefixes [][]byte, localVersion uint64, ch chan Row) {

}

func (dummyBucket *DummyBucket) LockReads() {
}

func (dummyBucket *DummyBucket) LockWrites() {
}

func (dummyBucket *DummyBucket) UnlockReads() {
}

func (dummyBucket *DummyBucket) UnlockWrites() {
}

func (dummyBucket *DummyBucket) Forget(keys [][]byte) error {
    dummyBucket.forgetCalls++

    return nil
}

func (dummyBucket *DummyBucket) Batch(batch *UpdateBatch) (map[string]*SiblingSet, error) {
    return nil, nil
}

func (dummyBucket *DummyBucket) Merge(siblingSets map[string]*SiblingSet) error {
    dummyBucket.mergeCalls++

    return nil
}

type MockConfigController struct {
    clusterController *ClusterController
    defaultClusterCommandResponse error
    clusterCommandCB func(ctx context.Context, commandBody interface{})
}

func NewMockConfigController(clusterController *ClusterController) *MockConfigController {
    return &MockConfigController{
        clusterController: clusterController,
    }
}

func (configController *MockConfigController) LogDump() (raftpb.Snapshot, []raftpb.Entry, error) {
    return raftpb.Snapshot{}, nil, nil
}

func (configController *MockConfigController) AddNode(ctx context.Context, nodeConfig NodeConfig) error {
    return nil
}

func (configController *MockConfigController) ReplaceNode(ctx context.Context, replacedNodeID uint64, replacementNodeID uint64) error {
    return nil
}

func (configController *MockConfigController) RemoveNode(ctx context.Context, nodeID uint64) error {
    return nil
}

func (configController *MockConfigController) ClusterCommand(ctx context.Context, commandBody interface{}) error {
    configController.notifyClusterCommand(ctx, commandBody)
    return configController.defaultClusterCommandResponse
}

func (configController *MockConfigController) SetDefaultClusterCommandResponse(err error) {
    configController.defaultClusterCommandResponse = err
}

func (configController *MockConfigController) onClusterCommand(cb func(ctx context.Context, commandBody interface{})) {
    configController.clusterCommandCB = cb
}

func (configController *MockConfigController) notifyClusterCommand(ctx context.Context, commandBody interface{}) {
    configController.clusterCommandCB(ctx, commandBody)
}

func (configController *MockConfigController) OnLocalUpdates(cb func(deltas []ClusterStateDelta)) {
}

func (configController *MockConfigController) OnClusterSnapshot(cb func(snapshotIndex uint64, snapshotId string)) {
}

func (configController *MockConfigController) ClusterController() *ClusterController {
    return configController.clusterController
}

func (configController *MockConfigController) SetClusterController(clusterController *ClusterController) {
    configController.clusterController = clusterController
}

func (configController *MockConfigController) Start() error {
    return nil
}

func (configController *MockConfigController) Stop() {
}

func (configController *MockConfigController) CancelProposals() {
}

type TestHTTPServer struct {
    port int
    r *mux.Router
    httpServer *http.Server
    listener net.Listener
    done chan int
}

func NewTestHTTPServer(port int) *TestHTTPServer {
    httpServer := &TestHTTPServer{
        port: port,
        done: make(chan int),
        r: mux.NewRouter(),
    }

    return httpServer
}

func (s *TestHTTPServer) Start() error {
    s.httpServer = &http.Server{
        Handler: s.r,
        WriteTimeout: 15 * time.Second,
        ReadTimeout: 15 * time.Second,
    }
    
    var listener net.Listener
    var err error

    listener, err = net.Listen("tcp", "localhost:" + strconv.Itoa(s.port))
    
    if err != nil {
        return err
    }
    
    s.listener = listener

    go func() {
        s.httpServer.Serve(s.listener)
        s.done <- 1
    }()

    return nil
}

func (s *TestHTTPServer) Stop() {
    s.listener.Close()
    <-s.done
}

func (s *TestHTTPServer) Router() *mux.Router {
    return s.r
}

var _ = Describe("BucketProxy", func() {
    Describe("RelayBucketProxyFactory", func() {
        Describe("#CreateBucketProxy", func() {
            Specify("If the specified bucket name is not valid it should return an ENoLocalBucket error", func() {
                relayBucketProxyFactory := &RelayBucketProxyFactory{
                    SitePool: &DummySitePool{
                        sites: map[string]Site{
                            "": &DummySite{ 
                                bucketList: NewBucketList(),
                            },
                        },
                    },
                }

                bucketProxy, err := relayBucketProxyFactory.CreateBucketProxy("cloud", "bucketName")

                Expect(bucketProxy).Should(BeNil())
                Expect(err).Should(Equal(ENoLocalBucket))
            })

            Specify("If the specified bucket name valid it should return a local bucket proxy for that bucket", func() {
                bucket := &DummyBucket{
                    name: "dummy",
                }

                bucketList := NewBucketList()
                bucketList.AddBucket(bucket)

                relayBucketProxyFactory := &RelayBucketProxyFactory{
                    SitePool: &DummySitePool{
                        sites: map[string]Site{
                            "": &DummySite{ 
                                bucketList: bucketList,
                            },
                        },
                    },
                }

                bucketProxy, err := relayBucketProxyFactory.CreateBucketProxy("WWRL000001", "dummy")
                _, ok := bucketProxy.(*RelayBucketProxy)

                Expect(ok).Should(BeTrue())
                Expect(bucketProxy).Should(Not(BeNil()))
                Expect(err).Should(BeNil())
            })
        })
    })

    Describe("RelayBucketProxy", func() {
        Describe("#Name", func() {
            Specify("Should return the result of Name of the Bucket it is a proxy for", func() {
                localBucketProxy := &RelayBucketProxy{
                    Bucket: &DummyBucket{
                        name: "default",
                    },
                }

                Expect(localBucketProxy.Name()).Should(Equal("default"))
            })
        })

        Describe("#MerkleTree", func() {
            Specify("Should return a DirectMerkleTreeProxy which proxy for the MerkleTree of the Bucket", func() {
                merkleTree, _ := NewMerkleTree(MerkleMinDepth)
                localBucketProxy := &RelayBucketProxy{
                    Bucket: &DummyBucket{
                        name: "default",
                        merkleTree: merkleTree,
                    },
                }

                merkleTreeProxy := localBucketProxy.MerkleTree()
                _, ok := merkleTreeProxy.(*DirectMerkleTreeProxy)

                Expect(ok).Should(BeTrue())
                Expect(merkleTreeProxy.(*DirectMerkleTreeProxy).MerkleTree()).Should(Equal(merkleTree))
            })
        })

        Describe("#GetSyncChildren", func() {
            Specify("Should return the result of GetSyncChildren of the Bucket it is a proxy for", func() {
               localBucketProxy := &RelayBucketProxy{
                    Bucket: &DummyBucket{
                        name: "default",
                    },
                }

                iter, err := localBucketProxy.GetSyncChildren(0)

                Expect(iter).Should(BeNil())
                Expect(err).Should(BeNil())
            })
        })

        Describe("#Merge", func() {
            Specify("Should call the Merge() method of the Bucekt it is a proxy for", func() {
                localBucketProxy := &RelayBucketProxy{
                    Bucket: &DummyBucket{
                        name: "default",
                    },
                }

                Expect(localBucketProxy.Merge(map[string]*SiblingSet{ "a": NewSiblingSet(map[*Sibling]bool{ }) })).Should(BeNil())
                Expect(localBucketProxy.Bucket.(*DummyBucket).mergeCalls).Should(Equal(1))
            })
        })

        Describe("#Forget", func() {
            Specify("Should call the Forget() method of the Bucket it is a proxy for", func() {
                localBucketProxy := &RelayBucketProxy{
                    Bucket: &DummyBucket{
                        name: "default",
                    },
                }

                Expect(localBucketProxy.Forget([][]byte{ []byte("a") })).Should(BeNil())
                Expect(localBucketProxy.Bucket.(*DummyBucket).forgetCalls).Should(Equal(1))
            })
        })

        Describe("#Close", func() {
            Specify("Should release the associated site in the site pool", func() {
                localBucketProxy := &RelayBucketProxy{
                    SitePool: &DummySitePool{
                    },
                    Bucket: &DummyBucket{
                        name: "default",
                    },
                    SiteID: "site1",
                }

                localBucketProxy.Close()

                Expect(localBucketProxy.SitePool.(*DummySitePool).released["site1"]).Should(Equal(1))
            })
        })
    })

    Describe("CloudRemoteBucketProxy", func() {
        Describe("#Name", func() {
            Specify("Should return the result of Name of the Bucket it is a proxy for", func() {
                remoteBucketProxy := &CloudRemoteBucketProxy{
                    BucketName: "default",
                }

                Expect(remoteBucketProxy.Name()).Should(Equal("default"))
            })
        })

        Describe("#MerkleTree", func() {
            var httpServer *TestHTTPServer
            var bucketSyncHTTP *BucketSyncHTTP
            var remoteBucket Bucket
            var remoteBucketProxy *CloudRemoteBucketProxy

            BeforeEach(func() {
                httpServer = NewTestHTTPServer(9000)
                bucketList := NewBucketList()
                merkleTree, _ := NewMerkleTree(MerkleMinDepth)
                merkleTree.SetNodeHash(merkleTree.RootNode(), Hash{}.SetLow(60).SetHigh(60))
                remoteBucket = &DummyBucket{
                    name: "default",
                    merkleTree: merkleTree,
                    syncChildren: map[uint32]SiblingSetIterator{
                        merkleTree.RootNode(): &CloudResponderMerkleNodeIterator{
                            CurrentIndex: -1,
                            MerkleKeys: rest.MerkleKeys{
                                Keys: []rest.Key{
                                    rest.Key{ Key: "a", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                                    rest.Key{ Key: "b", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                                },
                            },
                        },
                    },
                }

                bucketList.AddBucket(remoteBucket)
                site1 := &DummySite{
                    bucketList: bucketList,
                }

                partitions := NewDefaultPartitionPool()
                partitions.Add(NewDefaultPartition(0, &DummySitePool{
                    sites: map[string]Site{
                        "site1": site1,
                    },
                }))

                clusterController := &ClusterController{ PartitioningStrategy: &SimplePartitioningStrategy{ } }
                clusterController.AddNode(ClusterAddNodeBody{ NodeID: 1, NodeConfig: NodeConfig{ Capacity: 1, Address: PeerAddress{ NodeID: 1 } } })

                bucketSyncHTTP = &BucketSyncHTTP{
                    PartitionPool: partitions,
                    ClusterConfigController: NewMockConfigController(clusterController),
                }

                bucketSyncHTTP.Attach(httpServer.Router())
                httpServer.Start()
            })

            AfterEach(func() {
                httpServer.Stop()
            })

            Context("when there is an error performing the request to the peer specified by PeerAddress", func() {
                BeforeEach(func() {
                    remoteBucketProxy = &CloudRemoteBucketProxy{
                        SiteID: "site1",
                        BucketName: "default",
                        Client: *NewClient(ClientConfig{ }),
                        PeerAddress: PeerAddress{ Host: "localhost", Port: 9001 }, // wrong port to force error
                    }
                })

                It("should return a CloudResponderMerkleTreeProxy whose Error() method returns an error", func() {
                    merkleTreeProxy := remoteBucketProxy.MerkleTree()
                    _, ok := merkleTreeProxy.(*CloudResponderMerkleTreeProxy)

                    Expect(ok).Should(BeTrue())
                    Expect(merkleTreeProxy.Error()).Should(Not(BeNil()))
                })
            })

            Context("when the request to the peer specified by PeerAddress works", func() {
                BeforeEach(func() {
                    remoteBucketProxy = &CloudRemoteBucketProxy{
                        SiteID: "site1",
                        BucketName: "default",
                        Client: *NewClient(ClientConfig{ }),
                        PeerAddress: PeerAddress{ Host: "localhost", Port: 9000 },
                    }
                })

                It("should return a CloudResponderMerkleTreeProxy whose Error() method returns nil", func() {
                    merkleTreeProxy := remoteBucketProxy.MerkleTree()
                    _, ok := merkleTreeProxy.(*CloudResponderMerkleTreeProxy)

                    Expect(ok).Should(BeTrue())
                    Expect(merkleTreeProxy.Error()).Should(BeNil())
                })

                It("should return a CloudResponderMerkleTreeProxy whose RootNode() method returns the root node ID of the remote merkle tree", func() {
                    merkleTreeProxy := remoteBucketProxy.MerkleTree()

                    Expect(merkleTreeProxy.RootNode()).Should(Equal(remoteBucket.MerkleTree().RootNode()))
                })

                It("should return a CloudResponderMerkleTreeProxy whose Depth() method returns the depth of the remote merkle tree", func() {
                    merkleTreeProxy := remoteBucketProxy.MerkleTree()

                    Expect(merkleTreeProxy.Depth()).Should(Equal(remoteBucket.MerkleTree().Depth()))
                })

                It("should return a CloudResponderMerkleTreeProxy whose NodeLimit() method returns the node limit of the remote merkle tree", func() {
                    merkleTreeProxy := remoteBucketProxy.MerkleTree()

                    Expect(merkleTreeProxy.NodeLimit()).Should(Equal(remoteBucket.MerkleTree().NodeLimit()))
                })

                It("should return a CloudResponderMerkleTreeProxy whose NodeHash() method returns the same node hash as the remote merkle tree", func() {
                    merkleTreeProxy := remoteBucketProxy.MerkleTree()

                    Expect(merkleTreeProxy.NodeHash(remoteBucket.MerkleTree().RootNode())).Should(Equal(remoteBucket.MerkleTree().NodeHash(remoteBucket.MerkleTree().RootNode())))
                })

                It("should return a CloudResponderMerkleTreeProxy whose TranslateNode() method returns the same translation as the remote merkle tree", func() {
                    merkleTreeProxy := remoteBucketProxy.MerkleTree()

                    Expect(merkleTreeProxy.TranslateNode(remoteBucket.MerkleTree().RootNode(), MerkleMaxDepth)).Should(Equal(remoteBucket.MerkleTree().TranslateNode(remoteBucket.MerkleTree().RootNode(), MerkleMaxDepth)))
                })
            })
        })

        Describe("#GetSyncChildren", func() {
            var httpServer *TestHTTPServer
            var bucketSyncHTTP *BucketSyncHTTP
            var remoteBucket Bucket
            var remoteBucketProxy *CloudRemoteBucketProxy

            BeforeEach(func() {
                httpServer = NewTestHTTPServer(9000)
                bucketList := NewBucketList()
                merkleTree, _ := NewMerkleTree(MerkleMinDepth)
                merkleTree.SetNodeHash(merkleTree.RootNode(), Hash{}.SetLow(60).SetHigh(60))
                remoteBucket = &DummyBucket{
                    name: "default",
                    merkleTree: merkleTree,
                    syncChildren: map[uint32]SiblingSetIterator{
                        merkleTree.RootNode(): &CloudResponderMerkleNodeIterator{
                            CurrentIndex: -1,
                            MerkleKeys: rest.MerkleKeys{
                                Keys: []rest.Key{
                                    rest.Key{ Key: "a", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                                    rest.Key{ Key: "b", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                                },
                            },
                        },
                    },
                }

                bucketList.AddBucket(remoteBucket)

                site1 := &DummySite{
                    bucketList: bucketList,
                }

                partitions := NewDefaultPartitionPool()
                partitions.Add(NewDefaultPartition(0, &DummySitePool{
                    sites: map[string]Site{
                        "site1": site1,
                    },
                }))

                clusterController := &ClusterController{ PartitioningStrategy: &SimplePartitioningStrategy{ } }
                clusterController.AddNode(ClusterAddNodeBody{ NodeID: 1, NodeConfig: NodeConfig{ Capacity: 1, Address: PeerAddress{ NodeID: 1 } } })

                bucketSyncHTTP = &BucketSyncHTTP{
                    PartitionPool: partitions,
                    ClusterConfigController: NewMockConfigController(clusterController),
                }

                bucketSyncHTTP.Attach(httpServer.Router())
                httpServer.Start()
            })

            AfterEach(func() {
                httpServer.Stop()
            })

            Context("when there is an error performing the request to the peer specified by PeerAddress", func() {
                BeforeEach(func() {
                    remoteBucketProxy = &CloudRemoteBucketProxy{
                        SiteID: "site1",
                        BucketName: "default",
                        Client: *NewClient(ClientConfig{ }),
                        PeerAddress: PeerAddress{ Host: "localhost", Port: 9001 }, // wrong port to force error
                    }
                })

                It("should return an error", func() {
                    siblingSetIter, err := remoteBucketProxy.GetSyncChildren(remoteBucket.MerkleTree().RootNode())

                    Expect(siblingSetIter).Should(BeNil())
                    Expect(err).Should(Not(BeNil()))
                })
            })

            Context("when the request to the peer specified by PeerAddress works", func() {
                BeforeEach(func() {
                    remoteBucketProxy = &CloudRemoteBucketProxy{
                        SiteID: "site1",
                        BucketName: "default",
                        Client: *NewClient(ClientConfig{ }),
                        PeerAddress: PeerAddress{ Host: "localhost", Port: 9000 },
                    }
                })

                It("should return no error and a valid sibling set iterator", func() {
                    siblingSetIter, err := remoteBucketProxy.GetSyncChildren(remoteBucket.MerkleTree().RootNode())

                    Expect(siblingSetIter).Should(Not(BeNil()))
                    Expect(err).Should(BeNil())
                    Expect(siblingSetIter.Next()).Should(BeTrue())
                    Expect(siblingSetIter.Key()).Should(Equal([]byte("a")))
                    Expect(siblingSetIter.Next()).Should(BeTrue())
                    Expect(siblingSetIter.Key()).Should(Equal([]byte("b")))
                })
            })
        })

        Describe("#Close", func() {
        })
    })

    Describe("CloudResponderMerkleNodeIterator", func() {
        Describe("#Next", func() {
            Context("when the current index is >= the number of merkle keys - 1", func () {
                var remoteMerkleIterator *CloudResponderMerkleNodeIterator

                BeforeEach(func() {
                    remoteMerkleIterator = &CloudResponderMerkleNodeIterator{
                        MerkleKeys: rest.MerkleKeys{
                            Keys: []rest.Key{ },
                        },
                        CurrentIndex: -1,
                    }
                })

                It("Should not advance the current index", func() {
                    remoteMerkleIterator.Next()
                    Expect(remoteMerkleIterator.CurrentIndex).Should(Equal(0))
                })

                It("Should return false", func() {
                    Expect(remoteMerkleIterator.Next()).Should(BeFalse())
                })
            })

            Context("when the current index is < the number of merkle keys - 1", func () {
                var remoteMerkleIterator *CloudResponderMerkleNodeIterator

                BeforeEach(func() {
                    remoteMerkleIterator = &CloudResponderMerkleNodeIterator{
                        MerkleKeys: rest.MerkleKeys{
                            Keys: []rest.Key{ 
                                rest.Key{ Key: "a", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                                rest.Key{ Key: "b", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                            },
                        },
                        CurrentIndex: -1,
                    }
                })

                It("Should advance the current index", func() {
                    remoteMerkleIterator.Next()
                    Expect(remoteMerkleIterator.CurrentIndex).Should(Equal(0))
                })

                It("Should return false", func() {
                    Expect(remoteMerkleIterator.Next()).Should(BeTrue())
                })
            })
        })

        Describe("#Prefix", func() {
            It("Should return nil", func() {
                remoteMerkleIterator := &CloudResponderMerkleNodeIterator{ }

                Expect(remoteMerkleIterator.Prefix()).Should(BeNil())
            })
        })

        Describe("#Key", func() {
            Context("when the current index < 0", func() {
                var remoteMerkleIterator *CloudResponderMerkleNodeIterator

                BeforeEach(func() {
                    remoteMerkleIterator = &CloudResponderMerkleNodeIterator{
                        MerkleKeys: rest.MerkleKeys{
                            Keys: []rest.Key{ 
                                rest.Key{ Key: "a", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                                rest.Key{ Key: "b", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                            },
                        },
                        CurrentIndex: -1,
                    }
                })

                It("should return nil", func() {
                    Expect(remoteMerkleIterator.Key()).Should(BeNil())
                })
            })

            Context("when the current index == 0 and there are no elements in the Keys list", func() {
                var remoteMerkleIterator *CloudResponderMerkleNodeIterator

                BeforeEach(func() {
                    remoteMerkleIterator = &CloudResponderMerkleNodeIterator{
                        MerkleKeys: rest.MerkleKeys{
                            Keys: []rest.Key{},
                        },
                        CurrentIndex: 0,
                    }
                })

                It("should return nil", func() {
                    Expect(remoteMerkleIterator.Key()).Should(BeNil())
                })
            })

            Context("when the current index >= the number of elements in the keys list", func() {
                var remoteMerkleIterator *CloudResponderMerkleNodeIterator

                BeforeEach(func() {
                    remoteMerkleIterator = &CloudResponderMerkleNodeIterator{
                        MerkleKeys: rest.MerkleKeys{
                            Keys: []rest.Key{
                                rest.Key{ Key: "a", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                                rest.Key{ Key: "b", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                            },
                        },
                        CurrentIndex: 2,
                    }
                })

                It("should return nil", func() {
                    Expect(remoteMerkleIterator.Key()).Should(BeNil())
                })
            })

            Context("when the current index exists in the keys list", func() {
                var remoteMerkleIterator *CloudResponderMerkleNodeIterator

                BeforeEach(func() {
                    remoteMerkleIterator = &CloudResponderMerkleNodeIterator{
                        MerkleKeys: rest.MerkleKeys{
                            Keys: []rest.Key{
                                rest.Key{ Key: "a", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                                rest.Key{ Key: "b", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                            },
                        },
                        CurrentIndex: 0,
                    }
                })

                It("should return the key at the current index", func() {
                    Expect(remoteMerkleIterator.Key()).Should(Equal([]byte("a")))
                })
            })
        })

        Describe("#Value", func() {
            Context("when the current index < 0", func() {
                var remoteMerkleIterator *CloudResponderMerkleNodeIterator

                BeforeEach(func() {
                    remoteMerkleIterator = &CloudResponderMerkleNodeIterator{
                        MerkleKeys: rest.MerkleKeys{
                            Keys: []rest.Key{ 
                                rest.Key{ Key: "a", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                                rest.Key{ Key: "b", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                            },
                        },
                        CurrentIndex: -1,
                    }
                })

                It("should return nil", func() {
                    Expect(remoteMerkleIterator.Value()).Should(BeNil())
                })
            })

            Context("when the current index == 0 and there are no elements in the Keys list", func() {
                var remoteMerkleIterator *CloudResponderMerkleNodeIterator

                BeforeEach(func() {
                    remoteMerkleIterator = &CloudResponderMerkleNodeIterator{
                        MerkleKeys: rest.MerkleKeys{
                            Keys: []rest.Key{},
                        },
                        CurrentIndex: 0,
                    }
                })

                It("should return nil", func() {
                    Expect(remoteMerkleIterator.Value()).Should(BeNil())
                })
            })

            Context("when the current index >= the number of elements in the keys list", func() {
                var remoteMerkleIterator *CloudResponderMerkleNodeIterator

                BeforeEach(func() {
                    remoteMerkleIterator = &CloudResponderMerkleNodeIterator{
                        MerkleKeys: rest.MerkleKeys{
                            Keys: []rest.Key{
                                rest.Key{ Key: "a", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                                rest.Key{ Key: "b", Value: NewSiblingSet(map[*Sibling]bool{ }) },
                            },
                        },
                        CurrentIndex: 2,
                    }
                })

                It("should return nil", func() {
                    Expect(remoteMerkleIterator.Value()).Should(BeNil())
                })
            })

            Context("when the current index exists in the keys list", func() {
                var remoteMerkleIterator *CloudResponderMerkleNodeIterator
                var siblingSetA *SiblingSet
                var siblingSetB *SiblingSet

                BeforeEach(func() {
                    siblingSetA = NewSiblingSet(map[*Sibling]bool{ })
                    siblingSetB = NewSiblingSet(map[*Sibling]bool{ })

                    remoteMerkleIterator = &CloudResponderMerkleNodeIterator{
                        MerkleKeys: rest.MerkleKeys{
                            Keys: []rest.Key{
                                rest.Key{ Key: "a", Value: siblingSetA },
                                rest.Key{ Key: "b", Value: siblingSetB },
                            },
                        },
                        CurrentIndex: 0,
                    }
                })

                It("should return the key at the current index", func() {
                    Expect(remoteMerkleIterator.Value()).Should(Equal(siblingSetA))
                })
            })
        })

        Describe("#Error", func() {
            It("Should return nil", func() {
                remoteMerkleIterator := &CloudResponderMerkleNodeIterator{ }

                Expect(remoteMerkleIterator.Error()).Should(BeNil())
            })
        })

        Describe("iteration through a list of keys", func() {
            Context("The list of keys is not empty", func() {
                Specify("calls to Next() should return true twice: once for each item in the list", func() {
                    siblingSetA := NewSiblingSet(map[*Sibling]bool{ })
                    siblingSetB := NewSiblingSet(map[*Sibling]bool{ })

                    remoteMerkleIterator := &CloudResponderMerkleNodeIterator{
                        MerkleKeys: rest.MerkleKeys{
                            Keys: []rest.Key{
                                rest.Key{ Key: "a", Value: siblingSetA },
                                rest.Key{ Key: "b", Value: siblingSetB },
                            },
                        },
                        CurrentIndex: -1,
                    }

                    Expect(remoteMerkleIterator.Next()).Should(BeTrue())
                    Expect(remoteMerkleIterator.Key()).Should(Equal([]byte("a")))
                    Expect(remoteMerkleIterator.Value()).Should(Equal(siblingSetA))
                    Expect(remoteMerkleIterator.Next()).Should(BeTrue())
                    Expect(remoteMerkleIterator.Key()).Should(Equal([]byte("b")))
                    Expect(remoteMerkleIterator.Value()).Should(Equal(siblingSetB))
                    Expect(remoteMerkleIterator.Next()).Should(BeFalse())
                    Expect(remoteMerkleIterator.Key()).Should(BeNil())
                    Expect(remoteMerkleIterator.Value()).Should(BeNil())
                })
            })

            Context("The list of keys is empty", func() {
                Specify("calls to Next() should return false and calls to Key() and Value() should return nil", func() {
                    remoteMerkleIterator := &CloudResponderMerkleNodeIterator{
                        MerkleKeys: rest.MerkleKeys{
                            Keys: []rest.Key{ },
                        },
                        CurrentIndex: -1,
                    }

                    Expect(remoteMerkleIterator.Next()).Should(BeFalse())
                    Expect(remoteMerkleIterator.Key()).Should(BeNil())
                    Expect(remoteMerkleIterator.Value()).Should(BeNil())
                })
            })
        })
    })
})