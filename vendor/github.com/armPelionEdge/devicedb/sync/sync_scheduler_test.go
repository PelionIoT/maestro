package sync_test

import (
    "fmt"
    "math/rand"
    "sync"
    "time"

    . "github.com/armPelionEdge/devicedb/sync"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("SyncScheduler", func() {
    Describe("Peer", func() {
        var peer *Peer

        Describe("NextBucket", func() {
            Context("When the buckets array for that peer is empty", func() {
                BeforeEach(func() {
                    peer = NewPeer("A", []string{ })
                })

                It("Should return an empty string", func() {
                    Expect(peer.NextBucket()).Should(BeEmpty())
                })
            })

            Context("When the buckets array for that peer is not empty", func() {
                BeforeEach(func() {
                    peer = NewPeer("A", []string{ "default", "lww", "cloud" })
                })

                It("Should return the next bucket in the buckets array", func() {
                    Expect(peer.NextBucket()).Should(Equal("default"))
                })

                Context("And it is called multiple times in succession without Advance() being called", func() {
                    It("Should keep returning the same bucket", func() {
                        Expect(peer.NextBucket()).Should(Equal("default"))
                        Expect(peer.NextBucket()).Should(Equal("default"))
                        Expect(peer.NextBucket()).Should(Equal("default"))
                        Expect(peer.NextBucket()).Should(Equal("default"))
                    })
                })

                Context("And it is called multiple times with Advance() called in between", func() {
                    It("Should return the next bucket in the list each time until it rotates back to the beginning", func() {
                        Expect(peer.NextBucket()).Should(Equal("default"))
                        peer.Advance()
                        Expect(peer.NextBucket()).Should(Equal("lww"))
                        peer.Advance()
                        Expect(peer.NextBucket()).Should(Equal("cloud"))
                        peer.Advance()
                        Expect(peer.NextBucket()).Should(Equal("default"))
                        peer.Advance()
                        Expect(peer.NextBucket()).Should(Equal("lww"))
                        peer.Advance()
                        Expect(peer.NextBucket()).Should(Equal("cloud"))
                    })
                })
            })
        })
    })

    Describe("PeriodicSyncScheduler", func() {
        var syncScheduler *PeriodicSyncScheduler
        var syncPeriod time.Duration

        BeforeEach(func() {
            syncPeriod = time.Millisecond * 100
            syncScheduler = NewPeriodicSyncScheduler(syncPeriod)
        })

        Describe("Calling Next()", func() {
            Context("When there are some scheduled peers when Next() is invoked", func() {
                BeforeEach(func() {
                    syncScheduler.AddPeer("peer1", []string{ "lww", "default", "cloud" })
                    syncScheduler.AddPeer("peer2", []string{ "default", "cloud", "lww" })
                    syncScheduler.Schedule("peer2")
                    syncScheduler.Schedule("peer1")
                })

                It("Should wait for the specified sync period before returning", func() {
                    startTime := time.Now()
                    syncScheduler.Next()
                    Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                })

                It("Should return the next scheduled peer id and bucket for that peer", func() {
                    peer, bucket := syncScheduler.Next()

                    Expect(peer).Should(Equal("peer2"))
                    Expect(bucket).Should(Equal("default"))
                })

                Context("And when called multiple times in succession without Advance() being called in between", func() {
                    It("Should wait for the specified sync period before returning for each call", func() {
                        startTime := time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                        startTime = time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                        startTime = time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                    })

                    It("Should return the same peer and bucket each time", func() {
                        peer, bucket := syncScheduler.Next()

                        Expect(peer).Should(Equal("peer2"))
                        Expect(bucket).Should(Equal("default"))

                        peer, bucket = syncScheduler.Next()

                        Expect(peer).Should(Equal("peer2"))
                        Expect(bucket).Should(Equal("default"))

                        peer, bucket = syncScheduler.Next()

                        Expect(peer).Should(Equal("peer2"))
                        Expect(bucket).Should(Equal("default"))
                    })
                })

                Context("And when called multiple times in succession with Advance() called in between but without being re-scheduled", func() {
                    It("Should wait the specified sync period before returning for each call", func() {
                        startTime := time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                        syncScheduler.Advance()

                        startTime = time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                        syncScheduler.Advance()

                        startTime = time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))  
                    })

                    It("Should return the next peer and bucket each time until scheduled peers run out", func() {
                        peer, bucket := syncScheduler.Next()

                        Expect(peer).Should(Equal("peer2"))
                        Expect(bucket).Should(Equal("default"))

                        syncScheduler.Advance()

                        peer, bucket = syncScheduler.Next()

                        Expect(peer).Should(Equal("peer1"))
                        Expect(bucket).Should(Equal("lww"))

                        syncScheduler.Advance()

                        peer, bucket = syncScheduler.Next()

                        Expect(peer).Should(Equal(""))
                        Expect(bucket).Should(Equal(""))
                    })
                })

                Context("And when called multiple times in succession with Advance() called in between while being re-scheduled", func() {
                    It("Should wait the specified sync period before returning for each call", func() {
                        startTime := time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                        syncScheduler.Advance()

                        startTime = time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                        syncScheduler.Advance()
                        syncScheduler.Schedule("peer1")
                        syncScheduler.Schedule("peer2")

                        startTime = time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))  
                    })

                    It("Should return the next peer and bucket each time until repeated peers are encountered at which point the bucket for each peer should be rotated", func() {
                        peer, bucket := syncScheduler.Next()

                        Expect(peer).Should(Equal("peer2"))
                        Expect(bucket).Should(Equal("default"))

                        syncScheduler.Advance()

                        peer, bucket = syncScheduler.Next()

                        Expect(peer).Should(Equal("peer1"))
                        Expect(bucket).Should(Equal("lww"))

                        syncScheduler.Advance()
                        syncScheduler.Schedule("peer1")
                        syncScheduler.Schedule("peer2")

                        peer, bucket = syncScheduler.Next()

                        Expect(peer).Should(Equal("peer1"))
                        Expect(bucket).Should(Equal("default"))

                        syncScheduler.Advance()

                        peer, bucket = syncScheduler.Next()

                        Expect(peer).Should(Equal("peer2"))
                        Expect(bucket).Should(Equal("cloud"))
                    })
                })
            })

            Context("When there are no scheduled peers when Next() is invoked", func() {
                It("Should wait for the specified sync period before returning", func() {
                    startTime := time.Now()
                    syncScheduler.Next()
                    Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                })

                It("Should return an empty string for the peer id and bucket name", func() {
                    peer, bucket := syncScheduler.Next()

                    Expect(peer).Should(Equal(""))
                    Expect(bucket).Should(Equal(""))
                })
            })
        })

        Describe("#RemovePeer", func() {
            BeforeEach(func() {
                syncScheduler.AddPeer("peer1", []string{ "lww", "default", "cloud" })
                syncScheduler.AddPeer("peer2", []string{ "default", "cloud", "lww" })
                syncScheduler.Schedule("peer2")
                syncScheduler.Schedule("peer1")
            })

            It("Should remove that peer from the queue", func() {
                syncScheduler.RemovePeer("peer2")

                peer, bucket := syncScheduler.Next()

                Expect(peer).Should(Equal("peer1"))
                Expect(bucket).Should(Equal("lww"))

                syncScheduler.RemovePeer("peer1")

                peer, bucket = syncScheduler.Next()

                Expect(peer).Should(Equal(""))
                Expect(bucket).Should(Equal(""))
            })
        })
    })

    Describe("MultiSyncScheduler", func() {
        var syncScheduler *MultiSyncScheduler
        var syncPeriod time.Duration

        BeforeEach(func() {
            syncPeriod = time.Millisecond * 100
            syncScheduler = NewMultiSyncScheduler(syncPeriod)
        })

        Describe("Calling Next()", func() {
            Context("When there are some scheduled peers when Next() is invoked", func() {
                BeforeEach(func() {
                    syncScheduler.AddPeer("peer1", []string{ "lww", "default", "cloud" })
                    syncScheduler.AddPeer("peer2", []string{ "default", "cloud", "lww" })
                    syncScheduler.Schedule("peer2")
                    syncScheduler.Schedule("peer1")
                })

                Context("And when called multiple times in succession without Advance() being called in between", func() {
                    It("Should wait the sync period for each call", func() {
                        // If Advance() is not called in between we don't want the loop that is reading from Next()
                        // to keep trying to sumit a new initiator session if one is not available. So if next is called
                        // more than once without Advance() being called in between we need to wait the timeout period each time
                        startTime := time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))

                        startTime = time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                    })

                    It("Should return the same peer and bucket each time", func() {
                        peer, bucket := syncScheduler.Next()

                        Expect(peer).Should(Equal("peer2"))
                        Expect(bucket).Should(Equal("default"))

                        peer, bucket = syncScheduler.Next()

                        Expect(peer).Should(Equal("peer2"))
                        Expect(bucket).Should(Equal("default"))

                        peer, bucket = syncScheduler.Next()

                        Expect(peer).Should(Equal("peer2"))
                        Expect(bucket).Should(Equal("default"))
                    })
                })

                Context("And when called multiple times in succession with Advance() called in between but without being re-scheduled", func() {
                    It("Should wait the specified sync period for the first call while returning right away for each call after that until it is empty", func() {
                        startTime := time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                        syncScheduler.Advance()

                        startTime = time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime))
                        syncScheduler.Advance()

                        startTime = time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                    })

                    It("Should return the next peer and bucket each time until scheduled peers run out", func() {
                        peer, bucket := syncScheduler.Next()

                        Expect(peer).Should(Equal("peer2"))
                        Expect(bucket).Should(Equal("default"))

                        syncScheduler.Advance()

                        peer, bucket = syncScheduler.Next()

                        Expect(peer).Should(Equal("peer1"))
                        Expect(bucket).Should(Equal("lww"))

                        syncScheduler.Advance()

                        peer, bucket = syncScheduler.Next()

                        Expect(peer).Should(Equal(""))
                        Expect(bucket).Should(Equal(""))
                    })
                })

                Context("And when called multiple times in succession with Advance() called in between while being re-scheduled", func() {
                    It("Should wait the specified sync period before returning for the first call and the call after they are re-scheduled", func() {
                        startTime := time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                        syncScheduler.Advance()

                        startTime = time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime))
                        syncScheduler.Advance()
                        syncScheduler.Schedule("peer1")
                        syncScheduler.Schedule("peer2")

                        startTime = time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                        syncScheduler.Advance()

                        startTime = time.Now()
                        syncScheduler.Next()
                        Expect(time.Now()).Should(BeTemporally("~", startTime))
                    })

                    It("Should return the next peer and bucket each time until repeated peers are encountered at which point the bucket for each peer should be rotated", func() {
                        peer, bucket := syncScheduler.Next()

                        Expect(peer).Should(Equal("peer2"))
                        Expect(bucket).Should(Equal("default"))

                        syncScheduler.Advance()

                        peer, bucket = syncScheduler.Next()

                        Expect(peer).Should(Equal("peer1"))
                        Expect(bucket).Should(Equal("lww"))

                        syncScheduler.Advance()
                        syncScheduler.Schedule("peer1")
                        syncScheduler.Schedule("peer2")

                        peer, bucket = syncScheduler.Next()

                        Expect(peer).Should(Equal("peer1"))
                        Expect(bucket).Should(Equal("default"))

                        syncScheduler.Advance()

                        peer, bucket = syncScheduler.Next()

                        Expect(peer).Should(Equal("peer2"))
                        Expect(bucket).Should(Equal("cloud"))
                    })
                })
            })

           Context("When there are no scheduled peers when Next() is invoked", func() {
                It("Should wait for the specified sync period before returning", func() {
                    startTime := time.Now()
                    syncScheduler.Next()
                    Expect(time.Now()).Should(BeTemporally("~", startTime.Add(syncPeriod)))
                })

                It("Should return an empty string for the peer id and bucket name", func() {
                    peer, bucket := syncScheduler.Next()

                    Expect(peer).Should(Equal(""))
                    Expect(bucket).Should(Equal(""))
                })
            })
        })

        Describe("#RemovePeer", func() {
            BeforeEach(func() {
                syncScheduler.AddPeer("peer1", []string{ "lww", "default", "cloud" })
                syncScheduler.AddPeer("peer2", []string{ "default", "cloud", "lww" })
                syncScheduler.Schedule("peer2")
                syncScheduler.Schedule("peer1")
            })

            It("Should remove that peer from the heap", func() {
                syncScheduler.RemovePeer("peer2")

                peer, bucket := syncScheduler.Next()

                Expect(peer).Should(Equal("peer1"))
                Expect(bucket).Should(Equal("lww"))

                syncScheduler.RemovePeer("peer1")

                peer, bucket = syncScheduler.Next()

                Expect(peer).Should(Equal(""))
                Expect(bucket).Should(Equal(""))
            })
        })

        Specify("For any given peer they should not be synced with at the specified sync rate", func() {
            var peerChans map[string]chan string = make(map[string]chan string)
            var syncCountsMu sync.Mutex
            var syncCounts map[string]int = make(map[string]int)
            var buckets = []string{ "default", "lww", "cloud" }

            for i := 0; i < 1000; i++ {
                var peerID string = fmt.Sprintf("peer-%d", i)
                peerChans[peerID] = make(chan string)
                syncCounts[peerID] = 0
                syncScheduler.AddPeer(peerID, buckets)
            }

            for peerID, peerChan := range peerChans {
                go func(peerID string, peerChan chan string) {
                    //defer GinkgoRecover()

                    var nextBucket int

                    // Randomize when peers call schedule next
                    var scheduleTime time.Time = time.Now()
                    syncScheduler.Schedule(peerID)

                    for bucket := range peerChan {
                        fmt.Printf("Peer %s sync with bucket %s\n", peerID, bucket)
                        Expect(bucket).Should(Equal(buckets[nextBucket]))

                        syncCountsMu.Lock()
                        syncCounts[peerID]++
                        syncCountsMu.Unlock()
                        nextBucket = (nextBucket + 1) % len(buckets)

                        // should be timed correctly allowing for a jitter of 100 milliseconds
                        Expect(time.Now()).Should(BeTemporally("~", scheduleTime.Add(syncPeriod), time.Millisecond * 100))

                        // Randomize when peers call schedule next
                        <-time.After(time.Millisecond * time.Duration(rand.Uint32() % 500))

                        scheduleTime = time.Now()
                        syncScheduler.Schedule(peerID)
                    }
                }(peerID, peerChan)
            }

            // This should allow for each peer to receive exactly 10 sync events
            var syncEvents int

            for syncEvents < len(peerChans) * 10 {
                peer, bucket := syncScheduler.Next()

                if peer == "" {
                    continue
                }

                syncEvents++
                peerChans[peer] <- bucket
                syncScheduler.Advance()
            }

            for _, syncCount := range syncCounts {
                Expect(syncCount).Should(BeNumerically("~", 10, 5))
            }
        })
    })
})
