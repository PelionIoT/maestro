package server

import (
    "github.com/gorilla/websocket"
    "github.com/prometheus/client_golang/prometheus"
    "sync"
    "errors"
    "time"
    "crypto/tls"
    crand "crypto/rand"
    "fmt"
    "encoding/binary"
    "encoding/json"
    "strconv"
    "net/http"
    "io/ioutil"
    "bytes"

    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/historian"
    . "github.com/armPelionEdge/devicedb/alerts"
    . "github.com/armPelionEdge/devicedb/logging"
    ddbSync "github.com/armPelionEdge/devicedb/sync"
)

const (
    INCOMING = iota
    OUTGOING = iota
)

const SYNC_SESSION_WAIT_TIMEOUT_SECONDS = 5
const RECONNECT_WAIT_MAX_SECONDS = 32
const WRITE_WAIT_SECONDS = 10
const PONG_WAIT_SECONDS = 60
const PING_PERIOD_SECONDS = 40
const CLOUD_PEER_ID = "cloud"

var (
    prometheusRelayConnectionsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
        Namespace: "relays",
        Subsystem: "devicedb_internal",
        Name: "connections",
        Help: "The number of current relay connections",
    })
)

func init() {
    prometheus.MustRegister(prometheusRelayConnectionsGauge)
}

func randomID() string {
    randomBytes := make([]byte, 16)
    crand.Read(randomBytes)
    
    high := binary.BigEndian.Uint64(randomBytes[:8])
    low := binary.BigEndian.Uint64(randomBytes[8:])
    
    return fmt.Sprintf("%016x%016x", high, low)
}

type PeerJSON struct {
    Direction string `json:"direction"`
    ID string `json:"id"`
    Status string `json:"status"`
}

type Peer struct {
    id string
    connection *websocket.Conn
    direction int
    closed bool
    closeChan chan bool
    doneChan chan bool
    csLock sync.Mutex
    rttLock sync.Mutex
    roundTripTime time.Duration
    result error
    uri string
    historyURI string
    alertsURI string
    partitionNumber uint64
    siteID string
    httpClient *http.Client
    httpHistoryClient *http.Client
    httpAlertsClient *http.Client
    identityHeader string
}

func NewPeer(id string, direction int) *Peer {
    return &Peer{
        id: id,
        direction: direction,
        closeChan: make(chan bool, 1),
    }
}

func (peer *Peer) errors() error {
    return peer.result
}

func (peer *Peer) accept(connection *websocket.Conn) (chan *SyncMessageWrapper, chan *SyncMessageWrapper, error) {
    peer.csLock.Lock()
    defer peer.csLock.Unlock()
    
    if peer.closed {
        return nil, nil, errors.New("Peer closed")
    }
    
    peer.connection = connection
    
    incoming, outgoing := peer.establishChannels()
    
    return incoming, outgoing, nil
}

func (peer *Peer) connect(dialer *websocket.Dialer, uri string) (chan *SyncMessageWrapper, chan *SyncMessageWrapper, error) {
    reconnectWaitSeconds := 1
    
    peer.uri = uri
    peer.httpClient = &http.Client{ Transport: &http.Transport{ TLSClientConfig: dialer.TLSClientConfig } }

    var header http.Header = make(http.Header)

    if peer.identityHeader != "" {
        header.Set("X-WigWag-RelayID", peer.identityHeader)
    }
    
    for {
        peer.connection = nil

        conn, _, err := dialer.Dial(uri, header)
                
        if err != nil {
            Log.Warningf("Unable to connect to peer %s at %s: %v. Reconnecting in %ds...", peer.id, uri, err, reconnectWaitSeconds)
            
            select {
            case <-time.After(time.Second * time.Duration(reconnectWaitSeconds)):
            case <-peer.closeChan:
                Log.Debugf("Cancelled connection retry sequence for %s", peer.id)
                
                return nil, nil, errors.New("Peer closed")
            }
            
            if reconnectWaitSeconds != RECONNECT_WAIT_MAX_SECONDS {
                reconnectWaitSeconds *= 2
            }
        } else {
            peer.csLock.Lock()
            defer peer.csLock.Unlock()
            
            if !peer.closed {
                peer.connection = conn
                
                incoming, outgoing := peer.establishChannels()
                
                return incoming, outgoing, nil
            }
            
            Log.Debugf("Cancelled connection retry sequence for %s", peer.id)
            
            closeWSConnection(conn, websocket.CloseNormalClosure)
            
            return nil, nil, errors.New("Peer closed")
        }
    }
}

