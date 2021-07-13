package plugins

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
	"errors"
	"fmt"
	"plugin"
	"sync"
	"time"

	"github.com/PelionIoT/maestro/log"
	"github.com/PelionIoT/maestroSpecs"
)

type periodicPluginConfig struct {
	param    *maestroSpecs.CommonParam
	callees  map[string]*periodicPluginRoutine
	path     string
	callback PeriodicCallback
}

type periodicPluginRoutine struct {
	callee     func(num int, param *maestroSpecs.CommonParam) (contin bool, err error)
	owner      *periodicPluginConfig
	name       string
	num        int
	stop       bool
	intervalms int
	mut        sync.Mutex
}

// PeriodicCallback is a function spec which is used for an optionalk
// user supplied callback which is called ever time a plugin's periodic functions are called
type PeriodicCallback func(param interface{})

func validatePeriodicOpts(opts *maestroSpecs.PluginOpts, path string, p *plugin.Plugin) (conf *periodicPluginConfig, err error) {
	if opts != nil {
		if opts.Periodic != nil {
			for _, opt := range opts.Periodic {
				if len(opt.FunctionName) > 0 {
					var foundSym plugin.Symbol
					foundSym, err = p.Lookup(opt.FunctionName)
					// cast to PeriodicFunc as defined in maestroSpecs
					periodicFunc, ok := foundSym.(func(num int, param *maestroSpecs.CommonParam) (contin bool, err error))
					if ok {
						if conf == nil {
							conf = new(periodicPluginConfig)
							conf.callees = make(map[string]*periodicPluginRoutine)
							conf.param = new(maestroSpecs.CommonParam)
							conf.path = path
						}
						record := new(periodicPluginRoutine)
						record.callee = periodicFunc
						record.owner = conf
						record.intervalms = opt.IntervalMS
						record.name = opt.FunctionName
						conf.callees[opt.FunctionName] = record
					} else {
						err = fmt.Errorf("Symbol %s is not a PeriodicFunc", opt.FunctionName)
					}
				} else {
					err = errors.New("Invalid function name in periodic options")
				}
				if err == nil && opt.IntervalMS < maestroSpecs.MinimalIntervalForPeriodic {
					err = fmt.Errorf("Periodic function %s has bad interval setting", opt.FunctionName)
				}
				if err != nil {
					return
				}
			}
		}
	}
	return
}

func periodicRunner(r *periodicPluginRoutine) {
	for {
		r.mut.Lock()
		if r.stop {
			log.MaestroInfof("Plugin %s - periodic func %s has stopped\n", r.owner.path, r.name)
			break
		}
		r.mut.Unlock()
		log.MaestroDebugf("Plugin %s periodic func %s - calling %d time\n", r.owner.path, r.name, r.num)
		r.owner.param.Mut.Lock()
		c, e := r.callee(r.num, r.owner.param)
		if r.owner.callback != nil {
			r.owner.callback(r.owner.param.Param)
		}
		r.owner.param.Mut.Unlock()
		r.stop = !c
		if e != nil {
			log.MaestroErrorf("Error in plugin %s - periodic func %s - %s\n", r.owner.path, r.name, e.Error())
		}
		r.mut.Lock()
		if r.stop {
			log.MaestroInfof("Plugin %s - periodic func %s has stopped\n", r.owner.path, r.name)
			break
		}
		r.mut.Unlock()
		r.num++
		time.Sleep(time.Duration(r.intervalms) * time.Millisecond)
	}
	r.mut.Unlock()
}

func startPeriodicCalls(conf *periodicPluginConfig) (err error) {
	if conf != nil {
		for _, periodic := range conf.callees {
			go periodicRunner(periodic)
		}
	}
	return
}

func stopPeriodicCall(conf *periodicPluginConfig, routineName string) (err error) {
	if conf != nil {
		r, ok := conf.callees[routineName]
		if ok {
			r.mut.Lock()
			r.stop = true
			r.mut.Unlock()
		} else {
			err = fmt.Errorf("No periodic of name %s", routineName)
		}
	} else {
		err = fmt.Errorf("No periodic functions")
	}
	return
}
