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
    "encoding/binary"
    "encoding/gob"
    "bytes"
)

type Dot struct {
    NodeID string `json:"node"`
    Count uint64 `json:"count"`
}

type DVV struct {
    VVDot Dot `json:"dot"`
    VV map[string]uint64 `json:"vv"`
}

func NewDot(nodeID string, count uint64) *Dot {
    return &Dot{nodeID, count}
}

func NewDVV(dot *Dot, vv map[string]uint64) *DVV {
    return &DVV{*dot, vv}
}

func (dvv *DVV) Dot() Dot {
    return dvv.VVDot
}

func (dvv *DVV) Context() map[string]uint64 {
    return dvv.VV
}

func (dvv *DVV) HappenedBefore(otherDVV *DVV) bool {
    if _, ok := otherDVV.Context()[dvv.Dot().NodeID]; ok {
        return dvv.Dot().Count <= otherDVV.Context()[dvv.Dot().NodeID]
    }
    
    return false
}

func (dvv *DVV) Replicas() []string {
    replicas := make([]string, 0, len(dvv.Context()) + 1)
    dotNodeID := dvv.Dot().NodeID
    
    for nodeID, _ := range dvv.Context() {
        replicas = append(replicas, nodeID)
        
        if nodeID == dotNodeID {
            dotNodeID = ""
        }
    }
    
    if len(dotNodeID) > 0 {
        replicas = append(replicas, dotNodeID)
    }
    
    return replicas
}

func (dvv *DVV) MaxDot(nodeID string) uint64 {
    var maxDot uint64
    
    if dvv.Dot().NodeID == nodeID {
        if dvv.Dot().Count > maxDot {
            maxDot = dvv.Dot().Count
        }
    }
    
    if _, ok := dvv.Context()[nodeID]; ok {
        if dvv.Context()[nodeID] > maxDot {
            maxDot = dvv.Context()[nodeID]
        }
    }
    
    return maxDot
}

func (dvv *DVV) Equals(otherDVV *DVV) bool {
    if dvv.Dot().NodeID != otherDVV.Dot().NodeID || dvv.Dot().Count != otherDVV.Dot().Count {
        return false
    }
    
    if len(dvv.Context()) != len(otherDVV.Context()) {
        return false
    }

    for nodeID, count := range dvv.Context() {
        if _, ok := otherDVV.Context()[nodeID]; !ok {
            return false
        }
        
        if count != otherDVV.Context()[nodeID] {
            return false
        }
    }
    
    return true
}

func (dvv *DVV) Hash() Hash {
    var hash Hash
    
    for nodeID, count := range dvv.Context() {
        countBuffer := make([]byte, 8)
        binary.BigEndian.PutUint64(countBuffer, count)
        
        hash = hash.Xor(NewHash([]byte(nodeID))).Xor(NewHash(countBuffer))
    }
    
    countBuffer := make([]byte, 8)
    binary.BigEndian.PutUint64(countBuffer, dvv.Dot().Count)
    
    hash = hash.Xor(NewHash([]byte(dvv.Dot().NodeID))).Xor(NewHash(countBuffer))
    
    return hash
}

func (dvv *DVV) MarshalBinary() ([]byte, error) {
    var encoding bytes.Buffer
    encoder := gob.NewEncoder(&encoding)
    
    encoder.Encode(dvv.Context())
    encoder.Encode(dvv.Dot().NodeID)
    encoder.Encode(dvv.Dot().Count)
    
    return encoding.Bytes(), nil
}

func (dvv *DVV) UnmarshalBinary(data []byte) error {
    var dot Dot
    var versionVector map[string]uint64
    
    encoding := bytes.NewBuffer(data)
    decoder := gob.NewDecoder(encoding)
    
    decoder.Decode(&versionVector)
    decoder.Decode(&dot.NodeID)
    decoder.Decode(&dot.Count)
    
    dvv.VVDot = dot
    dvv.VV = versionVector
    
    return nil
}
