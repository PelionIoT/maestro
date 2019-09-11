package merkle_test

import (
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/merkle"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
    
    "unsafe"
    "math"
    "math/big"
    "strconv"
    "testing"
    "fmt"
    "strings"
)

func combination(n, k int64) *big.Rat {
    c := big.NewRat(1, 1)
    
    for i := n - k + 1; i <= n; i += 1 {
        c.Mul(c, big.NewRat(i, 1))
    }
    
    for i := int64(1); i <= k; i += 1 {
        c.Mul(c, big.NewRat(1, i))
    }
    
    return c
}

func binomialPDF(k, n uint32, p float64) float64 {
    t1 := big.NewFloat(math.Pow(p, float64(k)))
    t2 := big.NewFloat(math.Pow(float64(1 - p), float64(n - k)))
    var t3 big.Float
    t3.SetRat(combination(int64(n), int64(k)))
    
    var r big.Float
    
    r.Mul(t1, t2)
    r.Mul(&r, &t3)
    
    result, _ := r.Float64()
    
    return result
}

var _ = Describe("Merkle", func() {
    Describe("#LeafNode", func() {
        It("Should return the leaf node corresponding to a certain hash", func() {
            const depth uint8 = 10
            
            for i := 0; i < (1 << depth); i++ {
                var hash Hash = Hash{ }.SetHigh(uint64(i)).SetLow(0)
                var shiftAmount uint8 = uint8(unsafe.Sizeof(hash.High()))*8 - depth
                
                hash = hash.SetHigh(hash.High() << shiftAmount)
                
                Expect(LeafNode(&hash, depth)).Should(Equal(uint32(i | 0x1)))
            }
            
            var hash Hash = Hash{ }.SetHigh(math.MaxUint64).SetLow(0)
            Expect(LeafNode(&hash, depth)).Should(Equal(uint32((1 << depth) - 1)))
        })
    })
    
    Describe("#RootNode", func() {
        It("Should return the root node of the tree", func() {
            const depth uint8 = 4
            merkleTree, _ := NewMerkleTree(depth)
            
            Expect(merkleTree.RootNode()).Should(Equal(uint32(8)))
            Expect(merkleTree.LeftChild(8)).Should(Equal(uint32(4)))
            Expect(merkleTree.RightChild(8)).Should(Equal(uint32(12)))
            Expect(merkleTree.LeftChild(2)).Should(Equal(uint32(1)))
            Expect(merkleTree.RightChild(2)).Should(Equal(uint32(3)))
            Expect(merkleTree.LeftChild(1)).Should(Equal(uint32(1)))
            Expect(merkleTree.RightChild(1)).Should(Equal(uint32(1)))
        })
    })
    
    Describe("#ParentNode", func() {
        It("Should return the leaf node corresponding to a certain hash", func() {
            type nodeRange struct {
                min uint32
                max uint32
                parent uint32
            }
            
            const depth uint8 = 10
            
            nodeQueue := []nodeRange{ nodeRange{ 0, 1 << depth, 1 << depth } }
            
            for len(nodeQueue) > 0 {
                next := nodeQueue[0]
                nodeQueue = nodeQueue[1:]
                node := next.min + (next.max - next.min)/2
            
                Expect(ParentNode(node)).Should(Equal(next.parent))
                
                if next.max - next.min == 2 {
                    continue
                }
                
                nodeQueue = append(nodeQueue, nodeRange{ next.min, node, node })
                nodeQueue = append(nodeQueue, nodeRange{ node, next.max, node })
            }
        })
    })
    
    Describe("#Update", func() {
        It("Root hash should be the xor sum of the hashes of all sibling sets added to the tree", func() {
            const depth uint8 = 19
            const objectCount int = (1 << (depth+2))
            
            merkleTree, _ := NewMerkleTree(depth)
            
            siblingSet := NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), []byte("v8"), 1): true,
            })
            
            expectedHash := Hash{}
            histogram := make(map[uint32]uint32)
            
            for i := 1; i < 1 << depth; i += 2 {
                histogram[uint32(i)] = 0
            }
            
            for i := 0; i < objectCount; i += 1 {
                expectedHash = expectedHash.Xor(siblingSet.Hash([]byte("key"+strconv.Itoa(i))))
                _, objectHashes := merkleTree.Update(NewUpdate().AddDiff("key"+strconv.Itoa(i), nil, siblingSet))
            
                for leaf, _ := range objectHashes {
                    histogram[leaf] += 1
                }
            }
            
            Expect(merkleTree.RootHash()).Should(Equal(expectedHash))
    
            countDistribution := make(map[uint32]uint32)
            minCount := uint32(math.MaxUint32)
            maxCount := uint32(0)
            
            for i := 1; i < 1 << depth; i += 2 {
                count := histogram[uint32(i)]
                
                if minCount > count {
                    minCount = count
                }
                
                if maxCount < count {
                    maxCount = count
                }
                
                if _, ok := countDistribution[count]; !ok {
                    countDistribution[count] = 0
                }
                
                countDistribution[count] += 1
            }
            
            
            mean := float64(objectCount) / float64(1 << (depth - 1))
            variance := float64(0)
            cost := float64(0)
            
            fmt.Println()
            fmt.Println("[ACTUAL DISTRIBUTION]")
            
            for i := minCount; i <= maxCount; i += 1 {
                leafCount := uint32((1 << (depth - 1)))
                
                if _, ok := countDistribution[i]; !ok {
                    countDistribution[i] = 0
                }
                
                //countDistribution[i] = 100*countDistribution[i]/leafCount
                variance += (float64(countDistribution[i])/float64(leafCount))*math.Pow(float64(i) - float64(mean), 2.0)
                p := binomialPDF(i, uint32(objectCount), float64(1)/float64(1 << (depth - 1)))
                
                cost += math.Abs(p - float64(countDistribution[i])/float64(leafCount))
                
                fmt.Printf("%02d (%0.04f): %s\n", int(i), float64(countDistribution[i])/float64(leafCount), strings.Repeat("*", int(100*countDistribution[i]/leafCount)))
            }
            
            fmt.Println("[BINOMIAL DISTRIBUTION]")
            for i := minCount; i <= maxCount; i += 1 {
                p := binomialPDF(i, uint32(objectCount), float64(1)/float64(1 << (depth - 1)))
                
                fmt.Printf("%02d (%0.04f): %s\n", int(i), p, strings.Repeat("*", int(100*p)))
            }
        
            fmt.Printf("N: %d\n", objectCount)
            fmt.Printf("Mean: %f\n", mean)
            fmt.Printf("Standard Deviation: %f\n", math.Sqrt(variance))
            fmt.Printf("Cost: %04f\n", cost)
            // this is fragile and can be commented out. It's here to illustrate that
            // the count distribution should closely follow a binomial distribution
            Expect(cost).Should(BeNumerically("~", 0.0, 0.009))
        })
    })

    Describe("#PreviewUpdate", func() {
        It("PreviewUpdate should report leaf hashes that are the same as what happens when update is called", func() {
            const depth uint8 = 19
            const objectCount int = 100
            
            merkleTree, _ := NewMerkleTree(depth)
            
            siblingSet := NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), []byte("v8"), 1): true,
            })

            update := NewUpdate()
            
            for i := 0; i < objectCount; i += 1 {
                update.AddDiff("key"+strconv.Itoa(i), nil, siblingSet)
            }
            
            leafHashes, previewObjectHashes := merkleTree.PreviewUpdate(update)
            modifiedNodes, objectHashes := merkleTree.Update(update)

            Expect(previewObjectHashes).Should(Equal(objectHashes))

            for leafID, leafHash := range leafHashes {
                _, ok := modifiedNodes[leafID]

                Expect(ok).Should(BeTrue())
                Expect(merkleTree.NodeHash(leafID)).Should(Equal(leafHash))
            }
        })
    })
    
    Describe("#UndoUpdate", func() {
        It("UndoUpdate should restore merkle tree state to what it was before an Update", func() {
            const depth uint8 = 19
            const objectCount int = 100
            
            merkleTree, _ := NewMerkleTree(depth)
            
            siblingSet := NewSiblingSet(map[*Sibling]bool{
                NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), []byte("v8"), 1): true,
            })

            update1 := NewUpdate()
            update2 := NewUpdate()
            update3 := NewUpdate()
            
            for i := 0; i < objectCount; i += 1 {
                update1.AddDiff("key"+strconv.Itoa(i), nil, siblingSet)
            }

            for i := 0; i < objectCount; i += 1 {
                update2.AddDiff("key"+strconv.Itoa(i), nil, siblingSet)
            }
            
            for i := 0; i < objectCount; i += 1 {
                update3.AddDiff("key"+strconv.Itoa(i), nil, siblingSet)
            }

            merkleTree.Update(update1)
            merkleTree.Update(update2)
            merkleTree.Update(update3)

            merkleTree.UndoUpdate(update1)
            merkleTree.UndoUpdate(update3)
            merkleTree.UndoUpdate(update2)

            for i := uint32(0); i < merkleTree.NodeLimit(); i += 1 {
                Expect(merkleTree.NodeHash(i).High()).Should(Equal(uint64(0)))
                Expect(merkleTree.NodeHash(i).Low()).Should(Equal(uint64(0)))
            }
        })
    })
})

func BenchmarkMerkleUpdate(b *testing.B) {
    const depth uint8 = 10
            
    merkleTree, _ := NewMerkleTree(depth)
    siblingSet := NewSiblingSet(map[*Sibling]bool{
        NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
        NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
        NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
    })

    update := NewUpdate().AddDiff("keyA", nil, siblingSet)
    
    for i := 0; i < b.N; i += 1 {
        merkleTree.Update(update)
    }
}