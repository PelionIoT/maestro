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
	"testing"

	"github.com/armPelionEdge/gopsutil/disk"
)

func TestMain(m *testing.M) {
	m.Run()
}

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

	for _, part := range parts {
		if len(part.Mountpoint) > 0 {
			stat, err2 := disk.Usage(part.Mountpoint)
			if err2 != nil {
				log.Fatalf("Partitions() error %s", err2.Error())
			} else {
				fmt.Printf("[%s] Usage = %+v\n", part.Mountpoint, stat)
			}
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
