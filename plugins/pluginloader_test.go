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

	"github.com/PelionIoT/maestro/testtools"
)

var logger *testtools.NonProductionPrefixedLogger

func TestMain(m *testing.M) {
	logger = testtools.NewNonProductionPrefixedLogger("pluginloader_test: ")
	m.Run()
}
func TestPluginLoader(t *testing.T) {
	plugin, err := GetOrLoadMaestroPlugin(nil, logger, "../../maestro-plugins-template/testplatformplugin/testplatformplugin.so", nil)
	if err != nil {
		log.Fatalf("Ouch: %s\n", err.Error())
	}
	if plugin == nil {
		log.Fatalf("Why is plugin nil??\n")
	}
	log.Printf("Initial load ok.\n")
	// should be already loaded now - again...
	plugin, err = GetOrLoadMaestroPlugin(nil, logger, "../../maestro-plugins-template/testplatformplugin/testplatformplugin.so", nil)
	if err != nil {
		log.Fatalf("Ouch: %s\n", err.Error())
	}
	if plugin == nil {
		log.Fatalf("Why is plugin nil??\n")
	}
	log.Printf("Load from cache ok.\n")
}
