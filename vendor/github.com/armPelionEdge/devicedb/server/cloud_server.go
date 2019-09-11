package server

import (
    "crypto/tls"
    "net"
    "net/http"
    "time"
    "strconv"
    "github.com/gorilla/mux"
    "sync"

    . "github.com/armPelionEdge/devicedb/logging"
)

type CloudServerConfig struct {
    NodeID uint64
    ExternalPort int
    ExternalHost string
    InternalPort int
    InternalHost string
    RelayTLSConfig *tls.Config
}

type CloudServer struct {
    httpServer *http.Server
    relayHTTPServer *http.Server
    listener net.Listener
    relayListener net.Listener
    seedPort int
    seedHost string
    externalPort int
    externalHost string
    internalPort int
    internalHost string
    relayTLSConfig *tls.Config
    router *mux.Router
    stop chan int
    nodeID uint64
}

func NewCloudServer(serverConfig CloudServerConfig) *CloudServer {
    server := &CloudServer{
        externalHost: serverConfig.ExternalHost,
        externalPort: serverConfig.ExternalPort,
        internalHost: serverConfig.InternalHost,
        internalPort: serverConfig.InternalPort,
        relayTLSConfig: serverConfig.RelayTLSConfig,
        nodeID: serverConfig.NodeID,
        router: mux.NewRouter(),
    }

    return server
}

func (server *CloudServer) ExternalPort() int {
    return server.externalPort
}

func (server *CloudServer) ExternalHost() string {
    return server.externalHost
}

func (server *CloudServer) InternalPort() int {
    return server.internalPort
}

func (server *CloudServer) InternalHost() string {
    return server.internalHost
}

func (server *CloudServer) Router() *mux.Router {
    return server.router
}

func (server *CloudServer) IsHTTPOnly() bool {
    return server.externalHost == ""
}

func (server *CloudServer) Start() error {
    server.stop = make(chan int)

    server.httpServer = &http.Server{
        Handler: server.router,
        WriteTimeout: 45 * time.Second,
        ReadTimeout: 45 * time.Second,
    }

    server.relayHTTPServer = &http.Server{
        Handler: server.router,
        WriteTimeout: 15 * time.Second,
        ReadTimeout: 15 * time.Second,
    }
    
    var listener net.Listener
    var relayListener net.Listener
    var err error

    listener, err = net.Listen("tcp", server.InternalHost() + ":" + strconv.Itoa(server.InternalPort()))

    if err != nil {
        Log.Errorf("Error listening on port %d: %v", server.InternalPort(), err.Error())
        
        server.Stop()
        
        return err
    }

    server.listener = listener

    if !server.IsHTTPOnly() {
        relayListener, err = tls.Listen("tcp", server.ExternalHost() + ":" + strconv.Itoa(server.ExternalPort()), server.relayTLSConfig)

        if err != nil {
            Log.Errorf("Error setting up relay listener on port %d: %v", server.ExternalPort(), err.Error())
            
            server.Stop()
            
            return err
        }
        
        server.relayListener = relayListener

        Log.Infof("Listening external (%s:%d), internal (%s:%d)", server.ExternalHost(), server.ExternalPort(), server.InternalHost(), server.InternalPort())
    } else {
        Log.Infof("Listening (HTTP Only) (%s:%d)", server.InternalHost(), server.InternalPort())
    }

    var wg sync.WaitGroup
    wg.Add(2)

    go func() {
        err = server.httpServer.Serve(server.listener)
        server.Stop() // to ensure all other listeners shutdown
        wg.Done()
    }()

    if !server.IsHTTPOnly() {
        go func() {
            err = server.relayHTTPServer.Serve(server.relayListener)
            server.Stop() // to ensure all other listeners shutdown
            wg.Done()
        }()
    }

    wg.Wait()

    Log.Errorf("Server shutting down. Reason: %v", err)

    return err
}

func (server *CloudServer) Stop() {
    if server.listener != nil {
        server.listener.Close()
    }

    if server.relayListener != nil {
        server.relayListener.Close()
    }
}