func (peer *Peer) establishChannels() (chan *SyncMessageWrapper, chan *SyncMessageWrapper) {
    connection := peer.connection
    peer.doneChan = make(chan bool, 1)
    
    incoming := make(chan *SyncMessageWrapper)
    outgoing := make(chan *SyncMessageWrapper)
    
    go func() {
        pingTicker := time.NewTicker(time.Second * PING_PERIOD_SECONDS)

        for {
            select {
            case msg, ok := <-outgoing:
                // this lock ensures mutual exclusion with close message sending in peer.close()
                peer.csLock.Lock()
                connection.SetWriteDeadline(time.Now().Add(time.Second * WRITE_WAIT_SECONDS))

                if !ok {
                    connection.Close()
                    peer.csLock.Unlock()
                    return
                }

                err := connection.WriteJSON(msg)
                peer.csLock.Unlock()

                if err != nil {
                    Log.Errorf("Error writing to websocket for peer %s: %v", peer.id, err)
                    //return
                }
            case <-pingTicker.C:
                // this lock ensures mutual exclusion with close message sending in peer.close()
                Log.Infof("Sending a ping to peer %s", peer.id)
                peer.csLock.Lock()
                connection.SetWriteDeadline(time.Now().Add(time.Second * WRITE_WAIT_SECONDS))

                encodedPingTime, _ := time.Now().MarshalJSON()
                if err := connection.WriteMessage(websocket.PingMessage, encodedPingTime); err != nil {
                    Log.Errorf("Unable to send ping to peer %s: %v", peer.id, err.Error())
                }

                peer.csLock.Unlock()
            }
        }
    }()
    
    // incoming, outgoing, err
    go func() {
        defer close(peer.doneChan)

        connection.SetPongHandler(func(encodedPingTime string) error {
            var pingTime time.Time

            if err := pingTime.UnmarshalJSON([]byte(encodedPingTime)); err == nil {
                Log.Infof("Received pong from peer %s. Round trip time: %v", peer.id, time.Since(pingTime))
                peer.setRoundTripTime(time.Since(pingTime))
            } else {
                Log.Infof("Received pong from peer %s", peer.id)
            }

            connection.SetReadDeadline(time.Now().Add(time.Second * PONG_WAIT_SECONDS))

            return nil
        })
        
        for {
            var nextRawMessage rawSyncMessageWrapper
            var nextMessage SyncMessageWrapper

            // The pong handler is invoked in the same goroutine as ReadJSON (ReadJSON calls NextReader which calls advanceFrame which will invoke the pong handler
            // if it receives a pong frame). If the pong handler is invoked it will call SetReadDeadline in the same goroutine. In case a pong is never received
            // to reset the read deadline it it necessary to set the read deadline before every call to ReadJSON(). There was a bug where connections that were broken
            // were never receiving any data but kept attempting writes. This was because the read deadline was met by receiving a data frame right before the connection
            // broke and then no pong was ever received to again set the read deadline for the next call to ReadJSON() so ReadJSON() just hung so the broken connections
            // were never cleaned up.
            connection.SetReadDeadline(time.Now().Add(time.Second * PONG_WAIT_SECONDS))

            err := connection.ReadJSON(&nextRawMessage)
            
            if err != nil {
                if err.Error() == "websocket: close 1000 (normal)" {
                    Log.Infof("Received a normal websocket close message from peer %s", peer.id)
                } else {
                    Log.Errorf("Peer %s sent a misformatted message. Unable to parse: %v", peer.id, err)
                }
                
                peer.result = err
                
                close(incoming)
            
                return
            }
            
            nextMessage.SessionID = nextRawMessage.SessionID
            nextMessage.MessageType = nextRawMessage.MessageType
            nextMessage.Direction = nextRawMessage.Direction
            
            err = peer.typeCheck(&nextRawMessage, &nextMessage)
            
            if err != nil {
                peer.result = err
                
                close(incoming)
                
                return
            }
            
            nextMessage.nodeID = peer.id
            
            incoming <- &nextMessage
        }    
    }()
    
    return incoming, outgoing
}

func (peer *Peer) setRoundTripTime(duration time.Duration) {
    peer.rttLock.Lock()
    defer peer.rttLock.Unlock()

    peer.roundTripTime = duration
}

func (peer *Peer) getRoundTripTime() time.Duration {
    peer.rttLock.Lock()
    defer peer.rttLock.Unlock()

    return peer.roundTripTime
}

func (peer *Peer) typeCheck(rawMsg *rawSyncMessageWrapper, msg *SyncMessageWrapper) error {
    var err error = nil
    
    switch msg.MessageType {
    case SYNC_START:
        var start Start
        err = json.Unmarshal(rawMsg.MessageBody, &start)
        msg.MessageBody = start
    case SYNC_ABORT:
        var abort Abort
        err = json.Unmarshal(rawMsg.MessageBody, &abort)
        msg.MessageBody = abort
    case SYNC_NODE_HASH:
        var nodeHash MerkleNodeHash
        err = json.Unmarshal(rawMsg.MessageBody, &nodeHash)
        msg.MessageBody = nodeHash
    case SYNC_OBJECT_NEXT:
        var objectNext ObjectNext
        err = json.Unmarshal(rawMsg.MessageBody, &objectNext)
        msg.MessageBody = objectNext
    case SYNC_PUSH_MESSAGE:
        var pushMessage PushMessage
        err = json.Unmarshal(rawMsg.MessageBody, &pushMessage)
        msg.MessageBody = pushMessage
    case SYNC_PUSH_DONE:
        var pushDoneMessage PushDone
        err = json.Unmarshal(rawMsg.MessageBody, &pushDoneMessage)
        msg.MessageBody = pushDoneMessage
    }
    
    return err
}

func (peer *Peer) close(closeCode int) {
    peer.csLock.Lock()
    defer peer.csLock.Unlock()

    if !peer.closed {
        peer.closeChan <- true
        peer.closed = true
    }
        
    if peer.connection != nil {
        err := peer.connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, ""))
        
        if err != nil {
            return
        }
            
        select {
        case <-peer.doneChan:
        case <-time.After(time.Second):
        }
        
        peer.connection.Close()
    }
}

func (peer *Peer) isClosed() bool {
    return peer.closed
}

func (peer *Peer) toJSON(peerID string) *PeerJSON {
    var direction string
    var status string
    
    if peer.direction == INCOMING {
        direction = "incoming"
    } else {
        direction = "outgoing"
    }
    
    if peer.connection == nil {
        status = "down"
    } else {
        status = "up"
    }
    
    return &PeerJSON{
        Direction: direction,
        Status: status,
        ID: peerID,
    }
}

func (peer *Peer) useHistoryServer(tlsBaseConfig *tls.Config, historyServerName string, historyURI string, alertsServerName string, alertsURI string, noValidate bool) {
    peer.alertsURI = alertsURI
    peer.historyURI = historyURI
   
    tlsConfig := *tlsBaseConfig
    tlsConfig.InsecureSkipVerify = noValidate
    tlsConfig.ServerName = historyServerName
    tlsConfig.RootCAs = nil // ensures that this uses the default root CAs
    
    peer.httpHistoryClient = &http.Client{ Transport: &http.Transport{ TLSClientConfig: &tlsConfig } }

    tlsConfig = *tlsBaseConfig
    tlsConfig.InsecureSkipVerify = noValidate
    tlsConfig.ServerName = alertsServerName
    tlsConfig.RootCAs = nil

    peer.httpAlertsClient = &http.Client{ Transport: &http.Transport{ TLSClientConfig: &tlsConfig } }
}

