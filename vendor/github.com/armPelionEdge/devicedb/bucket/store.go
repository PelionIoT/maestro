
package bucket

import (
    "context"
    "io"
    "encoding/json"
    "encoding/binary"
    "time"
    "errors"
    "sort"
    "sync"
    "sync/atomic"

    . "github.com/armPelionEdge/devicedb/storage"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/logging"
    . "github.com/armPelionEdge/devicedb/merkle"
    . "github.com/armPelionEdge/devicedb/util"
    . "github.com/armPelionEdge/devicedb/resolver"
    . "github.com/armPelionEdge/devicedb/resolver/strategies"
)

const MAX_SORTING_KEY_LENGTH = 255
const StorageFormatVersion = "1"
const UpgradeFormatBatchSize = 100

var MASTER_MERKLE_TREE_PREFIX = []byte{ 0 }
var PARTITION_MERKLE_LEAF_PREFIX = []byte{ 1 }
var PARTITION_DATA_PREFIX = []byte{ 2 }
var NODE_METADATA_PREFIX = []byte{ 3 }

func NanoToMilli(v uint64) uint64 {
    return v / 1000000
}

func nodeBytes(node uint32) []byte {
    bytes := make([]byte, 4)
    
    binary.BigEndian.PutUint32(bytes, node)
    
    return bytes
}

func encodeMerkleLeafKey(nodeID uint32) []byte {
    nodeIDEncoding := nodeBytes(nodeID)
    result := make([]byte, 0, len(MASTER_MERKLE_TREE_PREFIX) + len(nodeIDEncoding))
    
    result = append(result, MASTER_MERKLE_TREE_PREFIX...)
    result = append(result, nodeIDEncoding...)
    
    return result
}

func decodeMerkleLeafKey(k []byte) (uint32, error) {
    k = k[len(MASTER_MERKLE_TREE_PREFIX):]
    
    if len(k) != 4 {
        return 0, errors.New("Invalid merkle leaf key")
    }
    
    return binary.BigEndian.Uint32(k[:4]), nil
}

func encodePartitionDataKey(k []byte) []byte {
    result := make([]byte, 0, len(PARTITION_DATA_PREFIX) + len(k))
    
    result = append(result, PARTITION_DATA_PREFIX...)
    result = append(result, k...)
    
    return result
}

func decodePartitionDataKey(k []byte) []byte {
    return k[len(PARTITION_DATA_PREFIX):]
}

func encodePartitionMerkleLeafKey(nodeID uint32, k []byte) []byte {
    nodeIDEncoding := nodeBytes(nodeID)
    result := make([]byte, 0, len(PARTITION_MERKLE_LEAF_PREFIX) + len(nodeIDEncoding) + len(k))
    
    result = append(result, PARTITION_MERKLE_LEAF_PREFIX...)
    result = append(result, nodeIDEncoding...)
    result = append(result, k...)
    
    return result
}

func decodePartitionMerkleLeafKey(k []byte) (uint32, []byte, error) {
    if len(k) < len(PARTITION_MERKLE_LEAF_PREFIX) + 5 {
        return 0, nil, errors.New("Invalid partition merkle leaf key")
    }
    
    k = k[len(PARTITION_MERKLE_LEAF_PREFIX):]
    
    return binary.BigEndian.Uint32(k[:4]), k[4:], nil
}

func encodeMetadataKey(k []byte) []byte {
    result := make([]byte, 0, len(NODE_METADATA_PREFIX) + len(k))
    result = append(result, PARTITION_MERKLE_LEAF_PREFIX...)
    result = append(result, k...)
    
    return result
}

type Store struct {
    nextRowID uint64    
    nodeID string
    storageDriver StorageDriver
    merkleTree *MerkleTree
    multiLock *MultiLock
    writesTryLock RWTryLock
    readsTryLock RWTryLock
    merkleLock *MultiLock
    conflictResolver ConflictResolver
    storageFormatVersion string
    monitor *Monitor
    watcherLock sync.Mutex
}

