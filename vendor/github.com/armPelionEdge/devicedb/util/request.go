package util

import (
    "sync"
)

type RequestMap struct {
    lock sync.Mutex
    requests map[uint64]chan interface{}
}

func NewRequestMap() *RequestMap {
    return &RequestMap{
        requests: make(map[uint64]chan interface{}),
    }
}

func (rm *RequestMap) MakeRequest(id uint64) <-chan interface{} {
    rm.lock.Lock()
    defer rm.lock.Unlock()

    if _, ok := rm.requests[id]; ok {
        return nil
    }

    requestChan := make(chan interface{}, 1)
    rm.requests[id] = requestChan

    return requestChan
}

func (rm *RequestMap) Respond(id uint64, r interface{}) {
    rm.lock.Lock()
    defer rm.lock.Unlock()

    if _, ok := rm.requests[id]; !ok {
        return
    }

    rm.requests[id] <- r
    close(rm.requests[id])
    delete(rm.requests, id)
}