package benchmarks

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "errors"
    "fmt"
    "io/ioutil"
    "math/rand"
    "os"
    "time"

    "github.com/armPelionEdge/devicedb/bucket"
    "github.com/armPelionEdge/devicedb/client"
    "github.com/armPelionEdge/devicedb/data"
    "github.com/armPelionEdge/devicedb/server"
    "github.com/armPelionEdge/devicedb/sync"
    "github.com/armPelionEdge/devicedb/util"
)

// Simulates many relays distributed across multiple sites submitting updates at a set rate
func BenchmarkManyRelays(cloudAddresses []string, internalAddresses []string, nSites int, nRelaysPerSite int, updatesPerSecond int, syncPeriodMS int) {
    //var relays []server.Server = make([]server.Server, 0, nSites * nRelaysPerSite)
    var nextPort = 10000

    apiClient := client.New(client.APIClientConfig{
        Servers: internalAddresses,
    })

    // Initialize cluster state
    for i := 0; i < nSites; i++ {
        siteID := fmt.Sprintf("site-%d", i)

        fmt.Fprintf(os.Stdout, "Create site %s\n", siteID)

        if err := apiClient.AddSite(context.TODO(), siteID); err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to create site: %v. Aborting...\n", err)

            os.Exit(1)
        }

        for j := 0; j < nRelaysPerSite; j++ {
            relayID := fmt.Sprintf("relay-%s-%d", siteID, j)

            fmt.Fprintf(os.Stdout, "Create relay %s\n", relayID)

            if err := apiClient.AddRelay(context.TODO(), relayID); err != nil {
                fmt.Fprintf(os.Stderr, "Error: Unable to add relay: %v. Aborting...\n", err)

                os.Exit(1)
            }
            
            fmt.Fprintf(os.Stdout, "Moving relay %s to site %s\n", relayID, siteID)

            if err := apiClient.MoveRelay(context.TODO(), relayID, siteID); err != nil {
                fmt.Fprintf(os.Stderr, "Error: Unable to move relay: %v. Aborting...\n", err)

                os.Exit(1)
            }

            relay, syncController, hub := makeRelay(relayID, syncPeriodMS, updatesPerSecond, nextPort)
            nextPort++

            go func() {
                err := relay.Start()

                fmt.Fprintf(os.Stderr, "Error: Relay start error: %v. Aborting...\n", err)

                os.Exit(1)
            }()

            go func(relay *server.Server, hub *server.Hub) {
                if updatesPerSecond == 0 {
                    return
                }
                
                for {
                    <-time.After(time.Second / time.Duration(updatesPerSecond))
                    updateBatch := bucket.NewUpdateBatch()
                    key := util.RandomString()
                    value := util.RandomString()
                    fmt.Fprintf(os.Stderr, "Writing %s to key %s\n", value, key)
                    updateBatch.Put([]byte(key), []byte(value), data.NewDVV(data.NewDot("", 0), map[string]uint64{ }))
                    updatedSiblingSets, err := relay.Buckets().Get("default").Batch(updateBatch)
                    hub.BroadcastUpdate("", "default", updatedSiblingSets, 64)

                    if err != nil {
                        fmt.Fprintf(os.Stderr, "Error: Unable to write batch: %v. Aborting...\n", err)

                        os.Exit(1)
                    }
                }
            }(relay, hub)

            syncController.Start()

            randomCloudAddress := cloudAddresses[int(rand.Uint32() % uint32(len(cloudAddresses)))]

            if err := hub.ConnectCloud("cloud", randomCloudAddress, "", "", "", "", true); err != nil {
                fmt.Fprintf(os.Stderr, "Error: Unable to initiate cloud connection process: %v. Aborting...\n", err)

                os.Exit(1)
            }
        }
    }

    <-time.After(time.Minute)
}

func makeRelay(relayID string, syncPeriodMS int, updatesPerSecond int, port int) (*server.Server, *server.SyncController, *server.Hub) {
    serverTLS, clientTLS, err := loadCerts("WWRL000000")

    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: Unable to load certificates: %v. Aborting...\n", err.Error())

        os.Exit(1)
    }

    syncController := server.NewSyncController(2, nil, sync.NewPeriodicSyncScheduler(time.Millisecond * time.Duration(syncPeriodMS)), 1000)
    hub := server.NewHub(relayID, syncController, clientTLS)
    relay, _ := server.NewServer(server.ServerConfig{
        DBFile: "/tmp/testdb-" + util.RandomString(),
        Port: port,
        ServerTLS: serverTLS,
        Hub: hub,
        MerkleDepth: 5,
    })

    return relay, syncController, hub
}

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