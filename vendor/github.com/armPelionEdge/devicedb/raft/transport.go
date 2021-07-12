package raft
//
 // Copyright (c) 2019 ARM Limited.
 //
 // SPDX-License-Identifier: MIT
 //
 // Permission is hereby granted, free of charge, to any person obtaining a copy
 // of this software and associated documentation files (the "Software"), to
 // deal in the Software without restriction, including without limitation the
 // rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
 // sell copies of the Software, and to permit persons to whom the Software is
 // furnished to do so, subject to the following conditions:
 //
 // The above copyright notice and this permission notice shall be included in all
 // copies or substantial portions of the Software.
 //
 // THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 // IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 // FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 // AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 // LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 // OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 // SOFTWARE.
 //


import (
    "time"
    "strings"
    "github.com/gorilla/mux"
    "github.com/coreos/etcd/raft/raftpb"

    . "github.com/armPelionEdge/devicedb/logging"

    "fmt"
    "errors"
    "net/http"
    "bytes"
    "io"
    "io/ioutil"
    "context"
    "sync"
)

var ESenderUnknown = errors.New("The receiver does not know who we are")
var EReceiverUnknown = errors.New("The sender does not know the receiver")
var ETimeout = errors.New("The sender timed out while trying to send the message to the receiver")

const (
    RequestTimeoutSeconds = 10
)

type PeerAddress struct {
    NodeID uint64
    Host string
    Port int
}

func (peerAddress *PeerAddress) ToHTTPURL(endpoint string) string {
    return fmt.Sprintf("http://%s:%d%s", peerAddress.Host, peerAddress.Port, endpoint)
}

func (peerAddress *PeerAddress) IsEmpty() bool {
    return peerAddress.NodeID == 0
}

type TransportHub struct {
    peers map[uint64]PeerAddress
    httpClient *http.Client
    onReceiveCB func(context.Context, raftpb.Message) error
    lock sync.Mutex
    localPeerID uint64
    defaultPeerAddress PeerAddress
}

func NewTransportHub(localPeerID uint64) *TransportHub {
    defaultTransport := http.DefaultTransport.(*http.Transport)
    transport := &http.Transport{}
    transport.MaxIdleConns = 0
    transport.MaxIdleConnsPerHost = 1000
    transport.IdleConnTimeout = defaultTransport.IdleConnTimeout

    hub := &TransportHub{
        localPeerID: localPeerID,
        peers: make(map[uint64]PeerAddress),
        httpClient: &http.Client{ 
            Timeout: time.Second * RequestTimeoutSeconds,
            Transport: transport,
        },
    }

    return hub
}

func (hub *TransportHub) SetLocalPeerID(id uint64) {
    hub.localPeerID = id
}

func (hub *TransportHub) SetDefaultRoute(host string, port int) {
    hub.defaultPeerAddress = PeerAddress{
        Host: host,
        Port: port,
    }
}

func (hub *TransportHub) PeerAddress(nodeID uint64) *PeerAddress {
    hub.lock.Lock()
    defer hub.lock.Unlock()
    
    peerAddress, ok := hub.peers[nodeID]

    if !ok {
        return nil
    }

    return &peerAddress
}

func (hub *TransportHub) AddPeer(peerAddress PeerAddress) {
    hub.lock.Lock()
    defer hub.lock.Unlock()
    hub.peers[peerAddress.NodeID] = peerAddress
}

func (hub *TransportHub) RemovePeer(peerAddress PeerAddress) {
    hub.lock.Lock()
    defer hub.lock.Unlock()
    delete(hub.peers, peerAddress.NodeID)
}

func (hub *TransportHub) UpdatePeer(peerAddress PeerAddress) {
    hub.lock.Lock()
    defer hub.lock.Unlock()
    hub.AddPeer(peerAddress)
}

func (hub *TransportHub) OnReceive(cb func(context.Context, raftpb.Message) error) {
    hub.onReceiveCB = cb
}

func (hub *TransportHub) Send(ctx context.Context, msg raftpb.Message, proxy bool) error {
    encodedMessage, err := msg.Marshal()

    if err != nil {
        return err
    }

    hub.lock.Lock()
    peerAddress, ok := hub.peers[msg.To]
    hub.lock.Unlock()

    if !ok {
        if hub.defaultPeerAddress.Host == "" {
            return EReceiverUnknown
        }

        peerAddress = hub.defaultPeerAddress
    }

    endpointURL := peerAddress.ToHTTPURL("/raftmessages")

    if proxy {
        endpointURL += "?forwarded=true"
    }

    request, err := http.NewRequest("POST", endpointURL, bytes.NewReader(encodedMessage))

    if err != nil {
        return err
    }

    resp, err := hub.httpClient.Do(request)
    
    if err != nil {
        if strings.Contains(err.Error(), "Timeout") {
            return ETimeout
        }

        return err
    }
    
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        errorMessage, err := ioutil.ReadAll(resp.Body)

        if resp.StatusCode == http.StatusForbidden {
            return ESenderUnknown
        }
        
        if err != nil {
            return err
        }
        
        return errors.New(fmt.Sprintf("Received error code from server: (%d) %s", resp.StatusCode, string(errorMessage)))
    } else {
        io.Copy(ioutil.Discard, resp.Body)
    }

    return nil
}

func (hub *TransportHub) Attach(router *mux.Router) {
    router.HandleFunc("/raftmessages", func(w http.ResponseWriter, r *http.Request) {
        raftMessage, err := ioutil.ReadAll(r.Body)

        if err != nil {
            Log.Warningf("POST /raftmessages: Unable to read message body")

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")
            
            return
        }

        var msg raftpb.Message

        err = msg.Unmarshal(raftMessage)

        if err != nil {
            Log.Warningf("POST /raftmessages: Unable to parse message body")

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, "\n")
            
            return
        }

        if msg.To != hub.localPeerID {
            Log.Infof("This message isn't bound for us. Need to forward")
            // This node is not bound for us. Forward it to its proper destination if we know it
            // This feature allows new nodes to use their seed node to send messages throughout the cluster before it knows the addresses of its neighboring nodes
            query := r.URL.Query()
            _, wasForwarded := query["forwarded"]

            // This message was already proxied by the sender node. The message can only travel for
            // one hop so we will return an error since we are not the node this is bound for
            if wasForwarded {
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusForbidden)
                io.WriteString(w, "\n")

                return
            }

            Log.Debugf("POST /raftmessages: Destination node (%d) for message is not this node. Will attempt to proxy the message to its proper recipient", msg.To)
            
            err := hub.Send(r.Context(), msg, true)

            if err != nil {
                Log.Warningf("POST /raftmessages: Unable to proxy message to node (%d): %v", msg.To, err.Error())
                
                w.Header().Set("Content-Type", "application/json; charset=utf8")
                w.WriteHeader(http.StatusInternalServerError)
                io.WriteString(w, "\n")
                
                return
            }
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusOK)
            io.WriteString(w, "\n")
            
            return
        }

        err = hub.onReceiveCB(r.Context(), msg)

        if err != nil {
            Log.Warningf("POST /raftmessages: Unable to receive message: %v", err.Error())

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")
            
            return
        }

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, "\n")
    }).Methods("POST")
}