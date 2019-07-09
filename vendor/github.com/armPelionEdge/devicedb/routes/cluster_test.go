package routes_test

import (
    "context"
    "errors"
    "encoding/json"
    "strings"
    "net/http"
    "net/http/httptest"

    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/raft"
    . "github.com/armPelionEdge/devicedb/routes"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "github.com/gorilla/mux"
)

var _ = Describe("Cluster", func() {
    var router *mux.Router
    var clusterEndpoint *ClusterEndpoint
    var clusterFacade *MockClusterFacade

    BeforeEach(func() {
        clusterFacade = &MockClusterFacade{ }
        router = mux.NewRouter()
        clusterEndpoint = &ClusterEndpoint{
            ClusterFacade: clusterFacade,
        }
        clusterEndpoint.Attach(router)
    })

    Describe("/cluster/nodes", func() {
        Describe("POST", func() {
            Context("When the message body is not a valid node config", func() {
                It("Should respond with status code http.StatusBadRequest", func() {
                    req, err := http.NewRequest("POST", "/cluster/nodes", strings.NewReader("asdf"))

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusBadRequest))
                })
            })

            Context("When the message body is a valid node config", func() {
                Context("But AddNode() returns an ECancelConfChange error", func() {
                    It("Should respond with status code http.StatusInternalServerError", func() {
                        nodeConfig := NodeConfig{
                            Address: PeerAddress{
                                NodeID: 7,
                                Host: "localhost",
                                Port: 9000,
                            },
                            Capacity: 1,
                        }
                        encodedNodeConfig, err := json.Marshal(nodeConfig)

                        Expect(err).Should(BeNil())

                        req, err := http.NewRequest("POST", "/cluster/nodes", strings.NewReader(string(encodedNodeConfig)))
                        addNodeCalled := make(chan int, 1)
                        clusterFacade.defaultAddNodeResponse = ECancelConfChange
                        clusterFacade.addNodeCB = func(ctx context.Context, nodeConfig NodeConfig) {
                            Expect(nodeConfig).Should(Equal(nodeConfig))

                            addNodeCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        
                        select {
                        case <-addNodeCalled:
                        default:
                            Fail("Request did not cause AddNode to be invoked")
                        }
                    })

                    It("Should respond with an EDuplicateNodeID body", func() {
                        nodeConfig := NodeConfig{
                            Address: PeerAddress{
                                NodeID: 7,
                                Host: "localhost",
                                Port: 9000,
                            },
                            Capacity: 1,
                        }
                        encodedNodeConfig, err := json.Marshal(nodeConfig)

                        Expect(err).Should(BeNil())

                        req, err := http.NewRequest("POST", "/cluster/nodes", strings.NewReader(string(encodedNodeConfig)))
                        addNodeCalled := make(chan int, 1)
                        clusterFacade.defaultAddNodeResponse = ECancelConfChange
                        clusterFacade.addNodeCB = func(ctx context.Context, nodeConfig NodeConfig) {
                            Expect(nodeConfig).Should(Equal(nodeConfig))

                            addNodeCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        var dbErr DBerror
                        Expect(json.Unmarshal(rr.Body.Bytes(), &dbErr)).Should(BeNil())
                        Expect(dbErr).Should(Equal(EDuplicateNodeID))
                        
                        select {
                        case <-addNodeCalled:
                        default:
                            Fail("Request did not cause AddNode to be invoked")
                        }
                    })
                })

                Context("But AddNode() returns an error other than an ECancelConfChange error", func() {
                    It("Should respond with status code http.StatusInternalServerError", func() {
                        nodeConfig := NodeConfig{
                            Address: PeerAddress{
                                NodeID: 7,
                                Host: "localhost",
                                Port: 9000,
                            },
                            Capacity: 1,
                        }
                        encodedNodeConfig, err := json.Marshal(nodeConfig)

                        Expect(err).Should(BeNil())

                        req, err := http.NewRequest("POST", "/cluster/nodes", strings.NewReader(string(encodedNodeConfig)))
                        addNodeCalled := make(chan int, 1)
                        clusterFacade.defaultAddNodeResponse = errors.New("Some error")
                        clusterFacade.addNodeCB = func(ctx context.Context, nodeConfig NodeConfig) {
                            Expect(nodeConfig).Should(Equal(nodeConfig))

                            addNodeCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        var dbErr DBerror
                        Expect(json.Unmarshal(rr.Body.Bytes(), &dbErr)).Should(BeNil())
                        Expect(dbErr).Should(Equal(EProposalError))
                        
                        select {
                        case <-addNodeCalled:
                        default:
                            Fail("Request did not cause AddNode to be invoked")
                        }
                    })

                    It("Should respond with an EProposalError body", func() {
                        nodeConfig := NodeConfig{
                            Address: PeerAddress{
                                NodeID: 7,
                                Host: "localhost",
                                Port: 9000,
                            },
                            Capacity: 1,
                        }
                        encodedNodeConfig, err := json.Marshal(nodeConfig)

                        Expect(err).Should(BeNil())

                        req, err := http.NewRequest("POST", "/cluster/nodes", strings.NewReader(string(encodedNodeConfig)))
                        addNodeCalled := make(chan int, 1)
                        clusterFacade.defaultAddNodeResponse = errors.New("Some error")
                        clusterFacade.addNodeCB = func(ctx context.Context, nodeConfig NodeConfig) {
                            Expect(nodeConfig).Should(Equal(nodeConfig))

                            addNodeCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        
                        select {
                        case <-addNodeCalled:
                        default:
                            Fail("Request did not cause AddNode to be invoked")
                        }
                    })
                })

                Context("And AddNode() returns no error", func() {
                    It("Should respond with status code http.StatusOK", func() {
                        nodeConfig := NodeConfig{
                            Address: PeerAddress{
                                NodeID: 7,
                                Host: "localhost",
                                Port: 9000,
                            },
                            Capacity: 1,
                        }
                        encodedNodeConfig, err := json.Marshal(nodeConfig)

                        Expect(err).Should(BeNil())

                        req, err := http.NewRequest("POST", "/cluster/nodes", strings.NewReader(string(encodedNodeConfig)))
                        addNodeCalled := make(chan int, 1)
                        clusterFacade.defaultAddNodeResponse = nil
                        clusterFacade.addNodeCB = func(ctx context.Context, nodeConfig NodeConfig) {
                            Expect(nodeConfig).Should(Equal(nodeConfig))

                            addNodeCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        Expect(rr.Code).Should(Equal(http.StatusOK))
                        
                        select {
                        case <-addNodeCalled:
                        default:
                            Fail("Request did not cause AddNode to be invoked")
                        }
                    })
                })
            })
        })
    })

    Describe("/cluster/nodes/{nodeID}", func() {
        Describe("DELETE", func() {
            Context("When a replacement node id is specified and the decommissioning flag is set", func() {
                It("Should respond with status code http.StatusBadRequest", func() {
                    req, err := http.NewRequest("DELETE", "/cluster/nodes/45?replace=4&decommission=true", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusBadRequest))
                })
            })

            Context("When the nodeID query parameter is not a base 10 encoded 64 bit number", func() {
                It("Should respond with status code http.StatusBadRequest", func() {
                    req, err := http.NewRequest("DELETE", "/cluster/nodes/asdf", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusBadRequest))
                })
            })

            Context("When the decommission flag is set", func() {
                Context("And the nodeID is set to zero", func() {
                    It("Should call Decommission() on the receiving node", func() {
                        req, err := http.NewRequest("DELETE", "/cluster/nodes/0?&decommission=true", nil)
                        decommissionCalled := make(chan int, 1)
                        clusterFacade.defaultDecommissionResponse = nil
                        clusterFacade.decommisionCB = func() {
                            decommissionCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        select {
                        case <-decommissionCalled:
                        default:
                            Fail("Request did not cause Decommission() to be called")
                        }
                    })

                    Context("And if the Decommission() call returns an error", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/0?&decommission=true", nil)
                            decommissionCalled := make(chan int, 1)
                            clusterFacade.defaultDecommissionResponse = errors.New("Some error")
                            clusterFacade.decommisionCB = func() {
                                decommissionCalled <- 1
                            }

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))

                            select {
                            case <-decommissionCalled:
                            default:
                                Fail("Request did not cause Decommission() to be called")
                            }
                        })
                    })

                    Context("And the Decommission() call is successful", func() {
                        It("Should respond with status code http.StatusOK", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/0?&decommission=true", nil)
                            decommissionCalled := make(chan int, 1)
                            clusterFacade.defaultDecommissionResponse = nil
                            clusterFacade.decommisionCB = func() {
                                decommissionCalled <- 1
                            }

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusOK))

                            select {
                            case <-decommissionCalled:
                            default:
                                Fail("Request did not cause Decommission() to be called")
                            }
                        })
                    })
                })

                Context("And the nodeID is set to the ID of the node receiving the request", func() {
                    It("Should call Decommission() on the receiving node", func() {
                        req, err := http.NewRequest("DELETE", "/cluster/nodes/34?&decommission=true", nil)
                        decommissionCalled := make(chan int, 1)
                        clusterFacade.localNodeID = 34
                        clusterFacade.defaultDecommissionResponse = nil
                        clusterFacade.decommisionCB = func() {
                            decommissionCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        select {
                        case <-decommissionCalled:
                        default:
                            Fail("Request did not cause Decommission() to be called")
                        }
                    })

                    Context("And if the Decommission() call returns an error", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/34?&decommission=true", nil)
                            decommissionCalled := make(chan int, 1)
                            clusterFacade.localNodeID = 34
                            clusterFacade.defaultDecommissionResponse = errors.New("Some error")
                            clusterFacade.decommisionCB = func() {
                                decommissionCalled <- 1
                            }

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))

                            select {
                            case <-decommissionCalled:
                            default:
                                Fail("Request did not cause Decommission() to be called")
                            }
                        })
                    })

                    Context("And the Decommission() call is successful", func() {
                        It("Should respond with status code http.StatusOK", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/34?&decommission=true", nil)
                            decommissionCalled := make(chan int, 1)
                            clusterFacade.localNodeID = 34
                            clusterFacade.defaultDecommissionResponse = nil
                            clusterFacade.decommisionCB = func() {
                                decommissionCalled <- 1
                            }

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusOK))

                            select {
                            case <-decommissionCalled:
                            default:
                                Fail("Request did not cause Decommission() to be called")
                            }
                        })
                    })
                })

                Context("And the nodeID is not set to the ID of the node receiving the request", func() {
                    Context("And the wasForwarded flag is set", func() {
                        It("Should respond with status code http.StatusForbidden", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/34?&decommission=true&forwarded=true", nil)
                            clusterFacade.localNodeID = 35

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusForbidden))
                        })
                    })

                    Context("And the wasForwarded flag is not set", func() {
                        Context("And the node receiving the request does not know how the address of the specified node", func() {
                            It("Should respond with status code http.StatusBadGateway", func() {
                                req, err := http.NewRequest("DELETE", "/cluster/nodes/34?&decommission=true", nil)
                                clusterFacade.localNodeID = 35
                                clusterFacade.defaultPeerAddress = PeerAddress{}

                                Expect(err).Should(BeNil())

                                rr := httptest.NewRecorder()
                                router.ServeHTTP(rr, req)

                                Expect(rr.Code).Should(Equal(http.StatusBadGateway))
                            })
                        })

                        Context("And the node receiving the request knows the address of the specified node", func() {
                            It("Should forward the request to the specified node, making sure to set the wasForwarded flag", func() {
                                req, err := http.NewRequest("DELETE", "/cluster/nodes/34?&decommission=true", nil)
                                clusterFacade.localNodeID = 35
                                clusterFacade.defaultPeerAddress = PeerAddress{
                                    NodeID: 34,
                                    Host: "localhost",
                                    Port: 8080,
                                }
                                decommissionPeerCalled := make(chan int, 1)
                                clusterFacade.decommisionPeerCB = func(nodeID uint64) {
                                    Expect(nodeID).Should(Equal(uint64(34)))
                                    decommissionPeerCalled <- 1
                                }

                                Expect(err).Should(BeNil())

                                rr := httptest.NewRecorder()
                                router.ServeHTTP(rr, req)

                                select {
                                case <-decommissionPeerCalled:
                                default:
                                    Fail("Request did not cause DecommissionPeer() to be invoked")
                                }
                            })

                            Context("And if the DecommissionPeer() returns an error", func() {
                                It("Should respond with status code http.StatusBadGateway", func() {
                                    req, err := http.NewRequest("DELETE", "/cluster/nodes/34?&decommission=true", nil)
                                    clusterFacade.localNodeID = 35
                                    clusterFacade.defaultPeerAddress = PeerAddress{
                                        NodeID: 34,
                                        Host: "localhost",
                                        Port: 8080,
                                    }
                                    clusterFacade.defaultDecommissionPeerResponse = errors.New("Some error")

                                    Expect(err).Should(BeNil())

                                    rr := httptest.NewRecorder()
                                    router.ServeHTTP(rr, req)

                                    Expect(rr.Code).Should(Equal(http.StatusBadGateway))
                                })
                            })

                            Context("And if DecommissionPeer() is successful", func() {
                                It("Should respond with status code http.StatusOK", func() {
                                    req, err := http.NewRequest("DELETE", "/cluster/nodes/34?&decommission=true", nil)
                                    clusterFacade.localNodeID = 35
                                    clusterFacade.defaultPeerAddress = PeerAddress{
                                        NodeID: 34,
                                        Host: "localhost",
                                        Port: 8080,
                                    }
                                    clusterFacade.defaultDecommissionPeerResponse = nil

                                    Expect(err).Should(BeNil())

                                    rr := httptest.NewRecorder()
                                    router.ServeHTTP(rr, req)

                                    Expect(rr.Code).Should(Equal(http.StatusOK))
                                })
                            })
                        })
                    })
                })
            })

            Context("When a replacement node id is specified", func() {
                Context("And the replacement node id cannot be parsed as a base 10 encoded uint64", func() {
                    It("Should respond with status code http.StatusBadRequest", func() {
                        req, err := http.NewRequest("DELETE", "/cluster/nodes/34?&replace=hi", nil)

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        Expect(rr.Code).Should(Equal(http.StatusBadRequest))
                    })
                })

                Context("And the nodeID is set to zero", func() {
                    It("Should call ReplaceNode() to remove the receiving node from the cluster and replace it with the specified node", func() {
                        req, err := http.NewRequest("DELETE", "/cluster/nodes/0?&replace=100", nil)
                        replaceNodeCalled := make(chan int, 1)
                        clusterFacade.localNodeID = 50
                        clusterFacade.replaceNodeCB = func(ctx context.Context, nodeID uint64, replacementNodeID uint64) {
                            Expect(nodeID).Should(Equal(uint64(50)))
                            Expect(replacementNodeID).Should(Equal(uint64(100)))

                            replaceNodeCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        select {
                        case <-replaceNodeCalled:
                        default:
                            Fail("Request should have caused ReplaceNode to be invoked")
                        }
                    })

                    Context("And if the call to ReplaceNode() returns an error", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/0?&replace=100", nil)
                            clusterFacade.localNodeID = 50
                            clusterFacade.defaultReplaceNodeResponse = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)
                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })
                    })

                    Context("And if the call to ReplaceNode() is successful", func() {
                        It("Should respond with status code http.StatusOK", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/0?&replace=100", nil)
                            clusterFacade.localNodeID = 50
                            clusterFacade.defaultReplaceNodeResponse = nil

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)
                            Expect(rr.Code).Should(Equal(http.StatusOK))
                        })
                    })
                })

                Context("And the nodeID is set to the ID of the receiving node", func() {
                    It("Should call ReplaceNode() to remove the receiving node from the cluster and replace it with the specified node", func() {
                        req, err := http.NewRequest("DELETE", "/cluster/nodes/50?&replace=100", nil)
                        replaceNodeCalled := make(chan int, 1)
                        clusterFacade.localNodeID = 50
                        clusterFacade.replaceNodeCB = func(ctx context.Context, nodeID uint64, replacementNodeID uint64) {
                            Expect(nodeID).Should(Equal(uint64(50)))
                            Expect(replacementNodeID).Should(Equal(uint64(100)))

                            replaceNodeCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        select {
                        case <-replaceNodeCalled:
                        default:
                            Fail("Request should have caused ReplaceNode to be invoked")
                        }
                    })

                    Context("And if the call to ReplaceNode() returns an error", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/50?&replace=100", nil)
                            clusterFacade.localNodeID = 50
                            clusterFacade.defaultReplaceNodeResponse = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)
                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })
                    })

                    Context("And if the call to ReplaceNode() is successful", func() {
                        It("Should respond with status code http.StatusOK", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/50?&replace=100", nil)
                            clusterFacade.localNodeID = 50
                            clusterFacade.defaultReplaceNodeResponse = nil

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)
                            Expect(rr.Code).Should(Equal(http.StatusOK))
                        })
                    })
                })

                Context("And the nodeID is non-zero and not set to the ID of the receiving node", func() {
                    It("Should call ReplaceNode() to remove the receiving node from the cluster and replace it with the specified node", func() {
                        req, err := http.NewRequest("DELETE", "/cluster/nodes/51?&replace=100", nil)
                        replaceNodeCalled := make(chan int, 1)
                        clusterFacade.localNodeID = 50
                        clusterFacade.replaceNodeCB = func(ctx context.Context, nodeID uint64, replacementNodeID uint64) {
                            Expect(nodeID).Should(Equal(uint64(51)))
                            Expect(replacementNodeID).Should(Equal(uint64(100)))

                            replaceNodeCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        select {
                        case <-replaceNodeCalled:
                        default:
                            Fail("Request should have caused ReplaceNode to be invoked")
                        }
                    })

                    Context("And if the call to ReplaceNode() returns an error", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/51?&replace=100", nil)
                            clusterFacade.localNodeID = 51
                            clusterFacade.defaultReplaceNodeResponse = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)
                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })
                    })

                    Context("And if the call to ReplaceNode() is successful", func() {
                        It("Should respond with status code http.StatusOK", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/51?&replace=100", nil)
                            clusterFacade.localNodeID = 51
                            clusterFacade.defaultReplaceNodeResponse = nil

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)
                            Expect(rr.Code).Should(Equal(http.StatusOK))
                        })
                    })
                })
            })

            Context("When a replacement node id is not specified", func() {
                Context("And the nodeID is set to zero", func() {
                    It("Should call RemoveNode() to remove the receiving node from the cluster", func() {
                        req, err := http.NewRequest("DELETE", "/cluster/nodes/0", nil)
                        removeNodeCalled := make(chan int, 1)
                        clusterFacade.localNodeID = 50
                        clusterFacade.removeNodeCB = func(ctx context.Context, nodeID uint64) {
                            Expect(nodeID).Should(Equal(uint64(50)))

                            removeNodeCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        select {
                        case <-removeNodeCalled:
                        default:
                            Fail("Request should have caused RemoveNode to be invoked")
                        }
                    })

                    Context("And if the call to RemoveNode() returns an error", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/0", nil)
                            clusterFacade.localNodeID = 50
                            clusterFacade.defaultRemoveNodeResponse = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })
                    })

                    Context("And if the call to RemoveNode() is successful", func() {
                        It("Should respond with status code http.StatusOK", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/0", nil)
                            clusterFacade.localNodeID = 50
                            clusterFacade.defaultRemoveNodeResponse = nil

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusOK))
                        })
                    })
                })

                Context("And the nodeID is set to the ID of the receiving node", func() {
                    It("Should call RemoveNode() to remove the receiving node from the cluster", func() {
                        req, err := http.NewRequest("DELETE", "/cluster/nodes/50", nil)
                        removeNodeCalled := make(chan int, 1)
                        clusterFacade.localNodeID = 50
                        clusterFacade.removeNodeCB = func(ctx context.Context, nodeID uint64) {
                            Expect(nodeID).Should(Equal(uint64(50)))

                            removeNodeCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        select {
                        case <-removeNodeCalled:
                        default:
                            Fail("Request should have caused RemoveNode to be invoked")
                        }
                    })

                    Context("And if the call to RemoveNode() returns an error", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/50", nil)
                            clusterFacade.localNodeID = 50
                            clusterFacade.defaultRemoveNodeResponse = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })
                    })

                    Context("And if the call to RemoveNode() is successful", func() {
                        It("Should respond with status code http.StatusOK", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/50", nil)
                            clusterFacade.localNodeID = 50
                            clusterFacade.defaultRemoveNodeResponse = nil

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusOK))
                        })
                    })
                })

                Context("And the nodeID is non-zero and not set to the ID of the receiving node", func() {
                    It("Should call RemoveNode() to remove the receiving node from the cluster", func() {
                        req, err := http.NewRequest("DELETE", "/cluster/nodes/100", nil)
                        removeNodeCalled := make(chan int, 1)
                        clusterFacade.localNodeID = 50
                        clusterFacade.removeNodeCB = func(ctx context.Context, nodeID uint64) {
                            Expect(nodeID).Should(Equal(uint64(100)))

                            removeNodeCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        select {
                        case <-removeNodeCalled:
                        default:
                            Fail("Request should have caused RemoveNode to be invoked")
                        }
                    })

                    Context("And if the call to RemoveNode() returns an error", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/100", nil)
                            clusterFacade.localNodeID = 50
                            clusterFacade.defaultRemoveNodeResponse = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })
                    })

                    Context("And if the call to RemoveNode() is successful", func() {
                        It("Should respond with status code http.StatusOK", func() {
                            req, err := http.NewRequest("DELETE", "/cluster/nodes/100", nil)
                            clusterFacade.localNodeID = 50
                            clusterFacade.defaultRemoveNodeResponse = nil

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusOK))
                        })
                    })
                })
            })
        })
    })
})
