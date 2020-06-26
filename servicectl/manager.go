package servicectl

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/armPelionEdge/greasego"
	"github.com/armPelionEdge/maestro/debugging"
	"github.com/armPelionEdge/maestro/defaults"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/maestroConfig"
	"github.com/armPelionEdge/maestro/storage"
	"github.com/armPelionEdge/maestro/wwrmi"
	"github.com/armPelionEdge/maestroSpecs"
	"github.com/armPelionEdge/stow"
	"github.com/boltdb/bolt"
	"github.com/coreos/go-systemd/dbus"
)

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

/**
 * This is the log manager.
 *
 * It is responsible for
 * - setting up the host's log interface
 * - monitoring the state of log interfaces
 * - doing specific things on log up / down
 * - running an mDNS server, to broadcast certain things about the gateway
 * - maintaining the state of interfaces in the storage subsystem
 */

const (
	//LogPrefix is the added output is logging for this file
	ServicectlPrefix = "ServicevctlManager: "
)



const (
	cDbName = "servicectlConfig"
)


//ServicectlManagerInstance initialized in InitServicectlManager()
type ServicectlManagerInstance struct {
	// for persistence
	db          *bolt.DB
	servicectlConfigDB *stow.Store

	// current config state
	servicectlConfigRunning  []maestroSpecs.Service
	servicectlConfig         *maestroSpecs.ServiceCtlConfigPayload
	CurrConfigCommit  ConfigCommit

	// Configs to be used for connecting to devicedb
	ddbConnConfig   *maestroConfig.DeviceDBConnConfig
	ddbConfigClient *maestroConfig.DDBRelayConfigClient
}

// ConfigCommit is the structure to hold commit change applications
type ConfigCommit struct {
	// Set this flag to true for the changes to commit, if this flag is false
	// the changes to configuration on these structs will not acted upon
	// by network manager. For exmaple, this flag will be initially false
	// so that user can change the config object in DeviceDB and verify that the
	// intented changes are captured correctly. Once verified set this flag to true
	// so that the changes will be applied by maestro. Once maestro complete the
	// changes the flag will be set to false by maestro.
	ConfigCommitFlag bool `yaml:"config_commit" json:"config_commit" log_group:"config_commit"`
	//Datetime of last update
	LastUpdateTimestamp string `yaml:"config_commit" json:"last_commit_timestamp" log_group:"config_commit"`
	//Total number of updates from boot
	TotalCommitCountFromBoot int `yaml:"config_commit" json:"total_commit_count_from_boot" log_group:"config_commit"`
}

//DDBServicectlConfigName name used in log writes
const DDBServicectlConfigName string = "MAESTRO_SERVICECTL_CONFIG_ID"

//DDBServicectlConfigCommit the ID used to commit changes
const DDBServicectlConfigCommit string = "MAESTRO_SERVICECTL_CONFIG_COMMIT_FLAG"

//DDBServicectlConfigGroupID used in log writes
const DDBServicectlConfigGroupID string = "servicectl_group"

var instance *ServicectlManagerInstance

// impl storage.StorageUser interface
func (servicectlManager *ServicectlManagerInstance) StorageInit(instance storage.MaestroDBStorageInterface) {
	gob.Register(&maestroSpecs.Service{})
	servicectlManager.servicectlConfigDB = stow.NewStore(instance.GetDb(), []byte(cDbName))
}

func (servicectlManager *ServicectlManagerInstance) StorageReady(instance storage.MaestroDBStorageInterface) {
	servicectlManager.db = instance.GetDb()
}

func (servicectlManager *ServicectlManagerInstance) StorageClosed(instance storage.MaestroDBStorageInterface) {

}

func newServicectlManagerInstance() (ret *ServicectlManagerInstance) {
	ret = new(ServicectlManagerInstance)

	return
}

//GetInstance get the global instance of the servicectl manager
func GetInstance() *ServicectlManagerInstance {
	if instance == nil {

		instance = newServicectlManagerInstance()

		instance.servicectlConfig = new(maestroSpecs.ServiceCtlConfigPayload)

		storage.RegisterStorageUser(instance)
		// internalTicker = time.NewTicker(time.Second * time.Duration(defaults.TASK_MANAGER_CLEAR_INTERVAL))
		// controlChan = make(chan *controlToken, 100) // buffer up to 100 events
	}
	return instance
}


func validateServicectlConfig(logconf maestroSpecs.Service) (ok bool, problem string) {
	log.MaestroInfof("in validateServicectlConfig\n")
	ok = true

	return
}