func (store *Store) Initialize(nodeID string, storageDriver StorageDriver, merkleDepth uint8, conflictResolver ConflictResolver) error {
    if conflictResolver == nil {
        conflictResolver = &MultiValue{}
    }

    store.nodeID = nodeID
    store.storageDriver = storageDriver
    store.multiLock = NewMultiLock()
    store.merkleLock = NewMultiLock()
    store.conflictResolver = conflictResolver
    store.merkleTree, _ = NewMerkleTree(merkleDepth)
    
    var err error
    dbMerkleDepth, storageFormatVersion, err := store.getStoreMetadata()
    
    if err != nil {
        Log.Errorf("Error retrieving database metadata for node %s: %v", nodeID, err)
        
        return err
    }

    store.storageFormatVersion = storageFormatVersion
    
    if dbMerkleDepth != merkleDepth || storageFormatVersion != StorageFormatVersion {
        if dbMerkleDepth != merkleDepth {
            Log.Debugf("Initializing node %s rebuilding merkle leafs with depth %d", nodeID, merkleDepth)
            
            err = store.RebuildMerkleLeafs()
            
            if err != nil {
                Log.Errorf("Error rebuilding merkle leafs for node %s: %v", nodeID, err)
                
                return err
            }
        }

        if storageFormatVersion != StorageFormatVersion {
            Log.Debugf("Initializing node %s. It is using an older storage format. Its keys need to be updated to the new storage format...", nodeID)

            err = store.UpgradeStorageFormat()

            if err != nil {
                Log.Errorf("Error while upgrading the storage format for node %s: %v", nodeID, err)

                return err
            }

            Log.Debugf("Upgrade to the latest storage format was successful. Now at storage format %s", StorageFormatVersion)
        }
        
        err = store.RecordMetadata()
        
        if err != nil {
            Log.Errorf("Error recording merkle depth metadata for node %s: %v", nodeID, err)
            
            return err
        }
    }

    err = store.calculateNextRowID()

    if err != nil {
        Log.Errorf("Error attempting to determine what the next row ID should be at node %s: %v", nodeID, err)

        return err
    }
    
    err = store.initializeMerkleTree()
    
    if err != nil {
        Log.Errorf("Error initializing node %s: %v", nodeID, err)
        
        return err
    }

    if store.nextRowID == 0 {
        store.monitor = NewMonitor(0)
    } else {
        store.monitor = NewMonitor(store.nextRowID - 1)
    }
    
    return nil
}

func (store *Store) initializeMerkleTree() error {
    iter, err := store.storageDriver.GetMatches([][]byte{ MASTER_MERKLE_TREE_PREFIX })
    
    if err != nil {
        return err
    }
    
    defer iter.Release()
    
    for iter.Next() {
        key := iter.Key()
        value := iter.Value()
        nodeID, err := decodeMerkleLeafKey(key)
            
        if err != nil {
            return err
        }
        
        if !store.merkleTree.IsLeaf(nodeID) {
            return errors.New("Invalid leaf node in master merkle keys")
        }
        
        hash := Hash{ }
        high := binary.BigEndian.Uint64(value[:8])
        low := binary.BigEndian.Uint64(value[8:])
        hash = hash.SetLow(low).SetHigh(high)
    
        store.merkleTree.UpdateLeafHash(nodeID, hash)
    }
    
    if iter.Error() != nil {
        return iter.Error()
    }
    
    return nil
}

func (store *Store) calculateNextRowID() error {
    iter, err := store.GetAll()

    if err != nil {
        return err
    }

    store.nextRowID = 0
    defer iter.Release()

    for iter.Next() {
        if iter.LocalVersion() >= store.nextRowID {
            store.nextRowID = iter.LocalVersion() + 1
        }
    }

    Log.Infof("Next row ID = %d", store.nextRowID)

    return iter.Error()
}

func (store *Store) getStoreMetadata() (uint8, string, error) {
    values, err := store.storageDriver.Get([][]byte{ encodeMetadataKey([]byte("merkleDepth")), encodeMetadataKey([]byte("storageFormatVersion")) })
    
    if err != nil {
        return 0, "", err
    }
    
    var merkleDepth uint8
    var storageFormatVersion string = "0"

    if values[0] != nil {
        merkleDepth = uint8(values[0][0])
    }

    if values[1] != nil {
        storageFormatVersion = string(values[1])
    }

    return merkleDepth, storageFormatVersion, nil
}

func (store *Store) RecordMetadata() error {
    batch := NewBatch()
    
    batch.Put(encodeMetadataKey([]byte("merkleDepth")), []byte{ byte(store.merkleTree.Depth()) })
    batch.Put(encodeMetadataKey([]byte("storageFormatVersion")), []byte(StorageFormatVersion))
    
    err := store.storageDriver.Batch(batch)
    
    if err != nil {
        return err
    }
    
    return nil
}

func (store *Store) RebuildMerkleLeafs() error {
    // Delete all keys starting with MASTER_MERKLE_TREE_PREFIX or PARTITION_MERKLE_LEAF_PREFIX
    iter, err := store.storageDriver.GetMatches([][]byte{ MASTER_MERKLE_TREE_PREFIX, PARTITION_MERKLE_LEAF_PREFIX })
    
    if err != nil {
        return err
    }
    
    defer iter.Release()
    
    for iter.Next() {
        batch := NewBatch()
        batch.Delete(iter.Key())
        err := store.storageDriver.Batch(batch)
        
        if err != nil {
            return err
        }
    }
    
    if iter.Error() != nil {
        return iter.Error()
    }
    
    iter.Release()

    // Scan through all the keys in this node and rebuild the merkle tree
    merkleTree, _ := NewMerkleTree(store.merkleTree.Depth())
    iter, err = store.storageDriver.GetMatches([][]byte{ PARTITION_DATA_PREFIX })
    
    if err != nil {
        return err
    }
    
    siblingSetIterator := NewBasicSiblingSetIterator(iter, store.storageFormatVersion)
    
    defer siblingSetIterator.Release()
    
    for siblingSetIterator.Next() {
        key := siblingSetIterator.Key()
        siblingSet := siblingSetIterator.Value()
        update := NewUpdate().AddDiff(string(key), nil, siblingSet)
        
        _, leafNodes := merkleTree.Update(update)
        batch := NewBatch()
        
        for leafID, _ := range leafNodes {
            for key, _:= range leafNodes[leafID] {
                batch.Put(encodePartitionMerkleLeafKey(leafID, []byte(key)), []byte{ })
            }
        }
        
        err := store.storageDriver.Batch(batch)
        
        if err != nil {
            return err
        }
    }
    
    for leafID := uint32(1); leafID < merkleTree.NodeLimit(); leafID += 2 {
        batch := NewBatch()
        
        if merkleTree.NodeHash(leafID).High() != 0 || merkleTree.NodeHash(leafID).Low() != 0 {
            leafHash := merkleTree.NodeHash(leafID).Bytes()
            batch.Put(encodeMerkleLeafKey(leafID), leafHash[:])
            
            err := store.storageDriver.Batch(batch)
            
            if err != nil {
                return err
            }
        }
    }
    
    return siblingSetIterator.Error()
}

