package watchdog

// Copyright (c) 2018, Arm Limited and affiliates.
// Copyright (c) 2023 Izuma Networks
//
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
	"plugin"
	"sync"
	"time"

	"github.com/PelionIoT/maestro/debugging"
	"github.com/PelionIoT/maestro/log"
	maestroPlugins "github.com/PelionIoT/maestro/plugins"
	"github.com/PelionIoT/maestroSpecs"
)

const (
	nowatchdog = 0
	dogok      = 1
	dognotok   = 2
	ctlOk      = 1
	ctlWakeup  = 2
	ctlStop    = 3
)

// Watchdog plugins should be written as a Go plugin
// Go plugins are essentially ELF shared objects in linux, very much like
// a standard .so file but Go style
// See: https://medium.com/learning-the-go-programming-language/writing-modular-go-programs-with-plugins-ec46381ee1a9

// This subsystem loads and runs all watchdog plugins

// only one watchdog can be run at a time
var activeWD maestroSpecs.Watchdog
var pluginWD *plugin.Plugin

var interval time.Duration
var ctlChan chan int

var reason string
var status int
var running bool
var runMutex sync.Mutex

type Plug struct {
	Path    string
	_       chan struct{}
	Symbols map[string]interface{}
}

// LoadWatchdog loads the specified watchdog plugin. Only
// one watchdog plugin can be run at any given time. There is no way to
// unload a watchdog. Once loaded the watchdog is active.
func LoadWatchdog(conf *maestroSpecs.WatchdogConfig) (err error) {
	if len(conf.Path) > 0 {
		pluginWD, err = plugin.Open(conf.Path)
		if err != nil {
			log.MaestroErrorf("Failed to load plugin: %s - %s", conf.Path, err.Error())
			return
		}

		maestroPlugins.InspectPlugin(pluginWD)

		var initSym plugin.Symbol
		var wdSym plugin.Symbol
		initSym, err = pluginWD.Lookup("InitMaestroPlugin")
		if err != nil {
			log.MaestroErrorf("Does not look like a maestro plugin - can't find InitMaestroPlug(): %s - %s", conf.Path, err.Error())
			return
		}
		initFunc, ok := initSym.(func() error)
		if ok {
			// initialize the plugin
			err = initFunc()
			if err == nil {
				// ok - now get the watchdog

				//		initSym, err = pluginWD.Lookup("InitMaestroPlugin")

				wdSym, err = pluginWD.Lookup("Watchdog")
				if err == nil {
					var activeWDp *maestroSpecs.Watchdog
					// activeWDp, ok = wdSym.(*maestroSpecs.Watchdog)
					activeWDp, ok = wdSym.(*maestroSpecs.Watchdog)
					if ok {
						activeWD = *activeWDp
						log.MaestroSuccessf("Watchdog plugin %s - load successfully.", conf.Path)
						logger := maestroPlugins.NewPluginLogger(conf.Path)
						err = activeWD.Setup(conf, logger)
						if err == nil {
							if conf.Disable {
								err = Disable()
								if err == nil {
									log.MaestroSuccessf("Watchdog plugin %s - disabled on start.", conf.Path)
								} else {
									log.MaestroErrorf("Watchdog plugin %s - failed to disable on start -> %s", conf.Path, err.Error())
								}
							} else {
								err = Enable()
								if err == nil {
									Start()
									log.MaestroSuccessf("Watchdog plugin %s - running.", conf.Path)
								} else {
									log.MaestroErrorf("Watchdog plugin %s - failed to enable -> %s", conf.Path, err.Error())
								}
							}
						} else {
							log.MaestroErrorf("Watchdog plugin %s - failed to start -> %s", conf.Path, err.Error())
						}
					} else {
						log.MaestroErrorf("Watchdog plugin %s - failed to cast 'Watchdog' symbol. Failing", conf.Path)
						err = errors.New("bad watchdog plugin")
					}
				} else {
					log.MaestroErrorf("Watchdog plugin %s - has no 'Watchdog' symbol. Failing", conf.Path)
				}
			} else {
				log.MaestroErrorf("Watchdog Plugin %s failed to init!", conf.Path)
			}
		} else {
			log.MaestroErrorf("Watchdog plugin %s - failed to cast 'InitMaestroPlugin' symbol. Failing", conf.Path)
			err = errors.New("bad plugin")
		}
	} else {
		pluginWD = nil
		err = errors.New("bad path")
	}
	return
}

// runs at an interval good enough to keep the watchdog up
func intervalRunner() {
	runMutex.Lock()
	if running {
		runMutex.Unlock()
		return
	}
	running = true
	runMutex.Unlock()
	if activeWD != nil {
		interval = activeWD.CriticalInterval()
	mainLoop:
		for {
			debugging.DEBUG_OUT("Watchdog: intervalRunner() top (interval %s)\n", interval)
			if activeWD == nil {
				break
			}
			select {
			case c := <-ctlChan:
				if c == ctlStop {
					err := activeWD.Disable()
					if err != nil {
						log.MaestroErrorf("watchdog: Disable failed: %s", err.Error())
					} else {
						log.MaestroDebugf("watchdog: Disable() called")
					}
					break
				}
				if c == ctlWakeup {
					continue
				}
				if c == ctlOk {
					err := activeWD.KeepAlive()
					if err != nil {
						log.MaestroErrorf("watchdog: KeepAlive failed: %s", err.Error())
						// need backoff
						interval = time.Second * 3
					} else {
						log.MaestroDebugf("watchdog: KeepAlive() called")
						interval = activeWD.CriticalInterval()
					}
				}
			case <-time.After(interval):
				switch status {
				case dogok:
					err := activeWD.KeepAlive()
					if err != nil {
						log.MaestroErrorf("watchdog: KeepAlive failed: %s", err.Error())
						// need backoff
						interval = time.Second * 3
					} else {
						log.MaestroDebugf("watchdog: KeepAlive() called")
						interval = activeWD.CriticalInterval()
					}
				case dognotok:
					err := activeWD.NotOk()
					if err != nil {
						log.MaestroErrorf("watchdog: NotOK (%s) failed: %s", reason, err.Error())
					} else {
						log.MaestroDebugf("watchdog: NotOK() called")
						interval = activeWD.CriticalInterval()
					}
				case nowatchdog:
					log.MaestroErrorf("watchdog: no active watchdog available.")
					break mainLoop
				}
			}
		}
	}
	runMutex.Lock()
	running = false
	runMutex.Unlock()
	return

}

// CriticalStop is called if the watchdog not be kept alive. A reason must be
// provided. Such reason will be logged.
func CriticalStop(reason string) {
	status = dognotok
}

// Ok tells the watchdog everything is ok. Should be called if a problem quickly resolved itself after CriticalStop()
func Ok() {
	ctlChan <- ctlOk
}

// Stop - stops the watchdog subsystem and any plugin
func Stop() {
	ctlChan <- ctlStop
}

// Disable will disable the watchdog. When disabled, the watchdog should not
// cause the system to reboot/rest. Some watchdog plugins may not be able to
// disable. If so, then an error is reported. If an error is returned, it should
// be assumed that the watchdog is still running.
func Disable() (err error) {
	if activeWD != nil {
		err = activeWD.Disable()
	}
	return
}

// Start will start the intervalRunner() inernal go routine which will ping the active watchdog
// periodically based on CriticalInterval()
func Start() {
	status = dogok
	go intervalRunner()
}

// Enable will re-enable the previously Disable()ed watchdog.
func Enable() (err error) {
	if activeWD != nil {
		err = activeWD.Enable()
	}
	return
}
