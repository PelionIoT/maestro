package error
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

type DBerror struct {
    Msg string `json:"message"`
    ErrorCode int `json:"code"`
}

func (dbError DBerror) Error() string {
    return dbError.Msg
}

func (dbError DBerror) Code() int {
    return dbError.ErrorCode
}

func (dbError DBerror) JSON() []byte {
    json, _ := json.Marshal(dbError)
    
    return json
}

const (
    eEMPTY = iota
    eLENGTH = iota
    eNO_VNODE = iota
    eSTORAGE = iota
    eCORRUPTED = iota
    eINVALID_KEY = iota
    eINVALID_BUCKET = iota
    eINVALID_BATCH = iota
    eMERKLE_RANGE = iota
    eINVALID_OP = iota
    eINVALID_CONTEXT = iota
    eUNAUTHORIZED = iota
    eINVALID_PEER = iota
    eREAD_BODY = iota
    eREQUEST_QUERY = iota
    eALERT_BODY = iota
    eNODE_CONFIG_BODY = iota
    eNODE_DECOMMISSIONING = iota
    ePROPOSAL_ERROR = iota
    eDUPLICATE_NODE_ID = iota
    eNO_SUCH_SITE = iota
    eNO_SUCH_RELAY = iota
    eNO_SUCH_BUCKET = iota
    eNO_QUORUM = iota
    eOPERATION_LOCKED = iota
    eSNAPSHOT_IN_PROGRESS = iota
    eSNAPSHOT_OPEN_FAILED = iota
    eSNAPSHOT_READ_FAILED = iota
)

var (
    EEmpty                 = DBerror{ "Parameter was empty or nil", eEMPTY }
    ELength                = DBerror{ "Parameter is too long", eLENGTH }
    ENoVNode               = DBerror{ "This node does not contain keys in this partition", eNO_VNODE }
    EStorage               = DBerror{ "The storage driver experienced an error", eSTORAGE }
    ECorrupted             = DBerror{ "The storage medium is corrupted", eCORRUPTED }
    EInvalidKey            = DBerror{ "A key was misformatted", eINVALID_KEY }
    EInvalidBucket         = DBerror{ "An invalid bucket was specified", eINVALID_BUCKET }
    EInvalidBatch          = DBerror{ "An invalid batch was specified", eINVALID_BATCH }
    EMerkleRange           = DBerror{ "An invalid merkle node was requested", eMERKLE_RANGE }
    EInvalidOp             = DBerror{ "An invalid operation was specified", eINVALID_OP }
    EInvalidContext        = DBerror{ "An invalid context was provided in an update", eINVALID_CONTEXT }
    EUnauthorized          = DBerror{ "Operation not permitted", eUNAUTHORIZED }
    EInvalidPeer           = DBerror{ "The specified peer is invalid", eINVALID_PEER }
    EReadBody              = DBerror{ "Unable to read request body", eREAD_BODY }
    ERequestQuery          = DBerror{ "Invalid query parameter format", eREQUEST_QUERY }
    EAlertBody             = DBerror{ "Invalid alert body. Body must be true or false", eALERT_BODY }
    ENodeConfigBody        = DBerror{ "Invalid node config body.", eNODE_CONFIG_BODY }
    ENodeDecommissioning   = DBerror{ "This node is in the process of leaving the cluster.", eNODE_DECOMMISSIONING }
    EDuplicateNodeID       = DBerror{ "The ID the node is using was already used by a cluster member at some point.", eDUPLICATE_NODE_ID }
    EProposalError         = DBerror{ "An error occurred while proposing cluster configuration change.", ePROPOSAL_ERROR }
    ESiteDoesNotExist      = DBerror{ "The specified site does not exist at this node.", eNO_SUCH_SITE }
    ERelayDoesNotExist     = DBerror{ "The specified relay does not exist at this node.", eNO_SUCH_RELAY }
    EBucketDoesNotExist    = DBerror{ "The site does not contain the specified bucket.", eNO_SUCH_BUCKET }
    ENoQuorum              = DBerror{ "The database operation was not able to achieve participation from the necessary number of replicas.", eNO_QUORUM }
    EOperationLocked       = DBerror{ "The attempted operation is currently locked on the partition that the specified data belongs to.", eOPERATION_LOCKED }
    ESnapshotInProgress    = DBerror{ "The specified snapshot is still in progress", eSNAPSHOT_IN_PROGRESS }
    ESnapshotOpenFailed    = DBerror{ "The snapshot could not be opened.", eSNAPSHOT_OPEN_FAILED }
    ESnapshotReadFailed    = DBerror{ "The snapshot could be opened, but it appears to be incomplete or invalid.", eSNAPSHOT_READ_FAILED }
)

func DBErrorFromJSON(encodedError []byte) (DBerror, error) {
    var dbError DBerror

    if err := json.Unmarshal(encodedError, &dbError); err != nil {
        return DBerror{}, err
    }

    return dbError, nil
}