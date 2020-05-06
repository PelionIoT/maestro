package logconfig

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
	"reflect"
	"strings"

	"github.com/armPelionEdge/greasego"
	//"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/maestroConfig"
	"github.com/armPelionEdge/maestroSpecs"
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
	fmt.Printf("in ConfigChangeHandler\n")
	instance = GetInstance()
	for configChange := range jobConfigChangeChan {
		fmt.Printf("ConfigChangeHandler:: group:%s field:%s old:%v new:%v\n", configChange.configgroup, configChange.fieldchanged, configChange.curvalue, configChange.futvalue)
		switch configChange.configgroup {
		case "name":
			instance.TargetConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index)
		case "filters":
			instance.FiltersConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index)
		case "format":
			instance.FormatConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index)
		case "opts":
			instance.OptsConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index)
		default:
			fmt.Printf("\nConfigChangeHook:Unknown field or group: %s:%s old:%v new:%v\n", configChange.configgroup, configChange.fieldchanged, configChange.curvalue, configChange.futvalue)
		}
	}
}

// ChangesStart is called before reporting any changes via multiple calls to SawChange. It will only be called
// if there is at least one change to report
func (cfgHook LogConfigChangeHook) ChangesStart(configgroup string) {
	fmt.Printf("ConfigChangeHook:ChangesStart: %s\n", configgroup)

	ConfigChangeHandler(configChangeRequestChan)
}

// SawChange is called whenever a field changes. It will be called only once for each field which is changed.
// It will always be called after ChangesStart is called
// If SawChange return true, then the value of futvalue will replace the value of current value
func (cfgHook LogConfigChangeHook) SawChange(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) (acceptchange bool) {
	fmt.Printf("ConfigChangeHook:SawChange: %s:%s old:%v new:%v index:%d\n", configgroup, fieldchanged, curvalue, futvalue, index)
	if configChangeRequestChan != nil {
		fieldnames := strings.Split(fieldchanged, ".")
		fmt.Printf("ConfigChangeHook:fieldnames: %v\n", fieldnames)
		configChangeRequestChan <- ConfigChangeInfo{configgroup, fieldnames[len(fieldnames)-1], fieldchanged, futvalue, curvalue, index}
	} else {
		fmt.Printf("ConfigChangeHook:Config change chan is nil, unable to process change")
	}

	return false //return false as we would apply only those we successfully processed
}

