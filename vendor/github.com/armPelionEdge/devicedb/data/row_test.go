package data_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    . "github.com/armPelionEdge/devicedb/data"
)

var _ = Describe("Row", func() {
    Describe("Decode", func() {
        Context("When format version is 0 and the data is encoded as a sibling set", func() {
            It("Should be decoded as a row where the siblings property matches the encoded sibling set", func() {
                siblingSet1 := NewSiblingSet(map[*Sibling]bool{
                    NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                    NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                    NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
                })

                var row Row
                encoded := siblingSet1.Encode()

                Expect(row.Decode(encoded, "0")).Should(BeNil())
                Expect(row.Siblings.Size()).Should(Equal(3))
            })
        })

        Context("When format version is not 0 and the data is encoded as a sibling set", func() {
            It("Should return an error", func() {
                siblingSet1 := NewSiblingSet(map[*Sibling]bool{
                    NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                    NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                    NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
                })

                var row Row
                encoded := siblingSet1.Encode()

                Expect(row.Decode(encoded, "1")).Should(Not(BeNil()))
            })
        })

        Context("When format version is 0 and the data is encoded as a row", func() {
            It("Should be decoded as a row", func() {
                siblingSet1 := NewSiblingSet(map[*Sibling]bool{
                    NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                    NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                    NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
                })

                var row Row = Row{
                    LocalVersion: 55,
                    Siblings: siblingSet1,
                }

                encoded := row.Encode()

                var newRow Row

                Expect(newRow.Decode(encoded, "0")).Should(BeNil())
                Expect(newRow.LocalVersion).Should(Equal(uint64(55)))
            })
        })

        Context("When format version is not 0 and the data is encoded as a row", func() {
            It("Should be properly decoded", func() {
                siblingSet1 := NewSiblingSet(map[*Sibling]bool{
                    NewSibling(NewDVV(NewDot("r1", 1), map[string]uint64{ "r2": 5, "r3": 2 }), []byte("v3"), 0): true,
                    NewSibling(NewDVV(NewDot("r1", 2), map[string]uint64{ "r2": 4, "r3": 3 }), nil, 0): true,
                    NewSibling(NewDVV(NewDot("r2", 6), map[string]uint64{ }), nil, 1): true,
                })

                var row Row = Row{
                    LocalVersion: 55,
                    Siblings: siblingSet1,
                }

                encoded := row.Encode()

                var newRow Row

                Expect(newRow.Decode(encoded, "1")).Should(BeNil())
                Expect(newRow.LocalVersion).Should(Equal(uint64(55)))
            })
        })
    })
})
