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
	"reflect"
	"strconv"
	"strings"

	"github.com/armPelionEdge/greasego"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/maestroConfig"
	"github.com/armPelionEdge/maestroSpecs"
)

// ChangeHook is base struct for our class
type ChangeHook struct {
	//struct implementing ConfigChangeHook intf
}

//ConfigChangeHandler is the go routine which waits on configChangeRequestChan and when it
//receives a message which is ConfigChangeInfo object, it calls corresponding
//process functions(see below) based on config group.
func ConfigChangeHandler(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {

	log.MaestroInfof("ConfigChangeHandler:: group:%s field:%s old:%v new:%v\n", configgroup, fieldchanged, curvalue, futvalue)
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
		log.MaestroInfof("\nConfigChangeHook:Unknown field or group: %s:%s old:%v new:%v\n", configgroup, fieldchanged, curvalue, futvalue)
	}
}

// ChangesStart is called before reporting any changes via multiple calls to SawChange. It will only be called
// if there is at least one change to report
func (cfgHook ChangeHook) ChangesStart(configgroup string) {
	log.MaestroInfof("ConfigChangeHook:ChangesStart: %s\n", configgroup)

	//ConfigChangeHandler(configChangeRequestChan)
}

// SawChange is called whenever a field changes. It will be called only once for each field which is changed.
// It will always be called after ChangesStart is called
// If SawChange return true, then the value of futvalue will replace the value of current value
func (cfgHook ChangeHook) SawChange(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) (acceptchange bool) {
	log.MaestroInfof("ConfigChangeHook:SawChange: %s:%s old:%v new:%v index:%d\n", configgroup, fieldchanged, curvalue, futvalue, index)
	/*if configChangeRequestChan != nil {
		fieldnames := strings.Split(fieldchanged, ".")
		log.MaestroInfof("ConfigChangeHook:fieldnames: %v\n", fieldnames)
		configChangeRequestChan <- ConfigChangeInfo{configgroup, fieldnames[len(fieldnames)-1], fieldchanged, futvalue, curvalue, index}
	} else {
		log.MaestroInfof("ConfigChangeHook:Config change chan is nil, unable to process change")
	}*/
	ConfigChangeHandler(configgroup, fieldchanged, futvalue, curvalue, index)

	return false //return false as we would apply only those we successfully processed
}

// ChangesComplete is called when all changes for a specific configgroup tagname
// If ChangesComplete returns true, then all changes in that group will be assigned to the current struct
func (cfgHook ChangeHook) ChangesComplete(configgroup string) (acceptallchanges bool) {
	log.MaestroInfof("ConfigChangeHook:ChangesComplete: %s\n", configgroup)

	inst := GetInstance()
	//only apply this after getting all the filters
	if configgroup == "filters" {

		log.MaestroInfof("In config group if\n")

		jsstruct, _ := json.Marshal(inst.logConfig)
		log.MaestroInfof("Applying: \n")
		log.MaestroInfof("%s\n", string(jsstruct))

		//here is where we will add our filters
		inst.ApplyChanges()
	}

	return false //return false as we would apply only those we successfully processed
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Helper functions
////////////////////////////////////////////////////////////////////////////////////////////////////

//AddLogFilter is the function to update the live running filters
func AddLogFilter(filterConfig maestroSpecs.LogFilter) error {
	log.MaestroInfof("in AddLogFilter\n")
	targID := greasego.GetTargetId(filterConfig.Target)
	if targID == 0 {
		return errors.New("target does not exist")
	}

	filter := greasego.NewGreaseLibFilter()
	greasego.AssignFromStruct(filter, filterConfig)

	filter.Target = targID
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

//DeleteLogFilter is used to remove an existing log filter from the live running logs is maestro
func DeleteLogFilter(filterConfig maestroSpecs.LogFilter) error {
	log.MaestroInfof("in DeleteLogFilter\n")
	targID := greasego.GetTargetId(filterConfig.Target)
	if targID == 0 {
		return errors.New("target does not exist")
	}

	filter := greasego.NewGreaseLibFilter()
	greasego.AssignFromStruct(filter, filterConfig)

	filter.Target = targID
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

//FileConfigChange Function to process Target config change
func (inst *logManagerInstance) FileConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("FileConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)

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

//NameConfigChange Function to process Target config change
func (inst *logManagerInstance) NameConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("NameConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)

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

//FiltersConfigChange Function to process filter config change
func (inst *logManagerInstance) FiltersConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("FiltersConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)

	splits := strings.Split(fieldchanged, ".")

	for i := 0; i < len(splits); i++ {
		log.MaestroInfof("index: %n value: %s\n", i, splits[i])
	}
	targetIndex, _ := strconv.Atoi(strings.TrimLeft(strings.TrimRight(splits[0], "]"), "Targets["))

	filterIndex, _ := strconv.Atoi(strings.TrimLeft(strings.TrimRight(splits[1], "]"), "Filters["))

	fieldName := ""

	if len(splits) > 2 {
		fieldName = splits[2]
	}

	log.MaestroInfof("targetIndex=%d\n", targetIndex)
	log.MaestroInfof("filterIndex=%d\n", filterIndex)
	log.MaestroInfof("fieldName=%d\n", fieldName)

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

//ApplyChanges this make the filter changes active
func (inst *logManagerInstance) ApplyChanges() {

	log.MaestroInfof("Entering ApplyChanges()\n")
	filterslen := len(inst.logConfig[0].Filters)

	//get the target id
	targID := greasego.GetTargetId(inst.logConfig[0].File)

	//make sure its legit
	if targID == 0 {
		log.MaestroInfof("FiltersConfigChange: Target ID not found for Target Name: %s\n", inst.logConfig[0].Name)
		return
	}

	log.MaestroInfof("targId=%d\n", targID)

	// if the existing target at index has a name and if the incoming filter specifies a target name,
	// then verify the names match
	if len(inst.logConfig[0].Name) > 0 {
		for i := 0; i < filterslen; i++ {
			targetName := inst.logConfig[0].Filters[i].Target
			if len(targetName) > 0 && (inst.logConfig[0].Name != targetName) {
				log.MaestroInfof("FiltersConfigChange: filter.Target.Name doesn't match existing target at index 0\n")
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

//FormatConfigChange Function to process format config change
func (inst *logManagerInstance) FormatConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("FormatConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
}

//OptsConfigChange Function to process opts config change
func (inst *logManagerInstance) OptsConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("OptsConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
}
