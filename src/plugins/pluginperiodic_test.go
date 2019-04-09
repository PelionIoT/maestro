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
	"log"
	"testing"
	"time"

	"github.com/armPelionEdge/maestro-plugins-template/teststructs"
	"github.com/armPelionEdge/maestroSpecs"
)

//var logger *testtools.NonProductionPrefixedLogger

// func TestMain(m *testing.M) {
// 	logger = testtools.NewNonProductionPrefixedLogger("pluginloader_test: ")
// 	m.Run()
// }
// func TestPluginLoader(t *testing.T) {
// }

func TestPeriodicStuff(t *testing.T) {
	opts := new(maestroSpecs.PluginOpts)

	opts.Periodic = make([]*maestroSpecs.PluginPeriodic, 0, 0)
	p := new(maestroSpecs.PluginPeriodic)
	p.FunctionName = "CallMePeriodically"
	p.IntervalMS = 1000 // every second
	opts.Periodic = append(opts.Periodic, p)

	testN := 0

	callback := func(param interface{}) {
		log.Printf("in callback()")
		if param != nil {
			stuff, ok := param.(*teststructs.Stuff)
			if !ok {
				log.Fatalf("Ouch - failed to cast to teststructs.Stuff - which is th data type used by the plugin.")
			} else {
				testN = stuff.PanicAtTheDisco
				if stuff.PanicAtTheDisco > 10 {
					log.Printf("Ok, test complete.")
				}
			}
		} else {
			log.Fatalf("Ouch - plugin's callback - param was nil (?)")
		}
	}

	plugin, err := GetOrLoadMaestroPlugin(opts, logger, "../../maestro-plugins-template/testpluginperiodic/testpluginperiodic.so", callback)
	if err != nil {
		log.Fatalf("Ouch: %s\n", err.Error())
	}
	if plugin == nil {
		log.Fatalf("Why is plugin nil??\n")
	}
	log.Printf("Initial load ok.\n")
	// should be already loaded now - again...
	plugin, err = GetOrLoadMaestroPlugin(opts, logger, "../../maestro-plugins-template/testpluginperiodic/testpluginperiodic.so", callback)
	if err != nil {
		log.Fatalf("Ouch: %s\n", err.Error())
	}
	if plugin == nil {
		log.Fatalf("Why is plugin nil??\n")
	}
	log.Printf("Load from cache ok. 15 second test\n")
	time.Sleep(15 * time.Second) // need enough time to hit 10
	if testN < 9 {
		log.Fatalf("Did not call callbacks enough. (testN == %d)", testN)
	} else {
		StopPeriodicOfPlugin("../../maestro-plugins-template/testpluginperiodic/testpluginperiodic.so", "CallMePeriodically")
	}
}
