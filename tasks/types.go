package tasks

// Copyright (c) 2018, Arm Limited and affiliates.
// Copyright (c) 2023 Izuma Networks
//
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
	"sync"
)

var tasks sync.Map    // threadsafe hashmap string (taskid): *MaestroTask
var handlers sync.Map // maps Op.Type : TaskHandler
var metaTasks sync.Map

func setupHashmaps() {
}