func (peer *Peer) pushEvents(events []*Event) error {
    // try to forward event to the cloud if failed or error response then return
    eventsJSON, _ := json.Marshal(MakeeventsFromEvents(events))
    request, err := http.NewRequest("POST", peer.historyURI, bytes.NewReader(eventsJSON))
    
    if err != nil {
        return err
    }
    
    request.Header.Add("Content-Type", "application/json")
    
    resp, err := peer.httpHistoryClient.Do(request)
    
    if err != nil {
        return err
    }
    
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        errorMessage, err := ioutil.ReadAll(resp.Body)
        
        if err != nil {
            return err
        }
        
        return errors.New(fmt.Sprintf("Received error code from server: (%d) %s", resp.StatusCode, string(errorMessage)))
    }
    
    return nil
}

func (peer *Peer) pushAlerts(alerts map[string]Alert) error {
    var alertsList []Alert = make([]Alert, 0, len(alerts))

    for _, alert := range alerts {
        alertsList = append(alertsList, alert)
    }

    alertsJSON, _ := json.Marshal(alertsList)
    request, err := http.NewRequest("POST", peer.alertsURI, bytes.NewReader(alertsJSON))
    
    if err != nil {
        return err
    }
    
    request.Header.Add("Content-Type", "application/json")
    
    resp, err := peer.httpHistoryClient.Do(request)
    
    if err != nil {
        return err
    }
    
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        errorMessage, err := ioutil.ReadAll(resp.Body)
        
        if err != nil {
            return err
        }
        
        return errors.New(fmt.Sprintf("Received error code from server: (%d) %s", resp.StatusCode, string(errorMessage)))
    }
    
    return nil
}

func closeWSConnection(conn *websocket.Conn, closeCode int) {
    done := make(chan bool)
    
    go func() {
        defer close(done)
        
        for {
            _, _, err := conn.ReadMessage()
            
            if err != nil {
                return
            }
        }
    }()
            
    err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, ""))
        
    if err != nil {
        return
    }
    
    select {
    case <-done:
    case <-time.After(time.Second):
    }
    
    conn.Close()
}

type Hub struct {
    id string
    tlsConfig *tls.Config
    peerMapLock sync.Mutex
    peerMap map[string]*Peer
    peerMapByPartitionNumber map[uint64]map[string]*Peer
    peerMapBySiteID map[string]map[string]*Peer
    syncController *SyncController
    forwardEvents chan int
    forwardAlerts chan int
    historian *Historian
    alertsMap *AlertMap
    purgeOnForward bool
    forwardBatchSize uint64
    forwardThreshold uint64
    forwardInterval uint64
    alertsForwardInterval uint64
}

func NewHub(id string, syncController *SyncController, tlsConfig *tls.Config) *Hub {
    hub := &Hub{
        syncController: syncController,
        tlsConfig: tlsConfig,
        peerMap: make(map[string]*Peer),
        peerMapByPartitionNumber: make(map[uint64]map[string]*Peer),
        peerMapBySiteID: make(map[string]map[string]*Peer),
        id: id,
        forwardEvents: make(chan int, 1),
        forwardAlerts: make(chan int, 1),
    }
    
    return hub
}

func (hub *Hub) Accept(connection *websocket.Conn, partitionNumber uint64, relayID string, siteID string, noValidate bool) error {
    conn := connection.UnderlyingConn()
    
    if _, ok := conn.(*tls.Conn); ok || relayID != "" {
        var peerID string
        var err error

        if _, ok := conn.(*tls.Conn); ok {
            peerID, err = hub.ExtractPeerID(conn.(*tls.Conn))
        } else {
            peerID = relayID
        }
        
        if err != nil {
            if !noValidate {
                Log.Warningf("Unable to accept peer connection: %v", err)
                
                closeWSConnection(connection, websocket.CloseNormalClosure)
                
                return err
            }

            peerID = relayID
        }

        if noValidate && relayID != "" {
            peerID = relayID
        }

        if peerID == "" {
            Log.Warningf("Unable to accept peer connection")

            closeWSConnection(connection, websocket.CloseNormalClosure)

            return errors.New("Relay id not known")
        }

        go func() {
            peer := NewPeer(peerID, INCOMING)
            peer.partitionNumber = partitionNumber
            peer.siteID = siteID
            
            if !hub.register(peer) {
                Log.Warningf("Rejected peer connection from %s because that peer is already connected", peerID)
                
                closeWSConnection(connection, websocket.CloseTryAgainLater)
                
                return
            }
            
            incoming, outgoing, err := peer.accept(connection)
            
            if err != nil {
                Log.Errorf("Unable to accept peer connection from %s: %v. Closing connection and unregistering peer", peerID, err)

                closeWSConnection(connection, websocket.CloseNormalClosure)

                hub.unregister(peer)
                
                return
            }
            
            Log.Infof("Accepted peer connection from %s", peerID)
            
            hub.syncController.addPeer(peer.id, outgoing)
                
            for msg := range incoming {
                hub.syncController.incoming <- msg
            }
            
            hub.syncController.removePeer(peer.id)
            hub.unregister(peer)
            
            Log.Infof("Disconnected from peer %s", peerID)
        }()
    } else {
        return errors.New("Cannot accept non-secure connections")
    }
        
    return nil
}

