package configMgr

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
	"encoding/gob"
	"fmt"
	"github.com/armPelionEdge/maestro/maestroConfig"
	"github.com/armPelionEdge/maestro/storage"
	"github.com/armPelionEdge/maestroSpecs"
	"github.com/armPelionEdge/stow"
	"github.com/boltdb/bolt"
)

/**
 * Stores configs for Jobs
 */

const c_JOBCONFIG_STORE = "jobConfigs"

// initialized in init()
type configManager struct {
	globalConfig *maestroConfig.YAMLMaestroConfig
	db           *bolt.DB
	store        *stow.Store
	_cacheNames  map[string]bool
	//	_cache
}

type ConfigObjectError struct {
	Name string
	Job  string
	Code int
	Aux  string
}

func (this *ConfigObjectError) Error() string {
	s := fmt.Sprintf("%d", this.Code)
	return "Error: " + this.Job + ":" + this.Name + " code:" + s + " " + this.Aux
}

const (
	CONFIG_NAME_INVALID = iota
	CONFIG_NO_STORAGE   = iota
)

var configMgrInstance *configManager

// impl storage.StorageUser interface
func (this *configManager) StorageInit(instance storage.MaestroDBStorageInterface) {

	this.db = instance.GetDb()
	this.store = stow.NewStore(instance.GetDb(), []byte(c_JOBCONFIG_STORE))
}

func (this *configManager) StorageReady(instance storage.MaestroDBStorageInterface) {

}

func (this *configManager) StorageClosed(instance storage.MaestroDBStorageInterface) {
}

func init() {
	gob.Register(&maestroSpecs.ConfigDefinitionPayload{})
	gob.Register(&maestroSpecs.ConfigFileDataPayload{})
	configMgrInstance = new(configManager)
	configMgrInstance._cacheNames = make(map[string]bool)
}

func StartJobConfigManager(config *maestroConfig.YAMLMaestroConfig) {
	storage.RegisterStorageUser(configMgrInstance)
	configMgrInstance.globalConfig = config
}

func JobConfigMgr() (ret *configManager) {
	return configMgrInstance
}

func makeJobConfigKey(jobname string, confname string) string {
	return jobname + "$" + confname
}

func (this *configManager) GetConfig(name string, jobname string) (conf maestroSpecs.ConfigDefinition, err error) {
	this.store.Get(makeJobConfigKey(jobname, name), &conf)
	return
}

func (this *configManager) SetConfig(name string, jobname string, conf maestroSpecs.ConfigDefinition) (err error) {
	if this.store == nil {
		err = &ConfigObjectError{Name: name, Job: jobname, Code: CONFIG_NO_STORAGE, Aux: "No storage setup."}
		return
	}
	if len(name) > 0 && len(jobname) > 0 {
		err = this.store.Put(makeJobConfigKey(jobname, name), &conf)
	} else {
		err = &ConfigObjectError{Name: name, Job: jobname, Code: CONFIG_NAME_INVALID, Aux: "SetConfig failed."}
	}
	return
}

func (this *configManager) GetConfigsByJob(jobname string) (config []maestroSpecs.ConfigDefinition, err error) {
	config = make([]maestroSpecs.ConfigDefinition, 0, 10)
	// use stow.IterateIf
	//	temp := maestroSpecs.ConfigDefinitionPayload{"testname","tesjob","data","encodingtest",nil,""}
	var temp maestroSpecs.ConfigDefinition
	this.store.IterateFromPrefixIf([]byte(makeJobConfigKey(jobname, "")), func(key []byte, val interface{}) bool {
		el, ok := val.(*maestroSpecs.ConfigDefinition)
		if ok {
			//			DEBUG_OUT("\n<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<< CONFIG cast %s %+v\n\n",string(key),el)
			config = append(config, (*el).Dup())
		} else {
			//			DEBUG_OUT("\n<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<< CONFIG cast FAILED %s \n\n",string(key))
		}
		return true
	}, &temp)
	return
}

func (this *configManager) DelConfig(name string, jobname string) {
	key := makeJobConfigKey(jobname, name)
	this.store.Delete(key)
	delete(this._cacheNames, key)
}

func (this *configManager) GetAllConfigs() (config []maestroSpecs.ConfigDefinition, err error) {
	config = make([]maestroSpecs.ConfigDefinition, 0, 10)
	// use stow.IterateIf
	var temp maestroSpecs.ConfigDefinition
	this._cacheNames = make(map[string]bool)
	this.store.IterateIf(func(key []byte, val interface{}) bool {
		this._cacheNames[string(key[:])] = true
		el, ok := val.(*maestroSpecs.ConfigDefinition)
		if ok {
			config = append(config, (*el).Dup())
		}
		return true
	}, &temp)
	return
}

func (this *configManager) LoadAllNames() (err error) {
	// use stow.IterateIf
	var temp maestroSpecs.ConfigDefinitionPayload
	this._cacheNames = make(map[string]bool)
	this.store.IterateIf(func(key []byte, val interface{}) bool {
		this._cacheNames[string(key[:])] = true
		return true
	}, &temp)
	return
}

func (this *configManager) Exists(name string, jobname string) bool {
	if this._cacheNames[makeJobConfigKey(jobname, name)] {
		return true
	} else {
		return false
	}
}
