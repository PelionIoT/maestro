package clusterio_test

import (
    . "github.com/armPelionEdge/devicedb/clusterio"
    . "github.com/armPelionEdge/devicedb/data"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("SiblingSetMergeIterator", func() {
    var readMerger *ReadMerger
    var mergeIterator *SiblingSetMergeIterator

    BeforeEach(func() {
        readMerger = NewReadMerger("default")
        mergeIterator = NewSiblingSetMergeIterator(readMerger)
    })

    Describe("#Next", func() {
        Context("No keys have been added with AddKey()", func() {
            It("Should return false", func() {
                Expect(mergeIterator.Next()).Should(BeFalse())
            })
        })

        Context("A single key has been added with AddKey()", func() {
            BeforeEach(func() {
                mergeIterator.AddKey("a", "b")
            })

            It("Should return true for the first call and then false for subsequent calls", func() {
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Next()).Should(BeFalse())
                Expect(mergeIterator.Next()).Should(BeFalse())
                Expect(mergeIterator.Next()).Should(BeFalse())
            })
        })

        Context("Several distinct keys were added with AddKey() with the same prefix", func() {
            BeforeEach(func() {
                mergeIterator.AddKey("a", "b")
                mergeIterator.AddKey("a", "c")
                mergeIterator.AddKey("a", "d")
            })

            It("Should return true for as many calls to Next() as there were keys inserted with calls to AddKey()", func() {
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Next()).Should(BeFalse())
                Expect(mergeIterator.Next()).Should(BeFalse())
                Expect(mergeIterator.Next()).Should(BeFalse())
            })
        })

        Context("Several distinct keys were added with AddKey() with differing prefixes", func() {
            BeforeEach(func() {
                mergeIterator.AddKey("a", "b")
                mergeIterator.AddKey("b", "c")
                mergeIterator.AddKey("c", "d")
            })

            It("Should return true for as many calls to Next() as there were keys inserted with calls to AddKey()", func() {
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Next()).Should(BeFalse())
                Expect(mergeIterator.Next()).Should(BeFalse())
                Expect(mergeIterator.Next()).Should(BeFalse())
            })
        })

        Context("Several keys were added with AddKey() with the same prefix and some keys are identical", func() {
            BeforeEach(func() {
                mergeIterator.AddKey("a", "b")
                mergeIterator.AddKey("a", "c")
                mergeIterator.AddKey("a", "b")
                mergeIterator.AddKey("a", "d")
            })

            It("Should return true for as many calls to Next() as there were distinct keys inserted", func() {
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Next()).Should(BeFalse())
                Expect(mergeIterator.Next()).Should(BeFalse())
                Expect(mergeIterator.Next()).Should(BeFalse())
            })
        })

        Context("Several keys were added with AddKey() with differing prefixes and some keys are identical", func() {
            BeforeEach(func() {
                mergeIterator.AddKey("a", "b")
                mergeIterator.AddKey("b", "c")
                mergeIterator.AddKey("c", "b")
                mergeIterator.AddKey("d", "d")
            })

            It("Should return true for as many calls to Next() as there were distinct keys inserted", func() {
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Next()).Should(BeFalse())
                Expect(mergeIterator.Next()).Should(BeFalse())
                Expect(mergeIterator.Next()).Should(BeFalse())
            })
        })
    })

    Describe("#Prefix", func() {
        Context("No keys have been added with AddKey()", func() {
            Context("And Next() has not been called yet", func() {
                It("Should return nil", func() {
                    Expect(mergeIterator.Prefix()).Should(BeNil())
                })
            })

            Context("And Next() has been called", func() {
                BeforeEach(func() {
                    mergeIterator.Next()
                })

                It("Should return nil", func() {
                    Expect(mergeIterator.Prefix()).Should(BeNil())
                })
            })
        })

        Context("Some keys have been added with AddKey()", func() {
            BeforeEach(func() {
                mergeIterator.AddKey("a", "b")
                mergeIterator.AddKey("a", "bb")
                mergeIterator.AddKey("b", "c")
                mergeIterator.AddKey("b", "cc")
                mergeIterator.AddKey("c", "d")
                mergeIterator.AddKey("c", "ddA")
            })

            It("Should return prefixes in the same order and cardinality with which they were encountered after subsequent calls to Next()", func() {
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Prefix()).Should(Equal([]byte("a")))
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Prefix()).Should(Equal([]byte("a")))
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Prefix()).Should(Equal([]byte("b")))
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Prefix()).Should(Equal([]byte("b")))
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Prefix()).Should(Equal([]byte("c")))
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Prefix()).Should(Equal([]byte("c")))
                Expect(mergeIterator.Next()).Should(BeFalse())
                Expect(mergeIterator.Prefix()).Should(BeNil())
            })

            Context("And Next() has not been called yet", func() {
                It("Should return nil", func() {
                    Expect(mergeIterator.Prefix()).Should(BeNil())
                })
            })

            Context("And Next() has been called more times than keys added", func() {
                It("Should return nil", func() {
                    Expect(mergeIterator.Next()).Should(BeTrue())
                    Expect(mergeIterator.Next()).Should(BeTrue())
                    Expect(mergeIterator.Next()).Should(BeTrue())
                    Expect(mergeIterator.Next()).Should(BeTrue())
                    Expect(mergeIterator.Next()).Should(BeTrue())
                    Expect(mergeIterator.Next()).Should(BeTrue())
                    Expect(mergeIterator.Next()).Should(BeFalse())
                    Expect(mergeIterator.Prefix()).Should(BeNil())
                })
            })
        })
    })

    Describe("#Key", func() {
        Context("Some keys have been added with AddKey()", func() {
            BeforeEach(func() {
                mergeIterator.AddKey("a", "b")
                mergeIterator.AddKey("a", "bb")
                mergeIterator.AddKey("b", "c")
                mergeIterator.AddKey("b", "cc")
                mergeIterator.AddKey("c", "d")
                mergeIterator.AddKey("c", "dd")
            })

            It("Should return keys in the same order that they were encountered in after subsequent calls to Next()", func() {
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Key()).Should(Equal([]byte("b")))
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Key()).Should(Equal([]byte("bb")))
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Key()).Should(Equal([]byte("c")))
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Key()).Should(Equal([]byte("cc")))
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Key()).Should(Equal([]byte("d")))
                Expect(mergeIterator.Next()).Should(BeTrue())
                Expect(mergeIterator.Key()).Should(Equal([]byte("dd")))
                Expect(mergeIterator.Next()).Should(BeFalse())
                Expect(mergeIterator.Key()).Should(BeNil())
            })

            Context("And Next() has not been called yet", func() {
                It("Should return nil", func() {
                    Expect(mergeIterator.Key()).Should(BeNil())
                })
            })

            Context("And Next() has been called more times than keys added", func() {
                It("Should return nil", func() {
                    Expect(mergeIterator.Next()).Should(BeTrue())
                    Expect(mergeIterator.Next()).Should(BeTrue())
                    Expect(mergeIterator.Next()).Should(BeTrue())
                    Expect(mergeIterator.Next()).Should(BeTrue())
                    Expect(mergeIterator.Next()).Should(BeTrue())
                    Expect(mergeIterator.Next()).Should(BeTrue())
                    Expect(mergeIterator.Next()).Should(BeFalse())
                    Expect(mergeIterator.Key()).Should(BeNil())
                })
            })
        })

        Context("No keys have been added with AddKey()", func() {
            Context("And Next() has not been called yet", func() {
                It("Should return nil", func() {
                    Expect(mergeIterator.Key()).Should(BeNil())
                })
            })

            Context("And Next() has been called", func() {
                It("Should return nil", func() {
                    Expect(mergeIterator.Next()).Should(BeFalse())
                    Expect(mergeIterator.Key()).Should(BeNil())
                })
            })
        })
    })

    Describe("#Value", func() {
        sibling1 := NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v1"), 0)
        sibling2 := NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), []byte("v2"), 0)
        sibling3 := NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), []byte("v3"), 0)
        
        siblingSet1 := NewSiblingSet(map[*Sibling]bool{
            sibling1: true,
            sibling2: true, // makes v5 obsolete
            sibling3: true,
        })
        
        sibling4 := NewSibling(NewDVV(NewDot("r2", 7), map[string]uint64{ "r2": 6 }), []byte("v4"), 0)
        sibling5 := NewSibling(NewDVV(NewDot("r3", 1), map[string]uint64{ }), []byte("v5"), 0)
        
        siblingSet2 := NewSiblingSet(map[*Sibling]bool{
            sibling1: true,
            sibling4: true, // makes v3 obsolete
            sibling5: true,
        })

        BeforeEach(func() {
            readMerger.InsertKeyReplica(0, "b", siblingSet1)
            readMerger.InsertKeyReplica(0, "b", siblingSet2)
            readMerger.InsertKeyReplica(0, "c", nil)
            mergeIterator.AddKey("a", "b")
            mergeIterator.AddKey("a", "b")
            mergeIterator.AddKey("b", "c")
            mergeIterator.AddKey("b", "c")
            mergeIterator.AddKey("c", "d")
            mergeIterator.AddKey("c", "d")
        })

        It("Should return value as obtained by calling readMerger.Get() on the key corresponding to the current key returned by Key()", func() {
            Expect(mergeIterator.Next()).Should(BeTrue())
            Expect(mergeIterator.Value()).Should(Equal(siblingSet1.Sync(siblingSet2)))
            Expect(mergeIterator.Next()).Should(BeTrue())
            Expect(mergeIterator.Value()).Should(BeNil())
            Expect(mergeIterator.Next()).Should(BeTrue())
            Expect(mergeIterator.Value()).Should(BeNil())
            Expect(mergeIterator.Next()).Should(BeFalse())
            Expect(mergeIterator.Value()).Should(BeNil())
        })
    })

    Describe("#Release", func() {
    })

    Describe("#Error", func() {
        It("Should return nil", func() {
            Expect(mergeIterator.Error()).Should(BeNil())
        })
    })
})
