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

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type CGroupSetup interface {
	GetMemLimit() uint64
}

type ContainerTemplate interface {
	GetName() string
	GetCGroupSetup() CGroupSetup
	GetExecCmd() string
	GetEnv() []string
	GetPreArgs() []string
	GetPostArgs() []string
	GetDependsOn() []string
	IsInheritEnv() bool
	// The process sends the magic maestro OK string on startup
	// In this case, maestro will wait for this string before setting
	// the job state to RUNNING
	// After this string it will redirect log output
	UsesOkString() bool
	GetDeferRunningStatus() uint32
	// Maestro will attempt to make the child process die
	// if Maestro itself dies. This is not always possible, and may not work
	// on non Linux systems
	IsDieOnParentDeath() bool
	// if true, then all Job data for which shares the same 'composite ID', which uses this
	// container type, will be sent to the started process via stdin
	// The data will be encoded as a JSON object of the format:
	// { "jobs" : [
	//      { JOB_CONFIG }
	//      ...
	//   ]
	// }
	// Where JOB_CONFIG the JSON encoded version of the job configuration which has that
	// composite ID
	IsSendCompositeJobsToStdin() bool
	// make a GREASE_ORIGIN_ID env var, and then when execvpe the new process, attach this env var
	// which will have the generated origin ID for the process. Useful for deviceJS
	IsSendGreaseOriginIdEnv() bool
	// If this container is used as a template for a composite process
	// then this string is sent as the 'process_config' value in the JSON object
	// (composite process config) which is sent via stdout
	GetCompositeConfig() string
	// Returns true if the template definiton can be modified
	// defaults to true if never set
	IsMutable() bool
}

type JobDefinition interface {
	GetJobName() string

	// If len > 0, then this Job can be placed in a composite process
	// identified by this string
	// So, if multiple Jobs can be in the same process, then each Job must share
	// this same value AND each Job must also use the same Container Template string
	GetCompositeId() string

	// get the template to use, if this is nil, then use the commands from
	// getExecCmd() / getArgs()
	GetContainerTemplate() string

	// the Config data and/or files to be used for this Job
	// this config, should match up with a defined Config object
	// for this Job name
	GetConfigName() string
	// this returns the config string which may have been provided with the
	// Job. This is not necessarily the current config or the one which
	// should be provided to the Job on startup
	GetStaticConfig() string
	// Maestro will attempt to make the child process die
	// if Maestro itself dies. This is not always possible, and may not work
	// on non Linux systems
	IsDieOnParentDeath() bool

	// a string to be sent via stdin on process startup
	GetMessageForProcess() string
	UsesOkString() bool
	GetDeferRunningStatus() uint32
	// the command to run. If the
	GetExecCmd() string
	GetArgs() []string
	GetEnv() []string
	GetPgid() int // a value of zero indicate create a new PGID
	IsInheritEnv() bool
	IsDaemonize() bool
	GetDependsOn() []string
	IsAutostart() bool
	GetRestartOnDependencyFail() bool
	IsRestart() bool
	GetRestartLimit() uint32
	GetRestartPause() uint32
	// Returns true if the job definiton can be modified
	// defaults to true if never set
	IsMutable() bool
}

const CHECKSUM_TYPE_SHA256 = "sha256sum"
const CHECKSUM_TYPE_MD5 = "md5sum"

type Checksum interface {
	GetType() string // CHECKSUM_TYPE_xxxx
	GetValue() string
}

const IMAGE_TYPE_DOCKER = "docker"
const IMAGE_TYPE_TARBALL = "tarball"
const IMAGE_TYPE_ZIP = "zip"
const IMAGE_TYPE_DEVICEJS_TARBALL = "devicejs_tarball"
const IMAGE_TYPE_DEVICEJS_ZIP = "devicejs_zip"

type ImageDefinition interface {
	GetJobName() string
	GetURL() string
	GetVersion() string
	GetSize() uint64
	GetChecksum() Checksum
	GetImageType() string // IMAGE_TYPE_xxxxx
	ComputeSignature() string
}

const OP_ADD = "add"
const OP_REMOVE = "rm"
const OP_UPDATE = "update"

