package routes_test

import (
    "context"
    "encoding/json"
    "errors"
    "net/http"
    "net/http/httptest"
    "strings"

    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/routes"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "github.com/gorilla/mux"
)

var _ = Describe("Relays", func() {
    var router *mux.Router
    var relaysEndpoint *RelaysEndpoint
    var clusterFacade *MockClusterFacade

    BeforeEach(func() {
        clusterFacade = &MockClusterFacade{ }
        router = mux.NewRouter()
        relaysEndpoint = &RelaysEndpoint{
            ClusterFacade: clusterFacade,
        }
        relaysEndpoint.Attach(router)
    })

    Describe("/relays/{relayID}", func() {
        Describe("PATCH", func() {
            Context("When the request body cannot be parsed as a RelaySettingsPatch", func() {
                It("Should respond with status code http.StatusBadRequest", func() {
                    req, err := http.NewRequest("PATCH", "/relays/WWRL000000", strings.NewReader("asdf"))

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusBadRequest))
                })
            })

            Context("When the request body is successfully parsed", func() {
                It("Should call MoveRelay() on the node facade using the relay ID provide in the path and siteID provided in the body", func() {
                    var relaySettingsPatch RelaySettingsPatch = RelaySettingsPatch{
                        Site: "site1",
                    }

                    encodedRelaySettingsPatch, err := json.Marshal(&relaySettingsPatch)

                    Expect(err).Should(BeNil())

                    req, err := http.NewRequest("PATCH", "/relays/WWRL000000", strings.NewReader(string(encodedRelaySettingsPatch)))

                    moveRelayCalled := make(chan int, 1)
                    clusterFacade.moveRelayCB = func(ctx context.Context, relayID string, siteID string) {
                        Expect(relayID).Should(Equal("WWRL000000"))
                        Expect(siteID).Should(Equal("site1"))
                        moveRelayCalled <- 1
                    }

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    select {
                    case <-moveRelayCalled:
                    default:
                        Fail("Should have invoked MoveRelay()")
                    }
                })

                Context("And if MoveRelay() returns an error", func() {
                    Context("And the error is ENoSuchRelay", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            var relaySettingsPatch RelaySettingsPatch = RelaySettingsPatch{
                                Site: "site1",
                            }

                            encodedRelaySettingsPatch, err := json.Marshal(&relaySettingsPatch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("PATCH", "/relays/WWRL000000", strings.NewReader(string(encodedRelaySettingsPatch)))

                            clusterFacade.defaultMoveRelayResponse = ENoSuchRelay

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with error body ERelayDoesNotExist", func() {
                            var relaySettingsPatch RelaySettingsPatch = RelaySettingsPatch{
                                Site: "site1",
                            }

                            encodedRelaySettingsPatch, err := json.Marshal(&relaySettingsPatch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("PATCH", "/relays/WWRL000000", strings.NewReader(string(encodedRelaySettingsPatch)))

                            clusterFacade.defaultMoveRelayResponse = ENoSuchRelay

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var encodedDBError DBerror

                            Expect(json.Unmarshal(rr.Body.Bytes(), &encodedDBError)).Should(BeNil())
                            Expect(encodedDBError).Should(Equal(ERelayDoesNotExist))
                        })
                    })

                    Context("And the error is ENoSuchSite", func() {
                        It("Should respond with status code http.StatusNotFound", func() {
                            var relaySettingsPatch RelaySettingsPatch = RelaySettingsPatch{
                                Site: "site1",
                            }

                            encodedRelaySettingsPatch, err := json.Marshal(&relaySettingsPatch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("PATCH", "/relays/WWRL000000", strings.NewReader(string(encodedRelaySettingsPatch)))

                            clusterFacade.defaultMoveRelayResponse = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusNotFound))
                        })

                        It("Should respond with error body ESiteDoesNotExist", func() {
                            var relaySettingsPatch RelaySettingsPatch = RelaySettingsPatch{
                                Site: "site1",
                            }

                            encodedRelaySettingsPatch, err := json.Marshal(&relaySettingsPatch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("PATCH", "/relays/WWRL000000", strings.NewReader(string(encodedRelaySettingsPatch)))

                            clusterFacade.defaultMoveRelayResponse = ENoSuchSite

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            var encodedDBError DBerror

                            Expect(json.Unmarshal(rr.Body.Bytes(), &encodedDBError)).Should(BeNil())
                            Expect(encodedDBError).Should(Equal(ESiteDoesNotExist))
                        })
                    })

                    Context("Otherwise", func() {
                        It("Should respond with status code http.StatusInternalServerError", func() {
                            var relaySettingsPatch RelaySettingsPatch = RelaySettingsPatch{
                                Site: "site1",
                            }

                            encodedRelaySettingsPatch, err := json.Marshal(&relaySettingsPatch)

                            Expect(err).Should(BeNil())

                            req, err := http.NewRequest("PATCH", "/relays/WWRL000000", strings.NewReader(string(encodedRelaySettingsPatch)))

                            clusterFacade.defaultMoveRelayResponse = errors.New("Some error")

                            Expect(err).Should(BeNil())

                            rr := httptest.NewRecorder()
                            router.ServeHTTP(rr, req)

                            Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                        })
                    })
                })

                Context("And if MoveRelay() is successful", func() {
                    It("Should respond with status code http.StatusOK", func() {
                        var relaySettingsPatch RelaySettingsPatch = RelaySettingsPatch{
                            Site: "site1",
                        }

                        encodedRelaySettingsPatch, err := json.Marshal(&relaySettingsPatch)

                        Expect(err).Should(BeNil())

                        req, err := http.NewRequest("PATCH", "/relays/WWRL000000", strings.NewReader(string(encodedRelaySettingsPatch)))

                        clusterFacade.defaultMoveRelayResponse = nil

                        Expect(err).Should(BeNil())

                        rr := httptest.NewRecorder()
                        router.ServeHTTP(rr, req)

                        Expect(rr.Code).Should(Equal(http.StatusOK))
                    })
                })
            })
        })

        Describe("PUT", func() {
            It("Should call AddRelay() on the node facade with the relay ID specified in the path", func() {
                req, err := http.NewRequest("PUT", "/relays/WWRL000000", nil)

                addRelayCalled := make(chan int, 1)
                clusterFacade.addRelayCB = func(ctx context.Context, relayID string) {
                    Expect(relayID).Should(Equal("WWRL000000"))
                    addRelayCalled <- 1
                }

                Expect(err).Should(BeNil())

                rr := httptest.NewRecorder()
                router.ServeHTTP(rr, req)

                select {
                case <-addRelayCalled:
                default:
                    Fail("Should have invoked AddRelay()")
                }
            })

            Context("And if AddRelay() returns an error", func() {
                It("Should respond with status code http.StatusInternalServerError", func() {
                    req, err := http.NewRequest("PUT", "/relays/WWRL000000", nil)

                    clusterFacade.defaultAddRelayResponse = errors.New("Some error")

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                })
            })

            Context("And if AddRelay() is successful", func() {
                It("Should respond with status code http.StatusOK", func() {
                    req, err := http.NewRequest("PUT", "/relays/WWRL000000", nil)

                    clusterFacade.defaultAddRelayResponse = nil

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusOK))
                })
            })
        })

        Describe("DELETE", func() {
            It("Should call RemoveRelay() on the node facade with the site ID specified in the path", func() {
                req, err := http.NewRequest("DELETE", "/relays/WWRL000000", nil)

                removeRelayCalled := make(chan int, 1)
                clusterFacade.removeRelayCB = func(ctx context.Context, relayID string) {
                    Expect(relayID).Should(Equal("WWRL000000"))
                    removeRelayCalled <- 1
                }

                Expect(err).Should(BeNil())

                rr := httptest.NewRecorder()
                router.ServeHTTP(rr, req)

                select {
                case <-removeRelayCalled:
                default:
                    Fail("Should have invoked RemoveRelay()")
                }
            })

            Context("And if RemoveRelay() returns an error", func() {
                It("Should respond with staus code http.StatusInternalServerError", func() {
                    req, err := http.NewRequest("DELETE", "/relays/WWRL000000", nil)

                    clusterFacade.defaultRemoveRelayResponse = errors.New("Some error")

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                })
            })

            Context("And if RemoveRelay() is successful", func() {
                It("Should respond with status code http.StatusOK", func() {
                    req, err := http.NewRequest("DELETE", "/relays/WWRL000000", nil)

                    clusterFacade.defaultRemoveRelayResponse = nil

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusOK))
                })
            })
        })
    })
})
