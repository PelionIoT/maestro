package server

import (
    "fmt"
    "io"
    "io/ioutil"
    "net"
    "errors"
    "net/http"
    "crypto/tls"
    "crypto/x509"
    "encoding/json"
    "encoding/hex"
    "time"
    "strconv"
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "net/http/pprof"

    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/bucket/builtin"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/merkle"
    . "github.com/armPelionEdge/devicedb/shared"
    . "github.com/armPelionEdge/devicedb/storage"
    . "github.com/armPelionEdge/devicedb/historian"
    . "github.com/armPelionEdge/devicedb/alerts"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/logging"
    ddbSync "github.com/armPelionEdge/devicedb/sync"
    . "github.com/armPelionEdge/devicedb/site"
    . "github.com/armPelionEdge/devicedb/transport"
)

const (
    defaultNodePrefix = iota
    cloudNodePrefix = iota
    lwwNodePrefix = iota
    localNodePrefix = iota
    historianPrefix = iota
    alertsMapPrefix = iota
)

type peerAddress struct {
    ID string `json:"id"`
    Host string `json:"host"`
    Port int `json:"port"`
}

type cloudAddress struct {
    ID string `json:"id"`
    Host string `json:"host"`
    Port int `json:"port"`
    NoValidate bool
    URI string `json:"uri"`
}

type AlertEventData struct {
    Metadata interface{} `json:"metadata"`
    Status bool `json:"status"`
}

type ServerConfig struct {
    DBFile string
    Port int
    MerkleDepth uint8
    NodeID string
    Hub *Hub
    ServerTLS *tls.Config
    PeerAddresses map[string]peerAddress
    SyncPushBroadcastLimit uint64
    GCInterval uint64
    GCPurgeAge uint64
    Cloud *cloudAddress
    History *cloudAddress
    Alerts *cloudAddress
    HistoryPurgeOnForward bool
    HistoryEventLimit uint64
    HistoryEventFloor uint64
    HistoryPurgeBatchSize int
    HistoryForwardBatchSize uint64
    HistoryForwardInterval uint64
    HistoryForwardThreshold uint64
    AlertsForwardInterval uint64
    SyncExplorationPathLimit uint32
}

func (sc *ServerConfig) LoadFromFile(file string) error {
    var ysc YAMLServerConfig
    
    err := ysc.LoadFromFile(file)
    
    if err != nil {
        return err
    }
    
    sc.GCInterval = ysc.GCInterval
    sc.GCPurgeAge = ysc.GCPurgeAge
    sc.DBFile = ysc.DBFile
    sc.Port = ysc.Port
    sc.MerkleDepth = ysc.MerkleDepth
    sc.SyncPushBroadcastLimit = ysc.SyncPushBroadcastLimit
    sc.SyncExplorationPathLimit = ysc.SyncExplorationPathLimit
    sc.PeerAddresses = make(map[string]peerAddress)
    
    rootCAs := x509.NewCertPool()
    
    if !rootCAs.AppendCertsFromPEM([]byte(ysc.TLS.RootCA)) {
        return errors.New("Could not append root CA to chain")
    }
    
    clientCertificate, _ := tls.X509KeyPair([]byte(ysc.TLS.ClientCertificate), []byte(ysc.TLS.ClientKey))
    serverCertificate, _ := tls.X509KeyPair([]byte(ysc.TLS.ServerCertificate), []byte(ysc.TLS.ServerKey))
    clientTLSConfig := &tls.Config{
        Certificates: []tls.Certificate{ clientCertificate },
        RootCAs: rootCAs,
        GetClientCertificate: func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
            return &clientCertificate, nil
        },
    }
    serverTLSConfig := &tls.Config{
        Certificates: []tls.Certificate{ serverCertificate },
        ClientCAs: rootCAs,
    }
    
    sc.ServerTLS = serverTLSConfig
    
    for _, yamlPeer := range ysc.Peers {
        if _, ok := sc.PeerAddresses[yamlPeer.ID]; ok {
            return errors.New(fmt.Sprintf("Duplicate entry for peer %s in config file", yamlPeer.ID))
        }
        
        sc.PeerAddresses[yamlPeer.ID] = peerAddress{
            ID: yamlPeer.ID,
            Host: yamlPeer.Host,
            Port: yamlPeer.Port,
        }
    }
    
    if ysc.Cloud != nil {
        sc.Cloud = &cloudAddress{
            ID: ysc.Cloud.ID,
            NoValidate: ysc.Cloud.NoValidate,
            URI: ysc.Cloud.URI,
        }

        sc.History = &cloudAddress{
            ID: ysc.Cloud.HistoryID,
            NoValidate: ysc.Cloud.NoValidate,
            URI: ysc.Cloud.HistoryURI,
        }

        sc.Alerts = &cloudAddress{
            ID: ysc.Cloud.AlertsID,
            NoValidate: ysc.Cloud.NoValidate,
            URI: ysc.Cloud.AlertsURI,
        }
    }
    
    sc.HistoryPurgeOnForward = ysc.History.PurgeOnForward
    sc.HistoryEventLimit = ysc.History.EventLimit
    sc.HistoryEventFloor = ysc.History.EventFloor
    sc.HistoryPurgeBatchSize = ysc.History.PurgeBatchSize
    sc.HistoryForwardBatchSize = ysc.History.ForwardBatchSize
    sc.HistoryForwardInterval = ysc.History.ForwardInterval
    sc.HistoryForwardThreshold = ysc.History.ForwardThreshold
    sc.AlertsForwardInterval = ysc.Alerts.ForwardInterval
    
    clientCertX509, _ := x509.ParseCertificate(clientCertificate.Certificate[0])
    serverCertX509, _ := x509.ParseCertificate(serverCertificate.Certificate[0])
    clientCN := clientCertX509.Subject.CommonName
    serverCN := serverCertX509.Subject.CommonName
    
    if len(clientCN) == 0 {
        return errors.New("The common name in the certificate is empty. The node ID must not be empty")
    }
     
    if clientCN != serverCN {
        return errors.New(fmt.Sprintf("Server and client certificates have differing common names(%s and %s). This is the string used to uniquely identify the node.", serverCN, clientCN))
    }
    
    sc.NodeID = clientCN
    sc.Hub = NewHub(sc.NodeID, NewSyncController(uint(ysc.MaxSyncSessions), nil, ddbSync.NewPeriodicSyncScheduler(time.Millisecond * time.Duration(ysc.SyncSessionPeriod)), sc.SyncExplorationPathLimit), clientTLSConfig)
    
    return nil
}

