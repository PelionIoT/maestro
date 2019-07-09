package util

import (
    "sync"
)

type RWTryLock struct {
    mu sync.Mutex
    rwMu sync.RWMutex
    writeLocked bool
}

func (lock *RWTryLock) WLock() {
    lock.mu.Lock()
    lock.writeLocked = true
    lock.mu.Unlock()

    lock.rwMu.Lock()

    lock.mu.Lock()
    lock.writeLocked = true
    lock.mu.Unlock()
}

func (lock *RWTryLock) WUnlock() {
    lock.mu.Lock()
    defer lock.mu.Unlock()

    lock.writeLocked = false
    lock.rwMu.Unlock()
}

func (lock *RWTryLock) TryRLock() bool {
    lock.mu.Lock()
    defer lock.mu.Unlock()

    if lock.writeLocked {
        return false
    }

    lock.rwMu.RLock()

    return true
}

func (lock *RWTryLock) RUnlock() {
    lock.rwMu.RUnlock()
}