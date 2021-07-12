package util
//
 // Copyright (c) 2019 ARM Limited.
 //
 // SPDX-License-Identifier: MIT
 //
 // Permission is hereby granted, free of charge, to any person obtaining a copy
 // of this software and associated documentation files (the "Software"), to
 // deal in the Software without restriction, including without limitation the
 // rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
 // sell copies of the Software, and to permit persons to whom the Software is
 // furnished to do so, subject to the following conditions:
 //
 // The above copyright notice and this permission notice shall be included in all
 // copies or substantial portions of the Software.
 //
 // THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 // IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 // FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 // AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 // LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 // OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 // SOFTWARE.
 //




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
