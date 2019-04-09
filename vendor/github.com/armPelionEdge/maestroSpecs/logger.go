package maestroSpecs

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

// A Logger interface is passed to some Maestro plugins, allowing the 
// plugin to Log to Maestro's internal logs, as part of the Maestro
// process.
type Logger interface {
    Info(a ...interface{})
    Infof(format string, a ...interface{})
    Success(a ...interface{})
    Successf(format string, a ...interface{})
    Warn(a ...interface{})
    Warnf(format string, a ...interface{})
    Error(a ...interface{})
    Errorf(format string, a ...interface{})
    Debug(a ...interface{})
    Debugf(format string, a ...interface{})
}

