package defaults

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

// DefaultUnixLogSocketSink is the default socket used by 'greaselog'
// and the grease-log-client node.js module. This is defined in
// grease_client.h in these projects
const DefaultUnixLogSocketSink = "/tmp/grease.socket"
const DefaultHttpUnixSocket = "/tmp/maestroapi.sock"
const DefaultSyslogSocket = "/dev/log" // standard Unix location

const DefaultConfigDBPath = "{{thisdir}}/maestroConfig.db"

const NUMBER_BANKS_WEBLOG = uint32(20)

const MAX_QUEUED_PROCESS_EVENTS = 100

const DEFAULT_STATS_INTERVAL = 300

const TASK_MANAGER_CLEAR_INTERVAL = 60 * 60 * 4 // 4 hours

const DefaultImagePath = "/var/maestro/images"

const DefaultScratchPath = "/tmp/maestro/images"

const MAX_CONCURRENT_DOWNLOADS = 4

const MIN_DISK_SPACE_SCRATCH_DISK = 5 * 1024 * 1024 // in bytes, 5MB
