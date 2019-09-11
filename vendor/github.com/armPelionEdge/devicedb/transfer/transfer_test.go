package transfer_test

import (
    "io"
    "errors"
    "strings"
    "encoding/json"

    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/transfer"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("Transfer", func() {
    Describe("IncomingTransfer", func() {
        Describe("#NextChunk", func() {
            Context("When the reader encounters an error", func() {
                It("should return an empty chunk and that error", func() {
                    incomingTransfer := NewIncomingTransfer(NewMockErrorReader([]error{ errors.New("Some error") }))

                    nextChunk, err := incomingTransfer.NextChunk()

                    Expect(nextChunk.IsEmpty()).Should(BeTrue())
                    Expect(err).Should(Equal(errors.New("Some error")))
                })
            })

            Context("When the reader is out of data", func() {
                // This is really just a special case of the spec above but it is worth
                // exploring and stating explicitly
                It("should return an empty chunk and an io.EOF error", func() {
                    // Create a reader for an empty string so there is no data to return
                    incomingTransfer := NewIncomingTransfer(strings.NewReader(""))

                    nextChunk, err := incomingTransfer.NextChunk()

                    Expect(nextChunk.IsEmpty()).Should(BeTrue())
                    Expect(err).Should(Equal(io.EOF))
                })
            })

            Context("When a chunk is not properly formatted", func() {
                It("should return an empty chunk and that error", func() {
                    chunk := PartitionChunk{}
                    encodedChunk, _ := json.Marshal(chunk)
                    // Create a stream with a valid chunk followed by a token that cannot be parsed as a chunk
                    incomingTransfer := NewIncomingTransfer(strings.NewReader(string(encodedChunk) + "\n" + "sfasdfasdfa"))

                    _, err := incomingTransfer.NextChunk()
                    Expect(err).Should(BeNil())
                    _, err = incomingTransfer.NextChunk()
                    Expect(err).Should(Not(BeNil()))
                })
            })

            Context("When it has already encountered an error on a previous call", func() {
                It("should return an empty chunk and that same error on any future calls", func() {
                    incomingTransfer := NewIncomingTransfer(NewMockErrorReader([]error{ errors.New("Some error 1"), errors.New("Some error 2") }))

                    nextChunk, err := incomingTransfer.NextChunk()

                    Expect(nextChunk.IsEmpty()).Should(BeTrue())
                    Expect(err).Should(Equal(errors.New("Some error 1")))

                    nextChunk, err = incomingTransfer.NextChunk()

                    Expect(nextChunk.IsEmpty()).Should(BeTrue())
                    Expect(err).Should(Equal(errors.New("Some error 1")))
                })
            })

            Context("When Cancel has already been called", func() {
                It("should return ETransferCancelled on all subsequent calls", func() {
                    chunk := PartitionChunk{}
                    encodedChunk, _ := json.Marshal(chunk)
                    // Create a stream with a valid chunk followed by a token that cannot be parsed as a chunk
                    incomingTransfer := NewIncomingTransfer(strings.NewReader(string(encodedChunk)))

                    incomingTransfer.Cancel()
                    nextChunk, err := incomingTransfer.NextChunk()
                    Expect(nextChunk.IsEmpty()).Should(BeTrue())
                    Expect(err).Should(Equal(ETransferCancelled))
                    nextChunk, err = incomingTransfer.NextChunk()
                    Expect(nextChunk.IsEmpty()).Should(BeTrue())
                    Expect(err).Should(Equal(ETransferCancelled))
                })
            })

            Context("When the next chunk is parsed correctly but its checksum is not valid", func() {
                It("should return EEntryChecksum", func() {
                    sibling := NewSibling(NewDVV(NewDot("A", 0), map[string]uint64{ }), []byte{ }, 0)
                    chunk := PartitionChunk{
                        Index: 1,
                        Checksum: Hash{ },
                        Entries: []Entry{
                            Entry{
                                Site: "site1",
                                Bucket: "default",
                                Key: "a",
                                Value: NewSiblingSet(map[*Sibling]bool{ sibling: true }),
                            },
                            Entry{
                                Site: "site1",
                                Bucket: "default",
                                Key: "b",
                                Value: NewSiblingSet(map[*Sibling]bool{ sibling: true }),
                            },
                        },
                    }
                    encodedChunk, _ := json.Marshal(chunk)
                    // Create a stream with a valid chunk followed by a token that cannot be parsed as a chunk
                    incomingTransfer := NewIncomingTransfer(strings.NewReader(string(encodedChunk)))

                    nextChunk, err := incomingTransfer.NextChunk()
                    Expect(nextChunk.IsEmpty()).Should(BeTrue())
                    Expect(err).Should(Equal(EEntryChecksum))
                })
            })
        })
    })

    Describe("OutgoingTransfer", func() {
        Describe("#NextChunk", func() {
            Context("When the partition iterator has less items (more than zero) left to return than the specified chunk size", func() {
                It("should return a PartitionChunk that contains those remaining entries and a nil error", func() {
                    partition := NewMockPartition(0, 0)
                    iterator := partition.MockIterator()
                    iterator.AppendNextState(true, "site1", "default", "a", nil, Hash{}, nil)
                    iterator.AppendNextState(true, "site1", "default", "b", nil, Hash{}, nil)
                    iterator.AppendNextState(false, "", "", "", nil, Hash{}, nil)

                    outgoingTransfer := NewOutgoingTransfer(partition, 4)

                    nextChunk, err := outgoingTransfer.NextChunk()

                    Expect(nextChunk).Should(Equal(PartitionChunk{
                        Index: 1,
                        Checksum: Hash{ },
                        Entries: []Entry{
                            Entry{
                                Site: "site1",
                                Bucket: "default",
                                Key: "a",
                                Value: nil,
                            },
                            Entry{
                                Site: "site1",
                                Bucket: "default",
                                Key: "b",
                                Value: nil,
                            },
                        },
                    }))
                    Expect(err).Should(BeNil())
                })

                Specify("After calling NextChunk the following call should return an empty chunk and an io.EOF error", func() {
                    partition := NewMockPartition(0, 0)
                    iterator := partition.MockIterator()
                    iterator.AppendNextState(true, "site1", "default", "a", nil, Hash{}, nil)
                    iterator.AppendNextState(true, "site1", "default", "b", nil, Hash{}, nil)
                    iterator.AppendNextState(false, "", "", "", nil, Hash{}, nil)

                    outgoingTransfer := NewOutgoingTransfer(partition, 4)

                    nextChunk, err := outgoingTransfer.NextChunk()

                    Expect(nextChunk).Should(Equal(PartitionChunk{
                        Index: 1,
                        Checksum: Hash{ },
                        Entries: []Entry{
                            Entry{
                                Site: "site1",
                                Bucket: "default",
                                Key: "a",
                                Value: nil,
                            },
                            Entry{
                                Site: "site1",
                                Bucket: "default",
                                Key: "b",
                                Value: nil,
                            },
                        },
                    }))
                    Expect(err).Should(BeNil())

                    nextChunk, err = outgoingTransfer.NextChunk()
                    Expect(nextChunk.IsEmpty()).Should(BeTrue())
                    Expect(err).Should(Equal(io.EOF))
                })
            })

            Context("When the partition iterator has exactly enough items left to meet the chunk size limit", func() {
                It("should return a PartitionChunk that contains the maximum allowed number of entries", func() {
                    partition := NewMockPartition(0, 0)
                    iterator := partition.MockIterator()
                    iterator.AppendNextState(true, "site1", "default", "a", nil, Hash{}, nil)
                    iterator.AppendNextState(true, "site1", "default", "b", nil, Hash{}, nil)
                    iterator.AppendNextState(false, "", "", "", nil, Hash{}, nil)

                    outgoingTransfer := NewOutgoingTransfer(partition, 2)

                    nextChunk, err := outgoingTransfer.NextChunk()

                    Expect(nextChunk).Should(Equal(PartitionChunk{
                        Index: 1,
                        Checksum: Hash{ },
                        Entries: []Entry{
                            Entry{
                                Site: "site1",
                                Bucket: "default",
                                Key: "a",
                                Value: nil,
                            },
                            Entry{
                                Site: "site1",
                                Bucket: "default",
                                Key: "b",
                                Value: nil,
                            },
                        },
                    }))
                    Expect(err).Should(BeNil())
                })

                Specify("After calling NextChunk all subsequent calls should return an empty chunk and an io.EOF error", func() {
                    partition := NewMockPartition(0, 0)
                    iterator := partition.MockIterator()
                    iterator.AppendNextState(true, "site1", "default", "a", nil, Hash{}, nil)
                    iterator.AppendNextState(true, "site1", "default", "b", nil, Hash{}, nil)
                    iterator.AppendNextState(false, "", "", "", nil, Hash{}, nil)

                    outgoingTransfer := NewOutgoingTransfer(partition, 2)

                    nextChunk, err := outgoingTransfer.NextChunk()

                    Expect(nextChunk).Should(Equal(PartitionChunk{
                        Index: 1,
                        Checksum: Hash{ },
                        Entries: []Entry{
                            Entry{
                                Site: "site1",
                                Bucket: "default",
                                Key: "a",
                                Value: nil,
                            },
                            Entry{
                                Site: "site1",
                                Bucket: "default",
                                Key: "b",
                                Value: nil,
                            },
                        },
                    }))
                    Expect(err).Should(BeNil())

                    nextChunk, err = outgoingTransfer.NextChunk()
                    Expect(nextChunk.IsEmpty()).Should(BeTrue())
                    Expect(err).Should(Equal(io.EOF))

                    nextChunk, err = outgoingTransfer.NextChunk()
                    Expect(nextChunk.IsEmpty()).Should(BeTrue())
                    Expect(err).Should(Equal(io.EOF))
                })

                Specify("After calling NextChunk the following call should call Release on the PartitionIterator", func() {
                    partition := NewMockPartition(0, 0)
                    iterator := partition.MockIterator()
                    iterator.AppendNextState(true, "site1", "default", "a", nil, Hash{}, nil)
                    iterator.AppendNextState(true, "site1", "default", "b", nil, Hash{}, nil)
                    iterator.AppendNextState(false, "", "", "", nil, Hash{}, nil)

                    outgoingTransfer := NewOutgoingTransfer(partition, 2)

                    Expect(iterator.ReleaseCallCount()).Should(Equal(0))
                    outgoingTransfer.NextChunk()
                    Expect(iterator.ReleaseCallCount()).Should(Equal(0))
                    outgoingTransfer.NextChunk()
                    Expect(iterator.ReleaseCallCount()).Should(Equal(1))
                })
            })

            Context("When the partition iterator encounters an error", func() {
                It("should return an empty chunk and that error on this and all subsequent calls", func() {
                    partition := NewMockPartition(0, 0)
                    iterator := partition.MockIterator()
                    iterator.AppendNextState(true, "site1", "default", "a", nil, Hash{}, nil)
                    iterator.AppendNextState(true, "site1", "default", "b", nil, Hash{}, nil)
                    iterator.AppendNextState(false, "", "", "", nil, Hash{}, errors.New("Some error"))

                    outgoingTransfer := NewOutgoingTransfer(partition, 2)

                    _, err := outgoingTransfer.NextChunk()
                    Expect(err).Should(BeNil())
                    _, err = outgoingTransfer.NextChunk()
                    Expect(err).Should(Equal(errors.New("Some error")))
                    _, err = outgoingTransfer.NextChunk()
                    Expect(err).Should(Equal(errors.New("Some error")))
                })

                It("should call Release on the PartitionIterator", func() {
                    partition := NewMockPartition(0, 0)
                    iterator := partition.MockIterator()
                    iterator.AppendNextState(true, "site1", "default", "a", nil, Hash{}, nil)
                    iterator.AppendNextState(true, "site1", "default", "b", nil, Hash{}, nil)
                    iterator.AppendNextState(false, "", "", "", nil, Hash{}, errors.New("Some error"))

                    outgoingTransfer := NewOutgoingTransfer(partition, 2)

                    Expect(iterator.ReleaseCallCount()).Should(Equal(0))
                    outgoingTransfer.NextChunk()
                    Expect(iterator.ReleaseCallCount()).Should(Equal(0))
                    outgoingTransfer.NextChunk()
                    Expect(iterator.ReleaseCallCount()).Should(Equal(1))
                })
            })

            Context("When the partition iterator has zero items left to return", func() {
                It("should return an empty chunk and an io.EOF error on this and all subsequent calls", func() {
                    partition := NewMockPartition(0, 0)
                    iterator := partition.MockIterator()
                    iterator.AppendNextState(false, "", "", "", nil, Hash{}, nil)

                    outgoingTransfer := NewOutgoingTransfer(partition, 2)

                    nextChunk, err := outgoingTransfer.NextChunk()
                    Expect(nextChunk.IsEmpty()).Should(BeTrue())
                    Expect(err).Should(Equal(io.EOF))
                    nextChunk, err = outgoingTransfer.NextChunk()
                    Expect(nextChunk.IsEmpty()).Should(BeTrue())
                    Expect(err).Should(Equal(io.EOF))
                })
                
                It("should call Release on the PartitionIterator", func() {
                    partition := NewMockPartition(0, 0)
                    iterator := partition.MockIterator()
                    iterator.AppendNextState(false, "", "", "", nil, Hash{}, nil)

                    outgoingTransfer := NewOutgoingTransfer(partition, 2)

                    Expect(iterator.ReleaseCallCount()).Should(Equal(0))
                    outgoingTransfer.NextChunk()
                    Expect(iterator.ReleaseCallCount()).Should(Equal(1))
                })
            })

            Context("After Cancel was called", func() {
                Context("If Cancel was called following an abnormal PartitionIterator error", func() {
                    It("should return an empty chunk and that error", func() {
                        partition := NewMockPartition(0, 0)
                        iterator := partition.MockIterator()
                        iterator.AppendNextState(false, "", "", "", nil, Hash{}, errors.New("Some error"))

                        outgoingTransfer := NewOutgoingTransfer(partition, 2)

                        _, err := outgoingTransfer.NextChunk()
                        Expect(err).Should(Equal(errors.New("Some error")))
                        outgoingTransfer.Cancel()
                        _, err = outgoingTransfer.NextChunk()
                        Expect(err).Should(Equal(errors.New("Some error")))
                    })
                })

                Context("If Cancel was called following an io.EOF error", func() {
                    It("should return an empty chunk and an io.EOF error", func() {
                        partition := NewMockPartition(0, 0)
                        iterator := partition.MockIterator()
                        iterator.AppendNextState(false, "", "", "", nil, Hash{}, nil)

                        outgoingTransfer := NewOutgoingTransfer(partition, 2)

                        _, err := outgoingTransfer.NextChunk()
                        Expect(err).Should(Equal(io.EOF))
                        outgoingTransfer.Cancel()
                        _, err = outgoingTransfer.NextChunk()
                        Expect(err).Should(Equal(io.EOF))
                    })
                })

                Context("If Cancel was called before any error occurred in NextChunk", func() {
                    It("should return an empty chunk and ETransferCancelled on this and all subsequent calls", func() {
                        partition := NewMockPartition(0, 0)
                        iterator := partition.MockIterator()
                        iterator.AppendNextState(false, "", "", "", nil, Hash{}, errors.New("Some error"))

                        outgoingTransfer := NewOutgoingTransfer(partition, 2)

                        outgoingTransfer.Cancel()
                        _, err := outgoingTransfer.NextChunk()
                        Expect(err).Should(Equal(ETransferCancelled))
                        _, err = outgoingTransfer.NextChunk()
                        Expect(err).Should(Equal(ETransferCancelled))
                    })
                })
            })

            Specify("The index of chunks should be monotonically increasing and should start from one inclusive", func() {
                partition := NewMockPartition(0, 0)
                iterator := partition.MockIterator()
                iterator.AppendNextState(true, "site1", "default", "a", nil, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "b", nil, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "c", nil, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "d", nil, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "e", nil, Hash{}, nil)
                iterator.AppendNextState(true, "site1", "default", "f", nil, Hash{}, nil)
                iterator.AppendNextState(false, "", "", "", nil, Hash{}, nil)

                outgoingTransfer := NewOutgoingTransfer(partition, 2)

                nextChunk, _ := outgoingTransfer.NextChunk()
                Expect(nextChunk.Index).Should(Equal(uint64(1)))
                nextChunk, _ = outgoingTransfer.NextChunk()
                Expect(nextChunk.Index).Should(Equal(uint64(2)))
                nextChunk, _ = outgoingTransfer.NextChunk()
                Expect(nextChunk.Index).Should(Equal(uint64(3)))
            })
        })

        Describe("#Cancel", func() {
            It("should call Release on the PartitionIterator", func() {
                partition := NewMockPartition(0, 0)
                iterator := partition.MockIterator()
                outgoingTransfer := NewOutgoingTransfer(partition, 2)

                Expect(iterator.ReleaseCallCount()).Should(Equal(0))
                outgoingTransfer.Cancel()
                Expect(iterator.ReleaseCallCount()).Should(Equal(1))
            })
        })
    })
})
