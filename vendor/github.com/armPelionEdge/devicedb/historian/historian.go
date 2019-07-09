package historian

import (
    "encoding/json"
    "encoding/binary"
    "encoding/base64"
    "crypto/rand"
    "fmt"
    "math"
    "sort"
    "sync"
    
    . "github.com/armPelionEdge/devicedb/storage"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/logging"
)

var (
    BY_TIME_PREFIX = []byte{ 0 }
    BY_SOURCE_AND_TIME_PREFIX = []byte{ 1 }
    BY_DATA_SOURCE_AND_TIME_PREFIX = []byte{ 2 }
    BY_SERIAL_NUMBER_PREFIX = []byte{ 3 }
    SEQUENTIAL_COUNTER_PREFIX = []byte{ 4 }
    CURRENT_SIZE_COUNTER_PREFIX = []byte{ 5 }
    HIGHEST_FORWARDED_INDEX_PREFIX = []byte{ 6 }
    DELIMETER = []byte(".")
)

func randomString() string {
    randomBytes := make([]byte, 16)
    rand.Read(randomBytes)
    
    high := binary.BigEndian.Uint64(randomBytes[:8])
    low := binary.BigEndian.Uint64(randomBytes[8:])
    
    return fmt.Sprintf("%05x%05x", high, low)
}

func timestampBytes(ts uint64) []byte {
    bytes := make([]byte, 8)
    
    binary.BigEndian.PutUint64(bytes, ts)
    
    return bytes
}

type HistoryQuery struct {
    MinSerial *uint64
    MaxSerial *uint64
    Sources []string
    Data *string
    Order string
    Before uint64
    After uint64
    Limit int
}

type Event struct {
    Timestamp uint64 `json:"timestamp"`
    SourceID string `json:"source"`
    Type string `json:"type"`
    Data string `json:"data"`
    UUID string `json:"uuid"`
    Serial uint64 `json:"serial"`
    Groups []string `json:"groups"`
}

func (event *Event) indexBySerial() []byte {
    sEncoding := timestampBytes(event.Serial)
    result := make([]byte, 0, len(BY_SERIAL_NUMBER_PREFIX) + len(sEncoding))
    
    result = append(result, BY_SERIAL_NUMBER_PREFIX...)
    result = append(result, sEncoding...)
    
    return result
}

func (event *Event) prefixBySerial() []byte {
    return event.indexBySerial()
}

func (event *Event) indexByTime() []byte {
    timestampEncoding := timestampBytes(event.Timestamp)
    uuid := []byte(event.UUID)
    result := make([]byte, 0, len(BY_TIME_PREFIX) + len(timestampEncoding) + len(DELIMETER) + len(uuid))
    
    result = append(result, BY_TIME_PREFIX...)
    result = append(result, timestampEncoding...)
    result = append(result, DELIMETER...)
    result = append(result, uuid...)
    
    return result
}

func (event *Event) prefixByTime() []byte {
    timestampEncoding := timestampBytes(event.Timestamp)
    result := make([]byte, 0, len(BY_TIME_PREFIX) + len(timestampEncoding) + len(DELIMETER))
    
    result = append(result, BY_TIME_PREFIX...)
    result = append(result, timestampEncoding...)
    result = append(result, DELIMETER...)
    
    return result
}

func (event *Event) indexBySourceAndTime() []byte {
    sourceEncoding := []byte(base64.StdEncoding.EncodeToString([]byte(event.SourceID)))
    timestampEncoding := timestampBytes(event.Timestamp)
    uuid := []byte(event.UUID)
    result := make([]byte, 0, len(BY_SOURCE_AND_TIME_PREFIX) + len(sourceEncoding) + len(DELIMETER) + len(timestampEncoding) + len(DELIMETER) + len(uuid))
    
    result = append(result, BY_SOURCE_AND_TIME_PREFIX...)
    result = append(result, sourceEncoding...)
    result = append(result, DELIMETER...)
    result = append(result, timestampEncoding...)
    result = append(result, DELIMETER...)
    result = append(result, uuid...)
    
    return result
}

