package routes_test

import (
    "encoding/json"
    "context"
    "errors"
    "net/http"
    "net/http/httptest"
    "strings"

    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/routes"
    . "github.com/armPelionEdge/devicedb/transport"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "github.com/gorilla/mux"
)

var _ = Describe("Sites", func() {
    var router *mux.Router
    var sitesEndpoint *SitesEndpoint
    var clusterFacade *MockClusterFacade

    BeforeEach(func() {
        clusterFacade = &MockClusterFacade{ }
        router = mux.NewRouter()
        sitesEndpoint = &SitesEndpoint{
            ClusterFacade: clusterFacade,
        }
        sitesEndpoint.Attach(router)
    })

    Describe("/sites/{siteID}", func() {
        Describe("PUT", func() {
            It("Should call AddSite() on the node facade with the site ID specified in the path", func() {
                req, err := http.NewRequest("PUT", "/sites/site1", nil)

                addSiteCalled := make(chan int, 1)
                clusterFacade.addSiteCB = func(ctx context.Context, siteID string) {
                    Expect(siteID).Should(Equal("site1"))
                    addSiteCalled <- 1
                }

                Expect(err).Should(BeNil())

                rr := httptest.NewRecorder()
                router.ServeHTTP(rr, req)

                select {
                case <-addSiteCalled:
                default:
                    Fail("Should have invoked AddSite()")
                }
            })

            Context("And if AddSite() returns an error", func() {
                It("Should respond with status code http.StatusInternalServerError", func() {
                    req, err := http.NewRequest("PUT", "/sites/site1", nil)

                    clusterFacade.defaultAddSiteResponse = errors.New("Some error")

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                })
            })

            Context("And if AddSite() is successful", func() {
                It("Should respond with status code http.StatusOK", func() {
                    req, err := http.NewRequest("PUT", "/sites/site1", nil)

                    clusterFacade.defaultAddSiteResponse = nil

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusOK))
                })
            })
        })
        
        Describe("DELETE", func() {
            It("Should call RemoveSite() on the node facade with the site ID specified in the path", func() {
                req, err := http.NewRequest("DELETE", "/sites/site1", nil)

                removeSiteCalled := make(chan int, 1)
                clusterFacade.removeSiteCB = func(ctx context.Context, siteID string) {
                    Expect(siteID).Should(Equal("site1"))
                    removeSiteCalled <- 1
                }

                Expect(err).Should(BeNil())

                rr := httptest.NewRecorder()
                router.ServeHTTP(rr, req)

                select {
                case <-removeSiteCalled:
                default:
                    Fail("Should have invoked RemoveSite()")
                }
            })

            Context("And if RemoveSite() returns an error", func() {
                It("Should respond with staus code http.StatusInternalServerError", func() {
                    req, err := http.NewRequest("DELETE", "/sites/site1", nil)

                    clusterFacade.defaultRemoveSiteResponse = errors.New("Some error")

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                })
            })

            Context("And if RemoveSite() is successful", func() {
                It("Should respond with status code http.StatusOK", func() {
                    req, err := http.NewRequest("DELETE", "/sites/site1", nil)

                    clusterFacade.defaultRemoveSiteResponse = nil

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusOK))
                })
            })
        })
    })

    Describe("/sites/{siteID}/buckets/{bucketID}/batches", func() {
        Describe("POST", func() {
            Context("When the provided body of the request cannot be parsed as a TransportUpdateBatch", func() {
                It("Should respond with status code http.StatusBadRequest", func() {
                    req, err := http.NewRequest("POST", "/sites/site1/buckets/default/batches", strings.NewReader("asdf"))

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusBadRequest))
                })
            })

            Context("When the provided TransportUpdateBatch cannot be convered into an UpdateBatch", func() {
                It("Should respond with status code http.StatusBadRequest", func() {
                    var transportUpdateBatch TransportUpdateBatch = []TransportUpdateOp{
                        TransportUpdateOp{
                            Type: "badop",
                            Key: "asdf",
                            Value: "adfs",
                            Context: "",
                        },
                    }

                    encodedTransportUpdateBatch, err := json.Marshal(&transportUpdateBatch)

                    Expect(err).Should(BeNil())

                    req, err := http.NewRequest("POST", "/sites/site1/buckets/default/batches", strings.NewReader(string(encodedTransportUpdateBatch)))

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusBadRequest))
                })
            })

            Context("When the provided body can be parsed as a TransporUpdateBatch and it is successfully converted to an UpdateBatch", func() {
                It("Should call Batch() on the node facade with the site ID and bucket specified in the path", func() {
                    var transportUpdateBatch TransportUpdateBatch = []TransportUpdateOp{
                        TransportUpdateOp{
                            Type: "put",
                            Key: "ABC",
                            Value: "123",
                            Context: "",
                        },
                    }

                    encodedTransportUpdateBatch, err := json.Marshal(&transportUpdateBatch)

                    Expect(err).Should(BeNil())

                    req, err := http.NewRequest("POST", "/sites/site1/buckets/default/batches", strings.NewReader(string(encodedTransportUpdateBatch)))

                    batchCalled := make(chan int, 1)
                    clusterFacade.batchCB = func(siteID string, bucket string, updateBatch *UpdateBatch) {
                        Expect(siteID).Should(Equal("site1"))
                        Expect(bucket).Should(Equal("default"))
                        Expect(updateBatch).Should(Not(BeNil()))

                        batchCalled <- 1
                    }

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    select {
                    case <-batchCalled:
                    default:
                        Fail("Should have invoked Batch()")
                    }
                })

                Context("And if Batch() returns an error", func() {
                    Context("And the error is ENoSuchSite", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            var transportUpdateBatch TransportUpdateBatch = []TransportUpdateOp{
                                TransportUpdateOp{
                                    Type: "put",
                                    Key: "ABC",
                                    Value: "123",
                                    Context: "",
                                },
                            }

                            encodedTransportUpdateBatch, err := json.Marshal(&transportUpdateBatch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/sites/site1/buckets/default/batches", strings.NewReader(string(encodedTransportUpdateBatch)))
                            clusterFacade.defaultBatchError = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with an ESiteDoesNotExist body", func() {
                            var transportUpdateBatch TransportUpdateBatch = []TransportUpdateOp{
                                TransportUpdateOp{
                                    Type: "put",
                                    Key: "ABC",
                                    Value: "123",
                                    Context: "",
                                },
                            }

                            encodedTransportUpdateBatch, err := json.Marshal(&transportUpdateBatch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/sites/site1/buckets/default/batches", strings.NewReader(string(encodedTransportUpdateBatch)))
                            clusterFacade.defaultBatchError = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var encodedDBError DBerror

                            Expect(json.Unmarshal(rr.Body.Bytes(), &encodedDBError)).Should(BeNil())
                            Expect(encodedDBError).Should(Equal(ESiteDoesNotExist))
                        })
                    })

                    Context("And the error is ENoSuchBucket", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            var transportUpdateBatch TransportUpdateBatch = []TransportUpdateOp{
                                TransportUpdateOp{
                                    Type: "put",
                                    Key: "ABC",
                                    Value: "123",
                                    Context: "",
                                },
                            }

                            encodedTransportUpdateBatch, err := json.Marshal(&transportUpdateBatch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/sites/site1/buckets/default/batches", strings.NewReader(string(encodedTransportUpdateBatch)))
                            clusterFacade.defaultBatchError = ENoSuchBucket

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with an EBucketDoesNotExist body", func() {
                            var transportUpdateBatch TransportUpdateBatch = []TransportUpdateOp{
                                TransportUpdateOp{
                                    Type: "put",
                                    Key: "ABC",
                                    Value: "123",
                                    Context: "",
                                },
                            }

                            encodedTransportUpdateBatch, err := json.Marshal(&transportUpdateBatch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/sites/site1/buckets/default/batches", strings.NewReader(string(encodedTransportUpdateBatch)))
                            clusterFacade.defaultBatchError = ENoSuchBucket

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var encodedDBError DBerror

                            Expect(json.Unmarshal(rr.Body.Bytes(), &encodedDBError)).Should(BeNil())
                            Expect(encodedDBError).Should(Equal(EBucketDoesNotExist))
                        })
                    })

                    Context("And the error is ENoQuorum", func() {
                        It("Should respond with status code http.StatusOK", func() {
                            var transportUpdateBatch TransportUpdateBatch = []TransportUpdateOp{
                                TransportUpdateOp{
                                    Type: "put",
                                    Key: "ABC",
                                    Value: "123",
                                    Context: "",
                                },
                            }

                            encodedTransportUpdateBatch, err := json.Marshal(&transportUpdateBatch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/sites/site1/buckets/default/batches", strings.NewReader(string(encodedTransportUpdateBatch)))
                            clusterFacade.defaultBatchError = ENoQuorum

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusOK))
                        })

                        It("Should respond with a BatchResult body indicating how many replicas received the update", func() {
                            var transportUpdateBatch TransportUpdateBatch = []TransportUpdateOp{
                                TransportUpdateOp{
                                    Type: "put",
                                    Key: "ABC",
                                    Value: "123",
                                    Context: "",
                                },
                            }

                            encodedTransportUpdateBatch, err := json.Marshal(&transportUpdateBatch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/sites/site1/buckets/default/batches", strings.NewReader(string(encodedTransportUpdateBatch)))
                            clusterFacade.defaultBatchError = ENoQuorum
                            clusterFacade.defaultBatchResponse = BatchResult{
                                NApplied: 2,
                                Replicas: 3,
                            }

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var batchResult BatchResult

                            Expect(json.Unmarshal(rr.Body.Bytes(), &batchResult)).Should(BeNil())
                            Expect(batchResult.NApplied).Should(Equal(uint64(2)))
                            Expect(batchResult.Replicas).Should(Equal(uint64(3)))
                        })
                    })

                    Context("Otherwise", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            var transportUpdateBatch TransportUpdateBatch = []TransportUpdateOp{
                                TransportUpdateOp{
                                    Type: "put",
                                    Key: "ABC",
                                    Value: "123",
                                    Context: "",
                                },
                            }

                            encodedTransportUpdateBatch, err := json.Marshal(&transportUpdateBatch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/sites/site1/buckets/default/batches", strings.NewReader(string(encodedTransportUpdateBatch)))
                            clusterFacade.defaultBatchError = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })

                        It("Should respond with an EStorage body", func() {
                            var transportUpdateBatch TransportUpdateBatch = []TransportUpdateOp{
                                TransportUpdateOp{
                                    Type: "put",
                                    Key: "ABC",
                                    Value: "123",
                                    Context: "",
                                },
                            }

                            encodedTransportUpdateBatch, err := json.Marshal(&transportUpdateBatch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/sites/site1/buckets/default/batches", strings.NewReader(string(encodedTransportUpdateBatch)))
                            clusterFacade.defaultBatchError = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var encodedDBError DBerror

                            Expect(json.Unmarshal(rr.Body.Bytes(), &encodedDBError)).Should(BeNil())
                            Expect(encodedDBError).Should(Equal(EStorage))
                        })
                    })
                })

                Context("And if Batch() is successful", func() {
                    It("Should respond with status code http.StatusOK", func() {
                        var transportUpdateBatch TransportUpdateBatch = []TransportUpdateOp{
                            TransportUpdateOp{
                                Type: "put",
                                Key: "ABC",
                                Value: "123",
                                Context: "",
                            },
                        }

                        encodedTransportUpdateBatch, err := json.Marshal(&transportUpdateBatch)

                        Expect(err).Should(BeNil())

                        req, err := http.NewRequest("POST", "/sites/site1/buckets/default/batches", strings.NewReader(string(encodedTransportUpdateBatch)))
                        clusterFacade.defaultBatchError = nil

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        Expect(rr.Code).Should(Equal(http.StatusOK))
                    })

                    It("Should respond with a BatchResult body indicating how many replicas received the update", func() {
                        var transportUpdateBatch TransportUpdateBatch = []TransportUpdateOp{
                            TransportUpdateOp{
                                Type: "put",
                                Key: "ABC",
                                Value: "123",
                                Context: "",
                            },
                        }

                        encodedTransportUpdateBatch, err := json.Marshal(&transportUpdateBatch)

                        Expect(err).Should(BeNil())

                        req, err := http.NewRequest("POST", "/sites/site1/buckets/default/batches", strings.NewReader(string(encodedTransportUpdateBatch)))
                        clusterFacade.defaultBatchError = nil
                        clusterFacade.defaultBatchResponse = BatchResult{
                            NApplied: 3,
                            Replicas: 3,
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        var batchResult BatchResult

                        Expect(json.Unmarshal(rr.Body.Bytes(), &batchResult)).Should(BeNil())
                        Expect(batchResult.NApplied).Should(Equal(uint64(3)))
                        Expect(batchResult.Replicas).Should(Equal(uint64(3)))
                    })
                })
            })
        })
    })

    Describe("/sites/{siteID}/buckets/{bucketID}/keys", func() {
        Describe("GET", func() {
            Context("When the request includes both \"key\" and \"prefix\" query parameters", func() {
                It("Should respond with status code http.StatusBadRequest", func() {
                    req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?key=key1&prefix=prefix1", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusBadRequest))
                })
            })

            Context("When the request includes neither \"key\" nor \"prefix\" query parameters", func() {
                It("Should respond with status code http.StatusOK", func() {
                    req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusOK))
                })

                It("Should respond with an empty JSON-encoded list APIEntry database objects", func() {
                    req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    var entries []APIEntry

                    Expect(json.Unmarshal(rr.Body.Bytes(), &entries)).Should(BeNil())
                    Expect(entries).Should(Equal([]APIEntry{ }))
                })
            })

            Context("When the request includes one or more \"key\" parameters", func() {
                It("Should call Get() on the node facade with the specified site, bucket and keys", func() {
                    req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?key=a&key=b", nil)
                    getCalled := make(chan int, 1)
                    clusterFacade.defaultGetResponse = []*SiblingSet{ nil, nil }
                    clusterFacade.getCB = func(siteID string, bucket string, keys [][]byte) {
                        Expect(siteID).Should(Equal("site1"))
                        Expect(bucket).Should(Equal("default"))
                        Expect(keys).Should(Equal([][]byte{ []byte("a"), []byte("b") }))

                        getCalled <- 1
                    }

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    select {
                    case <-getCalled:
                    default:
                        Fail("Request did not cause Get() to be invoked")
                    }
                })

                Context("And if Get() returns an error", func() {
                    Context("And the error is ENoSuchSite", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?key=a&key=b", nil)
                            clusterFacade.defaultGetResponseError = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with an ESiteDoesNotExist body", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?key=a&key=b", nil)
                            clusterFacade.defaultGetResponseError = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var encodedDBError DBerror

                            Expect(json.Unmarshal(rr.Body.Bytes(), &encodedDBError)).Should(BeNil())
                            Expect(encodedDBError).Should(Equal(ESiteDoesNotExist))
                        })
                    })

                    Context("And the error is ENoSuchBucket", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?key=a&key=b", nil)
                            clusterFacade.defaultGetResponseError = ENoSuchBucket

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with an EBucketDoesNotExist body", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?key=a&key=b", nil)
                            clusterFacade.defaultGetResponseError = ENoSuchBucket

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var encodedDBError DBerror

                            Expect(json.Unmarshal(rr.Body.Bytes(), &encodedDBError)).Should(BeNil())
                            Expect(encodedDBError).Should(Equal(EBucketDoesNotExist))
                        })
                    })

                    Context("And the error is ENoQuorum", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?key=a&key=b", nil)
                            clusterFacade.defaultGetResponseError = ENoQuorum

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })

                        It("Should respond with an ENoQuorum body", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?key=a&key=b", nil)
                            clusterFacade.defaultGetResponseError = ENoQuorum

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var encodedDBError DBerror

                            Expect(json.Unmarshal(rr.Body.Bytes(), &encodedDBError)).Should(BeNil())
                            Expect(encodedDBError).Should(Equal(ENoQuorum))
                        })
                    })

                    Context("Otherwise", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?key=a&key=b", nil)
                            clusterFacade.defaultGetResponseError = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })

                        It("Should respond with an EStorage body", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?key=a&key=b", nil)
                            clusterFacade.defaultGetResponseError = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var encodedDBError DBerror

                            Expect(json.Unmarshal(rr.Body.Bytes(), &encodedDBError)).Should(BeNil())
                            Expect(encodedDBError).Should(Equal(EStorage))
                        })
                    })
                })

                Context("And if Get() is successful", func() {
                    It("Should respond with status code http.StatusOK", func() {
                        req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?key=a&key=b", nil)
                        clusterFacade.defaultGetResponseError = nil
                        clusterFacade.defaultGetResponse = []*SiblingSet{ nil, nil }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        Expect(rr.Code).Should(Equal(http.StatusOK))
                    })

                    It("Should respond with a JSON-encoded list of APIEntrys with one entry per key", func() {
                        req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?key=a&key=b", nil)
                        clusterFacade.defaultGetResponseError = nil
                        clusterFacade.defaultGetResponse = []*SiblingSet{ nil, nil }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        var entries []APIEntry

                        Expect(json.Unmarshal(rr.Body.Bytes(), &entries)).Should(BeNil())
                        Expect(entries).Should(Equal([]APIEntry{ 
                            APIEntry{ Prefix: "", Key: "a", Siblings: nil, Context: "" },
                            APIEntry{ Prefix: "", Key: "b", Siblings: nil, Context: "" },
                        }))
                    })
                })
            })

            Context("When the request includes one or more \"prefix\" parameters", func() {
                It("Should call GetMatches() on the node facade with the specified site, bucket, and keys", func() {
                    req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                    getMatchesCalled := make(chan int, 1)
                    clusterFacade.defaultGetMatchesResponse = nil
                    clusterFacade.defaultGetMatchesResponse = NewMemorySiblingSetIterator()
                    clusterFacade.getMatchesCB = func(siteID string, bucket string, keys [][]byte) {
                        Expect(siteID).Should(Equal("site1"))
                        Expect(bucket).Should(Equal("default"))
                        Expect(keys).Should(Equal([][]byte{ []byte("a"), []byte("b") }))

                        getMatchesCalled <- 1
                    }

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    select {
                    case <-getMatchesCalled:
                    default:
                        Fail("Request did not cause GetMatches() to be invoked")
                    }
                })

                Context("And if GetMatches() returns an error", func() {
                    Context("And the error is ENoSuchSite", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultGetMatchesResponseError = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with an ESiteDoesNotExist body", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultGetMatchesResponseError = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var encodedDBError DBerror

                            Expect(json.Unmarshal(rr.Body.Bytes(), &encodedDBError)).Should(BeNil())
                            Expect(encodedDBError).Should(Equal(ESiteDoesNotExist))
                        })
                    })

                    Context("And the error is ENoSuchBucket", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultGetMatchesResponseError = ENoSuchBucket

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with an ENoSuchBucket body", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultGetMatchesResponseError = ENoSuchBucket

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var encodedDBError DBerror

                            Expect(json.Unmarshal(rr.Body.Bytes(), &encodedDBError)).Should(BeNil())
                            Expect(encodedDBError).Should(Equal(EBucketDoesNotExist))
                        })
                    })

                    Context("And the error is ENoQuorum", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultGetMatchesResponseError = ENoQuorum

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })

                        It("Should respond with an ENoQuorum body", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultGetMatchesResponseError = ENoQuorum

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var encodedDBError DBerror

                            Expect(json.Unmarshal(rr.Body.Bytes(), &encodedDBError)).Should(BeNil())
                            Expect(encodedDBError).Should(Equal(ENoQuorum))
                        })
                    })

                    Context("Otherwise", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultGetMatchesResponseError = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })

                        It("Should respond with an EStorage body", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultGetMatchesResponseError = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var encodedDBError DBerror

                            Expect(json.Unmarshal(rr.Body.Bytes(), &encodedDBError)).Should(BeNil())
                            Expect(encodedDBError).Should(Equal(EStorage))
                        })
                    })
                })

                Context("And if GetMatches() is successful", func() {
                    Context("And the returned iterator does not encounter an error", func() {
                        It("Should respond with status code http.StatusOK", func() {
                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultGetMatchesResponseError = nil
                            clusterFacade.defaultGetMatchesResponse = NewMemorySiblingSetIterator()

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusOK))
                        })

                        It("Should respond with a JSON-encoded list of APIEntrys objects, filtering out any entries that are tombstones", func() {
                            sibling := NewSibling(NewDVV(NewDot("", 0), map[string]uint64{ }), []byte("value"), 0)
                            defaultSiblingSet := NewSiblingSet(map[*Sibling]bool{ sibling: true })
                            tombstoneSet := NewSiblingSet(map[*Sibling]bool{ NewSibling(NewDVV(NewDot("", 0), map[string]uint64{ }), nil, 0): true })

                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultGetMatchesResponseError = nil
                            memorySiblingSetIterator := NewMemorySiblingSetIterator()
                            clusterFacade.defaultGetMatchesResponse = memorySiblingSetIterator
                            memorySiblingSetIterator.AppendNext([]byte("a"), []byte("asdf"), defaultSiblingSet, nil)
                            memorySiblingSetIterator.AppendNext([]byte("b"), []byte("bxyz"), tombstoneSet, nil)
                            memorySiblingSetIterator.AppendNext([]byte("c"), []byte("bxyz"), defaultSiblingSet, nil)

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var entries []APIEntry

                            Expect(json.Unmarshal(rr.Body.Bytes(), &entries)).Should(BeNil())
                            Expect(entries).Should(Equal([]APIEntry{ 
                                APIEntry{ Prefix: "a", Key: "asdf", Siblings: []string{ "value" }, Context: "e30=" },
                                APIEntry{ Prefix: "c", Key: "bxyz", Siblings: []string{ "value" }, Context: "e30=" },
                            }))
                        })
                    })

                    Context("And the returned iterator encounters an error", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            sibling := NewSibling(NewDVV(NewDot("", 0), map[string]uint64{ }), []byte("value"), 0)
                            defaultSiblingSet := NewSiblingSet(map[*Sibling]bool{ sibling: true })
                            tombstoneSet := NewSiblingSet(map[*Sibling]bool{ NewSibling(NewDVV(NewDot("", 0), map[string]uint64{ }), nil, 0): true })

                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultGetMatchesResponseError = nil
                            memorySiblingSetIterator := NewMemorySiblingSetIterator()
                            clusterFacade.defaultGetMatchesResponse = memorySiblingSetIterator
                            memorySiblingSetIterator.AppendNext([]byte("a"), []byte("asdf"), defaultSiblingSet, nil)
                            memorySiblingSetIterator.AppendNext([]byte("b"), []byte("xyz"), tombstoneSet, nil)
                            memorySiblingSetIterator.AppendNext(nil, nil, nil, errors.New("Some error"))

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })

                        It("Should respond with an EStorage body", func() {
                            sibling := NewSibling(NewDVV(NewDot("", 0), map[string]uint64{ }), []byte("value"), 0)
                            defaultSiblingSet := NewSiblingSet(map[*Sibling]bool{ sibling: true })
                            tombstoneSet := NewSiblingSet(map[*Sibling]bool{ NewSibling(NewDVV(NewDot("", 0), map[string]uint64{ }), nil, 0): true })

                            req, err := http.NewRequest("GET", "/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultGetMatchesResponseError = nil
                            memorySiblingSetIterator := NewMemorySiblingSetIterator()
                            clusterFacade.defaultGetMatchesResponse = memorySiblingSetIterator
                            memorySiblingSetIterator.AppendNext([]byte("a"), []byte("asdf"), defaultSiblingSet, nil)
                            memorySiblingSetIterator.AppendNext([]byte("b"), []byte("xyz"), tombstoneSet, nil)
                            memorySiblingSetIterator.AppendNext(nil, nil, nil, errors.New("Some error"))

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var encodedDBError DBerror

                            Expect(json.Unmarshal(rr.Body.Bytes(), &encodedDBError)).Should(BeNil())
                            Expect(encodedDBError).Should(Equal(EStorage))
                        })
                    })
                })
            })
        })
    })
})
