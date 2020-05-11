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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/armPelionEdge/greasego"
	//"github.com/armPelionEdge/maestro/log"

	"github.com/armPelionEdge/maestro/log"
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
func ConfigChangeHandler(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	fmt.Printf("in ConfigChangeHandler\n")

	fmt.Printf("ConfigChangeHandler:: group:%s field:%s old:%v new:%v\n", configgroup, fieldchanged, curvalue, futvalue)
	switch configgroup {
	case "file":
		instance.FileConfigChange(fieldchanged, futvalue, curvalue, index)
	case "name":
		instance.NameConfigChange(fieldchanged, futvalue, curvalue, index)
	case "filters":
		instance.FiltersConfigChange(fieldchanged, futvalue, curvalue, index)
	case "format":
		instance.FormatConfigChange(fieldchanged, futvalue, curvalue, index)
	case "opts":
		instance.OptsConfigChange(fieldchanged, futvalue, curvalue, index)
	default:
		fmt.Printf("\nConfigChangeHook:Unknown field or group: %s:%s old:%v new:%v\n", configgroup, fieldchanged, curvalue, futvalue)
	}
}

// ChangesStart is called before reporting any changes via multiple calls to SawChange. It will only be called
// if there is at least one change to report
func (cfgHook LogConfigChangeHook) ChangesStart(configgroup string) {
	fmt.Printf("ConfigChangeHook:ChangesStart: %s\n", configgroup)

	//ConfigChangeHandler(configChangeRequestChan)
}

// SawChange is called whenever a field changes. It will be called only once for each field which is changed.
// It will always be called after ChangesStart is called
// If SawChange return true, then the value of futvalue will replace the value of current value
func (cfgHook LogConfigChangeHook) SawChange(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) (acceptchange bool) {
	fmt.Printf("ConfigChangeHook:SawChange: %s:%s old:%v new:%v index:%d\n", configgroup, fieldchanged, curvalue, futvalue, index)
	/*if configChangeRequestChan != nil {
		fieldnames := strings.Split(fieldchanged, ".")
		fmt.Printf("ConfigChangeHook:fieldnames: %v\n", fieldnames)
		configChangeRequestChan <- ConfigChangeInfo{configgroup, fieldnames[len(fieldnames)-1], fieldchanged, futvalue, curvalue, index}
	} else {
		fmt.Printf("ConfigChangeHook:Config change chan is nil, unable to process change")
	}*/
	ConfigChangeHandler(configgroup, fieldchanged, futvalue, curvalue, index)

	return false //return false as we would apply only those we successfully processed
}

