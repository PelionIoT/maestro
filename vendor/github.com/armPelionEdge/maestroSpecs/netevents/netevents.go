package netevents

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

const (
	// Net Events:
	// emitted when an interface or network state has changed
	// including new address, interface up / down

	// Event types
	// NetEvent_InterfaceStateDown is emitted when a network state has changed
	InterfaceStateDown = "interface-state-down"
	// InterfaceStateUp is emitted when a network state has changed
	InterfaceStateUp = "interface-state-up"

	// CloudReachable is emitted when the cloud (symphony) has become reachable
	CloudReachable = "cloud-reachable"
	// CloudUnreachable is emitted when the cloud (symphony) has become reachable
	CloudUnreachable = "cloud-unreachable"
	// NewAddress is emitted when an interface has a new address
	NewAddress = "new-address"
)

// NetEventData - All network events has this data struct
type NetEventData struct {
	Type      string              `json:"type"`
	Interface *InterfaceEventData `json:"interface"`
}

// InterfaceEventData is used if the event involves a change to an interface
type InterfaceEventData struct {
	// Id represents the interfaces, such as "eth0"
	ID string `json:"id"`
	// Index represents the interfaces 'index' value
	Index     int    `json:"index"`
	Address   string `json:"address"`
	AddressV6 string `json:"addressV6"`
	LinkState string `json:"linkstate"`
}
