package data
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
    "encoding/gob"
    "bytes"
)

type Sibling struct {
    VectorClock *DVV `json:"clock"`
    BinaryValue []byte `json:"value"`
    PhysicalTimestamp uint64 `json:"timestamp"`
}

func NewSibling(clock *DVV, value []byte, timestamp uint64) *Sibling {
    return &Sibling{clock, value, timestamp}
}

func (sibling *Sibling) Clock() *DVV {
    return sibling.VectorClock
}

func (sibling *Sibling) Value() []byte {
    return sibling.BinaryValue
}

func (sibling *Sibling) IsTombstone() bool {
    return sibling.Value() == nil
}

func (sibling *Sibling) Timestamp() uint64 {
    return sibling.PhysicalTimestamp
}

func (sibling *Sibling) Hash() Hash {
    if sibling == nil || sibling.IsTombstone() {
        return Hash{[2]uint64{ 0, 0 }}
    }
    
    return NewHash(sibling.Value()).Xor(sibling.Clock().Hash())
}

// provides an ordering between siblings in order to break
// ties and decide which one to keep when two siblings have
// the same clock value. Favors keeping a value instead of a
// tombstone.
func (sibling *Sibling) Compare(otherSibling *Sibling) int {
    if sibling.IsTombstone() && !otherSibling.IsTombstone() {
        return -1
    } else if !sibling.IsTombstone() && otherSibling.IsTombstone() {
        return 1
    } else if sibling.IsTombstone() && otherSibling.IsTombstone() {
        if sibling.Timestamp() < otherSibling.Timestamp() {
            return -1
        } else if sibling.Timestamp() > otherSibling.Timestamp() {
            return 1
        } else {
            return 0
        }
    } else {
        return bytes.Compare(sibling.Value(), otherSibling.Value())
    }
}

func (sibling *Sibling) MarshalBinary() ([]byte, error) {
    var encoding bytes.Buffer
    encoder := gob.NewEncoder(&encoding)
    
    encoder.Encode(sibling.Clock())
    encoder.Encode(sibling.Timestamp())
    encoder.Encode(sibling.Value())
    
    return encoding.Bytes(), nil
}

func (sibling *Sibling) UnmarshalBinary(data []byte) error {
    var clock DVV
    var timestamp uint64
    var value []byte
    
    encoding := bytes.NewBuffer(data)
    decoder := gob.NewDecoder(encoding)
    
    decoder.Decode(&clock)
    decoder.Decode(&timestamp)
    decoder.Decode(&value)
    
    sibling.VectorClock = &clock
    sibling.PhysicalTimestamp = timestamp
    sibling.BinaryValue = value
    
    return nil
}
