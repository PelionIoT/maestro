package util_test

import (
    "sync"
    
    . "github.com/armPelionEdge/devicedb/util"
    
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("Multilock", func() {
    Describe("lock and unlock", func() {
        It("should serialize goroutines locking with the same partitioning key but allow others to run in parallel", func() {
            var wg sync.WaitGroup
            
            countA := 0
            countB := 0
            
            multiLock := NewMultiLock()
            multiLock.Lock([]byte("AAA"))
                
            go func() {
                multiLock.Lock([]byte("AAA"))
                
                countA += 1
            }()
        
            wg.Add(1)
            
            go func() {
                multiLock.Lock([]byte("BBB"))
                
                for i := 0; i < 1000000; i += 1 {
                    countB += 1
                }
                
                multiLock.Unlock([]byte("BBB"))
                wg.Done()
            }()
            
            wg.Wait()
            
            Expect(countA).Should(Equal(0))
            Expect(countB).Should(Equal(1000000))
        })
    })
})
