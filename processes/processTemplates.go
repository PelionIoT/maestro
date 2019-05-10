package processes

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

// type CGroupSetup interface {
// 	GetMemLimit() uint64
// }

// type ContainerTemplate interface {
// 	GetName() string
// 	GetCGroupSetup() CGroupSetup
// 	GetExecCmd() string
// 	GetEnv() []string
// 	GetPreArgs() []string
// 	GetPostArgs() []string	
// 	IsInheritEnv() bool	
// 	// The process sends the magic maestro OK string on startup
// 	// In this case, maestro will wait for this string before setting
// 	// the job state to RUNNING
// 	// After this string it will redirect log output
// 	UsesOkString() bool 
// }

// type JobDefinition interface {
// 	GetJobName() string

// 	// get the template to use, if this is nil, then use the commands from
// 	// getExecCmd() / getArgs()
// 	GetContainerTemplate() string

// 	// a string to be sent via stdin on process startup
// 	GetMessageForProcess() string

// 	// the command to run. If the 
// 	GetExecCmd() string
// 	GetArgs() []string	
// 	GetEnv() []string
// 	GetPgid() int  // a value of zero indicate create a new PGID
// 	IsInheritEnv() bool
// 	IsDaemonize() bool
// 	GetDependsOn() []string
// 	IsAutostart() bool
// 	GetRestartOnDependencyFail() bool
// 	IsRestart() bool 
// 	GetRestartLimit() uint32
// 	GetRestartPause() uint32

// }