func (store *Store) UpgradeStorageFormat() error {
    iter, err := store.GetAll()

    if err != nil {
        Log.Errorf("Unable to create iterator to upgrade storage format: %v", err.Error())

        return err
    }

    store.nextRowID = 0
    defer iter.Release()

    batchSize := 0
    batch := NewBatch()

    for iter.Next() {
        key := iter.Key()
        value := iter.Value()
        row := &Row{ LocalVersion: store.nextRowID, Siblings: value }

        store.nextRowID++

        batch.Put(encodePartitionDataKey(key), row.Encode())
        batchSize++

        if batchSize == UpgradeFormatBatchSize {
            if err := store.storageDriver.Batch(batch); err != nil {
                return err
            }

            batch = NewBatch()
            batchSize = 0
        }
    }

    if batchSize > 0 {
        if err := store.storageDriver.Batch(batch); err != nil {
            return err
        }
    }

    if iter.Error() != nil {
        Log.Errorf("Unable to iterate through store to upgrade storage format: %v", err.Error())

        return iter.Error()
    }

    store.storageFormatVersion = StorageFormatVersion

    return nil
}

func (store *Store) MerkleTree() *MerkleTree {
    return store.merkleTree
}

func (store *Store) GarbageCollect(tombstonePurgeAge uint64) error {
    iter, err := store.storageDriver.GetMatches([][]byte{ PARTITION_DATA_PREFIX })
    
    if err != nil {
        Log.Errorf("Garbage collection error: %s", err.Error())
            
        return EStorage
    }
    
    now := NanoToMilli(uint64(time.Now().UnixNano()))
    siblingSetIterator := NewBasicSiblingSetIterator(iter, store.storageFormatVersion)
    defer siblingSetIterator.Release()
    
    if tombstonePurgeAge > now {
        tombstonePurgeAge = now
    }
    
    for siblingSetIterator.Next() {
        var err error
        key := siblingSetIterator.Key()
        ssInitial := siblingSetIterator.Value()
        
        if !ssInitial.CanPurge(now - tombstonePurgeAge) {
            continue
        }
        
        store.lock([][]byte{ key })
        
        func() {
            // the key must be re-queried because at the time of iteration we did not have a lock
            // on the key in order to update it
            siblingSets, err := store.Get([][]byte{ key })
            
            if err != nil {
                return
            }
            
            siblingSet := siblingSets[0]
            
            if siblingSet == nil {
                return
            }
        
            if !siblingSet.CanPurge(now - tombstonePurgeAge) {
                return
            }
        
            Log.Debugf("GC: Purge tombstone at key %s. It is older than %d milliseconds", string(key), tombstonePurgeAge)
            leafID := store.merkleTree.LeafNode(key)
            
            batch := NewBatch()
            batch.Delete(encodePartitionMerkleLeafKey(leafID, key))
            batch.Delete(encodePartitionDataKey(key))
        
            err = store.storageDriver.Batch(batch)
        }()
        
        store.unlock([][]byte{ key }, false)
        
        if err != nil {
            Log.Errorf("Garbage collection error: %s", err.Error())
            
            return EStorage
        }
    }
    
    if iter.Error() != nil {
        Log.Errorf("Garbage collection error: %s", iter.Error().Error())
        
        return EStorage
    }
    
    return nil
}

func (store *Store) Get(keys [][]byte) ([]*SiblingSet, error) {
    if !store.readsTryLock.TryRLock() {
        return nil, EOperationLocked
    }

    defer store.readsTryLock.RUnlock()

    if len(keys) == 0 {
        Log.Warningf("Passed empty keys parameter in Get(%v)", keys)
        
        return nil, EEmpty
    }

    // Make a new array since we don't want to modify the input array
    keysCopy := make([][]byte, len(keys))

    for i := 0; i < len(keys); i += 1 {
        if len(keys[i]) == 0 {
            Log.Warningf("Passed empty key in Get(%v)", keys)
            
            return nil, EEmpty
        }
        
        if len(keys[i]) > MAX_SORTING_KEY_LENGTH {
            Log.Warningf("Key is too long %d > %d in Get(%v)", len(keys[i]), MAX_SORTING_KEY_LENGTH, keys)
            
            return nil, ELength
        }
        
        keysCopy[i] = encodePartitionDataKey(keys[i])
    }
    
    // use storage driver
    values, err := store.storageDriver.Get(keysCopy)
    
    if err != nil {
        Log.Errorf("Storage driver error in Get(%v): %s", keys, err.Error())
        
        return nil, EStorage
    }
    
    siblingSetList := make([]*SiblingSet, len(keys))
    
    for i := 0; i < len(keys); i += 1 {
        if values[i] == nil {
            siblingSetList[i] = nil
            
            continue
        }
        
        var row Row
        
        err := row.Decode(values[i], store.storageFormatVersion)
        
        if err != nil {
            Log.Errorf("Storage driver error in Get(%v): %s", keys, err.Error())
            
            return nil, EStorage
        }
        
        siblingSetList[i] = row.Siblings
    }
    
    return siblingSetList, nil
}

