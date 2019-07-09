package bucket_test

import (
    "context"
    "time"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/bucket"
)

var _ = Describe("Monitor", func() {
    Describe("#AddListener", func() {
        Context("When the supplied context is cancelled", func() {
            Specify("The supplied channel should be closed by the monitor", func() {
                monitor := NewMonitor(0)
                ctx, cancel := context.WithCancel(context.Background())
                deliveryChannel := make(chan data.Row)

                monitor.AddListener(ctx, [][]byte{ }, [][]byte{ }, deliveryChannel)

                select {
                case <-ctx.Done():
                    Fail("Context done channel should not have been closed")
                default:
                }

                cancel()

                select {
                case <-ctx.Done():
                case <-time.After(time.Second):
                    Fail("Context done channel should have been closed")
                }
            })
        })
    })

    Describe("#Notify", func() {
        Context("When the last delivered version is 0", func() {
            var monitor *Monitor

            BeforeEach(func() {
                monitor = NewMonitor(0)
            })

            Context("When there is a listener who specified a prefix matching this updates key", func() {
                var ctx context.Context
                var deliveryChannel chan data.Row

                BeforeEach(func() {
                    ctx, _ = context.WithCancel(context.Background())
                    deliveryChannel = make(chan data.Row)

                    monitor.AddListener(ctx, [][]byte{ }, [][]byte{ []byte("a") }, deliveryChannel)
                })

                Context("And the submitted update has a LocalVersion of 0", func() {
                    Specify("The update should be delivered to that listener right away", func() {
                        go monitor.Notify(data.Row{ Key: "abc", LocalVersion: 0, Siblings: &data.SiblingSet{ } })

                        select {
                        case update := <-deliveryChannel:
                            Expect(update.LocalVersion).Should(Equal(uint64(0)))
                            Expect(update.Key).Should(Equal("abc"))
                            Expect(*update.Siblings).Should(Equal(data.SiblingSet{ }))
                        case <-time.After(time.Second):
                            Fail("Should have received an update")
                        }
                    })
                })

                Context("And the submitted update has a LocalVersion of 1", func() {
                    Specify("The update should be delivered to that listener right away", func() {
                        go monitor.Notify(data.Row{ Key: "abc", LocalVersion: 1, Siblings: &data.SiblingSet{ } })

                        select {
                        case update := <-deliveryChannel:
                            Expect(update.LocalVersion).Should(Equal(uint64(1)))
                            Expect(update.Key).Should(Equal("abc"))
                            Expect(*update.Siblings).Should(Equal(data.SiblingSet{ }))
                        case <-time.After(time.Second):
                            Fail("Should have received an update")
                        }
                    })
                })

                Context("And the submitted update has a LocalVersoin greater than 1", func() {
                    Specify("The update should not be delivered to that listener until an update with version 0 or 1 is submitted", func() {
                        go monitor.Notify(data.Row{ Key: "abcd", LocalVersion: 2, Siblings: &data.SiblingSet{ } })

                        select {
                        case <-deliveryChannel:
                            Fail("Should not have received an update yet")                            
                        case <-time.After(time.Second):
                        }

                        go monitor.Notify(data.Row{ Key: "abc", LocalVersion: 1, Siblings: &data.SiblingSet{ } })
                        
                        select {
                        case update := <-deliveryChannel:
                            Expect(update.LocalVersion).Should(Equal(uint64(1)))
                            Expect(update.Key).Should(Equal("abc"))
                            Expect(*update.Siblings).Should(Equal(data.SiblingSet{ }))
                        case <-time.After(time.Second):
                            Fail("Should have received an update")
                        }

                        select {
                        case update := <-deliveryChannel:
                            Expect(update.LocalVersion).Should(Equal(uint64(2)))
                            Expect(update.Key).Should(Equal("abcd"))
                            Expect(*update.Siblings).Should(Equal(data.SiblingSet{ }))
                        case <-time.After(time.Second):
                            Fail("Should have received another update")
                        }

                        select {
                        case <-deliveryChannel:
                            Fail("Should not have received another update")
                        case <-time.After(time.Second):  
                        }
                    })
                })
            })
        })

        Context("When the last delivered version is greater than 0", func() {
            var monitor *Monitor

            BeforeEach(func() {
                monitor = NewMonitor(3)
            })

            Context("When there is a listener who specified a prefix matching this updates key", func() {
                var ctx context.Context
                var deliveryChannel chan data.Row

                BeforeEach(func() {
                    ctx, _ = context.WithCancel(context.Background())
                    deliveryChannel = make(chan data.Row)

                    monitor.AddListener(ctx, [][]byte{ }, [][]byte{ []byte("a") }, deliveryChannel)
                })

                Context("And the submitted update has a LocalVersion of 0", func() {
                    Specify("The update should be discarded and not delivered to the listener", func() {
                        go monitor.Notify(data.Row{ Key: "abc", LocalVersion: 0, Siblings: &data.SiblingSet{ } })

                        select {
                        case <-deliveryChannel:
                            Fail("Should not have received an update")
                        case <-time.After(time.Second):  
                        }
                    })
                })

                Context("And the submitted update has a LocalVersion less than the last delivered version and greater than 0", func() {
                    Specify("The update should be discarded and not delivered to the listener", func() {
                        go monitor.Notify(data.Row{ Key: "abc", LocalVersion: 1, Siblings: &data.SiblingSet{ } })

                        select {
                        case <-deliveryChannel:
                            Fail("Should not have received an update")
                        case <-time.After(time.Second):  
                        }
                    })
                })

                Context("And the submitted update has a LocalVersion equal to the last delivered version", func() {
                    Specify("The update should be discarded and not delivered to the listener", func() {
                        go monitor.Notify(data.Row{ Key: "abc", LocalVersion: 3, Siblings: &data.SiblingSet{ } })

                        select {
                        case <-deliveryChannel:
                            Fail("Should not have received an update")
                        case <-time.After(time.Second):  
                        }
                    })
                })

                Context("And the submitted update has a LocalVersion that is 1 + the last delivered version", func() {
                    Specify("The update should be delivered to that listener right away", func() {
                        go monitor.Notify(data.Row{ Key: "abc", LocalVersion: 4, Siblings: &data.SiblingSet{ } })

                        select {
                        case update := <-deliveryChannel:
                            Expect(update.LocalVersion).Should(Equal(uint64(4)))
                            Expect(update.Key).Should(Equal("abc"))
                            Expect(*update.Siblings).Should(Equal(data.SiblingSet{ }))
                        case <-time.After(time.Second):
                            Fail("Should have received an update")
                        }
                    })
                })

                Context("And the submitted update has a LocalVersion that is greater than 1 + the last delivered version", func() {
                    Specify("The update should not be delivered to that listener until an update with LocalVersion equal to 1 + the last delivered version is submitted", func() {
                        go monitor.Notify(data.Row{ Key: "abcdef", LocalVersion: 5, Siblings: &data.SiblingSet{ } })
                        go monitor.Notify(data.Row{ Key: "abcdg", LocalVersion: 6, Siblings: &data.SiblingSet{ } })
                        go monitor.Notify(data.Row{ Key: "abcd", LocalVersion: 8, Siblings: &data.SiblingSet{ } })

                        select {
                        case <-deliveryChannel:
                            Fail("Should not have received an update yet")                            
                        case <-time.After(time.Second):
                        }

                        go monitor.Notify(data.Row{ Key: "abc", LocalVersion: 4, Siblings: &data.SiblingSet{ } })
                        
                        select {
                        case update := <-deliveryChannel:
                            Expect(update.LocalVersion).Should(Equal(uint64(4)))
                            Expect(update.Key).Should(Equal("abc"))
                            Expect(*update.Siblings).Should(Equal(data.SiblingSet{ }))
                        case <-time.After(time.Second):
                            Fail("Should have received an update")
                        }

                        select {
                        case update := <-deliveryChannel:
                            Expect(update.LocalVersion).Should(Equal(uint64(5)))
                            Expect(update.Key).Should(Equal("abcdef"))
                            Expect(*update.Siblings).Should(Equal(data.SiblingSet{ }))
                        case <-time.After(time.Second):
                            Fail("Should have received another update")
                        }

                        select {
                        case update := <-deliveryChannel:
                            Expect(update.LocalVersion).Should(Equal(uint64(6)))
                            Expect(update.Key).Should(Equal("abcdg"))
                            Expect(*update.Siblings).Should(Equal(data.SiblingSet{ }))
                        case <-time.After(time.Second):
                            Fail("Should have received another update")
                        }

                        select {
                        case <-deliveryChannel:
                            Fail("Should not have received another update")
                        case <-time.After(time.Second):  
                        }

                        go monitor.Notify(data.Row{ Key: "aaaa", LocalVersion: 7, Siblings: &data.SiblingSet{ } })
                        
                        select {
                        case update := <-deliveryChannel:
                            Expect(update.LocalVersion).Should(Equal(uint64(7)))
                            Expect(update.Key).Should(Equal("aaaa"))
                            Expect(*update.Siblings).Should(Equal(data.SiblingSet{ }))
                        case <-time.After(time.Second):
                            Fail("Should have received another update")
                        }

                        select {
                        case update := <-deliveryChannel:
                            Expect(update.LocalVersion).Should(Equal(uint64(8)))
                            Expect(update.Key).Should(Equal("abcd"))
                            Expect(*update.Siblings).Should(Equal(data.SiblingSet{ }))
                        case <-time.After(time.Second):
                            Fail("Should have received another update")
                        }

                        select {
                        case <-deliveryChannel:
                            Fail("Should not have received another update")
                        case <-time.After(time.Second):  
                        }
                    })
                })
            })
        })
    })

    Describe("#DiscardIDRange", func() {
    })
})
