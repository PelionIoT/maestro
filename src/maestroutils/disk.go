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

import (
	"syscall"
)

type DiskStats struct {
	All   uint64
	Free  uint64
	Used  uint64
	Avail uint64 // this is the amount available to the program / caller

	// Type    int64
	// Bsize   int64
	// Blocks  uint64
	// Bfree   uint64
	// Bavail  uint64
	// Files   uint64
	// Ffree   uint64
	// Fsid    Fsid
	// Namelen int64
	// Frsize  int64
	// Flags   int64
	// Spare   [4]int64
}

func DiskUsageAtPath(path string) (disk DiskStats) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return
	}
	disk.All = fs.Blocks * uint64(fs.Bsize)
	disk.Avail = fs.Bavail * uint64(fs.Bsize)
	disk.Free = fs.Bfree * uint64(fs.Bsize)
	disk.Used = disk.All - disk.Free
	return
}
