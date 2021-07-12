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


type Diff struct {
    key string
    oldSiblingSet *SiblingSet
    newSiblingSet *SiblingSet
}

func (diff Diff) OldSiblingSet() *SiblingSet {
    return diff.oldSiblingSet
}

func (diff Diff) NewSiblingSet() *SiblingSet {
    return diff.newSiblingSet
}

func (diff Diff) Key() string {
    return diff.key
}

type Update struct {
    diffs map[string]Diff
}

func NewUpdate() *Update {
    return &Update{ map[string]Diff{ } }
}

func (update *Update) AddDiff(key string, oldSiblingSet *SiblingSet, newSiblingSet *SiblingSet) *Update {
    update.diffs[key] = Diff{key, oldSiblingSet, newSiblingSet}
    
    return update
}

func (update *Update) Iter() <-chan Diff {
    ch := make(chan Diff)
    
    go func() {
        for _, diff := range update.diffs {
            ch <- diff
        }
        
        close(ch)
    } ()
    
    return ch
}

func (update *Update) Size() int {
    return len(update.diffs)
}