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
	"encoding/json"
	"fmt"
	"log"
	"time"
	"testing"

	"github.com/armPelionEdge/gopsutil/disk"
)

func TestMain(m *testing.M) {
	m.Run()
}

// Test to check disk stats from gopsutil package
func TestDiskStatGopsutils(t *testing.T) {
	parts, err := disk.Partitions(false)
	if err != nil {
		log.Fatalf("Partitions() error %s", err.Error())
	} else {
		fmt.Printf("Partitions() = %+v\n", parts)
	}

	for _, part := range parts {
		amap, err2 := disk.IOCounters(part.Device)
		if err2 != nil {
			log.Fatalf("Partitions() error %s", err2.Error())
		} else {
			fmt.Printf("[%s] IOCounters = %+v\n", part.Device, amap)
		}
	}

	usagedata := make(map[string]*disk.UsageStat)
	var dat *disk.UsageStat

	for _, part := range parts {
		if len(part.Mountpoint) > 0 {
			dat, err = disk.Usage(part.Mountpoint)
			if err != nil {
				log.Fatal("Partitions() error: ", err.Error())
			} else {
				usagedata[part.Mountpoint] = dat
				fmt.Printf("[%s] Usage = %+v\n", part.Mountpoint, dat)
			}
		}
	}
}

// Test SysStats as per production file
// sys_stats: # system stats intervals  vm_stats: every: "15s" name: vm  disk_stats: every: "30s" name: disk
func TestProdStats(t *testing.T) {
	mgr := GetManager()
	conf := &StatsConfig{
		DiskStats: &DiskConfig{
			statConfig: statConfig{
				Name:  "disk",
				Every: "30s",
			},
		},
		VMStats: &VMConfig{
			statConfig: statConfig{
				Name:  "vm",
				Every: "15s",
			},
		},
	}

	ok, err := mgr.ReadConfig(conf)
	if ok {
		fmt.Println("sysstats read config ok. Starting")
		mgr.Start()
	} else {
		log.Fatal("mgr.ReadConfig failed: ", err.Error())
	}

	// Terminate loop after timeout, in future use config parameter to stops stats
	timeout := time.After(35*time.Second)
	for {
		select {
		// Got a timeout! fail with a timeout error
		case <-timeout:
			return
		}
	}
}

func TestDiskStatCall(t *testing.T) {
	mgr := GetManager()

	conf := &StatsConfig{
		DiskStats: &DiskConfig{
			statConfig: statConfig{
				Name:  "usage",
				Every: "5s",
			},
			// WhiteListMountpoints: []string{
			// 	"/",
			// 	"/mnt/main2",
			// },
			IgnorePartitionsRegex: []string{
				"\\/dev\\/nvme.*",
			},
			IgnoreMountpointsRegex: []string{
				"\\/snap.*",
			},

			// IgnoreMountpoints: []string{
			// 	"/",
			// 	"/mnt/main2",
			// },
		},
	}

	mgr.ReadConfig(conf)

	stat, err := mgr.OneTimeRun("usage")

	if err != nil {
		log.Fatal("Partitions() error ", err.Error())
	} else {
		fmt.Printf("stat: %+v", stat)
	}

	json, err := json.MarshalIndent(stat, " ", "   ")

	if err != nil {
		log.Fatal("Cloud not turn stat into JSON: ", err.Error())
	}

	fmt.Printf("as JSON:\n%s", string(json))

}

// To run test:
// sub out for your directories
// sudo GOROOT=/opt/go PATH="$PATH:/opt/go/bin" GOPATH=/home/ed/work/gostuff LD_LIBRARY_PATH=../../greasego/deps/lib /opt/go/bin/go test