// ChangesComplete is called when all changes for a specific configgroup tagname
// If ChangesComplete returns true, then all changes in that group will be assigned to the current struct
func (cfgHook LogConfigChangeHook) ChangesComplete(configgroup string) (acceptallchanges bool) {
	fmt.Printf("ConfigChangeHook:ChangesComplete: %s\n", configgroup)
	return false //return false as we would apply only those we successfully processed
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Helper functions
////////////////////////////////////////////////////////////////////////////////////////////////////
func AddLogFilter(filterConfig maestroSpecs.LogFilter) error {
	fmt.Printf("in AddLogFilter\n")
	targId := greasego.GetTargetId(filterConfig.Target)
	if targId == 0 {
		return errors.New("target does not exist")
	}

	filter := greasego.NewGreaseLibFilter()
	greasego.AssignFromStruct(filter, filterConfig)

	filter.Target = targId
	greasego.SetFilterValue(filter, greasego.GREASE_LIB_SET_FILTER_TARGET, filter.Target)

	if len(filterConfig.Levels) > 0 {
		mask := maestroConfig.ConvertLevelStringToUint32Mask(filterConfig.Levels)
		greasego.SetFilterValue(filter, greasego.GREASE_LIB_SET_FILTER_MASK, mask)
	}

	if len(filterConfig.Tag) > 0 {
		tag := maestroConfig.ConvertTagStringToUint32(filterConfig.Tag)
		greasego.SetFilterValue(filter, greasego.GREASE_LIB_SET_FILTER_MASK, tag)
	}

	added := greasego.AddFilter(filter)
	if added != 0 {
		return errors.New("failed to add filter")
	}

	return nil
}

func DeleteLogFilter(filterConfig maestroSpecs.LogFilter) error {
	fmt.Printf("in DeleteLogFilter\n")
	targId := greasego.GetTargetId(filterConfig.Target)
	if targId == 0 {
		return errors.New("target does not exist")
	}

	filter := greasego.NewGreaseLibFilter()
	greasego.AssignFromStruct(filter, filterConfig)

	filter.Target = targId
	greasego.SetFilterValue(filter, greasego.GREASE_LIB_SET_FILTER_TARGET, filter.Target)

	if len(filterConfig.Levels) > 0 {
		mask := maestroConfig.ConvertLevelStringToUint32Mask(filterConfig.Levels)
		greasego.SetFilterValue(filter, greasego.GREASE_LIB_SET_FILTER_MASK, mask)
	}

	if len(filterConfig.Tag) > 0 {
		tag := maestroConfig.ConvertTagStringToUint32(filterConfig.Tag)
		greasego.SetFilterValue(filter, greasego.GREASE_LIB_SET_FILTER_MASK, tag)
	}

	removed := greasego.DeleteFilter(filter)
	if removed != 0 {
		return errors.New("failed to remove filter")
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Functions to process the parameters which are changed
////////////////////////////////////////////////////////////////////////////////////////////////////

//Function to process Target config change
func (inst *logManagerInstance) TargetConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	fmt.Printf("TargetConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
}

//Function to process filter config change
func (inst *logManagerInstance) FiltersConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	fmt.Printf("FiltersConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	filterslen := reflect.ValueOf(futvalue).Len()

	// verify we have an existing target because we don't currently support adding new targets
	if len(inst.logConfig) <= index {
		fmt.Printf("FiltersConfigChange: No targets exist.  Adding new targets is not supported.\n")
		return
	}
	targId := greasego.GetTargetId(inst.logConfig[index].Name)
	if targId == 0 {
		fmt.Printf("FiltersConfigChange: Target ID not found for Target Name: %s\n", inst.logConfig[index].Name)
		return
	}

	// if the existing target at index has a name and if the incoming filter specifies a target name,
	// then verify the names match
	if len(inst.logConfig[index].Name) > 0 {
		for i := 0; i < filterslen; i++ {
			target_name := reflect.ValueOf(futvalue).Index(i).FieldByName("Target").String()
			if len(target_name) > 0 && (inst.logConfig[index].Name != target_name) {
				fmt.Printf("FiltersConfigChange: filter.Target.Name doesn't match existing target at index %d\n", index)
				return
			}
		}
	}

	// FIXME: the incoming filter config completely replaces any existing config which is known
	//     as "replace" behavior for the "Existing" flag.  Do we need to handle "override" flag?

	// For "replace" behavior, we delete all existing filters for the given target
	for _, filter := range inst.logConfig[index].Filters {
		DeleteLogFilter(filter)
	}
	// now that all existing filters have been deleted, clear the local config array
	inst.logConfig[index].Filters = make([]maestroSpecs.LogFilter, filterslen)

	// create the new filters based on the incoming config in futvalue
	for i := 0; i < filterslen; i++ {
		inst.logConfig[index].Filters[i].Levels = reflect.ValueOf(futvalue).Index(i).FieldByName("Levels").String()
		inst.logConfig[index].Filters[i].Tag = reflect.ValueOf(futvalue).Index(i).FieldByName("Tag").String()
		inst.logConfig[index].Filters[i].Pre = reflect.ValueOf(futvalue).Index(i).FieldByName("Pre").String()
		inst.logConfig[index].Filters[i].Post = reflect.ValueOf(futvalue).Index(i).FieldByName("Post").String()
		inst.logConfig[index].Filters[i].PostFmtPreMsg = reflect.ValueOf(futvalue).Index(i).FieldByName("PostFmtPreMsg").String()

		// configure greasego
		AddLogFilter(inst.logConfig[index].Filters[i])

		// FIXME: push the new config to maestroDB here?
	}
}

//Function to process format config change
func (inst *logManagerInstance) FormatConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	fmt.Printf("FormatConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
}

//Function to process opts config change
func (inst *logManagerInstance) OptsConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	fmt.Printf("OptsConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
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
	fmt.Printf("in ConfigApplyHandler\n")
	/*	for applyChange := range jobConfigApplyRequestChan {
		fmt.Printf("ConfigApplyHandler::Received a apply change message: %v\n", applyChange)
		if applyChange {
			instance = GetInstance()
			fmt.Printf("ConfigApplyHandler::Processing apply change: %v\n", instance.CurrConfigCommit.ConfigCommitFlag)
			instance.submitConfigAndSync(instance.logConfig)
			//Setup the intfs using new config
			instance.setupInterfaces()
			instance.CurrConfigCommit.ConfigCommitFlag = false
			instance.CurrConfigCommit.LastUpdateTimestamp = time.Now().Format(time.RFC850)
			instance.CurrConfigCommit.TotalCommitCountFromBoot = instance.CurrConfigCommit.TotalCommitCountFromBoot + 1
			//Now write out the updated commit config
			err := instance.ddbConfigClient.Config(DDB_LOG_CONFIG_COMMIT_FLAG).Put(&instance.CurrConfigCommit)
			if err == nil {
				fmt.Printf("Updating commit config object to devicedb succeeded.\n")
			} else {
				fmt.Printf("Unable to update commit config object to devicedb\n")
			}
		}
	}*/
}

// ChangesStart is called before reporting any changes via multiple calls to SawChange. It will only be called
// if there is at least one change to report
func (cfgHook CommitConfigChangeHook) ChangesStart(configgroup string) {
	fmt.Printf("CommitChangeHook:ChangesStart: %s\n", configgroup)
	if configApplyRequestChan == nil {
		configApplyRequestChan = make(chan bool, 10)
		go ConfigApplyHandler(configApplyRequestChan)
	}
}

// SawChange is called whenever a field changes. It will be called only once for each field which is changed.
// It will always be called after ChangesStart is called
// If SawChange return true, then the value of futvalue will replace the value of current value
func (cfgHook CommitConfigChangeHook) SawChange(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) (acceptchange bool) {
	fmt.Printf("CommitChangeHook:SawChange: %s:%s old:%v new:%v index:%d\n", configgroup, fieldchanged, curvalue, futvalue, index)
	instance = GetInstance()
	switch fieldchanged {
	case "ConfigCommitFlag":
		instance.CurrConfigCommit.ConfigCommitFlag = reflect.ValueOf(futvalue).Bool()
		if instance.CurrConfigCommit.ConfigCommitFlag == true {
			//flag set to true, apply the new config
			fmt.Printf("\nCommitChangeHook:commit flag set, applying changes")
			configApplyRequestChan <- true
		}
	case "LastUpdateTimestamp":
	case "TotalCommitCountFromBoot":
	default:
		fmt.Printf("\nCommitChangeHook:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
	return false //return false as we would apply only those we successfully processed
}

// ChangesComplete is called when all changes for a specific configgroup tagname
// If ChangesComplete returns true, then all changes in that group will be assigned to the current struct
func (cfgHook CommitConfigChangeHook) ChangesComplete(configgroup string) (acceptallchanges bool) {
	fmt.Printf("CommitChangeHook:ChangesComplete: %s\n", configgroup)
	return false //return false as we would apply only those we successfully processed
}
