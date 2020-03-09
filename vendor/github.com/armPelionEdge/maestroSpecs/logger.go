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
type Logger interface {
	Info(a ...interface{})
	Infof(format string, a ...interface{})
	Success(a ...interface{})
	Successf(format string, a ...interface{})
	Warn(a ...interface{})
	Warnf(format string, a ...interface{})
	Error(a ...interface{})
	Errorf(format string, a ...interface{})
	Debug(a ...interface{})
	Debugf(format string, a ...interface{})
}

type LogConfigPayload struct {
	Targets []*LogTarget `yaml:"targets" json:"targets"`
}

type LogRotate struct {
	MaxFiles      uint32 `yaml:"max_files,omitempty" greaseAssign:"Max_files"`
	RotateOnStart bool   `yaml:"rotate_on_start,omitempty" greaseAssign:"Rotate_on_start"`
	MaxFileSize   uint32 `yaml:"max_file_size,omitempty" greaseAssign:"Max_file_size"`
	MaxTotalSize  uint64 `yaml:"max_total_size,omitempty" greaseAssign:"Max_total_size"`
}

type LogTarget struct {
	File                     string                           `yaml:"file,omitempty" greaseAssign:"File" log_group:"name"`
	TTY                      string                           `yaml:"tty,omitempty" greaseAssign:"TTY" log_group:"name"`
	Format                   LogFormat                        `yaml:"format,omitempty" log_group:"format"`
	Filters                  []LogFilter                      `yaml:"filters" log_group:"filters"`
	Rotate                   LogRotate                        `yaml:"rotate,omitempty" greaseAssign:"FileOpts" log_group:"opts"`
	ExampleFileOpts          greasego.GreaseLibTargetFileOpts `greaseType:"FileOpts" log_group:"opts"`
	Delim                    string                           `yaml:"delim,omitempty" greaseAssign:"Delim" log_group:"format"`
	FormatPre                string                           `yaml:"format_pre,omitempty" greaseAssign:"Format_pre" log_group:"format"`
	FormatTime               string                           `yaml:"format_time,omitempty" greaseAssign:"Format_time" log_group:"format"`
	FormatLevel              string                           `yaml:"format_level,omitempty" greaseAssign:"Format_level" log_group:"format"`
	FormatTag                string                           `yaml:"format_tag,omitempty" greaseAssign:"Format_tag" log_group:"format"`
	FormatOrigin             string                           `yaml:"format_origin,omitempty" greaseAssign:"Format_origin" log_group:"format"`
	FormatPost               string                           `yaml:"format_post,omitempty" greaseAssign:"Format_post" log_group:"format"`
	FormatPreMsg             string                           `yaml:"format_pre_msg,omitempty" greaseAssign:"Format_pre_msg" log_group:"format"`
	Name                     string                           `yaml:"name,omitempty" greaseAssign:"Name" log_group:"name"`
	Flag_json_escape_strings bool                             `yaml:"flag_json_escape_strings"`
	Existing                 string                           `yaml:"existing" json:"existing" log_group:"config_opts"`
}

type LogFormat struct {
	Time   string `yaml:"time,omitempty"`
	Level  string `yaml:"level,omitempty"`
	Tag    string `yaml:"tag,omitempty"`
	Origin string `yaml:"origin,omitempty"`
}

type LogFilter struct {
	Target string `yaml:"target,omitempty"` // target: "default",
	Levels string `yaml:"levels,omitempty"`
	Tag    string `yaml:"tag,omitempty"`

	Pre           string `yaml:"format_pre,omitempty" greaseAssign:"Format_pre"`
	Post          string `yaml:"format_post,omitempty" greaseAssign:"Format_post"`
	PostFmtPreMsg string `yaml:"fmt_post_pre_msg,omitempty"  greaseAssign:"Format_post_pre_msg"`
	//	format_pre *string`
	//	format_post *string
	//	format_post_pre_msg *string
}