func (hub *Hub) ConnectCloud(serverName, uri, historyServerName, historyURI, alertsServerName, alertsURI string, noValidate bool) error {
    if noValidate {
        Log.Warningf("The cloud.noValidate option is set to true. The cloud server's certificate chain and identity will not be verified. !!! THIS OPTION SHOULD NOT BE SET TO TRUE IN PRODUCTION !!!")
    }
    
    dialer, err := hub.dialer(serverName, noValidate, true)
    
    if err != nil {
        return err
    }
    
    go func() {
        peer := NewPeer(CLOUD_PEER_ID, OUTGOING)
    
        // simply try to reserve a spot in the peer map
        if !hub.register(peer) {
            return
        }
    
        for {
            // connect will return an error once the peer is disconnected for good
            peer.useHistoryServer(hub.tlsConfig, historyServerName, historyURI, alertsServerName, alertsURI, noValidate)
            peer.identityHeader = hub.id
            incoming, outgoing, err := peer.connect(dialer, uri)
            
            if err != nil {
                break
            }
            
            Log.Infof("Connected to devicedb cloud")
            
            hub.syncController.addPeer(peer.id, outgoing)
        
            // incoming is closed when the peer is disconnected from either end
            for msg := range incoming {
                hub.syncController.incoming <- msg
            }
        
            hub.syncController.removePeer(peer.id)
            
            if websocket.IsCloseError(peer.errors(), websocket.CloseNormalClosure) {
                Log.Infof("Disconnected from devicedb cloud")
            
                break
            }
            
            Log.Infof("Disconnected from devicedb cloud. Reconnecting...")
            <-time.After(time.Second)
        }
        
        hub.unregister(peer)
    }()
    
    return nil
}

func (hub *Hub) Connect(peerID, host string, port int) error {
    dialer, err := hub.dialer(peerID, false, false)
    
    if peerID == CLOUD_PEER_ID {
        Log.Warningf("Peer ID is not allowed to be %s since it is reserved for the cloud connection. This node will not connect to this peer", CLOUD_PEER_ID)
        
        return errors.New("Peer ID is not allowed to be " + CLOUD_PEER_ID)
    }
    
    if err != nil {
        return err
    }    
    
    go func() {
        peer := NewPeer(peerID, OUTGOING)
    
        // simply try to reserve a spot in the peer map
        if !hub.register(peer) {
            return
        }
    
        for {
            // connect will return an error once the peer is disconnected for good
            incoming, outgoing, err := peer.connect(dialer, "wss://" + host + ":" + strconv.Itoa(port) + "/sync")
            
            if err != nil {
                break
            }
            
            Log.Infof("Connected to peer %s", peer.id)
            
            hub.syncController.addPeer(peer.id, outgoing)
        
            // incoming is closed when the peer is disconnected from either end
            for msg := range incoming {
                hub.syncController.incoming <- msg
            }
        
            hub.syncController.removePeer(peer.id)
            
            if websocket.IsCloseError(peer.errors(), websocket.CloseNormalClosure) {
                Log.Infof("Disconnected from peer %s", peer.id)
            
                break
            }
            
            Log.Infof("Disconnected from peer %s. Reconnecting...", peer.id)
            <-time.After(time.Second)
        }
        
        hub.unregister(peer)
    }()
    
    return nil
}

func (hub *Hub) Disconnect(peerID string) {
    hub.peerMapLock.Lock()
    defer hub.peerMapLock.Unlock()
    
    peer, ok := hub.peerMap[peerID]
    
    if ok {
        peer.close(websocket.CloseNormalClosure)
    }
}

func (hub *Hub) PeerStatus(peerID string) (connected bool, pingTime time.Duration) {
    hub.peerMapLock.Lock()
    defer hub.peerMapLock.Unlock()

    peer, ok := hub.peerMap[peerID]
    
    if ok {
        return true, peer.getRoundTripTime()
    }

    return false, 0
}

func (hub *Hub) ReconnectPeer(peerID string) {
    hub.peerMapLock.Lock()
    defer hub.peerMapLock.Unlock()
    
    peer, ok := hub.peerMap[peerID]
    
    if ok {
        peer.close(websocket.CloseTryAgainLater)
    }
}

func (hub *Hub) ReconnectPeerByPartition(partitionNumber uint64) {
    hub.peerMapLock.Lock()
    defer hub.peerMapLock.Unlock()

    if peers, ok := hub.peerMapByPartitionNumber[partitionNumber]; ok {
        for _, peer := range peers {
            peer.close(websocket.CloseTryAgainLater)
        }
    }
}

func (hub *Hub) ReconnectPeerBySite(siteID string) {
    hub.peerMapLock.Lock()
    defer hub.peerMapLock.Unlock()

    if peers, ok := hub.peerMapBySiteID[siteID]; ok {
        for _, peer := range peers {
            peer.close(websocket.CloseTryAgainLater)
        }
    }
}

func (hub *Hub) dialer(peerID string, noValidate bool, useDefaultRootCAs bool) (*websocket.Dialer, error) {
    if hub.tlsConfig == nil {
        return nil, errors.New("No tls config provided")
    }
    
    tlsConfig := hub.tlsConfig.Clone()
    
    if useDefaultRootCAs {
        tlsConfig.RootCAs = nil
    }
    
    tlsConfig.InsecureSkipVerify = noValidate
    tlsConfig.ServerName = peerID
    
    dialer := &websocket.Dialer{
        TLSClientConfig: tlsConfig,
    }
    
    return dialer, nil
}

func (hub *Hub) register(peer *Peer) bool {
    hub.peerMapLock.Lock()
    defer hub.peerMapLock.Unlock()
    
    if _, ok := hub.peerMap[peer.id]; ok {
        return false
    }
    
    Log.Debugf("Register peer %s", peer.id)
    hub.peerMap[peer.id] = peer
    
    if _, ok := hub.peerMapByPartitionNumber[peer.partitionNumber]; !ok {
        hub.peerMapByPartitionNumber[peer.partitionNumber] = make(map[string]*Peer)
    }

    if _, ok := hub.peerMapBySiteID[peer.siteID]; !ok {
        hub.peerMapBySiteID[peer.siteID] = make(map[string]*Peer)
    }

    hub.peerMapByPartitionNumber[peer.partitionNumber][peer.id] = peer
    hub.peerMapBySiteID[peer.siteID][peer.id] = peer
    
    return true
}