func (event *Event) prefixBySourceAndTime() []byte {
    sourceEncoding := []byte(base64.StdEncoding.EncodeToString([]byte(event.SourceID)))
    timestampEncoding := timestampBytes(event.Timestamp)
    result := make([]byte, 0, len(BY_SOURCE_AND_TIME_PREFIX) + len(sourceEncoding) + len(DELIMETER) + len(timestampEncoding) + len(DELIMETER))
    
    result = append(result, BY_SOURCE_AND_TIME_PREFIX...)
    result = append(result, sourceEncoding...)
    result = append(result, DELIMETER...)
    result = append(result, timestampEncoding...)
    result = append(result, DELIMETER...)
    
    return result
}

func (event *Event) indexByDataSourceAndTime() []byte {
    sourceEncoding := []byte(base64.StdEncoding.EncodeToString([]byte(event.SourceID)))
    dataEncoding := []byte(base64.StdEncoding.EncodeToString([]byte(event.Data)))
    timestampEncoding := timestampBytes(event.Timestamp)
    uuid := []byte(event.UUID)
    result := make([]byte, 0, len(BY_SOURCE_AND_TIME_PREFIX) + len(dataEncoding) + len(DELIMETER) + len(sourceEncoding) + len(DELIMETER) + len(timestampEncoding) + len(DELIMETER) + len(uuid))
    
    result = append(result, BY_DATA_SOURCE_AND_TIME_PREFIX...)
    result = append(result, dataEncoding...)
    result = append(result, DELIMETER...)
    result = append(result, sourceEncoding...)
    result = append(result, DELIMETER...)
    result = append(result, timestampEncoding...)
    result = append(result, DELIMETER...)
    result = append(result, uuid...)
    
    return result
}

func (event *Event) prefixByDataSourceAndTime() []byte {
    sourceEncoding := []byte(base64.StdEncoding.EncodeToString([]byte(event.SourceID)))
    dataEncoding := []byte(base64.StdEncoding.EncodeToString([]byte(event.Data)))
    timestampEncoding := timestampBytes(event.Timestamp)
    result := make([]byte, 0, len(BY_SOURCE_AND_TIME_PREFIX) + len(dataEncoding) + len(DELIMETER) + len(sourceEncoding) + len(DELIMETER) + len(timestampEncoding) + len(DELIMETER))
    
    result = append(result, BY_DATA_SOURCE_AND_TIME_PREFIX...)
    result = append(result, dataEncoding...)
    result = append(result, DELIMETER...)
    result = append(result, sourceEncoding...)
    result = append(result, DELIMETER...)
    result = append(result, timestampEncoding...)
    result = append(result, DELIMETER...)
    
    return result
}
 
type Historian struct {
    storageDriver StorageDriver
    nextID uint64
    currentSize uint64
    forwardIndex uint64
    logLock sync.Mutex
    // event limit describes the maximum number of events
    // allowed to be stored in the history log
    // before old events must be purged.
    eventLimit uint64
    // event floor describes the minimum number of events
    // that must be remembered after log rotation occurs
    eventFloor uint64
    // when events are purged from the history log this
    // value describes how many events are deleted
    // per batch. In other words, if 100 events are
    // being purged and the purge batch size is 20
    // then 5 batches  will be applied to the data store.
    purgeBatchSize int
}

func NewHistorian(storageDriver StorageDriver, eventLimit uint64, eventFloor uint64, purgeBatchSize int) *Historian {
    var nextID uint64
    var currentSize uint64
    var forwardIndex uint64
    
    values, err := storageDriver.Get([][]byte{ SEQUENTIAL_COUNTER_PREFIX, CURRENT_SIZE_COUNTER_PREFIX, HIGHEST_FORWARDED_INDEX_PREFIX })
    
    if err == nil && len(values[0]) == 8 {
        nextID = binary.BigEndian.Uint64(values[0])
    }
    
    if err == nil && len(values[1]) == 8 {
        currentSize = binary.BigEndian.Uint64(values[1])
    }

    if err == nil && len(values[2]) == 8 {
        forwardIndex = binary.BigEndian.Uint64(values[2])
    }
    
    historian := &Historian{
        storageDriver: storageDriver,
        nextID: nextID + 1,
        forwardIndex: forwardIndex,
        currentSize: currentSize,
        eventLimit: eventLimit,
        eventFloor: eventFloor,
        purgeBatchSize: purgeBatchSize,
    }
    
    historian.RotateLog()
    
    return historian
}

