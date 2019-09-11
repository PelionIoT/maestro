// This file provides utility functions that make backwards compatibility with original DeviceDB (written in NodeJS)
// easier
package compatibility

import (
    "encoding/json"
    "sort"
    "strings"
    "errors"
    "fmt"

    . "github.com/armPelionEdge/devicedb/shared"
    . "github.com/armPelionEdge/devicedb/storage"
    . "github.com/armPelionEdge/devicedb/logging"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/merkle"
    . "github.com/armPelionEdge/devicedb/bucket/builtin"
)

func SiblingToNormalizedJSON(sibling *Sibling) string {
    return valueToJSON(sibling.Value()) + dotToJSON(sibling.Clock().Dot()) + contextToJSON(sibling.Clock().Context())
}

func valueToJSON(value []byte) string {
    if value == nil {
        return "null"
    }
    
    j, _ := json.Marshal(string(value))
    
    return string(j)
}

func dotToJSON(dot Dot) string {
    nodeIDJSON, _ := json.Marshal(dot.NodeID)
    countJSON, _ := json.Marshal(dot.Count)
    
    return "[" + string(nodeIDJSON) + "," + string(countJSON) + "]"
}

func contextToJSON(context map[string]uint64) string {
    keys := make([]string, 0, len(context))
    
    for k, _ := range context {
        keys = append(keys, k)
    }
    
    sort.Strings(keys)
    
    j := "["
    
    for i := 0; i < len(keys); i += 1 {
        nodeIDJSON, _ := json.Marshal(keys[i])
        countJSON, _ := json.Marshal(context[keys[i]])
        
        j += "[[" + string(nodeIDJSON) + "," + string(countJSON) + "],null]"
        
        if i != len(keys) - 1 {
            j += ","
        }
    }
    
    j += "]"
    
    return j
}

func HashSiblingSet(key string, siblingSet *SiblingSet) Hash {
    siblingsJSON := make([]string, 0, siblingSet.Size())
    
    for sibling := range siblingSet.Iter() {
        siblingsJSON = append(siblingsJSON, SiblingToNormalizedJSON(sibling))
    }
    
    sort.Strings(siblingsJSON)
    
    j, _ := json.Marshal(siblingsJSON)
    
    return NewHash([]byte(key + string(j)))
}

func UpgradeLegacyDatabase(legacyDatabasePath string, serverConfig YAMLServerConfig) error {
    bucketDataPrefix := "cache."
    bucketNameMapping := map[string]string {
        "shared": "default",
        "lww": "lww",
        "cloud": "cloud",
        "local": "local",
    }
    
    legacyDatabaseDriver := NewLevelDBStorageDriver(legacyDatabasePath, nil)
    newDBStorageDriver := NewLevelDBStorageDriver(serverConfig.DBFile, nil)
    bucketList := NewBucketList()
    
    err := newDBStorageDriver.Open()
    
    if err != nil {
        return err
    }
    
    defer newDBStorageDriver.Close()
    
    err = legacyDatabaseDriver.Open()
    
    if err != nil {
        return err
    }
    
    defer legacyDatabaseDriver.Close()
    
    defaultBucket, _ := NewDefaultBucket("", NewPrefixedStorageDriver([]byte{ 0 }, newDBStorageDriver), serverConfig.MerkleDepth)
    cloudBucket, _ := NewCloudBucket("", NewPrefixedStorageDriver([]byte{ 1 }, newDBStorageDriver), serverConfig.MerkleDepth, RelayMode)
    lwwBucket, _ := NewLWWBucket("", NewPrefixedStorageDriver([]byte{ 2 }, newDBStorageDriver), serverConfig.MerkleDepth)
    localBucket, _ := NewLocalBucket("", NewPrefixedStorageDriver([]byte{ 3 }, newDBStorageDriver), MerkleMinDepth)
    
    bucketList.AddBucket(defaultBucket)
    bucketList.AddBucket(cloudBucket)
    bucketList.AddBucket(lwwBucket)
    bucketList.AddBucket(localBucket)
    
    iter, err := legacyDatabaseDriver.GetMatches([][]byte{ []byte(bucketDataPrefix) })
    
    if err != nil {
        return err
    }
    
    for iter.Next() {
        key := iter.Key()[len(iter.Prefix()):]
        value := iter.Value()
        keyParts := strings.Split(string(key), ".")
        
        if len(keyParts) < 2 {
            iter.Release()
            
            return errors.New(fmt.Sprintf("Key was invalid: %s", key))
        }
        
        legacyBucketName := keyParts[0]
        newBucketName, ok := bucketNameMapping[legacyBucketName]
        
        if !ok {
            Log.Warningf("Cannot translate object at %s because %s is not a recognized bucket name", key, legacyBucketName)
            
            continue
        }
        
        if !bucketList.HasBucket(newBucketName) {
            Log.Warningf("Cannot translate object at %s because %s is not a recognized bucket name", key, newBucketName)
            
            continue
        }
        
        bucket := bucketList.Get(newBucketName)
        siblingSet, err := DecodeLegacySiblingSet(value, legacyBucketName == "lww")
        
        if err != nil {
            Log.Warningf("Unable to decode object at %s (%s): %v", key, string(value), err)
            
            continue
        }
    
        nonPrefixedKey := string(key)[len(legacyBucketName) + 1:]
        err = bucket.Merge(map[string]*SiblingSet{
            nonPrefixedKey: siblingSet,
        })
        
        if err != nil {
            Log.Warningf("Unable to migrate object at %s (%s): %v", key, string(value), err)
        } else {
            //Log.Debugf("Migrated object in legacy bucket %s at key %s", legacyBucketName, nonPrefixedKey)
        }
    }
    
    if iter.Error() != nil {
        Log.Errorf("An error occurred while scanning through the legacy database: %v")
    }
    
    iter.Release()
    
    return nil
}