func (hub *Hub) unregister(peer *Peer) {
    hub.peerMapLock.Lock()
    defer hub.peerMapLock.Unlock()

    if _, ok := hub.peerMap[peer.id]; ok {
        Log.Debugf("Unregister peer %s", peer.id)
    }
    
    delete(hub.peerMap, peer.id)
    delete(hub.peerMapByPartitionNumber[peer.partitionNumber], peer.id)
    delete(hub.peerMapBySiteID[peer.siteID], peer.id)

    if len(hub.peerMapByPartitionNumber[peer.partitionNumber]) == 0 {
        delete(hub.peerMapByPartitionNumber, peer.partitionNumber)
    }

    if len(hub.peerMapBySiteID[peer.siteID]) == 0 {
        delete(hub.peerMapBySiteID, peer.siteID)
    }
}

func (hub *Hub) Peers() []*PeerJSON {
    hub.peerMapLock.Lock()
    defer hub.peerMapLock.Unlock()
    peers := make([]*PeerJSON, 0, len(hub.peerMap))
    
    for peerID, ps := range hub.peerMap {
        peers = append(peers, ps.toJSON(peerID))
    }
    
    return peers
}

func (hub *Hub) ExtractPeerID(conn *tls.Conn) (string, error) {
    // VerifyClientCertIfGiven
    verifiedChains := conn.ConnectionState().VerifiedChains
    
    if len(verifiedChains) != 1 {
        return "", errors.New("Invalid client certificate")
    }
    
    peerID := verifiedChains[0][0].Subject.CommonName
    
    return peerID, nil
}

func (hub *Hub) SyncController() *SyncController {
    return hub.syncController
}

func (hub *Hub) ForwardEvents() {
    if hub.historian.LogSerial() - hub.historian.ForwardIndex() - 1 >= hub.forwardThreshold {
        select {
        case hub.forwardEvents <- 1:
        default:
        }
    }
}

func (hub *Hub) StartForwardingEvents() {
    go func() {
        for {
            select {
            case <-hub.forwardEvents:
            case <-time.After(time.Millisecond * time.Duration(hub.forwardInterval)):
            }

            Log.Info("Begin event forwarding to the cloud")
            
            var cloudPeer *Peer
            
            hub.peerMapLock.Lock()
            cloudPeer, ok := hub.peerMap[CLOUD_PEER_ID]
            hub.peerMapLock.Unlock()
            
            if !ok {
                Log.Info("No cloud present. Nothing to forward to")
                
                continue
            }

            for hub.historian.ForwardIndex() < hub.historian.LogSerial() - 1 {
                minSerial := hub.historian.ForwardIndex() + 1
                eventIterator, err := hub.historian.Query(&HistoryQuery{ MinSerial: &minSerial, Limit: int(hub.forwardBatchSize) })
                
                if err != nil {
                    Log.Criticalf("Unable to query event history: %v. No more events will be forwarded to the cloud", err)
                    
                    return
                }

                var highestIndex uint64 = minSerial
                var batch []*Event = make([]*Event, 0, int(hub.forwardBatchSize))
                
                for eventIterator.Next() {
                    if eventIterator.Event().Serial > highestIndex {
                        highestIndex = eventIterator.Event().Serial
                    }

                    batch = append(batch, eventIterator.Event())
                }

                if eventIterator.Error() != nil {
                    Log.Criticalf("Unable to query event history. Event iterator error: %v. No more events will be forwarded to the cloud.", eventIterator.Error())
                    
                    return
                }

                Log.Debugf("Forwarding events %d to %d (inclusive) to the cloud.", minSerial, highestIndex)

                if err := cloudPeer.pushEvents(batch); err != nil {
                    Log.Warningf("Unable to push events to the cloud: %v. Event forwarding process will resume later.", err)

                    break
                }

                if err := hub.historian.SetForwardIndex(highestIndex); err != nil {
                    Log.Criticalf("Unable to update forwarding index after push: %v. No more events will be forwarded to the cloud", err)

                    return
                }

                if hub.purgeOnForward {
                    maxSerial := highestIndex + 1
                    
                    if err := hub.historian.Purge(&HistoryQuery{ MaxSerial: &maxSerial }); err != nil {
                        Log.Warningf("Unable to purge events after push: %v.")
                    }
                }
            }
            
            Log.Info("History forwarding complete. Sleeping...")
        }
    }()
}

func (hub *Hub) ForwardAlerts() {
    // select {
    // case hub.forwardAlerts <- 1:
    // default:
    // }
}

func (hub *Hub) StartForwardingAlerts() {
    go func() {
        for {
            select {
            case <-hub.forwardAlerts:
            case <-time.After(time.Millisecond * time.Duration(hub.alertsForwardInterval)):
            }

            Log.Info("Begin alert forwarding to the cloud")

            var cloudPeer *Peer
            
            hub.peerMapLock.Lock()
            cloudPeer, ok := hub.peerMap[CLOUD_PEER_ID]
            hub.peerMapLock.Unlock()
            
            if !ok {
                Log.Info("No cloud present. Nothing to forward to")
                
                continue
            }
            
            alerts, err := hub.alertsMap.GetAlerts()

            if err != nil {
                Log.Criticalf("Unable to query alerts map: %v. No more alerts will be forwarded to the cloud", err)

                return
            }

            if len(alerts) == 0 {
                Log.Infof("No new alerts. Nothing to forward")

                continue
            }

            if err := cloudPeer.pushAlerts(alerts); err != nil {
                Log.Warningf("Unable to push alerts to the cloud: %v. Alert forwarding process will resume later.", err)

                continue
            }

            if err := hub.alertsMap.ClearAlerts(alerts); err != nil {
                Log.Criticalf("Unable to clear alerts from the alert store after forwarding: %v. No more alerts will be forwarded to the cloud", err)

                return
            }

            Log.Info("Alert forwarding complete. Sleeping...")
        }
    }()
}

