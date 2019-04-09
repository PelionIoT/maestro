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
	"net/http"
)

type IdentityHeader struct {
	ClientType string `json:"clientType"`
	RelayID string `json:"relayID"`
	AccountID string `json:"accountID"`
}

func (identity *IdentityHeader) Header() http.Header {
	if identity.ClientType == "" {
		return http.Header{ }
	}

	header := http.Header{ }
	encoded, _ := json.Marshal(identity)

	header.Add("X-WigWag-Identity", string(encoded))

	return header
}

func (identity *IdentityHeader) FromHeader(header http.Header) error {
	err := json.Unmarshal([]byte(header.Get("X-WigWag-Identity")), identity)

	if err != nil {
		return err
	}

	return nil
}
