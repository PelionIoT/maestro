package maestroSpecs;

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

const DefaultUnixLogSocketSink = "/tmp/grease.sock"

const NUMBER_BANKS_WEBLOG = uint32(20)

const MAX_QUEUED_PROCESS_EVENTS = 100

const DEFAULT_STATS_INTERVAL = 300

const MAX_TASK_LIFE = 60*60*24*7 // 1 week in seconds