// ChangesComplete is called when all changes for a specific configgroup tagname
// If ChangesComplete returns true, then all changes in that group will be assigned to the current struct
func (cfgHook LogConfigChangeHook) ChangesComplete(configgroup string) (acceptallchanges bool) {
	fmt.Printf("ConfigChangeHook:ChangesComplete: %s\n", configgroup)

	inst := GetInstance()
	//only apply this after getting all the filters
	if configgroup == "filters" {

		fmt.Println("In config group if")

		jsstruct, _ := json.Marshal(inst.logConfig)
		fmt.Printf("Applying: \n")
		fmt.Println(string(jsstruct))

		//here is where we will add our filters
		inst.ApplyChanges()
	}

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
func (inst *logManagerInstance) FileConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	fmt.Printf("FileConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)

	splits := strings.Split(fieldchanged, ".")

	targetIndex, _ := strconv.Atoi(strings.TrimLeft(strings.TrimRight(splits[0], "]"), "Targets["))
	fieldName := splits[1]

	switch fieldName {
	case "File":
		inst.logConfig[targetIndex].File = reflect.ValueOf(futvalue).String()
	default:
		log.MaestroWarnf("FileConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//Function to process Target config change
func (inst *logManagerInstance) NameConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	fmt.Printf("NameConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)

	splits := strings.Split(fieldchanged, ".")

	targetIndex, _ := strconv.Atoi(strings.TrimLeft(strings.TrimRight(splits[0], "]"), "Targets["))
	fieldName := splits[1]

	switch fieldName {
	case "Name":
		inst.logConfig[targetIndex].Name = reflect.ValueOf(futvalue).String()
	default:
		log.MaestroWarnf("NameConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//Function to process filter config change
func (inst *logManagerInstance) FiltersConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	fmt.Printf("FiltersConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)

	splits := strings.Split(fieldchanged, ".")

	for i := 0; i < len(splits); i++ {
		println("index: %n value: %s", i, splits[i])
	}
	targetIndex, _ := strconv.Atoi(strings.TrimLeft(strings.TrimRight(splits[0], "]"), "Targets["))

	filterIndex, _ := strconv.Atoi(strings.TrimLeft(strings.TrimRight(splits[1], "]"), "Filters["))

	fieldName := ""

	if len(splits) > 2 {
		fieldName = splits[2]
	}

	println("targetIndex=", targetIndex)
	println("filterIndex=", filterIndex)
	println("fieldName=", fieldName)

	switch fieldName {
	case "Target":
		inst.logConfig[targetIndex].Filters[filterIndex].Target = reflect.ValueOf(futvalue).String()
	case "Levels":
		inst.logConfig[targetIndex].Filters[filterIndex].Levels = reflect.ValueOf(futvalue).String()
	case "Pre":
		inst.logConfig[targetIndex].Filters[filterIndex].Pre = reflect.ValueOf(futvalue).String()
	case "Post":
		inst.logConfig[targetIndex].Filters[filterIndex].Post = reflect.ValueOf(futvalue).String()
	default:
		log.MaestroWarnf("FiltersConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}
func (inst *logManagerInstance) ApplyChanges() {

	fmt.Println("Entering ApplyChanges()")
	filterslen := len(inst.logConfig[0].Filters)

	// verify we have an existing target because we don't currently support adding new targets
	/*if len(inst.logConfig) <= index {
		fmt.Printf("FiltersConfigChange: No targets exist.  Adding new targets is not supported.\n")
		return
	}*/
	targId := greasego.GetTargetId(inst.logConfig[0].File)
	if targId == 0 {
		fmt.Printf("FiltersConfigChange: Target ID not found for Target Name: %s\n", inst.logConfig[0].Name)
		return
	}
	fmt.Println("targId=", targId)

	// if the existing target at index has a name and if the incoming filter specifies a target name,
	// then verify the names match
	if len(inst.logConfig[0].Name) > 0 {
		for i := 0; i < filterslen; i++ {
			target_name := inst.logConfig[0].Filters[i].Target
			if len(target_name) > 0 && (inst.logConfig[0].Name != target_name) {
				fmt.Printf("FiltersConfigChange: filter.Target.Name doesn't match existing target at index 0\n")
				return
			}
		}
	}

	// FIXME: the incoming filter config completely replaces any existing config which is known
	//     as "replace" behavior for the "Existing" flag.  Do we need to handle "override" flag?

	// For "replace" behavior, we delete all existing filters for the given target
	for _, filter := range inst.logConfig[0].Filters {
		DeleteLogFilter(filter)
	}
	// now that all existing filters have been deleted, clear the local config array
	//inst.logConfig[0].Filters = make([]maestroSpecs.LogFilter, filterslen)

	// create the new filters based on the incoming config in futvalue
	for i := 0; i < filterslen; i++ {
		// configure greasego
		AddLogFilter(inst.logConfig[0].Filters[i])

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
}

// SawChange is called whenever a field changes. It will be called only once for each field which is changed.
// It will always be called after ChangesStart is called
// If SawChange return true, then the value of futvalue will replace the value of current value
func (cfgHook CommitConfigChangeHook) SawChange(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) (acceptchange bool) {
	fmt.Printf("CommitChangeHook:SawChange: %s:%s old:%v new:%v index:%d\n", configgroup, fieldchanged, curvalue, futvalue, index)
	/*instance = GetInstance()
	switch fieldchanged {
	case "ConfigCommitFlag":
	case "LastUpdateTimestamp":
	case "TotalCommitCountFromBoot":
	default:
		fmt.Printf("\nCommitChangeHook:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}*/
	return false //return false as we would apply only those we successfully processed
}

// ChangesComplete is called when all changes for a specific configgroup tagname
// If ChangesComplete returns true, then all changes in that group will be assigned to the current struct
func (cfgHook CommitConfigChangeHook) ChangesComplete(configgroup string) (acceptallchanges bool) {
	fmt.Printf("CommitChangeHook:ChangesComplete: %s\n", configgroup)

	return false //return false as we would apply only those we successfully processed
}