type Server struct {
    bucketList *BucketList
    httpServer *http.Server
    listener net.Listener
    storageDriver StorageDriver
    port int
    upgrader websocket.Upgrader
    hub *Hub
    serverTLS *tls.Config
    id string
    syncPushBroadcastLimit uint64
    garbageCollector *GarbageCollector
    historian *Historian
    alertsMap *AlertMap
    merkleDepth uint8
}

func NewServer(serverConfig ServerConfig) (*Server, error) {
    if serverConfig.MerkleDepth < MerkleMinDepth || serverConfig.MerkleDepth > MerkleMaxDepth {
        serverConfig.MerkleDepth = MerkleDefaultDepth
    }
    
    if len(serverConfig.NodeID) == 0 {
        serverConfig.NodeID = "Node"
    }
    
    upgrader := websocket.Upgrader{
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
    }
    
    storageDriver := NewLevelDBStorageDriver(serverConfig.DBFile, nil)
    nodeID := serverConfig.NodeID
    server := &Server{ NewBucketList(), nil, nil, storageDriver, serverConfig.Port, upgrader, serverConfig.Hub, serverConfig.ServerTLS, nodeID, serverConfig.SyncPushBroadcastLimit, nil, nil, nil, serverConfig.MerkleDepth }
    err := server.storageDriver.Open()
    
    if err != nil {
        if err != ECorrupted {
            Log.Errorf("Error creating server: %v", err.Error())
            
            return nil, err
        }

        Log.Error("Database is corrupted. Attempting automatic recovery now...")

        recoverError := server.recover()

        if recoverError != nil {
            Log.Criticalf("Unable to recover corrupted database. Reason: %v", recoverError.Error())
            Log.Critical("Database daemon will now exit")

            return nil, EStorage
        }

        Log.Info("Database recovery successful!")
    }
    
    defaultBucket, _ := NewDefaultBucket(nodeID, NewPrefixedStorageDriver([]byte{ defaultNodePrefix }, storageDriver), serverConfig.MerkleDepth)
    cloudBucket, _ := NewCloudBucket(nodeID, NewPrefixedStorageDriver([]byte{ cloudNodePrefix }, storageDriver), serverConfig.MerkleDepth, RelayMode)
    lwwBucket, _ := NewLWWBucket(nodeID, NewPrefixedStorageDriver([]byte{ lwwNodePrefix }, storageDriver), serverConfig.MerkleDepth)
    localBucket, _ := NewLocalBucket(nodeID, NewPrefixedStorageDriver([]byte{ localNodePrefix }, storageDriver), MerkleMinDepth)
    
    server.historian = NewHistorian(NewPrefixedStorageDriver([]byte{ historianPrefix }, storageDriver), serverConfig.HistoryEventLimit, serverConfig.HistoryEventFloor, serverConfig.HistoryPurgeBatchSize)
    server.alertsMap = NewAlertMap(NewAlertStore(NewPrefixedStorageDriver([]byte{ alertsMapPrefix }, storageDriver)))
    
    server.bucketList.AddBucket(defaultBucket)
    server.bucketList.AddBucket(lwwBucket)
    server.bucketList.AddBucket(cloudBucket)
    server.bucketList.AddBucket(localBucket)
    
    server.garbageCollector = NewGarbageCollector(server.bucketList, serverConfig.GCInterval, serverConfig.GCPurgeAge)
    
    if server.hub != nil && server.hub.syncController != nil {
        server.hub.historian = server.historian
        server.hub.alertsMap = server.alertsMap
        server.hub.purgeOnForward = serverConfig.HistoryPurgeOnForward
        server.hub.forwardBatchSize = serverConfig.HistoryForwardBatchSize
        server.hub.forwardThreshold = serverConfig.HistoryForwardThreshold
        server.hub.forwardInterval = serverConfig.HistoryForwardInterval
        server.hub.alertsForwardInterval = serverConfig.AlertsForwardInterval
    }
    
    if server.hub != nil && server.hub.syncController != nil {
        site := NewRelaySiteReplica(nodeID, server.bucketList)
        sitePool := &RelayNodeSitePool{ Site: site }
        bucketProxyFactory := &ddbSync.RelayBucketProxyFactory{ SitePool: sitePool }
        server.hub.syncController.bucketProxyFactory = bucketProxyFactory
    }
    
    if server.hub != nil && serverConfig.PeerAddresses != nil {
        for _, pa := range serverConfig.PeerAddresses {
            server.hub.Connect(pa.ID, pa.Host, pa.Port)
        }
    }
    
    if server.hub != nil && serverConfig.Cloud != nil {
        server.hub.ConnectCloud(serverConfig.Cloud.ID, serverConfig.Cloud.URI, serverConfig.History.ID, serverConfig.History.URI, serverConfig.Alerts.ID, serverConfig.Alerts.URI, serverConfig.Cloud.NoValidate)
    }
    
    return server, nil
}

