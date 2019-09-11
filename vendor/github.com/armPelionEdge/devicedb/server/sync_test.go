package server_test

import (
    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/merkle"
    . "github.com/armPelionEdge/devicedb/server"
    . "github.com/armPelionEdge/devicedb/util"
    ddbSync "github.com/armPelionEdge/devicedb/sync"
    
    "time"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

const MERKLE_EXPLORATION_PATH_LIMIT = 100

var _ = Describe("Sync", func() {
    Describe("Integration", func() {
        Context("Equal merkle depths", func() {
            var server1 *Server
            var server2 *Server
            var server1BucketProxy ddbSync.BucketProxy
            var server2BucketProxy ddbSync.BucketProxy
            stop1 := make(chan int)
            stop2 := make(chan int)
            
            BeforeEach(func() {
                server1, _ = NewServer(ServerConfig{
                    DBFile: "/tmp/testdb-" + RandomString(),
                    Port: 8080,
                    NodeID: "nodeA",
                })
                server2, _ = NewServer(ServerConfig{
                    DBFile: "/tmp/testdb-" + RandomString(),
                    Port: 9090,
                    NodeID: "nodeB",
                })
                
                go func() {
                    server1.Start()
                    stop1 <- 1
                }()
                
                go func() {
                    server2.Start()
                    stop2 <- 1
                }()
                
                time.Sleep(time.Millisecond * 200)

                server1BucketProxy = &ddbSync.RelayBucketProxy{ Bucket: server1.Buckets().Get("default") }
                server2BucketProxy = &ddbSync.RelayBucketProxy{ Bucket: server2.Buckets().Get("default") }
            })
            
            AfterEach(func() {
                server1.Stop()
                server2.Stop()
                <-stop1
                <-stop2
            })
            
            Context("Both empty", func() {
                It("should result in both terminating before any hash traversal happens beyond the root", func() {
                    var message *SyncMessageWrapper = nil
                    direction := 0
                    
                    initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                    responderSyncSession := NewResponderSyncSession(server2BucketProxy)
                    
                    initiatorStateTransitions := []int{ START, HANDSHAKE, ROOT_HASH_COMPARE }
                    responderStateTransitions := []int{ START, HASH_COMPARE, HASH_COMPARE }
                
                    for initiatorSyncSession.State() != END || responderSyncSession.State() != END {
                        if direction == 0 {
                            Expect(initiatorStateTransitions[0]).Should(Equal(initiatorSyncSession.State()))
                            message = initiatorSyncSession.NextState(message)
                            direction = 1
                            initiatorStateTransitions = initiatorStateTransitions[1:]
                        } else {
                            Expect(responderStateTransitions[0]).Should(Equal(responderSyncSession.State()))
                            message = responderSyncSession.NextState(message)
                            direction = 0
                            responderStateTransitions = responderStateTransitions[1:]
                        }
                    }
                    
                    Expect(initiatorStateTransitions).Should(Equal([]int{ }))
                    Expect(responderStateTransitions).Should(Equal([]int{ }))
                })
            })
            
            Context("One empty the other has an object", func() {
                It("should result in the initiator receiving the object it doesn't have", func() {
                    // write a value to key "OBJ1" at the responder
                    updateBatch := NewUpdateBatch()
                    updateBatch.Put([]byte("OBJ1"), []byte("hello"), NewDVV(NewDot("", 0), map[string]uint64{ }))
                    _, err := server2.Buckets().Get("default").Batch(updateBatch)
                    
                    Expect(err).Should(BeNil())
                    
                    var message *SyncMessageWrapper = nil
                    direction := 0
                    
                    initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                    responderSyncSession := NewResponderSyncSession(server2BucketProxy)
                    
                    for initiatorSyncSession.State() != END || responderSyncSession.State() != END {
                        if direction == 0 {
                            message = initiatorSyncSession.NextState(message)
                            direction = 1
                        } else {
                            message = responderSyncSession.NextState(message)
                            direction = 0
                        }
                    }
                    
                    siblingSets, err := server1.Buckets().Get("default").Get([][]byte{ []byte("OBJ1") })
                    
                    Expect(err).Should(BeNil())
                    Expect(len(siblingSets)).Should(Equal(1))
                    Expect(siblingSets[0].Value()).Should(Equal([]byte("hello")))
                    
                    for i := uint32(1); i < server1.Buckets().Get("default").MerkleTree().NodeLimit(); i += 1 {
                        v1 := server1.Buckets().Get("default").MerkleTree().NodeHash(i)
                        v2 := server2.Buckets().Get("default").MerkleTree().NodeHash(i)
                        
                        Expect(v1).Should(Equal(v2))
                    }
                    
                    Expect(server1.Buckets().Get("default").MerkleTree().RootHash()).Should(Not(Equal(NewHash([]byte{ }).SetLow(0).SetHigh(0))))
                })
            })
            
            Context("Both have objects", func() {
                populate := func(bucket Bucket, count int) []string {
                    keys := make([]string, count)
                    
                    for i := 0; i < count; i += 1 {
                        keys[i] = "key" + RandomString()
                        
                        updateBatch := NewUpdateBatch()
                        updateBatch.Put([]byte(keys[i]), []byte(RandomString()), NewDVV(NewDot("", 0), map[string]uint64{ }))
                        bucket.Batch(updateBatch)
                    }
                    
                    return keys
                }
                
                sync := func(initiator ddbSync.BucketProxy, responder ddbSync.BucketProxy) {
                    var message *SyncMessageWrapper = nil
                    direction := 0
                    
                    initiatorSyncSession := NewInitiatorSyncSession(123, initiator, MERKLE_EXPLORATION_PATH_LIMIT, true)
                    responderSyncSession := NewResponderSyncSession(responder)
                    
                    for initiatorSyncSession.State() != END || responderSyncSession.State() != END {
                        if direction == 0 {
                            message = initiatorSyncSession.NextState(message)
                            direction = 1
                        } else {
                            message = responderSyncSession.NextState(message)
                            direction = 0
                        }
                    }
                }
                
                get := func(b Bucket, key string) string {
                    siblingSets, _ := b.Get([][]byte{ []byte(key) })
                    
                    return string(siblingSets[0].Value())
                }
                
                It("should result in both having the same objects after several sync iterations", func() {
                    keys1 := populate(server1.Buckets().Get("default"), 100)
                    keys2 := populate(server2.Buckets().Get("default"), 100)
                    
                    for i := 0; i < 2*len(keys1); i += 1 {
                        sync(server1BucketProxy, server2BucketProxy)
                        sync(server2BucketProxy, server1BucketProxy)
                    }
                    
                    for _, k := range keys1 {
                        Expect(get(server1.Buckets().Get("default"), k)).Should(Equal(get(server2.Buckets().Get("default"), k)))
                    }
                    
                    for _, k := range keys2 {
                        Expect(get(server1.Buckets().Get("default"), k)).Should(Equal(get(server2.Buckets().Get("default"), k)))
                    }
                    
                    for i := uint32(1); i < server1.Buckets().Get("default").MerkleTree().NodeLimit(); i += 1 {
                        v1 := server1.Buckets().Get("default").MerkleTree().NodeHash(i)
                        v2 := server2.Buckets().Get("default").MerkleTree().NodeHash(i)
                        
                        Expect(v1).Should(Equal(v2))
                    }
                })
            })
        })
            
        Context("Initiator has a smaller merkle depth", func() {
            var server1 *Server
            var server2 *Server
            var server1BucketProxy ddbSync.BucketProxy
            var server2BucketProxy ddbSync.BucketProxy
            stop1 := make(chan int)
            stop2 := make(chan int)
            
            BeforeEach(func() {
                server1, _ = NewServer(ServerConfig{
                    DBFile: "/tmp/testdb-" + RandomString(),
                    Port: 8080,
                    MerkleDepth: MerkleMinDepth,
                })
                
                server2, _ = NewServer(ServerConfig{
                    DBFile: "/tmp/testdb-" + RandomString(),
                    Port: 9090,
                    MerkleDepth: MerkleDefaultDepth,
                })
                
                go func() {
                    server1.Start()
                    stop1 <- 1
                }()
                
                go func() {
                    server2.Start()
                    stop2 <- 1
                }()
                
                time.Sleep(time.Millisecond * 200)

                server1BucketProxy = &ddbSync.RelayBucketProxy{ Bucket: server1.Buckets().Get("default") }
                server2BucketProxy = &ddbSync.RelayBucketProxy{ Bucket: server2.Buckets().Get("default") }
            })
            
            AfterEach(func() {
                server1.Stop()
                server2.Stop()
                <-stop1
                <-stop2
            })
            
            It("should result in the initiator receiving the object it doesn't have", func() {
                // write a value to key "OBJ1" at the responder
                updateBatch := NewUpdateBatch()
                updateBatch.Put([]byte("OBJ1"), []byte("hello"), NewDVV(NewDot("", 0), map[string]uint64{ }))
                _, err := server2.Buckets().Get("default").Batch(updateBatch)
                
                Expect(err).Should(BeNil())
                
                var message *SyncMessageWrapper = nil
                direction := 0
                
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                responderSyncSession := NewResponderSyncSession(server2BucketProxy)
                
                for initiatorSyncSession.State() != END || responderSyncSession.State() != END {
                    if direction == 0 {
                        message = initiatorSyncSession.NextState(message)
                        direction = 1
                    } else {
                        message = responderSyncSession.NextState(message)
                        direction = 0
                    }
                }
                
                siblingSets, err := server1.Buckets().Get("default").Get([][]byte{ []byte("OBJ1") })
                
                Expect(err).Should(BeNil())
                Expect(len(siblingSets)).Should(Equal(1))
                Expect(siblingSets[0].Value()).Should(Equal([]byte("hello")))
                
                for i := uint32(1); i < server1.Buckets().Get("default").MerkleTree().NodeLimit(); i += 1 {
                    v1 := server1.Buckets().Get("default").MerkleTree().NodeHash(i)
                    v2 := server2.Buckets().Get("default").MerkleTree().NodeHash(server1.Buckets().Get("default").MerkleTree().TranslateNode(i, MerkleDefaultDepth))
                    
                    Expect(v1).Should(Equal(v2))
                }
                
                Expect(server1.Buckets().Get("default").MerkleTree().RootHash()).Should(Not(Equal(NewHash([]byte{ }).SetLow(0).SetHigh(0))))
            })
        })
        
        Context("Initiator has a larger merkle depth", func() {
            var server1 *Server
            var server2 *Server
            var server1BucketProxy ddbSync.BucketProxy
            var server2BucketProxy ddbSync.BucketProxy
            stop1 := make(chan int)
            stop2 := make(chan int)
            
            BeforeEach(func() {
                server1, _ = NewServer(ServerConfig{
                    DBFile: "/tmp/testdb-" + RandomString(),
                    Port: 8080,
                    MerkleDepth: MerkleDefaultDepth,
                })
                
                server2, _ = NewServer(ServerConfig{
                    DBFile: "/tmp/testdb-" + RandomString(),
                    Port: 9090,
                    MerkleDepth: MerkleMinDepth,
                })
                
                go func() {
                    server1.Start()
                    stop1 <- 1
                }()
                
                go func() {
                    server2.Start()
                    stop2 <- 1
                }()
                
                time.Sleep(time.Millisecond * 200)

                server1BucketProxy = &ddbSync.RelayBucketProxy{ Bucket: server1.Buckets().Get("default") }
                server2BucketProxy = &ddbSync.RelayBucketProxy{ Bucket: server2.Buckets().Get("default") }
            })
            
            AfterEach(func() {
                server1.Stop()
                server2.Stop()
                <-stop1
                <-stop2
            })
            
            It("should result in the initiator receiving the object it doesn't have", func() {
                // write a value to key "OBJ1" at the responder
                updateBatch := NewUpdateBatch()
                updateBatch.Put([]byte("OBJ1"), []byte("hello"), NewDVV(NewDot("", 0), map[string]uint64{ }))
                _, err := server2.Buckets().Get("default").Batch(updateBatch)
                
                Expect(err).Should(BeNil())
                
                var message *SyncMessageWrapper = nil
                direction := 0
                
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                responderSyncSession := NewResponderSyncSession(server2BucketProxy)
                
                for initiatorSyncSession.State() != END || responderSyncSession.State() != END {
                    if direction == 0 {
                        message = initiatorSyncSession.NextState(message)
                        direction = 1
                    } else {
                        message = responderSyncSession.NextState(message)
                        direction = 0
                    }
                }
                
                siblingSets, err := server1.Buckets().Get("default").Get([][]byte{ []byte("OBJ1") })
                
                Expect(err).Should(BeNil())
                Expect(len(siblingSets)).Should(Equal(1))
                Expect(siblingSets[0].Value()).Should(Equal([]byte("hello")))
                
                for i := uint32(1); i < server2.Buckets().Get("default").MerkleTree().NodeLimit(); i += 1 {
                    v1 := server1.Buckets().Get("default").MerkleTree().NodeHash(server2.Buckets().Get("default").MerkleTree().TranslateNode(i, MerkleDefaultDepth))
                    v2 := server2.Buckets().Get("default").MerkleTree().NodeHash(i)
                    
                    Expect(v1).Should(Equal(v2))
                }
                
                Expect(server1.Buckets().Get("default").MerkleTree().RootHash()).Should(Not(Equal(NewHash([]byte{ }).SetLow(0).SetHigh(0))))
            })
        })
    })
    
    Describe("InitiatorSyncSession", func() {
        Describe("#NextState", func() {
            var server1 *Server
            var server1BucketProxy ddbSync.BucketProxy
            stop1 := make(chan int)
            
            BeforeEach(func() {
                server1, _ = NewServer(ServerConfig{
                    DBFile: "/tmp/testdb-" + RandomString(),
                    Port: 8080,
                })
                
                go func() {
                    server1.Start()
                    stop1 <- 1
                }()
                
                time.Sleep(time.Millisecond * 200)

                server1BucketProxy = &ddbSync.RelayBucketProxy{ Bucket: server1.Buckets().Get("default") }
            })
            
            AfterEach(func() {
                server1.Stop()
                <-stop1
            })
            
            It("START -> HANDSHAKE", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(START)
                
                req := initiatorSyncSession.NextState(nil)
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_START))
                Expect(req.MessageBody.(Start).ProtocolVersion).Should(Equal(PROTOCOL_VERSION))
                Expect(req.MessageBody.(Start).MerkleDepth).Should(Equal(server1.Buckets().Get("default").MerkleTree().Depth()))
                Expect(req.MessageBody.(Start).Bucket).Should(Equal("default"))
                Expect(initiatorSyncSession.State()).Should(Equal(HANDSHAKE))
            })
            
            It("HANDSHAKE -> ROOT_HASH_COMPARE", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(HANDSHAKE)
                
                rootNode := server1.Buckets().Get("default").MerkleTree().RootNode()
                rootHash := server1.Buckets().Get("default").MerkleTree().NodeHash(rootNode)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_START,
                    MessageBody: Start{
                        ProtocolVersion: PROTOCOL_VERSION,
                        MerkleDepth: 50,
                        Bucket: "default",
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_NODE_HASH))
                Expect(req.MessageBody.(MerkleNodeHash).NodeID).Should(Equal(server1.Buckets().Get("default").MerkleTree().TranslateNode(rootNode, 50)))
                Expect(req.MessageBody.(MerkleNodeHash).HashHigh).Should(Equal(rootHash.High()))
                Expect(req.MessageBody.(MerkleNodeHash).HashLow).Should(Equal(rootHash.Low()))
                Expect(initiatorSyncSession.State()).Should(Equal(ROOT_HASH_COMPARE))
                Expect(initiatorSyncSession.ResponderDepth()).Should(Equal(uint8(50)))
                Expect(initiatorSyncSession.PeekExplorationQueue()).Should(Equal(server1.Buckets().Get("default").MerkleTree().RootNode()))
            })
            
            It("HANDSHAKE -> END nil message", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(HANDSHAKE)
                
                req := initiatorSyncSession.NextState(nil)
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(initiatorSyncSession.State()).Should(Equal(END))
                Expect(initiatorSyncSession.ResponderDepth()).Should(Equal(uint8(0)))
            })
            
            It("HANDSHAKE -> END non nil message", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(HANDSHAKE)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_PUSH_MESSAGE,
                    MessageBody: Start{
                        ProtocolVersion: PROTOCOL_VERSION,
                        MerkleDepth: server1.Buckets().Get("default").MerkleTree().Depth(),
                        Bucket: "default",
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(initiatorSyncSession.State()).Should(Equal(END))
                Expect(initiatorSyncSession.ResponderDepth()).Should(Equal(uint8(0)))
            })
            
            It("ROOT_HASH_COMPARE -> END nil message", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(ROOT_HASH_COMPARE)
                
                req := initiatorSyncSession.NextState(nil)
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(initiatorSyncSession.State()).Should(Equal(END))
            })
            
            It("ROOT_HASH_COMPARE -> END non nil message", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(ROOT_HASH_COMPARE)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_PUSH_MESSAGE,
                    MessageBody: Start{
                        ProtocolVersion: PROTOCOL_VERSION,
                        MerkleDepth: server1.Buckets().Get("default").MerkleTree().Depth(),
                        Bucket: "default",
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(initiatorSyncSession.State()).Should(Equal(END))
            })
            
            It("ROOT_HASH_COMPARE -> END root hashes match", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(ROOT_HASH_COMPARE)
                
                rootNode := server1.Buckets().Get("default").MerkleTree().RootNode()
                rootHash := server1.Buckets().Get("default").MerkleTree().NodeHash(rootNode)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_NODE_HASH,
                    MessageBody: MerkleNodeHash{
                        NodeID: rootNode,
                        HashHigh: rootHash.High(),
                        HashLow: rootHash.Low(),
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(initiatorSyncSession.State()).Should(Equal(END))
            })
            
            It("ROOT_HASH_COMPARE -> LEFT_HASH_COMPARE", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(ROOT_HASH_COMPARE)
                initiatorSyncSession.PushExplorationQueue(server1.Buckets().Get("default").MerkleTree().RootNode())
                initiatorSyncSession.SetResponderDepth(20)
                
                rootNode := server1.Buckets().Get("default").MerkleTree().RootNode()
                rootNodeLeftChild := server1.Buckets().Get("default").MerkleTree().LeftChild(rootNode)
                rootHash := server1.Buckets().Get("default").MerkleTree().NodeHash(rootNode)
                rootNodeLeftChildHash := server1.Buckets().Get("default").MerkleTree().NodeHash(rootNodeLeftChild)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_NODE_HASH,
                    MessageBody: MerkleNodeHash{
                        NodeID: rootNode,
                        HashHigh: rootHash.High() + 1,
                        HashLow: rootHash.Low(),
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_NODE_HASH))
                Expect(req.MessageBody.(MerkleNodeHash).NodeID).Should(Equal(server1.Buckets().Get("default").MerkleTree().TranslateNode(rootNodeLeftChild, 20)))
                Expect(req.MessageBody.(MerkleNodeHash).HashHigh).Should(Equal(rootNodeLeftChildHash.High()))
                Expect(req.MessageBody.(MerkleNodeHash).HashLow).Should(Equal(rootNodeLeftChildHash.Low()))
                Expect(initiatorSyncSession.State()).Should(Equal(LEFT_HASH_COMPARE))
            })
            
            It("ROOT_HASH_COMPARE -> DB_OBJECT_PUSH", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(ROOT_HASH_COMPARE)
                initiatorSyncSession.PushExplorationQueue(server1.Buckets().Get("default").MerkleTree().RootNode())
                initiatorSyncSession.SetResponderDepth(1)
                
                rootNode := server1.Buckets().Get("default").MerkleTree().RootNode()
                rootHash := server1.Buckets().Get("default").MerkleTree().NodeHash(rootNode)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_NODE_HASH,
                    MessageBody: MerkleNodeHash{
                        NodeID: rootNode,
                        HashHigh: rootHash.High() + 1,
                        HashLow: rootHash.Low(),
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_OBJECT_NEXT))
                Expect(req.MessageBody.(ObjectNext).NodeID).Should(Equal(server1.Buckets().Get("default").MerkleTree().TranslateNode(rootNode, 1)))
                Expect(initiatorSyncSession.State()).Should(Equal(DB_OBJECT_PUSH))
            })
            
            It("LEFT_HASH_COMPARE -> END nil message", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(LEFT_HASH_COMPARE)
                
                req := initiatorSyncSession.NextState(nil)
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(initiatorSyncSession.State()).Should(Equal(END))
            })
            
            It("LEFT_HASH_COMPARE -> END non nil message", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(LEFT_HASH_COMPARE)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_PUSH_MESSAGE,
                    MessageBody: Start{
                        ProtocolVersion: PROTOCOL_VERSION,
                        MerkleDepth: server1.Buckets().Get("default").MerkleTree().Depth(),
                        Bucket: "default",
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(initiatorSyncSession.State()).Should(Equal(END))
            })
            
            It("LEFT_HASH_COMPARE -> RIGHT_HASH_COMPARE add left hash to queue", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                rootNode := server1.Buckets().Get("default").MerkleTree().RootNode()
                rootNodeLeftChild := server1.Buckets().Get("default").MerkleTree().LeftChild(rootNode)
                rootNodeRightChild := server1.Buckets().Get("default").MerkleTree().RightChild(rootNode)
                rootLeftChildHash := server1.Buckets().Get("default").MerkleTree().NodeHash(rootNodeLeftChild)
                
                initiatorSyncSession.SetState(LEFT_HASH_COMPARE)
                initiatorSyncSession.PushExplorationQueue(server1.Buckets().Get("default").MerkleTree().RootNode())
                initiatorSyncSession.SetResponderDepth(4)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_NODE_HASH,
                    MessageBody: MerkleNodeHash{
                        NodeID: rootNodeLeftChild,
                        HashHigh: rootLeftChildHash.High() + 1,
                        HashLow: rootLeftChildHash.Low(),
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_NODE_HASH))
                Expect(req.MessageBody.(MerkleNodeHash).NodeID).Should(Equal(server1.Buckets().Get("default").MerkleTree().TranslateNode(rootNodeRightChild, 4)))
                Expect(initiatorSyncSession.State()).Should(Equal(RIGHT_HASH_COMPARE))
                Expect(initiatorSyncSession.ExplorationQueueSize()).Should(Equal(uint32(2)))
                Expect(initiatorSyncSession.PopExplorationQueue()).Should(Equal(rootNode))
                Expect(initiatorSyncSession.PopExplorationQueue()).Should(Equal(rootNodeLeftChild))
            })
            
            It("LEFT_HASH_COMPARE -> RIGHT_HASH_COMPARE ignore left hash", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                rootNode := server1.Buckets().Get("default").MerkleTree().RootNode()
                rootNodeLeftChild := server1.Buckets().Get("default").MerkleTree().LeftChild(rootNode)
                rootNodeRightChild := server1.Buckets().Get("default").MerkleTree().RightChild(rootNode)
                rootLeftChildHash := server1.Buckets().Get("default").MerkleTree().NodeHash(rootNodeLeftChild)
                
                initiatorSyncSession.SetState(LEFT_HASH_COMPARE)
                initiatorSyncSession.PushExplorationQueue(server1.Buckets().Get("default").MerkleTree().RootNode())
                initiatorSyncSession.SetResponderDepth(4)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_NODE_HASH,
                    MessageBody: MerkleNodeHash{
                        NodeID: rootNodeLeftChild,
                        HashHigh: rootLeftChildHash.High(),
                        HashLow: rootLeftChildHash.Low(),
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_NODE_HASH))
                Expect(req.MessageBody.(MerkleNodeHash).NodeID).Should(Equal(server1.Buckets().Get("default").MerkleTree().TranslateNode(rootNodeRightChild, 4)))
                Expect(initiatorSyncSession.State()).Should(Equal(RIGHT_HASH_COMPARE))
                Expect(initiatorSyncSession.ExplorationQueueSize()).Should(Equal(uint32(1)))
                Expect(initiatorSyncSession.PopExplorationQueue()).Should(Equal(rootNode))
                Expect(initiatorSyncSession.PopExplorationQueue()).Should(Equal(uint32(0)))
            })
            
            It("RIGHT_HASH_COMPARE -> END nil message", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(RIGHT_HASH_COMPARE)
                
                req := initiatorSyncSession.NextState(nil)
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(initiatorSyncSession.State()).Should(Equal(END))
            })
            
            It("RIGHT_HASH_COMPARE -> END non nil message", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(RIGHT_HASH_COMPARE)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_PUSH_MESSAGE,
                    MessageBody: Start{
                        ProtocolVersion: PROTOCOL_VERSION,
                        MerkleDepth: server1.Buckets().Get("default").MerkleTree().Depth(),
                        Bucket: "default",
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(initiatorSyncSession.State()).Should(Equal(END))
            })

            It("RIGHT_HASH_COMPARE -> END empty exploration queue", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)

                rootNode := server1.Buckets().Get("default").MerkleTree().RootNode()
                rootNodeRightChild := server1.Buckets().Get("default").MerkleTree().RightChild(rootNode)
                rootRightChildHash := server1.Buckets().Get("default").MerkleTree().NodeHash(rootNodeRightChild)
                
                initiatorSyncSession.SetState(RIGHT_HASH_COMPARE)
                initiatorSyncSession.PushExplorationQueue(server1.Buckets().Get("default").MerkleTree().RootNode())
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_NODE_HASH,
                    MessageBody: MerkleNodeHash{
                        NodeID: rootNodeRightChild,
                        HashHigh: rootRightChildHash.High(),
                        HashLow: rootRightChildHash.Low(),
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(initiatorSyncSession.State()).Should(Equal(END))
            })
            
            It("RIGHT_HASH_COMPARE -> LEFT_HASH_COMPARE add right hash to queue", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                rootNode := server1.Buckets().Get("default").MerkleTree().RootNode()
                rootNodeLeftChild := server1.Buckets().Get("default").MerkleTree().LeftChild(rootNode)
                rootNodeLeftLeftChild := server1.Buckets().Get("default").MerkleTree().LeftChild(rootNodeLeftChild)
                rootNodeRightChild := server1.Buckets().Get("default").MerkleTree().RightChild(rootNode)
                rootRightChildHash := server1.Buckets().Get("default").MerkleTree().NodeHash(rootNodeRightChild)
                
                initiatorSyncSession.SetState(RIGHT_HASH_COMPARE)
                initiatorSyncSession.PushExplorationQueue(rootNode)
                initiatorSyncSession.PushExplorationQueue(rootNodeLeftChild)
                initiatorSyncSession.SetResponderDepth(3)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_NODE_HASH,
                    MessageBody: MerkleNodeHash{
                        NodeID: rootNodeRightChild,
                        HashHigh: rootRightChildHash.High() + 1,
                        HashLow: rootRightChildHash.Low(),
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_NODE_HASH))
                Expect(req.MessageBody.(MerkleNodeHash).NodeID).Should(Equal(server1.Buckets().Get("default").MerkleTree().TranslateNode(rootNodeLeftLeftChild, 3)))
                Expect(initiatorSyncSession.State()).Should(Equal(LEFT_HASH_COMPARE))
                Expect(initiatorSyncSession.ExplorationQueueSize()).Should(Equal(uint32(2)))
                Expect(initiatorSyncSession.PopExplorationQueue()).Should(Equal(uint32(rootNodeLeftChild)))
                Expect(initiatorSyncSession.PopExplorationQueue()).Should(Equal(uint32(rootNodeRightChild)))
            })

            It("RIGHT_HASH_COMPARE -> LEFT_HASH_COMPARE ignore right hash", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                rootNode := server1.Buckets().Get("default").MerkleTree().RootNode()
                rootNodeLeftChild := server1.Buckets().Get("default").MerkleTree().LeftChild(rootNode)
                rootNodeLeftLeftChild := server1.Buckets().Get("default").MerkleTree().LeftChild(rootNodeLeftChild)
                rootNodeRightChild := server1.Buckets().Get("default").MerkleTree().RightChild(rootNode)
                rootRightChildHash := server1.Buckets().Get("default").MerkleTree().NodeHash(rootNodeRightChild)
                
                initiatorSyncSession.SetState(RIGHT_HASH_COMPARE)
                initiatorSyncSession.PushExplorationQueue(rootNode)
                initiatorSyncSession.PushExplorationQueue(rootNodeLeftChild)
                initiatorSyncSession.SetResponderDepth(3)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_NODE_HASH,
                    MessageBody: MerkleNodeHash{
                        NodeID: rootNodeRightChild,
                        HashHigh: rootRightChildHash.High(),
                        HashLow: rootRightChildHash.Low(),
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_NODE_HASH))
                Expect(req.MessageBody.(MerkleNodeHash).NodeID).Should(Equal(server1.Buckets().Get("default").MerkleTree().TranslateNode(rootNodeLeftLeftChild, 3)))
                Expect(initiatorSyncSession.State()).Should(Equal(LEFT_HASH_COMPARE))
                Expect(initiatorSyncSession.ExplorationQueueSize()).Should(Equal(uint32(1)))
                Expect(initiatorSyncSession.PopExplorationQueue()).Should(Equal(uint32(rootNodeLeftChild)))
            })

            It("RIGHT_HASH_COMPARE -> LEFT_HASH_COMPARE ignore right hash because limit hit", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, 1, true)
                
                rootNode := server1.Buckets().Get("default").MerkleTree().RootNode()
                rootNodeLeftChild := server1.Buckets().Get("default").MerkleTree().LeftChild(rootNode)
                rootNodeLeftLeftChild := server1.Buckets().Get("default").MerkleTree().LeftChild(rootNodeLeftChild)
                rootNodeRightChild := server1.Buckets().Get("default").MerkleTree().RightChild(rootNode)
                rootRightChildHash := server1.Buckets().Get("default").MerkleTree().NodeHash(rootNodeRightChild)
                
                initiatorSyncSession.SetState(RIGHT_HASH_COMPARE)
                initiatorSyncSession.PushExplorationQueue(rootNode)
                initiatorSyncSession.PushExplorationQueue(rootNodeLeftChild)
                initiatorSyncSession.SetResponderDepth(3)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_NODE_HASH,
                    MessageBody: MerkleNodeHash{
                        NodeID: rootNodeRightChild,
                        HashHigh: rootRightChildHash.High() + 1,
                        HashLow: rootRightChildHash.Low(),
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_NODE_HASH))
                Expect(req.MessageBody.(MerkleNodeHash).NodeID).Should(Equal(server1.Buckets().Get("default").MerkleTree().TranslateNode(rootNodeLeftLeftChild, 3)))
                Expect(initiatorSyncSession.State()).Should(Equal(LEFT_HASH_COMPARE))
                Expect(initiatorSyncSession.ExplorationQueueSize()).Should(Equal(uint32(1)))
                Expect(initiatorSyncSession.PopExplorationQueue()).Should(Equal(uint32(rootNodeLeftChild)))
            })
            
            It("RIGHT_HASH_COMPARE -> DB_OBJECT_PUSH", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                rootNode := server1.Buckets().Get("default").MerkleTree().RootNode()
                rootNodeLeftChild := server1.Buckets().Get("default").MerkleTree().LeftChild(rootNode)
                rootNodeRightChild := server1.Buckets().Get("default").MerkleTree().RightChild(rootNode)
                rootRightChildHash := server1.Buckets().Get("default").MerkleTree().NodeHash(rootNodeRightChild)
                
                initiatorSyncSession.SetState(RIGHT_HASH_COMPARE)
                initiatorSyncSession.PushExplorationQueue(rootNode)
                initiatorSyncSession.PushExplorationQueue(rootNodeLeftChild)
                initiatorSyncSession.SetResponderDepth(2)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_NODE_HASH,
                    MessageBody: MerkleNodeHash{
                        NodeID: rootNodeRightChild,
                        HashHigh: rootRightChildHash.High() + 1,
                        HashLow: rootRightChildHash.Low(),
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_OBJECT_NEXT))
                Expect(req.MessageBody.(ObjectNext).NodeID).Should(Equal(server1.Buckets().Get("default").MerkleTree().TranslateNode(rootNodeLeftChild, 2)))
                Expect(initiatorSyncSession.State()).Should(Equal(DB_OBJECT_PUSH))
                Expect(initiatorSyncSession.ExplorationQueueSize()).Should(Equal(uint32(2)))
                Expect(initiatorSyncSession.PopExplorationQueue()).Should(Equal(uint32(rootNodeLeftChild)))
                Expect(initiatorSyncSession.PopExplorationQueue()).Should(Equal(uint32(rootNodeRightChild)))

            })
            
            It("DB_OBJECT_PUSH -> END nil message", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(DB_OBJECT_PUSH)
                
                req := initiatorSyncSession.NextState(nil)
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(initiatorSyncSession.State()).Should(Equal(END))
                Expect(initiatorSyncSession.State()).Should(Equal(END))
            })
            
            It("DB_OBJECT_PUSH -> END non nil message", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)
                
                initiatorSyncSession.SetState(DB_OBJECT_PUSH)
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_ABORT,
                    MessageBody: Abort{ },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(initiatorSyncSession.State()).Should(Equal(END))
                Expect(initiatorSyncSession.State()).Should(Equal(END))
            })

            It("DB_OBJECT_PUSH -> DB_OBJECT_PUSH with SYNC_PUSH_DONE", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)

                rootNode := server1.Buckets().Get("default").MerkleTree().RootNode()
                rootNodeLeftChild := server1.Buckets().Get("default").MerkleTree().LeftChild(rootNode)
                rootNodeRightChild := server1.Buckets().Get("default").MerkleTree().RightChild(rootNode)
                
                initiatorSyncSession.SetState(DB_OBJECT_PUSH)
                initiatorSyncSession.PushExplorationQueue(rootNodeLeftChild)
                initiatorSyncSession.PushExplorationQueue(rootNodeRightChild)
                initiatorSyncSession.SetResponderDepth(2)

                Expect(initiatorSyncSession.PeekExplorationQueue()).Should(Equal(rootNodeLeftChild))
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_PUSH_DONE,
                    MessageBody: PushDone{ },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_OBJECT_NEXT))
                Expect(req.MessageBody.(ObjectNext).NodeID).Should(Equal(server1.Buckets().Get("default").MerkleTree().TranslateNode(rootNodeRightChild, 2)))
                Expect(initiatorSyncSession.State()).Should(Equal(DB_OBJECT_PUSH))
            })

            
            It("DB_OBJECT_PUSH -> DB_OBJECT_PUSH with SYNC_OBJECT_NEXT", func() {
                initiatorSyncSession := NewInitiatorSyncSession(123, server1BucketProxy, MERKLE_EXPLORATION_PATH_LIMIT, true)

                rootNode := server1.Buckets().Get("default").MerkleTree().RootNode()
                rootNodeLeftChild := server1.Buckets().Get("default").MerkleTree().LeftChild(rootNode)
                rootNodeRightChild := server1.Buckets().Get("default").MerkleTree().RightChild(rootNode)
                
                initiatorSyncSession.SetState(DB_OBJECT_PUSH)
                initiatorSyncSession.PushExplorationQueue(rootNodeLeftChild)
                initiatorSyncSession.PushExplorationQueue(rootNodeRightChild)
                initiatorSyncSession.SetResponderDepth(2)

                Expect(initiatorSyncSession.PeekExplorationQueue()).Should(Equal(rootNodeLeftChild))
                
                req := initiatorSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_PUSH_MESSAGE,
                    MessageBody: PushMessage{ 
                        Key: "abc",
                        Value: NewSiblingSet(map[*Sibling]bool{ }),
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_OBJECT_NEXT))
                Expect(req.MessageBody.(ObjectNext).NodeID).Should(Equal(server1.Buckets().Get("default").MerkleTree().TranslateNode(rootNodeLeftChild, 2)))
                Expect(initiatorSyncSession.State()).Should(Equal(DB_OBJECT_PUSH))
            })
        })
    })
    
    Describe("ResponderSyncSession", func() {
        Describe("#NextState", func() {
            var server1 *Server
            var server1BucketProxy ddbSync.BucketProxy
            stop1 := make(chan int)
            
            BeforeEach(func() {
                server1, _ = NewServer(ServerConfig{
                    DBFile: "/tmp/testdb-" + RandomString(),
                    Port: 8080,
                })
                
                go func() {
                    server1.Start()
                    stop1 <- 1
                }()
                
                time.Sleep(time.Millisecond * 200)
                
                server1BucketProxy = &ddbSync.RelayBucketProxy{ Bucket: server1.Buckets().Get("default") }
            })
            
            AfterEach(func() {
                server1.Stop()
                <-stop1
            })
            
            It("START -> END nil message", func() {
                responderSyncSession := NewResponderSyncSession(server1BucketProxy)
                
                responderSyncSession.SetState(START)
                
                req := responderSyncSession.NextState(nil)
                
                Expect(req.SessionID).Should(Equal(uint(0)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(responderSyncSession.State()).Should(Equal(END))
            })
            
            It("START -> END non nil message", func() {
                responderSyncSession := NewResponderSyncSession(server1BucketProxy)
                
                responderSyncSession.SetState(START)
                
                req := responderSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_ABORT,
                    MessageBody: Abort{ },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(responderSyncSession.State()).Should(Equal(END))
            })
            
            It("START -> HASH_COMPARE", func() {
                responderSyncSession := NewResponderSyncSession(server1BucketProxy)
                
                responderSyncSession.SetState(START)
                
                req := responderSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_START,
                    MessageBody: Start{
                        ProtocolVersion: PROTOCOL_VERSION,
                        MerkleDepth: 10,
                        Bucket: "default",
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(123)))
                Expect(req.MessageType).Should(Equal(SYNC_START))
                Expect(req.MessageBody.(Start).ProtocolVersion).Should(Equal(PROTOCOL_VERSION))
                Expect(req.MessageBody.(Start).MerkleDepth).Should(Equal(server1.Buckets().Get("default").MerkleTree().Depth()))
                Expect(req.MessageBody.(Start).Bucket).Should(Equal("default"))
                Expect(responderSyncSession.State()).Should(Equal(HASH_COMPARE))
                Expect(responderSyncSession.InitiatorDepth()).Should(Equal(uint8(10)))
            })
            
            It("HASH_COMPARE -> END nil message", func() {
                responderSyncSession := NewResponderSyncSession(server1BucketProxy)
                
                responderSyncSession.SetState(HASH_COMPARE)
                
                req := responderSyncSession.NextState(nil)
                
                Expect(req.SessionID).Should(Equal(uint(0)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(responderSyncSession.State()).Should(Equal(END))
            })
            
            It("HASH_COMPARE -> END non nil message", func() {
                responderSyncSession := NewResponderSyncSession(server1BucketProxy)
                
                responderSyncSession.SetState(HASH_COMPARE)
                
                req := responderSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_ABORT,
                    MessageBody: Abort{ },
                })
                
                Expect(req.SessionID).Should(Equal(uint(0)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(responderSyncSession.State()).Should(Equal(END))
            })
            
            It("HASH_COMPARE -> END SYNC_NODE_HASH message with 0 node ID", func() {
                responderSyncSession := NewResponderSyncSession(server1BucketProxy)
                
                responderSyncSession.SetState(HASH_COMPARE)
                
                req := responderSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_NODE_HASH,
                    MessageBody: MerkleNodeHash{ 
                        NodeID: 0,
                        HashHigh: 0,
                        HashLow: 0,
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(0)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(responderSyncSession.State()).Should(Equal(END))
            })
            
            It("HASH_COMPARE -> END SYNC_NODE_HASH message with limit node ID", func() {
                responderSyncSession := NewResponderSyncSession(server1BucketProxy)
                
                responderSyncSession.SetState(HASH_COMPARE)
                
                req := responderSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_NODE_HASH,
                    MessageBody: MerkleNodeHash{ 
                        NodeID: server1.Buckets().Get("default").MerkleTree().NodeLimit(),
                        HashHigh: 0,
                        HashLow: 0,
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(0)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(responderSyncSession.State()).Should(Equal(END))
            })
            
            It("HASH_COMPARE -> HASH_COMPARE", func() {
                responderSyncSession := NewResponderSyncSession(server1BucketProxy)
                
                responderSyncSession.SetInitiatorDepth(3)
                responderSyncSession.SetState(HASH_COMPARE)
                
                nodeID := server1.Buckets().Get("default").MerkleTree().NodeLimit() - 1
                nodeHash := server1.Buckets().Get("default").MerkleTree().NodeHash(nodeID)
                
                req := responderSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_NODE_HASH,
                    MessageBody: MerkleNodeHash{ 
                        NodeID: nodeID,
                        HashHigh: 0,
                        HashLow: 0,
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(0)))
                Expect(req.MessageType).Should(Equal(SYNC_NODE_HASH))
                Expect(req.MessageBody.(MerkleNodeHash).NodeID).Should(Equal(server1.Buckets().Get("default").MerkleTree().TranslateNode(nodeID, 3)))
                Expect(req.MessageBody.(MerkleNodeHash).HashHigh).Should(Equal(nodeHash.High()))
                Expect(req.MessageBody.(MerkleNodeHash).HashLow).Should(Equal(nodeHash.Low()))
                Expect(responderSyncSession.State()).Should(Equal(HASH_COMPARE))
            })
            
            It("HASH_COMPARE -> DB_OBJECT_PUSH SYNC_OBJECT_NEXT message with empty node", func() {
                responderSyncSession := NewResponderSyncSession(server1BucketProxy)
                
                responderSyncSession.SetState(HASH_COMPARE)
                
                req := responderSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_OBJECT_NEXT,
                    MessageBody: ObjectNext{ 
                        NodeID: 1,
                    },
                })
                
                Expect(req.SessionID).Should(Equal(uint(0)))
                Expect(req.MessageType).Should(Equal(SYNC_PUSH_DONE))
                Expect(responderSyncSession.State()).Should(Equal(DB_OBJECT_PUSH))
            })
            
            It("DB_OBJECT_PUSH -> END nil message", func() {
                responderSyncSession := NewResponderSyncSession(server1BucketProxy)
                
                responderSyncSession.SetState(DB_OBJECT_PUSH)
                
                req := responderSyncSession.NextState(nil)
                
                Expect(req.SessionID).Should(Equal(uint(0)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(responderSyncSession.State()).Should(Equal(END))
            })
            
            It("DB_OBJECT_PUSH -> END non nil message", func() {
                responderSyncSession := NewResponderSyncSession(server1BucketProxy)
                
                responderSyncSession.SetState(DB_OBJECT_PUSH)
                
                req := responderSyncSession.NextState(&SyncMessageWrapper{
                    SessionID: 123,
                    MessageType: SYNC_ABORT,
                    MessageBody: Abort{ },
                })
                
                Expect(req.SessionID).Should(Equal(uint(0)))
                Expect(req.MessageType).Should(Equal(SYNC_ABORT))
                Expect(responderSyncSession.State()).Should(Equal(END))
            })
        })
    })
})