func (historian *Historian) LogSize() uint64 {
    return historian.currentSize
}

func (historian *Historian) LogSerial() uint64 {
    return historian.nextID
}

func (historian *Historian) SetLogSerial(s uint64) error {
    historian.logLock.Lock()
    defer historian.logLock.Unlock()
    
    if s < historian.nextID - 1 {
        return nil
    }
    
    batch := NewBatch()
    batch.Put(SEQUENTIAL_COUNTER_PREFIX, timestampBytes(s))
    
    err := historian.storageDriver.Batch(batch)
    
    if err != nil {
        Log.Errorf("Storage driver error in SetLogSerial(%v): %s", s, err.Error())
        
        return EStorage
    }
    
    historian.nextID = s + 1
    
    return nil
}

func (historian *Historian) ForwardIndex() uint64 {
    return historian.forwardIndex
}

func (historian *Historian) SetForwardIndex(i uint64) error {
    historian.logLock.Lock()
    defer historian.logLock.Unlock()

    batch := NewBatch()
    batch.Put(HIGHEST_FORWARDED_INDEX_PREFIX, timestampBytes(i))
    
    err := historian.storageDriver.Batch(batch)
    
    if err != nil {
        Log.Errorf("Storage driver error in SetForwardIndex(%v): %s", i, err.Error())
        
        return EStorage
    }

    historian.forwardIndex = i

    return nil
}

func (historian *Historian) LogEvent(event *Event) error {
    // events must be logged sequentially to preserve the invariant that when a batch is written to
    // disk no other batch that has been written before it has a serial number greater than itself
    historian.logLock.Lock()
    defer historian.logLock.Unlock()
    
    // indexed by time
    // indexed by resourceid + time
    // indexed by eventdata + resourceID + time
    event.UUID = randomString()
    event.Serial = historian.nextID
    
    batch := NewBatch()
    marshaledEvent, err := json.Marshal(event)
    
    if err != nil {
        Log.Errorf("Could not marshal event to JSON: %v", err.Error())
        
        return EStorage
    }
    
    batch.Put(event.indexByTime(), []byte(marshaledEvent))
    batch.Put(event.indexBySourceAndTime(), []byte(marshaledEvent))
    batch.Put(event.indexByDataSourceAndTime(), []byte(marshaledEvent))
    batch.Put(event.indexBySerial(), []byte(marshaledEvent))
    batch.Put(SEQUENTIAL_COUNTER_PREFIX, timestampBytes(event.Serial))
    batch.Put(CURRENT_SIZE_COUNTER_PREFIX, timestampBytes(historian.currentSize + 1))
    
    err = historian.storageDriver.Batch(batch)
    
    if err != nil {
        Log.Errorf("Storage driver error in LogEvent(%v): %s", event, err.Error())
        
        return EStorage
    }
    
    historian.nextID += 1
    historian.currentSize += 1
    
    err = historian.RotateLog()
    
    if err != nil {
        return EStorage
    }
    
    return nil
}

