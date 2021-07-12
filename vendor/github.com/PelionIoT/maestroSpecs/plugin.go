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

import "sync"

// API is the container providing access to various maestro APIs
// for plugin's to use
type API struct {
	Stuff interface{}
}

// InitMaestroPluginFunc is the type of InitMaestroPlugin() which is called
// anytime a plugin is loaded by maestro
type InitMaestroPluginFunc func(opts *PluginOpts, api *API, log Logger) error

// MinimalIntervalForPeriodic is the lowest acceptable interval (in milliseconds) for
// a function to periodically execute in a plugin
const MinimalIntervalForPeriodic = 200

// CommonParam is a container for a plugin to hold whatever data they may want
// to be kept / passed between one or more PeriodicFunc routines. Only one periodic
// routing in a plugin will access this CommonParam at any time, and hence no
// periodic routines in a plugin will run in parallel.
// Note: this means any plugin with a periodic function which hangs, will
// hang the entire plugin's periodic functions
type CommonParam struct {
	Param interface{}
	Mut   sync.Mutex
}

// PeriodicFunc is the call spec for any function
// maestro should call periodically. contin should return true
// if the periodic function should continue to be called
// error returns when an error occurs in the PeriodicFunc but
// does not stop the function from continuing to be called if not nil
type PeriodicFunc func(num int, param *CommonParam) (contin bool, err error)

// PluginPeriodic is used to describe plugins which have a symbol
// which should be called periodically
type PluginPeriodic struct {
	// FunctionName is a symbol name of a function with a call spec
	// of PeriodicFunc which is in the plugin
	FunctionName string `yaml:"func" json:"func"`
	// PeriodicInterval is the milliseconds between a PeriodicFunc completing a call,
	// and a new call being started
	IntervalMS int `yaml:"interval_ms" json:"interval_ms"`
}

type PluginOpts struct {
	Periodic []*PluginPeriodic `yaml:"periodic" json:"periodic"`
}
