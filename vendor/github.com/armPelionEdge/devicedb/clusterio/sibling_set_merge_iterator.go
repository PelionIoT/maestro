package clusterio

import (
    "sort"

    . "github.com/armPelionEdge/devicedb/data"
)

type SiblingSetMergeIterator struct {
    readMerger NodeReadMerger
    keys [][]string
    prefixes []string
    keySet map[string]bool
    prefixIndexes map[string]int
    currentPrefixIndex int
    currentKeyIndex int
}

func NewSiblingSetMergeIterator(readMerger NodeReadMerger) *SiblingSetMergeIterator {
    return &SiblingSetMergeIterator{
        readMerger: readMerger,
        keys: make([][]string, 0),
        prefixes: make([]string, 0),
        keySet: make(map[string]bool),
        prefixIndexes: make(map[string]int),
        currentKeyIndex: -1,
        currentPrefixIndex: -1,
    }
}

func (iter *SiblingSetMergeIterator) AddKey(prefix string, key string) {
    if _, ok := iter.keySet[key]; ok {
        // Ignore this request. This key was already added before
        return
    }

    iter.keySet[key] = true

    if _, ok := iter.prefixIndexes[prefix]; !ok {
        // this is a prefix that hasn't been seen before. insert a new key list for this prefix
        iter.prefixIndexes[prefix] = len(iter.keys)
        iter.keys = append(iter.keys, []string{ })
        iter.prefixes = append(iter.prefixes, prefix)
    }

    prefixIndex := iter.prefixIndexes[prefix]
    iter.keys[prefixIndex] = append(iter.keys[prefixIndex], key)
}

func (iter *SiblingSetMergeIterator) SortKeys() {
    for _, keys := range iter.keys {
        sort.Strings(keys)
    }
}

func (iter *SiblingSetMergeIterator) Next() bool {
    if iter.currentPrefixIndex < 0 {
        iter.currentPrefixIndex = 0
    }
    
    if !(iter.currentPrefixIndex < len(iter.keys)) {
        return false
    }

    iter.currentKeyIndex++

    if iter.currentKeyIndex >= len(iter.keys[iter.currentPrefixIndex]) {
        iter.currentPrefixIndex++
        iter.currentKeyIndex = 0
    }

    return iter.currentPrefixIndex < len(iter.keys) && iter.currentKeyIndex < len(iter.keys[iter.currentPrefixIndex])
}

func (iter *SiblingSetMergeIterator) Prefix() []byte {
    if iter.currentPrefixIndex < 0 || iter.currentPrefixIndex >= len(iter.keys) || len(iter.keys) == 0 {
        return nil
    }

    return []byte(iter.prefixes[iter.currentPrefixIndex])
}

func (iter *SiblingSetMergeIterator) Key() []byte {
    if iter.currentPrefixIndex < 0 || iter.currentPrefixIndex >= len(iter.keys) || len(iter.keys) == 0 {
        return nil
    }

    if iter.currentKeyIndex >= len(iter.keys[iter.currentPrefixIndex]) {
        return nil
    }

    return []byte(iter.keys[iter.currentPrefixIndex][iter.currentKeyIndex])
}

func (iter *SiblingSetMergeIterator) Value() *SiblingSet {
    if iter.currentPrefixIndex < 0 || iter.currentPrefixIndex >= len(iter.keys) || len(iter.keys) == 0 {
        return nil
    }

    if iter.currentKeyIndex >= len(iter.keys[iter.currentPrefixIndex]) {
        return nil
    }

    return iter.readMerger.Get(iter.keys[iter.currentPrefixIndex][iter.currentKeyIndex])
}

func (iter *SiblingSetMergeIterator) LocalVersion() uint64 {
    return 0
}

func (iter *SiblingSetMergeIterator) Release() {
}

func (iter *SiblingSetMergeIterator) Error() error {
    return nil
}