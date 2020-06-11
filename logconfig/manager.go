package logconfig

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

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
	LogPrefix = "LogManager: "
)

//LogData is the internal wrapper, used to store log settings
type LogData struct {
	// the settings as they should be
	StoredLogconfig []maestroSpecs.LogTarget
	// the settings as they are right now
	RunningLogconfig []maestroSpecs.LogTarget

	// last time the log config was changed
	// ms since Unix epoch
	LastUpdated int64
	// updated everytime we see a netlink message about the interface
	lastFlags uint32
	flagsSet  bool
}

//MarshalJSON is a helper function to load json data
func (logdata *LogData) MarshalJSON() ([]byte, error) {
	type Alias LogData

	return json.Marshal(&struct {
		//		LastSeen int64 `json:"lastSeen"`
		*Alias
	}{
		//		LastSeen: u.LastSeen.Unix(),
		Alias: (*Alias)(logdata),
	})
}

func (logdata *LogData) hasFlags() bool {
	return logdata.flagsSet
}

func (logdata *LogData) setFlags(v uint32) {
	logdata.flagsSet = true
	logdata.lastFlags = v
}

const (
	cDbName = "logConfig"
)

type logThreadMessage struct {
	cmd     int
	logname string // sometimes used
}

//logManagerInstance initialized in InitLogManager()
type logManagerInstance struct {
	// for persistence
	db          *bolt.DB
	logConfigDB *stow.Store

	byLogName   sync.Map // map of identifier to struct -> 'eth2':&networkInterfaceData{}
	indexToName sync.Map

	watcherWorkChannel chan logThreadMessage

	// mostly used for testing
	threadCountEnable         bool
	interfaceThreadCount      int
	interfaceThreadCountMutex sync.Mutex
	threadCountChan           chan logThreadMessage

	// current config state
	logConfigRunning  []maestroSpecs.LogTarget
	logConfigFuture   []maestroSpecs.LogTarget
	logConfigOriginal []maestroSpecs.LogTarget
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

//DDBLogConfigName name used in log writes
const DDBLogConfigName string = "MAESTRO_LOG_CONFIG_ID"

//DDBLogConfigCommit the ID used to commit changes
const DDBLogConfigCommit string = "MAESTRO_LOG_CONFIG_COMMIT_FLAG"

//DDBLogConfigGroupID used in log writes
const DDBLogConfigGroupID string = "log_group"

var instance *logManagerInstance

// impl storage.StorageUser interface
func (logManager *logManagerInstance) StorageInit(instance storage.MaestroDBStorageInterface) {
	gob.Register(&maestroSpecs.LogTarget{})
	gob.Register(&LogData{})
	logManager.logConfigDB = stow.NewStore(instance.GetDb(), []byte(cDbName))
}

func (logManager *logManagerInstance) StorageReady(instance storage.MaestroDBStorageInterface) {
	logManager.db = instance.GetDb()
	err := logManager.loadAllLogData()
	if err != nil {
		log.MaestroDebugf("LogManager: Failed to load existing log interface settings: %s\n", err.Error())
	}
}

func (logManager *logManagerInstance) StorageClosed(instance storage.MaestroDBStorageInterface) {

}

func newLogManagerInstance() (ret *logManagerInstance) {
	ret = new(logManagerInstance)
	ret.watcherWorkChannel = make(chan logThreadMessage, 10) // use a buffered channel
	ret.threadCountChan = make(chan logThreadMessage)

	return
}

//GetInstance get the global instance of the log manager
func GetInstance() *logManagerInstance {
	if instance == nil {

		instance = newLogManagerInstance()
		// provide a blank log config for now (until one is provided)
		//instance.logConfigRunning = new([]maestroSpecs.LogTarget)

		storage.RegisterStorageUser(instance)
		// internalTicker = time.NewTicker(time.Second * time.Duration(defaults.TASK_MANAGER_CLEAR_INTERVAL))
		// controlChan = make(chan *controlToken, 100) // buffer up to 100 events
	}
	return instance
}

func (logdata *LogData) setRunningConfig(logconfig []maestroSpecs.LogTarget) (ret *LogData) {
	log.MaestroInfof("in setRunningConfig\n")

	ret.RunningLogconfig = logconfig // copy that entire struct
	return
}

func validateLogConfig(logconf maestroSpecs.LogTarget) (ok bool, problem string) {
	log.MaestroInfof("in validateLogConfig\n")
	ok = true

	return
}

// to be ran on startup, or new interface creation
//
func resetLogDataFromStoredConfig(logtarget maestroSpecs.LogTarget) error {
	log.MaestroInfof("in resetLogDataFromStoredConfig\n")

	return nil
}

func (logManager *logManagerInstance) submitConfigAndSync(config []maestroSpecs.LogTarget) {
	log.MaestroInfof("in submitConfigAndSync\n")
	logManager.submitConfig(config)
	var ddbLogConfig maestroSpecs.LogConfigPayload

	for _, target := range config {
		target := target
		ddbLogConfig.Targets = append(ddbLogConfig.Targets, &target)
	}
	log.MaestroInfof("LogManager: Updating devicedb config.\n")
	err := logManager.ddbConfigClient.Config(DDBLogConfigName).Put(&ddbLogConfig)
	if err != nil {
		log.MaestroDebugf("LogManager: Unable to put log config in devicedb err: %v\n", err)
	}
}

func (logManager *logManagerInstance) submitConfig(config []maestroSpecs.LogTarget) {
	log.MaestroInfof("in submitConfig\n")
	logManager.logConfigRunning = config

	debugging.DEBUG_OUT("targets:", len(config))
	for n := 0; n < len(config); n++ {
		confOk, problem := validateLogConfig(config[n])
		if !confOk {
			log.MaestroDebugf("LogManager: Target config problem: \"%s\"  Skipping interface config.\n", problem)
			continue
		}

		jsstruct, _ := json.Marshal(config[n])
		log.MaestroInfof("saving: \n")
		log.MaestroInfof("%s\n", string(jsstruct))

		targetname := config[n].File

		log.MaestroInfof("targetname=%s\n", targetname)

		var storedconfig maestroSpecs.LogTarget
		err := logManager.logConfigDB.Get(targetname, &storedconfig)
		if err != nil {
			log.MaestroInfof("LogManager: sumbitConfig: Get failed: %v\n", err)
			if err != stow.ErrNotFound {
				log.MaestroErrorf("LogManager: problem with database on if %s - Get: %s\n", targetname, err.Error())
			} else {
				//cache our local copy. Do we need to do this anymore?
				//logManager.byLogName.Set(targetname, unsafe.Pointer(&config[n]))
				//write this out to the DB
				err := logManager.logConfigDB.Put(targetname, config[n])
				if err != nil {
					log.MaestroErrorf("LogManager: Problem (1) storing config for interface [%s]: %s\n", config[n].Name, err)
				}
			}
		} else {
			// entry existed, so override
			if config[n].Existing == "replace" || config[n].Existing == "override" {
				// do i need to do this?
				//logManager.byLogName.Set(targetname, unsafe.Pointer(&config[n]))
				//write this out to the DB
				err := logManager.logConfigDB.Put(targetname, config[n])
				if err != nil {
					log.MaestroErrorf("LogManager: Problem (2) storing config for interface [%s]: %s\n", config[n].Name, err)
				} else {
					log.MaestroInfof("LogManager: log target [%s] - setting \"replace\"\n", config[n].Name)
				}

			} else {
				// else do nothingm db has priority
			}
		}
	}
}

// loads all existing data from the DB, for an interface
// If one or more interfaces data have problems, it will keep loading
// If reading the DB fails completely, it will error out
func (logManager *logManagerInstance) loadAllLogData() (err error) {
	log.MaestroInfof("in loadAllLogData\n")
	var temp LogData
	logManager.logConfigDB.IterateIf(func(key []byte, val interface{}) bool {
		logname := string(key[:])
		logdata, ok := val.(maestroSpecs.LogTarget)
		if ok {
			//this probably isn't even remotely necessary
			err2 := resetLogDataFromStoredConfig(logdata)
			if err2 == nil {
				//this.newInterfaceMutex.Lock()
				// if there is an existing in-memory entry, overwrite it
				log.MaestroInfof("LogManager:loadAllLogData: Loading config for: %s.\n", logname)
				logManager.byLogName.Store(logname, &logdata)
				//this.newInterfaceMutex.Unlock()
				debugging.DEBUG_OUT("loadAllLogData() see if: %s --> %+v\n", logname, logdata)
			} else {
				log.MaestroDebugf("LogManager: Critical problem with interface [%s] config. Not loading config.\n", logname)
			}
		} else {
			err = errors.New("Internal DB corruption")
			debugging.DEBUG_OUT("LogManager: internal DB corruption - @if %s\n", logname)
			log.MaestroDebugf("LogManager: internal DB corruption")
		}
		return true
	}, &temp)
	return
}

func (logManager *logManagerInstance) SetTargetConfigByName(targetname string, logconfig []maestroSpecs.LogTarget) (err error) {

	return
}

//This function is called during bootup. It waits for devicedb to be up and running to connect to it, once connected it calls
//SetupDeviceDBConfig
func (logManager *logManagerInstance) initDeviceDBConfig() error {
	var err error

	logManager.ddbConfigClient, err = maestroConfig.GetDDBRelayConfigClient(logManager.ddbConnConfig)
	if logManager.ddbConfigClient == nil {
		return err
	}
	log.MaestroInfof("LogManager: successfully connected to devicedb\n")

	err = logManager.SetupDeviceDBConfig()
	if err != nil {
		log.MaestroErrorf("LogManager: error setting up config using devicedb: %v", err)
	} else {
		log.MaestroInfof("LogManager: successfully read config from devicedb\n")
	}

	return err
}

//SetupDeviceDBConfig reads the config from devicedb and if its new it applies the new config.
//It also sets up the config update handlers for all the tags/groups.
func (logManager *logManagerInstance) SetupDeviceDBConfig() (err error) {
	var ddbLogConfig maestroSpecs.LogConfigPayload

	//Create a config analyzer object, required for registering the config change hook and diff the config objects.
	configLogAna := maestroSpecs.NewConfigAnalyzer(DDBLogConfigGroupID)
	if configLogAna == nil {
		log.MaestroDebugf("LogManager: Failed to create config analyzer object, unable to fetch config from devicedb")
		errUpdated := errors.New("Failed to create config analyzer object, unable to fetch config from devicedb")
		return errUpdated
	}

	err = logManager.ddbConfigClient.Config(DDBLogConfigName).Get(&ddbLogConfig)
	if err != nil {
		for _, target := range logManager.logConfigRunning {
			target := target
			ddbLogConfig.Targets = append(ddbLogConfig.Targets, &target)
		}
		log.MaestroDebugf("LogManager: No log config found in devicedb or unable to connect to devicedb err: %v. Let's put the current running config.\n", err)
		err = logManager.ddbConfigClient.Config(DDBLogConfigName).Put(&ddbLogConfig)
		if err != nil {
			log.MaestroDebugf("LogManager: Unable to put log config in devicedb err:%v, config will not be monitored from devicedb\n", err)
			errUpdated := errors.New(fmt.Sprintf("\nUnable to put log config in devicedb err:%v, config will not be monitored from devicedb\n", err))
			return errUpdated
		}
	} else {
		//We found a config in devicedb, lets try to use and reconfigure log if its an updated one
		log.MaestroInfof("LogManager: Found a valid config in devicedb [%v], will try to use and reconfigure log if its an updated one\n", ddbLogConfig.Targets)
		log.MaestroInfof("MRAY DiffChanges being called\n")
		var temp maestroSpecs.LogConfigPayload
		for _, target := range logManager.logConfigRunning {
			target := target
			temp.Targets = append(temp.Targets, &target)
		}
		identical, _, _, err := configLogAna.DiffChanges(temp, ddbLogConfig)
		if !identical && (err == nil) {
			//The configs are different, lets go ahead reconfigure the intfs
			log.MaestroInfof("LogManager: New log config found from devicedb, reconfigure nework using new config\n")
			logManager.logConfigRunning = make([]maestroSpecs.LogTarget, len(ddbLogConfig.Targets))
			for i, target := range ddbLogConfig.Targets {
				logManager.logConfigRunning[i] = *target
			}
			logManager.submitConfigAndSync(logManager.logConfigRunning)
			// //Setup the intfs using new config
			// this.setupTargets()
		} else {
			log.MaestroInfof("LogManager: New log config found from devicedb, but its same as boot config, no need to re-configure\n")
		}
	}

	//Now start a monitor for the log config in devicedb
	//Add config change hook for all property groups, we can use the same interface
	var logConfigChangeHook ChangeHook

	//log target keys
	configLogAna.AddHook("name", logConfigChangeHook)
	configLogAna.AddHook("filters", logConfigChangeHook)
	configLogAna.AddHook("format", logConfigChangeHook)
	configLogAna.AddHook("opts", logConfigChangeHook)

	//Add monitor for this config
	var origLogConfig, updatedLogConfig maestroSpecs.LogConfigPayload

	//Provide a copy of current log config monitor to Config monitor, not the actual config we use, this would prevent config monitor
	//directly updating the running config(this.logConfigRunning).
	for _, target := range logManager.logConfigRunning {
		target := target
		origLogConfig.Targets = append(origLogConfig.Targets, &target)
	}

	log.MaestroInfof("adding log monitor config\n")

	//Adding monitor config
	logManager.ddbConfigClient.AddMonitorConfig(&origLogConfig, &updatedLogConfig, DDBLogConfigName, configLogAna)

	//Add config change hook for all property groups, we can use the same interface
	var commitConfigChangeHook CommitConfigChangeHook
	configLogAna.AddHook("config_commit", commitConfigChangeHook)

	//Add monitor for this object
	var updatedConfigCommit ConfigCommit
	log.MaestroInfof("LoggingManager: Adding monitor for config commit object\n")
	logManager.ddbConfigClient.AddMonitorConfig(&logManager.CurrConfigCommit, &updatedConfigCommit, DDBLogConfigCommit, configLogAna)

	return
}

//GetLogLibVersion get verson of log library
func GetLogLibVersion() string {
	return greasego.GetGreaseLibVersion()
}

// InitLogManager be called on startup.
// LogTarget will come from config file
// Storage should be started already.
func InitLogManager(config *maestroConfig.YAMLMaestroConfig) (err error) {

	inst := GetInstance()
	inst.logConfigRunning = config.Targets
	inst.ddbConnConfig = config.DDBConnConfig

	log.MaestroInfof("LogManager: Initializing %v %v\n", inst.logConfigRunning, inst.ddbConnConfig)

	greasego.StartGreaseLib(func() {
		debugging.DEBUG_OUT("Grease start cb: Got to here 1\n")
	})
	greasego.SetupStandardLevels()
	greasego.SetupStandardTags()

	log.SetGoLoggerReady()

	if config.LinuxKernelLog && config.LinuxKernelLogLegacy {
		return errors.New("Invalid Config: You can't have both linuxKernelLog: true AND linuxKernelLogLegacy: true. Choose one")
	}
	if config.LinuxKernelLog {
		kernelSink := greasego.NewGreaseLibSink(greasego.GREASE_LIB_SINK_KLOG2, nil)
		greasego.AddSink(kernelSink)
	}
	if config.LinuxKernelLogLegacy {
		kernelSink := greasego.NewGreaseLibSink(greasego.GREASE_LIB_SINK_KLOG, nil)
		greasego.AddSink(kernelSink)
	}

	unixLogSocket := config.GetUnixLogSocket()
	debugging.DEBUG_OUT("UnixLogSocket: %s\n", unixLogSocket)
	unixSockSink := greasego.NewGreaseLibSink(greasego.GREASE_LIB_SINK_UNIXDGRAM, &unixLogSocket)
	greasego.AddSink(unixSockSink)

	syslogSock := config.GetSyslogSocket()
	if len(syslogSock) > 0 {
		syslogSink := greasego.NewGreaseLibSink(greasego.GREASE_LIB_SINK_SYSLOGDGRAM, &syslogSock)
		greasego.AddSink(syslogSink)
	}

	// First, setup the internal maestro logging system to deal with toCloud target
	// This requires creating the default Symphony client:
	var symphonyClient *wwrmi.Client
	var symphonyErr error
	if config.Symphony != nil {
		symphonyClient, symphonyErr = wwrmi.GetMainClient(config.Symphony)
	} else {
		log.MaestroDebugf("Symphony / RMI API server not configured.\n")
	}

	debugging.DEBUG_OUT("targets:", len(config.Targets))
	for n := 0; n < len(config.Targets); n++ {
		if len(config.Targets[n].File) > 0 { // honor any substitution vars for the File targets
			config.Targets[n].File = maestroConfig.GetInterpolatedConfigString(config.Targets[n].File)
		}
		opts := greasego.NewGreaseLibTargetOpts()
		greasego.AssignFromStruct(opts, config.Targets[n]) //, reflect.TypeOf(config.Targets[n]))

		if config.Targets[n].Flag_json_escape_strings {
			greasego.TargetOptsSetFlags(opts, greasego.GREASE_JSON_ESCAPE_STRINGS)
		}

		debugging.DEBUG_OUT("%+v\n", opts.FileOpts)
		debugging.DEBUG_OUT("%+v\n", opts)
		debugging.DEBUG_OUT("%+v\n", *opts.Format_time)

		if strings.Compare(config.Targets[n].Name, "toCloud") == 0 {
			log.MaestroInfof("\nFound toCloud target-------->\n")
			opts.NumBanks = defaults.NUMBER_BANKS_WEBLOG
			//			DEBUG(_count := 0)
			if config.Symphony != nil && symphonyClient != nil && symphonyErr == nil {
				opts.TargetCB = wwrmi.TargetCB
			} else {
				log.MaestroInfof("Log: 'toCloud' target is enabled, but Symphony API is not configured. Will not work.")
				// skip this target
				continue
			}

			// func(err *greasego.GreaseError, data *greasego.TargetCallbackData){
			// 	DEBUG(_count++)
			// 	debugging.DEBUG_OUT("}}}}}}}}}}}} TargetCB_count called %d times\n",_count);
			// 	if(err != nil) {
			// 		fmt.Printf("ERROR in toCloud target CB %s\n", err.Str)
			// 	} else {
			// 		buf := data.GetBufferAsSlice()
			// 		DEBUG(s := string(buf))
			// 		debugging.DEBUG_OUT("CALLBACK %+v ---->%s<----\n\n",data,s);
			// 		client.SubmitLogs(data,buf)
			// 	}
			// }
		}

		func(n int, opts *greasego.GreaseLibTargetOpts) {
			greasego.AddTarget(opts, func(err *greasego.GreaseError, optsId int, targId uint32) {
				debugging.DEBUG_OUT("IN CALLBACK %d\n", optsId)
				if err != nil {
					log.MaestroDebugf("ERROR on creating target: %s\n", err.Str)
				} else {
					// after the Target is added, we can setup the Filters for it
					if len(config.Targets[n].Filters) > 0 {
						for l := 0; l < len(config.Targets[n].Filters); l++ {
							debugging.DEBUG_OUT("Have filter %+v\n", config.Targets[n].Filters[l])
							filter := greasego.NewGreaseLibFilter()
							filter.Target = targId
							// handle the strings:
							greasego.AssignFromStruct(filter, config.Targets[n].Filters[l]) //, reflect.TypeOf(config.Targets[n].Filters[l]))
							greasego.SetFilterValue(filter, greasego.GREASE_LIB_SET_FILTER_TARGET, targId)
							if len(config.Targets[n].Filters[l].Levels) > 0 {
								mask := maestroConfig.ConvertLevelStringToUint32Mask(config.Targets[n].Filters[l].Levels)
								greasego.SetFilterValue(filter, greasego.GREASE_LIB_SET_FILTER_MASK, mask)
							}
							if len(config.Targets[n].Filters[l].Tag) > 0 {
								tag := maestroConfig.ConvertTagStringToUint32(config.Targets[n].Filters[l].Tag)
								greasego.SetFilterValue(filter, greasego.GREASE_LIB_SET_FILTER_MASK, tag)
							}
							debugging.DEBUG_OUT("Filter -----------> %+v\n", filter)
							filterid := greasego.AddFilter(filter)
							debugging.DEBUG_OUT("Filter ID: %d\n", filterid)
						}
					} else {
						// by default, send all traffic to any target

					}
				}
			})
		}(n, opts) // use anonymous function to preserve 'n' before callback completes

	}

	// should not start workers until after greasego is setup
	if config.Symphony != nil {
		if symphonyErr != nil {
			log.MaestroDebugf("Symphony / RMI client is not configured correctly or has failed: %s\n", symphonyErr.Error())
		} else {
			symphonyClient.StartWorkers()
			log.MaestroInfof("Maestro RMI workers started")
			log.MaestroInfof("Symphony / RMI client workers started.")
		}
	}

	client := log.NewSymphonyClient("http://127.0.0.1:9443/submitLog/1", config.ClientId, defaults.NUMBER_BANKS_WEBLOG, 30*time.Second)
	client.Start()

	log.MaestroInfof("init device db config\n")
	go inst.initDeviceDBConfig()

	return
}
