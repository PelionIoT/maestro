package relaymq
//
// Copyright (c) 2018, Arm Limited and affiliates.
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
    eSTORAGE = iota
    eCORRUPTED = iota
)

var (
    EEmpty                 = DBerror{ "Parameter was empty or nil", eEMPTY }
    ELength                = DBerror{ "Parameter is too long", eLENGTH }
    EStorage               = DBerror{ "The storage driver experienced an error", eSTORAGE }
    ECorrupted             = DBerror{ "The storage medium is corrupted", eCORRUPTED }
)
