package log

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
	"reflect"
	"strings"
	"time"

	"github.com/armPelionEdge/maestro/log"
)

type LogConfigChangeHook struct {
	//struct implementing ConfigChangeHook intf
}

type ConfigChangeInfo struct {
	//struct capturing the job args to be carried out when a config change happens
	configgroup        string      //Config group
	fieldchanged       string      //Name of the field which is changed
	canonicalfieldname string      //Canonocal Name(which includes the parent struct name) of the field which is changed
	futvalue           interface{} //This corresponds to the new value of the changed field
	curvalue           interface{} //This corresponds to the old/current value of the changed field
	index              int         //Index of the element if the changed field is an array, otherwise this fied
}

var configChangeRequestChan chan ConfigChangeInfo = nil

//This is the go routine which waits on configChangeRequestChan and when it
//receives a message which is ConfigChangeInfo object, it calls corresponding
//process functions(see below) based on config group.
func ConfigChangeHandler(jobConfigChangeChan <-chan ConfigChangeInfo) {

	instance = GetInstance()
	for configChange := range jobConfigChangeChan {
		log.MaestroInfof("ConfigChangeHandler:: group:%s field:%s old:%v new:%v\n", configChange.configgroup, configChange.fieldchanged, configChange.curvalue, configChange.futvalue)
		switch configChange.configgroup {
		case "target":
			instance.TargetConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index)
		case "levels":
			instance.LevelConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index)
		case "tag":
			instance.TagConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index)
		case "pre":
			instance.PreConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index)
		case "post":
			instance.PostConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index)
		case "post-fmt-pre-msg":
			instance.PostFmtPreMsgConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index)
		default:
			log.MaestroWarnf("\nConfigChangeHook:Unknown field or group: %s:%s old:%v new:%v\n", configChange.configgroup, configChange.fieldchanged, configChange.curvalue, configChange.futvalue)
		}
	}
}

// ChangesStart is called before reporting any changes via multiple calls to SawChange. It will only be called
// if there is at least one change to report
func (cfgHook NetworkConfigChangeHook) ChangesStart(configgroup string) {
	log.MaestroInfof("ConfigChangeHook:ChangesStart: %s\n", configgroup)
	if configChangeRequestChan == nil {
		configChangeRequestChan = make(chan ConfigChangeInfo, 100)
		go ConfigChangeHandler(configChangeRequestChan)
	}
}

// SawChange is called whenever a field changes. It will be called only once for each field which is changed.
// It will always be called after ChangesStart is called
// If SawChange return true, then the value of futvalue will replace the value of current value
func (cfgHook NetworkConfigChangeHook) SawChange(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) (acceptchange bool) {
	log.MaestroInfof("ConfigChangeHook:SawChange: %s:%s old:%v new:%v index:%d\n", configgroup, fieldchanged, curvalue, futvalue, index)
	if configChangeRequestChan != nil {
		fieldnames := strings.Split(fieldchanged, ".")
		log.MaestroInfof("ConfigChangeHook:fieldnames: %v\n", fieldnames)
		configChangeRequestChan <- ConfigChangeInfo{configgroup, fieldnames[len(fieldnames)-1], fieldchanged, futvalue, curvalue, index}
	} else {
		log.MaestroErrorf("ConfigChangeHook:Config change chan is nil, unable to process change")
	}

	return false //return false as we would apply only those we successfully processed
}

// ChangesComplete is called when all changes for a specific configgroup tagname
// If ChangesComplete returns true, then all changes in that group will be assigned to the current struct
func (cfgHook NetworkConfigChangeHook) ChangesComplete(configgroup string) (acceptallchanges bool) {
	log.MaestroInfof("ConfigChangeHook:ChangesComplete: %s\n", configgroup)
	return false //return false as we would apply only those we successfully processed
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Functions to process the parameters which are changed
////////////////////////////////////////////////////////////////////////////////////////////////////

//Function to process Dhcp config change
func (inst *logManagerInstance) TargetConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("TargetConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
}

//////////////////////////////////////////////////////////////////////////////////////////
// Monitor for ConfigChangeHook
//////////////////////////////////////////////////////////////////////////////////////////
type CommitConfigChangeHook struct {
	//struct implementing CommitConfigChangeHook intf
}

var configApplyRequestChan chan bool = nil

//This is the go routine which waits on jobConfigApplyRequestChan and when it
//receives an updated config it submits the config and sets up the interfaces based
//on new configuration
func ConfigApplyHandler(jobConfigApplyRequestChan <-chan bool) {
	for applyChange := range jobConfigApplyRequestChan {
		log.MaestroInfof("ConfigApplyHandler::Received a apply change message: %v\n", applyChange)
		if applyChange {
			instance = GetInstance()
			log.MaestroInfof("ConfigApplyHandler::Processing apply change: %v\n", instance.CurrConfigCommit.ConfigCommitFlag)
			instance.submitConfig(instance.networkConfig)
			//Setup the intfs using new config
			instance.setupInterfaces()
			instance.CurrConfigCommit.ConfigCommitFlag = false
			instance.CurrConfigCommit.LastUpdateTimestamp = time.Now().Format(time.RFC850)
			instance.CurrConfigCommit.TotalCommitCountFromBoot = instance.CurrConfigCommit.TotalCommitCountFromBoot + 1
			//Now write out the updated commit config
			err := instance.ddbConfigClient.Config(DDB_NETWORK_CONFIG_COMMIT_FLAG).Put(&instance.CurrConfigCommit)
			if err == nil {
				log.MaestroInfof("Updating commit config object to devicedb succeeded.\n")
			} else {
				log.MaestroErrorf("Unable to update commit config object to devicedb\n")
			}
		}
	}
}

// ChangesStart is called before reporting any changes via multiple calls to SawChange. It will only be called
// if there is at least one change to report
func (cfgHook CommitConfigChangeHook) ChangesStart(configgroup string) {
	log.MaestroInfof("CommitChangeHook:ChangesStart: %s\n", configgroup)
	if configApplyRequestChan == nil {
		configApplyRequestChan = make(chan bool, 10)
		go ConfigApplyHandler(configApplyRequestChan)
	}
}

// SawChange is called whenever a field changes. It will be called only once for each field which is changed.
// It will always be called after ChangesStart is called
// If SawChange return true, then the value of futvalue will replace the value of current value
func (cfgHook CommitConfigChangeHook) SawChange(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) (acceptchange bool) {
	log.MaestroInfof("CommitChangeHook:SawChange: %s:%s old:%v new:%v index:%d\n", configgroup, fieldchanged, curvalue, futvalue, index)
	instance = GetInstance()
	switch fieldchanged {
	case "ConfigCommitFlag":
		instance.CurrConfigCommit.ConfigCommitFlag = reflect.ValueOf(futvalue).Bool()
		if instance.CurrConfigCommit.ConfigCommitFlag == true {
			//flag set to true, apply the new config
			log.MaestroWarnf("\nCommitChangeHook:commit flag set, applying changes")
			configApplyRequestChan <- true
		}
	case "LastUpdateTimestamp":
	case "TotalCommitCountFromBoot":
	default:
		log.MaestroWarnf("\nCommitChangeHook:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
	return false //return false as we would apply only those we successfully processed
}

// ChangesComplete is called when all changes for a specific configgroup tagname
// If ChangesComplete returns true, then all changes in that group will be assigned to the current struct
func (cfgHook CommitConfigChangeHook) ChangesComplete(configgroup string) (acceptallchanges bool) {
	log.MaestroInfof("CommitChangeHook:ChangesComplete: %s\n", configgroup)
	return false //return false as we would apply only those we successfully processed
}
