package maestroSpecs

import (
	"github.com/armPelionEdge/maestro/greasego"
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
	Targets []*LogTarget `yaml:"targets" json:"targets" log_group:"targets"`
}

type LogRotate struct {
	MaxFiles      uint32 `yaml:"max_files,omitempty" json:"max_files" greaseAssign:"Max_files"`
	RotateOnStart bool   `yaml:"rotate_on_start,omitempty" json:"rotate_on_start" greaseAssign:"Rotate_on_start"`
	MaxFileSize   uint32 `yaml:"max_file_size,omitempty" json:"max_file_size" greaseAssign:"Max_file_size"`
	MaxTotalSize  uint64 `yaml:"max_total_size,omitempty" json:"max_total_size" greaseAssign:"Max_total_size"`
}

type LogTarget struct {
	File                     string                           `yaml:"file,omitempty" json:"file" greaseAssign:"File" log_group:"name"`
	TTY                      string                           `yaml:"tty,omitempty" json:"tty" greaseAssign:"TTY" log_group:"name"`
	Format                   LogFormat                        `yaml:"format,omitempty" json:"format" log_group:"format"`
	Filters                  []LogFilter                      `yaml:"filters" json:"filters" log_group:"filters"`
	Rotate                   LogRotate                        `yaml:"rotate,omitempty" json:"rotate" greaseAssign:"FileOpts" log_group:"opts"`
	ExampleFileOpts          greasego.GreaseLibTargetFileOpts `greaseType:"FileOpts" json:"example_file_opts" log_group:"opts"`
	Delim                    string                           `yaml:"delim,omitempty" json:"delim" greaseAssign:"Delim" log_group:"format"`
	FormatPre                string                           `yaml:"format_pre,omitempty" json:"format_pre" greaseAssign:"Format_pre" log_group:"format"`
	FormatTime               string                           `yaml:"format_time,omitempty" json:"format_time" greaseAssign:"Format_time" log_group:"format"`
	FormatLevel              string                           `yaml:"format_level,omitempty" json:"format_level" greaseAssign:"Format_level" log_group:"format"`
	FormatTag                string                           `yaml:"format_tag,omitempty" json:"format_tag" greaseAssign:"Format_tag" log_group:"format"`
	FormatOrigin             string                           `yaml:"format_origin,omitempty" json:"format_origin" greaseAssign:"Format_origin" log_group:"format"`
	FormatPost               string                           `yaml:"format_post,omitempty" json:"format_post" greaseAssign:"Format_post" log_group:"format"`
	FormatPreMsg             string                           `yaml:"format_pre_msg,omitempty" json:"format_pre_msg" greaseAssign:"Format_pre_msg" log_group:"format"`
	Name                     string                           `yaml:"name,omitempty" json:"name" greaseAssign:"Name" log_group:"name"`
	Flag_json_escape_strings bool                             `yaml:"flag_json_escape_strings" json:"flag_json_escape_strings" log_group:"opts"`
	Existing                 string                           `yaml:"existing" json:"existing" json:"delim" log_group:"opts"`
}

type LogFormat struct {
	Time   string `yaml:"time,omitempty" json:"time" log_group:"format"`
	Level  string `yaml:"level,omitempty" json:"level" log_group:"format"`
	Tag    string `yaml:"tag,omitempty" json:"tag" log_group:"format"`
	Origin string `yaml:"origin,omitempty" json:"origin" log_group:"format"`
}

type LogFilter struct {
	Target string `yaml:"target,omitempty" json:"target" log_group:"filters"` // target: "default",
	Levels string `yaml:"levels,omitempty" json:"levels" log_group:"filters"`
	Tag    string `yaml:"tag,omitempty" json:"tag" log_group:"filters"`

	Pre           string `yaml:"format_pre,omitempty" json:"pre" greaseAssign:"Format_pre" log_group:"filters"`
	Post          string `yaml:"format_post,omitempty" json:"post" greaseAssign:"Format_post" log_group:"filters"`
	PostFmtPreMsg string `yaml:"fmt_post_pre_msg,omitempty" json:"post_fmt_pre_msg" greaseAssign:"Format_post_pre_msg" log_group:"filters"`
	//	format_pre *string`
	//	format_post *string
	//	format_post_pre_msg *string
}