func (servicectlManager *ServicectlManagerInstance) submitConfigAndSync(config []maestroSpecs.Service) {
	log.MaestroInfof("in submitConfigAndSync\n")
	servicectlManager.submitConfig(config)
	var ddbServicectlConfig maestroSpecs.ServicectlConfigPayload

	for _, service := range config {
		service := service
		ddbServicectlConfig.Services = append(ddbServicectlConfig.Services, &service)
	}
	log.MaestroInfof("servicectlManager: Updating devicedb config.\n")
	err := servicectlManager.ddbConfigClient.Config(DDBServicectlConfigName).Put(&ddbServicectlConfig)
	if err != nil {
		log.MaestroDebugf("servicectlManager: Unable to put servicectl config in devicedb err: %v\n", err)
	}
	log.MaestroWarnf("servicectlManager: Setting Servicectl config commit flag to false\n")
	servicectlManager.CurrConfigCommit.ConfigCommitFlag = false
	servicectlManager.CurrConfigCommit.LastUpdateTimestamp = ""
	servicectlManager.CurrConfigCommit.TotalCommitCountFromBoot = servicectlManager.CurrConfigCommit.TotalCommitCountFromBoot + 1
	err = servicectlManager.ddbConfigClient.Config(DDBServicectlConfigCommit).Put(&servicectlManager.CurrConfigCommit)
	if err != nil {
		log.MaestroErrorf("servicectlManager: Unable to put Servicectl commit flag in devicedb err:%v, config will not be monitored from devicedb\n", err)
	}

}

