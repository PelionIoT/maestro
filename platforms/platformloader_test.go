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
	"fmt"
	"log"
	"testing"

	maestroLog "github.com/PelionIoT/maestro/log"
	"github.com/PelionIoT/maestro/testtools"
	"github.com/PelionIoT/maestroSpecs/templates"
)

var logger *testtools.NonProductionPrefixedLogger

func TestMain(m *testing.M) {
	logger = testtools.NewNonProductionPrefixedLogger("pluginloader_test: ")
	maestroLog.DisableGreaseLog()
	m.Run()
}
func TestPlatformPlugin(t *testing.T) {
	dict := templates.NewTemplateVarDictionary()
	err := ReadWithPlatformReader(dict, "plugin:../../maestro-plugins-template/testplatformplugin/testplatformplugin.so", nil, logger)
	if err != nil {
		log.Fatalf("Error on ReadWithPlatformReader: %s\n", err.Error())
	}
	fmt.Printf("dict:\n%s", dict.DumpAsString())
	// again - should used cached load of plugin...
	err = ReadWithPlatformReader(dict, "plugin:../../maestro-plugins-template/testplatformplugin/testplatformplugin.so", nil, logger)
	if err != nil {
		log.Fatalf("Error on ReadWithPlatformReader: %s\n", err.Error())
	}
	fmt.Printf("dict:\n%s", dict.DumpAsString())
	// again - should used cached load of plugin...
	var key string
	var cert string
	key, cert, err = GeneratePlatformKeys(dict, "some_deviceid", "someaccount_id", "plugin:../../maestro-plugins-template/testplatformplugin/testplatformplugin.so", nil, logger)
	if err != nil {
		log.Fatalf("Error on GeneratePlatformKeys: %s\n", err.Error())
	}
	fmt.Printf("key:%s\n", key)
	fmt.Printf("cert:%s\n", cert)
	err = WritePlatformKeys(dict, key, cert, "plugin:../../maestro-plugins-template/testplatformplugin/testplatformplugin.so", nil, logger)
	if err != nil {
		log.Fatalf("Error on WritePlatformReader: %s\n", err.Error())
	}

	// plugin, err := GetOrLoadMaestroPlugin(logger, "../../maestro-plugins-template/testplatformplugin/testplatformplugin.so")
	// if err != nil {
	// 	log.Fatalf("Ouch: %s\n", err.Error())
	// }
	// if plugin == nil {
	// 	log.Fatalf("Why is plugin nil??\n")
	// }
	// log.Printf("Initial load ok.\n")
	// // should be already loaded now - again...
	// plugin, err = GetOrLoadMaestroPlugin(logger, "../../maestro-plugins-template/testplatformplugin/testplatformplugin.so")
	// if err != nil {
	// 	log.Fatalf("Ouch: %s\n", err.Error())
	// }
	// if plugin == nil {
	// 	log.Fatalf("Why is plugin nil??\n")
	// }
	// log.Printf("Load from cache ok.\n")
}

func TestPlatformFSOnly(t *testing.T) {
	dict := templates.NewTemplateVarDictionary()
	platformOpts := make(map[string]interface{})
	platformOpts["identityPath"] = "./identity.json"
	err := SetPlatformReaderOpts(platformOpts, "fsonly", nil, logger)
	if err != nil {
		log.Fatalf("Error on SetPlatformReaderOpts: %s\n", err.Error())
	}
	err = ReadWithPlatformReader(dict, "fsonly", nil, logger)
	if err != nil {
		log.Fatalf("Error on ReadWithPlatformReader: %s\n", err.Error())
	}
	fmt.Printf("dict:\n%s", dict.DumpAsString())
	// again - should used cached load of plugin...
	err = ReadWithPlatformReader(dict, "fsonly", nil, logger)
	if err != nil {
		log.Fatalf("Error on ReadWithPlatformReader: %s\n", err.Error())
	}
	fmt.Printf("dict:\n%s", dict.DumpAsString())
	// again - should used cached load of plugin...
	var key string
	var cert string
	key, cert, err = GeneratePlatformKeys(dict, "some_deviceid", "someaccount_id", "fsonly", nil, logger)
	if err != nil {
		log.Fatalf("Error on GeneratePlatformKeys: %s\n", err.Error())
	}
	fmt.Printf("key:%s\n", key)
	fmt.Printf("cert:%s\n", cert)
	err = WritePlatformKeys(dict, key, cert, "fsonly", nil, logger)
	if err != nil {
		log.Fatalf("Error on WritePlatformReader: %s\n", err.Error())
	}

	// plugin, err := GetOrLoadMaestroPlugin(logger, "../../maestro-plugins-template/testplatformplugin/testplatformplugin.so")
	// if err != nil {
	// 	log.Fatalf("Ouch: %s\n", err.Error())
	// }
	// if plugin == nil {
	// 	log.Fatalf("Why is plugin nil??\n")
	// }
	// log.Printf("Initial load ok.\n")
	// // should be already loaded now - again...
	// plugin, err = GetOrLoadMaestroPlugin(logger, "../../maestro-plugins-template/testplatformplugin/testplatformplugin.so")
	// if err != nil {
	// 	log.Fatalf("Ouch: %s\n", err.Error())
	// }
	// if plugin == nil {
	// 	log.Fatalf("Why is plugin nil??\n")
	// }
	// log.Printf("Load from cache ok.\n")
}
