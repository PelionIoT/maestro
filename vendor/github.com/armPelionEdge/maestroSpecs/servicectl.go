package maestroSpecs

import (
	"github.com/armPelionEdge/greasego"
)

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

// A Logger interface is passed to some Maestro plugins, allowing the
// plugin to Log to Maestro's internal logs, as part of the Maestro
// process.


type ServiceCtlConfigPayload struct {
	Services []*Service `yaml:"services" json:"services" servicectl_group:"services"`
}

type Service struct {
	//name of the service
	Name                     string                           `yaml:"name,omitempty" json:"name" servicectl_group:"servicectl"`
	// enable:true is equivalent to "systemctl enable servicename", enable:false is equivalent to "systemctl disable servicename"
	Enable				     bool							  `yaml:"enable,omitempty" json:"enable" servicectl_group:"servicectl"`
	StartonBoot				 bool							  `yaml:"start_on_boot,omitempty" json:"start_on_boot" servicectl_group:"servicectl"`
	StartedonBoot            bool
	// shows the status of the service running | dead | stopped
	Status	                 string
	// period to update the status parameter mentioned above
	StatusUpdatePeriod       uint64                           `yaml:"status_update_period,omitempty" json:"status_update_period" servicectl_group:"servicectl"`
	// restart the service if set to true and set it back to false if restart successful.
	RestartService           bool                             `yaml:"restart_service,omitempty" json:"restart_service" servicectl_group:"servicectl"`
	// start the service if set to true and set it back to false if start successful.
	StartService             bool                             `yaml:"start_service,omitempty" json:"start_service" servicectl_group:"servicectl"`
	// stop the service if set to true and set it back to false if stop successful.
	StopService              bool                             `yaml:"stop_service,omitempty" json:"stop_service" servicectl_group:"servicectl"`
	IsEnabled                bool
	IsRunning 				 bool
}

