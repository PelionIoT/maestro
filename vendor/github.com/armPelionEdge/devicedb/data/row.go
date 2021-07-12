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
    "encoding/json"
)

type Row struct {
    Key string `json:"key"`
    LocalVersion uint64 `json:"localVersion"`
    Siblings *SiblingSet `json:"siblings"`
}

func (row *Row) Encode() []byte {
    result, _ := json.Marshal(row)

    return result
}

func (row *Row) Decode(encodedRow []byte, formatVersion string) error {
    if formatVersion == "0" {
        var siblingSet SiblingSet

        err := siblingSet.Decode(encodedRow)

        if err == nil {
            row.LocalVersion = 0
            row.Siblings = &siblingSet

            return nil
        }

        // if it fails to decode using format vesion
        // 0 which was without the row type then it might
        // have been converted already to the new format
        // version in a partially completed upgrade before.
        // in this case, try to decode it as a regular row
        // before returning an error
    }

    return json.Unmarshal(encodedRow, row)
}