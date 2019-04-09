package sysstats

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
	"context"

	"github.com/WigWagCo/gopsutil/mem"
)

type VMConfig struct {
	//	Every string `yaml:"every" json:"every"`
	// see: https://github.com/go-yaml/yaml/pull/94/files/e90bcf783f7abddaa0ee0994a09e536498744e49
	statConfig `yaml:",inline"`
}

func (stat *VMConfig) RunStat(ctx context.Context, configChanged bool) (ret *StatPayload, err error) {
	var thestat *mem.VirtualMemoryStat
	thestat, err = mem.VirtualMemoryWithContext(ctx)
	if err == nil {
		ret = &StatPayload{
			Name:     stat.Name,
			StatData: thestat,
		}
	}
	return
}

func (stat *VMConfig) GetPace() int64 {
	return stat.statConfig.pace
}
