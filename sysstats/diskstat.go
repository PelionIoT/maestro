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
	"regexp"

	"github.com/PelionIoT/gopsutil/disk"
	"github.com/PelionIoT/maestro/log"
)

type DiskConfig struct {
	statConfig `yaml:",inline"`
	// IgnorePartitions - any named partition in this list will be ignored.
	// Example would be: "/tmpfs"
	IgnorePartitions []string `yaml:"ignore_partitions"`
	// IgnorePartitionsRegex - any partition name matching this regex will be ignored
	// Example would be: "\/snap\/.*" would ignore all /snap related paths
	IgnorePartitionsRegex []string `yaml:"ignore_partitions_regex"`

	// WhiteListPartitions while not empty will be the only partitions
	// the disk stat will gather statistics on.
	WhiteListPartitions []string `yaml:"white_list_partitions"`
	// IgnoreMountpoints - any named partition in this list will be ignored.
	// Example would be: "/var/lib/docker/aufs"
	IgnoreMountpoints []string `yaml:"ignore_mountpoints"`
	// IgnoreMountpointsRegex - any mountpoint name matching this regex will be ignored
	// Example would be: "/var/lib/docker\/.*" would ignore /var/lib/docker/aufs
	IgnoreMountpointsRegex []string `yaml:"ignore_mountpoints_regex"`
	// WhiteListPartitions while not empty will be the only partitions
	// the disk stat will gather statistics on.
	// Example "/"
	WhiteListMountpoints []string `yaml:"white_list_mountpoints"`

	// internal representations:
	whiteListMounts        bool
	whiteListParts         bool
	whiteListedParts       map[string]bool
	blackListedParts       map[string]bool
	blackListedPartsRegex  []*regexp.Regexp
	whiteListedMounts      map[string]bool
	blackListedMounts      map[string]bool
	blackListedMountsRegex []*regexp.Regexp
}

func (stat *DiskConfig) RunStat(ctx context.Context, configChanged bool) (ret *StatPayload, err error) {
	// var stat *mem.VirtualMemoryStat
	// stat, err = mem.VirtualMemory()
	// ret = stats
	if configChanged {
		stat.whiteListedParts = make(map[string]bool)
		stat.blackListedParts = make(map[string]bool)
		stat.whiteListedMounts = make(map[string]bool)
		stat.blackListedMounts = make(map[string]bool)
		// do initial setup of the internal values
		if len(stat.WhiteListMountpoints) > 0 {
			stat.whiteListMounts = true
			for _, item := range stat.WhiteListMountpoints {
				stat.whiteListedMounts[item] = true
			}
		} else {
			stat.whiteListMounts = false
		}

		if len(stat.WhiteListPartitions) > 0 {
			stat.whiteListParts = true
			for _, item := range stat.WhiteListPartitions {
				stat.whiteListedParts[item] = true
			}
		} else {
			stat.whiteListParts = false
		}

		for _, item := range stat.IgnoreMountpoints {
			stat.blackListedMounts[item] = true
		}
		for _, item := range stat.IgnorePartitions {
			stat.blackListedParts[item] = true
		}

		stat.blackListedMountsRegex = make([]*regexp.Regexp, 0, 5)
		stat.blackListedPartsRegex = make([]*regexp.Regexp, 0, 5)

		for _, item := range stat.IgnoreMountpointsRegex {
			compiled, err2 := regexp.Compile(item)
			if err2 != nil {
				log.MaestroErrorf("Bad regex (will ignore) in stat %s - IgnoreMountpointsRegex: %s\n", stat.Name, err2.Error())
			} else {
				stat.blackListedMountsRegex = append(stat.blackListedMountsRegex, compiled)
			}
		}
		for _, item := range stat.IgnorePartitionsRegex {
			compiled, err2 := regexp.Compile(item)
			if err2 != nil {
				log.MaestroErrorf("Bad regex (will ignore) in stat %s - IgnorePartitionsRegex: %s\n", stat.Name, err2.Error())
			} else {
				stat.blackListedPartsRegex = append(stat.blackListedPartsRegex, compiled)
			}
		}
	}

	// find all partitions...

	var partsraw, parts, final []disk.PartitionStat

	partsraw, err = disk.PartitionsWithContext(ctx, false)

	if err != nil {
		return
	}

	// first - filter out only partitions we want
	if stat.whiteListParts {
		// white list only?
		for _, v := range partsraw {
			_, exist := stat.whiteListedParts[v.Device]
			if exist {
				parts = append(parts, v)
			}
		}
	} else {
		// remove any black listed partitions
	nextPart:
		for _, v := range partsraw {
			_, exist := stat.blackListedParts[v.Device]
			if !exist {
				// only consider non-blacklisted parts
				for _, regex := range stat.blackListedPartsRegex {
					if regex.Match([]byte(v.Device)) {
						// skip any black listed regex matches
						continue nextPart
					}
				}
				parts = append(parts, v)
			}
		}
	}

	// now that we have a list of valid partions...
	if stat.whiteListMounts {
		for _, v := range parts {
			_, exist := stat.whiteListedMounts[v.Mountpoint]
			if exist {
				final = append(final, v)
			}
		}
	} else {
	nextMountpoint:
		for _, v := range parts {
			_, exist := stat.blackListedMounts[v.Mountpoint]
			if !exist {
				// only consider non-blacklisted parts
				for _, regex := range stat.blackListedMountsRegex {
					if regex.Match([]byte(v.Mountpoint)) {
						// skip any black listed regex matches
						continue nextMountpoint
					}
				}
				final = append(final, v)
			}
		}
	}

	ok := true

	// map by mountpoint name
	usagedata := make(map[string]*disk.UsageStat)
	var dat *disk.UsageStat

	for _, v := range final {
		ok, err = continueCheckContext(ctx)
		if !ok {
			return
		}

		dat, err = disk.UsageWithContext(ctx, v.Mountpoint)
		if err == nil {
			usagedata[v.Mountpoint] = dat
		}
	}

	ret = &StatPayload{
		Name:     stat.Name,
		StatData: usagedata,
	}

	return
}

func (stat *DiskConfig) GetPace() int64 {
	return stat.statConfig.pace
}