func (servicectlManager *ServicectlManagerInstance) submitConfig(config []maestroSpecs.Service) {
	log.MaestroInfof("in submitConfig\n")
	servicectlManager.servicectlConfigRunning = config

	debugging.DEBUG_OUT("services:", len(config))
	for n := 0; n < len(config); n++ {
		confOk, problem := validateServicectlConfig(config[n])
		if !confOk {
			log.MaestroDebugf("servicectlManager: Service config problem: \"%s\"  Skipping service config.\n", problem)
			continue
		}

		jsstruct, _ := json.Marshal(config[n])
		log.MaestroInfof("saving: \n")
		log.MaestroInfof("%s\n", string(jsstruct))

		servicename := config[n].Name

		log.MaestroInfof("servicename=%s\n", servicename)

		var storedconfig maestroSpecs.Service
		err := servicectlManager.servicectlConfigDB.Get(servicename, &storedconfig)
		if err != nil {
			log.MaestroInfof("servicectlManager: sumbitConfig: Get failed: %v\n", err)
			if err != stow.ErrNotFound {
				log.MaestroErrorf("servicectlManager: problem with database on if %s - Get: %s\n", servicename, err.Error())
			} else {
				//write this out to the DB
				err := servicectlManager.servicectlConfigDB.Put(servicename, config[n])
				if err != nil {
					log.MaestroErrorf("servicectlManager: Problem (1) storing config for interface [%s]: %s\n", config[n].Name, err)
				} else {
					ResetServiceCtl(config[n])
				}
			}
		} else {
			// entry existed, so override
		//	if config[n].Existing == "replace" || config[n].Existing == "override" {
				//write this out to the DB
				err := servicectlManager.servicectlConfigDB.Put(servicename, config[n])
				if err != nil {
					log.MaestroErrorf("servicectlManager: Problem (2) storing config for interface [%s]: %s\n", config[n].Name, err)
				} else {
					log.MaestroInfof("servicectlManager: log target [%s] - setting \"replace\"\n", config[n].Name)
				}

		//	} else {
				// else do nothingm db has priority
		//	}
		}
	}
}
/*
//name of the service
Name                     string                           `yaml:"name,omitempty" json:"name" servicectl_group:"name"`
// enable:true is equivalent to "systemctl enable servicename", enable:false is equivalent to "systemctl disable servicename"
Enable				     bool							  `yaml:"enable,omitempty" json:"enable" servicectl_group:"settings"`
// shows the status of the service "systemctl status servicename"
Status	                 string                           `yaml:"status,omitempty" json:"status" servicectl_group:"status"`
// period to update the status parameter mentioned above
StatusUpdatePeriod       uint32                           `yaml:"status_update_period,omitempty" json:"status_update_period" servicectl_group:"status"`
// restart the service if set to true and set it back to false if restart successful.
RestartService           bool                             `yaml:"restart_service,omitempty" json:"restart_service" servicectl_group:"control"`
// start the service if set to true and set it back to false if start successful.
StartService             bool                             `yaml:"start_service,omitempty" json:"start_service" servicectl_group:"control"`
// stop the service if set to true and set it back to false if stop successful.
StopService              bool                             `yaml:"stop_service,omitempty" json:"stop_service" servicectl_group:"control"`
*/
//reset the systemd config for a particular service
func (servicectlManager *ServicectlManagerInstance) SetAllServiceConfig(services []maestroSpecs.Service) {
	systemd, err := dbus.NewSystemdConnection()
	if err != nil {
		log.MaestroErrorf("servicectlManager: Error establishing connection to systemd %v\n", err)
		return
	}
	for n := 0; n < len(services); n++ {
		var str []string
		str[0] = services[n].Name
		if(services[n].Enable) {
			enablement_info, EnableUnitFileChange, err := systemd.EnableUnitFiles(str, false, false)
			log.MaestroInfof("Enabling service %s  Results: %v , %v, %v",services[n].Name, enablement_info, EnableUnitFileChange, err)
			fmt.Printf("%v , %v, %v", enablement_info, EnableUnitFileChange, err)
			services[n].IsEnabled = true
		} else {
			DisableUnitFileChange, err := systemd.DisableUnitFiles(str, false)
			fmt.Printf("%v, %v", DisableUnitFileChange, err)
			services[n].IsEnabled = false
		}

		if(services[n].StartedonBoot == false && services[n].StartonBoot == true) {
			id, err := systemd.StartUnit(services[n].Name, "replace", nil)
			if err != nil {
				log.MaestroErrorf("servicectlManager: Error while starting service %s : %v", services[n].Name, err)
			} else {
				log.MaestroInfof("Started service %s with PID %v",services[n].Name, id)
			}
		}

		if(services[n].StartService) {
			id, err := systemd.StartUnit(services[n].Name, "replace", nil)
			if err != nil {
				log.MaestroErrorf("servicectlManager: Error while starting service %s : %v", services[n].Name, err)
			} else {
				log.MaestroInfof("Started service %s with PID %v",services[n].Name, id)
				services[n].StartService = false
			}
			
		}

		if(services[n].StopService) {
			id, err := systemd.StopUnit(services[n].Name, "replace", nil)
			fmt.Printf("%v %v ", id, err)
			if err != nil {
				log.MaestroErrorf("servicectlManager: Error while stopping service %s : %v", services[n].Name, err)
			} else {
				log.MaestroInfof("Stopped service %s with PID %v",services[n].Name, id)
				services[n].StopService = false
			}
			
		}
		
		if(services[n].RestartService) {
			id, err := systemd.RestartUnit(services[n].Name, "replace", nil)
			fmt.Printf("%v %v ", id, err)
			if err != nil {
				log.MaestroErrorf("servicectlManager: Error  while restarting service %s : %v", services[n].Name, err)
			} else {
				log.MaestroInfof("Restarted service %s with PID %v",services[n].Name, id)
				services[n].RestartService = false
			}
			
		}
		ServiceState, err := systemd.GetUnitProperty(services[n].Name, "SubState")
		services[n].Status = ServiceState.Value.Value().(string)
		log.MaestroInfof("Current Status of the service %s : %s",services[n].Name, services[n].Status)
		if(services[n].Status == "running") {
			services[n].IsRunning = true
		} else {
			services[n].IsRunning = false
		}
	}
	systemd.Close()
	//saving updated config info and service info to local database
	servicectlManager.submitConfigAndSync(services)

}


//This function is called during bootup. It waits for devicedb to be up and running to connect to it, once connected it calls
//SetupDeviceDBConfig
func (servicectlManager *ServicectlManagerInstance) initDeviceDBConfig() error {
	var err error

	servicectlManager.ddbConfigClient, err = maestroConfig.GetDDBRelayConfigClient(servicectlManager.ddbConnConfig)
	if servicectlManager.ddbConfigClient == nil {
		return err
	}
	log.MaestroInfof("servicectlManager: successfully connected to devicedb\n")

	err = servicectlManager.SetupDeviceDBConfig()
	if err != nil {
		log.MaestroErrorf("servicectlManager: error setting up config using devicedb: %v", err)
	} else {
		log.MaestroInfof("servicectlManager: successfully read config from devicedb\n")
	}

	return err
}

