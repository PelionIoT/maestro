package util_test

import (
    "time"

    . "github.com/armPelionEdge/devicedb/util"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("RwTryLock", func() {
    Describe("#TryRLock", func() {
        Context("When a call to WLock() is blocked", func() {
            It("Should return false", func() {
                var lock RWTryLock

                lock.TryRLock()

                writeLockSucceeded := make(chan int)

                go func() {
                    lock.WLock()
                    writeLockSucceeded <- 1
                }()

                go func() {
                    <-time.After(time.Millisecond * 100)
                    Expect(lock.TryRLock()).Should(BeFalse())
                }()

                select {
                case <-writeLockSucceeded:
                    Fail("Write lock should have remained blocked")
                case <-time.After(time.Second):
                }
            })
        })

        Context("When a call to WLock() has already completed and WUnlock() has not been called yet", func() {
            It("Should return false", func() {
                var lock RWTryLock

                lock.WLock()
                Expect(lock.TryRLock()).Should(BeFalse())
            })
        })

        Context("When a call to WLock() has not been made yet", func() {
            It("Should return true", func() {
                var lock RWTryLock

                Expect(lock.TryRLock()).Should(BeTrue())
            })
        })

        Context("When a call to WLock() has been made and it has been unlocked by a call to WUnlock()", func() {
            It("Should return true", func() {
                var lock RWTryLock

                lock.WLock()
                lock.WUnlock()
                Expect(lock.TryRLock()).Should(BeTrue())
            })
        })
    })

    Describe("#WLock", func() {
        Context("When there are no readers currently", func() {
            It("Should not block", func() {
                var lock RWTryLock

                writeLockSucceeded := make(chan int)

                go func() {
                    lock.WLock()
                    writeLockSucceeded <- 1
                }()

                select {
                case <-writeLockSucceeded:
                case <-time.After(time.Second):
                    Fail("Write lock should be done")
                }
            })
        })

        Context("When calls to TryRLock() have already succeeded", func() {
            It("Should block until there are a corresponding number of calls made to RUnlock()", func() {
                var lock RWTryLock

                Expect(lock.TryRLock()).Should(BeTrue())
                Expect(lock.TryRLock()).Should(BeTrue())
                Expect(lock.TryRLock()).Should(BeTrue())

                writeLockSucceeded := make(chan int)

                go func() {
                    lock.WLock()
                    writeLockSucceeded <- 1
                }()

                select {
                case <-writeLockSucceeded:
                    Fail("Write lock should block until all reads are done")
                default:
                }

                lock.RUnlock()

                select {
                case <-writeLockSucceeded:
                    Fail("Write lock should block until all reads are done")
                default:
                }

                lock.RUnlock()

                select {
                case <-writeLockSucceeded:
                    Fail("Write lock should block until all reads are done")
                default:
                }

                lock.RUnlock()

                select {
                case <-writeLockSucceeded:
                case <-time.After(time.Second):
                    Fail("Write lock should be done")
                }
            })
        })
    })
})
