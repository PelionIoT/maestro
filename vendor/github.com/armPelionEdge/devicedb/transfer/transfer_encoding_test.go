package transfer_test

import (
    "io"
    "bufio"
    "encoding/json"
    "errors"

    . "github.com/armPelionEdge/devicedb/transfer"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("TransferEncoding", func() {
    Describe("JSONPartitionReader", func() {
        Describe("#Read", func() {
            Context("When p is a nil array", func() {
                It("should return zero bytes read and no error", func() {
                    reader := &JSONPartitionReader{ }
                    n, err := reader.Read(nil)

                    Expect(n).Should(Equal(0))
                    Expect(err).Should(BeNil())
                })
            })

            Context("When p is an empty array", func() {
                It("should return zero bytes read and no error", func() {
                    reader := &JSONPartitionReader{ }
                    n, err := reader.Read([]byte{ })

                    Expect(n).Should(Equal(0))
                    Expect(err).Should(BeNil())
                })
            })

            Context("When p is an array whose length is less than the current chunk", func() {
                It("should fill p up with as many bytes of the next chunk as p can hold, return that number of bytes and no error", func() {
                    firstChunk := PartitionChunk{ Entries: []Entry{ Entry{ } } }
                    firstChunkEncoded, _ := json.Marshal(firstChunk)

                    // make buffer only half as large as it needs to be to hold the encoded next chunk
                    buffer := make([]byte, len(firstChunkEncoded) / 2)
                    mockPartitionTransfer := NewMockPartitionTransfer().AppendNextChunkResponse(firstChunk, nil)
                    reader := &JSONPartitionReader{ PartitionTransfer: mockPartitionTransfer }

                    n, err := reader.Read(buffer)

                    Expect(mockPartitionTransfer.CancelCallCount()).Should(Equal(0))
                    Expect(mockPartitionTransfer.NextChunkCallCount()).Should(Equal(1))
                    Expect(buffer).Should(Equal(firstChunkEncoded[:len(buffer)]))
                    Expect(n).Should(Equal(len(buffer)))
                    Expect(err).Should(BeNil())
                })
            })

            Specify("When a call to NextChunk returns an error the same error should be returned by the reader and no bytes should be written to the buffer",func() {
                // make buffer only half as large as it needs to be to hold the encoded next chunk
                buffer := make([]byte, 8)
                mockPartitionTransfer := NewMockPartitionTransfer().AppendNextChunkResponse(PartitionChunk{ }, errors.New("Some error"))
                reader := &JSONPartitionReader{ PartitionTransfer: mockPartitionTransfer }

                n, err := reader.Read(buffer)

                Expect(mockPartitionTransfer.CancelCallCount()).Should(Equal(0))
                Expect(mockPartitionTransfer.NextChunkCallCount()).Should(Equal(1))
                Expect(buffer).Should(Equal([]byte{ 0, 0, 0, 0, 0, 0, 0, 0 }))
                Expect(n).Should(Equal(0))
                Expect(err).Should(Equal(errors.New("Some error")))
            })

            Context("When the last call to Read ended exactly at the current chunk", func() {
                firstChunk := PartitionChunk{ Entries: []Entry{ Entry{ } } }
                firstChunkEncoded, _ := json.Marshal(firstChunk)

                var reader *JSONPartitionReader
                var mockPartitionTransfer *MockPartitionTransfer

                BeforeEach(func() {
                    // Make this buffer exactly big enough to hold the entire encoding for the first chunk
                    buffer := make([]byte, len(firstChunkEncoded))
                    mockPartitionTransfer = NewMockPartitionTransfer().AppendNextChunkResponse(firstChunk, nil)
                    reader = &JSONPartitionReader{ PartitionTransfer: mockPartitionTransfer }
                    n, err := reader.Read(buffer)

                    Expect(mockPartitionTransfer.CancelCallCount()).Should(Equal(0))
                    Expect(mockPartitionTransfer.NextChunkCallCount()).Should(Equal(1))
                    Expect(buffer).Should(Equal(firstChunkEncoded))
                    Expect(n).Should(Equal(len(firstChunkEncoded)))
                    Expect(err).Should(BeNil())
                })

                Specify("The next call to Read should put a newline character at the start of the buffer to delimit the two chunks", func() {
                    secondChunk := firstChunk
                    secondChunkEncoded, _ := json.Marshal(secondChunk)

                    mockPartitionTransfer.AppendNextChunkResponse(secondChunk, nil)
                    buffer := make([]byte, len(secondChunkEncoded))
                    n, err := reader.Read(buffer)
                    Expect(mockPartitionTransfer.CancelCallCount()).Should(Equal(0))
                    Expect(mockPartitionTransfer.NextChunkCallCount()).Should(Equal(2))
                    Expect(buffer[:1]).Should(Equal([]byte("\n")))
                    Expect(buffer[1:]).Should(Equal(secondChunkEncoded[:len(secondChunkEncoded) - 1]))
                    Expect(n).Should(Equal(len(firstChunkEncoded)))
                    Expect(err).Should(BeNil())
                })

                Context("And that was the last chunk in the transfer", func() {
                    It("Should return 0 bytes written and an io.EOF error", func() {
                        // an empty chunk indicates the end of a transfer
                        mockPartitionTransfer.AppendNextChunkResponse(PartitionChunk{}, nil)

                        buffer := make([]byte, len(firstChunkEncoded))
                        n, err := reader.Read(buffer)
                        Expect(mockPartitionTransfer.CancelCallCount()).Should(Equal(0))
                        Expect(mockPartitionTransfer.NextChunkCallCount()).Should(Equal(2))
                        for _, v := range buffer {
                            Expect(v).Should(Equal(byte(0)))
                        }
                        Expect(n).Should(Equal(0))
                        Expect(err).Should(Equal(io.EOF))
                    })
                })
            })
        })
    })

    Describe("TransferEncoder", func() {
        Describe("#Encode", func() {
            Context("When there are no errors while making calls to NextChunk in the underlying partition transfer", func() {
                It("should return a reader that contains a stream of newline-delimited, json-encoded chunks", func() {
                    nChunks := 100
                    mockPartitionTransfer := NewMockPartitionTransfer()

                    for i := 0; i < nChunks; i++ {
                        chunk := PartitionChunk{ 
                            Index: uint64(i),
                            Entries: []Entry{ Entry{ }, Entry{ }, Entry{ } },
                        }

                        mockPartitionTransfer.AppendNextChunkResponse(chunk, nil)
                    }

                    transferEncoder := NewTransferEncoder(mockPartitionTransfer)
                    reader, err := transferEncoder.Encode()
                    Expect(reader).Should(Not(BeNil()))
                    Expect(err).Should(BeNil())
                    scanner := bufio.NewScanner(reader)

                    for i := 0; i < nChunks; i++ {
                        Expect(scanner.Scan()).Should(BeTrue())
                        var partitionChunk PartitionChunk
                        bytes := scanner.Bytes()
                        Expect(json.Unmarshal(bytes, &partitionChunk)).Should(BeNil())
                        Expect(partitionChunk.Entries).Should(Equal([]Entry{ Entry{ }, Entry{ }, Entry{ } }))
                        Expect(partitionChunk.Index).Should(Equal(uint64(i)))
                    }

                    Expect(scanner.Scan()).Should(BeFalse())
                    Expect(scanner.Err()).Should(BeNil())
                })
            })

            Context("When there is an error while making calls to NextChunk in the underlying partition transfer", func() {
                Specify("The reader should return an error", func() {
                    nChunks := 100
                    mockPartitionTransfer := NewMockPartitionTransfer()

                    for i := 0; i < nChunks; i++ {
                        chunk := PartitionChunk{ 
                            Index: uint64(i),
                            Entries: []Entry{ Entry{ }, Entry{ }, Entry{ } },
                        }

                        mockPartitionTransfer.AppendNextChunkResponse(chunk, nil)
                    }

                    mockPartitionTransfer.AppendNextChunkResponse(PartitionChunk{}, errors.New("Some error"))

                    transferEncoder := NewTransferEncoder(mockPartitionTransfer)
                    reader, err := transferEncoder.Encode()
                    Expect(reader).Should(Not(BeNil()))
                    Expect(err).Should(BeNil())
                    scanner := bufio.NewScanner(reader)

                    for i := 0; i < nChunks + 1; i++ {
                        if !scanner.Scan() {
                            break
                        }

                        var partitionChunk PartitionChunk
                        bytes := scanner.Bytes()
                        Expect(json.Unmarshal(bytes, &partitionChunk)).Should(BeNil())
                        Expect(partitionChunk.Entries).Should(Equal([]Entry{ Entry{ }, Entry{ }, Entry{ } }))
                        Expect(partitionChunk.Index).Should(Equal(uint64(i)))
                    }

                    Expect(scanner.Err()).Should(Equal(errors.New("Some error")))
                })
            })
        })
    })

    Describe("#TransferDecoder", func() {
    })
})
