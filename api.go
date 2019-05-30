package maestro

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

type Msg_StartProcess struct {
	Path        string   `json:"path"`
	Arguments   []string `json:"arguments"`
	Environment []string `json:"environment"`
	// enherit the standard environment that
	// maestro was started in?
	InheritEnv  bool   `json:"inheritEnv"`
	ContainerId string `json:"ContainerId"`
	Pgid        int    `json:"pgid"`
	Daemonize   bool   `json:"daemonize"`
}

type Msg_StopProcess struct {
}

type Msg_JobStartRequest struct {
	Job                     string   `json:"job"`
	ContainerTemplate       string   `json:"container_template"`
	Message                 string   `json:"message"`
	NoAutoStart             bool     `json:"no_autostart"`
	Restart                 bool     `json:"restart"`
	RestartOnDependencyFail bool     `json:"restart_on_dependency_fail"`
	RestartLimit            uint32   `json:"restart_limit"`
	RestartPause            uint32   `json:"restart_pause"`
	DependsOn               []string `json:"depends_on"`
	Pgid                    int      `json:"pgid"`
	ExecCmd                 string   `json:"exec_cmd"`
	ExecArgs                []string `json:"exec_args"`
	Env                     []string `json:"env"`
	InheritEnv              bool     `json:"inherit_env"`
	Daemonize               bool     `json:"daemonize"` // create a new SID for this process ?
}

func (this *Msg_JobStartRequest) GetJobName() string {
	return this.Job
}
func (this *Msg_JobStartRequest) GetContainerTemplate() string {
	return this.ContainerTemplate
}
func (this *Msg_JobStartRequest) GetMessageForProcess() string {
	return this.Message
}
func (this *Msg_JobStartRequest) GetExecCmd() string {
	return this.ExecCmd
}
func (this *Msg_JobStartRequest) GetArgs() []string {
	return this.ExecArgs
}
func (this *Msg_JobStartRequest) GetEnv() []string {
	return this.Env
}
func (this *Msg_JobStartRequest) GetPgid() int {
	return this.Pgid
}
func (this *Msg_JobStartRequest) IsInheritEnv() bool {
	return this.InheritEnv
}
func (this *Msg_JobStartRequest) IsDaemonize() bool {
	return this.Daemonize
}
func (this *Msg_JobStartRequest) GetDependsOn() []string {
	return this.DependsOn
}
func (this *Msg_JobStartRequest) IsAutostart() bool {
	return !this.NoAutoStart
}
func (this *Msg_JobStartRequest) GetRestartOnDependencyFail() bool {
	return this.RestartOnDependencyFail
}
func (this *Msg_JobStartRequest) IsRestart() bool {
	return this.Restart
}
func (this *Msg_JobStartRequest) GetRestartLimit() uint32 {
	return this.RestartLimit
}
func (this *Msg_JobStartRequest) GetRestartPause() uint32 {
	return this.RestartPause
}

type Msg_StopJob struct {
}

type Msg_CreateContainer struct {
}

type Msg_TeardownContainer struct {
}

type Msg_SendImage struct {
}

type Msg_EraseImage struct {
}
