package softRelay

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

type keyPair struct {
	Key  string `json:"key"`
	Cert string `json:"certificate"`
}

type keyCa struct {
	CA           string `json:"ca"`
	Intermediate string `json:"intermediate"`
}

type sslStuff struct {
	Intermediate *keyPair `json:"intermediate"`
	// No longer used:
	Client *keyPair `json:"client"`
	Server *keyPair `json:"server"`
	CA     *keyCa   `json:"ca"`
}

type eepromData struct {
	Batch            string    `json:"batch"`
	Month            string    `json:"month"`
	Year             string    `json:"year"`
	RadioConfig      string    `json:"radioConfig"`
	HardwareVersion  string    `json:"HardwareVersion"`
	RelayID          string    `json:"relayID"`
	EthernetMAC      []int     `json:"ethernetMAC"`
	SixBMAC          []int     `json:"sixBMAC"`
	RelaySecret      string    `json:"relaySecret"`
	PairingCode      string    `json:"pairingCode"`
	SSL              *sslStuff `json:"ssl"`
	CloudURL         string    `json:"cloudURL"`
	DevicejsCloudURL string    `json:"devicejsCloudURL"`
	DevicedbCloudURL string    `json:"devicedbCloudURL"`
	RelayMQCloudURL  string    `json:"relaymqCloudURL"`
	HistoryCloudURL  string    `json:"historyCloudURL"`
}
