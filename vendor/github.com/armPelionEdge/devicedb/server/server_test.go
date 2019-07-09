package server_test

import (
    "fmt"
    "time"
    "net/http"
    "bytes"
    "encoding/json"
    "bufio"
    
    . "github.com/armPelionEdge/devicedb/server"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/util"
    . "github.com/armPelionEdge/devicedb/transport"
    ddbSync "github.com/armPelionEdge/devicedb/sync"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

func url(u string, server *Server) string {
    return "http://localhost:" + fmt.Sprintf("%d", server.Port()) + u
}

func buffer(j string) *bytes.Buffer {
    return bytes.NewBuffer([]byte(j))
}

var _ = Describe("Server", func() {
    var client *http.Client
    var server *Server
    var hub *Hub
    var syncController *SyncController
    stop := make(chan int)
    
    BeforeEach(func() {
        _, clientTLS, err := loadCerts("WWRL000000")

        Expect(err).Should(Not(HaveOccurred()))

        syncController = NewSyncController(2, nil, ddbSync.NewPeriodicSyncScheduler(SYNC_PERIOD_MS), 1000)
        hub = NewHub("", syncController, clientTLS)

        client = &http.Client{ Transport: &http.Transport{ DisableKeepAlives: true } }
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
    
    Describe("ServerConfig", func() {
        It("should load the config from a file", func() {
            var sc ServerConfig
            
            err := sc.LoadFromFile("../test_configs/test_config_1.yaml")
           
            Expect(err).Should(BeNil())
        })
    })
    
    Describe("POST /{bucket}/values", func() {
        Context("The values being queried are empty", func() {
            It("should return nil for every key", func() {
                resp, err := client.Post(url("/default/values", server), "application/json", buffer(`[ "key1", "key2", "key3" ]`))
                
                Expect(err).Should(BeNil())
            
                siblingSets := make([]*SiblingSet, 0)
                decoder := json.NewDecoder(resp.Body)
                err = decoder.Decode(&siblingSets)
                
                Expect(err).Should(BeNil())
                Expect(siblingSets).Should(Equal([]*SiblingSet{ nil, nil, nil }))
                Expect(resp.StatusCode).Should(Equal(http.StatusOK))
            })
        })
        
        It("Should return 404 with EInvalidBucket in the body if the bucket specified is invalid", func() {
            resp, err := client.Post(url("/invalidbucket/values", server), "application/json", buffer(`[ "key1", "key2", "key3" ]`))
                
            Expect(err).Should(BeNil())
            defer resp.Body.Close()
            
            var dberr DBerror
            decoder := json.NewDecoder(resp.Body)
            err = decoder.Decode(&dberr)
            
            Expect(err).Should(BeNil())
            Expect(dberr).Should(Equal(EInvalidBucket))
            Expect(resp.StatusCode).Should(Equal(http.StatusNotFound))
        })
        
        It("Should return 400 with EInvalidKey in the body if a null key value was specified", func() {
            resp, err := client.Post(url("/default/values", server), "application/json", buffer(`[ null, "key2", "key3" ]`))
                
            Expect(err).Should(BeNil())
            defer resp.Body.Close()
            
            var dberr DBerror
            decoder := json.NewDecoder(resp.Body)
            err = decoder.Decode(&dberr)
            
            Expect(err).Should(BeNil())
            Expect(dberr).Should(Equal(EInvalidKey))
            Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))
        })
        
        It("Should return 400 with EInvalidKey in the body if an empty key was specified", func() {
            resp, err := client.Post(url("/default/values", server), "application/json", buffer(`[ "", "key2", "key3" ]`))
                
            Expect(err).Should(BeNil())
            defer resp.Body.Close()
            
            var dberr DBerror
            decoder := json.NewDecoder(resp.Body)
            err = decoder.Decode(&dberr)
            
            Expect(err).Should(BeNil())
            Expect(dberr).Should(Equal(EInvalidKey))
            Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))
        })
        
        It("Should return 400 with EInvalidKey in the body if the request body is not an array", func() {
            resp, err := client.Post(url("/default/values", server), "application/json", buffer(`null`))
                
            Expect(err).Should(BeNil())
            defer resp.Body.Close()
            
            var dberr DBerror
            decoder := json.NewDecoder(resp.Body)
            err = decoder.Decode(&dberr)
            
            Expect(err).Should(BeNil())
            Expect(dberr).Should(Equal(EInvalidKey))
            Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))
            
            resp, err = http.Post(url("/default/values", server), "application/json", buffer(`{ "0": "key1" }`))
                
            Expect(err).Should(BeNil())
           
            fmt.Printf("BODY: %v\n", resp.Body)
            decoder = json.NewDecoder(resp.Body)
            err = decoder.Decode(&dberr)
            
            Expect(err).Should(BeNil())
            Expect(dberr).Should(Equal(EInvalidKey))
            Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))
        })
    })
    
    Describe("POST /{bucket}/matches", func() {
        Context("The values being queried are empty", func() {
            It("should return nil for every key", func() {
                updateBatch := NewUpdateBatch()
                updateBatch.Put([]byte("key1"), []byte("value1"), NewDVV(NewDot("", 0), map[string]uint64{ }))
                updateBatch.Put([]byte("key2"), []byte("value2"), NewDVV(NewDot("", 0), map[string]uint64{ }))
                updateBatch.Put([]byte("key3"), []byte("value3"), NewDVV(NewDot("", 0), map[string]uint64{ }))
                var transportUpdateBatch TransportUpdateBatch = make([]TransportUpdateOp, len(updateBatch.Batch().Ops()))
                transportUpdateBatch.FromUpdateBatch(updateBatch)
                
                jsonBytes, _ := json.Marshal(transportUpdateBatch)
            
                resp, err := client.Post(url("/default/batch", server), "application/json", bytes.NewBuffer(jsonBytes))
                
                Expect(err).Should(BeNil())
                resp.Body.Close()
                
                resp, err = client.Post(url("/default/matches", server), "application/json", buffer(`[ "key1", "key2", "key3" ]`))
                defer resp.Body.Close()
                
                values := [][]string{
                    []string{ "key1", "value1", },
                    []string{ "key2", "value2", },
                    []string{ "key3", "value3", },
                }
                
                scanner := bufio.NewScanner(resp.Body)
                
                itemCount := 0
                
                for scanner.Scan() {
                    itemCount += 1
                    l := values[0]
                    values = values[1:]
                    
                    Expect(scanner.Text()).Should(Equal(l[0]))
                    Expect(scanner.Scan()).Should(BeTrue())
                    Expect(scanner.Text()).Should(Equal(l[0]))
                    Expect(scanner.Scan()).Should(BeTrue())
                    
                    var siblingSet TransportSiblingSet
                    
                    fmt.Println(scanner.Text())
                    decoder := json.NewDecoder(bytes.NewBuffer(scanner.Bytes()))
                    err = decoder.Decode(&siblingSet)
                    Expect(err).Should(BeNil())
                    Expect(siblingSet.Siblings[0]).Should(Equal(l[1]))
                }
                
                Expect(itemCount).Should(Equal(3))
                Expect(scanner.Err()).Should(BeNil())
            })
        })
        
        It("Should return 404 with EInvalidBucket in the body if the bucket specified is invalid", func() {
            resp, err := client.Post(url("/invalidbucket/matches", server), "application/json", buffer(`[ "key1", "key2", "key3" ]`))
                
            Expect(err).Should(BeNil())
            defer resp.Body.Close()
            
            var dberr DBerror
            decoder := json.NewDecoder(resp.Body)
            err = decoder.Decode(&dberr)
            
            Expect(err).Should(BeNil())
            Expect(dberr).Should(Equal(EInvalidBucket))
            Expect(resp.StatusCode).Should(Equal(http.StatusNotFound))
        })
        
        It("Should return 400 with EInvalidKey in the body if a null key value was specified", func() {
            resp, err := client.Post(url("/default/matches", server), "application/json", buffer(`[ null, "key2", "key3" ]`))
                
            Expect(err).Should(BeNil())
            defer resp.Body.Close()
            
            var dberr DBerror
            decoder := json.NewDecoder(resp.Body)
            err = decoder.Decode(&dberr)
            
            Expect(err).Should(BeNil())
            Expect(dberr).Should(Equal(EInvalidKey))
            Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))
        })
        
        It("Should return 400 with EInvalidKey in the body if an empty key was specified", func() {
            resp, err := client.Post(url("/default/matches", server), "application/json", buffer(`[ "", "key2", "key3" ]`))
                
            Expect(err).Should(BeNil())
            defer resp.Body.Close()
            
            var dberr DBerror
            decoder := json.NewDecoder(resp.Body)
            err = decoder.Decode(&dberr)
            
            Expect(err).Should(BeNil())
            Expect(dberr).Should(Equal(EInvalidKey))
            Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))
        })
        
        It("Should return 400 with EInvalidKey in the body if the request body is not an array", func() {
            resp, err := client.Post(url("/default/matches", server), "application/json", buffer(`null`))
                
            Expect(err).Should(BeNil())
            defer resp.Body.Close()
            
            var dberr DBerror
            decoder := json.NewDecoder(resp.Body)
            err = decoder.Decode(&dberr)
            
            Expect(err).Should(BeNil())
            Expect(dberr).Should(Equal(EInvalidKey))
            Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))
            
            resp, err = http.Post(url("/default/values", server), "application/json", buffer(`{ "0": "key1" }`))
                
            Expect(err).Should(BeNil())
            
            decoder = json.NewDecoder(resp.Body)
            err = decoder.Decode(&dberr)
            
            Expect(err).Should(BeNil())
            Expect(dberr).Should(Equal(EInvalidKey))
            Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))
        })
    })
    
    Describe("POST /{bucket}/batch", func() {
        It("should put the values specified", func() {
            updateBatch := NewUpdateBatch()
            updateBatch.Put([]byte("key1"), []byte("value1"), NewDVV(NewDot("", 0), map[string]uint64{ }))
            updateBatch.Put([]byte("key2"), []byte("value2"), NewDVV(NewDot("", 0), map[string]uint64{ }))
            updateBatch.Put([]byte("key3"), []byte("value3"), NewDVV(NewDot("", 0), map[string]uint64{ }))
            var transportUpdateBatch TransportUpdateBatch = make([]TransportUpdateOp, len(updateBatch.Batch().Ops()))
            transportUpdateBatch.FromUpdateBatch(updateBatch)
            
            jsonBytes, _ := json.Marshal(transportUpdateBatch)
        
            resp, err := client.Post(url("/default/batch", server), "application/json", bytes.NewBuffer(jsonBytes))
            
            Expect(err).Should(BeNil())
            defer resp.Body.Close()
            Expect(resp.StatusCode).Should(Equal(http.StatusOK))
            
            resp, err = client.Post(url("/default/values", server), "application/json", buffer(`[ "key1", "key2", "key3" ]`))
            
            Expect(err).Should(BeNil())
        
            siblingSets := make([]*TransportSiblingSet, 0)
            decoder := json.NewDecoder(resp.Body)
            err = decoder.Decode(&siblingSets)
            
            Expect(err).Should(BeNil())
            Expect(siblingSets[0].Siblings[0]).Should(Equal("value1"))
            Expect(siblingSets[1].Siblings[0]).Should(Equal("value2"))
            Expect(siblingSets[2].Siblings[0]).Should(Equal("value3"))
            Expect(resp.StatusCode).Should(Equal(http.StatusOK))
        })

        Describe("Regression test for bug with BroadcastUpdate where it would block if Connect had been called for a particular peer but it had not yet connected", func() {
            It("should not hang on second batch call", func() {
                updateBatch := NewUpdateBatch()
                updateBatch.Put([]byte("key1"), []byte("value1"), NewDVV(NewDot("", 0), map[string]uint64{ }))
                updateBatch.Put([]byte("key2"), []byte("value2"), NewDVV(NewDot("", 0), map[string]uint64{ }))
                updateBatch.Put([]byte("key3"), []byte("value3"), NewDVV(NewDot("", 0), map[string]uint64{ }))
                var transportUpdateBatch TransportUpdateBatch = make([]TransportUpdateOp, len(updateBatch.Batch().Ops()))
                transportUpdateBatch.FromUpdateBatch(updateBatch)
              
                // This connects to nothing so it will fail and keep re-trying
                Expect(hub.ConnectCloud("", "localhost", "", "http://localhost:54545", "", "http://localhost:32232", true)).Should(Not(HaveOccurred()))

                <-time.After(time.Second)

                jsonBytes, _ := json.Marshal(transportUpdateBatch)
           
                responseReceived := make(chan int)

                go func() {
                    client.Post(url("/default/batch", server), "application/json", bytes.NewBuffer(jsonBytes))

                    responseReceived <- 1
                }()

                select {
                case <-responseReceived:
                case <-time.After(time.Second):
                    Fail("The request should not hang")
                }

                <-time.After(time.Second)
            })
        })
    })
})
