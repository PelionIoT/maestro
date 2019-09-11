package routes_test

import (
    "errors"
    "encoding/json"
    "net/http"
    "net/http/httptest"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    . "github.com/armPelionEdge/devicedb/routes"

    "github.com/gorilla/mux"
)

var _ = Describe("Snapshot", func() {
    var router *mux.Router
    var snapshotEndpoint *SnapshotEndpoint
    var clusterFacade *MockClusterFacade

    BeforeEach(func() {
        clusterFacade = &MockClusterFacade{ }
        router = mux.NewRouter()
        snapshotEndpoint = &SnapshotEndpoint{
            ClusterFacade: clusterFacade,
        }
        snapshotEndpoint.Attach(router)
    })

    Describe("/snapshot", func() {
        Describe("POST", func() {
            Context("When LocalSnapshot() returns an error", func() {
                It("Should respond with status code http.StatusInternalServerError", func() {
                    clusterFacade.defaultLocalSnapshotError = errors.New("Some error")

                    req, err := http.NewRequest("POST", "/snapshot", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                })
            })

            Context("When LocalSnapshot() does not return an error", func() {
                It("Should respond with status code http.StatusOK", func() {
                    req, err := http.NewRequest("POST", "/snapshot", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusOK))
                })

                It("Should respond with a body that is the JSON encoded snapshot returned by LocalSnapshot()", func() {
                    clusterFacade.defaultLocalSnapshotResponse = Snapshot{
                        UUID: "abc",
                        Status: SnapshotProcessing,
                    }

                    req, err := http.NewRequest("POST", "/snapshot", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    var snapshot Snapshot

                    Expect(json.Unmarshal(rr.Body.Bytes(), &snapshot)).Should(BeNil())
                    Expect(snapshot).Should(Equal(clusterFacade.defaultLocalSnapshotResponse))
                })
            })
        })
    })
})
