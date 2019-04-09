package relaymq

import (
    "net"
    "net/http"
    "time"
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "net/http/pprof"
    "strconv"
    "encoding/json"
    "io"
	"fmt"
)

type MessageBody struct {
	Message string `json:"message"`
}

type ServerConfig struct {
    Port int
    AmqpConfig AMQPConfig
}

func (sc *ServerConfig) LoadFromFile(file string) error {
    var ysc YAMLServerConfig
    
    err := ysc.LoadFromFile(file)
    
    if err != nil {
        return err
    }
    
    sc.Port = ysc.Port
    sc.AmqpConfig.Channels = ysc.Broker.Channels
    sc.AmqpConfig.Host = ysc.Broker.Host
    sc.AmqpConfig.Port = ysc.Broker.Port
    sc.AmqpConfig.Username = ysc.Broker.Username
    sc.AmqpConfig.Password = ysc.Broker.Password
    sc.AmqpConfig.PrefetchLimit = ysc.Broker.PrefetchLimit
    
    return nil
}

type Server struct {
    httpServer *http.Server
    listener net.Listener
    serverConfig ServerConfig
    upgrader websocket.Upgrader
    hub *Hub
    queueManager *AMQPQueueManager    
}

func NewServer(serverConfig ServerConfig) (*Server, error) {
    upgrader := websocket.Upgrader{
    	ReadBufferSize:  1024,
    	WriteBufferSize: 1024,
    }

    queueManager := NewAMQPQueueManager(&serverConfig.AmqpConfig)
    server := &Server{ nil, nil, serverConfig, upgrader, NewHub(queueManager), queueManager }

    return server, nil
}

func (server *Server) Port() int {
    return server.serverConfig.Port
}

func (server *Server) Start() error {
    r := mux.NewRouter()
    
    r.HandleFunc("/queues/{queue}", func(w http.ResponseWriter, r *http.Request) {
        queueName := mux.Vars(r)["queue"]

        var messageBody MessageBody
        decoder := json.NewDecoder(r.Body)
        err := decoder.Decode(&messageBody)

        if err != nil {
            log.Warningf("POST /queues/{queue}: %v", err.Error())
            
            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, "\n")
            
            return
        }

        var identity IdentityHeader
        err = identity.FromHeader(r.Header)

        if err != nil {
            log.Warningf("Client provided an invalid identity header: %s", err.Error())

            return
        }
        
        fullQueueName := fmt.Sprintf("%s-%s-%s", identity.AccountID, identity.RelayID, queueName)
        err = server.queueManager.Publish(fullQueueName, messageBody.Message)

        if err != nil {
            log.Warningf("POST /queues/{queue}: %v", err.Error())

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")

            return
        }

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        io.WriteString(w, "\n")
    }).Methods("POST")

    r.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
        conn, err := server.upgrader.Upgrade(w, r, nil)
        
        if err != nil {
            return
        }

        var identity IdentityHeader
        err = identity.FromHeader(r.Header)

        if err != nil {
            log.Warningf("Client provided an invalid identity header: %s", err.Error())

            return
        }

        server.hub.Accept(identity.AccountID, identity.RelayID, conn)
    }).Methods("GET")
    
    r.HandleFunc("/debug/pprof/", pprof.Index)
    r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
    r.HandleFunc("/debug/pprof/profile", pprof.Profile)
    r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
    
    server.httpServer = &http.Server{
        Handler: r,
        WriteTimeout: 15 * time.Second,
        ReadTimeout: 15 * time.Second,
    }
    
    var listener net.Listener
    var err error

    listener, err = net.Listen("tcp", "0.0.0.0:" + strconv.Itoa(server.Port()))
    
    if err != nil {
        log.Errorf("Error listening on port %d: %s", server.Port(), err.Error())
        
        return err
    }
    
    server.listener = listener

    log.Infof("Listening on port %d", server.Port())
    server.queueManager.Start()

    err = server.httpServer.Serve(server.listener)

    log.Errorf("Server shutting down. Reason: %v", err)

    return err
}

func (server *Server) Stop() error {
    if server.listener != nil {
        server.listener.Close()
    }

    server.queueManager.Stop()
    
    return nil
}
