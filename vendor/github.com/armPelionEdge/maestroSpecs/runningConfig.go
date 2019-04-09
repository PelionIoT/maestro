package maestroSpecs

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

// Composite configurations
// used for setting up a system to a particular state


type RunningConfig interface {
	GetJobs() []JobDefinition
	GetTemplates() []ContainerTemplate
	GetImages() []ImageDefinition
}

type RunningConfigOp interface {
	GetOp() string
	GetConfig() RunningConfig
	GetFlags() []string
}


// This payload setups an entire running configuration, comprised of Jobs/Templates/Images
type RunningConfigPayload struct {
	Jobs []JobDefinitionPayload `yaml:"jobs" json:"jobs"`
	Templates []ContainerTemplatePayload `yaml:"templates" json:"templates"`
	Images []ImageDefinitionPayload `yaml:"images" json:"images"`
	Configs []ConfigDefinition `yaml:"configs" json:"configs"`
}


// if listed in Flags, this will make maestro remove any
// images no longer in use
const FLAG_RUNNING_CONFIG_REMOVE_IMAGES = "removeOldImages"

// if set the maestro will stop anytime any of the config
// fails to install
const FLAG_RUNNING_CONFIG_FAIL_FAST = "failFast"

// the update OP will change the running config.
// Jobs not in the current running config, are stopped & removed
// Jobs in the new Running config are added & started.
const OP_RUNNING_CONFIG_UPDATE = "update"

type RunningConfigOpPayload struct {
	Op string `yaml:"op" json:"op"`
	Config *RunningConfigPayload `yaml:"conf" json:"conf"`
	Flags []string `yaml:"flags" json:"flags"`
}
