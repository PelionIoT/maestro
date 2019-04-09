package configMgr

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

import (
	"github.com/WigWagCo/maestroSpecs"
	IFDEBUG("fmt")
)

type CompositeProcessConfigPayload struct {
	Moduleconfigs map[string]maestroSpecs.JobDefinition  `json:"moduleconfigs,omitempty"`
	ProcessConfig string `json:"process_config,omitempty"`
	// this is the path on the local file system to maestro local APIs
	// It is used for getting configs
	// Later this will be differentiated from a 'root' socket path
	// which will allow additional APIs. 
    MaestroUserSocketPath string `json:"maestro_user_socket_path,omitempty"`
    // For privleged APIs, which will starting and stopping processes, etc.
    // UNIMPLEMENTED
    MaestroRootSocketPath string `json:"maestro_root_socket_path,omitempty"`
}

func NewCompositeProcessConfigPayload() (ret *CompositeProcessConfigPayload) {
	ret = new(CompositeProcessConfigPayload)
	ret.Moduleconfigs = make(map[string]maestroSpecs.JobDefinition)
	DEBUG_OUT("UNIX %s %s\n",JobConfigMgr().globalConfig.GetHttpUnixSocket(), JobConfigMgr().globalConfig.HttpUnixSocket)
	ret.MaestroUserSocketPath = JobConfigMgr().globalConfig.GetHttpUnixSocket()
	return
}