func (server *Server) Port() int {
    return server.port
}

func (server *Server) Buckets() *BucketList {
    return server.bucketList
}

func (server *Server) History() *Historian {
    return server.historian
}

func (server *Server) AlertsMap() *AlertMap {
    return server.alertsMap
}

func (server *Server) StartGC() {
    server.garbageCollector.Start()
}

func (server *Server) StopGC() {
    server.garbageCollector.Stop()
}

func (server *Server) recover() error {
    recoverError := server.storageDriver.Recover()

    if recoverError != nil {
        Log.Criticalf("Unable to recover corrupted database. Reason: %v", recoverError.Error())

        return EStorage
    }

    Log.Infof("Rebuilding merkle trees...")

    for i := 0; i < localNodePrefix; i += 1 {
        tempBucket, _ := NewDefaultBucket("temp", NewPrefixedStorageDriver([]byte{ byte(i) }, server.storageDriver), server.merkleDepth)
        rebuildError := tempBucket.RebuildMerkleLeafs()

        if rebuildError != nil {
            Log.Errorf("Unable to rebuild merkle tree for node %d. Reason: %v", i, rebuildError.Error())

            return rebuildError
        }

        recordError := tempBucket.RecordMetadata()

        if recordError != nil {
            Log.Errorf("Unable to rebuild node metadata for node %d. Reason: %v", i, recordError.Error())

            return recordError
        }
    }

    return nil
}

