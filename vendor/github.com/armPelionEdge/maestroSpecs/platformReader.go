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
	"github.com/armPelionEdge/maestroSpecs/templates"
)

// PlatformReader interface spec. Exported platforms should expose an instance of this interface named 'PlatformReader'
type PlatformReader interface {
	// SetOptsPlatform is called upon intitialization of the platform reader
	// an implementation is not required to do anyything, but this can allow the reader
	// to setup correctly
	SetOptsPlatform(map[string]interface{}) (err error)
	GetPlatformVars(dict *templates.TemplateVarDictionary, log Logger) (err error)
}

// PlatformKeyWriter interface spec. Exported platforms should expose an instance of this interface named 'PlatformKeyWriter'
type PlatformKeyWriter interface {
	WritePlatformDeviceKeyNCert(dict *templates.TemplateVarDictionary, key string, cert string, log Logger) (err error)
	GeneratePlatformDeviceKeyNCert(dict *templates.TemplateVarDictionary, deviceid string, accountid string, log Logger) (key string, cert string, err error)
}

// implemented by a library which provides a way to pull off needed
// platform specific variables, typically out of NVRAM, secure flash or EEPROM
// use the Logger to log any errors or information

// GetPlatformVars shold be implemented by any platform reader plugin
type GetPlatformVars func(dict *templates.TemplateVarDictionary, log Logger) (err error)

// GeneratePlatformDeviceKeyNCert generates key and cert for the platform
type GeneratePlatformDeviceKeyNCert func(dict *templates.TemplateVarDictionary, deviceid string, accountid string, log Logger) (key string, cert string, err error)

// WritePlatformDeviceKeyNCert writes a key and cert for the platform
type WritePlatformDeviceKeyNCert func(dict *templates.TemplateVarDictionary, key string, cert string, log Logger) (err error)
