package configs

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

type ProcessesConfig struct {
    // specify in milliseconds
    ReaperInterval int `yaml:"reaper_interval"`
}

var ProcessesDefaults *ProcessesConfig

func init() {
    ProcessesDefaults = &ProcessesConfig{ ReaperInterval: 2000 }
}

func (config *ProcessesConfig) GetReaperInterval() (int) {
    if config.ReaperInterval > 0 {
        return config.ReaperInterval
    } else {
        return ProcessesDefaults.ReaperInterval
    }
}

