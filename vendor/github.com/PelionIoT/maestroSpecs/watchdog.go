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

import "time"

// Watchdog plugins should be written as a Go plugin
// Go plugins are essentially ELF shared objects in linux, very much like
// a standard .so file but Go style
// See: https://medium.com/learning-the-go-programming-language/writing-modular-go-programs-with-plugins-ec46381ee1a9

// WatchdogConfig is the config structure used to configure any watchdog plugin
// It will be passed to the watchdog Setup() procedure
type WatchdogConfig struct {
	// Path is used by maestro and the config file, it is not for the plugin.
	// This the location of the ELF plugin object itself
	Path string `yaml:"path" json:"path"`
	// Disable, if true, then the watchdog plugin will load and immediately after Setup()
	// Disable() will be called
	Disable bool `yaml:"disable" json:"disable"`
	// Opt1 is a generic implementor optional parameter
	Opt1 string `yaml:"opt1" json:"opt1"`
	// Opt2 is a generic implementor optional parameter
	Opt2 string `yaml:"opt2" json:"opt2"`
}

// WatchdogLogger is passed in to the watchdog as a way for it to log
// output / problems.
type WatchdogLogger interface {
	Errorf(format string, a ...interface{})
	Debugf(format string, a ...interface{})
	Infof(format string, a ...interface{})
}

// Watchdog plugins must meet this inteface spec
// All calls are blocking. The implementor should not block indefinitely
type Watchdog interface {
	// Called by Maestro upon load. If the watchdog needs a setup procedure this
	// should occur here. An error returned will prevent further calls from Maestro
	// including KeepAlive()
	Setup(config *WatchdogConfig, logger WatchdogLogger) error
	// CriticalInterval returns the time interval the watchdog *must* be called
	// in order to keep the system alive. The implementer should build in enough buffer
	// to this for the plugin to do its work of keep any hw / sw watchdog alive.
	// Maestro will call KeepAlive() at this interval or quicker, but never slower.
	CriticalInterval() time.Duration
	// KeepAlive is called by Maestro to keep the watch dog up. A call of KeepAlive
	// means the watchdog plug can safely assume all is normal on the system
	KeepAlive() error
	// NotOk is called by Maestro at the same interval as KeepAlive, but in leiu of it
	// when the system is not meeting the criteria to keep the watch dog up
	NotOk() error
	// Disabled is called when Maestro desires, typically from a command it recieved, to
	// disabled the watchdog. Usually for debugging. Implementors should disable the watchdog
	// or if this is impossible, they can return an error.
	Disable() error
	// Enable() is not called normally. But if Disable() is called, the a call of Enable()
	// should renable the watchdog if it was disabled.
	Enable() error
}