func (hub *Hub) BroadcastUpdate(siteID string, bucket string, update map[string]*SiblingSet, n uint64) {
    // broadcast the specified update to at most n peers, or all peers if n is non-positive
    var count uint64 = 0

    hub.peerMapLock.Lock()
    defer hub.peerMapLock.Unlock()

    peers := hub.peerMapBySiteID[siteID]

    for peerID, _ := range peers {
        if !hub.syncController.bucketProxyFactory.OutgoingBuckets(peerID)[bucket] {
            continue
        }

        if n != 0 && count == n {
            break
        }

        hub.syncController.BroadcastUpdate(peerID, bucket, update, n)
        count += 1
    }
}

type SyncSession struct {
    receiver chan *SyncMessageWrapper
    sender chan *SyncMessageWrapper
    sessionState interface{ }
    waitGroup *sync.WaitGroup
    peerID string
    sessionID uint
}

type SyncController struct {
    bucketProxyFactory ddbSync.BucketProxyFactory
    incoming chan *SyncMessageWrapper
    peers map[string]chan *SyncMessageWrapper
    waitGroups map[string]*sync.WaitGroup
    initiatorSessionsMap map[string]map[uint]*SyncSession
    responderSessionsMap map[string]map[uint]*SyncSession
    initiatorSessions chan *SyncSession
    responderSessions chan *SyncSession
    maxSyncSessions uint
    nextSessionID uint
    mapMutex sync.RWMutex
    syncScheduler ddbSync.SyncScheduler
    explorationPathLimit uint32
}

func NewSyncController(maxSyncSessions uint, bucketProxyFactory ddbSync.BucketProxyFactory, syncScheduler ddbSync.SyncScheduler, explorationPathLimit uint32) *SyncController {
    syncController := &SyncController{
        bucketProxyFactory: bucketProxyFactory,
        incoming: make(chan *SyncMessageWrapper),
        peers: make(map[string]chan *SyncMessageWrapper),
        waitGroups: make(map[string]*sync.WaitGroup),
        initiatorSessionsMap: make(map[string]map[uint]*SyncSession),
        responderSessionsMap: make(map[string]map[uint]*SyncSession),
        initiatorSessions: make(chan *SyncSession),
        responderSessions: make(chan *SyncSession),
        maxSyncSessions: maxSyncSessions,
        nextSessionID: 1,
        syncScheduler: syncScheduler,
        explorationPathLimit: explorationPathLimit,
    }
    
    go func() {
        // multiplex incoming messages accross sync sessions
        for msg := range syncController.incoming {
            syncController.receiveMessage(msg)
        }
    }()
    
    return syncController
}

func (s *SyncController) addPeer(peerID string, w chan *SyncMessageWrapper) error {
    prometheusRelayConnectionsGauge.Inc()
    s.mapMutex.Lock()
    defer s.mapMutex.Unlock()
    
    if _, ok := s.peers[peerID]; ok {
        return errors.New("Peer already registered")
    }
    
    s.peers[peerID] = w
    s.waitGroups[peerID] = &sync.WaitGroup{ }
    s.initiatorSessionsMap[peerID] = make(map[uint]*SyncSession)
    s.responderSessionsMap[peerID] = make(map[uint]*SyncSession)

    var buckets []string = make([]string, 0, len(s.bucketProxyFactory.IncomingBuckets(peerID)))

    for bucket, _ := range s.bucketProxyFactory.IncomingBuckets(peerID) {
        buckets = append(buckets, bucket)
    }

    s.syncScheduler.AddPeer(peerID, buckets)
    s.syncScheduler.Schedule(peerID)
    
    return nil
}

func (s *SyncController) removePeer(peerID string) {
    prometheusRelayConnectionsGauge.Dec()
    s.mapMutex.Lock()
    
    if _, ok := s.peers[peerID]; !ok {
        s.mapMutex.Unlock()
        return
    }
    
    for _, syncSession := range s.initiatorSessionsMap[peerID] {
        close(syncSession.receiver)
    }
    
    for _, syncSession := range s.responderSessionsMap[peerID] {
        close(syncSession.receiver)
    }

    delete(s.initiatorSessionsMap, peerID)
    delete(s.responderSessionsMap, peerID)

    s.syncScheduler.RemovePeer(peerID)

    wg := s.waitGroups[peerID]
    s.mapMutex.Unlock()
    wg.Wait()

    s.mapMutex.Lock()
    close(s.peers[peerID])
    delete(s.peers, peerID)
    delete(s.waitGroups, peerID)
    s.mapMutex.Unlock()
}

func (s *SyncController) addResponderSession(peerID string, sessionID uint, bucketName string) bool {
    if !s.bucketProxyFactory.OutgoingBuckets(peerID)[bucketName] {
        Log.Errorf("Unable to add responder session %d for peer %s because bucket %s does not allow outgoing messages to this peer", sessionID, peerID, bucketName)
        
        return false
    }
    
    bucketProxy, err := s.bucketProxyFactory.CreateBucketProxy(peerID, bucketName)

    if err != nil {
        Log.Errorf("Unable to add responder session %d for peer %s because a bucket proxy could not be created for bucket %s: %v", sessionID, peerID, bucketName, err)

        return false
    }
    
    s.mapMutex.Lock()
    defer s.mapMutex.Unlock()
    
    if _, ok := s.responderSessionsMap[peerID]; !ok {
        Log.Errorf("Unable to add responder session %d for peer %s because this peer is not registered with the sync controller", sessionID, peerID)
        
        return false
    }
    
    if _, ok := s.responderSessionsMap[peerID][sessionID]; ok {
        Log.Errorf("Unable to add responder session %d for peer %s because a responder session with this id for this peer already exists", sessionID, peerID)
        
        return false
    }
    
    newResponderSession := &SyncSession{
        receiver: make(chan *SyncMessageWrapper, 1),
        sender: s.peers[peerID],
        sessionState: NewResponderSyncSession(bucketProxy),
        waitGroup: s.waitGroups[peerID],
        peerID: peerID,
        sessionID: sessionID,
    }
    
    s.responderSessionsMap[peerID][sessionID] = newResponderSession
    newResponderSession.waitGroup.Add(1)
    
    select {
    case s.responderSessions <- newResponderSession:
        Log.Debugf("Added responder session %d for peer %s and bucket %s", sessionID, peerID, bucketName)
        
        return true
    default:
        Log.Warningf("Unable to add responder session %d for peer %s and bucket %s because there are already %d responder sync sessions", sessionID, peerID, bucketName, s.maxSyncSessions)
        
        delete(s.responderSessionsMap[peerID], sessionID)
        newResponderSession.waitGroup.Done()
        return false
    }
}

