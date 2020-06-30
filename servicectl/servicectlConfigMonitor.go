package servicectl

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
	"fmt"

	"github.com/armPelionEdge/maestro/log"
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

	//fmt.Printf("ConfigChangeHandler:: group:%s field:%s old:%v new:%v\n", configgroup, fieldchanged, curvalue, futvalue)
	switch configgroup {
	case "servicectl":
		inst.ServicectlConfigChange(fieldchanged, futvalue, curvalue, index)
	default:
		//fmt.Printf("\nConfigChangeHook:Unknown field or group: %s:%s old:%v new:%v\n", configgroup, fieldchanged, curvalue, futvalue)
	}
}

// ChangesStart is called before reporting any changes via multiple calls to SawChange. It will only be called
// if there is at least one change to report
func (cfgHook ChangeHook) ChangesStart(configgroup string) {
	fmt.Printf("ConfigChangeHook:ChangesStart: %s\n", configgroup)
	// FIXME!
	// clear all the changes here? elsewhere?
	inst := GetInstance()
	for _, service := range inst.servicectlConfigRunning {
		service := service
		inst.servicectlConfig.Services = append(inst.servicectlConfig.Services, service)
	}
}

// SawChange is called whenever a field changes. It will be called only once for each field which is changed.
// It will always be called after ChangesStart is called
// If SawChange return true, then the value of futvalue will replace the value of current value
func (cfgHook ChangeHook) SawChange(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) (acceptchange bool) {
	fmt.Printf("ConfigChangeHook:SawChange: %s:%s old:%v new:%v index:%d\n", configgroup, fieldchanged, curvalue, futvalue, index)
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
	fmt.Printf("ConfigChangeHook:ChangesComplete: %s\n", configgroup)

	//inst := GetInstance()

	//jsstruct2, _ := json.Marshal(inst.logConfigFuture)
	//fmt.Printf("Applying: \n")
	//fmt.Printf("%s\n", string(jsstruct2))
	inst := GetInstance()
	inst.SetAllServiceConfig(inst.servicectlConfig.Services)


	return false //return false as we would apply only those we successfully processed
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Helper functions
////////////////////////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////////////////////////
// Functions to process the parameters which are changed
////////////////////////////////////////////////////////////////////////////////////////////////////

//Function to process servicetl config change
func (inst *ServicectlManagerInstance) ServicectlConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	fmt.Printf("NameConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)

	switch fieldchanged {
	case "Name":
		inst.servicectlConfig.Services[index].Name = reflect.ValueOf(futvalue).String()
	case "StatusUpdatePeriod":
		inst.servicectlConfig.Services[index].StatusUpdatePeriod = uint64(reflect.ValueOf(futvalue).Uint())
	case "StartService":
		inst.servicectlConfig.Services[index].StartService = reflect.ValueOf(futvalue).Bool()
	case "RestartService":
		inst.servicectlConfig.Services[index].RestartService = reflect.ValueOf(futvalue).Bool()
	case "StopService":
		inst.servicectlConfig.Services[index].StopService = reflect.ValueOf(futvalue).Bool()
	case "Enable":
		inst.servicectlConfig.Services[index].Enable = reflect.ValueOf(futvalue).Bool()
	case "StartonBoot":
		inst.servicectlConfig.Services[index].StartonBoot = reflect.ValueOf(futvalue).Bool()
	default:
		log.MaestroWarnf("NameConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}







