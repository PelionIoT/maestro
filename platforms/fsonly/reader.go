package fsonly

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
	"errors"

	"github.com/armPelionEdge/maestro/platforms/common"
	"github.com/armPelionEdge/maestroSpecs"
	"github.com/armPelionEdge/maestroSpecs/templates"
)

// The template platform reader for a gateway which has no secure storage or hardware encryption chip / TrustZone.
// This platform reader just gets information by reading an identity.json file off the file system.

// "encoding/hex"

// IDENTITY_JSON_DEFAULT_PATH is the default location of identity.json file, which comes out of
// provision tools. /config is used on soft gateway
const defaultIdentityJSONPath = "/config/identity.json"

var identityJSONPath = defaultIdentityJSONPath

type platformInstance struct {
}

func SetOptsPlatform(opts map[string]interface{}) (err error) {
	v, ok := opts["identityPath"]
	if ok {
		s, ok2 := v.(string)
		if ok2 {
			if len(s) > 0 {
				identityJSONPath = s
			} else {
				err = errors.New("identityPath must be a string of length >= 1")
			}
		} else {
			err = errors.New("identityPath is wrong type")
		}
	}
	return
}

// PlatformReader is a required export for a platform module
var PlatformReader maestroSpecs.PlatformReader

func GetPlatformVars(dict *templates.TemplateVarDictionary, log maestroSpecs.Logger) (err error) {
	if len(identityJSONPath) > 0 {
		_, err = common.ReadIdentityFile(identityJSONPath, dict, log)
		if err != nil {
			err = errors.New("Failed to read identity file")
		}
	} else {
		err = errors.New("No path for identity file")
	}
	return
}

func (reader *platformInstance) GetPlatformVars(dict *templates.TemplateVarDictionary, log maestroSpecs.Logger) (err error) {
	err = GetPlatformVars(dict, log)
	return
}

func (reader *platformInstance) SetOptsPlatform(opts map[string]interface{}) (err error) {
	err = SetOptsPlatform(opts)
	return

}

func init() {
	inst := new(platformInstance)
	PlatformReader = inst
}