func (s *SyncController) addInitiatorSession(peerID string, sessionID uint, bucketName string) bool {
    if !s.bucketProxyFactory.IncomingBuckets(peerID)[bucketName] {
        Log.Errorf("Unable to add initiator session %d for peer %s because bucket %s does not allow incoming messages from this peer", sessionID, peerID, bucketName)
        
        return false
    }
    
    bucketProxy, err := s.bucketProxyFactory.CreateBucketProxy(peerID, bucketName)
    
    if err != nil {
        Log.Errorf("Unable to add initiator session %d for peer %s because a bucket proxy could not be created for bucket %s: %v", sessionID, peerID, bucketName, err)

        return false
    }

    s.mapMutex.Lock()
    defer s.mapMutex.Unlock()
    
    if _, ok := s.initiatorSessionsMap[peerID]; !ok {
        Log.Errorf("Unable to add initiator session %d for peer %s because this peer is not registered with the sync controller", sessionID, peerID)
        
        return false
    }
    
    newInitiatorSession := &SyncSession{
        receiver: make(chan *SyncMessageWrapper, 1),
        sender: s.peers[peerID],
        sessionState: NewInitiatorSyncSession(sessionID, bucketProxy, s.explorationPathLimit, s.bucketProxyFactory.OutgoingBuckets(peerID)[bucketName]),
        waitGroup: s.waitGroups[peerID],
        peerID: peerID,
        sessionID: sessionID,
    }
    
    // check map to see if it has this one
    if _, ok := s.initiatorSessionsMap[peerID][sessionID]; ok {
        Log.Errorf("Unable to add initiator session %d for peer %s because an initiator session with this id for this peer already exists", sessionID, peerID)
        
        return false
    }
    
    s.initiatorSessionsMap[peerID][sessionID] = newInitiatorSession
    newInitiatorSession.waitGroup.Add(1)
    
    select {
    case s.initiatorSessions <- newInitiatorSession:
        Log.Debugf("Added initiator session %d for peer %s and bucket %s", sessionID, peerID, bucketName)
        
        s.syncScheduler.Advance()
        s.initiatorSessionsMap[peerID][sessionID].receiver <- nil
        
        return true
    default:
        Log.Warningf("Unable to add initiator session %d for peer %s and bucket %s because there are already %d initiator sync sessions", sessionID, peerID, bucketName, s.maxSyncSessions)
        
        delete(s.initiatorSessionsMap[peerID], sessionID)
        newInitiatorSession.waitGroup.Done()
        return false
    }
}

func (s *SyncController) removeResponderSession(responderSession *SyncSession) {
    s.mapMutex.Lock()
    
    if _, ok := s.responderSessionsMap[responderSession.peerID]; ok {
        delete(s.responderSessionsMap[responderSession.peerID], responderSession.sessionID)
    }
    
    s.mapMutex.Unlock()
    responderSession.waitGroup.Done()
    
    Log.Debugf("Removed responder session %d for peer %s", responderSession.sessionID, responderSession.peerID)
}

func (s *SyncController) removeInitiatorSession(initiatorSession *SyncSession) {
    s.mapMutex.Lock()
    
    if _, ok := s.initiatorSessionsMap[initiatorSession.peerID]; ok {
        s.syncScheduler.Schedule(initiatorSession.peerID)
        delete(s.initiatorSessionsMap[initiatorSession.peerID], initiatorSession.sessionID)
    }
    
    s.mapMutex.Unlock()
    initiatorSession.waitGroup.Done()
    
    Log.Debugf("Removed initiator session %d for peer %s", initiatorSession.sessionID, initiatorSession.peerID)
}

func (s *SyncController) sendAbort(peerID string, sessionID uint, direction uint) {
    s.mapMutex.RLock()
    defer s.mapMutex.RUnlock()

    w := s.peers[peerID]
    
    if w != nil {
        w <- &SyncMessageWrapper{
            SessionID: sessionID,
            MessageType: SYNC_ABORT,
            MessageBody: &Abort{ },
            Direction: direction,
        }
    }
}

