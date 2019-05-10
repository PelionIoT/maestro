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

	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestroSpecs"
)

type Plug struct {
	Path    string
	_       chan struct{}
	Symbols map[string]interface{}
}

// NewPluginLogger returns a new logger object handed to a plugin
func NewPluginLogger(pluginName string) (logger *log.PrefixedLogger) {
	logger = log.NewPrefixedLogger("plugin: " + pluginName)
	return
}

// loadedPluigins is a map of path:*pluginToken, where each plugin is loaded
// and it's InitMaestroPlugin() has been called
var loadedPlugins *sync.Map

const (
	PLUGIN_LOADING = iota
	PLUGIN_LOADED  = iota
	PLUGIN_FAILED  = iota
)

type pluginToken struct {
	plugin *plugin.Plugin
	// only locked when loading
	cond         *sync.Cond
	lock         sync.Mutex
	loadErr      error
	status       int
	opts         *maestroSpecs.PluginOpts
	periodicConf *periodicPluginConfig
}

func newPluginToken() (ret *pluginToken) {
	ret = new(pluginToken)
	ret.cond = sync.NewCond(&ret.lock)
	return
}

func init() {
	loadedPlugins = &sync.Map{}
}

// GetOrLoadMaestroPlugin loads a plugin. It may block while loading the plugin
// and while waiting for another caller to load the plugin (in which case it will
// unblock for the other caller and this caller). If the plugin is already loaded
// it will return the plugin without blocking.
func GetOrLoadMaestroPlugin(opts *maestroSpecs.PluginOpts, logger maestroSpecs.Logger, path string, callback PeriodicCallback) (plugin *plugin.Plugin, err error) {
	pluginEmpty := newPluginToken()
	pluginEmpty.lock.Lock()
	i, loaded := loadedPlugins.LoadOrStore(path, pluginEmpty)
	var ok bool
	pluginT, ok := i.(*pluginToken)
	if !ok {
		err = errors.New("bad cast")
		return
	}
	if !loaded { // if we have not started this plugin before
		pluginT.cond = sync.NewCond(&pluginT.lock)
		pluginT.status = PLUGIN_LOADING
		pluginT.opts = opts
		pluginT.lock.Unlock()
		plugin, err = loadAndInitPlugin(opts, logger, path, pluginT, callback)
	} else {
		// unlock the not needed empty token
		pluginEmpty.lock.Unlock()
		pluginT.lock.Lock()
		for pluginT.status == PLUGIN_LOADING {
			pluginT.cond.Wait()
		}
		if pluginT.status == PLUGIN_LOADED {
			plugin = pluginT.plugin
		} else {
			err = pluginT.loadErr
		}
		pluginT.lock.Unlock()
	}
	return
}

func loadAndInitPlugin(opts *maestroSpecs.PluginOpts, logger maestroSpecs.Logger, path string, token *pluginToken, callback PeriodicCallback) (ret *plugin.Plugin, err error) {
	if len(path) > 0 {
		ret, err = plugin.Open(path)
		if err != nil {
			log.MaestroErrorf("Failed to load plugin: %s - %s", path, err.Error())
			return
		}

		InspectPlugin(ret)

		var initSym plugin.Symbol

		// TODO - populate API
		initSym, err = ret.Lookup("InitMaestroPlugin")
		if err != nil {
			log.MaestroErrorf("Does not look like a maestro plugin - can't find InitMaestroPlug(): %s - %s", path, err.Error())
			return
		}
		initFunc, ok := initSym.(func(opts *maestroSpecs.PluginOpts, api *maestroSpecs.API, log maestroSpecs.Logger) error)
		if ok {
			// initialize the plugin
			api := new(maestroSpecs.API)
			err = initFunc(opts, api, logger)
			if err == nil {
				log.MaestroSuccessf("plugin %s - Loaded and initialized ok.", path)

				var conf *periodicPluginConfig
				conf, err = validatePeriodicOpts(opts, path, ret)
				if err == nil {
					if conf != nil {
						conf.callback = callback
					}
					token.lock.Lock()
					token.periodicConf = conf
					err = startPeriodicCalls(token.periodicConf)
					token.lock.Unlock()
				}
			} else {
				log.MaestroErrorf("Platform Plugin %s failed to init!", path)
			}
		} else {
			log.MaestroErrorf("Platform plugin %s - failed to cast 'InitMaestroPlugin' symbol. Failing", path)
			err = errors.New("bad plugin")
		}
	} else {
		err = errors.New("bad path")
	}
	if err == nil {
		token.lock.Lock()
		token.plugin = ret
		token.status = PLUGIN_LOADED
		loadedPlugins.Store(path, token)
		token.lock.Unlock()
		token.cond.Broadcast()
	} else {
		token.lock.Lock()
		token.status = PLUGIN_FAILED
		token.loadErr = err
		token.lock.Unlock()
		token.cond.Broadcast()
	}
	return
}

// StopPeriodicOfPlugin - stops periodically calling the periodic function of
// routineName in the plugin already loaded with path
func StopPeriodicOfPlugin(path string, routineName string) (err error) {
	i, ok := loadedPlugins.Load(path)
	if ok {
		var pluginT *pluginToken
		pluginT, ok = i.(*pluginToken)
		if !ok {
			err = errors.New("bad cast")
		} else {
			err = stopPeriodicCall(pluginT.periodicConf, routineName)
		}
	} else {
		err = fmt.Errorf("No plugin loaded of path %s", path)
	}
	return
}