func (store *Store) GetMatches(keys [][]byte) (SiblingSetIterator, error) {
    if !store.readsTryLock.TryRLock() {
        return nil, EOperationLocked
    }

    defer store.readsTryLock.RUnlock()

    if len(keys) == 0 {
        Log.Warningf("Passed empty keys parameter in GetMatches(%v)", keys)
        
        return nil, EEmpty
    }
   
    // Make a new array since we don't want to modify the input array
    keysCopy := make([][]byte, len(keys))

    for i := 0; i < len(keys); i += 1 {
        if len(keys[i]) == 0 {
            Log.Warningf("Passed empty key in GetMatches(%v)", keys)
            
            return nil, EEmpty
        }
        
        if len(keys[i]) > MAX_SORTING_KEY_LENGTH {
            Log.Warningf("Key is too long %d > %d in GetMatches(%v)", len(keys[i]), MAX_SORTING_KEY_LENGTH, keys)
            
            return nil, ELength
        }
        
        keysCopy[i] = encodePartitionDataKey(keys[i])
    }
    
    iter, err := store.storageDriver.GetMatches(keysCopy)
    
    if err != nil {
        Log.Errorf("Storage driver error in GetMatches(%v): %s", keys, err.Error())
            
        return nil, EStorage
    }
    
    return NewBasicSiblingSetIterator(iter, store.storageFormatVersion), nil
}

func (store *Store) GetAll() (SiblingSetIterator, error) {
    if !store.readsTryLock.TryRLock() {
        return nil, EOperationLocked
    }

    defer store.readsTryLock.RUnlock()

    iter, err := store.storageDriver.GetMatches([][]byte{ encodePartitionDataKey([]byte{ }) })

    if err != nil {
        Log.Errorf("Storage driver error in GetAll(): %s", err.Error())
            
        return nil, EStorage
    }
    
    return NewBasicSiblingSetIterator(iter, store.storageFormatVersion), nil
}

func (store *Store) GetSyncChildren(nodeID uint32) (SiblingSetIterator, error) {
    if !store.readsTryLock.TryRLock() {
        return nil, EOperationLocked
    }

    defer store.readsTryLock.RUnlock()

    if nodeID >= store.merkleTree.NodeLimit() {
        return nil, EMerkleRange
    }
    
    min := encodePartitionMerkleLeafKey(store.merkleTree.SubRangeMin(nodeID), []byte{})
    max := encodePartitionMerkleLeafKey(store.merkleTree.SubRangeMax(nodeID), []byte{})
    
    iter, err := store.storageDriver.GetRange(min, max)
    
    if err != nil {
        Log.Errorf("Storage driver error in GetSyncChildren(%v): %s", nodeID, err.Error())
        
        return nil, EStorage
    }

    return NewMerkleChildrenIterator(iter, store.storageDriver, store.storageFormatVersion), nil
}

func (store *Store) Forget(keys [][]byte) error {
    for _, key := range keys {
        if key == nil {
            continue
        }

        store.lock([][]byte{ key })
        
        siblingSets, err := store.Get([][]byte{ key })
        
        if err != nil {
            Log.Errorf("Unable to forget key %s due to storage error: %v", string(key), err)

            store.unlock([][]byte{ key }, false)

            return EStorage
        }
        
        siblingSet := siblingSets[0]
        
        if siblingSet == nil {
            store.unlock([][]byte{ key }, false)

            continue
        }

        // Update merkle tree to reflect deletion
        leafID := store.merkleTree.LeafNode(key)
        newLeafHash := store.merkleTree.NodeHash(leafID).Xor(siblingSet.Hash(key))
        store.merkleTree.UpdateLeafHash(leafID, newLeafHash)

        batch := NewBatch()
        batch.Delete(encodePartitionMerkleLeafKey(leafID, key))
        batch.Delete(encodePartitionDataKey(key))
        leafHashBytes := newLeafHash.Bytes()
        batch.Put(encodeMerkleLeafKey(leafID), leafHashBytes[:])
    
        err = store.storageDriver.Batch(batch)
        
        store.unlock([][]byte{ key }, false)
        
        if err != nil {
            Log.Errorf("Unable to forget key %s due to storage error: %v", string(key), err.Error())
            
            return EStorage
        }

        Log.Debugf("Forgot key %s", string(key))
    }
    
    return nil
}

