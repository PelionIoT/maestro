package data_test

import (
    . "github.com/armPelionEdge/devicedb/data"
    
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
    
    "sort"
)

var _ = Describe("Dvv", func() {
    Describe("#HappenedBefore", func() {
        It("should return true if and only if the dot number is less than or equal to the corresponding entry in the other dvvs clock", func() {
            clock1 := NewDVV(NewDot("r1", 1), map[string]uint64{ })
            clock2 := NewDVV(NewDot("r2", 1), map[string]uint64{ "r1": 1 })
            
            Expect(clock1.HappenedBefore(clock2)).Should(BeTrue())  
            Expect(clock2.HappenedBefore(clock1)).Should(BeFalse())
            
            clock1 = NewDVV(NewDot("r1", 1), map[string]uint64{ })
            clock2 = NewDVV(NewDot("r2", 2), map[string]uint64{ })
            
            Expect(clock1.HappenedBefore(clock2)).Should(BeFalse())
            Expect(clock2.HappenedBefore(clock1)).Should(BeFalse())
        })
    })
    
    Describe("#Replicas", func() {
        It("should return a set of replica names contained in the clock", func() {
            clock1 := NewDVV(NewDot("r1", 1), map[string]uint64{ })
            clock2 := NewDVV(NewDot("r1", 2), map[string]uint64{ "r1": 1, "r2": 1 })
            
            replicas1 := clock1.Replicas()
            replicas2 := clock2.Replicas()
            
            sort.Strings(replicas1)
            sort.Strings(replicas2)
            
            Expect(replicas1).Should(Equal([]string{ "r1" }))
            Expect(replicas2).Should(Equal([]string{ "r1", "r2" }))
        })
    })
    
    Describe("#MaxDot", func() {
        It("should return the max known integer representing the latest known event from a given replica according to a dvv clock", func() {
            clock1 := NewDVV(NewDot("r1", 1), map[string]uint64{ })
            clock2 := NewDVV(NewDot("r1", 2), map[string]uint64{ "r1": 1, "r2": 1 })
            clock3 := NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 3, "r3": 4 })
            
            Expect(clock1.MaxDot("r1")).Should(Equal(uint64(1)))
            Expect(clock2.MaxDot("r1")).Should(Equal(uint64(2)))
            Expect(clock2.MaxDot("r2")).Should(Equal(uint64(1)))
            Expect(clock3.MaxDot("r1")).Should(Equal(uint64(1)))
            Expect(clock3.MaxDot("r2")).Should(Equal(uint64(3)))
            Expect(clock3.MaxDot("r3")).Should(Equal(uint64(4)))
        })
        
        It("should return 0 if the given replica is not represented in the dvv clock", func() {
            clock1 := NewDVV(NewDot("r1", 1), map[string]uint64{ })
            clock2 := NewDVV(NewDot("r1", 2), map[string]uint64{ "r1": 1, "r2": 1 })
            clock3 := NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 3, "r3": 4 })
            
            Expect(clock1.MaxDot("r5")).Should(Equal(uint64(0)))
            Expect(clock2.MaxDot("r5")).Should(Equal(uint64(0)))
            Expect(clock3.MaxDot("r5")).Should(Equal(uint64(0)))
        })
    })
})
