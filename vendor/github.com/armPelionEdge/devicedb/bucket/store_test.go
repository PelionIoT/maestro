package bucket_test

import (
    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/storage"
    . "github.com/armPelionEdge/devicedb/merkle"
    . "github.com/armPelionEdge/devicedb/data"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "time"
    "crypto/rand"
    "encoding/binary"
    "fmt"
)

func randomString() string {
    randomBytes := make([]byte, 16)
    rand.Read(randomBytes)
    
    high := binary.BigEndian.Uint64(randomBytes[:8])
    low := binary.BigEndian.Uint64(randomBytes[8:])
    
    return fmt.Sprintf("%05x%05x", high, low)
}

func makeNewStorageDriver() StorageDriver {
    return NewLevelDBStorageDriver("/tmp/testdb-" + randomString(), nil)
}

var _ = Describe("Store", func() {
    makeStore := func(nodeID string) *Store {
        storageEngine := makeNewStorageDriver()
        storageEngine.Open()
        defer storageEngine.Close()
        store := &Store{ }
        store.Initialize(nodeID, storageEngine, MerkleMinDepth, nil)
        
        return store
    }
        
    Describe("#Get", func() {
        It("should return EEmpty if the keys slice is nil", func() {
            store := makeStore("nodeA")
            ss, err := store.Get(nil)
            
            Expect(err.(DBerror).Code()).Should(Equal(EEmpty.Code()))
            Expect(ss).Should(BeNil())
        })
        
        It("should return EEmpty if the keys slice is empty", func() {
            store := makeStore("nodeA")
            ss, err := store.Get([][]byte{ })
            
            Expect(err.(DBerror).Code()).Should(Equal(EEmpty.Code()))
            Expect(ss).Should(BeNil())
        })
        
        It("should return EEmpty if the keys slice contains a nil key", func() {
            store := makeStore("nodeA")
            ss, err := store.Get([][]byte{ nil })
            
            Expect(err.(DBerror).Code()).Should(Equal(EEmpty.Code()))
            Expect(ss).Should(BeNil())
        })
        
        It("should return EEmpty if the keys slice contains an empty key", func() {
            store := makeStore("nodeA")
            ss, err := store.Get([][]byte{ []byte{ } })
            
            Expect(err.(DBerror).Code()).Should(Equal(EEmpty.Code()))
            Expect(ss).Should(BeNil())
        })
        
        It("should return ELength if the keys slice contains a key that is too long", func() {
            longKey := [MAX_SORTING_KEY_LENGTH+1]byte{ }
            store := makeStore("nodeA")
            ss, err := store.Get([][]byte{ longKey[:] })
            
            Expect(err.(DBerror).Code()).Should(Equal(ELength.Code()))
            Expect(ss).Should(BeNil())
        })
        
        It("should return EStorage if a storage driver error occurs", func() {
            store := makeStore("nodeA")
            ss, err := store.Get([][]byte{ []byte{ 0 } })
            
            // will cause an error because the storage driver was closed at the end of makeStore()
            Expect(err.(DBerror).Code()).Should(Equal(EStorage.Code()))
            Expect(ss).Should(BeNil())
        })
        
        It("should return EStorage if the data is misformatted", func() {
            encodePartitionDataKey := func(k []byte) []byte {
                result := make([]byte, 0, len(PARTITION_DATA_PREFIX) + len(k))
                
                result = append(result, PARTITION_DATA_PREFIX...)
                result = append(result, k...)
                
                return result
            }
            
            storageEngine := makeNewStorageDriver()
            storageEngine.Open()
            defer storageEngine.Close()
           
            store := &Store{}
            store.Initialize("nodeA", storageEngine, MerkleMinDepth, nil)
            batch := NewBatch()
            batch.Put(encodePartitionDataKey([]byte("sorting1")), []byte("value123"))
            err := storageEngine.Batch(batch)
            
            Expect(err).Should(BeNil())
            
            ss, err := store.Get([][]byte{ []byte("sorting1") })
            
            // will cause an error because the storage driver was closed at the end of makeStore()
            Expect(err.(DBerror).Code()).Should(Equal(EStorage.Code()))
            Expect(ss).Should(BeNil())
        })

        It("should not modify the keys array it receives", func() {
            store := makeStore("nodeA")
            keys := [][]byte{ []byte("a"), []byte("b") }
            store.Get(keys)

            Expect(keys[0]).Should(Equal([]byte("a")))
            Expect(keys[1]).Should(Equal([]byte("b")))
        })
    })
    
    Describe("#GetMatches", func() {
        It("should return EEmpty if the keys slice is nil", func() {
            store := makeStore("nodeA")
            ss, err := store.GetMatches(nil)
            
            Expect(err.(DBerror).Code()).Should(Equal(EEmpty.Code()))
            Expect(ss).Should(BeNil())
        })
        
        It("should return EEmpty if the keys slice is empty", func() {
            store := makeStore("nodeA")
            ss, err := store.GetMatches([][]byte{ })
            
            Expect(err.(DBerror).Code()).Should(Equal(EEmpty.Code()))
            Expect(ss).Should(BeNil())
        })
        
        It("should return EEmpty if the keys slice contains a nil key", func() {
            store := makeStore("nodeA")
            ss, err := store.GetMatches([][]byte{ nil })
            
            Expect(err.(DBerror).Code()).Should(Equal(EEmpty.Code()))
            Expect(ss).Should(BeNil())
        })
        
        It("should return EEmpty if the keys slice contains an empty key", func() {
            store := makeStore("nodeA")
            ss, err := store.GetMatches([][]byte{ []byte{ } })
            
            Expect(err.(DBerror).Code()).Should(Equal(EEmpty.Code()))
            Expect(ss).Should(BeNil())
        })
        
        It("should return ELength if the keys slice contains a key that is too long", func() {
            longKey := [MAX_SORTING_KEY_LENGTH+1]byte{ }
            store := makeStore("nodeA")
            ss, err := store.GetMatches([][]byte{ longKey[:] })
            
            Expect(err.(DBerror).Code()).Should(Equal(ELength.Code()))
            Expect(ss).Should(BeNil())
        })
        
        It("should return EStorage if a storage driver error occurs", func() {
            store := makeStore("nodeA")
            ss, err := store.GetMatches([][]byte{ []byte{ 0 } })
            
            // will cause an error because the storage driver was closed at the end of makeStore()
            Expect(err.(DBerror).Code()).Should(Equal(EStorage.Code()))
            Expect(ss).Should(BeNil())
        })
        
        It("should return EStorage if the data is misformatted", func() {
            encodePartitionDataKey := func(k []byte) []byte {
                result := make([]byte, 0, len(PARTITION_DATA_PREFIX) + len(k))
                
                result = append(result, PARTITION_DATA_PREFIX...)
                result = append(result, k...)
                
                return result
            }
            
            storageEngine := makeNewStorageDriver()
            storageEngine.Open()
            defer storageEngine.Close()
            
            store := &Store{}
            store.Initialize("nodeA", storageEngine, MerkleMinDepth, nil)
            batch := NewBatch()
            batch.Put(encodePartitionDataKey([]byte("sorting1")), []byte("value123"))
            err := storageEngine.Batch(batch)
            
            Expect(err).Should(BeNil())
            
            ss, err := store.GetMatches([][]byte{ []byte("sorting1") })
            
            // will cause an error because the storage driver was closed at the end of makeStore()
            Expect(err).Should(BeNil())
            Expect(ss).Should(Not(BeNil()))
            
            Expect(ss.Next()).Should(BeFalse())
            Expect(ss.Key()).Should(BeNil())
            Expect(ss.Error().(DBerror).Code()).Should(Equal(EStorage.Code()))
        })
        
        It("should not modify the keys array it receives", func() {
            store := makeStore("nodeA")
            keys := [][]byte{ []byte("a"), []byte("b") }
            store.GetMatches(keys)

            Expect(keys[0]).Should(Equal([]byte("a")))
            Expect(keys[1]).Should(Equal([]byte("b")))
        })
    })
    
    Describe("#Batch", func() {
        It("should return EEmpty if the batch is nil", func() {
            store := makeStore("nodeA")
            ss, err := store.Batch(nil)
            
            Expect(err.(DBerror).Code()).Should(Equal(EEmpty.Code()))
            Expect(ss).Should(BeNil())
        })
        
        It("should return ELength if one of the sorting keys in an op is too long", func() {
            key := [MAX_SORTING_KEY_LENGTH+1]byte{ }
            updateBatch := NewUpdateBatch()
            _, err := updateBatch.Put(key[:], []byte("hello"), NewDVV(NewDot("", 0), map[string]uint64{ }))
            
            Expect(err.(DBerror).Code()).Should(Equal(ELength.Code()))
        })
        
        It("should return EEmpty if one of the sorting keys is empty", func() {
            key := []byte{ }
            updateBatch := NewUpdateBatch()
            _, err := updateBatch.Put(key[:], []byte("hello"), NewDVV(NewDot("", 0), map[string]uint64{ }))
            
            Expect(err.(DBerror).Code()).Should(Equal(EEmpty.Code()))
        })
        
        It("should return EStorage if the driver encounters an error", func() {
            key := []byte(randomString())
            updateBatch := NewUpdateBatch()
            updateBatch.Put(key, []byte("hello"), NewDVV(NewDot("", 0), map[string]uint64{ }))
        
            store := makeStore("nodeA")
            ss, err := store.Batch(updateBatch)
        
            // storage driver should be closed at this point
            Expect(err.(DBerror).Code()).Should(Equal(EStorage.Code()))
            Expect(ss).Should(BeNil())
        })
        
        It("should return EStorage if one of the ops affects a key that is already in the database but is corrupt or misformatted", func() {
            encodePartitionDataKey := func(k []byte) []byte {
                result := make([]byte, 0, len(PARTITION_DATA_PREFIX) + len(k))
                
                result = append(result, PARTITION_DATA_PREFIX...)
                result = append(result, k...)
                
                return result
            }
            
            storageEngine := makeNewStorageDriver()
            storageEngine.Open()
            defer storageEngine.Close()
            
            store := &Store{}
            store.Initialize("nodeA", storageEngine, MerkleMinDepth, nil)
            batch := NewBatch()
            batch.Put(encodePartitionDataKey([]byte("sorting1")), []byte("value123"))
            err := storageEngine.Batch(batch)
            
            Expect(err).Should(BeNil())
            
            updateBatch := NewUpdateBatch()
            updateBatch.Put([]byte("sorting1"), []byte("value123"), NewDVV(NewDot("", 0), map[string]uint64{ }))
            
            ss, err := store.Batch(updateBatch)
            
            // will cause an error because the storage driver was closed at the end of makeStore()
            Expect(err.(DBerror).Code()).Should(Equal(EStorage.Code()))
            Expect(ss).Should(BeNil())
        })
    })
    
    Describe("#Merge", func() {
        It("should return EEmpty if the sibling sets map is nil", func() {
            store := makeStore("nodeA")
            err := store.Merge(nil)
            
            Expect(err.(DBerror).Code()).Should(Equal(EEmpty.Code()))
        })
    })

    Describe("#Forget", func() {
        It("forget should result in no trace of a key being kept in the database including removing it from the merkle tree", func() {
            storageEngine := makeNewStorageDriver()
            storageEngine.Open()
            defer storageEngine.Close()
            
            store := &Store{}
            store.Initialize("nodeA", storageEngine, MerkleMinDepth, nil)
            updateBatch := NewUpdateBatch()
            updateBatch.Put([]byte("keyA"), []byte("value123"), NewDVV(NewDot("", 0), map[string]uint64{ }))
            updateBatch.Put([]byte("keyB"), []byte("value456"), NewDVV(NewDot("", 0), map[string]uint64{ }))
            
            _, err := store.Batch(updateBatch)
            
            Expect(err).Should(BeNil())
            
            values, err := store.Get([][]byte{ []byte("keyA"), []byte("keyB") })
    
            Expect(err).Should(BeNil())
            Expect(values[0].Value()).Should(Equal([]byte("value123")))
            Expect(values[0].IsTombstoneSet()).Should(BeFalse())
            Expect(values[1].Value()).Should(Equal([]byte("value456")))
            Expect(values[1].IsTombstoneSet()).Should(BeFalse())
            
            err = store.Forget([][]byte{ []byte("keyA") })

            Expect(err).Should(BeNil())
            
            values, err = store.Get([][]byte{ []byte("keyA"), []byte("keyB") })
    
            Expect(err).Should(BeNil())
            Expect(values[0]).Should(BeNil())
            Expect(values[1].Value()).Should(Equal([]byte("value456")))
            Expect(values[1].IsTombstoneSet()).Should(BeFalse())
            Expect(store.MerkleTree().RootHash()).Should(Equal(values[1].Hash([]byte("keyB"))))

            // test idempotence
            err = store.Forget([][]byte{ []byte("keyA") })

            Expect(err).Should(BeNil())
            
            values, err = store.Get([][]byte{ []byte("keyA"), []byte("keyB") })
    
            Expect(err).Should(BeNil())
            Expect(values[0]).Should(BeNil())
            Expect(values[1].Value()).Should(Equal([]byte("value456")))
            Expect(values[1].IsTombstoneSet()).Should(BeFalse())
            Expect(store.MerkleTree().RootHash()).Should(Equal(values[1].Hash([]byte("keyB"))))

            err = store.Forget([][]byte{ []byte("keyB") })

            Expect(err).Should(BeNil())
            
            values, err = store.Get([][]byte{ []byte("keyA"), []byte("keyB") })
    
            Expect(err).Should(BeNil())
            Expect(values[0]).Should(BeNil())
            Expect(values[1]).Should(BeNil())
            Expect(store.MerkleTree().RootHash()).Should(Equal(Hash{ }))

            err = store.Forget([][]byte{ []byte("keyB") })

            Expect(err).Should(BeNil())
            
            values, err = store.Get([][]byte{ []byte("keyA"), []byte("keyB") })
    
            Expect(err).Should(BeNil())
            Expect(values[0]).Should(BeNil())
            Expect(values[1]).Should(BeNil())
            Expect(store.MerkleTree().RootHash()).Should(Equal(Hash{ }))
        })

        It("should not modify the keys array it receives", func() {
            store := makeStore("nodeA")
            keys := [][]byte{ []byte("a"), []byte("b") }
            store.Forget(keys)

            Expect(keys[0]).Should(Equal([]byte("a")))
            Expect(keys[1]).Should(Equal([]byte("b")))
        })
    })
    
    Describe("#GarbageCollect", func() {
        Context("key A is a tombstone set that is older than 5 seconds", func() {    
            It("garbage collecting anything with a purge age of 5 seconds should result in key A being deleted from the data store and merkle leaf store", func() {
                storageEngine := makeNewStorageDriver()
                storageEngine.Open()
                defer storageEngine.Close()
                
                store := &Store{}
                store.Initialize("nodeA", storageEngine, MerkleMinDepth, nil)
                updateBatch := NewUpdateBatch()
                updateBatch.Put([]byte("keyA"), []byte("value123"), NewDVV(NewDot("", 0), map[string]uint64{ }))
                updateBatch.Put([]byte("keyB"), []byte("value456"), NewDVV(NewDot("", 0), map[string]uint64{ }))
                
                ss, err := store.Batch(updateBatch)
                
                Expect(err).Should(BeNil())
                
                values, err := store.Get([][]byte{ []byte("keyA"), []byte("keyB") })
        
                Expect(err).Should(BeNil())
                Expect(values[0].Value()).Should(Equal([]byte("value123")))
                Expect(values[0].IsTombstoneSet()).Should(BeFalse())
                Expect(values[1].Value()).Should(Equal([]byte("value456")))
                Expect(values[1].IsTombstoneSet()).Should(BeFalse())
                
                updateBatch = NewUpdateBatch()
                updateBatch.Delete([]byte("keyA"), NewDVV(NewDot("", 0), map[string]uint64{ }))
                
                ss, err = store.Batch(updateBatch)
                
                Expect(err).Should(BeNil())
                Expect(ss["keyA"].Value()).Should(BeNil())
                
                values, err = store.Get([][]byte{ []byte("keyA"), []byte("keyB") })
        
                Expect(err).Should(BeNil())
                Expect(values[0].Value()).Should(BeNil())
                Expect(values[0].IsTombstoneSet()).Should(BeTrue())
                Expect(values[1].Value()).Should(Equal([]byte("value456")))
                Expect(values[1].IsTombstoneSet()).Should(BeFalse())
                
                time.Sleep(time.Second * time.Duration(2))
                err = store.GarbageCollect(1000)
                
                Expect(err).Should(BeNil())
                
                values, err = store.Get([][]byte{ []byte("keyA"), []byte("keyB") })
        
                Expect(err).Should(BeNil())
                Expect(values[0]).Should(BeNil())
                Expect(values[1].Value()).Should(Equal([]byte("value456")))
                Expect(values[1].IsTombstoneSet()).Should(BeFalse())
                
                updateBatch.Delete([]byte("keyA"), NewDVV(NewDot("", 0), map[string]uint64{ }))
                updateBatch.Delete([]byte("keyB"), NewDVV(NewDot("", 0), map[string]uint64{ }))
            
                _, err = store.Batch(updateBatch)
                
                Expect(err).Should(BeNil())
                
                err = store.GarbageCollect(1000)
                
                Expect(err).Should(BeNil())
                
                values, err = store.Get([][]byte{ []byte("keyA"), []byte("keyB") })
        
                Expect(err).Should(BeNil())
                Expect(values[0]).Should(BeNil())
                Expect(values[1].Value()).Should(BeNil())
                Expect(values[1].IsTombstoneSet()).Should(BeTrue())
                
                time.Sleep(time.Second * time.Duration(2))
                
                err = store.GarbageCollect(1000)
                
                Expect(err).Should(BeNil())
                
                values, err = store.Get([][]byte{ []byte("keyA"), []byte("keyB") })
        
                Expect(err).Should(BeNil())
                Expect(values[0]).Should(BeNil())
                Expect(values[1]).Should(BeNil())
            })
        })
    })
    
    Context("a key does not exist in the node", func() {
        var (
            storageEngine StorageDriver
            store *Store
        )
        
        BeforeEach(func() {
            storageEngine = makeNewStorageDriver()
            storageEngine.Open()
            
            store = &Store{}
            store.Initialize("nodeA", storageEngine, MerkleMinDepth, nil)
        })
        
        AfterEach(func() {
            storageEngine.Close()
        })
        
        It("should report nil as the value at that key when queried", func() {
            key := []byte(randomString())
            values, err := store.Get([][]byte{ key })
        
            Expect(len(values)).Should(Equal(1))
            Expect(values[0]).Should(BeNil())
            Expect(err).Should(BeNil())
        })
        
        It("should not iterate over that key when queried with an iterator", func() {
            key := []byte(randomString())
            iter, err := store.GetMatches([][]byte{ key })
        
            Expect(iter).Should(Not(BeNil()))
            Expect(err).Should(BeNil())
            Expect(iter.Next()).Should(BeFalse())
        })    
    })
    
    Context("a key exists in the node", func() {
        var (
            storageEngine StorageDriver
            store *Store
            key []byte
        )
        
        BeforeEach(func() {
            storageEngine = makeNewStorageDriver()
            storageEngine.Open()
            
            store = &Store{}
            store.Initialize("nodeA", storageEngine, MerkleMinDepth, nil)
            
            key = []byte(randomString())
            updateBatch := NewUpdateBatch()
            updateBatch.Put(key, []byte("hello"), NewDVV(NewDot("", 0), map[string]uint64{ }))
            store.Batch(updateBatch)
        })
        
        AfterEach(func() {
            storageEngine.Close()
        })
        
        It("should contain a sibling set for that key when queried", func() {
            values, err := store.Get([][]byte{ key })
        
            Expect(err).Should(BeNil())
            Expect(len(values)).Should(Equal(1))
            Expect(values[0]).Should(Not(BeNil()))
            Expect(values[0].Size()).Should(Equal(1))
        
            for sibling := range values[0].Iter() {
                Expect(string(sibling.Value())).Should(Equal("hello"))
            }
            
            iter, err := store.GetMatches([][]byte{ key })
            
            Expect(err).Should(BeNil())
            Expect(iter.Next()).Should(BeTrue())
            Expect(iter.Key()).Should(Equal(key))
            
            siblingSet := iter.Value()
            
            Expect(siblingSet.Size()).Should(Equal(1))
            
            for sibling := range siblingSet.Iter() {
                Expect(string(sibling.Value())).Should(Equal("hello"))
            }
            
            Expect(iter.Next()).Should(BeFalse())
        })
        
        It("should remove all siblings from a sibling set when deleting that key with an empty context", func() {
            updateBatch := NewUpdateBatch()
            updateBatch.Delete(key, NewDVV(NewDot("", 0), map[string]uint64{ }))
            _, err := store.Batch(updateBatch)
            
            Expect(err).Should(BeNil())
            
            values, err := store.Get([][]byte{ key })
        
            Expect(err).Should(BeNil())
            Expect(len(values)).Should(Equal(1))
            Expect(values[0]).Should(Not(BeNil()))
            Expect(values[0].Size()).Should(Equal(1))
            Expect(values[0].IsTombstoneSet()).Should(BeTrue())
        })
        
        It("should add new siblings to a sibling set when putting that key with a parallel context", func() {
            updateBatch := NewUpdateBatch()
            updateBatch.Put(key, []byte("hello1"), NewDVV(NewDot("", 0), map[string]uint64{ "nodeA": 0 }))
            _, err := store.Batch(updateBatch)
            
            Expect(err).Should(BeNil())
            
            values, err := store.Get([][]byte{ key })
        
            Expect(err).Should(BeNil())
            Expect(len(values)).Should(Equal(1))
            Expect(values[0]).Should(Not(BeNil()))
            Expect(values[0].Size()).Should(Equal(2))
            
            for sibling := range values[0].Iter() {
                Expect(string(sibling.Value()) == "hello" || string(sibling.Value()) == "hello1").Should(BeTrue())
            }
        })
    })
})