func (store *Store) updateInit(keys [][]byte) (map[string]*SiblingSet, error) {
    siblingSetMap := map[string]*SiblingSet{ }
    
    // db objects
    for i := 0; i < len(keys); i += 1 {
        keys[i] = encodePartitionDataKey(keys[i])
    }
    
    values, err := store.storageDriver.Get(keys)
    
    if err != nil {
        Log.Errorf("Storage driver error in updateInit(%v): %s", keys, err.Error())
        
        return nil, EStorage
    }
    
    for i := 0; i < len(keys); i += 1 {
        var row Row
        key := decodePartitionDataKey(keys[i])
        siblingSetBytes := values[0]
        
        if siblingSetBytes == nil {
            siblingSetMap[string(key)] = NewSiblingSet(map[*Sibling]bool{ })
        } else {
            err := row.Decode(siblingSetBytes, store.storageFormatVersion)
            
            if err != nil {
                Log.Warningf("Could not decode sibling set in updateInit(%v): %s", keys, err.Error())
                
                return nil, EStorage
            }
            
            siblingSetMap[string(key)] = row.Siblings
        }
        
        values = values[1:]
    }
    
    return siblingSetMap, nil
}

func (store *Store) batch(update *Update, merkleTree *MerkleTree) (*Batch, []Row) {
    _, leafNodes := merkleTree.Update(update)
    batch := NewBatch()
    updatedRows := make([]Row, 0, update.Size())

    // WRITE PARTITION MERKLE LEAFS
    for leafID, _ := range leafNodes {
        leafHash := merkleTree.NodeHash(leafID).Bytes()
        batch.Put(encodeMerkleLeafKey(leafID), leafHash[:])
        
        for key, _ := range leafNodes[leafID] {
            batch.Put(encodePartitionMerkleLeafKey(leafID, []byte(key)), []byte{ })
        }
    }
    
    // WRITE PARTITION OBJECTS
    // atomically increment nextRowID by the amount of new IDs we need (one per key) to allocate enough IDs
    // for this batch update.
    nextRowID := atomic.AddUint64(&store.nextRowID, uint64(update.Size())) - uint64(update.Size())

    for diff := range update.Iter() {
        key := []byte(diff.Key())
        siblingSet := diff.NewSiblingSet()
        row := &Row{ Key: diff.Key(), LocalVersion: nextRowID, Siblings: siblingSet }
        updatedRows = append(updatedRows, *row)

        nextRowID++
        
        batch.Put(encodePartitionDataKey(key), row.Encode())
    }

    return batch, updatedRows
}

func (store *Store) updateToSibling(o Op, c *DVV, oldestTombstone *Sibling) *Sibling {
    if o.IsDelete() {
        if oldestTombstone == nil {
            return NewSibling(c, nil, NanoToMilli(uint64(time.Now().UnixNano())))
        } else {
            return NewSibling(c, nil, oldestTombstone.Timestamp())
        }
    } else {
        return NewSibling(c, o.Value(), NanoToMilli(uint64(time.Now().UnixNano())))
    }
}

func (store *Store) Batch(batch *UpdateBatch) (map[string]*SiblingSet, error) {
    if !store.writesTryLock.TryRLock() {
        return nil, EOperationLocked
    }

    defer store.writesTryLock.RUnlock()

    if batch == nil {
        Log.Warningf("Passed nil batch parameter in Batch(%v)", batch)
        
        return nil, EEmpty
    }
        
    keys := make([][]byte, 0, len(batch.Batch().Ops()))
    update := NewUpdate()

    for key, _ := range batch.Batch().Ops() {
        keyBytes := []byte(key)
        
        keys = append(keys, keyBytes)
    }
    
    store.lock(keys)
    defer store.unlock(keys, true)

    merkleTree := store.merkleTree
    siblingSets, err := store.updateInit(keys)
    
    //return nil, nil
    if err != nil {
        return nil, err
    }
    
    for key, op := range batch.Batch().Ops() {
        context := batch.Context()[key]
        siblingSet := siblingSets[key]
    
        if siblingSet.Size() == 0 && op.IsDelete() {
            delete(siblingSets, key)
            
            continue
        }
        
        updateContext := context.Context()
        
        if len(updateContext) == 0 {
            updateContext = siblingSet.Join()
        }
        
        updateClock := siblingSet.Event(updateContext, store.nodeID)
        var newSibling *Sibling
        
        if siblingSet.IsTombstoneSet() {
            newSibling = store.updateToSibling(op, updateClock, siblingSet.GetOldestTombstone())
        } else {
            newSibling = store.updateToSibling(op, updateClock, nil)
        }
        
        updatedSiblingSet := siblingSet.Discard(updateClock).Sync(NewSiblingSet(map[*Sibling]bool{ newSibling: true }))
    
        siblingSets[key] = updatedSiblingSet
        
        updatedSiblingSet = store.conflictResolver.ResolveConflicts(updatedSiblingSet)
        
        update.AddDiff(key, siblingSet, updatedSiblingSet)
    }
    
    storageBatch, updatedRows := store.batch(update, merkleTree)
    err = store.storageDriver.Batch(storageBatch)
    
    if err != nil {
        Log.Errorf("Storage driver error in Batch(%v): %s", batch, err.Error())

        store.discardIDRange(updatedRows)
        store.merkleTree.UndoUpdate(update)
        
        return nil, EStorage
    }

    store.notifyWatchers(updatedRows)
    
    return siblingSets, nil
}

