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