func (historian *Historian) Query(query *HistoryQuery) (*EventIterator, error) {
    var ranges [][2][]byte
    var direction int
    
    if query.Order == "desc" {
        direction = BACKWARD
    } else {
        direction = FORWARD
    }
    
    // query.Before is unspecified so default to max range
    if query.Before == 0 {
        query.Before = math.MaxUint64
    }
    
    // ensure consistent ordering from multiple sources
    sort.Strings(query.Sources)

    if query.MinSerial != nil {
        ranges = make([][2][]byte, 1)
        
        ranges[0] = [2][]byte{
            (&Event{ Serial: *query.MinSerial }).prefixBySerial(),
            (&Event{ Serial: math.MaxUint64 }).prefixBySerial(),
        }
    } else if query.MaxSerial != nil {
        ranges = make([][2][]byte, 1)
        
        ranges[0] = [2][]byte{
            (&Event{ Serial: 0 }).prefixBySerial(),
            (&Event{ Serial: *query.MaxSerial }).prefixBySerial(),
        }
    } else if len(query.Sources) == 0 {
        // time -> indexByTime
        ranges = make([][2][]byte, 1)
        
        ranges[0] = [2][]byte{
            (&Event{ Timestamp: query.After }).prefixByTime(),
            (&Event{ Timestamp: query.Before }).prefixByTime(),
        }
    } else if query.Data == nil {
        // sources + time -> indexBySourceAndTime
        ranges = make([][2][]byte, 0, len(query.Sources))
        
        for _, source := range query.Sources {
            ranges = append(ranges, [2][]byte{
                (&Event{ SourceID: source, Timestamp: query.After }).prefixBySourceAndTime(),
                (&Event{ SourceID: source, Timestamp: query.Before }).prefixBySourceAndTime(),
            })
        }
    } else {
        // data + sources + time -> indexByDataSourceAndTime
        ranges = make([][2][]byte, 0, len(query.Sources))
        
        for _, source := range query.Sources {
            ranges = append(ranges, [2][]byte{
                (&Event{ Data: *query.Data, SourceID: source, Timestamp: query.After }).prefixByDataSourceAndTime(),
                (&Event{ Data: *query.Data, SourceID: source, Timestamp: query.Before }).prefixByDataSourceAndTime(),
            })
        }
    }
    
    iter, err := historian.storageDriver.GetRanges(ranges, direction)
    
    if err != nil {
        Log.Errorf("Storage driver error in Query(%v): %s", query, err.Error())
        
        return nil, err
    }
    
    return NewEventIterator(iter, query.Limit), nil
}

func (historian *Historian) Purge(query *HistoryQuery) error {
    historian.logLock.Lock()
    defer historian.logLock.Unlock()
    
    return historian.purge(query)
}

func (historian *Historian) purge(query *HistoryQuery) error {
    eventIterator, err := historian.Query(query)
    
    if err != nil {
        return err
    }

    var eventBatch []*Event

    if historian.purgeBatchSize <= 0 {
        eventBatch = make([]*Event, 1)
    } else {
        eventBatch = make([]*Event, historian.purgeBatchSize)
    }

    var currentBatchSize int = 0

    for eventIterator.Next() {
        eventBatch[currentBatchSize] = eventIterator.Event()
        currentBatchSize++

        if currentBatchSize < len(eventBatch) {
            continue
        }

        // Precondition at this point: currentBatchSize == len(eventBatch)
        err := historian.purgeEvents(eventBatch)
        
        if err != nil {
            return err
        }

        // This needs to be reset since the next events belong to a new batch
        currentBatchSize = 0
    }
    
    if eventIterator.Error() != nil {
        Log.Errorf("Storage driver error in Purge(%v): %s", query, err.Error())
        
        return eventIterator.Error()
    }

    // This needs to be called after the main loop exits since
    // there might have been a partial batch at the end of the
    // list of events that were iterated through. Since in the 
    // loop above the batch only gets written once it is full
    // there can be some leftover.
    err = historian.purgeEvents(eventBatch[:currentBatchSize])

    if err != nil {
        return err
    }
    
    return nil
}

