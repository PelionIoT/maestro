package transport
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
    "encoding/base64"

    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/logging"
)

type TransportRow struct {
    Key string `json:"key"`
    LocalVersion uint64 `json:"serial"`
    Context string `json:"context"`
    Siblings []string `json:"siblings"`
}

func (tr *TransportRow) FromRow(row *Row) error {
    if row == nil || row.Siblings == nil {
        return nil
    }

    context, err := EncodeContext(row.Siblings.Join())

    if err != nil {
        Log.Warningf("Unable to encode context: %v", err)

        return EInvalidContext
    }

    tr.LocalVersion = row.LocalVersion
    tr.Key = row.Key
    tr.Context = context
    tr.Siblings = make([]string, 0, row.Siblings.Size())

    for sibling := range row.Siblings.Iter() {
        if !sibling.IsTombstone() {
            tr.Siblings = append(tr.Siblings, string(sibling.Value()))
        }
    }

    return nil
}

type TransportSiblingSet struct {
    Siblings []string `json:"siblings"`
    Context string `json:"context"`
}

func (tss *TransportSiblingSet) FromSiblingSet(siblingSet *SiblingSet) error {
    if siblingSet == nil {
        tss.Siblings = nil
        tss.Context = ""

        return nil
    }

    context, err := EncodeContext(siblingSet.Join())
    
    if err != nil {
        Log.Warningf("Unable to encode context: %v", err)
        
        return EInvalidContext
    }
    
    tss.Context = context
    tss.Siblings = make([]string, 0, siblingSet.Size())
    
    for sibling := range siblingSet.Iter() {
        if !sibling.IsTombstone() {
            tss.Siblings = append(tss.Siblings, string(sibling.Value()))
        }
    }
    
    return nil
}

func EncodeContext(context map[string]uint64) (string, error) {
    var encodedContext string
    
    rawJSON, err := json.Marshal(context)
    
    if err != nil {
        return "", err
    }
    
    encodedContext = base64.StdEncoding.EncodeToString(rawJSON)
    
    return encodedContext, nil
}

func DecodeContext(context string) (map[string]uint64, error) {
    var decodedContext map[string]uint64
    
    rawJSON, err := base64.StdEncoding.DecodeString(context)
    
    if err != nil {
        return nil, err
    }
    
    err = json.Unmarshal(rawJSON, &decodedContext)
    
    if err != nil {
        return nil, err
    }
    
    return decodedContext, nil
}

type TransportUpdateBatch []TransportUpdateOp

type TransportUpdateOp struct {
    Type string `json:"type"`
    Key string `json:"key"`
    Value string `json:"value"`
    Context string `json:"context"`
}

func (tub TransportUpdateBatch) ToUpdateBatch(updateBatch *UpdateBatch) error {
    var tempUpdateBatch = NewUpdateBatch()
    
    for _, tuo := range tub {
        if tuo.Type != "put" && tuo.Type != "delete" {
            Log.Warningf("%s is not a valid operation", tuo.Type)
            
            return EInvalidOp
        }
            
        var context map[string]uint64
        var err error
    
        if len(tuo.Context) != 0 {
            context, err = DecodeContext(tuo.Context)
            
            if err != nil {
                Log.Warningf("Could not decode context string %s in update operation: %v", tuo.Context, err)
                
                return EInvalidContext
            }
        }
    
        if tuo.Type == "put" {
            _, err = tempUpdateBatch.Put([]byte(tuo.Key), []byte(tuo.Value), NewDVV(NewDot("", 0), context))
        } else {
            _, err = tempUpdateBatch.Delete([]byte(tuo.Key), NewDVV(NewDot("", 0), context))
        }
        
        if err != nil {
            return err
        }
    }
    
    updateBatch.RawBatch = tempUpdateBatch.RawBatch
    updateBatch.Contexts = tempUpdateBatch.Contexts
    
    return nil
}

func (tub TransportUpdateBatch) FromUpdateBatch(updateBatch *UpdateBatch) error {
    if len(tub) != len(updateBatch.Batch().Ops()) {
        Log.Warningf("Transport update batch is not the same size as the update batch")
        
        return ELength
    }
    
    index := 0
    
    for k, op := range updateBatch.Batch().Ops() {
        context, ok := updateBatch.Context()[k]
        
        if !ok || context == nil {
            context = NewDVV(NewDot("", 0), map[string]uint64{ })
        }
        
        encodedContext, _ := EncodeContext(context.Context())
        
        if op.IsDelete() {
            tub[index] = TransportUpdateOp{
                Type: "delete",
                Key: k,
                Value: "",
                Context: encodedContext,
            }
        } else {
            tub[index] = TransportUpdateOp{
                Type: "put",
                Key: k,
                Value: string(op.Value()),
                Context: encodedContext,
            }
        }
        
        index += 1
    }
    
    return nil
}
