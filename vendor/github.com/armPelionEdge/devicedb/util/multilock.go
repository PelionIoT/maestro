package util

import (
    "sync"
)

type MultiLock struct {
    mapLock sync.Mutex
    conditionMap map[string]*sync.RWMutex
    countMap map[string]uint64
}

func NewMultiLock() *MultiLock {
    var ml MultiLock
    
    ml.conditionMap = make(map[string]*sync.RWMutex)
    ml.countMap = make(map[string]uint64)
    
    return &ml
}

func (multiLock *MultiLock) Lock(partitioningKey []byte) {
    multiLock.mapLock.Lock()
    
    lock, ok := multiLock.conditionMap[string(partitioningKey)]
    
    if !ok {
        lock = new(sync.RWMutex)
        
        multiLock.conditionMap[string(partitioningKey)] = lock
        multiLock.countMap[string(partitioningKey)] = 0
    }
    
    multiLock.countMap[string(partitioningKey)] += 1
    
    multiLock.mapLock.Unlock()
    
    lock.Lock()
}

func (multiLock *MultiLock) Unlock(partitioningKey []byte) {
    multiLock.mapLock.Lock()

    lock, ok := multiLock.conditionMap[string(partitioningKey)]
    
    if !ok {
        multiLock.mapLock.Unlock()
        
        return
    }

    multiLock.countMap[string(partitioningKey)] -= 1
    
    if multiLock.countMap[string(partitioningKey)] == 0 {
        delete(multiLock.conditionMap, string(partitioningKey))
        delete(multiLock.countMap, string(partitioningKey))
    }
    
    multiLock.mapLock.Unlock()
    
    lock.Unlock()
}