func (server *Server) Start() error {
    r := mux.NewRouter()
    
    r.HandleFunc("/{bucket}/merkleRoot", func(w http.ResponseWriter, r *http.Request) {
        bucket := mux.Vars(r)["bucket"]
        
        if !server.bucketList.HasBucket(bucket) {
            Log.Warningf("POST /{bucket}/merkleRoot: Invalid bucket")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(EInvalidBucket.JSON()) + "\n")
            
            return
        }
        
        hashBytes := server.bucketList.Get(bucket).MerkleTree().RootHash().Bytes()
        
        w.Header().Set("Content-Type", "text/plain; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, hex.EncodeToString(hashBytes[:]))
    }).Methods("GET")

    r.HandleFunc("/{bucket}/watch", func(w http.ResponseWriter, r *http.Request) {
        query := r.URL.Query()
        
        var keys [][]byte = make([][]byte, 0)
        var prefixes [][]byte = make([][]byte, 0)
        var lastSerial uint64
    
        if qKeys, ok := query["key"]; ok {
            keys = make([][]byte, len(qKeys))

            for i, key := range qKeys {
                keys[i] = []byte(key)
            }
        }

        if qPrefixes, ok := query["prefix"]; ok {
            prefixes = make([][]byte, len(qPrefixes))

            for i, prefix := range qPrefixes {
                prefixes[i] = []byte(prefix)
            }
        }

        if qLastSerials, ok := query["lastSerial"]; ok {
            ls, err := strconv.ParseUint(qLastSerials[0], 10, 64)

            if err != nil {
                Log.Warningf("GET /{bucket}/watch: Invalid lastSerial specified")

                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusBadRequest)
                io.WriteString(w, string(EInvalidKey.JSON()) + "\n")
                
                return
            }

            lastSerial = ls
        }

        bucket := mux.Vars(r)["bucket"]

        if !server.bucketList.HasBucket(bucket) {
            Log.Warningf("GET /{bucket}/watch: Invalid bucket")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(EInvalidBucket.JSON()) + "\n")
            
            return
        }

        var ch chan Row = make(chan Row)
        go server.bucketList.Get(bucket).Watch(r.Context(), keys, prefixes, lastSerial, ch)

        flusher, _ := w.(http.Flusher)

        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("Connection", "keep-alive")

        // It is important not to break out of this loop early
        // If the channel is not read until it is closed it will
        // block future upates
        for update := range ch {
            // This is a marker intended to indicate
            // the end of the initial query to find
            // any missed updates based on the lastSerial
            // field
            if update.Key == "" {
                fmt.Fprintf(w, "data: \n\n")
                flusher.Flush()
                continue
            }

            var transportUpdate TransportRow
            
            if err := transportUpdate.FromRow(&update); err != nil {
                Log.Errorf("Encountered an error while converting an update to its transport format: %v", err)
                continue
            }

            encodedUpdate, err := json.Marshal(transportUpdate)

            if err != nil {
                Log.Errorf("Encountered an error while encoding an update to JSON: %v", err)
                continue
            }

            _, err = fmt.Fprintf(w, "data: %s\n\n", string(encodedUpdate))

            flusher.Flush()
            
            if err != nil {
                Log.Errorf("Encountered an error while writing an update to the event stream for a watcher: %v", err)
                continue
            }
        }
    }).Methods("GET")
    
    r.HandleFunc("/{bucket}/values", func(w http.ResponseWriter, r *http.Request) {
        startTime := time.Now()
        bucket := mux.Vars(r)["bucket"]
        
        if !server.bucketList.HasBucket(bucket) {
            Log.Warningf("POST /{bucket}/values: Invalid bucket")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(EInvalidBucket.JSON()) + "\n")
            
            return
        }
        
        var keysArray *[]string
        decoder := json.NewDecoder(r.Body)
        err := decoder.Decode(&keysArray)
        
        if err != nil || keysArray == nil {
            Log.Warningf("POST /{bucket}/values: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EInvalidKey.JSON()) + "\n")
            
            return
        }
        
        keys := *keysArray
        
        if len(keys) == 0 {
            siblingSetsJSON, _ := json.Marshal([]*TransportSiblingSet{ })
        
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusOK)
            io.WriteString(w, string(siblingSetsJSON))
            
            return
        }
        
        byteKeys := make([][]byte, 0, len(keys))
        
        for _, k := range keys {
            if len(k) == 0 {
                Log.Warningf("POST /{bucket}/values: Empty key")
            
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusBadRequest)
                io.WriteString(w, string(EInvalidKey.JSON()) + "\n")
                
                return
            }
            
            byteKeys = append(byteKeys, []byte(k))
        }

        siblingSets, err := server.bucketList.Get(bucket).Get(byteKeys)
        
        if err != nil {
            Log.Warningf("POST /{bucket}/values: Internal server error")
        
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, string(err.(DBerror).JSON()) + "\n")
            
            return
        }
        
        transportSiblingSets := make([]*TransportSiblingSet, 0, len(siblingSets))
        
        for _, siblingSet := range siblingSets {
            if siblingSet == nil {
                transportSiblingSets = append(transportSiblingSets, nil)
                
                continue
            }
            
            var transportSiblingSet TransportSiblingSet
            err := transportSiblingSet.FromSiblingSet(siblingSet)
            
            if err != nil {
                Log.Warningf("POST /{bucket}/values: Internal server error")
        
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusInternalServerError)
                io.WriteString(w, string(err.(DBerror).JSON()) + "\n")
                
                return
            }
            
            transportSiblingSets = append(transportSiblingSets, &transportSiblingSet)
        }
        
        siblingSetsJSON, _ := json.Marshal(transportSiblingSets)
        
        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, string(siblingSetsJSON))

        Log.Debugf("Get from bucket %s: %v took %s", bucket, keys, time.Since(startTime))
    }).Methods("POST")
    
    r.HandleFunc("/{bucket}/matches", func(w http.ResponseWriter, r *http.Request) {
        startTime := time.Now()
        bucket := mux.Vars(r)["bucket"]
        
        if !server.bucketList.HasBucket(bucket) {
            Log.Warningf("POST /{bucket}/matches: Invalid bucket")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(EInvalidBucket.JSON()) + "\n")
            
            return
        }    
    
        var keysArray *[]string
        decoder := json.NewDecoder(r.Body)
        err := decoder.Decode(&keysArray)
        
        if err != nil || keysArray == nil {
            Log.Warningf("POST /{bucket}/matches: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EInvalidKey.JSON()) + "\n")
            
            return
        }
        
        keys := *keysArray
        
        if len(keys) == 0 {
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusOK)
            io.WriteString(w, "\n")
            
            return
        }
        
        byteKeys := make([][]byte, 0, len(keys))
        
        for _, k := range keys {
            if len(k) == 0 {
                Log.Warningf("POST /{bucket}/matches: Empty key")
            
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusBadRequest)
                io.WriteString(w, string(EInvalidKey.JSON()) + "\n")
                
                return
            }
            
            byteKeys = append(byteKeys, []byte(k))
        }
        
        ssIterator, err := server.bucketList.Get(bucket).GetMatches(byteKeys)
        
        if err != nil {
            Log.Warningf("POST /{bucket}/matches: Internal server error")
        
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, string(err.(DBerror).JSON()) + "\n")
            
            return
        }
        
        defer ssIterator.Release()
    
        flusher, _ := w.(http.Flusher)
        
        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.WriteHeader(http.StatusOK)
        
        for ssIterator.Next() {
            key := ssIterator.Key()
            prefix := ssIterator.Prefix()
            nextSiblingSet := ssIterator.Value()
            
            if nextSiblingSet.IsTombstoneSet() {
                continue
            }
            
            var nextTransportSiblingSet TransportSiblingSet
            
            err := nextTransportSiblingSet.FromSiblingSet(nextSiblingSet)
            
            if err != nil {
                Log.Warningf("POST /{bucket}/matches: Internal server error")
        
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusInternalServerError)
                io.WriteString(w, string(err.(DBerror).JSON()) + "\n")
                
                return
            }
            
            siblingSetsJSON, _ := json.Marshal(&nextTransportSiblingSet)
            
            _, err = fmt.Fprintf(w, "%s\n%s\n%s\n", string(prefix), string(key), string(siblingSetsJSON))
            flusher.Flush()
            
            if err != nil {
                return
            }
        }

        Log.Debugf("Get matches from bucket %s: %v took %s", bucket, keys, time.Since(startTime))
    }).Methods("POST")
    
    r.HandleFunc("/events/{sourceID}/{type}", func(w http.ResponseWriter, r *http.Request) {
        query := r.URL.Query()
        
        var category string
        var groups []string
    
        if categories, ok := query["category"]; ok {
            category = categories[0]
        }

        if _, ok := query["group"]; ok {
            groups = query["group"]
        } else {
            groups = []string{ }
        }

        eventType := mux.Vars(r)["type"]
        sourceID := mux.Vars(r)["sourceID"]
        body, err := ioutil.ReadAll(r.Body)
        
        if err != nil {
            Log.Warningf("PUT /events/{type}/{sourceID}: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EReadBody.JSON()) + "\n")
            
            return
        }
        
        if len(eventType) == 0 {
            Log.Warningf("PUT /events/{type}/{sourceID}: Empty event type")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EEmpty.JSON()) + "\n")
            
            return
        }
        
        if len(sourceID) == 0 {
            Log.Warningf("PUT /events/{type}/{sourceID}: Source id empty")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EEmpty.JSON()) + "\n")
            
            return
        }
        
        timestamp := NanoToMilli(uint64(time.Now().UnixNano()))

        if category == "alerts" {
            var alertData AlertEventData

            err = json.Unmarshal(body, &alertData)
            
            if err != nil {
                Log.Warningf("PUT /events/{type}/{sourceID}: Unable to parse alert body %s", body)
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusBadRequest)
                io.WriteString(w, string(EAlertBody.JSON()) + "\n")
                
                return
            }

            err = server.alertsMap.UpdateAlert(Alert{
                Key: sourceID,
                Level: eventType,
                Timestamp: timestamp,
                Metadata: alertData.Metadata,
                Status: alertData.Status,
            })

            server.hub.ForwardAlerts()
        } else {
            err = server.historian.LogEvent(&Event{
                Timestamp: timestamp,
                SourceID: sourceID,
                Type: eventType,
                Data: string(body),
                Groups: groups,
            })

            server.hub.ForwardEvents()
        }
        
        if err != nil {
            Log.Warningf("POST /events/{type}/{sourceID}: Internal server error")
        
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, string(err.(DBerror).JSON()) + "\n")
            
            return
        }
        
        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, "\n")
    }).Methods("PUT")
    
    r.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
        query := r.URL.Query()
        
        var historyQuery HistoryQuery
    
        if sources, ok := query["source"]; ok {
            historyQuery.Sources = make([]string, 0, len(sources))
            
            for _, source := range sources {
                if len(source) != 0 {
                    historyQuery.Sources = append(historyQuery.Sources, source)
                }
            }
        } else {
            historyQuery.Sources = make([]string, 0)
        }
        
        if _, ok := query["limit"]; ok {
            limitString := query.Get("limit")
            limit, err := strconv.Atoi(limitString)
            
            if err != nil {
                Log.Warningf("GET /events: %v", err)
            
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusBadRequest)
                io.WriteString(w, string(ERequestQuery.JSON()) + "\n")
                
                return
            }
            
            historyQuery.Limit = limit
        }
        
        if _, ok := query["sortOrder"]; ok {
            sortOrder := query.Get("sortOrder")
            
            if sortOrder == "desc" || sortOrder == "asc" {
                historyQuery.Order = sortOrder
            }
        }
        
        if _, ok := query["data"]; ok {
            data := query.Get("data")
            
            historyQuery.Data = &data
        }
        
        if _, ok := query["maxAge"]; ok {
            maxAgeString := query.Get("maxAge")
            maxAge, err := strconv.Atoi(maxAgeString)
            
            if err != nil {
                Log.Warningf("GET /events: %v", err)
            
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusBadRequest)
                io.WriteString(w, string(ERequestQuery.JSON()) + "\n")
                
                return
            }
            
            if maxAge <= 0 {
                Log.Warningf("GET /events: Non positive age specified")
            
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusBadRequest)
                io.WriteString(w, string(ERequestQuery.JSON()) + "\n")
                
                return
            }
            
            nowMS := NanoToMilli(uint64(time.Now().UnixNano()))
            historyQuery.After = nowMS - uint64(maxAge)
        } else {
            if _, ok := query["afterTime"]; ok {
                afterString := query.Get("afterTime")
                after, err := strconv.Atoi(afterString)
            
                if err != nil {
                    Log.Warningf("GET /events: %v", err)
                
                    w.Header().Set("Content-Type", "application/json; charset=utf8")
                    w.WriteHeader(http.StatusBadRequest)
                    io.WriteString(w, string(ERequestQuery.JSON()) + "\n")
                    
                    return
                }
                
                if after < 0 {
                    Log.Warningf("GET /events: Non positive after specified")
            
                    w.Header().Set("Content-Type", "application/json; charset=utf8")
                    w.WriteHeader(http.StatusBadRequest)
                    io.WriteString(w, string(ERequestQuery.JSON()) + "\n")
                    
                    return
                }
                
                historyQuery.After = uint64(after)
            }
            
            if _, ok := query["beforeTime"]; ok {
                beforeString := query.Get("beforeTime")
                before, err := strconv.Atoi(beforeString)
            
                if err != nil {
                    Log.Warningf("GET /events: %v", err)
                
                    w.Header().Set("Content-Type", "application/json; charset=utf8")
                    w.WriteHeader(http.StatusBadRequest)
                    io.WriteString(w, string(ERequestQuery.JSON()) + "\n")
                    
                    return
                }
                
                if before < 0 {
                    Log.Warningf("GET /events: Non positive before specified")
            
                    w.Header().Set("Content-Type", "application/json; charset=utf8")
                    w.WriteHeader(http.StatusBadRequest)
                    io.WriteString(w, string(ERequestQuery.JSON()) + "\n")
                    
                    return
                }
                
                historyQuery.Before = uint64(before)
            }
        }
        
        eventIterator, err := server.historian.Query(&historyQuery)
        
        if err != nil {
            Log.Warningf("GET /events: Internal server error")
        
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, string(err.(DBerror).JSON()) + "\n")
            
            return
        }
        
        defer eventIterator.Release()
    
        flusher, _ := w.(http.Flusher)
        
        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.WriteHeader(http.StatusOK)
        
        for eventIterator.Next() {
            event := eventIterator.Event()
            eventJSON, _ := json.Marshal(&event)
            
            _, err = fmt.Fprintf(w, "%s\n", string(eventJSON))
            flusher.Flush()
            
            if err != nil {
                return
            }
        }
    }).Methods("GET")
    
    r.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
        query := r.URL.Query()
        
        var historyQuery HistoryQuery
    
        if _, ok := query["maxAge"]; ok {
            maxAgeString := query.Get("maxAge")
            maxAge, err := strconv.Atoi(maxAgeString)
            
            if err != nil {
                Log.Warningf("GET /events: %v", err)
            
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusBadRequest)
                io.WriteString(w, string(ERequestQuery.JSON()) + "\n")
                
                return
            }
            
            if maxAge <= 0 {
                Log.Warningf("GET /events: Non positive age specified")
            
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusBadRequest)
                io.WriteString(w, string(ERequestQuery.JSON()) + "\n")
                
                return
            }
            
            nowMS := NanoToMilli(uint64(time.Now().UnixNano()))
            historyQuery.After = nowMS - uint64(maxAge)
        } else {
            if _, ok := query["afterTime"]; ok {
                afterString := query.Get("afterTime")
                after, err := strconv.Atoi(afterString)
            
                if err != nil {
                    Log.Warningf("GET /events: %v", err)
                
                    w.Header().Set("Content-Type", "application/json; charset=utf8")
                    w.WriteHeader(http.StatusBadRequest)
                    io.WriteString(w, string(ERequestQuery.JSON()) + "\n")
                    
                    return
                }
                
                if after < 0 {
                    Log.Warningf("GET /events: Non positive after specified")
            
                    w.Header().Set("Content-Type", "application/json; charset=utf8")
                    w.WriteHeader(http.StatusBadRequest)
                    io.WriteString(w, string(ERequestQuery.JSON()) + "\n")
                    
                    return
                }
                
                historyQuery.After = uint64(after)
            }
            
            if _, ok := query["beforeTime"]; ok {
                beforeString := query.Get("beforeTime")
                before, err := strconv.Atoi(beforeString)
            
                if err != nil {
                    Log.Warningf("GET /events: %v", err)
                
                    w.Header().Set("Content-Type", "application/json; charset=utf8")
                    w.WriteHeader(http.StatusBadRequest)
                    io.WriteString(w, string(ERequestQuery.JSON()) + "\n")
                    
                    return
                }
                
                if before < 0 {
                    Log.Warningf("GET /events: Non positive before specified")
            
                    w.Header().Set("Content-Type", "application/json; charset=utf8")
                    w.WriteHeader(http.StatusBadRequest)
                    io.WriteString(w, string(ERequestQuery.JSON()) + "\n")
                    
                    return
                }
                
                historyQuery.Before = uint64(before)
            }
        }
        
        err := server.historian.Purge(&historyQuery)
        
        if err != nil {
            Log.Warningf("GET /events: Internal server error")
        
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, string(err.(DBerror).JSON()) + "\n")
            
            return
        }
        
        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, "\n")
    }).Methods("DELETE")
    
    r.HandleFunc("/{bucket}/batch", func(w http.ResponseWriter, r *http.Request) {
        startTime := time.Now()
        bucket := mux.Vars(r)["bucket"]
        
        if !server.bucketList.HasBucket(bucket) {
            Log.Warningf("POST /{bucket}/batch: Invalid bucket")
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, string(EInvalidBucket.JSON()) + "\n")
            
            return
        }
        
        if !server.bucketList.Get(bucket).ShouldAcceptWrites("") {
            Log.Warningf("POST /{bucket}/batch: Attempted to read from %s bucket", bucket)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusUnauthorized)
            io.WriteString(w, string(EUnauthorized.JSON()) + "\n")
            
            return
        }
        
        var updateBatch UpdateBatch
        var transportUpdateBatch TransportUpdateBatch
        decoder := json.NewDecoder(r.Body)
        err := decoder.Decode(&transportUpdateBatch)
        
        if err != nil {
            Log.Warningf("POST /{bucket}/batch: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EInvalidBatch.JSON()) + "\n")
            
            return
        }
        
        err = transportUpdateBatch.ToUpdateBatch(&updateBatch)
        
        if err != nil {
            Log.Warningf("POST /{bucket}/batch: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EInvalidBatch.JSON()) + "\n")
            
            return
        }
        
        updatedSiblingSets, err := server.bucketList.Get(bucket).Batch(&updateBatch)
        
        if err != nil {
            Log.Warningf("POST /{bucket}/batch: Internal server error")
        
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, string(err.(DBerror).JSON()) + "\n")
            
            return
        }
   
        if server.hub != nil {
            server.hub.BroadcastUpdate("", bucket, updatedSiblingSets, server.syncPushBroadcastLimit)
        }
        
        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, "\n")

        Log.Debugf("Batch update to bucket %s took %s", bucket, time.Since(startTime))
    }).Methods("POST")
    
    r.HandleFunc("/peers", func(w http.ResponseWriter, r *http.Request) {
        // { id: peerID, direction: direction, status: status }
        var peers []*PeerJSON
        
        if server.hub != nil {
            peers = server.hub.Peers()
        } else {
            peers = make([]*PeerJSON, 0)
        }
        
        peersJSON, _ := json.Marshal(peers)
        
        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, string(peersJSON) + "\n")
    }).Methods("GET")
    
    r.HandleFunc("/peers/{peerID}", func(w http.ResponseWriter, r *http.Request) {
        peerID := mux.Vars(r)["peerID"]
        
        var pa peerAddress
        decoder := json.NewDecoder(r.Body)
        err := decoder.Decode(&pa)
        
        if err != nil {
            Log.Warningf("PUT /peer/{peerID}: %v", err)
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, string(EInvalidPeer.JSON()) + "\n")
            
            return
        }
        
        if server.hub != nil {
            server.hub.Connect(peerID, pa.Host, pa.Port)
        }
        
        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, "\n")
    }).Methods("PUT")
    
    r.HandleFunc("/peers/{peerID}", func(w http.ResponseWriter, r *http.Request) {
        peerID := mux.Vars(r)["peerID"]
        
        if server.hub != nil {
            server.hub.Disconnect(peerID)
        }
        
        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, "\n")
    }).Methods("DELETE")
    
    r.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
        if server.hub == nil {
            // log error
            
            return
        }
        
        conn, err := server.upgrader.Upgrade(w, r, nil)
        
        if err != nil {
            return
        }
        
        server.hub.Accept(conn, 0, "", "", false)
    }).Methods("GET")

    r.HandleFunc("/debug/pprof/", pprof.Index)
    r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
    r.HandleFunc("/debug/pprof/profile", pprof.Profile)
    r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
    
    server.httpServer = &http.Server{
        Handler: r,
        WriteTimeout: 0,
        ReadTimeout: 15 * time.Second,
    }
    
    var listener net.Listener
    var err error

    if server.serverTLS == nil {
        listener, err = net.Listen("tcp", "0.0.0.0:" + strconv.Itoa(server.Port()))
    } else {
        server.serverTLS.ClientAuth = tls.VerifyClientCertIfGiven
        listener, err = tls.Listen("tcp", "0.0.0.0:" + strconv.Itoa(server.Port()), server.serverTLS)
    }
    
    if err != nil {
        Log.Errorf("Error listening on port: %d", server.port)
        
        server.Stop()
        
        return err
    }
    
    err = server.storageDriver.Open()
    
    if err != nil {
        if err != ECorrupted {
            Log.Errorf("Error opening storage driver: %v", err.Error())
            
            return EStorage
        }

        Log.Error("Database is corrupted. Attempting automatic recovery now...")

        recoverError := server.recover()

        if recoverError != nil {
            Log.Criticalf("Unable to recover corrupted database. Reason: %v", recoverError.Error())
            Log.Critical("Database daemon will now exit")

            return EStorage
        }
    }
    
    server.listener = listener

    Log.Infof("Node %s listening on port %d", server.id, server.port)

    err = server.httpServer.Serve(server.listener)

    Log.Errorf("Node %s server shutting down. Reason: %v", server.id, err)

    return err
}

func (server *Server) Stop() error {
    if server.listener != nil {
        server.listener.Close()
    }
    
    server.storageDriver.Close()
    
    return nil
}