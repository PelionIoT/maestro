package platforms

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
	"plugin"
	"strings"

	"github.com/armPelionEdge/maestroSpecs/templates"

	"github.com/armPelionEdge/maestro/log"
	maestroPlugins "github.com/armPelionEdge/maestro/plugins"
	"github.com/armPelionEdge/maestroSpecs"
)

// ReadWithPlatformReader calls the plugin's GetPlatformVars() function
func SetPlatformReaderOpts(platformOpts map[string]interface{}, path string, opts *maestroSpecs.PluginOpts, logger maestroSpecs.Logger) (err error) {

	if strings.HasPrefix(path, "plugin:") {
		s := strings.Split(path, ":")
		//		err = platforms.ReadWithPlatformReader(maestroConfig.GetGlobalConfigDictionary(), s[1])
		path = s[1]

		var myplugin *plugin.Plugin
		myplugin, err = maestroPlugins.GetOrLoadMaestroPlugin(opts, logger, path, nil)
		if err == nil {
			var setOptsSym plugin.Symbol
			setOptsSym, err = myplugin.Lookup("SetOptsPlatform")
			if err == nil {
				var setOpts func(opts map[string]interface{}) error
				// var readerP *maestroSpecs.PlatformReader
				// activeWDp, ok = wdSym.(*maestroSpecs.Watchdog)
				var ok bool
				setOpts, ok = setOptsSym.(func(opts map[string]interface{}) error)
				if ok {
					log.MaestroSuccessf("Platform plugin %s - cached / load successfully.\n", path)
					// logger := maestroPlugins.NewPluginLogger(path)
					err = setOpts(platformOpts)
				} else {
					log.MaestroErrorf("Platform plugin %s - failed to cast 'SetOptsPlatform' symbol. Failing\n", path)
					err = errors.New("bad platform plugin")
				}
			} else {
				log.MaestroErrorf("Platform plugin %s - has no 'SetOptsPlatform' symbol. Failing\n", path)
				err = errors.New("Not implemented")
			}
		}
	} else {
		p, ok := builtinsByPlatformName[path]
		if ok {
			// logger := maestroPlugins.NewPluginLogger(path)
			if p.reader != nil {
				err = p.reader.SetOptsPlatform(platformOpts)
			} else {
				err = errors.New("Not implemented")
			}
		} else {
			err = errors.New("Unknown platform")
		}
	}

	return
}

// ReadWithPlatformReader calls the plugin's GetPlatformVars() function
func ReadWithPlatformReader(dict *templates.TemplateVarDictionary, path string, opts *maestroSpecs.PluginOpts, logger maestroSpecs.Logger) (err error) {
	// if we have specifically stated use a plugin, or
	// its a valid

	if strings.HasPrefix(path, "plugin:") {
		s := strings.Split(path, ":")
		//		err = platforms.ReadWithPlatformReader(maestroConfig.GetGlobalConfigDictionary(), s[1])
		path = s[1]

		var myplugin *plugin.Plugin
		myplugin, err = maestroPlugins.GetOrLoadMaestroPlugin(opts, logger, path, nil)
		if err == nil {
			var readerSym plugin.Symbol
			readerSym, err = myplugin.Lookup("GetPlatformVars")
			if err == nil {
				var reader func(dict *templates.TemplateVarDictionary, log maestroSpecs.Logger) error
				// var readerP *maestroSpecs.PlatformReader
				// activeWDp, ok = wdSym.(*maestroSpecs.Watchdog)
				var ok bool
				reader, ok = readerSym.(func(dict *templates.TemplateVarDictionary, log maestroSpecs.Logger) error)
				if ok {
					log.MaestroSuccessf("Platform plugin %s - cached / load successfully.\n", path)
					// logger := maestroPlugins.NewPluginLogger(path)
					err = reader(dict, logger)
				} else {
					log.MaestroErrorf("Platform plugin %s - failed to cast 'GetPlatformVars' symbol. Failing\n", path)
					err = errors.New("bad platform plugin")
				}
			} else {
				log.MaestroErrorf("Platform plugin %s - has no 'GetPlatformVars' symbol. Failing\n", path)
				err = errors.New("Not implemented")
			}
		}
	} else {
		p, ok := builtinsByPlatformName[path]
		if ok {
			// logger := maestroPlugins.NewPluginLogger(path)
			if p.reader != nil {
				err = p.reader.GetPlatformVars(dict, logger)
			} else {
				err = errors.New("Not implemented")
			}
		} else {
			err = errors.New("Unknown platform")
		}
	}

	return
}