const OP_FLAG_RESTART = "restart" // force a restart

const OP_TYPE_IMAGE = "image"
const OP_TYPE_JOB = "job"

type Operation interface {
	GetType() string // operation type
	GetOp() string
	GetParams() []string
	GetTaskId() string
}

type ImageOperation interface {
	GetType() string // operation type
	GetOp() string
	GetParams() []string
	GetTaskId() string
	GetImageDefinition() ImageDefinition
	GetAppName() string
	GetVersion() string
}

type JobOperation interface {
	GetType() string // operation type
	GetOp() string
	GetFlags() []string
	GetParams() []string
	GetJobPayload() JobDefinition
	GetTaskId() string
}

type ContainerTemplOperation interface {
	GetOp() string
	GetParams() []string
	GetTemplate() *ContainerTemplate
}

type StatsConfig interface {
	GetInterval() uint32
	GetConfig_CheckMem() (uint32, bool)
}

// implements processes/processTemplates.go:CGroupSetup
type CGroupSetupConfig struct {
	MemLimit uint64 `yaml:"mem_limit"`
}

// func NewImageOpPayload() *ImageOpPayload {
// 	return &ImageOpPayload{Type:OP_TYPE_IMAGE}
// }
// func (this *ImageOpPayload) GetType() string {
// 	return "job"
// }
// func (this *ImageOpPayload) GetOp() string {
// 	return this.Op
// }
// func (this *ImageOpPayload) GetFlags() []string {
// 	return this.Flags
// }
// func (this *ImageOpPayload) GetParams() []string {
// 	return this.Params
// }
// func (this *ImageOpPayload) GetImagePayload() ImageDefinition {
// 	return this.Image
// }
// func (this *ImageOpPayload) GetTaskId() string {
// 	return this.TaskId
// }

type genericOpPayload struct {
	Type   string `yaml:"type" json:"type"` // should always be 'job'
	Op     string `yaml:"op" json:"op"`
	TaskId string `yaml:"task_id" json:"task_id"`
}

type JobOpPayload struct {
	Type   string `yaml:"type" json:"type"` // should always be 'job'
	Op     string `yaml:"op" json:"op"`
	Flags  []string
	Job    *JobDefinitionPayload `yaml:"job" json:"job"`
	Params []string              `yaml:"params" json:"params"`
	TaskId string                `yaml:"task_id" json:"task_id"`
}

func NewJobOpPayload() *JobOpPayload {
	return &JobOpPayload{Type: OP_TYPE_JOB}
}
func (this *JobOpPayload) GetType() string {
	return OP_TYPE_JOB
}
func (this *JobOpPayload) GetOp() string {
	return this.Op
}
func (this *JobOpPayload) GetFlags() []string {
	return this.Flags
}
func (this *JobOpPayload) GetParams() []string {
	return this.Params
}
func (this *JobOpPayload) GetJobPayload() JobDefinition {
	return this.Job
}
func (this *JobOpPayload) GetTaskId() string {
	return this.TaskId
}

type ImageOpPayload struct {
	Type    string `yaml:"type" json:"type"` // should always be 'job'
	Op      string `yaml:"op" json:"op"`
	AppName string `yaml:"app_name" json:"app_name"` // not always needed, especially with those stated in
	Version string `yaml:"app_name" json:"app_name"` // config file
	Flags   []string
	Image   *ImageDefinitionPayload `yaml:"image" json:"image"`
	Params  []string                `yaml:"params" json:"params"`
	TaskId  string                  `yaml:"task_id" json:"task_id"`
}

func NewImageOpPayload() *ImageOpPayload {
	return &ImageOpPayload{Type: "image"}
}
func (this *ImageOpPayload) GetType() string {
	return this.Type
}
func (this *ImageOpPayload) GetOp() string {
	return this.Op
}
func (this *ImageOpPayload) GetAppName() string {
	return this.AppName
}
func (this *ImageOpPayload) GetVersion() string {
	return this.Version
}
func (this *ImageOpPayload) GetFlags() []string {
	return this.Flags
}
func (this *ImageOpPayload) GetParams() []string {
	return this.Params
}
func (this *ImageOpPayload) GetImageDefinition() ImageDefinition {
	return this.Image
}
func (this *ImageOpPayload) GetTaskId() string {
	return this.TaskId
}

