package raft_test

import (
    "errors"
    "fmt"
    "encoding/binary"
    "crypto/rand"
    "github.com/coreos/etcd/raft"
    "github.com/coreos/etcd/raft/raftpb"

    . "github.com/armPelionEdge/devicedb/raft"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
    . "github.com/armPelionEdge/devicedb/storage"
    "math"
)

func randomString() string {
    randomBytes := make([]byte, 16)
    rand.Read(randomBytes)
    
    high := binary.BigEndian.Uint64(randomBytes[:8])
    low := binary.BigEndian.Uint64(randomBytes[8:])
    
    return fmt.Sprintf("%05x%05x", high, low)
}

func ek(index uint64) []byte {
    key := make([]byte, 0, len(KeyPrefixEntry) + 8)
    indexBytes := make([]byte, 8)

    binary.BigEndian.PutUint64(indexBytes, index)

    key = append(key, KeyPrefixEntry...)
    key = append(key, indexBytes...)

    return key
}

func ev(index uint64) []byte {
    entry := raftpb.Entry{ Index: index, Term: index }

    data, _ := entry.Marshal()

    return data
}

var _ = Describe("Storage", func() {
    var storageDriver StorageDriver
    var raftStorage *RaftStorage

    BeforeEach(func() {
        storageDriver = NewLevelDBStorageDriver("/tmp/testraftstore-" + randomString(), nil)
        raftStorage = NewRaftStorage(storageDriver)
    })

    AfterEach(func() {
        raftStorage.Close()
    })

    Describe("Setting and querying decommissioning flag", func() {
        Specify("Decommissioning flag should not be raised if SetDecommissioningFlag has never been called", func() {
            Expect(raftStorage.Open()).Should(BeNil())
            isDecommissioning, err := raftStorage.IsDecommissioning()
            Expect(err).Should(BeNil())
            Expect(isDecommissioning).Should(BeFalse())
        })

        Specify("Decommissioning flag should always be raised once SetDecommissioningFlag is called even after restart", func() {
            Expect(raftStorage.Open()).Should(BeNil())
            isDecommissioning, err := raftStorage.IsDecommissioning()
            Expect(err).Should(BeNil())
            Expect(isDecommissioning).Should(BeFalse())
            Expect(raftStorage.SetDecommissioningFlag()).Should(BeNil())
            isDecommissioning, err = raftStorage.IsDecommissioning()
            Expect(err).Should(BeNil())
            Expect(isDecommissioning).Should(BeTrue())
            raftStorage.Close()
            isDecommissioning, err = raftStorage.IsDecommissioning()
            Expect(err).Should(Not(BeNil()))
            Expect(isDecommissioning).Should(BeFalse())
            raftStorage.Open()
            isDecommissioning, err = raftStorage.IsDecommissioning()
            Expect(err).Should(BeNil())
            Expect(isDecommissioning).Should(BeTrue())
        })
    })

    Describe("#Open", func() {
        It("if opening for the first time there should be no entries, the snapshot should be empty, and the hard state should be empty", func() {
            Expect(raftStorage.Open()).Should(BeNil())
            Expect(raftStorage.Snapshot()).Should(Equal(raftpb.Snapshot{ }))

            hardState, _, err := raftStorage.InitialState()
            Expect(err).Should(BeNil())
            Expect(hardState).Should(Equal(raftpb.HardState{ }))

            firstIndex, _ := raftStorage.FirstIndex()
            lastIndex, _ := raftStorage.LastIndex()

            Expect(firstIndex).Should(Equal(uint64(1)))
            Expect(lastIndex).Should(Equal(uint64(0)))

            _, err = raftStorage.Entries(0, 1, math.MaxUint64)

            Expect(err).Should(Equal(raft.ErrCompacted))

            _, err = raftStorage.Entries(1, 1, math.MaxUint64)

            Expect(err).Should(Equal(raft.ErrUnavailable))
        })

        It("should return an error if the snapshot is corrupted or not formatted correctly on disk", func() {
            batch := NewBatch()
            batch.Put(KeySnapshot, []byte{ 8, 3, 9, 2 })

            Expect(storageDriver.Open()).Should(BeNil())
            Expect(storageDriver.Batch(batch)).Should(BeNil())
            Expect(storageDriver.Close()).Should(BeNil())

            Expect(raftStorage.Open()).Should(Not(BeNil()))
        })

        It("should return an error if the hard state is corrupted or not formatted correctly on disk", func() {
            batch := NewBatch()
            batch.Put(KeyHardState, []byte{ 8, 3, 9, 2 })

            Expect(storageDriver.Open()).Should(BeNil())
            Expect(storageDriver.Batch(batch)).Should(BeNil())
            Expect(storageDriver.Close()).Should(BeNil())

            Expect(raftStorage.Open()).Should(Not(BeNil()))
        })

        It("should return an error if the entries on disk do not contain monotonically increasing indices", func() {
            batch := NewBatch()

            batch.Put(ek(3), ev(3))
            batch.Put(ek(4), ev(4))
            batch.Put(ek(5), ev(5))
            batch.Put(ek(7), ev(7))

            Expect(storageDriver.Open()).Should(BeNil())
            Expect(storageDriver.Batch(batch)).Should(BeNil())
            Expect(storageDriver.Close()).Should(BeNil())

            Expect(raftStorage.Open()).Should(Equal(errors.New("Entry indices are not monotonically increasing")))
        })

        It("should return an error if the index in the entries on disk do not match the index implied by their key", func() {
            batch := NewBatch()

            batch.Put(ek(3), ev(3))
            batch.Put(ek(4), ev(4))
            batch.Put(ek(5), ev(5))
            batch.Put(ek(6), ev(4))

            Expect(storageDriver.Open()).Should(BeNil())
            Expect(storageDriver.Batch(batch)).Should(BeNil())
            Expect(storageDriver.Close()).Should(BeNil())

            Expect(raftStorage.Open()).Should(Equal(errors.New("Encoded entry index does not match index in its key")))
        })
    })

    Describe("#SetHardState", func() {
        It("should return an error and not modify the memory storage object if it fails to persist hard state to disk", func() {
            newHardState := raftpb.HardState{ Term: 55, Vote: 999 }

            Expect(raftStorage.Open()).Should(BeNil())
            Expect(storageDriver.Close()).Should(BeNil()) // close storage driver early to induce a storage error
            Expect(raftStorage.SetHardState(newHardState)).Should(Not(BeNil()))

            // verify that the memory hard state is still empty
            hardState, _, err := raftStorage.InitialState()
            Expect(err).Should(BeNil())
            Expect(hardState).Should(Equal(raftpb.HardState{ }))
        })

        It("should change both on disk hard state and in memory hard state", func() {
            newHardState := raftpb.HardState{ Term: 55, Vote: 999 }

            Expect(raftStorage.Open()).Should(BeNil())
            Expect(raftStorage.SetHardState(newHardState)).Should(BeNil())

            // verify that the memory hard state is still empty
            hardState, _, err := raftStorage.InitialState()
            Expect(err).Should(BeNil())
            Expect(hardState).Should(Equal(newHardState))

            Expect(raftStorage.Close()).Should(BeNil())

            // open a new raft storage object to verify that the contents of hard state were persisted
            newRaftStorage := NewRaftStorage(storageDriver)
            Expect(newRaftStorage.Open()).Should(BeNil())
            hardState, _, err = newRaftStorage.InitialState()
            Expect(err).Should(BeNil())
            Expect(hardState).Should(Equal(newHardState))
        })
    })

    Describe("#ApplySnapshot", func() {
        It("should return an error and not modify the memory storage object if it fails to persist snapshot and entries state to disk", func() {
            newSnapshot := raftpb.Snapshot{ Data: []byte{ 0, 1, 2 }, Metadata: raftpb.SnapshotMetadata{ Index: 44, Term: 44 } }

            Expect(raftStorage.Open()).Should(BeNil())
            Expect(storageDriver.Close()).Should(BeNil()) // close storage driver early to induce a storage error
            Expect(raftStorage.ApplySnapshot(newSnapshot)).Should(Not(BeNil()))

            Expect(raftStorage.Snapshot()).Should(Equal(raftpb.Snapshot{ }))
        })

        It("should change both on disk snapshot state and in memory snapshot state", func() {
            newSnapshot := raftpb.Snapshot{ Data: []byte{ 0, 1, 2 }, Metadata: raftpb.SnapshotMetadata{ Index: 44, Term: 44 } }

            Expect(raftStorage.Open()).Should(BeNil())
            Expect(raftStorage.ApplySnapshot(newSnapshot)).Should(BeNil())

            Expect(raftStorage.Snapshot()).Should(Equal(newSnapshot))

            firstIndex, _ := raftStorage.FirstIndex()
            lastIndex, _ := raftStorage.LastIndex()

            Expect(firstIndex).Should(Equal(uint64(45)))
            Expect(lastIndex).Should(Equal(uint64(44)))

            // open a new raft storage object to verify that the contents of snapshot were persisted
            Expect(storageDriver.Close()).Should(BeNil())
            newRaftStorage := NewRaftStorage(storageDriver)
            Expect(newRaftStorage.Open()).Should(BeNil())
            Expect(newRaftStorage.Snapshot()).Should(Equal(newSnapshot))
            firstIndex, _ = newRaftStorage.FirstIndex()
            lastIndex, _ = newRaftStorage.LastIndex()
            Expect(firstIndex).Should(Equal(uint64(45)))
            Expect(lastIndex).Should(Equal(uint64(44)))
        })

        It("should purge all entries from disk up to the index of the snapshot", func() {
            newSnapshot := raftpb.Snapshot{ Data: []byte{ 0, 1, 2 }, Metadata: raftpb.SnapshotMetadata{ Index: 44, Term: 44 } }

            Expect(raftStorage.Open()).Should(BeNil())

            entries := make([]raftpb.Entry, 50)

            for i := uint64(0); i < uint64(50); i++ {
                entries[i] = raftpb.Entry{ Index: i, Term: i }
            }

            Expect(raftStorage.Append(entries)).Should(BeNil())
            Expect(raftStorage.ApplySnapshot(newSnapshot)).Should(BeNil())
            Expect(raftStorage.Snapshot()).Should(Equal(newSnapshot))

            firstIndex, _ := raftStorage.FirstIndex()
            lastIndex, _ := raftStorage.LastIndex()

            Expect(firstIndex).Should(Equal(uint64(45)))
            Expect(lastIndex).Should(Equal(uint64(44)))

            // open a new raft storage object to verify that the contents of snapshot were persisted
            Expect(storageDriver.Close()).Should(BeNil())
            newRaftStorage := NewRaftStorage(storageDriver)
            Expect(newRaftStorage.Open()).Should(BeNil())
            Expect(newRaftStorage.Snapshot()).Should(Equal(newSnapshot))
            firstIndex, _ = newRaftStorage.FirstIndex()
            lastIndex, _ = newRaftStorage.LastIndex()
            Expect(firstIndex).Should(Equal(uint64(45)))
            Expect(lastIndex).Should(Equal(uint64(44)))
        })
    })

    Describe("#CreateSnapshot", func() {
        It("should create a snapshot from the current state of the storage and compact the log up to that point", func() {
            Expect(raftStorage.Open()).Should(BeNil())

            entries := make([]raftpb.Entry, 50)

            for i := uint64(0); i < uint64(50); i++ {
                entries[i] = raftpb.Entry{ Index: i, Term: i }
            }

            Expect(raftStorage.Append(entries)).Should(BeNil())
            snap, err := raftStorage.CreateSnapshot(25, nil, []byte{ 0, 1, 2 })

            Expect(err).Should(BeNil())
            Expect(snap).Should(Equal(raftpb.Snapshot{ 
                Metadata: raftpb.SnapshotMetadata{
                    Index: 25,
                    Term: 25,
                },
                Data: []byte{ 0, 1, 2 },
            }))

            firstIndex, _ := raftStorage.FirstIndex()
            lastIndex, _ := raftStorage.LastIndex()
            Expect(firstIndex).Should(Equal(uint64(26)))
            Expect(lastIndex).Should(Equal(uint64(49)))

            // open a new raft storage object to verify that the contents of snapshot were persisted
            Expect(storageDriver.Close()).Should(BeNil())
            newRaftStorage := NewRaftStorage(storageDriver)
            Expect(newRaftStorage.Open()).Should(BeNil())
            Expect(newRaftStorage.Snapshot()).Should(Equal(raftpb.Snapshot{
                Metadata: raftpb.SnapshotMetadata{
                    Index: 25,
                    Term: 25,
                },
                Data: []byte{ 0, 1, 2 },
            }))
            firstIndex, _ = newRaftStorage.FirstIndex()
            lastIndex, _ = newRaftStorage.LastIndex()
            Expect(firstIndex).Should(Equal(uint64(26)))
            Expect(lastIndex).Should(Equal(uint64(49)))
        })

        It("should create a snapshot from the current state of the storage and compact the log up to that point", func() {
            Expect(raftStorage.Open()).Should(BeNil())

            entries := make([]raftpb.Entry, 50)

            for i := uint64(0); i < uint64(50); i++ {
                entries[i] = raftpb.Entry{ Index: i, Term: i }
            }

            Expect(raftStorage.Append(entries)).Should(BeNil())
            snap, err := raftStorage.CreateSnapshot(49, nil, []byte{ 0, 1, 2 })

            Expect(err).Should(BeNil())
            Expect(snap).Should(Equal(raftpb.Snapshot{ 
                Metadata: raftpb.SnapshotMetadata{
                    Index: 49,
                    Term: 49,
                },
                Data: []byte{ 0, 1, 2 },
            }))

            firstIndex, _ := raftStorage.FirstIndex()
            lastIndex, _ := raftStorage.LastIndex()
            Expect(firstIndex).Should(Equal(uint64(50)))
            Expect(lastIndex).Should(Equal(uint64(49)))

            // open a new raft storage object to verify that the contents of snapshot were persisted
            Expect(storageDriver.Close()).Should(BeNil())
            newRaftStorage := NewRaftStorage(storageDriver)
            Expect(newRaftStorage.Open()).Should(BeNil())
            Expect(newRaftStorage.Snapshot()).Should(Equal(raftpb.Snapshot{
                Metadata: raftpb.SnapshotMetadata{
                    Index: 49,
                    Term: 49,
                },
                Data: []byte{ 0, 1, 2 },
            }))
            firstIndex, _ = newRaftStorage.FirstIndex()
            lastIndex, _ = newRaftStorage.LastIndex()
            Expect(firstIndex).Should(Equal(uint64(50)))
            Expect(lastIndex).Should(Equal(uint64(49)))
        })
    })

    Describe("#Append", func() {
        It("appending entries whose indexes have already been compacted should result in those entries being truncated", func() {
            Expect(raftStorage.Open()).Should(BeNil())

            entries := make([]raftpb.Entry, 50)

            for i := uint64(0); i < uint64(50); i++ {
                entries[int(i)] = raftpb.Entry{ Index: uint64(i), Term: uint64(i) }
            }

            Expect(raftStorage.Append(entries)).Should(BeNil())
            raftStorage.CreateSnapshot(25, nil, []byte{ })
            Expect(raftStorage.Append([]raftpb.Entry{ raftpb.Entry{ Index: 24, Term: 24 }, raftpb.Entry{ Index: 25, Term: 26 }, raftpb.Entry{ Index: 26, Term: 29 }, raftpb.Entry{ Index: 27, Term: 29 } })).Should(BeNil())

            ents, err := raftStorage.Entries(27, 28, math.MaxUint64)

            Expect(err).Should(BeNil())
            Expect(ents[0]).Should(Equal(raftpb.Entry{ Index: 27, Term: 29 }))

            firstIndex, _ := raftStorage.FirstIndex()
            lastIndex, _ := raftStorage.LastIndex()
            Expect(firstIndex).Should(Equal(uint64(26)))
            Expect(lastIndex).Should(Equal(uint64(27)))

            // open a new raft storage object to verify that the contents of snapshot were persisted
            Expect(storageDriver.Close()).Should(BeNil())
            newRaftStorage := NewRaftStorage(storageDriver)
            Expect(newRaftStorage.Open()).Should(BeNil())
            Expect(newRaftStorage.Snapshot()).Should(Equal(raftpb.Snapshot{
                Metadata: raftpb.SnapshotMetadata{
                    Index: 25,
                    Term: 25,
                },
                Data: []byte{ },
            }))
            firstIndex, _ = newRaftStorage.FirstIndex()
            lastIndex, _ = newRaftStorage.LastIndex()
            Expect(firstIndex).Should(Equal(uint64(26)))
            Expect(lastIndex).Should(Equal(uint64(27)))
        })

        It("new entries overwrite old ones", func() {
            Expect(raftStorage.Open()).Should(BeNil())

            entries := make([]raftpb.Entry, 50)

            for i := uint64(0); i < uint64(50); i++ {
                entries[int(i)] = raftpb.Entry{ Index: uint64(i), Term: uint64(i) }
            }

            Expect(raftStorage.Append(entries)).Should(BeNil())
            Expect(raftStorage.Append([]raftpb.Entry{ raftpb.Entry{ Index: 48, Term: 100 }, raftpb.Entry{ Index: 49, Term: 102 }, raftpb.Entry{ Index: 50, Term: 103 }, raftpb.Entry{ Index: 51, Term: 104 } })).Should(BeNil())

            ents, err := raftStorage.Entries(48, 52, math.MaxUint64)

            Expect(err).Should(BeNil())
            Expect(ents[0]).Should(Equal(raftpb.Entry{ Index: 48, Term: 100 }))
            Expect(ents[1]).Should(Equal(raftpb.Entry{ Index: 49, Term: 102 }))
            Expect(ents[2]).Should(Equal(raftpb.Entry{ Index: 50, Term: 103 }))
            Expect(ents[3]).Should(Equal(raftpb.Entry{ Index: 51, Term: 104 }))

            firstIndex, _ := raftStorage.FirstIndex()
            lastIndex, _ := raftStorage.LastIndex()
            Expect(firstIndex).Should(Equal(uint64(1)))
            Expect(lastIndex).Should(Equal(uint64(51)))

            // open a new raft storage object to verify that the contents of snapshot were persisted
            Expect(storageDriver.Close()).Should(BeNil())
            newRaftStorage := NewRaftStorage(storageDriver)
            Expect(newRaftStorage.Open()).Should(BeNil())
            Expect(newRaftStorage.Snapshot()).Should(Equal(raftpb.Snapshot{ }))
            firstIndex, _ = newRaftStorage.FirstIndex()
            lastIndex, _ = newRaftStorage.LastIndex()
            Expect(firstIndex).Should(Equal(uint64(1)))
            Expect(lastIndex).Should(Equal(uint64(51)))
            
            ents, err = newRaftStorage.Entries(48, 52, math.MaxUint64)

            Expect(err).Should(BeNil())
            Expect(ents[0]).Should(Equal(raftpb.Entry{ Index: 48, Term: 100 }))
            Expect(ents[1]).Should(Equal(raftpb.Entry{ Index: 49, Term: 102 }))
            Expect(ents[2]).Should(Equal(raftpb.Entry{ Index: 50, Term: 103 }))
            Expect(ents[3]).Should(Equal(raftpb.Entry{ Index: 51, Term: 104 }))
        })
    })
})