func (store *Store) Merge(siblingSets map[string]*SiblingSet) error {
    if !store.writesTryLock.TryRLock() {
        return EOperationLocked
    }

    defer store.writesTryLock.RUnlock()

    if siblingSets == nil {
        Log.Warningf("Passed nil sibling sets in Merge(%v)", siblingSets)
        
        return EEmpty
    }
    
    keys := make([][]byte, 0, len(siblingSets))
    
    for key, _ := range siblingSets {
        keys = append(keys, []byte(key))
    }
    
    store.lock(keys)
    defer store.unlock(keys, true)
    
    merkleTree := store.merkleTree
    mySiblingSets, err := store.updateInit(keys)
    
    if err != nil {
        return err
    }
    
    update := NewUpdate()
        
    for _, key := range keys {
        key = decodePartitionDataKey(key)
        siblingSet := siblingSets[string(key)]
        mySiblingSet := mySiblingSets[string(key)]
        
        if siblingSet == nil {
            continue
        }
        
        if mySiblingSet == nil {
            mySiblingSet = NewSiblingSet(map[*Sibling]bool{ })
        }

        updatedSiblingSet := mySiblingSet.MergeSync(siblingSet, store.nodeID)

        for sibling := range updatedSiblingSet.Iter() {
            if !mySiblingSet.Has(sibling) {
                updatedSiblingSet = store.conflictResolver.ResolveConflicts(updatedSiblingSet)
                
                update.AddDiff(string(key), mySiblingSet, updatedSiblingSet)
            }
        }
    }

    if update.Size() != 0 {
        batch, updatedRows := store.batch(update, merkleTree)
        err := store.storageDriver.Batch(batch)

        if err != nil {
            Log.Errorf("Storage driver error in Merge(%v): %s", siblingSets, err.Error())

            store.discardIDRange(updatedRows)
            store.merkleTree.UndoUpdate(update)

            return EStorage
        }

        store.notifyWatchers(updatedRows)
    }
    
    return nil
}

func (store *Store) Watch(ctx context.Context, keys [][]byte, prefixes [][]byte, localVersion uint64, ch chan Row) {
    store.addWatcher(ctx, keys, prefixes, localVersion, ch)
}

func (store *Store) addWatcher(ctx context.Context, keys [][]byte, prefixes [][]byte, localVersion uint64, ch chan Row) error {
    store.watcherLock.Lock()
    defer store.watcherLock.Unlock()

    keysCopy := make([][]byte, len(keys))

    for i := 0; i < len(keys); i += 1 {
        keysCopy[i] = encodePartitionDataKey(keys[i])
    }
    
    // use storage driver
    values, err := store.storageDriver.Get(keysCopy)
    
    if err != nil {
        Log.Errorf("Storage driver error in addWatcher(): %s", err.Error())
        
        close(ch)

        return EStorage
    }
    
    for i := 0; i < len(keys); i += 1 {
        if values[i] == nil {
            continue
        }
        
        var row Row
        
        err := row.Decode(values[i], store.storageFormatVersion)
        
        if err != nil {
            Log.Errorf("Storage driver error in addWatcher(): %s", err.Error())
            
            close(ch)

            return EStorage
        }

        if row.LocalVersion > localVersion {
            ch <- row
        }
    }

    prefixesCopy := make([][]byte, len(prefixes))

    for i := 0; i < len(prefixes); i += 1 {
        prefixesCopy[i] = encodePartitionDataKey(prefixes[i])
    }
    
    iter, err := store.storageDriver.GetMatches(prefixesCopy)
    
    if err != nil {
        Log.Errorf("Storage driver error in addWatcher(): %s", err.Error())
            
        close(ch)

        return EStorage
    }
    
    ssIter := NewBasicSiblingSetIterator(iter, store.storageFormatVersion)

    for ssIter.Next() {
        if ssIter.LocalVersion() <= localVersion {
            continue
        }

        var row Row
        
        row.Key = string(ssIter.Key())
        row.LocalVersion = ssIter.LocalVersion()
        row.Siblings = ssIter.Value()

        ch <- row
    }

    if ssIter.Error() != nil {
        Log.Errorf("Storage driver error in addWatcher(): %s", err.Error())

        close(ch)

        return EStorage
    }

    ch <- Row{}
    
    store.monitor.AddListener(ctx, keys, prefixes, ch)

    return nil
}

func (store *Store) notifyWatchers(updatedRows []Row) {
    store.watcherLock.Lock()
    defer store.watcherLock.Unlock()    

    // submit update to watcher collection
    for _, update := range updatedRows {
        store.monitor.Notify(update)
    }
}

func (store *Store) discardIDRange(updatedRows []Row) {
    store.watcherLock.Lock()
    defer store.watcherLock.Unlock()

    if len(updatedRows) == 0 {
        return
    }

    store.monitor.DiscardIDRange(updatedRows[0].LocalVersion, updatedRows[len(updatedRows) - 1].LocalVersion)
}

