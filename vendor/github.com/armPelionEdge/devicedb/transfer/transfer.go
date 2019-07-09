package transfer

import (
    "bufio"
    "io"
    "errors"
    "encoding/json"
    "math"

    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/logging"
    . "github.com/armPelionEdge/devicedb/partition"
)

var ETransferCancelled = errors.New("Cancelled")
var EEntryChecksum = errors.New("Unable to reproduce the checksum for the entry in the partition chunk")

const (
    DefaultScanBufferSize = 100
)

func ChecksumEntries(entries []Entry) Hash {
    hash := Hash{ }

    for _, entry := range entries {
        hash = hash.Xor(entry.Value.Hash([]byte(entry.Key)))
    }

    return hash
}

type EntryFilter func(Entry) bool

type PartitionTransfer interface {
    NextChunk() (PartitionChunk, error)
    UseFilter(EntryFilter)
    Cancel()
}

type IncomingTransfer struct {
    scanner *bufio.Scanner
    err error
}

func NewIncomingTransfer(reader io.Reader) *IncomingTransfer {
    scanner := bufio.NewScanner(reader)
    scanner.Buffer(make([]byte, DefaultScanBufferSize), math.MaxInt32)

    return &IncomingTransfer{
        scanner: scanner,
    }
}

func (transfer *IncomingTransfer) UseFilter(entryFilter EntryFilter) {
}

func (transfer *IncomingTransfer) NextChunk() (PartitionChunk, error) {
    if transfer.err != nil {
        return PartitionChunk{}, transfer.err
    }

    if !transfer.scanner.Scan() {
        if transfer.scanner.Err() != nil {
            return PartitionChunk{}, transfer.scanner.Err()
        }

        return PartitionChunk{}, io.EOF
    }

    encoded := transfer.scanner.Bytes()
    var nextPartitionChunk PartitionChunk

    if err := json.Unmarshal(encoded, &nextPartitionChunk); err != nil {
        transfer.err = err

        return PartitionChunk{}, transfer.err
    }

    checksum := ChecksumEntries(nextPartitionChunk.Entries)

    if checksum.High() != nextPartitionChunk.Checksum.High() || checksum.Low() != nextPartitionChunk.Checksum.Low() {
        Log.Criticalf("Checksums don't match received high: %d calculated high: %d received low: %d calculated low: %d", nextPartitionChunk.Checksum.High(), checksum.High(), nextPartitionChunk.Checksum.Low(), checksum.Low())

        return PartitionChunk{}, EEntryChecksum
    }

    return nextPartitionChunk, nil
}

func (transfer *IncomingTransfer) Cancel() {
    transfer.err = ETransferCancelled
}

type OutgoingTransfer struct {
    partitionIterator PartitionIterator
    chunkSize int
    nextChunkIndex uint64
    entryFilter EntryFilter
    err error
}

func NewOutgoingTransfer(partition Partition, chunkSize int) *OutgoingTransfer {
    if chunkSize <= 0 {
        chunkSize = DefaultChunkSize
    }

    return &OutgoingTransfer{
        partitionIterator: partition.Iterator(),
        chunkSize: chunkSize,
        nextChunkIndex: 1,
    }
}

func (transfer *OutgoingTransfer) UseFilter(entryFilter EntryFilter) {
    transfer.entryFilter = entryFilter
}

func (transfer *OutgoingTransfer) NextChunk() (PartitionChunk, error) {
    if transfer.err != nil {
        return PartitionChunk{}, transfer.err
    }

    entries := make([]Entry, 0, transfer.chunkSize)

    for transfer.partitionIterator.Next() {
        entry := Entry{
            Site: transfer.partitionIterator.Site(),
            Bucket: transfer.partitionIterator.Bucket(),
            Key: transfer.partitionIterator.Key(),
            Value: transfer.partitionIterator.Value(),
        }

        // See if this value should be allowed or if it should be filtered out
        if transfer.entryFilter != nil && !transfer.entryFilter(entry) {
            continue
        }

        entries = append(entries, entry)

        if len(entries) == transfer.chunkSize {
            index := transfer.nextChunkIndex
            transfer.nextChunkIndex++

            return PartitionChunk{
                Index: index,
                Entries: entries,
                Checksum: ChecksumEntries(entries),
            }, nil
        }
    }

    transfer.partitionIterator.Release()

    if transfer.partitionIterator.Error() != nil {
        transfer.err = transfer.partitionIterator.Error()

        return PartitionChunk{}, transfer.err
    }

    transfer.err = io.EOF

    if len(entries) == 0 {
        return PartitionChunk{}, transfer.err
    }

    index := transfer.nextChunkIndex
    transfer.nextChunkIndex++

    return PartitionChunk{
        Index: index,
        Entries: entries,
        Checksum: ChecksumEntries(entries),
    }, nil
}

func (transfer *OutgoingTransfer) Cancel() {
    if transfer.err == nil {
        transfer.err = ETransferCancelled
    }

    transfer.partitionIterator.Release()
}