type TaskIdResponse interface {
	GetTaskId() string
}
type TaskIdResponsePayload struct {
	TaskId string `yaml:"task_id" json:"task_id"`
}

func (this *TaskIdResponsePayload) GetTaskId() string {
	return this.TaskId
}

const TASK_QUEUED = "queued"
const TASK_RUNNING = "running"
const TASK_COMPLETE = "complete"
const TASK_FAILED = "failed"

type TaskStatus interface {
	GetTaskStatus() string
	GetData() string
}
type TaskStatusPayload struct {
	TaskId string `yaml:"task_id" json:"task_id"`
	Status string `yaml:"status" json:"status"`
	Data   string `yaml:"data" json:"data"`
}

func (this *TaskStatusPayload) GetStatus() string {
	return this.Status
}
func (this *TaskStatusPayload) GetData() string {
	return this.Data
}

const JOB_OK_STRING = "!!OK!!"
const JOB_END_CONFIG_STRING = "ENDCONFIG"

// special container names:
// deviceJS_process is a special container name for deviceJS scripts
// This is currently the only type of process supporting "compositeID"
// which allows multiple Jobs to be in the same OS process
const CONTAINER_DEVICEJS_PROCESS = "deviceJS_process"

// as above, but uses node 8.x instead of older node.js
const CONTAINER_DEVICEJS_PROCESS_NODE8 = "deviceJS_process_node8"

// implements processes/processTemplates.go:ContainerTemplate
type ContainerTemplatePayload struct {
	Name   string             `yaml:"name"`
	CGroup *CGroupSetupConfig `yaml:"cgroup" json:"cgroup"`

	// if true, the image will be ran chroot-ed, wherever it is installed
	ChrootImage bool
	// applies only for job's without images. Provides a path for the process should be chroot()
	ChrootPath string `yaml:"chroot_path" json:"chroot_path"`

	InheritEnv   bool     `yaml:"inherit_env" json:"inherit_env"`
	AddEnv       []string `yaml:"add_env" json:"add_env"`
	ExecCmd      string   `yaml:"exec_cmd" json:"exec_cmd"`
	ExecPreArgs  []string `yaml:"exec_pre_args" json:"exec_pre_args"`
	ExecPostArgs []string `yaml:"exec_post_args" json:"exec_post_args"`

	DependsOn []string `yaml:"depends_on" json:"depends_on"`

	Restart                 bool   `yaml:"restart" json:"restart"`
	RestartOnDependencyFail bool   `yaml:"restart_on_dependency_fail" json:"restart_on_dependency_fail"`
	RestartLimit            uint32 `yaml:"restart_limit" json:"restart_limit"`
	RestartPause            uint32 `yaml:"restart_pause" json:"restart_pause"`
	DieOnParentDeath        bool   `yaml:"die_on_parent_death" json:"die_on_parent_death"`
	// if set to true, then all Job data for which shares the same 'composite ID', which uses this
	// container type, will be sent to the started process via stdin
	// The data will be encoded as a JSON object of the format:
	// { "jobs" : [
	//      { JOB_CONFIG }
	//      ...
	//   ]
	// }
	// Where JOB_CONFIG the JSON encoded version of the job configuration which has that
	// composite ID
	SendCompositeJobsToStdin bool `yaml:"send_composite_jobs_to_stdin" json:"send_composite_jobs_to_stdin"`
	// If this container is used as a template for a composite process
	// then this string is sent as the 'process_config' value in the JSON object
	// (composite process config) which is sent via stdout
	CompositeConfig string `yaml:"composite_config" json:"composite_config"`
	// make a GREASE_ORIGIN_ID env var, and then when execvpe the new process, attach this env var
	// which will have the generated origin ID for the process. Useful for deviceJS
	SendGreaseOriginIdEnv bool `yaml:"send_grease_origin_id" json:"send_grease_origin_id"`
	// this is a maestro thing. It will wait for the process to send back
	// a string of "!!OK!!" on STDOUT - until this is sent back, the process is marked
	// as "STARTING" but not "RUNNING" and it's STDOUT is not sent into the logger
	OkString bool `yaml:"uses_ok_string" json:"uses_ok_string"`
	// Defers the RUNNING status change on a Job for N milliseconds
	// This is not compatible with the uses_ok_string option above.
	// This is designed to ask Maestro to wait for a processes to fully start
	// before trying to start another Job who's dependcies are waiting
	// on this process
	DeferRunningStatus uint32 `yaml:"defer_running_status" json:"defer_running_status"`
	// if set, then if the OK message is not recieved in X ms, the job will be concedered failed
	// and the process will be killed
	TimeoutForOk uint32 `yaml:"timeout_for_ok" json:"timeout_for_ok"`
	// true if the definition can not be modified
	// Note: In the config file, this should be set to true, if you wish Maestro to
	// not save settings in the config file to the maestro database.
	Immutable bool `yaml:"immutable" json:"immutable"`
}