func (store *Store) sortedLockKeys(keys [][]byte) ([]string, []string) {
    leafSet := make(map[string]bool, len(keys))
    keyStrings := make([]string, 0, len(keys))
    nodeStrings := make([]string, 0, len(keys))
    
    for _, key := range keys {
        keyStrings = append(keyStrings, string(key))
        leafSet[string(nodeBytes(store.merkleTree.LeafNode(key)))] = true
    }
    
    for node, _ := range leafSet {
        nodeStrings = append(nodeStrings, node)
    }
    
    sort.Strings(keyStrings)
    sort.Strings(nodeStrings)
    
    return keyStrings, nodeStrings
}

func (store *Store) lock(keys [][]byte) {
    keyStrings, nodeStrings := store.sortedLockKeys(keys)
    
    for _, key := range keyStrings {
        store.multiLock.Lock([]byte(key))
    }
    
    for _, key := range nodeStrings {
        store.merkleLock.Lock([]byte(key))
    }
}

func (store *Store) unlock(keys [][]byte, keysArePrefixed bool) {
    tKeys := make([][]byte, 0, len(keys))
    
    for _, key := range keys {
        if keysArePrefixed {
            tKeys = append(tKeys, key[1:])
        } else {
            tKeys = append(tKeys, key)
        }
    }
    
    keyStrings, nodeStrings := store.sortedLockKeys(tKeys)
    
    for _, key := range keyStrings {
        store.multiLock.Unlock([]byte(key))
    }
    
    for _, key := range nodeStrings {
        store.merkleLock.Unlock([]byte(key))
    }
}

func (store *Store) LockWrites() {
    store.writesTryLock.WLock()
}

func (store *Store) UnlockWrites() {
    store.writesTryLock.WUnlock()
}

func (store *Store) LockReads() {
    store.readsTryLock.WLock()
}

func (store *Store) UnlockReads() {
    store.readsTryLock.WUnlock()
}

type UpdateBatch struct {
    RawBatch *Batch `json:"batch"`
    Contexts map[string]*DVV `json:"context"`
}

func NewUpdateBatch() *UpdateBatch {
    return &UpdateBatch{ NewBatch(), map[string]*DVV{ } }
}

func (updateBatch *UpdateBatch) Batch() *Batch {
    return updateBatch.RawBatch
}

func (updateBatch *UpdateBatch) Context() map[string]*DVV {
    return updateBatch.Contexts
}

func (updateBatch *UpdateBatch) ToJSON() ([]byte, error) {
    return json.Marshal(updateBatch)
}

func (updateBatch *UpdateBatch) FromJSON(reader io.Reader) error {
    var tempUpdateBatch UpdateBatch
    
    decoder := json.NewDecoder(reader)
    err := decoder.Decode(&tempUpdateBatch)
    
    if err != nil {
        return err
    }
    
    updateBatch.Contexts = map[string]*DVV{ }
    updateBatch.RawBatch = NewBatch()
    
    for k, op := range tempUpdateBatch.Batch().Ops() {
        context, ok := tempUpdateBatch.Context()[k]
        
        if !ok || context == nil {
            context = NewDVV(NewDot("", 0), map[string]uint64{ })
        }
        
        err = nil
    
        if op.IsDelete() {
            _, err = updateBatch.Delete(op.Key(), context)
        } else {
            _, err = updateBatch.Put(op.Key(), op.Value(), context)
        }
        
        if err != nil {
            return err
        }
    }
    
    return nil
}

func (updateBatch *UpdateBatch) Put(key []byte, value []byte, context *DVV) (*UpdateBatch, error) {
    if len(key) == 0 {
        Log.Warningf("Passed an empty key to Put(%v, %v, %v)", key, value, context)
        
        return nil, EEmpty
    }
    
    if len(key) > MAX_SORTING_KEY_LENGTH {
        Log.Warningf("Key is too long %d > %d in Put(%v, %v, %v)", len(key), MAX_SORTING_KEY_LENGTH, key, value, context)
        
        return nil, ELength
    }
    
    if context == nil {
        Log.Warningf("Passed a nil context to Put(%v, %v, %v)", key, value, context)
        
        return nil, EEmpty
    }
    
    if value == nil {
        value = []byte{ }
    }
    
    updateBatch.Batch().Put(key, value)
    updateBatch.Context()[string(key)] = context
    
    return updateBatch, nil
}

func (updateBatch *UpdateBatch) Delete(key []byte, context *DVV) (*UpdateBatch, error) {
    if len(key) == 0 {
        Log.Warningf("Passed an empty key to Delete(%v, %v)", key, context)
        
        return nil, EEmpty
    }
    
    if len(key) > MAX_SORTING_KEY_LENGTH {
        Log.Warningf("Key is too long %d > %d in Delete(%v, %v)", len(key), MAX_SORTING_KEY_LENGTH, key, context)
        
        return nil, ELength
    }
    
    if context == nil {
        Log.Warningf("Passed a nil context to Delete(%v, %v)", key, context)
        
        return nil, EEmpty
    }
    
    updateBatch.Batch().Delete(key)
    updateBatch.Context()[string(key)] = context
    
    return updateBatch, nil
}

type MerkleChildrenIterator struct {
    dbIterator StorageIterator
    storageDriver StorageDriver
    parseError error
    currentKey []byte
    currentValue *SiblingSet
    storageFormatVersion string
    currentLocalVersion uint64
}

