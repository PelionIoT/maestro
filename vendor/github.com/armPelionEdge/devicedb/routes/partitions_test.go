package routes_test

import (
    "errors"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"

    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/routes"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "github.com/gorilla/mux"
)

var _ = Describe("Partitions", func() {
    var router *mux.Router
    var partitionsEndpoint *PartitionsEndpoint
    var clusterFacade *MockClusterFacade

    BeforeEach(func() {
        clusterFacade = &MockClusterFacade{ }
        router = mux.NewRouter()
        partitionsEndpoint = &PartitionsEndpoint{
            ClusterFacade: clusterFacade,
        }
        partitionsEndpoint.Attach(router)
    })

    Describe("/partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/merges", func() {
        Describe("POST", func() {
            Context("When the provided body of the request cannot be parsed as a map[string]*SiblingSet", func() {
                It("Should respond with status code http.StatusBadRequest", func() {
                    req, err := http.NewRequest("POST", "/partitions/45/sites/site1/buckets/default/merges", strings.NewReader("asdf"))

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusBadRequest))
                })
            })

            Context("When the partition ID cannot be parsed as a base 10 encoded uint64", func() {
                It("Should respond with status code http.StatusBadRequest", func() {
                    patch := map[string]*SiblingSet{ }
                    encodedPatch, err := json.Marshal(patch)

                    Expect(err).Should(BeNil())

                    req, err := http.NewRequest("POST", "/partitions/badpartitionid/sites/site1/buckets/default/merges", strings.NewReader(string(encodedPatch)))

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusBadRequest))
                })
            })

            Context("When the body and partition ID are parsed without error", func() {
                Context("And a \"broadcast\" query parameter is set", func() {
                    It("Should invoke LocalMerge() using the partition, site, bucket, and patch passed into the request and set broadcastToRelays to true", func() {
                        patch := map[string]*SiblingSet{ }
                        encodedPatch, err := json.Marshal(patch)

                        Expect(err).Should(BeNil())

                        req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/merges?broadcast=true", strings.NewReader(string(encodedPatch)))
                        localMergeCalled := make(chan int, 1)
                        clusterFacade.localMergeCB = func(partition uint64, siteID string, bucket string, patch map[string]*SiblingSet, broadcastToRelays bool) {
                            Expect(partition).Should(Equal(uint64(68)))
                            Expect(siteID).Should(Equal("site1"))
                            Expect(bucket).Should(Equal("default"))
                            Expect(broadcastToRelays).Should(BeTrue())

                            localMergeCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        select {
                        case <-localMergeCalled:
                        default:
                            Fail("Request did not cause LocalMerge() to be invoked")
                        }
                    })
                })

                Context("And a \"broadcast\" query parameter is not set", func() {
                    It("Should invoke LocalMerge() using the partition, site, bucket, and patch passed into the request and set broadcastToRelays to false", func() {
                        patch := map[string]*SiblingSet{ }
                        encodedPatch, err := json.Marshal(patch)

                        Expect(err).Should(BeNil())

                        req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/merges", strings.NewReader(string(encodedPatch)))
                        localMergeCalled := make(chan int, 1)
                        clusterFacade.localMergeCB = func(partition uint64, siteID string, bucket string, patch map[string]*SiblingSet, broadcastToRelays bool) {
                            Expect(partition).Should(Equal(uint64(68)))
                            Expect(siteID).Should(Equal("site1"))
                            Expect(bucket).Should(Equal("default"))
                            Expect(broadcastToRelays).Should(BeFalse())

                            localMergeCalled <- 1
                        }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        select {
                        case <-localMergeCalled:
                        default:
                            Fail("Request did not cause LocalMerge() to be invoked")
                        }
                    })
                })

                Context("And LocalMerge() returns an error", func() {
                    Context("And the error is ENoSuchPartition", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            patch := map[string]*SiblingSet{ }
                            encodedPatch, err := json.Marshal(patch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/merges", strings.NewReader(string(encodedPatch)))
                            clusterFacade.defaultLocalMergeResponse = ENoSuchPartition

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })
                    })

                    Context("And the error is ENoSuchSite", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            patch := map[string]*SiblingSet{ }
                            encodedPatch, err := json.Marshal(patch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/merges", strings.NewReader(string(encodedPatch)))
                            clusterFacade.defaultLocalMergeResponse = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with body as JSON-encoded error ESiteDoesNotExist", func() {
                            patch := map[string]*SiblingSet{ }
                            encodedPatch, err := json.Marshal(patch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/merges", strings.NewReader(string(encodedPatch)))
                            clusterFacade.defaultLocalMergeResponse = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            dbErr, err := DBErrorFromJSON(rr.Body.Bytes())

                            Expect(err).Should(BeNil())
                            Expect(dbErr).Should(Equal(ESiteDoesNotExist))
                        })
                    })

                    Context("And the error is ENoSuchBucket", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            patch := map[string]*SiblingSet{ }
                            encodedPatch, err := json.Marshal(patch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/merges", strings.NewReader(string(encodedPatch)))
                            clusterFacade.defaultLocalMergeResponse = ENoSuchBucket

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with body as JSON-encoded error EBucketDoesNotExist", func() {
                            patch := map[string]*SiblingSet{ }
                            encodedPatch, err := json.Marshal(patch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/merges", strings.NewReader(string(encodedPatch)))
                            clusterFacade.defaultLocalMergeResponse = ENoSuchBucket

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            dbErr, err := DBErrorFromJSON(rr.Body.Bytes())

                            Expect(err).Should(BeNil())
                            Expect(dbErr).Should(Equal(EBucketDoesNotExist))
                        })
                    })

                    // ENoQuorum should indicate a case where the batch was applied locally but should
                    // not count toward the write quorum because the local node is currently in the process of
                    // obtaining a copy of that partition's data
                    Context("And the error is ENoQuorum", func() {
                        It("Should respond with status code http.StatusOK", func() {
                            patch := map[string]*SiblingSet{ }
                            encodedPatch, err := json.Marshal(patch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/merges", strings.NewReader(string(encodedPatch)))
                            clusterFacade.defaultLocalMergeResponse = ENoQuorum

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusOK))
                        })

                        It("Should respond with a JSON-encoded BatchResult body where NApplied is set to 0", func() {
                            patch := map[string]*SiblingSet{ }
                            encodedPatch, err := json.Marshal(patch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/merges", strings.NewReader(string(encodedPatch)))
                            clusterFacade.defaultLocalMergeResponse = ENoQuorum

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var batchResult BatchResult

                            Expect(json.Unmarshal(rr.Body.Bytes(), &batchResult)).Should(BeNil())
                            Expect(batchResult.NApplied).Should(Equal(uint64(0)))
                        })
                    })

                    Context("Otherwise", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            patch := map[string]*SiblingSet{ }
                            encodedPatch, err := json.Marshal(patch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/merges", strings.NewReader(string(encodedPatch)))
                            clusterFacade.defaultLocalMergeResponse = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })
                    })
                })

                Context("And LocalBatch() is successful", func() {
                    It("Should respond with status code http.StatusOK", func() {
                        patch := map[string]*SiblingSet{ }
                        encodedPatch, err := json.Marshal(patch)

                        Expect(err).Should(BeNil())

                        req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/merges", strings.NewReader(string(encodedPatch)))
                        clusterFacade.defaultLocalMergeResponse = nil

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        Expect(rr.Code).Should(Equal(http.StatusOK))
                    })
                    
                    It("Should respond with a JSON-encoded BatchResult body where NApplied is set to 1", func() {
                        patch := map[string]*SiblingSet{ }
                        encodedPatch, err := json.Marshal(patch)

                        Expect(err).Should(BeNil())

                        req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/merges", strings.NewReader(string(encodedPatch)))
                        clusterFacade.defaultLocalMergeResponse = nil

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        var batchResult BatchResult

                        Expect(json.Unmarshal(rr.Body.Bytes(), &batchResult)).Should(BeNil())
                        Expect(batchResult.NApplied).Should(Equal(uint64(1)))
                    })
                })
            })
        })
    })

    Describe("/partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/batches", func() {
        Describe("POST", func() {
            Context("When the provided body of the request cannot be parsed as an UpdateBatch", func() {
                It("Should respond with status code http.StatusBadRequest", func() {
                    req, err := http.NewRequest("POST", "/partitions/45/sites/site1/buckets/default/batches", strings.NewReader("asdf"))

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusBadRequest))
                })
            })

            Context("When the partition ID cannot be parsed as a base 10 encoded uint64", func() {
                It("Should respond with status code http.StatusBadRequest", func() {
                    updateBatch := NewUpdateBatch()
                    encodedUpdateBatch, err := updateBatch.ToJSON()

                    Expect(err).Should(BeNil())

                    req, err := http.NewRequest("POST", "/partitions/badpartitionid/sites/site1/buckets/default/batches", strings.NewReader(string(encodedUpdateBatch)))

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusBadRequest))
                })
            })

            Context("When the body and partition ID are parsed without error", func() {
                It("Should invoke LocalBatch() using the partition, site, bucket, and update batch passed into the request", func() {
                    updateBatch := NewUpdateBatch()
                    encodedUpdateBatch, err := updateBatch.ToJSON()

                    Expect(err).Should(BeNil())

                    req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/batches", strings.NewReader(string(encodedUpdateBatch)))
                    localBatchCalled := make(chan int, 1)
                    clusterFacade.localBatchCB = func(partition uint64, siteID string, bucket string, updateBatch *UpdateBatch) {
                        Expect(partition).Should(Equal(uint64(68)))
                        Expect(siteID).Should(Equal("site1"))
                        Expect(bucket).Should(Equal("default"))

                        localBatchCalled <- 1
                    }

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    select {
                    case <-localBatchCalled:
                    default:
                        Fail("Request did not cause LocalBatch() to be invoked")
                    }
                })

                Context("And LocalBatch() returns an error", func() {
                    Context("And the error is ENoSuchPartition", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            updateBatch := NewUpdateBatch()
                            encodedUpdateBatch, err := updateBatch.ToJSON()

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/batches", strings.NewReader(string(encodedUpdateBatch)))
                            clusterFacade.defaultLocalBatchError = ENoSuchPartition

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })
                    })

                    Context("And the error is ENoSuchSite", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            updateBatch := NewUpdateBatch()
                            encodedUpdateBatch, err := updateBatch.ToJSON()

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/batches", strings.NewReader(string(encodedUpdateBatch)))
                            clusterFacade.defaultLocalBatchError = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with body as JSON-encoded error ESiteDoesNotExist", func() {
                            updateBatch := NewUpdateBatch()
                            encodedUpdateBatch, err := updateBatch.ToJSON()

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/batches", strings.NewReader(string(encodedUpdateBatch)))
                            clusterFacade.defaultLocalBatchError = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            dbErr, err := DBErrorFromJSON(rr.Body.Bytes())

                            Expect(err).Should(BeNil())
                            Expect(dbErr).Should(Equal(ESiteDoesNotExist))
                        })
                    })

                    Context("And the error is ENoSuchBucket", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            updateBatch := NewUpdateBatch()
                            encodedUpdateBatch, err := updateBatch.ToJSON()

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/batches", strings.NewReader(string(encodedUpdateBatch)))
                            clusterFacade.defaultLocalBatchError = ENoSuchBucket

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with body as JSON-encoded error EBucketDoesNotExist", func() {
                            updateBatch := NewUpdateBatch()
                            encodedUpdateBatch, err := updateBatch.ToJSON()

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/batches", strings.NewReader(string(encodedUpdateBatch)))
                            clusterFacade.defaultLocalBatchError = ENoSuchBucket

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            dbErr, err := DBErrorFromJSON(rr.Body.Bytes())

                            Expect(err).Should(BeNil())
                            Expect(dbErr).Should(Equal(EBucketDoesNotExist))
                        })
                    })

                    // ENoQuorum should indicate a case where the batch was applied locally but should
                    // not count toward the write quorum because the local node is currently in the process of
                    // obtaining a copy of that partition's data
                    Context("And the error is ENoQuorum", func() {
                        It("Should respond with status code http.StatusOK", func() {
                            updateBatch := NewUpdateBatch()
                            encodedUpdateBatch, err := updateBatch.ToJSON()

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/batches", strings.NewReader(string(encodedUpdateBatch)))
                            clusterFacade.defaultLocalBatchError = ENoQuorum

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusOK))
                        })

                        It("Should respond with a JSON-encoded BatchResult body where NApplied is set to 0", func() {
                            updateBatch := NewUpdateBatch()
                            encodedUpdateBatch, err := updateBatch.ToJSON()

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/batches", strings.NewReader(string(encodedUpdateBatch)))
                            clusterFacade.defaultLocalBatchError = ENoQuorum

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var batchResult BatchResult

                            Expect(json.Unmarshal(rr.Body.Bytes(), &batchResult)).Should(BeNil())
                            Expect(batchResult.NApplied).Should(Equal(uint64(0)))
                        })
                    })

                    Context("Otherwise", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            updateBatch := NewUpdateBatch()
                            encodedUpdateBatch, err := updateBatch.ToJSON()

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/batches", strings.NewReader(string(encodedUpdateBatch)))
                            clusterFacade.defaultLocalBatchError = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })
                    })
                })

                Context("And LocalBatch() is successful", func() {
                    It("Should respond with status code http.StatusOK", func() {
                        updateBatch := NewUpdateBatch()
                        encodedUpdateBatch, err := updateBatch.ToJSON()

                        Expect(err).Should(BeNil())

                        req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/batches", strings.NewReader(string(encodedUpdateBatch)))
                        clusterFacade.defaultLocalBatchError = nil

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        Expect(rr.Code).Should(Equal(http.StatusOK))
                    })
                    
                    It("Should respond with a JSON-encoded BatchResult body where NApplied is set to 1 and Patch is the patch returned from LocalBatch()", func() {
                        updateBatch := NewUpdateBatch()
                        encodedUpdateBatch, err := updateBatch.ToJSON()

                        Expect(err).Should(BeNil())

                        req, err := http.NewRequest("POST", "/partitions/68/sites/site1/buckets/default/batches", strings.NewReader(string(encodedUpdateBatch)))
                        clusterFacade.defaultLocalBatchPatch = map[string]*SiblingSet{ }
                        clusterFacade.defaultLocalBatchError = nil

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        var batchResult BatchResult

                        Expect(json.Unmarshal(rr.Body.Bytes(), &batchResult)).Should(BeNil())
                        Expect(batchResult.NApplied).Should(Equal(uint64(1)))
                        Expect(batchResult.Patch).Should(Equal(map[string]*SiblingSet{ }))
                    })
                })
            })
        })
    })

    Describe("/partitions/{partitionID}/sites/{siteID}/buckets/{bucketID}/keys", func() {
        Describe("GET", func() {
            Context("When the partition ID cannot be parsed as a base 10 encoded uint64", func() {
                It("Should respond with status code http.StatusBadRequest", func() {
                    req, err := http.NewRequest("GET", "/partitions/badpartition/sites/site1/buckets/default/keys?key=a", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusBadRequest))
                })
            })

            Context("When the request includes both \"key\" and \"prefix\" query parameters", func() {
                It("Should respond with status code http.StatusBadRequest", func() {
                    req, err := http.NewRequest("GET", "/partitions/45/sites/site1/buckets/default/keys?key=key1&prefix=prefix1", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusBadRequest))
                })
            })

            Context("When the request includes neither \"key\" nor \"prefix\" query parameters", func() {
                It("Should respond with status code http.StatusOK", func() {
                    req, err := http.NewRequest("GET", "/partitions/45/sites/site1/buckets/default/keys", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusOK))
                })

                It("Should respond with an empty JSON-encoded list InternalEntry database objects", func() {
                    req, err := http.NewRequest("GET", "/partitions/45/sites/site1/buckets/default/keys", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    var entries []InternalEntry

                    Expect(json.Unmarshal(rr.Body.Bytes(), &entries)).Should(BeNil())
                    Expect(entries).Should(Equal([]InternalEntry{ }))
                })
            })

            Context("When the request includes one or more \"key\" parameters", func() {
                It("Should call LocalGet() on the node facade with the specified site, bucket and keys", func() {
                    req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?key=a&key=b", nil)
                    localGetCalled := make(chan int, 1)
                    clusterFacade.defaultLocalGetResponse = []*SiblingSet{ nil, nil }
                    clusterFacade.localGetCB = func(partition uint64, siteID string, bucket string, keys [][]byte) {
                        Expect(partition).Should(Equal(uint64(68)))
                        Expect(siteID).Should(Equal("site1"))
                        Expect(bucket).Should(Equal("default"))
                        Expect(keys).Should(Equal([][]byte{ []byte("a"), []byte("b") }))

                        localGetCalled <- 1
                    }

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    select {
                    case <-localGetCalled:
                    default:
                        Fail("Request did not cause LocalGet() to be invoked")
                    }
                })

                Context("And if LocalGet() returns an error", func() {
                    Context("And the error is ENoSuchSite", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?key=a&key=b", nil)
                            clusterFacade.defaultLocalGetResponseError = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with body as JSON-encoded error EBucketDoesNotExist", func() {
                            req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?key=a&key=b", nil)
                            clusterFacade.defaultLocalGetResponseError = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            dbErr, err := DBErrorFromJSON(rr.Body.Bytes())

                            Expect(err).Should(BeNil())
                            Expect(dbErr).Should(Equal(ESiteDoesNotExist))
                        })
                    })

                    Context("And the error is ENoSuchBucket", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?key=a&key=b", nil)
                            clusterFacade.defaultLocalGetResponseError = ENoSuchBucket

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with body as JSON-encoded error EBucketDoesNotExist", func() {
                            req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?key=a&key=b", nil)
                            clusterFacade.defaultLocalGetResponseError = ENoSuchBucket

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            dbErr, err := DBErrorFromJSON(rr.Body.Bytes())

                            Expect(err).Should(BeNil())
                            Expect(dbErr).Should(Equal(EBucketDoesNotExist))
                        })
                    })

                    Context("Otherwise", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?key=a&key=b", nil)
                            clusterFacade.defaultLocalGetResponseError = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })
                    })
                })

                Context("And if LocalGet() is successful", func() {
                    It("Should respond with status code http.StatusOK", func() {
                        req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?key=a&key=b", nil)
                        clusterFacade.defaultLocalGetResponseError = nil
                        clusterFacade.defaultLocalGetResponse = []*SiblingSet{ nil, nil }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        Expect(rr.Code).Should(Equal(http.StatusOK))
                    })

                    It("Should respond with a JSON-encoded list of InternalEntries with one entry per key", func() {
                        req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?key=a&key=b", nil)
                        clusterFacade.defaultLocalGetResponseError = nil
                        clusterFacade.defaultLocalGetResponse = []*SiblingSet{ nil, nil }

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        var entries []InternalEntry

                        Expect(json.Unmarshal(rr.Body.Bytes(), &entries)).Should(BeNil())
                        Expect(entries).Should(Equal([]InternalEntry{ 
                            InternalEntry{ Prefix: "", Key: "a", Siblings: nil },
                            InternalEntry{ Prefix: "", Key: "b", Siblings: nil },
                        }))
                    })
                })
            })

            Context("When the request includes one or more \"prefix\" parameters", func() {
                It("Should call LocalGetMatches() on the node facade with the specified site, bucket, and keys", func() {
                    req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                    localGetMatchesCalled := make(chan int, 1)
                    clusterFacade.defaultLocalGetMatchesResponse = NewMemorySiblingSetIterator()
                    clusterFacade.localGetMatchesCB = func(partition uint64, siteID string, bucket string, keys [][]byte) {
                        Expect(partition).Should(Equal(uint64(68)))
                        Expect(siteID).Should(Equal("site1"))
                        Expect(bucket).Should(Equal("default"))
                        Expect(keys).Should(Equal([][]byte{ []byte("a"), []byte("b") }))

                        localGetMatchesCalled <- 1
                    }

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    select {
                    case <-localGetMatchesCalled:
                    default:
                        Fail("Request did not cause LocalGetMatches() to be invoked")
                    }
                })

                Context("And if LocalGetMatches() returns an error", func() {
                    Context("And the error is ENoSuchSite", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultLocalGetMatchesResponseError = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with body as JSON-encoded error EBucketDoesNotExist", func() {
                            req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultLocalGetMatchesResponseError = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            dbErr, err := DBErrorFromJSON(rr.Body.Bytes())

                            Expect(err).Should(BeNil())
                            Expect(dbErr).Should(Equal(ESiteDoesNotExist))
                        })
                    })

                    Context("And the error is ENoSuchBucket", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultLocalGetMatchesResponseError = ENoSuchBucket

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with body as JSON-encoded error EBucketDoesNotExist", func() {
                            req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultLocalGetMatchesResponseError = ENoSuchBucket

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            dbErr, err := DBErrorFromJSON(rr.Body.Bytes())

                            Expect(err).Should(BeNil())
                            Expect(dbErr).Should(Equal(EBucketDoesNotExist))
                        })
                    })

                    Context("Otherwise", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultLocalGetMatchesResponseError = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })
                    })
                })

                Context("And if LocalGetMatches() is successful", func() {
                    Context("And the returned iterator does not encounter an error", func() {
                        It("Should respond with status code http.StatusOK", func() {
                            req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultLocalGetMatchesResponseError = nil
                            memorySiblingSetIterator := NewMemorySiblingSetIterator()
                            clusterFacade.defaultLocalGetMatchesResponse = memorySiblingSetIterator
                            memorySiblingSetIterator.AppendNext([]byte("a"), []byte("asdf"), nil, nil)
                            memorySiblingSetIterator.AppendNext([]byte("b"), []byte("xyz"), nil, nil)

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusOK))
                        })

                        It("Should respond with a JSON-encoded list of InternalEntry objects", func() {
                            req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultLocalGetMatchesResponseError = nil
                            memorySiblingSetIterator := NewMemorySiblingSetIterator()
                            clusterFacade.defaultLocalGetMatchesResponse = memorySiblingSetIterator
                            memorySiblingSetIterator.AppendNext([]byte("a"), []byte("asdf"), nil, nil)
                            memorySiblingSetIterator.AppendNext([]byte("b"), []byte("bxyz"), nil, nil)

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var entries []InternalEntry

                            Expect(json.Unmarshal(rr.Body.Bytes(), &entries)).Should(BeNil())
                            Expect(entries).Should(Equal([]InternalEntry{ 
                                InternalEntry{ Prefix: "a", Key: "asdf", Siblings: nil },
                                InternalEntry{ Prefix: "b", Key: "bxyz", Siblings: nil },
                            }))
                        })
                    })

                    Context("And the returned iterator encounters an error", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            req, err := http.NewRequest("GET", "/partitions/68/sites/site1/buckets/default/keys?prefix=a&prefix=b", nil)
                            clusterFacade.defaultLocalGetMatchesResponseError = nil
                            memorySiblingSetIterator := NewMemorySiblingSetIterator()
                            clusterFacade.defaultLocalGetMatchesResponse = memorySiblingSetIterator
                            memorySiblingSetIterator.AppendNext([]byte("a"), []byte("asdf"), nil, nil)
                            memorySiblingSetIterator.AppendNext([]byte("b"), []byte("xyz"), nil, nil)
                            memorySiblingSetIterator.AppendNext(nil, nil, nil, errors.New("Some error"))

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })
                    })
                })
            })
        })
    })
})