//SetupDeviceDBConfig reads the config from devicedb and if its new it applies the new config.
//It also sets up the config update handlers for all the tags/groups.
func (servicectlManager *ServicectlManagerInstance) SetupDeviceDBConfig() (err error) {
	var ddbServicectlConfig maestroSpecs.ServicectlConfigPayload

	//Create a config analyzer object, required for registering the config change hook and diff the config objects.
	configServicectlAna := maestroSpecs.NewConfigAnalyzer(DDBServicectlConfigGroupID)
	if configServicectlAna == nil {
		log.MaestroDebugf("servicectlManager: Failed to create config analyzer object, unable to fetch config from devicedb")
		errUpdated := errors.New("Failed to create config analyzer object, unable to fetch config from devicedb")
		return errUpdated
	}

	err = servicectlManager.ddbConfigClient.Config(DDBServicectlConfigName).Get(&ddbServicectlConfig)
	if err != nil {
		for _, service := range servicectlManager.servicectlConfigRunning {
			service := service
			ddbServicectlConfig.Services = append(ddbServicectlConfig.Services, &service)
		}
		log.MaestroDebugf("servicectlManager: No servicectl config found in devicedb or unable to connect to devicedb err: %v. Let's put the current running config.\n", err)
		err = servicectlManager.ddbConfigClient.Config(DDBServicectlConfigName).Put(&ddbServicectlConfig)
		if err != nil {
			log.MaestroDebugf("servicectlManager: Unable to put servicectl config in devicedb err:%v, config will not be monitored from devicedb\n", err)
			errUpdated := errors.New(fmt.Sprintf("\nUnable to put servicectl config in devicedb err:%v, config will not be monitored from devicedb\n", err))
			return errUpdated
		}
	} else {
		//We found a config in devicedb, lets try to use and reconfigure servicectl if its an updated one
		log.MaestroInfof("servicectlManager: Found a valid config in devicedb [%v], will try to use and reconfigure servicectl if its an updated one\n", ddbServicectlConfig.Services)
		var temp maestroSpecs.ServicectlConfigPayload
		for _, service := range servicectlManager.servicectlConfigRunning {
			service := service
			temp.Services = append(temp.Services, &service)
		}
		identical, _, _, err := configServicectlAna.DiffChanges(temp, ddbServicectlConfig)
		if !identical && (err == nil) {
			//The configs are different, lets go ahead reconfigure the intfs
			log.MaestroInfof("servicectlManager: New servicectl config found from devicedb, reconfigure servicectl using new config\n")
			servicectlManager.servicectlConfigRunning = make([]maestroSpecs.Service, len(ddbServicectlConfig.Services))
			for i, service := range ddbServicectlConfig.Services {
				servicectlManager.servicectlConfigRunning[i] = *service
			}
			//setup the services using the config
			servicectlManager.SetAllServiceConfig(servicectlManager.servicectlConfigRunning)
			// this.setupTargets()
		} else {
			log.MaestroInfof("servicectlManager: New servicectl config found from devicedb, but its same as boot config, no need to re-configure\n")
		}
	}

	//Now start a monitor for the servicectl config in devicedb
	//Add config change hook for all property groups, we can use the same interface
	var servicectlConfigChangeHook ChangeHook

	//servicectl groups
	configServicectlAna.AddHook("servicectl", servicectlConfigChangeHook)

	//Add monitor for this config
	var origServicectlConfig, updatedServicectlConfig maestroSpecs.ServicectlConfigPayload

	//Provide a copy of current servicectl config monitor to Config monitor, not the actual config we use, this would prevent config monitor
	//directly updating the running config(this.servicectlConfigRunning).
	for _, service := range servicectlManager.servicectlConfigRunning {
		service := service
		origServicectlConfig.Services = append(origServicectlConfig.Service, &service)
	}

	log.MaestroInfof("adding servicectl monitor config\n")

	//Adding monitor config
	servicectlManager.ddbConfigClient.AddMonitorConfig(&origServicectlConfig, &updatedServicectlConfig, DDBServicectlConfigName, configServicectlAna)

	return
}


// InitServicectlManager be called on startup.
// ServicectlConfig will come from config file
// Storage should be started already.
func InitServicectlManager(config *maestroConfig.YAMLMaestroConfig) (err error) {

	inst := GetInstance()
	inst.ddbConnConfig = config.DDBConnConfig
		
	//read servicectl config from config file
	NoOfServices := len(config.Services)
	
	for _, service := range config.Services {
		service := service
		inst.servicectlConfig.Services = append(ddbServicectlConfig.Services, &service)
	}


	//setup the services using the config
	inst.SetAllServiceConfig(inst.servicectlConfig.Services)

	log.MaestroInfof("servicectlManager: Initializing %v %v\n", inst.servicectlConfigRunning, inst.ddbConnConfig)

	log.MaestroInfof("init device db config\n")
	go inst.initDeviceDBConfig()

	return
}
