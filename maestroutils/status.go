package maestroutils

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

// so 6f97294 is filled in by the build script
// TODO include third party licenses
const (
	versionString = `
maestro 0.1.0
commit 6f97294
build date: Sun May 19 15:18:59 CDT 2019
(c) 2018. WigWag Inc. 
`
)

// var started time.Time

func init() {
	// started = time.Now()
}

// Version returns a string representing the version of maestro we are on
// along with some extra info
func Version() (ret string) {
	// uptime := time.Since(started)
	// ret = fmt.Sprintf(versionString, started.Format("Mon Jan 2 15:04:05 MST (-0700) 2006"), uptime.String())
	ret = versionString
	return
}
