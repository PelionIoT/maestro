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
    "crypto/md5"
    "encoding/binary"
)

const (
    HASH_SIZE_BYTES = 16
)

type Hash struct {
    Hash[2] uint64
}

func NewHash(input []byte) Hash {
    var newHash Hash
    
    sum := md5.Sum(input)
        
    newHash.Hash[1] = binary.BigEndian.Uint64(sum[0:8])
    newHash.Hash[0] = binary.BigEndian.Uint64(sum[8:16])
    
    return newHash
}

func (hash Hash) Xor(otherHash Hash) Hash {
    return Hash{[2]uint64{ hash.Hash[0] ^ otherHash.Hash[0], hash.Hash[1] ^ otherHash.Hash[1] }}
}

func (hash Hash) Bytes() [16]byte {
    var result [16]byte
    
    binary.BigEndian.PutUint64(result[0:8], hash.High())
    binary.BigEndian.PutUint64(result[8:16], hash.Low())
    
    return result
}

func (hash Hash) Low() uint64 {
    return hash.Hash[0]
}

func (hash Hash) SetLow(l uint64) Hash {
    hash.Hash[0] = l
    
    return hash
}

func (hash Hash) High() uint64 {
    return hash.Hash[1]
}

func (hash Hash) SetHigh(h uint64) Hash {
    hash.Hash[1] = h
    
    return hash
}
