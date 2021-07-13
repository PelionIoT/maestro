package log

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
	"fmt"
	"os"
	"strings"

	"github.com/op/go-logging"
)

var Log = logging.MustGetLogger("maestro")
var log = Log
var loggingBackend logging.LeveledBackend

func init() {
	var format = logging.MustStringFormatter(`%{color}%{time:15:04:05.000} ▶ %{level:.4s} %{shortfile}%{color:reset} %{message}`)
	var backend = logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	loggingBackend = logging.AddModuleLevel(backendFormatter)

	logging.SetBackend(loggingBackend)
}

func SetLoggingLevel(ll string) {
	logLevel, err := logging.LogLevel(strings.ToUpper(ll))

	if err != nil {
		logLevel = logging.ERROR
	}

	loggingBackend.SetLevel(logLevel, "")
}

func MaestroInfo(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[INFO]  %s", s)
}

func MaestroInfof(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[INFO]  %s", s)
}

func MaestroSuccess(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[OK]    %s", s)
}

func MaestroSuccessf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[OK]    %s", s)
}

func MaestroWarn(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[WARN]  %s", s)
}

func MaestroWarnf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[WARN]  %s", s)
}

func MaestroError(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[ERROR] %s", s)
}

func MaestroErrorf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[ERROR] %s", s)
}

func MaestroDebug(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[debug] %s", s)
}

func MaestroDebugf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[debug] %s", s)
}

type PrefixedLogger struct {
	prefix          string
	prefixinterface interface{}
}

func (this *PrefixedLogger) Info(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[INFO] %s %s", this.prefix, s)
}

func (this *PrefixedLogger) Infof(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[INFO] %s %s", this.prefix, s)
}

func (this *PrefixedLogger) Success(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[OK]   %s %s", this.prefix, s)
}

func (this *PrefixedLogger) Successf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[OK]   %s %s", this.prefix, s)
}

func (this *PrefixedLogger) Warn(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[WARN] %s %s", this.prefix, s)
}

func (this *PrefixedLogger) Warnf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[WARN] %s %s", this.prefix, s)
}

func (this *PrefixedLogger) Error(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[ERROR] %s %s", this.prefix, s)
}

func (this *PrefixedLogger) Errorf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[ERROR] %s %s", this.prefix, s)
}

func (this *PrefixedLogger) Debug(a ...interface{}) {
	s := fmt.Sprintln(a...)
	fmt.Printf("[debug] %s %s", this.prefix, s)
}

func (this *PrefixedLogger) Debugf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[debug] %s %s", this.prefix, s)
}

func NewPrefixedLogger(s string) (ret *PrefixedLogger) {
	ret = new(PrefixedLogger)
	ret.prefix = fmt.Sprintf("[%s] ", s)
	ret.prefixinterface = interface{}(ret.prefix)
	return
}

func UpdatePrefixedLogger(l *PrefixedLogger, s string) {
	l.prefix = s
	l.prefixinterface = interface{}(l.prefix)
}
