package node_test

import (
    "crypto/tls"
    "crypto/x509"
    "errors"
    "io/ioutil"
    "time"

    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/node"
    . "github.com/armPelionEdge/devicedb/raft"
    . "github.com/armPelionEdge/devicedb/server"
    . "github.com/armPelionEdge/devicedb/storage"
    . "github.com/armPelionEdge/devicedb/util"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

func loadCerts(id string) (*tls.Config, *tls.Config, error) {
    clientCertificate, err := tls.LoadX509KeyPair("../test_certs/" + id + ".client.cert.pem", "../test_certs/" + id + ".client.key.pem")
    
    if err != nil {
        return nil, nil, err
    }
    
    serverCertificate, err := tls.LoadX509KeyPair("../test_certs/" + id + ".server.cert.pem", "../test_certs/" + id + ".server.key.pem")
    
    if err != nil {
        return nil, nil, err
    }
    
    rootCAChain, err := ioutil.ReadFile("../test_certs/ca-chain.cert.pem")
    
    if err != nil {
        return nil, nil, err
    }
    
    rootCAs := x509.NewCertPool()
    if !rootCAs.AppendCertsFromPEM(rootCAChain) {
        return nil, nil, errors.New("Could not append certs to chain")
    }
    
    var serverTLSConfig = &tls.Config{
        Certificates: []tls.Certificate{ serverCertificate },
        ClientCAs: rootCAs,
    }
    var clientTLSConfig = &tls.Config{
        Certificates: []tls.Certificate{ clientCertificate },
        RootCAs: rootCAs,
    }
    
    return serverTLSConfig, clientTLSConfig, nil
}

func tempServer(internalPort int, externalPort int) *CloudServer {
    // Use relay certificates in place of some cloud certs
    serverTLS, _, err := loadCerts("WWRL000000")

    Expect(err).Should(BeNil())

    return NewCloudServer(CloudServerConfig{
        ExternalHost: "localhost",
        ExternalPort: externalPort,
        InternalHost: "localhost",
        InternalPort: internalPort,
        NodeID: 1,
        RelayTLSConfig: serverTLS,
    })
}

func tempStorageDriver() StorageDriver {
    return NewLevelDBStorageDriver("/tmp/testdb-" + RandomString(), nil)
}

var _ = Describe("ClusterNode", func() {
    var throwAwayStorageDriver StorageDriver
    var throwAwayCloudServer *CloudServer

    BeforeEach(func() {
        throwAwayStorageDriver = tempStorageDriver()
        throwAwayCloudServer = tempServer(8080, 9090)
    })

    AfterEach(func() {
    })

    Describe("#Start", func() {
        Context("The node was already assigned an ID", func() {
            It("Should use the same ID for this node", func() {
                clusterNode := New(ClusterNodeConfig{
                    StorageDriver: throwAwayStorageDriver,
                    CloudServer: throwAwayCloudServer,
                    Capacity: 1,
                })

                memoryStore := NewRaftMemoryStorage()
                clusterNode.UseRaftStore(memoryStore)
                memoryStore.SetNodeID(45)

                startResult := make(chan error)
                nodeInitialized := make(chan int)

                clusterNode.OnInitialized(func() {
                    defer GinkgoRecover()

                    nodeInitialized <- 1
                })

                go func() {
                    options := NodeInitializationOptions{
                        StartCluster: true,
                        ClusterSettings: ClusterSettings{
                            Partitions: 4,
                            ReplicationFactor: 3,
                        },
                    }

                    startResult <- clusterNode.Start(options)
                }()

                select {
                case <-nodeInitialized:
                case <-startResult:
                    Fail("Node stopped prematurely")
                case <-time.After(time.Second * 100):
                    Fail("Test timed out")
                }

                clusterNode.Stop()

                select {
                case <-startResult:
                case <-time.After(time.Second * 100):
                    Fail("Test timed out")
                }

                generatednodeID, _ := memoryStore.NodeID()
                Expect(generatednodeID).Should(Equal(uint64(45)))
            })
        })

        Context("The node has not been assigned an ID yet", func() {
            It("Should generate a new ID for this node", func() {
                clusterNode := New(ClusterNodeConfig{
                    StorageDriver: throwAwayStorageDriver,
                    CloudServer: throwAwayCloudServer,
                    Capacity: 1,
                })

                memoryStore := NewRaftMemoryStorage()
                clusterNode.UseRaftStore(memoryStore)

                nodeID, _ := memoryStore.NodeID()
                Expect(nodeID).Should(Equal(uint64(0)))

                startResult := make(chan error)
                nodeInitialized := make(chan int)

                clusterNode.OnInitialized(func() {
                    defer GinkgoRecover()

                    nodeInitialized <- 1
                })

                go func() {
                    options := NodeInitializationOptions{
                        StartCluster: true,
                        ClusterSettings: ClusterSettings{
                            Partitions: 4,
                            ReplicationFactor: 3,
                        },
                    }

                    startResult <- clusterNode.Start(options)
                }()

                select {
                case <-nodeInitialized:
                case <-startResult:
                    Fail("Node stopped prematurely")
                case <-time.After(time.Second * 100):
                    Fail("Test timed out")
                }

                clusterNode.Stop()

                select {
                case <-startResult:
                case <-time.After(time.Second * 100):
                    Fail("Test timed out")
                }

                generatednodeID, _ := memoryStore.NodeID()
                Expect(generatednodeID).Should(Not(Equal(uint64(0))))
            })
        })

        Context("The node is not yet part of a cluster", func() {
            Context("And the initialization options are set to create a new cluster", func() {
                It("should create a new cluster and add the node to that cluster", func() {
                    clusterNode := New(ClusterNodeConfig{
                        StorageDriver: throwAwayStorageDriver,
                        CloudServer: throwAwayCloudServer,
                        Capacity: 1,
                    })

                    startResult := make(chan error)
                    nodeInitialized := make(chan int)

                    clusterNode.OnInitialized(func() {
                        defer GinkgoRecover()

                        Expect(clusterNode.ClusterConfigController().ClusterController().State.ClusterSettings.Partitions).Should(Equal(uint64(4)))
                        Expect(clusterNode.ClusterConfigController().ClusterController().State.ClusterSettings.ReplicationFactor).Should(Equal(uint64(3)))
                        Expect(len(clusterNode.ClusterConfigController().ClusterController().State.Nodes)).Should(Equal(1))

                        nodeInitialized <- 1
                    })

                    go func() {
                        options := NodeInitializationOptions{
                            StartCluster: true,
                            ClusterSettings: ClusterSettings{
                                Partitions: 4,
                                ReplicationFactor: 3,
                            },
                        }

                        startResult <- clusterNode.Start(options)
                    }()

                    // Wait until we hear both replication factor set and partitions set
                    select {
                    case <-nodeInitialized:
                    case <-startResult:
                        Fail("Server stopped before initialization")
                    case <-time.After(time.Second * 100):
                        Fail("Test timed out")
                    }

                    // Wait for node to shut down
                    clusterNode.Stop()

                    select {
                    case <-startResult:
                    case <-time.After(time.Second):
                        Fail("Test timed out")
                    }
                })
            })

            Context("And the initialization options are set to join an existing cluster", func() {
                It("should add the node to that cluster", func() {
                    seedNodeStorageDriver := tempStorageDriver()
                    seedNodeServer := tempServer(8080, 9090)
                    newNodeStorageDriver := tempStorageDriver()
                    newNodeServer := tempServer(8181, 9191)

                    seedNode := New(ClusterNodeConfig{
                        StorageDriver: seedNodeStorageDriver,
                        CloudServer: seedNodeServer,
                        Capacity: 1,
                    })

                    newNode := New(ClusterNodeConfig{
                        StorageDriver: newNodeStorageDriver,
                        CloudServer: newNodeServer,
                        Capacity: 1,
                    })

                    seedNodeStartResult := make(chan error)
                    newNodeStartResult := make(chan error)
                    seedNodeInitialized := make(chan int)
                    newNodeInitialized := make(chan int)

                    seedNode.OnInitialized(func() {
                        seedNodeInitialized <- 1
                    })

                    newNode.OnInitialized(func() {
                        newNodeInitialized <- 1
                    })

                    go func() {
                        options := NodeInitializationOptions{
                            StartCluster: true,
                            ClusterSettings: ClusterSettings{
                                Partitions: 4,
                                ReplicationFactor: 3,
                            },
                        }

                        seedNodeStartResult <- seedNode.Start(options)
                    }()

                    // Wait for seed node to initialize
                    select {
                    case <-seedNodeInitialized:
                    case <-seedNodeStartResult:
                        Fail("Seed node failed to initialize")
                    case <-time.After(time.Minute):
                        Fail("Test timed out")
                    }

                    go func() {
                        options := NodeInitializationOptions{
                            JoinCluster: true,
                            SeedNodeHost: seedNodeServer.InternalHost(),
                            SeedNodePort: seedNodeServer.InternalPort(),
                        }

                        newNodeStartResult <- newNode.Start(options)
                    }()

                    select {
                    case <-newNodeInitialized:
                    case <-newNodeStartResult:
                        Fail("New node failed to initialize")
                    case <-time.After(time.Minute):
                        Fail("Test timed out")
                    }

                    // Wait for node to shut down
                    seedNode.Stop()
                    newNode.Stop()

                    select {
                    case <-seedNodeStartResult:
                    case <-time.After(time.Second):
                        Fail("Test timed out")
                    }

                    select {
                    case <-newNodeStartResult:
                    case <-time.After(time.Second):
                        Fail("Test timed out")
                    }
                })
            })
        })

        Context("The node is part of a cluster", func() {
            Context("And it has been set to be decomissioned", func() {
                It("should put the node into decomissioning mode", func() {
                    nodeStorageDriver := tempStorageDriver()
                    nodeServer := tempServer(8080, 9090)

                    node := New(ClusterNodeConfig{
                        StorageDriver: nodeStorageDriver,
                        CloudServer: nodeServer,
                        Capacity: 1,
                    })

                    memoryStore := NewRaftMemoryStorage()
                    node.UseRaftStore(memoryStore)

                    nodeStartResult := make(chan error)
                    nodeInitialized := make(chan int)

                    node.OnInitialized(func() {
                        nodeInitialized <- 1
                    })

                    go func() {
                        options := NodeInitializationOptions{
                            StartCluster: true,
                            ClusterSettings: ClusterSettings{
                                Partitions: 4,
                                ReplicationFactor: 3,
                            },
                        }

                        nodeStartResult <- node.Start(options)
                    }()

                    // Wait for seed node to initialize
                    select {
                    case <-nodeInitialized:
                    case <-nodeStartResult:
                        Fail("Node failed to initialize")
                    case <-time.After(time.Minute):
                        Fail("Test timed out")
                    }

                    // Wait for node to shut down
                    node.Stop()

                    select {
                    case <-nodeStartResult:
                    case <-time.After(time.Second):
                        Fail("Test timed out")
                    }

                    // Now that node has been shut down we will restart it after setting the decommissioning flag
                    memoryStore.SetDecommissioningFlag()
                    node2StorageDriver := tempStorageDriver()
                    node2Server := tempServer(8080, 9090)

                    node2 := New(ClusterNodeConfig{
                        StorageDriver: node2StorageDriver,
                        CloudServer: node2Server,
                        Capacity: 1,
                    })

                    memoryStore.SetIsEmpty(false)
                    node2.UseRaftStore(memoryStore)

                    node2StartResult := make(chan error)

                    go func() {
                        options := NodeInitializationOptions{
                            StartCluster: true,
                            ClusterSettings: ClusterSettings{
                                Partitions: 4,
                                ReplicationFactor: 3,
                            },
                        }

                        node2StartResult <- node2.Start(options)
                    }()

                    select {
                    case err := <-node2StartResult:
                        Expect(err).Should(Equal(EDecommissioned))
                    case <-time.After(time.Minute):
                        Fail("Test timed out")
                    }
                })
            })
        })

        Context("The node used to be part of a cluster but has since been removed", func() {
            It("Should return ERemoved", func() {
                nodeStorageDriver := tempStorageDriver()
                nodeServer := tempServer(8080, 9090)

                node := New(ClusterNodeConfig{
                    StorageDriver: nodeStorageDriver,
                    CloudServer: nodeServer,
                    Capacity: 1,
                })

                memoryStore := NewRaftMemoryStorage()
                node.UseRaftStore(memoryStore)

                nodeStartResult := make(chan error)
                nodeInitialized := make(chan int)

                node.OnInitialized(func() {
                    nodeInitialized <- 1
                })

                go func() {
                    options := NodeInitializationOptions{
                        StartCluster: true,
                        ClusterSettings: ClusterSettings{
                            Partitions: 4,
                            ReplicationFactor: 3,
                        },
                    }

                    nodeStartResult <- node.Start(options)
                }()

                // Wait for seed node to initialize
                select {
                case <-nodeInitialized:
                case <-nodeStartResult:
                    Fail("Node failed to initialize")
                case <-time.After(time.Minute):
                    Fail("Test timed out")
                }

                // Wait for node to shut down
                node.Stop()

                select {
                case <-nodeStartResult:
                case <-time.After(time.Second):
                    Fail("Test timed out")
                }

                // Now that node has been shut down we will restart it after setting the decommissioning flag
                memoryStore.SetDecommissioningFlag()
                node2StorageDriver := tempStorageDriver()
                node2Server := tempServer(8080, 9090)

                node2 := New(ClusterNodeConfig{
                    StorageDriver: node2StorageDriver,
                    CloudServer: node2Server,
                    Capacity: 1,
                })

                memoryStore.SetIsEmpty(false)
                node2.UseRaftStore(memoryStore)

                node2StartResult := make(chan error)

                go func() {
                    options := NodeInitializationOptions{
                        StartCluster: true,
                        ClusterSettings: ClusterSettings{
                            Partitions: 4,
                            ReplicationFactor: 3,
                        },
                    }

                    node2StartResult <- node2.Start(options)
                }()

                select {
                case err := <-node2StartResult:
                    Expect(err).Should(Equal(EDecommissioned))
                case <-time.After(time.Minute):
                    Fail("Test timed out")
                }

                // Start node again and ensure that it returns ERemoved
                node3StorageDriver := tempStorageDriver()
                node3Server := tempServer(8080, 9090)

                node3 := New(ClusterNodeConfig{
                    StorageDriver: node3StorageDriver,
                    CloudServer: node3Server,
                    Capacity: 1,
                })

                node3.UseRaftStore(memoryStore)

                node3StartResult := make(chan error)

                go func() {
                    options := NodeInitializationOptions{
                        StartCluster: true,
                        ClusterSettings: ClusterSettings{
                            Partitions: 4,
                            ReplicationFactor: 3,
                        },
                    }

                    node3StartResult <- node3.Start(options)
                }()

                select {
                case err := <-node3StartResult:
                    Expect(err).Should(Equal(ERemoved))
                case <-time.After(time.Minute):
                    Fail("Test timed out")
                }
            })
        })
    })
})
