package data_test

import (
    "reflect"
    
    . "github.com/armPelionEdge/devicedb/data"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("SiblingSet", func() {
    Describe("#Sync", func() {
        It("should return a new sibling set that removes obsolete siblings by combining two sibling sets", func() {
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
                sibling4: true, // makes v3 obsolete
                sibling5: true,
            })
            
            syncedSet := NewSiblingSet(map[*Sibling]bool{
                sibling1: true,
                sibling2: true,
                sibling4: true,
            })
            
            Expect(siblingSet1.Sync(siblingSet2)).Should(Equal(syncedSet))
            Expect(siblingSet2.Sync(siblingSet1)).Should(Equal(syncedSet))
        })

        It("Should resolve the situation where two siblings have the same clock but different values", func() {
            sibling1 := NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v1"), 0)
            sibling2 := NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v2"), 0)
            
            siblingSet1 := NewSiblingSet(map[*Sibling]bool{
                sibling1: true,
            })

            siblingSet2 := NewSiblingSet(map[*Sibling]bool{
                sibling2: true,
            })
            
            Expect(siblingSet1.Sync(siblingSet2)).Should(Equal(siblingSet2.Sync(siblingSet1)))
            Expect(sibling1.Compare(sibling2)).Should(Equal(-1))
            Expect(siblingSet1.Sync(siblingSet2)).Should(Equal(NewSiblingSet(map[*Sibling]bool{
                sibling2: true,
            })))            
        })
    })

    Describe("#MergeSync", func() {
        It("should return a new sibling set that removes obsolete siblings by combining two sibling sets", func() {
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
                sibling4: true, // makes v3 obsolete
                sibling5: true,
            })
            
            syncedSet := NewSiblingSet(map[*Sibling]bool{
                sibling1: true,
                sibling2: true,
                sibling4: true,
            })
            
            Expect(siblingSet1.MergeSync(siblingSet2, "r1")).Should(Equal(syncedSet))
            Expect(siblingSet2.MergeSync(siblingSet1, "r1")).Should(Equal(syncedSet))
        })

        It("Should generate a new event", func() {
            sibling1 := NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ }), []byte("v1"), 0)
            sibling2 := NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r1": 1 }), []byte("v2"), 0)
            
            siblingSetR1 := NewSiblingSet(map[*Sibling]bool{
                sibling1: true,
            })

            siblingSetR2 := NewSiblingSet(map[*Sibling]bool{
                sibling2: true,
            })
            
            syncedSet1 := siblingSetR1.MergeSync(siblingSetR2, "r1")
            syncedSet2 := siblingSetR2.MergeSync(siblingSetR1, "r1")
            syncedSet3 := siblingSetR1.MergeSync(siblingSetR2, "r2")
            syncedSet4 := siblingSetR2.MergeSync(siblingSetR1, "r2")
            
            var m map[string]*Sibling = make(map[string]*Sibling)

            for sibling := range syncedSet1.Iter() {
                if _, ok := m[string(sibling.Value())]; ok {
                    Fail("Duplicate version")
                }

                m[string(sibling.Value())] = sibling
            }

            Expect(m["v1"]).Should(Equal(NewSibling(NewDVV(NewDot("r1", 3), map[string]uint64{ }), []byte("v1"), 0)))
            Expect(m["v2"]).Should(Equal(sibling2))
            Expect(len(m)).Should(Equal(2))
            
            Expect(syncedSet3).Should(Equal(NewSiblingSet(map[*Sibling]bool{
                sibling2: true,
            })))
            Expect(syncedSet4).Should(Equal(syncedSet3))
            Expect(syncedSet2).Should(Equal(syncedSet3))

            Expect(syncedSet1.MergeSync(syncedSet3, "r1")).Should(Equal(syncedSet1))
            Expect(syncedSet3.MergeSync(syncedSet1, "r1")).Should(Equal(syncedSet1))
            Expect(syncedSet1.MergeSync(syncedSet3, "r2")).Should(Equal(syncedSet1))
            Expect(syncedSet3.MergeSync(syncedSet1, "r2")).Should(Equal(syncedSet1))
        })
    })

    Describe("#Diff", func() {
        It("should return a new sibling set that includes only siblings from the other sibling set that are not included in the sibling set, ", func() {
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
            
            Expect(siblingSet1.Diff(siblingSet2)).Should(Equal(diff1))
            Expect(siblingSet2.Diff(siblingSet1)).Should(Equal(diff2))
            // Applying a diff to a set should make it equal the result of merging those sets
            Expect(siblingSet2.Sync(siblingSet2.Diff(siblingSet1))).Should(Equal(siblingSet1.Sync(siblingSet2)))
            Expect(siblingSet1.Sync(siblingSet1.Diff(siblingSet2))).Should(Equal(siblingSet1.Sync(siblingSet2)))
            Expect(siblingSet2.Sync(siblingSet2.Diff(siblingSet1.Sync(siblingSet2)))).Should(Equal(siblingSet1.Sync(siblingSet2)))
            Expect(siblingSet1.Sync(siblingSet1.Diff(siblingSet1.Sync(siblingSet2)))).Should(Equal(siblingSet1.Sync(siblingSet2)))
            Expect(siblingSet1.Sync(siblingSet2).Diff(siblingSet1)).Should(Equal(NewSiblingSet(map[*Sibling]bool{ })))
            Expect(siblingSet1.Sync(siblingSet2).Diff(siblingSet2)).Should(Equal(NewSiblingSet(map[*Sibling]bool{ })))
        })
    })
    
    Describe("#Join", func() {
        It("should return a clock that summarizes the total causal context of a giving sibling set", func() {
            siblingSet1 := NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v1"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), []byte("v2"), 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), []byte("v3"), 0): true,
            })
            
            siblingSet2 := NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r2", 7), map[string]uint64{ "r2": 6 }), []byte("v4"), 0): true,
                NewSibling(NewDVV(NewDot("r3", 1), map[string]uint64{ }), []byte("v5"), 0): true,
            })
            
            causalContext1 := map[string]uint64{
                "r1": 2,
                "r2": 6,
                "r3": 3,
            }
            
            causalContext2 := map[string]uint64{
                "r2": 7,
                "r3": 1,
            }
            
            syncedContext := map[string]uint64{
                "r1": 2,
                "r2": 7,
                "r3": 3,
            }
            
            Expect(siblingSet1.Join()).Should(Equal(causalContext1))
            Expect(siblingSet2.Join()).Should(Equal(causalContext2))
            Expect(siblingSet1.Sync(siblingSet2).Join()).Should(Equal(syncedContext))
        })
    })
    
    Describe("#Discard", func() {
        It("should discard all siblings from a sibling set that are made obsolete given a certain context specified by a dvv clock", func() {
            sibling1 := NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v1"), 0)
            sibling2 := NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), []byte("v2"), 0)
            sibling3 := NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), []byte("v3"), 0)
            
            siblingSet1 := NewSiblingSet(map[*Sibling]bool{
                sibling1: true,
                sibling2: true,
                sibling3: true,
            })
            
            afterNoneContext := NewDVV(NewDot("r3", 33), map[string]uint64{ })
            afterAllContext := NewDVV(NewDot("r3", 23), map[string]uint64{ "r1": 2, "r2": 6, "r3": 3 })
            afterSomeContext := NewDVV(NewDot("r2", 7), map[string]uint64{ "r1": 1, "r2": 5, "r3": 2 })
            
            afterNoneDiscardSet := NewSiblingSet(map[*Sibling]bool{
                sibling1: true,
                sibling2: true,
                sibling3: true,
            })
            
            afterAllDiscardSet := NewSiblingSet(map[*Sibling]bool{ })
            
            afterSomeDiscardSet := NewSiblingSet(map[*Sibling]bool{
                sibling2: true,
                sibling3: true,
            })
            
            Expect(siblingSet1.Discard(afterNoneContext)).Should(Equal(afterNoneDiscardSet))
            Expect(siblingSet1.Discard(afterAllContext)).Should(Equal(afterAllDiscardSet))
            Expect(siblingSet1.Discard(afterSomeContext)).Should(Equal(afterSomeDiscardSet))
        })
    })
    
    Describe("#Event", func() {
        It("should create a new dvv clock where the dot for the given replica is the max of all the dots for that replica in the causal history of all the siblings + 1", func() {
            siblingSet1 := NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v1"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), []byte("v2"), 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), []byte("v3"), 0): true,
            })
            
            afterNoneContext := map[string]uint64{ }
            afterAllContext := map[string]uint64{ "r1": 2, "r2": 6, "r3": 23 }
            afterSomeContext := map[string]uint64{ "r1": 1, "r2": 5, "r3": 2 }
            
            Expect(siblingSet1.Event(afterNoneContext, "r1")).Should(Equal(NewDVV(NewDot("r1", 3), afterNoneContext)))
            Expect(siblingSet1.Event(afterNoneContext, "r2")).Should(Equal(NewDVV(NewDot("r2", 7), afterNoneContext)))
            Expect(siblingSet1.Event(afterNoneContext, "r3")).Should(Equal(NewDVV(NewDot("r3", 4), afterNoneContext)))
            Expect(siblingSet1.Event(afterAllContext, "r1")).Should(Equal(NewDVV(NewDot("r1", 3), afterAllContext)))
            Expect(siblingSet1.Event(afterAllContext, "r2")).Should(Equal(NewDVV(NewDot("r2", 7), afterAllContext)))
            Expect(siblingSet1.Event(afterAllContext, "r3")).Should(Equal(NewDVV(NewDot("r3", 24), afterAllContext)))
            Expect(siblingSet1.Event(afterSomeContext, "r1")).Should(Equal(NewDVV(NewDot("r1", 3), afterSomeContext)))
            Expect(siblingSet1.Event(afterSomeContext, "r2")).Should(Equal(NewDVV(NewDot("r2", 7), afterSomeContext)))
            Expect(siblingSet1.Event(afterSomeContext, "r3")).Should(Equal(NewDVV(NewDot("r3", 4), afterSomeContext)))
        })
    })
    
    Describe("#IsTombstoneSet", func() {
        It("should return true if the sibling set is empty", func() {
            siblingSet := NewSiblingSet(map[*Sibling]bool{ })
            
            Expect(siblingSet.IsTombstoneSet()).Should(BeTrue())
        })
        
        It("should return true if all siblings are tombstones", func() {
            siblingSet := NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 0): true,
            })
            
            Expect(siblingSet.IsTombstoneSet()).Should(BeTrue())
        })
        
        It("should return false if one of the siblings is not a tombstone", func() {
            siblingSet := NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), []byte("v1"), 0): true,
            })
            
            Expect(siblingSet.IsTombstoneSet()).Should(BeFalse())
        })
    })
    
    Describe("#GetOldestTombstone", func() {
        It("should return nil if the sibling set is empty", func() {
            siblingSet := NewSiblingSet(map[*Sibling]bool{ })
            
            Expect(siblingSet.GetOldestTombstone()).Should(BeNil())
        })
        
        It("should return nil if all siblings in the sibling set are non-tombstones", func() {
            siblingSet := NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), []byte("v2"), 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), []byte("v1"), 0): true,
            })
            
            Expect(siblingSet.GetOldestTombstone()).Should(BeNil())
        })
        
        It("should return the tombstone sibling with the smallest timestamp if there is exactly one tombstone sibling", func() {
            siblingSet := NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), []byte("v2"), 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            Expect(siblingSet.GetOldestTombstone()).Should(Equal(NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1)))
        })
        
        It("should return the tombstone sibling with the smallest timestamp if there is more than one tombstone sibling", func() {
            siblingSet := NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            Expect(siblingSet.GetOldestTombstone()).Should(Equal(NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0)))
        })
    })
    
    Describe("#Iter", func() {
        It("should allow iteration through the whole sibling set", func() {
            expected := map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            }
            
            siblingSet := NewSiblingSet(expected)
        
            count := 0
            
            for sibling := range siblingSet.Iter() {
                Expect(expected[sibling]).Should(BeTrue())
                count += 1
            }
            
            Expect(count).Should(Equal(len(expected)))
        })
    })
    
    Describe("#Encode+#Decode", func() {
        It("should encode the sibling set such that it can be decoded to an equivalent set", func() {
            var decodedSiblingSet SiblingSet
            var originalSiblingSet *SiblingSet = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            (&decodedSiblingSet).Decode(originalSiblingSet.Encode())
            
            Expect(decodedSiblingSet.Size()).Should(Equal(3))
            Expect(originalSiblingSet.Size()).Should(Equal(3))
            
            for decodedSibling := range decodedSiblingSet.Iter() {
                for originalSibling := range originalSiblingSet.Iter() {
                    if reflect.DeepEqual(decodedSibling, originalSibling) {
                        decodedSiblingSet.Delete(decodedSibling)
                        originalSiblingSet.Delete(originalSibling)
                    }
                }
            }
            
            Expect(decodedSiblingSet.Size()).Should(Equal(0))
            Expect(originalSiblingSet.Size()).Should(Equal(0))
        })
        
        It("should encode the sibling set such that it can be decoded to an equivalent set if the set is empty", func() {
            var decodedSiblingSet SiblingSet
            var originalSiblingSet *SiblingSet = NewSiblingSet(map[*Sibling]bool{ })
            
            (&decodedSiblingSet).Decode(originalSiblingSet.Encode())
            
            Expect(decodedSiblingSet.Size()).Should(Equal(0))
            Expect(originalSiblingSet.Size()).Should(Equal(0))
            
            for decodedSibling := range decodedSiblingSet.Iter() {
                for originalSibling := range originalSiblingSet.Iter() {
                    if reflect.DeepEqual(decodedSibling, originalSibling) {
                        decodedSiblingSet.Delete(decodedSibling)
                        originalSiblingSet.Delete(originalSibling)
                    }
                }
            }
            
            Expect(decodedSiblingSet.Size()).Should(Equal(0))
            Expect(originalSiblingSet.Size()).Should(Equal(0))
        })
    })
    
    Describe("#MarshalJSON+#UnmarshalJSON", func() {
        It("should encode the sibling set such that it can be decoded to an equivalent set", func() {
            var decodedSiblingSet SiblingSet
            var originalSiblingSet *SiblingSet = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            json, _ := originalSiblingSet.MarshalJSON()
            
            (&decodedSiblingSet).UnmarshalJSON(json)
            
            Expect(decodedSiblingSet.Size()).Should(Equal(3))
            Expect(originalSiblingSet.Size()).Should(Equal(3))
            
            for decodedSibling := range decodedSiblingSet.Iter() {
                for originalSibling := range originalSiblingSet.Iter() {
                    if reflect.DeepEqual(decodedSibling, originalSibling) {
                        decodedSiblingSet.Delete(decodedSibling)
                        originalSiblingSet.Delete(originalSibling)
                    }
                }
            }
            
            Expect(decodedSiblingSet.Size()).Should(Equal(0))
            Expect(originalSiblingSet.Size()).Should(Equal(0))
        })
        
        It("should encode the sibling set such that it can be decoded to an equivalent set if the set is empty", func() {
            var decodedSiblingSet SiblingSet
            var originalSiblingSet *SiblingSet = NewSiblingSet(map[*Sibling]bool{ })
            
            json, _ := originalSiblingSet.MarshalJSON()
            
            (&decodedSiblingSet).UnmarshalJSON(json)
            
            Expect(decodedSiblingSet.Size()).Should(Equal(0))
            Expect(originalSiblingSet.Size()).Should(Equal(0))
            
            for decodedSibling := range decodedSiblingSet.Iter() {
                for originalSibling := range originalSiblingSet.Iter() {
                    if reflect.DeepEqual(decodedSibling, originalSibling) {
                        decodedSiblingSet.Delete(decodedSibling)
                        originalSiblingSet.Delete(originalSibling)
                    }
                }
            }
            
            Expect(decodedSiblingSet.Size()).Should(Equal(0))
            Expect(originalSiblingSet.Size()).Should(Equal(0))
        })
    })
    
    Describe("#Hash", func() {
        It("should equal zero if the sibling set is a tombstone set", func() {
            siblingSet1 := NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            siblingSet2 := NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            var siblingSet3 *SiblingSet
            
            Expect(siblingSet1.Hash([]byte("keyA"))).Should(Equal(siblingSet2.Hash([]byte("keyA"))))
            Expect(siblingSet1.Hash([]byte("keyA"))).Should(Equal(siblingSet2.Hash([]byte("keyB"))))
            Expect(siblingSet1.Hash([]byte("keyA")).Bytes()).Should(Equal([16]byte{ 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0 }))
            Expect(siblingSet2.Hash([]byte("keyA")).Bytes()).Should(Equal([16]byte{ 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0 }))
            Expect(siblingSet3.Hash([]byte("keyA")).Bytes()).Should(Equal([16]byte{ 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0 }))
        })
        
        It("should produce the same hash value for sibling sets that are equivalent given equivalent keys", func() {
            var siblingSet1 *SiblingSet
            var siblingSet2 *SiblingSet
            
            siblingSet1 = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            siblingSet2 = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            Expect(siblingSet1.Hash([]byte("keyA"))).Should(Equal(siblingSet2.Hash([]byte("keyA"))))
            
            siblingSet1 = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            siblingSet2 = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
            })
            
            Expect(siblingSet1.Hash([]byte("keyA"))).Should(Equal(siblingSet2.Hash([]byte("keyA"))))
        })
        
        It("should produce different hashes for sibling sets that are equivalent except for different keys", func() {
            var siblingSet1 *SiblingSet
            var siblingSet2 *SiblingSet
            
            siblingSet1 = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            siblingSet2 = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            Expect(siblingSet1.Hash([]byte("keyA"))).Should(Not(Equal(siblingSet2.Hash([]byte("keyB")))))
            
            siblingSet1 = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            siblingSet2 = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
            })
            
            Expect(siblingSet1.Hash([]byte("keyA"))).Should(Not(Equal(siblingSet2.Hash([]byte("keyB")))))
        })
        
        It("should produce different hashes for sibling sets that have different contents and the same keys", func() {
            var siblingSet1 *SiblingSet
            var siblingSet2 *SiblingSet
            
            siblingSet1 = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 2 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            siblingSet2 = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 3 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 2 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            Expect(siblingSet1.Hash([]byte("keyA"))).Should(Not(Equal(siblingSet2.Hash([]byte("keyA")))))
            
            siblingSet1 = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            siblingSet2 = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), []byte("asdf"), 0): true,
            })
            
            Expect(siblingSet1.Hash([]byte("keyA"))).Should(Not(Equal(siblingSet2.Hash([]byte("keyA")))))
            
            siblingSet1 = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
            })
            
            siblingSet2 = NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2, "r4": 1 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
            })
            
            Expect(siblingSet1.Hash([]byte("keyA"))).Should(Not(Equal(siblingSet2.Hash([]byte("keyA")))))
        })
    })
})
