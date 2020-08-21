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

	inst := GetInstance()

	if inst.logConfigFuture == nil {
		inst.logConfigFuture = make([]maestroSpecs.LogTarget, 1)
	}

	//fmt.Printf("ConfigChangeHandler:: group:%s field:%s old:%v new:%v\n", configgroup, fieldchanged, curvalue, futvalue)
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
		//fmt.Printf("\nConfigChangeHook:Unknown field or group: %s:%s old:%v new:%v\n", configgroup, fieldchanged, curvalue, futvalue)
	}
}

// ChangesStart is called before reporting any changes via multiple calls to SawChange. It will only be called
// if there is at least one change to report
func (cfgHook ChangeHook) ChangesStart(configgroup string) {
	//fmt.Printf("ConfigChangeHook:ChangesStart: %s\n", configgroup)

	logManager := GetInstance()

	if logManager.logConfigOriginal == nil {
		logManager.logConfigOriginal = make([]maestroSpecs.LogTarget, len(logManager.logConfigRunning))
		logManager.logConfigOriginal = logManager.logConfigRunning
	}
	// FIXME!
	// clear all the changes here? elsewhere?
}

// SawChange is called whenever a field changes. It will be called only once for each field which is changed.
// It will always be called after ChangesStart is called
// If SawChange return true, then the value of futvalue will replace the value of current value
func (cfgHook ChangeHook) SawChange(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) (acceptchange bool) {
	//fmt.Printf("ConfigChangeHook:SawChange: %s:%s old:%v new:%v index:%d\n", configgroup, fieldchanged, curvalue, futvalue, index)
	/*if configChangeRequestChan != nil {
		fieldnames := strings.Split(fieldchanged, ".")
		//fmt.Printf("ConfigChangeHook:fieldnames: %v\n", fieldnames)
		configChangeRequestChan <- ConfigChangeInfo{configgroup, fieldnames[len(fieldnames)-1], fieldchanged, futvalue, curvalue, index}
	} else {
		//fmt.Printf("ConfigChangeHook:Config change chan is nil, unable to process change")
	}*/
	ConfigChangeHandler(configgroup, fieldchanged, futvalue, curvalue, index)

	return false //return false as we would apply only those we successfully processed
}

