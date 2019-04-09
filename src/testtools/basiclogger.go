package testtools

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

import "fmt"

// NonProductionPrefixedLogger is for non-production logging / tests
type NonProductionPrefixedLogger struct {
	prefix          string
	prefixinterface interface{}
}

func (this *NonProductionPrefixedLogger) Info(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[INFO] %s %s", this.prefix, s)
}

func (this *NonProductionPrefixedLogger) Infof(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[INFO] %s %s\n", this.prefix, s)
}

func (this *NonProductionPrefixedLogger) Success(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[OK]   %s %s", this.prefix, s)
}

func (this *NonProductionPrefixedLogger) Successf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[OK]   %s %s\n", this.prefix, s)
}

func (this *NonProductionPrefixedLogger) Warn(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[WARN] %s %s", this.prefix, s)
}

func (this *NonProductionPrefixedLogger) Warnf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[WARN] %s %s\n", this.prefix, s)
}

func (this *NonProductionPrefixedLogger) Error(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[ERROR] %s %s", this.prefix, s)
}

func (this *NonProductionPrefixedLogger) Errorf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[ERROR] %s %s\n", this.prefix, s)
}

func (this *NonProductionPrefixedLogger) Debug(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[debug] %s %s", this.prefix, s)
}

func (this *NonProductionPrefixedLogger) Debugf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[debug] %s %s\n", this.prefix, s)
}

func NewNonProductionPrefixedLogger(s string) (ret *NonProductionPrefixedLogger) {
	ret = new(NonProductionPrefixedLogger)
	ret.prefix = fmt.Sprintf("[%s] ", s)
	ret.prefixinterface = interface{}(ret.prefix)
	return
}

func UpdateNonProductionPrefixedLogger(l *NonProductionPrefixedLogger, s string) {
	l.prefix = s
	l.prefixinterface = interface{}(l.prefix)
}