type ChecksumObj struct {
	Type  string `yaml:"type" json:"type"` // CHECKSUM_TYPE_xxxx
	Value string `yaml:"value" json:"value"`
}

func (this *ChecksumObj) GetType() string {
	return this.Type
}
func (this *ChecksumObj) GetValue() string {
	return this.Value
}

type ImageDefinitionPayload struct {
	Job       string       `yaml:"job" json:"job"` // the job this image is associated with
	Version   string       `yaml:"version" json:"version"`
	URL       string       `yaml:"url" json:"url"`
	Size      uint64       `yaml:"size" json:"size"`
	Checksum  *ChecksumObj `yaml:"checksum" json:"checksum"`
	ImageType string       `yaml:"image_type" json:"image_type"` // IMAGE_TYPE_xxxxx
}

func (this *ImageDefinitionPayload) GetJobName() string {
	return this.Job
}
func (this *ImageDefinitionPayload) GetVersion() string {
	return this.Version
}
func (this *ImageDefinitionPayload) GetURL() string {
	return this.URL
}
func (this *ImageDefinitionPayload) GetSize() uint64 {
	return this.Size
}
func (this *ImageDefinitionPayload) GetChecksum() Checksum {
	return this.Checksum
}
func (this *ImageDefinitionPayload) GetImageType() string {
	return this.ImageType
}

// Computed a signature of the operation based on specific fields.
// To deal with an API call which does the same thing multiple times
func (this *ImageDefinitionPayload) ComputeSignature() string {
	s := this.Job + ":" + this.Version + ":"
	s += fmt.Sprintf("%d:", this.Size)
	if this.Checksum != nil {
		s += this.Checksum.Value
	}
	out := sha256.Sum256([]byte(s))
	return string(out[:])
}

