package raft_test

import (
    . "github.com/armPelionEdge/devicedb/raft"
    "github.com/coreos/etcd/raft/raftpb"
    "github.com/gorilla/mux"
    "net"
    "net/http"
    "strconv"
    "time"
    "context"
    "errors"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var (
    PortIndex = 1
    PortBase = 9000
    SenderNodeID = uint64(1)
    ReceiverNodeID = uint64(2)
    ThirdNodeID = uint64(3)
)

type TestHTTPServer struct {
    port int
    r *mux.Router
    httpServer *http.Server
    listener net.Listener
    done chan int
}

func NewTestHTTPServer(port int) *TestHTTPServer {
    httpServer := &TestHTTPServer{
        port: port,
        done: make(chan int),
        r: mux.NewRouter(),
    }

    return httpServer
}

func (s *TestHTTPServer) Start() error {
    s.httpServer = &http.Server{
        Handler: s.r,
        WriteTimeout: 15 * time.Second,
        ReadTimeout: 15 * time.Second,
    }
    
    var listener net.Listener
    var err error

    listener, err = net.Listen("tcp", "localhost:" + strconv.Itoa(s.port))
    
    if err != nil {
        return err
    }
    
    s.listener = listener

    go func() {
        s.httpServer.Serve(s.listener)
        s.done <- 1
    }()

    return nil
}

func (s *TestHTTPServer) Stop() {
    s.listener.Close()
    <-s.done
}

func (s *TestHTTPServer) Router() *mux.Router {
    return s.r
}

var _ = Describe("Transport", func() {
    var ReceiverPort int
    var SenderPort int
    var receiverServer *TestHTTPServer
    var senderServer *TestHTTPServer
    var thirdNodeServer *TestHTTPServer
    var sender *TransportHub
    var receiver *TransportHub
    var thirdNode *TransportHub

    BeforeSuite(func() {
        thirdNodeServer = NewTestHTTPServer(PortBase - 1)
        thirdNode = NewTransportHub(ThirdNodeID)

        thirdNode.OnReceive(func(ctx context.Context, msg raftpb.Message) error {
            return nil
        })

        thirdNode.Attach(thirdNodeServer.Router())
        thirdNodeServer.Start()
        <-time.After(time.Millisecond * 200)
    })

    BeforeEach(func() {
        PortIndex += 1
        ReceiverPort = PortBase + 2*PortIndex
        SenderPort = PortBase + 2*PortIndex + 1

        receiverServer = NewTestHTTPServer(ReceiverPort)
        senderServer = NewTestHTTPServer(SenderPort)
        sender = NewTransportHub(SenderNodeID)
        receiver = NewTransportHub(ReceiverNodeID)

        receiver.OnReceive(func(ctx context.Context, msg raftpb.Message) error {
            return nil
        })
        
        sender.OnReceive(func(ctx context.Context, msg raftpb.Message) error {
            return nil
        })

        receiver.Attach(receiverServer.Router())
        sender.Attach(senderServer.Router())

        senderServer.Start()
        receiverServer.Start()
        <-time.After(time.Millisecond * 200)
    })

    AfterEach(func() {
        receiverServer.Stop()
        senderServer.Stop()
    })

    Describe("Sending a message", func() {
        Context("The recipient is not known by the sender", func() {
            Context("The default route is not set", func() {
                It("Send should result in an EReceiverUnknown error", func() {
                    Expect(sender.Send(context.TODO(), raftpb.Message{
                        From: SenderNodeID,
                        To: ReceiverNodeID,
                    }, false)).Should(Equal(EReceiverUnknown))
                })
            })

            Context("The default route is set", func() {
                Context("The default route node is not reachable", func() {
                    Specify("Send should return an error", func() {
                        // Nothing should ever have been set up on PortBase
                        sender.SetDefaultRoute("localhost", PortBase)

                        Expect(sender.Send(context.TODO(), raftpb.Message{
                            From: SenderNodeID,
                            To: ReceiverNodeID,
                        }, false)).Should(Not(BeNil()))
                    })
                })

                Context("The default route node is reachable", func() {
                    BeforeEach(func() {
                        sender.SetDefaultRoute("localhost", ReceiverPort)
                    })

                    Context("The default route node knows the recipient", func() {
                        BeforeEach(func() {
                            receiver.AddPeer(PeerAddress{
                                NodeID: ThirdNodeID,
                                Host: "localhost",
                                Port: PortBase - 1,
                            })
                        })

                        Specify("The default route node should forward the message to the recipient", func() {
                            var receivedMessageOnThirdNodeServer bool

                            thirdNode.OnReceive(func(ctx context.Context, msg raftpb.Message) error {
                                receivedMessageOnThirdNodeServer = true

                                return nil
                            })

                            Expect(sender.Send(context.TODO(), raftpb.Message{
                                From: SenderNodeID,
                                To: ThirdNodeID,
                            }, false)).Should(BeNil())

                            Expect(receivedMessageOnThirdNodeServer).Should(BeTrue())
                        })
                    })

                    Context("The default route node does not know the recipient", func() {
                        Specify("Send should return an error", func() {
                            var receivedMessageOnThirdNodeServer bool

                            thirdNode.OnReceive(func(ctx context.Context, msg raftpb.Message) error {
                                receivedMessageOnThirdNodeServer = true

                                return nil
                            })

                            Expect(sender.Send(context.TODO(), raftpb.Message{
                                From: SenderNodeID,
                                To: ThirdNodeID,
                            }, false)).Should(Not(BeNil()))

                            Expect(receivedMessageOnThirdNodeServer).Should(BeFalse())
                        })
                    })
                })
            })
        })

        Context("The recipient is known by the sender", func() {
            BeforeEach(func() {
                sender.AddPeer(PeerAddress{
                    NodeID: ReceiverNodeID,
                    Host: "localhost",
                    Port: ReceiverPort,
                })
            })

            Context("The sender is known by the recipient", func() {
                BeforeEach(func() {
                    receiver.AddPeer(PeerAddress{
                        NodeID: SenderNodeID,
                        Host: "localhost",
                        Port: SenderPort,
                    })
                })

                Specify("Send should return nil once the receiver has processed the message and responded with an acknowledgment", func() {
                    receiver.OnReceive(func(ctx context.Context, msg raftpb.Message) error {
                        return nil
                    })

                    Expect(sender.Send(context.TODO(), raftpb.Message{
                        From: SenderNodeID,
                        To: ReceiverNodeID,
                    }, false)).Should(BeNil())
                })

                Specify("Send should return an error if the receiver encountered an error processing the message and responded with an error code", func() {
                    receiver.OnReceive(func(ctx context.Context, msg raftpb.Message) error {
                        return errors.New("Something bad happened")
                    })

                    Expect(sender.Send(context.TODO(), raftpb.Message{
                        From: SenderNodeID,
                        To: ReceiverNodeID,
                    }, false)).Should(Not(BeNil()))
                })

                Specify("Send shoult time out and return an ETimeout error if the receiver does not respond within the timeout window", func() {
                    receiver.OnReceive(func(ctx context.Context, msg raftpb.Message) error {
                        <-time.After(time.Second * RequestTimeoutSeconds + time.Second * 2)

                        return nil
                    })

                    Expect(sender.Send(context.TODO(), raftpb.Message{
                        From: SenderNodeID,
                        To: ReceiverNodeID,
                    }, false)).Should(Equal(ETimeout))
                })
            })
        })
    })
})