// ChangesComplete is called when all changes for a specific configgroup tagname
// If ChangesComplete returns true, then all changes in that group will be assigned to the current struct
func (cfgHook ChangeHook) ChangesComplete(configgroup string) (acceptallchanges bool) {
	//fmt.Printf("ConfigChangeHook:ChangesComplete: %s\n", configgroup)

	//inst := GetInstance()

	//jsstruct2, _ := json.Marshal(inst.logConfigFuture)
	//fmt.Printf("Applying: \n")
	//fmt.Printf("%s\n", string(jsstruct2))

	return false //return false as we would apply only those we successfully processed
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Helper functions
////////////////////////////////////////////////////////////////////////////////////////////////////

//AddLogFilter is the function to update the live running filters
func AddLogFilter(filterConfig maestroSpecs.LogFilter) error {
	//fmt.Printf("in AddLogFilter\n")
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
	//fmt.Printf("in DeleteLogFilter\n")
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
	//fmt.Printf("FileConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)

	splits := strings.Split(fieldchanged, ".")

	targetIndex, _ := strconv.Atoi(strings.TrimLeft(strings.TrimRight(splits[0], "]"), "Targets["))
	fieldName := splits[1]

	switch fieldName {
	case "File":
		inst.logConfigFuture[targetIndex].File = reflect.ValueOf(futvalue).String()
	default:
		log.MaestroWarnf("FileConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//NameConfigChange Function to process Target config change
func (inst *logManagerInstance) NameConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	//fmt.Printf("NameConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)

	splits := strings.Split(fieldchanged, ".")

	targetIndex, _ := strconv.Atoi(strings.TrimLeft(strings.TrimRight(splits[0], "]"), "Targets["))
	fieldName := splits[1]

	switch fieldName {
	case "Name":
		inst.logConfigFuture[targetIndex].Name = reflect.ValueOf(futvalue).String()
	default:
		log.MaestroWarnf("NameConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//FiltersConfigChange Function to process filter config change
func (inst *logManagerInstance) FiltersConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	//fmt.Printf("FiltersConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)

	//jsstruct2, _ := json.Marshal(inst.logConfigRunning)
	//fmt.Printf("Running: \n")
	//fmt.Printf("%s\n", string(jsstruct2))

	//jsstruct3, _ := json.Marshal(inst.logConfigFuture)
	//fmt.Printf("future before: \n")
	//fmt.Printf("%s\n", string(jsstruct3))

	//fmt.Printf("typeof=%s\n", reflect.TypeOf(futvalue))

	//pull our indexes out of the field name since we have 2
	splits := strings.Split(fieldchanged, ".")

	for i := 0; i < len(splits); i++ {
		//fmt.Printf("index: %d value: %s\n", i, splits[i])
	}
	targetIndex, _ := strconv.Atoi(strings.TrimLeft(strings.TrimRight(splits[0], "]"), "Targets["))

	filterIndex, _ := strconv.Atoi(strings.TrimLeft(strings.TrimRight(splits[1], "]"), "Filters["))

	//if we get a struct we can just set it in the filter value
	if reflect.TypeOf(futvalue).String() == "[]maestroSpecs.LogFilter" {
		filters := futvalue.([]maestroSpecs.LogFilter)

		//jsstruct, _ := json.Marshal(filters)
		//fmt.Printf("new: \n")
		//fmt.Printf("%s\n", string(jsstruct))

		inst.logConfigFuture[targetIndex].Filters = filters

	} else { // we need to set each field and parse it apart

		fieldName := ""

		//extract the field name from the nastiness
		if len(splits) > 2 {
			fieldName = splits[2]
		}

		//fmt.Printf("targetIndex=%d\n", targetIndex)
		//fmt.Printf("filterIndex=%d\n", filterIndex)
		//fmt.Printf("fieldName=%s\n", fieldName)
		//fmt.Printf("inst.logConfigFuture length=%d\n", len(inst.logConfigFuture))
		//fmt.Printf("inst.logConfigFuture[targetIndex].Filters length=%d\n", len(inst.logConfigFuture[targetIndex].Filters))

		//if we don't have any targets
		if inst.logConfigFuture == nil {
			inst.logConfigFuture = make([]maestroSpecs.LogTarget, 1)
		}

		//if we need a new target
		if targetIndex >= len(inst.logConfigFuture) {
			inst.logConfigFuture = append(inst.logConfigFuture, maestroSpecs.LogTarget{})
		}

		if inst.logConfigFuture[targetIndex].Filters == nil {
			inst.logConfigFuture[targetIndex].Filters = make([]maestroSpecs.LogFilter, 1)

			if filterIndex < len(inst.logConfigOriginal[targetIndex].Filters) {
				inst.logConfigFuture[targetIndex].Filters[filterIndex] = inst.logConfigOriginal[targetIndex].Filters[filterIndex]
			}
		}

		if filterIndex >= len(inst.logConfigFuture[targetIndex].Filters) {
			inst.logConfigFuture[targetIndex].Filters = append(inst.logConfigFuture[targetIndex].Filters, maestroSpecs.LogFilter{})

			if filterIndex < len(inst.logConfigOriginal[targetIndex].Filters) {
				inst.logConfigFuture[targetIndex].Filters[filterIndex] = inst.logConfigOriginal[targetIndex].Filters[filterIndex]
			}
		}

		//fmt.Printf("running file name=%s\n", inst.logConfigOriginal[targetIndex].File)
		inst.logConfigFuture[targetIndex].File = inst.logConfigOriginal[targetIndex].File

		switch fieldName {
		case "Target":
			inst.logConfigFuture[targetIndex].Filters[filterIndex].Target = reflect.ValueOf(futvalue).String()

			//this is mickey mouse as hell, but what this routine expects me to have a copy of an existing filter key
			/*if targetIndex < len(inst.logConfigRunning) {
				if filterIndex < len(inst.logConfigRunning[targetIndex].Filters) {
					inst.logConfigFuture[targetIndex].Filters[filterIndex].Levels = inst.logConfigRunning[targetIndex].Filters[filterIndex].Levels
					inst.logConfigFuture[targetIndex].Filters[filterIndex].Pre = inst.logConfigRunning[targetIndex].Filters[filterIndex].Pre
					inst.logConfigFuture[targetIndex].Filters[filterIndex].Post = inst.logConfigRunning[targetIndex].Filters[filterIndex].Post
				}
			}*/
		case "Levels":
			inst.logConfigFuture[targetIndex].Filters[filterIndex].Levels = reflect.ValueOf(futvalue).String()
		case "Pre":
			inst.logConfigFuture[targetIndex].Filters[filterIndex].Pre = reflect.ValueOf(futvalue).String()
		case "Post":
			inst.logConfigFuture[targetIndex].Filters[filterIndex].Post = reflect.ValueOf(futvalue).String()
		default:
			log.MaestroWarnf("FiltersConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
		}

		//jsstruct, _ := json.Marshal(inst.logConfigFuture)
		//fmt.Printf("future after: \n")
		//fmt.Printf("%s\n", string(jsstruct))

	}
}

//ApplyChanges this make the filter changes active
func (inst *logManagerInstance) ApplyChanges() {

	if inst.logConfigFuture == nil {
		inst.logConfigFuture = make([]maestroSpecs.LogTarget, 1)
	}

	//fmt.Printf("Entering ApplyChanges()\n")
	filterslen := len(inst.logConfigFuture[0].Filters)

	//get the target id
	targID := greasego.GetTargetId(inst.logConfigFuture[0].File)

	//make sure its legit
	if targID == 0 {
		//fmt.Printf("FiltersConfigChange: Target ID not found for Target Name: %s\n", inst.logConfigFuture[0].Name)
		return
	}

	//fmt.Printf("targId=%d\n", targID)

	// if the existing target at index has a name and if the incoming filter specifies a target name,
	// then verify the names match
	if len(inst.logConfigFuture[0].Name) > 0 {
		for i := 0; i < filterslen; i++ {
			targetName := inst.logConfigFuture[0].Filters[i].Target
			if len(targetName) > 0 && (inst.logConfigFuture[0].Name != targetName) {
				//fmt.Printf("FiltersConfigChange: filter.Target.Name doesn't match existing target at index 0\n")
				return
			}
		}
	}

	/////////////////////////////////////////////////////////////
	// We need to get all the log targets so we can modify them
	/////////////////////////////////////////////////////////////
	LogTargets := make([]maestroSpecs.LogTarget, 0)

	targets, filters := greasego.GetAllTargetsAndFilters()

	// add all existing targets
	for _, target := range targets {
		var t maestroSpecs.LogTarget
		if target.Name != nil {
			t.Name = *target.Name
		}
		if target.File != nil {
			t.File = *target.File
		}
		if target.TTY != nil {
			t.TTY = *target.TTY
		}
		LogTargets = append(LogTargets, t)
	}

	// backfill the filters
	for _, filter := range filters {
		targetname := greasego.GetTargetName(filter.Target)
		for i, target := range LogTargets {
			if target.Name == targetname {
				var f maestroSpecs.LogFilter
				f.Target = target.Name
				f.Tag = maestroConfig.ConvertTagUint32ToString(filter.Tag)
				f.Levels = maestroConfig.ConvertLevelUint32MaskToString(filter.Mask)
				LogTargets[i].Filters = append(LogTargets[i].Filters, f)
			}
		}
	}

	////////////////////////////////////////////////////////////////////
	// Now that we have the filters we need to choosse to do a replace
	// or override
	//
	// replace   = destroy all filter in the target and replace
	//             them with the new one
	// override  = only modify the differences in the targets filters
	////////////////////////////////////////////////////////////////////
	if inst.logConfigFuture[0].Existing == "override" {
		//fmt.Printf("inside override\n")
		// For override we need to walk all of our existing filters and add
		// the missing elements
		for _, filter := range inst.logConfigFuture[0].Filters {
			DeleteLogFilter(filter)
		}
		// For "replace" behavior, we delete all existing filters for the given target
		for x := 0; x < len(LogTargets); x++ {
			if LogTargets[x].File == inst.logConfigFuture[0].File {
				for y := 0; y < len(LogTargets[x].Filters); y++ {
					for z := 0; z < len(inst.logConfigFuture[0].Filters); z++ {
						if LogTargets[x].Filters[y].Levels == inst.logConfigFuture[0].Filters[z].Levels {
							newFilter := LogTargets[x].Filters[y]
							if inst.logConfigFuture[0].Filters[z].Target != "" {
								newFilter.Target = inst.logConfigFuture[0].Filters[z].Target
							}
							if inst.logConfigFuture[0].Filters[z].Tag != "" {
								newFilter.Tag = inst.logConfigFuture[0].Filters[z].Tag
							}
							if inst.logConfigFuture[0].Filters[z].Pre != "" {
								newFilter.Pre = inst.logConfigFuture[0].Filters[z].Pre
							}
							if inst.logConfigFuture[0].Filters[z].Post != "" {
								newFilter.Post = inst.logConfigFuture[0].Filters[z].Post
							}
							if inst.logConfigFuture[0].Filters[z].PostFmtPreMsg != "" {
								newFilter.PostFmtPreMsg = inst.logConfigFuture[0].Filters[z].PostFmtPreMsg
							}
							//we now set oun new filter back
							inst.logConfigFuture[0].Filters[z] = newFilter

							//we need to kill it so we can add it back in the end
							DeleteLogFilter(inst.logConfigFuture[0].Filters[z])
						}
					}
				}
			}
		}
	} else {
		//fmt.Printf("inside replace\n")
		// For "replace" behavior, we delete all existing filters for the given target
		for w := 0; w < len(inst.logConfigFuture); w++ {
			for x := 0; x < len(LogTargets); x++ {
				if LogTargets[x].File == inst.logConfigFuture[w].File {
					//fmt.Printf("Delete file=%s filters\n", LogTargets[x].File)
					for y := 0; y < len(LogTargets[x].Filters); y++ {
						DeleteLogFilter(LogTargets[x].Filters[y])
						//fmt.Printf("Deleted: %s\n", LogTargets[x].Filters[y].Levels)
						//LogTargets[x].Filters[y] = maestroSpecs.LogFilter{}
					}
				}
			}
		}
	}

	//fmt.Printf("Performed: %s\n", inst.logConfigFuture[0].Existing)

	// now that we have override or replace setup we just add all the filters back
	for x := 0; x < len(inst.logConfigFuture); x++ {
		for y := 0; y < filterslen; y++ {
			// configure greasego
			AddLogFilter(inst.logConfigFuture[x].Filters[y])
			//fmt.Printf("Added: %s\n", inst.logConfigFuture[x].Filters[y].Levels)
		}
	}
	//store the config back to the database
	inst.submitConfig(inst.logConfigFuture)
	//last but not least clear the array now that this has been called
	inst.logConfigRunning = inst.logConfigFuture
	inst.logConfigFuture = make([]maestroSpecs.LogTarget, 1)
}

//FormatConfigChange Function to process format config change
func (inst *logManagerInstance) FormatConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	//fmt.Printf("FormatConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
}

//OptsConfigChange Function to process opts config change
func (inst *logManagerInstance) OptsConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	//fmt.Printf("OptsConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)

	inst.logConfigFuture[index].Existing = reflect.ValueOf(futvalue).String()
	//fmt.Printf("existing=%s\n", inst.logConfigFuture[index].Existing)
}

//////////////////////////////////////////////////////////////////////////////////////////
// Monitor for ConfigChangeHook
//////////////////////////////////////////////////////////////////////////////////////////

// CommitConfigChangeHook is the handler to commit and apply changes
type CommitConfigChangeHook struct {
}

// ChangesStart is called before reporting any changes via multiple calls to SawChange. It will only be called
// if there is at least one change to report
func (cfgHook CommitConfigChangeHook) ChangesStart(configgroup string) {
	//fmt.Printf("CommitChangeHook:ChangesStart: %s\n", configgroup)

	inst := GetInstance()
	//only apply this after getting all the filters
	//fmt.Printf("In config group if\n")

	//jsstruct, _ := json.Marshal(inst.logConfigRunning)
	//fmt.Printf("Running: \n")
	//fmt.Printf("%s\n", string(jsstruct))

	//jsstruct2, _ := json.Marshal(inst.logConfigFuture)
	//fmt.Printf("Applying: \n")
	//fmt.Printf("%s\n", string(jsstruct2))

	//here is where we will add our filters
	inst.ApplyChanges()

	inst.logConfigFuture = make([]maestroSpecs.LogTarget, 1)
	inst.CurrConfigCommit.ConfigCommitFlag = false
}

// SawChange is called whenever a field changes. It will be called only once for each field which is changed.
// It will always be called after ChangesStart is called
// If SawChange return true, then the value of futvalue will replace the value of current value
func (cfgHook CommitConfigChangeHook) SawChange(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) (acceptchange bool) {
	//fmt.Printf("CommitChangeHook:SawChange: %s:%s old:%v new:%v index:%d\n", configgroup, fieldchanged, curvalue, futvalue, index)

	return false //return false as we would apply only those we successfully processed
}

// ChangesComplete is called when all changes for a specific configgroup tagname
// If ChangesComplete returns true, then all changes in that group will be assigned to the current struct
func (cfgHook CommitConfigChangeHook) ChangesComplete(configgroup string) (acceptallchanges bool) {
	//fmt.Printf("CommitChangeHook:ChangesComplete: %s\n", configgroup)
	return false //return false as we would apply only those we successfully processed
}
