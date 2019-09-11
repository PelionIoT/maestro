package transfer_test

import (
    . "github.com/armPelionEdge/devicedb/transfer"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("Chunk", func() {
    Describe("#IsEmpty", func() {
        Context("When chunk.Entries is nil", func() {
            Specify("It should return true", func() {
                partitionChunk := PartitionChunk{
                    Entries: nil,
                }

                Expect(partitionChunk.IsEmpty()).Should(BeTrue())
            })
        })

        Context("When chunk.Entries is an empty array", func() {
            Specify("It should return true", func() {
                partitionChunk := PartitionChunk{
                    Entries: []Entry{ },
                }

                Expect(partitionChunk.IsEmpty()).Should(BeTrue())
            })
        })

        Context("When chunk.Entries is an array with at least one element", func() {
            Specify("It should return false", func() {
                partitionChunk := PartitionChunk{
                    Entries: []Entry{ Entry{ } },
                }

                Expect(partitionChunk.IsEmpty()).Should(BeFalse())
            })
        })
    })
})
