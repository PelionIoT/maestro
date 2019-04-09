package maestroSpecs
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


type JobStatus struct {
	Job               string `yaml:"job" json:"job"`
	CompositeId       string `yaml:"composite_id" json:"composite_id"`
	ContainerTemplate string `yaml:"container_template" json:"container_template"`
	// the PID of the process running this Job. If this is a composite process (has more than one job)
	// then this is the PID of that process
	Pid int `yaml:"pid" json:"pid"`
	// A status of "RUNNING" means the process is running and the Pid will be non-zero
	Status string `yaml:"status" json:"status"`
}

type RunningStatus struct {
	Jobs []JobStatus `yaml:"jobs" json:"jobs"`
	// Uptime is how long maestro has been running
	Uptime int64
	// CpuUptime is how long this machine has been running
	CpuUptome int64
}
