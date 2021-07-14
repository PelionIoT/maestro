package platforms

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
	"github.com/PelionIoT/maestro/platforms/fsonly"
	"github.com/PelionIoT/maestro/platforms/testplatform"
	"github.com/PelionIoT/maestroSpecs"
)

type builtin struct {
	reader maestroSpecs.PlatformReader
}

var builtinsByPlatformName map[string]builtin

func init() {
	// load up all the builtin platforms so we can access them by string
	builtinsByPlatformName = make(map[string]builtin)

	builtinsByPlatformName["testplatform"] = builtin{
		reader: testplatform.PlatformReader,
	}
	builtinsByPlatformName["fsonly"] = builtin{
		reader: fsonly.PlatformReader,
	}

}
