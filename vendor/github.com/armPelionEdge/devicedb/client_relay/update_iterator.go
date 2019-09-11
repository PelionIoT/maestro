package client_relay

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"github.com/armPelionEdge/devicedb/transport"
)

type UpdateIterator interface {
	// Move to the next result. Returns
	// false if there is an error or if
	// there are no more results to iterate
	// through. If there is an error, the
	// Error() function will return the
	// error that occurred
	Next() bool
	// Return the next update
	Update() Update
	// Return the error that occurred
	// while iterating
	Error() error
}

type StreamedUpdateIterator struct {
	reader io.ReadCloser
	scanner *bufio.Scanner
	closed bool
	err error
	update Update
}

func (iter *StreamedUpdateIterator) Next() bool {
	if iter.closed {
		return false
	}

	if iter.scanner == nil {
		iter.scanner = bufio.NewScanner(iter.reader)
	}

	// data: %s line
	if !iter.scanner.Scan() {
		if iter.scanner.Err() != nil {
			iter.err = iter.scanner.Err()
		}

		iter.close()

		return false
	}

	if !strings.HasPrefix(iter.scanner.Text(), "data: ") {
		// protocol error.
		iter.err = errors.New("Protocol error")

		iter.close()

		return false
	}

	encodedUpdate := iter.scanner.Text()[len("data: "):]

	if encodedUpdate == "" {
		// this is a marker indicating the last of the initial
		// pushes of missed messages
		iter.update = Update{}
	} else {
		var update transport.TransportRow

		if err := json.Unmarshal([]byte(encodedUpdate), &update); err != nil {
			iter.err = err

			iter.close()

			return false
		}

		iter.update = Update{
			Key: update.Key,
			Serial: update.LocalVersion,
			Context: update.Context,
			Siblings: update.Siblings,
		}
	}

	// consume newline between "data: %s" lines
	if !iter.scanner.Scan() {
		if iter.scanner.Err() != nil {
			iter.err = iter.scanner.Err()
		}

		iter.close()

		return false
	}

	return true
}

func (iter *StreamedUpdateIterator) close() {
	iter.update = Update{}
	iter.closed = true
	iter.reader.Close()
}

func (iter *StreamedUpdateIterator) Update() Update {
	return iter.update
}

func (iter *StreamedUpdateIterator) Error() error {
	return iter.err
}