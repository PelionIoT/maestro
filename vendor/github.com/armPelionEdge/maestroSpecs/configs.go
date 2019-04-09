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

type ConfigFileData interface {
	GetURL() string
	GetChecksum() Checksum
	GetRelativePath() string
	GetSize() uint64 
}

const DEFAULT_CONFIG_NAME = "default"

type ConfigDefinition interface {
	// A unique identifier for this config
	// The name must be unique across all other configs, for this 'Job name'
	GetConfigName() string

	// The Job this Config is associated with
	GetJobName() string
	// For simple and smaller configs, a string may just be provided. This will be 
	// store in the Maestro DB
	GetData() (data string, encoding string)
	// For configs which require their own data files, etc, arbitrary files may be
	// downloaded. 
	GetFiles() []ConfigFileData
	// Last time/date modified, in ISO 8601 format
	// see RFC https://www.ietf.org/rfc/rfc3339.txt
	GetModTime() string

	// Duplicates a new copy of the config in memory
	Dup() ConfigDefinition
}

type ConfigFileDataPayload struct {
	URL string `yaml:"url" json:"url"`
	RelativePath string `yaml:"dest_relative_path" json:"dest_relative_path"`
	Size uint64 `yaml:"size" json:"size"`
	Checksum ChecksumObj `yaml:"checksum" json:"checksum"`	
}

type ConfigDefinitionPayload struct {
	Name string `yaml:"name" json:"name"`   // the identifier for this config 
	Job string `yaml:"job" json:"job"`   // the job this image is associated with
	Data string `yaml:"data" json:"data"`
	// the encoding type. Only 'base64' and 'utf8' are supported.
	// an empty or missing value is assumed to be utf8
	Encoding string `yaml:"encoding" json:"encoding"`
	Files []ConfigFileDataPayload `yaml:"files" json:"files"`
	ModTime string `yaml:"mod_time" json:"mod_time"`
}

func (this *ConfigFileDataPayload) GetURL() string {
	return this.URL
}
func (this *ConfigFileDataPayload) GetRelativePath() string {
	return this.RelativePath
}
func (this *ConfigFileDataPayload) GetChecksum() Checksum {
	return &this.Checksum
}
func (this *ConfigFileDataPayload) GetSize() uint64 {
	return this.Size
}


func (this *ConfigDefinitionPayload) GetConfigName() string {
	return this.Name
}
func (this *ConfigDefinitionPayload) GetJobName() string {
	return this.Job
}
func (this *ConfigDefinitionPayload) GetData() (data string, encoding string) {
	if len(this.Encoding) < 1 {
		encoding = "utf8"
	} else {
		encoding = this.Encoding
	}
	data = this.Data
	return
}
func (this *ConfigDefinitionPayload) GetFiles() []ConfigFileData {
	ret := make([]ConfigFileData, len(this.Files))
	for i := range this.Files {
	    ret[i] = &this.Files[i]
	}
	return ret
}
func (this *ConfigDefinitionPayload) GetModTime() string {
	return this.ModTime
}

func (this *ConfigDefinitionPayload) Dup() ConfigDefinition {
	ret := &ConfigDefinitionPayload {
		Name: this.Name,
		Job: this.Job,
		Data: this.Data,
		Encoding: this.Encoding,
		ModTime: this.ModTime,
	}

	if len(this.Files) > 0 {
		ret.Files = make([]ConfigFileDataPayload,len(this.Files))
		copy(ret.Files,this.Files)
	}
	return ret
}
