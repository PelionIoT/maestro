package clusterio_test

import (
    . "github.com/armPelionEdge/devicedb/clusterio"
    . "github.com/armPelionEdge/devicedb/data"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("ReadMerger", func() {
    Describe("#Patch", func() {
        Specify("For each key inserted it should return a corresponding entry in the patch", func() {
            readMerger := NewReadMerger("default")

            readMerger.InsertKeyReplica(0, "a", nil)
            readMerger.InsertKeyReplica(0, "b", nil)
            readMerger.InsertKeyReplica(0, "c", nil)
            readMerger.InsertKeyReplica(1, "d", nil)

            patch := readMerger.Patch(0)
            Expect(len(patch)).Should(Equal(4))
            _, ok := patch["a"]
            Expect(ok).Should(BeTrue())
            _, ok = patch["b"]
            Expect(ok).Should(BeTrue())
            _, ok = patch["c"]
            Expect(ok).Should(BeTrue())
            _, ok = patch["d"]
            Expect(ok).Should(BeTrue())

            patch = readMerger.Patch(1)
            Expect(len(patch)).Should(Equal(4))
            _, ok = patch["a"]
            Expect(ok).Should(BeTrue())
            _, ok = patch["b"]
            Expect(ok).Should(BeTrue())
            _, ok = patch["c"]
            Expect(ok).Should(BeTrue())
            _, ok = patch["d"]
            Expect(ok).Should(BeTrue())
        })

        Specify("For each key inserted the sibling set in the patch should be non-nil", func() {
            readMerger := NewReadMerger("default")

            readMerger.InsertKeyReplica(0, "a", nil)
            readMerger.InsertKeyReplica(0, "b", nil)
            readMerger.InsertKeyReplica(0, "c", nil)
            readMerger.InsertKeyReplica(1, "d", nil)

            patch := readMerger.Patch(0)
            Expect(patch["a"]).Should(Equal(NewSiblingSet(map[*Sibling]bool{ })))
            Expect(patch["b"]).Should(Equal(NewSiblingSet(map[*Sibling]bool{ })))
            Expect(patch["c"]).Should(Equal(NewSiblingSet(map[*Sibling]bool{ })))
            Expect(patch["d"]).Should(Equal(NewSiblingSet(map[*Sibling]bool{ })))

            patch = readMerger.Patch(1)
            Expect(patch["a"]).Should(Equal(NewSiblingSet(map[*Sibling]bool{ })))
            Expect(patch["b"]).Should(Equal(NewSiblingSet(map[*Sibling]bool{ })))
            Expect(patch["c"]).Should(Equal(NewSiblingSet(map[*Sibling]bool{ })))
            Expect(patch["d"]).Should(Equal(NewSiblingSet(map[*Sibling]bool{ })))
        })

        Specify("For each key applying the sibling set in the patch to the sibling set for that key at the specified node should make that key equal to the merged sibling set", func() {
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
            
            diff1 := NewSiblingSet(map[*Sibling]bool{
                sibling4: true,
            })

            diff2 := NewSiblingSet(map[*Sibling]bool{
                sibling2: true,
            })

            readMerger := NewReadMerger("default")

            readMerger.InsertKeyReplica(0, "a", siblingSet1)
            readMerger.InsertKeyReplica(1, "a", siblingSet2)

            patch := readMerger.Patch(0)
            Expect(patch["a"]).Should(Equal(diff1))
            Expect(siblingSet1.Sync(patch["a"])).Should(Equal(siblingSet1.Sync(siblingSet2)))
            patch = readMerger.Patch(1)
            Expect(patch["a"]).Should(Equal(diff2))
            Expect(siblingSet2.Sync(patch["a"])).Should(Equal(siblingSet1.Sync(siblingSet2)))
        })

        Specify("If a particular key is already equal to the merged sibling set at the specified node then the patch should be an empty set", func() {
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
            
            diff1 := NewSiblingSet(map[*Sibling]bool{
            })

            diff2 := NewSiblingSet(map[*Sibling]bool{
                sibling2: true,
            })

            readMerger := NewReadMerger("default")

            readMerger.InsertKeyReplica(0, "a", siblingSet1.Sync(siblingSet2))
            readMerger.InsertKeyReplica(1, "a", siblingSet2)

            patch := readMerger.Patch(0)
            Expect(patch["a"]).Should(Equal(diff1))
            Expect(siblingSet1.Sync(siblingSet2).Sync(patch["a"])).Should(Equal(siblingSet1.Sync(siblingSet2)))
            patch = readMerger.Patch(1)
            Expect(patch["a"]).Should(Equal(diff2))
            Expect(siblingSet2.Sync(patch["a"])).Should(Equal(siblingSet1.Sync(siblingSet2)))
        })
    })

    Describe("#Get", func() {
        Context("An instance of that key was never inserted", func() {
            It("Should return nil for that key", func() {
                readMerger := NewReadMerger("default")
                Expect(readMerger.Get("a")).Should(BeNil())
            })
        })

        Context("All instances of that key were nil", func() {
            It("Should return nil for that key", func() {
                readMerger := NewReadMerger("default")
                readMerger.InsertKeyReplica(0, "a", nil)
                readMerger.InsertKeyReplica(1, "b", nil)
                readMerger.InsertKeyReplica(2, "c", nil)
                Expect(readMerger.Get("a")).Should(BeNil())
            })
        })

        Context("At least one instance of that key were non-nil", func() {
            It("Should return the sibilng set resulting from merging all inserted sets for that key", func() {
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
                
                readMerger := NewReadMerger("default")

                readMerger.InsertKeyReplica(0, "a", siblingSet1)
                readMerger.InsertKeyReplica(1, "a", siblingSet2)
                readMerger.InsertKeyReplica(2, "a", nil)
                Expect(readMerger.Get("a")).Should(Equal(siblingSet1.Sync(siblingSet2)))
            })
        })

        Context("The bucket used is \"lww\"", func() {
            It("Should resolve the set to only one sibling leaving the last written value based on timestamp", func() {
                sibling1 := NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v1"), 100)
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
                
                readMerger := NewReadMerger("lww")

                readMerger.InsertKeyReplica(0, "a", siblingSet1)
                readMerger.InsertKeyReplica(1, "a", siblingSet2)
                readMerger.InsertKeyReplica(2, "a", nil)
                Expect(readMerger.Get("a")).Should(Equal(NewSiblingSet(map[*Sibling]bool{ sibling1: true })))
            })
        })
    })

    Describe("#Nodes", func() {
        It("Should return a set of nodes involved in the read", func() {
            readMerger := NewReadMerger("default")
            readMerger.InsertKeyReplica(0, "a", nil)
            readMerger.InsertKeyReplica(1, "b", nil)
            readMerger.InsertKeyReplica(2, "c", nil)

            Expect(readMerger.Nodes()).Should(Equal(map[uint64]bool{
                0: true,
                1: true,
                2: true,
            }))
        })
    })
})