// implements processes/processTemplates.go:JobDefinition
type JobDefinitionPayload struct {
	Job               string `yaml:"job" json:"job"`
	CompositeId       string `yaml:"composite_id" json:"composite_id"`
	ContainerTemplate string `yaml:"container_template" json:"container_template"`
	// The configuration for this Job
	// If the Config has the Data field set up, it will be sent this config on stdin
	// once its process has started
	// If the Job is part of a 'composite process', it's config will be sent as part of an
	// array of strings, with the array and string encoded in JSON
	// The Config's "name" is "default" unless config_name is stated
	Config string `yaml:"config" json:"config"`
	// The name of the config. If config_name is not provided, it will be
	// the string "default"
	ConfigName string `yaml:"config_name" json:"config_name"`
	// Sent to the process on stdout after any 'data' config
	Message string `yaml:"message" json:"message"`
	// this is a maestro thing. It will wait for the process to send back
	// a string of "!!OK!!" on STDOUT - until this is sent back, the process is marked
	// as "STARTING" but not "RUNNING" and it's STDOUT is not sent into the logger
	WaitForOk bool `yaml:"wait_for_ok" json:"wait_for_ok"`
	// Defers the RUNNING status change on a Job for N milliseconds
	// This is not compatible with the wait_for_ok option above.
	// This is designed to ask Maestro to wait for a processes to fully start
	// before trying to start another Job who's dependcies are waiting
	// on this process
	DeferRunningStatus uint32 `yaml:"defer_running_status" json:"defer_running_status"`
	// if a Job is defined it will automatically start. Set this to
	// true to prevent a Job from starting automatically
	NoAutoStart             bool     `yaml:"no_autostart" json:"no_autostart"`
	Restart                 bool     `yaml:"restart" json:"restart"`
	RestartOnDependencyFail bool     `yaml:"restart_on_dependency_fail" json:"restart_on_dependency_fail"`
	RestartLimit            uint32   `yaml:"restart_limit" json:"restart_limit"`
	RestartPause            uint32   `yaml:"restart_pause" json:"restart_pause"`
	DependsOn               []string `yaml:"depends_on" json:"depends_on"`
	ExecCmd                 string   `yaml:"exec_cmd" json:"exec_cmd"`
	ExecArgs                []string `yaml:"exec_args" json:"exec_args"`
	Pgid                    int      `yaml:"pgid" json:"pgid"`
	Env                     []string `yaml:"env" json:"env"`
	InheritEnv              bool     `yaml:"inherit_env" json:"inherit_env"`
	DieOnParentDeath        bool     `yaml:"die_on_parent_death" json:"die_on_parent_death"`
	Daemonize               bool     `yaml:"daemonize" json:"daemonize"` // create a new SID for this process ?
	// true if the definition can not be modified
	// Note: In the config file, this should be set to true, if you wish Maestro to
	// not save settings in the config file to the maestro database.
	Immutable bool `yaml:"immutable" json:"immutable"`
}

type StatsConfigPayload struct {
	Interval     uint32 `yaml:"interval" json:"interval"`
	CheckMem     bool   `yaml:"check_mem" json:"check_mem"`
	CheckMemPace uint32 `yaml:"check_mem_pace" json:"check_mem_pace"`
}

func (this *ContainerTemplatePayload) GetName() string {
	return this.Name
}

func (this *ContainerTemplatePayload) GetCGroupSetup() CGroupSetup {
	return this.CGroup
}
func (this *ContainerTemplatePayload) GetEnv() []string {
	return this.AddEnv
}
func (this *ContainerTemplatePayload) GetDependsOn() []string {
	return this.DependsOn
}

func (this *ContainerTemplatePayload) GetExecCmd() string {
	return this.ExecCmd
}
func (this *ContainerTemplatePayload) GetPreArgs() []string {
	return this.ExecPreArgs
}
func (this *ContainerTemplatePayload) GetPostArgs() []string {
	return this.ExecPostArgs
}
func (this *ContainerTemplatePayload) IsInheritEnv() bool {
	return this.InheritEnv
}
func (this *ContainerTemplatePayload) UsesOkString() bool {
	return this.OkString
}
func (this *ContainerTemplatePayload) GetDeferRunningStatus() uint32 {
	return this.DeferRunningStatus
}

func (this *ContainerTemplatePayload) IsDieOnParentDeath() bool {
	return this.DieOnParentDeath
}
func (this *ContainerTemplatePayload) IsSendCompositeJobsToStdin() bool {
	return this.SendCompositeJobsToStdin
}
func (this *ContainerTemplatePayload) IsSendGreaseOriginIdEnv() bool {
	return this.SendGreaseOriginIdEnv
}
func (this *ContainerTemplatePayload) GetCompositeConfig() string {
	return this.CompositeConfig
}
func (this *ContainerTemplatePayload) IsMutable() bool {
	return !this.Immutable
}

func (this *CGroupSetupConfig) GetMemLimit() uint64 {
	return this.MemLimit
}