func NewMerkleChildrenIterator(iter StorageIterator, storageDriver StorageDriver, storageFormatVersion string) *MerkleChildrenIterator {
    return &MerkleChildrenIterator{ iter, storageDriver, nil, nil, nil, storageFormatVersion, 0 }
    // not actually the prefix for all keys in the range, but it will be a consistent length
    // prefix := encodePartitionMerkleLeafKey(nodeID, []byte{ })
}

func (mIterator *MerkleChildrenIterator) Next() bool {
    mIterator.currentKey = nil
    mIterator.currentValue = nil
    mIterator.currentLocalVersion = 0
    
    if !mIterator.dbIterator.Next() {
        if mIterator.dbIterator.Error() != nil {
            Log.Errorf("Storage driver error in Next(): %s", mIterator.dbIterator.Error())
        }
        
        mIterator.Release()
        
        return false
    }
    
    _, key, err := decodePartitionMerkleLeafKey(mIterator.dbIterator.Key())
    
    if err != nil {
        Log.Errorf("Corrupt partition merkle leaf key in Next(): %v", mIterator.dbIterator.Key())
        
        mIterator.Release()
        
        return false
    }
    
    values, err := mIterator.storageDriver.Get([][]byte{ encodePartitionDataKey(key) })
    
    if err != nil {
        Log.Errorf("Storage driver error in Next(): %s", err)
        
        mIterator.Release()
        
        return false
    }
    
    value := values[0]

    var row Row
    
    mIterator.parseError = row.Decode(value, mIterator.storageFormatVersion)
    
    if mIterator.parseError != nil {
        Log.Errorf("Storage driver error in Next() key = %v, value = %v: %s", key, value, mIterator.parseError.Error())
        
        mIterator.Release()
        
        return false
    }
    
    mIterator.currentKey = key
    mIterator.currentValue = row.Siblings
    mIterator.currentLocalVersion = row.LocalVersion
    
    return true
}

func (mIterator *MerkleChildrenIterator) Prefix() []byte {
    return nil
}

func (mIterator *MerkleChildrenIterator) Key() []byte {
    return mIterator.currentKey
}

func (mIterator *MerkleChildrenIterator) Value() *SiblingSet {
    return mIterator.currentValue
}

func (mIterator *MerkleChildrenIterator) LocalVersion() uint64 {
    return mIterator.currentLocalVersion
}

func (mIterator *MerkleChildrenIterator) Release() {
    mIterator.dbIterator.Release()
}

func (mIterator *MerkleChildrenIterator) Error() error {
    if mIterator.parseError != nil {
        return EStorage
    }
        
    if mIterator.dbIterator.Error() != nil {
        return EStorage
    }
    
    return nil
}

type BasicSiblingSetIterator struct {
    dbIterator StorageIterator
    parseError error
    currentKey []byte
    currentValue *SiblingSet
    storageFormatVersion string
    currentLocalVersion uint64    
}

func NewBasicSiblingSetIterator(dbIterator StorageIterator, storageFormatVersion string) *BasicSiblingSetIterator {
    return &BasicSiblingSetIterator{ dbIterator, nil, nil, nil, storageFormatVersion, 0 }
}

func (ssIterator *BasicSiblingSetIterator) Next() bool {
    ssIterator.currentKey = nil
    ssIterator.currentValue = nil
    ssIterator.currentLocalVersion = 0
    
    if !ssIterator.dbIterator.Next() {
        if ssIterator.dbIterator.Error() != nil {
            Log.Errorf("Storage driver error in Next(): %s", ssIterator.dbIterator.Error())
        }
        
        return false
    }

    var row Row    
    
    ssIterator.parseError = row.Decode(ssIterator.dbIterator.Value(), ssIterator.storageFormatVersion)
    
    if ssIterator.parseError != nil {
        Log.Errorf("Storage driver error in Next() key = %v, value = %v: %s", ssIterator.dbIterator.Key(), ssIterator.dbIterator.Value(), ssIterator.parseError.Error())
        
        ssIterator.Release()
        
        return false
    }
    
    ssIterator.currentKey = ssIterator.dbIterator.Key()
    ssIterator.currentValue = row.Siblings
    ssIterator.currentLocalVersion = row.LocalVersion
    
    return true
}

func (ssIterator *BasicSiblingSetIterator) Prefix() []byte {
    return decodePartitionDataKey(ssIterator.dbIterator.Prefix())
}

func (ssIterator *BasicSiblingSetIterator) Key() []byte {
    if ssIterator.currentKey == nil {
        return nil
    }
    
    return decodePartitionDataKey(ssIterator.currentKey)
}

func (ssIterator *BasicSiblingSetIterator) Value() *SiblingSet {
    return ssIterator.currentValue
}

func (ssIterator *BasicSiblingSetIterator) LocalVersion() uint64 {
    return ssIterator.currentLocalVersion
}

func (ssIterator *BasicSiblingSetIterator) Release() {
    ssIterator.dbIterator.Release()
}

func (ssIterator *BasicSiblingSetIterator) Error() error {
    if ssIterator.parseError != nil {
        return EStorage
    }
        
    if ssIterator.dbIterator.Error() != nil {
        return EStorage
    }
    
    return nil
}
