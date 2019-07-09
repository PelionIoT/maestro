package client_relay_test

import (
    "fmt"
    "time"
    "bytes"
    "crypto/tls"
    "crypto/x509"
    "io/ioutil"
    "errors"
    "context"
    
    . "github.com/armPelionEdge/devicedb/server"
    . "github.com/armPelionEdge/devicedb/util"
    ddbSync "github.com/armPelionEdge/devicedb/sync"
    "github.com/armPelionEdge/devicedb/client_relay"
    clientlib "github.com/armPelionEdge/devicedb/client"
    

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

const SYNC_PERIOD_MS = 10

func url(u string, server *Server) string {
    return "http://localhost:" + fmt.Sprintf("%d", server.Port()) + u
}

func buffer(j string) *bytes.Buffer {
    return bytes.NewBuffer([]byte(j))
}

var _ = Describe("Client", func() {
    var client client_relay.Client
    var server *Server
    var hub *Hub
    var syncController *SyncController
    stop := make(chan int)
    
    BeforeEach(func() {
        _, clientTLS, err := loadCerts("WWRL000000")

        Expect(err).Should(Not(HaveOccurred()))

        syncController = NewSyncController(2, nil, ddbSync.NewPeriodicSyncScheduler(SYNC_PERIOD_MS), 1000)
        hub = NewHub("", syncController, clientTLS)

        client = client_relay.New(client_relay.Config{
            ServerURI: "http://localhost:8080",
        })

        server, _ = NewServer(ServerConfig{
            DBFile: "/tmp/testdb-" + RandomString(),
            Port: 8080,
            Hub: hub,
        })
        
        go func() {
            server.Start()
            stop <- 1
        }()
        
        time.Sleep(time.Millisecond * 100)
    })
    
    AfterEach(func() {
        server.Stop()
        <-stop
    })

    Describe("Watcher", func() {
        It("Should work", func() {
            ctx, cancel := context.WithCancel(context.Background())
            updates, errors := client.Watch(ctx, "default", []string{ "a" }, []string{ }, 0)

            go func() {
                for {
                    select {
                    case <-time.After(time.Second):
                    case <-ctx.Done():
                        return
                    }

                    batch := clientlib.NewBatch()
                    batch.Put("a", "b", "")
                    client.Batch(ctx, "default", *batch)
                }              
            }()

            go func() {
                <-time.After(time.Second * 10)
                cancel()
            }()

            for {
                select {
                case update, ok := <-updates:
                    if !ok {
                        return
                    }

                    fmt.Printf("Update received: Key: %s, Serial: %d Empty: %v\n", update.Key, update.Serial, update.IsEmpty())
                case err, ok := <-errors:
                    if !ok {
                        return
                    }

                    fmt.Printf("Error received: %v\n", err)
                }
            }
            
            fmt.Printf("Watcher closed\n")            
        })
    })
    
    Describe("CRUD", func() {
        It("Should work", func() {
            batch := clientlib.NewBatch()
            batch.Put("a", "b", "")
            batch.Put("aa", "c", "")
            batch.Put("aaa", "d", "")
            batch.Put("aaaa", "e", "")            
            batch.Put("x", "c", "")
            
            Expect(client.Batch(context.TODO(), "default", *batch)).Should(BeNil())

            result, err := client.Get(context.TODO(), "default", []string{ "a", "x" })

            Expect(err).Should(BeNil())
            Expect(result[0].Siblings).Should(Equal([]string{ "b" }))
            Expect(result[1].Siblings).Should(Equal([]string{ "c" }))
            
            batch = clientlib.NewBatch()
            batch.Delete("a", "")

            Expect(client.Batch(context.TODO(), "default", *batch)).Should(BeNil())            

            result, err = client.Get(context.TODO(), "default", []string{ "a", "x" })
            Expect(err).Should(BeNil())
            Expect(result[0].Siblings).Should(Equal([]string{ }))
            Expect(result[1].Siblings).Should(Equal([]string{ "c" }))

            result, err = client.Get(context.TODO(), "default", []string{ "y", "z" })
            Expect(err).Should(BeNil())
            Expect(result[0]).Should(BeNil())
            Expect(result[1]).Should(BeNil())

            iter, err := client.GetMatches(context.TODO(), "default", []string{ "a" })

            Expect(err).Should(BeNil())
            Expect(iter.Next()).Should(BeTrue())
            Expect(iter.Prefix()).Should(Equal("a"))
            Expect(iter.Key()).Should(Equal("aa"))
            Expect(iter.Entry().Siblings).Should(Equal([]string{ "c" }))
            Expect(iter.Next()).Should(BeTrue())
            Expect(iter.Prefix()).Should(Equal("a"))
            Expect(iter.Key()).Should(Equal("aaa"))
            Expect(iter.Entry().Siblings).Should(Equal([]string{ "d" }))
            Expect(iter.Next()).Should(BeTrue())
            Expect(iter.Prefix()).Should(Equal("a"))
            Expect(iter.Key()).Should(Equal("aaaa"))
            Expect(iter.Entry().Siblings).Should(Equal([]string{ "e" }))            
            Expect(iter.Next()).Should(BeFalse())
            Expect(iter.Key()).Should(Equal(""))
            Expect(iter.Prefix()).Should(Equal(""))
            Expect(iter.Entry()).Should(Equal(clientlib.Entry{}))
        })
    })
})
