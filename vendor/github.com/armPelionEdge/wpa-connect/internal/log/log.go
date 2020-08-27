/*
 Copyright (c) 2020 ARM Limited and affiliates.
 Copyright (c) 2016 Mark Berner
 SPDX-License-Identifier: MIT
 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to
 deal in the Software without restriction, including without limitation the
 rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
 sell copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:
 The above copyright notice and this permission notice shall be included in all
 copies or substantial portions of the Software.
 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 SOFTWARE.
*/
package log

import (
	"github.com/op/go-logging"
	"os"
)

var (
	Log = logging.MustGetLogger("")
)

func SetSilentMode() {
	initialize(defaultModeFormatter(), logging.WARNING)
}

func SetInfoMode() {
	initialize(defaultModeFormatter(), logging.INFO)
}

func SetVerboseMode() {
	initialize(defaultModeFormatter(), logging.DEBUG)
}

func SetDebugMode() {
	initialize(debugModeFormatter(), logging.DEBUG)
}

func initialize(formatter logging.Formatter, level logging.Level) {
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, formatter)
	logging.SetBackend(backendFormatter)
	logging.SetLevel(level, "")
}

func init() {
	SetInfoMode()
}

func debugModeFormatter() logging.Formatter {
	return logging.MustStringFormatter(
		`%{color}%{level:.1s} %{module} %{time:15:04:05.000} %{shortfile} %{message}%{color:reset}`,
	)
}

func defaultModeFormatter() logging.Formatter {
	return logging.MustStringFormatter(
		`%{level:.1s} %{module} %{time:15:04:05.000} %{message}`,
	)
}