func (this *JobDefinitionPayload) GetJobName() string {
	return this.Job
}
func (this *JobDefinitionPayload) GetCompositeId() string {
	return this.CompositeId
}
func (this *JobDefinitionPayload) GetContainerTemplate() string {
	return this.ContainerTemplate
}
func (this *JobDefinitionPayload) GetMessageForProcess() string {
	return this.Message
}
func (this *JobDefinitionPayload) GetConfigName() string {
	if len(this.ConfigName) > 0 {
		return this.ConfigName
	} else {
		return DEFAULT_CONFIG_NAME
	}
}
func (this *JobDefinitionPayload) GetStaticConfig() string {
	return this.Config
}
func (this *JobDefinitionPayload) GetExecCmd() string {
	return this.ExecCmd
}
func (this *JobDefinitionPayload) GetArgs() []string {
	return this.ExecArgs
}
func (this *JobDefinitionPayload) GetEnv() []string {
	return this.Env
}
func (this *JobDefinitionPayload) IsInheritEnv() bool {
	return this.InheritEnv
}
func (this *JobDefinitionPayload) IsDaemonize() bool {
	return this.Daemonize
}
func (this *JobDefinitionPayload) UsesOkString() bool {
	return this.WaitForOk
}
func (this *JobDefinitionPayload) GetDeferRunningStatus() uint32 {
	return this.DeferRunningStatus
}
func (this *JobDefinitionPayload) IsDieOnParentDeath() bool {
	return this.DieOnParentDeath
}
func (this *JobDefinitionPayload) GetPgid() int {
	return this.Pgid
}

func (this *JobDefinitionPayload) GetDependsOn() []string {
	return this.DependsOn
}
func (this *JobDefinitionPayload) IsAutostart() bool {
	return !this.NoAutoStart
}
func (this *JobDefinitionPayload) GetRestartOnDependencyFail() bool {
	return this.RestartOnDependencyFail
}
func (this *JobDefinitionPayload) IsRestart() bool {
	return this.Restart
}
func (this *JobDefinitionPayload) GetRestartLimit() uint32 {
	return this.RestartLimit
}
func (this *JobDefinitionPayload) GetRestartPause() uint32 {
	return this.RestartPause
}
func (this *JobDefinitionPayload) IsMutable() bool {
	return !this.Immutable
}

// Interface defintion here is defined in processes/processMgmt.go:ProcessStatsConfig:
func (stats StatsConfigPayload) GetInterval() (ret uint32) {
	if stats.Interval == 0 {
		ret = DEFAULT_STATS_INTERVAL
	} else {
		ret = stats.Interval
	}
	return
}

func (stats StatsConfigPayload) GetConfig_CheckMem() (pace uint32, doit bool) {
	if !stats.CheckMem {
		doit = false
		pace = 0
	} else {
		doit = true
		if stats.CheckMemPace < 1 {
			pace = 1
		} else {
			pace = stats.CheckMemPace
		}
	}
	return
}

type BatchOpPayload struct {
	TaskId string            `json:"task_id"`
	Ops    []json.RawMessage `json:"ops"` // an array of Operation Payload JSON elements
}

// retrives all Operations as a single array of Operations
// Each Operation can be cast to it's appropriate Operation type
// Note: this can be an expensive op
func (this *BatchOpPayload) GetOps() (ret []Operation, err error) {
	ret = make([]Operation, 0, len(this.Ops))

	var op genericOpPayload
	for _, opval := range this.Ops {
		err = json.Unmarshal([]byte(opval), &op)

		switch op.Type {
		case OP_TYPE_JOB:
			opJob := new(JobOpPayload)
			err = json.Unmarshal([]byte(opval), opJob)
			if err != nil {
				return
			} else {
				ret = append(ret, opJob)
			}

		case OP_TYPE_IMAGE:
			opImg := new(ImageOpPayload)
			err = json.Unmarshal([]byte(opval), opImg)
			if err != nil {
				return
			} else {
				ret = append(ret, opImg)
			}
		default:
			apierr := new(APIError)
			apierr.HttpStatusCode = 406
			apierr.ErrorString = "Unknown Op Type in batch request"
			apierr.Detail = "Op was \"" + op.Type + "\""
			err = apierr
			return
		}

	}
	return
}

func (this *BatchOpPayload) GetTaskId() (id string) {
	return this.TaskId
}

type BatchOperation interface {
	GetOps() (ret []Operation, err error)
	GetTaskId() string
}
