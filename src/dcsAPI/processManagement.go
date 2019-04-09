package dcsAPI

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

import "github.com/armPelionEdge/maestroSpecs"


/**
 * Types in golang regarding API messages from WigWag DCS services
 *
 * Process & application management schemas
 *
 */


type Job struct {
	Job string `json:"job"`
	CompositeId string `json:"composite_id"`
	ContainerTemplate string `json:"container_template"`
	Config string `json:"config"`
	Disabled bool `json:"disabled"`
}

// type Image struct {
// 	Job string `json:"job"`
// 	Version string `json:"version"`
// 	URL string `json:"url"`
// 	Size uint32 `json:"size"`	
// 	Checksum maestroSpecs.ChecksumObj `json:"checksum"`
// 	ImageType string `json:"image_type"`  // devicejs_tarball
// }

type AppConfig struct {
	Name string `json:"name"`
	Job string `json:"job"`
	Data string `json:"data"`  // the actual config
	Encoding string `json:"utf8"` // always utf8 for now
}

// Still need an example of this:
type Template struct {
	Name string `json:"name"`  
}

// This type is analagous to the DCS APIs /relayconfigurations
// 
type GatewayConfigMessage struct {
	Jobs []Job `json:"jobs"`
	Images []maestroSpecs.ImageDefinitionPayload `json:"images"`
	Configs []AppConfig `json:"configs"`
	TaskId string `json:"taskID"`
	StatusEndpoint string `json:"statusEndpoint"`
}
