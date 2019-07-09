package bucket

import (
    "container/heap"
    "context"   
    "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/logging"
    "strings"
    "sync"
)

type UpdateHeap []*data.Row

func (h UpdateHeap) Len() int {
    return len(h)
}

func (h UpdateHeap) Less(i, j int) bool {
    return h[i].LocalVersion < h[j].LocalVersion
}

func (h UpdateHeap) Swap(i, j int) {
    h[i], h[j] = h[j], h[i]
}

func (h *UpdateHeap) Push(x interface{}) {
    *h = append(*h, x.(*data.Row))
}

func (h *UpdateHeap) Pop() interface{} {
    old := *h
    n := len(old)
    x := old[n - 1]
    *h = old[0 : n - 1]

    return x
}

type listener struct {
    keys [][]byte
    prefixes [][]byte
    ch chan data.Row
}

func (l *listener) isListeningFor(update data.Row) bool {
    for _, key := range l.keys {
        if update.Key == string(key) {
            return true
        }
    }

    for _, prefix := range l.prefixes {
        if strings.HasPrefix(update.Key, string(prefix)) {
            return true
        }
    }

    return false
}

type Monitor struct {
    listeners map[*listener]bool
    mu sync.Mutex
    previousVersion uint64
    updateHeap *UpdateHeap
}

func NewMonitor(startVersion uint64) *Monitor {
    updateHeap := &UpdateHeap{ }
    heap.Init(updateHeap)

    return &Monitor{
        listeners: make(map[*listener]bool),
        previousVersion: startVersion,
        updateHeap: updateHeap,
    }
}

func (monitor *Monitor) AddListener(ctx context.Context, keys [][]byte, prefixes [][]byte, ch chan data.Row) {
    monitor.mu.Lock()
    defer monitor.mu.Unlock()

    Log.Infof("Add listener for keys %v and prefixes %v", keys, prefixes)
    
    var newListener listener

    newListener.keys = keys
    newListener.prefixes = prefixes
    newListener.ch = ch
    monitor.listeners[&newListener] = true

    go func() {
        <-ctx.Done()
        monitor.mu.Lock()
        defer monitor.mu.Unlock()
        Log.Infof("Remove listener for keys %v and prefixes %v", keys, prefixes)
        delete(monitor.listeners, &newListener)        
        close(ch)
    }()
}

func (monitor *Monitor) Notify(update data.Row) {
    monitor.mu.Lock()
    defer monitor.mu.Unlock()

    monitor.submitUpdate(update)
}

// This should be called if an ID range was reserved for a series of updates but those
// updates failed to be submitted so the range needs to be discarded
func (monitor *Monitor) DiscardIDRange(low uint64, high uint64) {
    monitor.mu.Lock()
    defer monitor.mu.Unlock()

    for i := low; i <= high; i++ {
        monitor.submitUpdate(data.Row{ Key: "", LocalVersion: i, Siblings: nil })
    }
}

func (monitor *Monitor) submitUpdate(update data.Row) {
    if update.LocalVersion < monitor.previousVersion || update.LocalVersion == monitor.previousVersion && update.LocalVersion != 0 {
        Log.Criticalf("An update was submitted to the monitor with key %s and version %d but the lowest expected version is %d. This update will not be sent. This should not happen and represents a bug in the watcher system.", update.Key, update.LocalVersion, monitor.previousVersion)
        
        return
    }

    heap.Push(monitor.updateHeap, &update)

    monitor.flushUpdates()
}

func (monitor *Monitor) flushUpdates() {
    h := *monitor.updateHeap

    // The reason this has to check if the localversion is 0 is in case it is the first update ever submitted.
    for monitor.updateHeap.Len() > 0 && (h[0].LocalVersion == 0 || h[0].LocalVersion == monitor.previousVersion + 1) {
        monitor.previousVersion = h[0].LocalVersion
        h = *monitor.updateHeap
        nextUpdate := heap.Pop(monitor.updateHeap).(*data.Row)

        if nextUpdate.Key == "" {
            // indicates a discarded update index. should just skip this
            continue
        }

        monitor.sendUpdate(*nextUpdate)
    }
}

func (monitor *Monitor) sendUpdate(update data.Row) {
    if len(monitor.listeners) > 0 {
        Log.Debugf("Monitor notifying listeners of update %d to key %s", update.LocalVersion, update.Key)
    }
    
    for l, _ := range monitor.listeners {
        if l.isListeningFor(update) {
            l.ch <- update
        }
    }
}