func (historian *Historian) purgeEvents(events []*Event) error {
    batch := NewBatch()
   
    for _, event := range events {
        batch.Delete(event.indexByTime())
        batch.Delete(event.indexBySourceAndTime())
        batch.Delete(event.indexByDataSourceAndTime())
        batch.Delete(event.indexBySerial())
    }

    batch.Put(CURRENT_SIZE_COUNTER_PREFIX, timestampBytes(historian.currentSize - uint64(len(events))))
    
    err := historian.storageDriver.Batch(batch)
    
    if err != nil {
        Log.Errorf("Storage driver error in purgeEvents(): %s", err.Error())
        
        return EStorage
    }
    
    historian.currentSize -= uint64(len(events))
    
    return nil
}

// The behavior of log rotation is determined by the
// relationship between eventLimit, eventFloor, and
// purgeBatchSize. Log rotation occurs when the log
// contains eventLimit events. Log rotation will
// delete old events until there are eventFloor
// events remaining in the log. Configuring these
// two values can adjust the performance characteristics
// of the history log and how aggressively disk space is
// conserved. 
// 
// Here are some example configurations and
// their performance characteristics:
// 
// eventLimit = 100000 eventFloor = 99999
//   In this case the most events ever stored
//   on disk will be 100000. However, after
//   there are at least 99999 events written to
//   the log every new log entry will require
//   a log rotation operation to occur in order
//   to keep the log size under the limit.
// eventLimit = 200000 eventFloor = 100000
//   In this case a purge operation only occurs
//   once every 100000 writes instead of on every
//   write.
//
// if eventFloor is greater than or equal to
// eventLimit then eventFloor is ignored and
// eventLimit is used as the event floor
func (historian *Historian) RotateLog() error {
    if historian.eventLimit != 0 && historian.currentSize > historian.eventLimit {
        var minSerial uint64 = 0
        var err error

        if historian.eventFloor < historian.eventLimit {
            Log.Debugf("Purging oldest %d items from history log (currentSize=%d eventFloor=%d)", int(historian.currentSize - historian.eventFloor), historian.currentSize, historian.eventFloor)
            err = historian.purge(&HistoryQuery{ MinSerial: &minSerial, Limit: int(historian.currentSize - historian.eventFloor) })
        } else {
            Log.Debugf("Purging oldest %d items from history log (currentSize=%d eventLimit=%d)", int(historian.currentSize - historian.eventLimit), historian.currentSize, historian.eventFloor)
            err = historian.purge(&HistoryQuery{ MinSerial: &minSerial, Limit: int(historian.currentSize - historian.eventLimit) })
        }
        
        if err != nil {
            return err
        }
    }
    
    return nil
}

type EventIterator struct {
    dbIterator StorageIterator
    parseError error
    currentEvent *Event
    limit uint64
    eventsSeen uint64
}

func NewEventIterator(iterator StorageIterator, limit int) *EventIterator {
    if limit <= 0 {
        limit = 0
    }
    
    return &EventIterator{
        dbIterator: iterator,
        parseError: nil,
        currentEvent: nil,
        limit: uint64(limit),
        eventsSeen: 0,
    }
}

func (ei *EventIterator) Next() bool {
    ei.currentEvent = nil
    
    if !ei.dbIterator.Next() {
        if ei.dbIterator.Error() != nil {
            Log.Errorf("Storage driver error in Next(): %s", ei.dbIterator.Error())
        }
        
        return false
    }

    var event Event
    
    ei.parseError = json.Unmarshal(ei.dbIterator.Value(), &event)
    
    if ei.parseError != nil {
        Log.Errorf("Storage driver error in Next() key = %v, value = %v: %s", ei.dbIterator.Key(), ei.dbIterator.Value(), ei.parseError.Error())
        
        ei.Release()
        
        return false
    }
    
    ei.currentEvent = &event
    ei.eventsSeen += 1
    
    if ei.limit != 0 && ei.eventsSeen == ei.limit {
        ei.Release()
    }
    
    return true
}

func (ei *EventIterator) Event() *Event {
    return ei.currentEvent
}

func (ei *EventIterator) Release() {
    ei.dbIterator.Release()
}

func (ei *EventIterator) Error() error {
    if ei.parseError != nil {
        return EStorage
    }
        
    if ei.dbIterator.Error() != nil {
        return EStorage
    }
    
    return nil
}