func DecodeLegacySiblingSet(data []byte, lww bool) (*SiblingSet, error) {
    var lss legacySiblingSet
    
    err := json.Unmarshal(data, &lss)
    
    if err != nil {
        return nil, err
    }
    
    return lss.ToSiblingSet(lww), nil
}

type legacySiblingSet []legacySibling

func (lss *legacySiblingSet) ToSiblingSet(lww bool) *SiblingSet {
    var siblings map[*Sibling]bool = make(map[*Sibling]bool, len(*lss))
    
    for _, ls := range *lss {
        siblings[ls.ToSibling(lww)] = true
    }
    
    return NewSiblingSet(siblings)
}

type legacySibling struct {
    Value *string `json:"value"`
    Clock legacyDVV `json:"clock"`
    CreationTime uint64 `json:"creationTime"`
}

type legacyLWWValue struct {
    Value *string `json:"value"`
    Timestamp *uint64 `json:"timestamp"`
}

func (ls *legacySibling) ToSibling(lww bool) *Sibling {
    var value []byte
    
    if ls.Value != nil {
        var lwwValue legacyLWWValue
        
        value = []byte(*ls.Value)

        if lww {
            err := json.Unmarshal(value, &lwwValue)
            
            if err == nil && lwwValue.Timestamp != nil {
                value = nil

                if lwwValue.Value != nil {
                    value = []byte(*lwwValue.Value)
                }
            }
        }
    }
    
    return NewSibling(ls.Clock.ToDVV(), value, ls.CreationTime)
}

type legacyDVV struct {
    Dot legacyDot `json:"dot"`
    Context []legacyDot `json:"context"`
}

func (ldvv *legacyDVV) ToDVV() *DVV {
    var context map[string]uint64 = make(map[string]uint64, len(ldvv.Context))
    
    for _, ld := range ldvv.Context {
        context[ld.node] = ld.count
    }
    
    return NewDVV(ldvv.Dot.ToDot(), context)
}

type legacyDot struct {
    node string
    count uint64
}

func (ld *legacyDot) ToDot() *Dot {
    return NewDot(ld.node, ld.count)
}

func (ld *legacyDot) MarshalJSON() ([]byte, error) {
    var a [2]interface{ }
    
    a[0] = ld.node
    a[1] = ld.count
    
    return json.Marshal(a)
}

func (ld *legacyDot) UnmarshalJSON(data []byte) error {
    var a [2]json.RawMessage
    
    err := json.Unmarshal(data, &a)
    
    if err != nil {
        return err
    }
    
    err = json.Unmarshal(a[0], &ld.node)
    
    if err != nil {
        return err
    }
    
    err = json.Unmarshal(a[1], &ld.count)
    
    if err != nil {
        return err
    }
    
    return nil
}
