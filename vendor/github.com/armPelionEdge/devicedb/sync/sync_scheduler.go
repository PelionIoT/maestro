package sync

import (
    "container/heap"
    "sync"
    "time"
)

type Peer struct {
    id string
    buckets []string
    nextBucket int
    nextSyncTime time.Time
}

func NewPeer(id string, buckets []string) *Peer {
    return &Peer{
        id: id,
        buckets: buckets,
        nextBucket: 0,
    }
}

func (peer *Peer) NextBucket() string {
    if len(peer.buckets) == 0 {
        return ""
    }

    return peer.buckets[peer.nextBucket]
}

func (peer *Peer) Advance() {
    if len(peer.buckets) == 0 {
        peer.nextBucket = 0
    }

    peer.nextBucket = (peer.nextBucket + 1) % len(peer.buckets)
}

type PeerHeap []*Peer

func (h PeerHeap) Len() int {
    return len(h)
}

func (h PeerHeap) Less(i, j int) bool {
    return h[i].nextSyncTime.Before(h[j].nextSyncTime)
}

func (h PeerHeap) Swap(i, j int) {
    h[i], h[j] = h[j], h[i]
}

func (h *PeerHeap) Push(x interface{}) {
    *h = append(*h, x.(*Peer))
}

func (h *PeerHeap) Pop() interface{} {
    old := *h
    n := len(old)
    x := old[n - 1]
    *h = old[0 : n - 1]

    return x
}

type SyncScheduler interface {
    AddPeer(peerID string, buckets []string)
    RemovePeer(peerID string)
    Next() (string, string)
    Advance()
    Schedule(peerID string)
}

// Sync queue optimized for relays
// that provides a new sync partner
// at a fixed rate
type PeriodicSyncScheduler struct {
    syncPeriod time.Duration
    peers map[string]*Peer
    queue []*Peer
    mu sync.Mutex
}

func NewPeriodicSyncScheduler(syncPeriod time.Duration) *PeriodicSyncScheduler {
    return &PeriodicSyncScheduler{
        syncPeriod: syncPeriod,
        peers: make(map[string]*Peer),
        queue: make([]*Peer, 0),
    }
}

func (syncScheduler *PeriodicSyncScheduler) AddPeer(peerID string, buckets []string) {
    syncScheduler.mu.Lock()
    defer syncScheduler.mu.Unlock()

    if _, ok := syncScheduler.peers[peerID]; ok {
        return
    }

    syncScheduler.peers[peerID] = NewPeer(peerID, buckets)
}

func (syncScheduler *PeriodicSyncScheduler) RemovePeer(peerID string) {
    syncScheduler.mu.Lock()
    defer syncScheduler.mu.Unlock()

    newQueue := make([]*Peer, 0, len(syncScheduler.queue))

    for _, peer := range syncScheduler.queue {
        if peer.id == peerID {
            continue
        }

        newQueue = append(newQueue, peer)
    }

    syncScheduler.queue = newQueue
    delete(syncScheduler.peers, peerID)
}

func (syncScheduler *PeriodicSyncScheduler) Next() (string, string) {
    <-time.After(syncScheduler.syncPeriod)

    syncScheduler.mu.Lock()
    defer syncScheduler.mu.Unlock()

    if len(syncScheduler.queue) == 0 {
        return "", ""
    }

    return syncScheduler.queue[0].id, syncScheduler.queue[0].NextBucket()
}

func (syncScheduler *PeriodicSyncScheduler) Advance() {
    syncScheduler.mu.Lock()
    defer syncScheduler.mu.Unlock()

    if len(syncScheduler.queue) == 0 {
        return
    }

    peer := syncScheduler.queue[0]
    syncScheduler.queue = syncScheduler.queue[1:]
    peer.Advance()
}

func (syncScheduler *PeriodicSyncScheduler) Schedule(peerID string) {
    syncScheduler.mu.Lock()
    defer syncScheduler.mu.Unlock()

    if _, ok := syncScheduler.peers[peerID]; !ok {
        return
    }

    syncScheduler.queue = append(syncScheduler.queue, syncScheduler.peers[peerID])
}

// Optimized for cloud servers that need
// to coordinate sync sessions with hundreds
// or thousands of relays at once. Tries to ensure
// that for any particular peer, sync sessions with
// that peer are scheduled periodically attempting
// to minimize jitter between the scheduled time
// and the actual time.
type MultiSyncScheduler struct {
    syncPeriod time.Duration
    peers map[string]*Peer
    heap *PeerHeap
    mu sync.Mutex
    lastPeer *Peer
}

func NewMultiSyncScheduler(syncPeriod time.Duration) *MultiSyncScheduler {
    peerHeap := &PeerHeap{ }
    heap.Init(peerHeap)

    return &MultiSyncScheduler{
        syncPeriod: syncPeriod,
        peers: make(map[string]*Peer),
        heap: peerHeap,
    }
}

func (syncScheduler *MultiSyncScheduler) AddPeer(peerID string, buckets []string) {
    syncScheduler.mu.Lock()
    defer syncScheduler.mu.Unlock()

    if _, ok := syncScheduler.peers[peerID]; ok {
        return
    }

    syncScheduler.peers[peerID] = NewPeer(peerID, buckets)
}

func (syncScheduler *MultiSyncScheduler) RemovePeer(peerID string) {
    syncScheduler.mu.Lock()
    defer syncScheduler.mu.Unlock()

    for i := 0; i < syncScheduler.heap.Len(); i++ {
        h := *syncScheduler.heap
        peer := h[i]

        if peer.id == peerID {
            heap.Remove(syncScheduler.heap, i)
        }
    }

    delete(syncScheduler.peers, peerID)
    syncScheduler.lastPeer = nil
}

func (syncScheduler *MultiSyncScheduler) Next() (string, string) {
    syncScheduler.mu.Lock()

    if syncScheduler.heap.Len() == 0 {
        syncScheduler.mu.Unlock()

        // Wait for the default timeout
        <-time.After(syncScheduler.syncPeriod)

        return "", ""
    }


    h := *syncScheduler.heap
    peer := h[0]
    now := time.Now()
    syncTime := peer.nextSyncTime

    // Was Next() called again before calling Advance()?
    if syncScheduler.lastPeer == peer {
        syncScheduler.mu.Unlock()

        <-time.After(syncScheduler.syncPeriod)

        return peer.id, peer.NextBucket()
    }

    syncScheduler.lastPeer = peer

    // unlock the mutex. It is needed only
    // to synchronize access to the heap
    // and peers map
    syncScheduler.mu.Unlock()

    // If we need to wait a while before
    // returning, do so
    if now.Before(syncTime) {
        <-time.After(syncTime.Sub(now))
    }

    return peer.id, peer.NextBucket()
}

func (syncScheduler *MultiSyncScheduler) Advance() {
    syncScheduler.mu.Lock()
    defer syncScheduler.mu.Unlock()

    if syncScheduler.heap.Len() == 0 {
        return
    }

    h := *syncScheduler.heap
    h[0].Advance()
    heap.Pop(syncScheduler.heap)
    syncScheduler.lastPeer = nil
}

func (syncScheduler *MultiSyncScheduler) Schedule(peerID string) {
    syncScheduler.mu.Lock()
    defer syncScheduler.mu.Unlock()

    if _, ok := syncScheduler.peers[peerID]; !ok {
        return
    }

    // Schedule the next sync with this peer after syncPeriod duration
    syncScheduler.peers[peerID].nextSyncTime = time.Now().Add(syncScheduler.syncPeriod)
    heap.Push(syncScheduler.heap, syncScheduler.peers[peerID])
}