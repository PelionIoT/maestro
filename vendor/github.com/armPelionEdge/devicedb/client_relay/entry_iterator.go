package client_relay

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"github.com/armPelionEdge/devicedb/client"
	"github.com/armPelionEdge/devicedb/transport"
)

type EntryIterator interface {
	// Move to the next result. Returns
	// false if there is an error or if
	// there are no more results to iterate
	// through. If there is an error, the
	// Error() function will return the
	// error that occurred
	Next() bool
	// Return the prefix that matches
	// the key for the current result
	Prefix() string
	// Return the key for the current
	// result
	Key() string
	// Return the value for the
	// current result
	Entry() client.Entry
	// Return the error that occurred
	// while iterating
	Error() error
}

type StreamedEntryIterator struct {
	reader io.ReadCloser
	scanner *bufio.Scanner
	closed bool
	err error
	key string
	prefix string
	entry client.Entry
}

func (iter *StreamedEntryIterator) Next() bool {
	if iter.closed {
		return false
	}

	if iter.scanner == nil {
		iter.scanner = bufio.NewScanner(iter.reader)
	}

	// prefix
	if !iter.scanner.Scan() {
		if iter.scanner.Err() != nil {
			iter.err = iter.scanner.Err()
		}

		iter.close()

		return false
	}

	iter.prefix = iter.scanner.Text()
	
	// key
	if !iter.scanner.Scan() {
		if iter.scanner.Err() != nil {
			iter.err = iter.scanner.Err()
		} else {
			iter.err = errors.New("Incomplete stream")
		}

		iter.close()

		return false
	}

	iter.key = iter.scanner.Text()

	// entry
	if !iter.scanner.Scan() {
		if iter.scanner.Err() != nil {
			iter.err = iter.scanner.Err()
		} else {
			iter.err = errors.New("Incomplete stream")
		}

		iter.close()

		return false
	}

	var siblingSet transport.TransportSiblingSet	

	if err := json.Unmarshal(iter.scanner.Bytes(), &siblingSet); err != nil {
		iter.err = err

		iter.close()

		return false
	}

	iter.entry.Context = siblingSet.Context
	iter.entry.Siblings = siblingSet.Siblings

	return true
}

func (iter *StreamedEntryIterator) close() {
	iter.prefix = ""
	iter.key = ""
	iter.entry = client.Entry{}
	iter.closed = true
	iter.reader.Close()
}

func (iter *StreamedEntryIterator) Prefix() string {
	return iter.prefix
}

func (iter *StreamedEntryIterator) Key() string {
	return iter.key
}

func (iter *StreamedEntryIterator) Entry() client.Entry {
	return iter.entry
}

func (iter *StreamedEntryIterator) Error() error {
	return iter.err
}