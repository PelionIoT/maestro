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

var _ = Describe("LogDump", func() {
    var router *mux.Router
    var logDumpEndpoint *LogDumpEndpoint
    var clusterFacade *MockClusterFacade

    BeforeEach(func() {
        clusterFacade = &MockClusterFacade{ }
        router = mux.NewRouter()
        logDumpEndpoint = &LogDumpEndpoint{
            ClusterFacade: clusterFacade,
        }
        logDumpEndpoint.Attach(router)
    })

    Describe("/log_dump", func() {
        Describe("GET", func() {
            Context("When LocalLogDump() returns an error", func() {
                It("Should respond with status code http.StatusInternalServerError", func() {
                    clusterFacade.defaultLocalLogDumpError = errors.New("Some error")

                    req, err := http.NewRequest("GET", "/log_dump", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusInternalServerError))
                })
            })

            Context("When LocalLogDump() does not return an error", func() {
                It("Should respond with status code http.StatusOK", func() {
                    req, err := http.NewRequest("GET", "/log_dump", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    Expect(rr.Code).Should(Equal(http.StatusOK))
                })

                It("Should respond with a body that is the JSON encoded log dupm returned by LogDump()", func() {
                    clusterFacade.defaultLocalLogDumpResponse = LogDump{
                        BaseSnapshot: LogSnapshot{ Index: 22 },
                        Entries: []LogEntry{ },
                        CurrentSnapshot: LogSnapshot{ Index: 23 },
                    }

                    req, err := http.NewRequest("GET", "/log_dump", nil)

                    Expect(err).Should(BeNil())

                    rr := httptest.NewRecorder()
                    router.ServeHTTP(rr, req)

                    var logDump LogDump

                    Expect(json.Unmarshal(rr.Body.Bytes(), &logDump)).Should(BeNil())
                    Expect(logDump).Should(Equal(clusterFacade.defaultLocalLogDumpResponse))
                })
            })
        })
    })
})
