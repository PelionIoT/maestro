package routes_test

import (
    "fmt"
    "net/http"
    "net/url"
    "time"

    . "github.com/armPelionEdge/devicedb/routes"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "github.com/onsi/gomega/ghttp"
)

var _ = Describe("Sync", func() {
    var router *mux.Router
    var syncEndpoint *SyncEndpoint
    var clusterFacade *MockClusterFacade
    var server *ghttp.Server

    BeforeEach(func() {
        clusterFacade = &MockClusterFacade{ }
        router = mux.NewRouter()
        syncEndpoint = &SyncEndpoint{
            ClusterFacade: clusterFacade,
            Upgrader: websocket.Upgrader{
                ReadBufferSize:  1024,
                WriteBufferSize: 1024,
            },
        }
    
        syncEndpoint.Attach(router)
        server = ghttp.NewServer()
        server.AppendHandlers(func(w http.ResponseWriter, r *http.Request) {
            router.ServeHTTP(w, r)
        })
    })

    Describe("/sync", func() {
        Describe("GET", func() {
            It("Should call AcceptRelayConnection() on the node facade", func() {
                acceptRelayConnectionCalled := make(chan int, 1)
                clusterFacade.acceptRelayConnectionCB = func(conn *websocket.Conn) {
                    acceptRelayConnectionCalled <- 1
                }

                dialer := &websocket.Dialer{ }
                u, err := url.Parse(server.URL())

                Expect(err).Should(Not(HaveOccurred()))

                _, _, err = dialer.Dial(fmt.Sprintf("ws://%s/sync", u.Host), nil)

                Expect(err).Should(Not(HaveOccurred()))

                select {
                case <-acceptRelayConnectionCalled:
                case <-time.After(time.Second):
                    Fail("Should have invoked AcceptRelayConnection()")
                }
            })
        })
    })
})
