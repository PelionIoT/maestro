package client_relay
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