func (s *SyncController) receiveMessage(msg *SyncMessageWrapper) {
    nodeID := msg.nodeID
    sessionID := msg.SessionID    
    
    if msg.Direction == REQUEST {
        if msg.MessageType == SYNC_START {
            if !s.addResponderSession(nodeID, sessionID, msg.MessageBody.(Start).Bucket) {
                s.sendAbort(nodeID, sessionID, RESPONSE)
            }
        }
        
        s.mapMutex.RLock()
        defer s.mapMutex.RUnlock()
        
        if _, ok := s.responderSessionsMap[nodeID]; !ok {
            // s.sendAbort(nodeID, sessionID, RESPONSE)
            
            return
        }
        
        if _, ok := s.responderSessionsMap[nodeID][sessionID]; !ok {
            // s.sendAbort(nodeID, sessionID, RESPONSE)
            
            return
        }
        
        // This select statement means that the function does not block if
        // the read loop in runInitiatorSession is not running if the session
        // is being removed currently. This works due to the synchronous nature
        // of the protocol.
        select {
        case s.responderSessionsMap[nodeID][sessionID].receiver <- msg:
        default:
        }
    } else if msg.Direction == RESPONSE { // RESPONSE
        s.mapMutex.RLock()
        defer s.mapMutex.RUnlock()
        
        if _, ok := s.initiatorSessionsMap[nodeID]; !ok {
            // s.sendAbort(nodeID, sessionID, REQUEST)
            
            return
        }
        
        if _, ok := s.initiatorSessionsMap[nodeID][sessionID]; !ok {
            // s.sendAbort(nodeID, sessionID, REQUEST)
            
            return
        }
        
        select {
        case s.initiatorSessionsMap[nodeID][sessionID].receiver <- msg:
        default:
        }
    } else if msg.MessageType == SYNC_PUSH_MESSAGE {
        pushMessage := msg.MessageBody.(PushMessage)
        
        if !s.bucketProxyFactory.IncomingBuckets(nodeID)[pushMessage.Bucket] {
            Log.Errorf("Ignoring push message from %s because this node does not accept incoming pushes from bucket %s from that node", nodeID, pushMessage.Bucket)
            
            return
        }

        key := pushMessage.Key
        value := pushMessage.Value
        bucketProxy, err := s.bucketProxyFactory.CreateBucketProxy(nodeID, pushMessage.Bucket)

        if err != nil {
            Log.Errorf("Ignoring push message from %s for bucket %s because an error occurred while creating a bucket proxy: %v", nodeID, pushMessage.Bucket, err)

            return
        }

        err = bucketProxy.Merge(map[string]*SiblingSet{ key: value })
        
        if err != nil {
            Log.Errorf("Unable to merge object from peer %s into key %s in bucket %s: %v", nodeID, key, pushMessage.Bucket, err)
        } else {
            Log.Infof("Merged object from peer %s into key %s in bucket %s", nodeID, key, pushMessage.Bucket)
        }
    }
}

func (s *SyncController) runInitiatorSession() {
    for initiatorSession := range s.initiatorSessions {
        state := initiatorSession.sessionState.(*InitiatorSyncSession)
        
        for {
            var receivedMessage *SyncMessageWrapper
            
            select {
            case receivedMessage = <-initiatorSession.receiver:
            case <-time.After(time.Second * SYNC_SESSION_WAIT_TIMEOUT_SECONDS):
                Log.Warningf("[%s-%d] timeout", initiatorSession.peerID, initiatorSession.sessionID)
            }
            
            initialState := state.State()
            
            var m *SyncMessageWrapper = state.NextState(receivedMessage)
            
            m.Direction = REQUEST
    
            if receivedMessage == nil {
                Log.Debugf("[%s-%d] nil : (%s -> %s) : %s", initiatorSession.peerID, initiatorSession.sessionID, StateName(initialState), StateName(state.State()), MessageTypeName(m.MessageType))
            } else {
                Log.Debugf("[%s-%d] %s : (%s -> %s) : %s", initiatorSession.peerID, initiatorSession.sessionID, MessageTypeName(receivedMessage.MessageType), StateName(initialState), StateName(state.State()), MessageTypeName(m.MessageType))
            }
            
            if m != nil {
                initiatorSession.sender <- m
            }
            
            if state.State() == END {
                break
            }
        }
        
        s.removeInitiatorSession(initiatorSession)
    }
}

func (s *SyncController) runResponderSession() {
    for responderSession := range s.responderSessions {
        state := responderSession.sessionState.(*ResponderSyncSession)
        
        for {
            var receivedMessage *SyncMessageWrapper
            
            select {
            case receivedMessage = <-responderSession.receiver:
            case <-time.After(time.Second * SYNC_SESSION_WAIT_TIMEOUT_SECONDS):
                Log.Warningf("[%s-%d] timeout", responderSession.peerID, responderSession.sessionID)
            }
            
            initialState := state.State()
            
            var m *SyncMessageWrapper = state.NextState(receivedMessage)
        
            m.Direction = RESPONSE
            
            if receivedMessage == nil {
                Log.Debugf("[%s-%d] nil : (%s -> %s) : %s", responderSession.peerID, responderSession.sessionID, StateName(initialState), StateName(state.State()), MessageTypeName(m.MessageType))
            } else {
                Log.Debugf("[%s-%d] %s : (%s -> %s) : %s", responderSession.peerID, responderSession.sessionID, MessageTypeName(receivedMessage.MessageType), StateName(initialState), StateName(state.State()), MessageTypeName(m.MessageType))
            }
            
            if m != nil {
                responderSession.sender <- m
            }
            
            if state.State() == END {
                break
            }
        }
        
        s.removeResponderSession(responderSession)
    }
}

func (s *SyncController) StartInitiatorSessions() {
    for i := 0; i < int(s.maxSyncSessions); i += 1 {
        go s.runInitiatorSession()
    }
    
    go func() {
        for {
            peerID, bucketName := s.syncScheduler.Next()
            
            s.mapMutex.RLock()

            if peerID == "" {
                s.mapMutex.RUnlock()

                continue
            }

            s.mapMutex.RUnlock()
            
            if s.addInitiatorSession(peerID, s.nextSessionID, bucketName) {
                s.nextSessionID += 1
            } else {
            }
        }
    }()
}

func (s *SyncController) StartResponderSessions() {
    for i := 0; i < int(s.maxSyncSessions); i += 1 {
        go s.runResponderSession()
    }    
}

func (s *SyncController) Start() {
    s.StartInitiatorSessions()
    s.StartResponderSessions()
}

func (s *SyncController) BroadcastUpdate(peerID string, bucket string, update map[string]*SiblingSet, n uint64) {
    s.mapMutex.RLock()
    defer s.mapMutex.RUnlock()
   
    for key, value := range update {
        msg := &SyncMessageWrapper{
            SessionID: 0,
            MessageType: SYNC_PUSH_MESSAGE,
            MessageBody: PushMessage{
                Key: key,
                Value: value,
                Bucket: bucket,
            },
            Direction: PUSH,
        }
       
        w := s.peers[peerID]

        if w != nil {
            Log.Debugf("Push object at key %s in bucket %s to peer %s", key, bucket, peerID)
            w <- msg
        }
    }
}
