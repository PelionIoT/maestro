package logconfig

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/armPelionEdge/greasego"
	"github.com/armPelionEdge/hashmap"
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

	byLogName   *hashmap.HashMap // map of identifier to struct -> 'eth2':&networkInterfaceData{}
	indexToName *hashmap.HashMap

	watcherWorkChannel chan logThreadMessage

	// mostly used for testing
	threadCountEnable         bool
	interfaceThreadCount      int
	interfaceThreadCountMutex sync.Mutex
	threadCountChan           chan logThreadMessage

	// current config state
	logConfig []maestroSpecs.LogTarget

	// Configs to be used for connecting to devicedb
	ddbConnConfig    *maestroConfig.DeviceDBConnConfig
	ddbConfigMonitor *maestroConfig.DDBMonitor
	ddbConfigClient  *maestroConfig.DDBRelayConfigClient
}

//DDBLogConfigName name used in log writes
const DDBLogConfigName string = "MAESTRO_LOG_CONFIG_ID"

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
		//instance.logConfig = new([]maestroSpecs.LogTarget)

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
	logManager.logConfig = config

	debugging.DEBUG_OUT("targets:", len(config))
	for n := 0; n < len(config); n++ {
		confOk, problem := validateLogConfig(config[n])
		if !confOk {
			log.MaestroDebugf("LogManager: Target config problem: \"%s\"  Skipping interface config.\n", problem)
			continue
		}
		targetname := config[n].Name

		var storedconfig maestroSpecs.LogTarget
		err := logManager.logConfigDB.Get(targetname, &storedconfig)
		if err != nil {
			log.MaestroDebugf("LogManager: sumbitConfig: Get failed: %v\n", err)
			if err != stow.ErrNotFound {
				log.MaestroDebugf("LogManager: problem with database on if %s - Get: %s\n", targetname, err.Error())
			} else {
				//cache our local copy. Do we need to do this anymore?
				logManager.byLogName.Set(targetname, unsafe.Pointer(&config[n]))
				//write this out to the DB
				err := logManager.logConfigDB.Put(targetname, config[n])
				if err != nil {
					log.MaestroDebugf("LogManager: Problem (1) storing config for interface [%s]: %s\n", config[n].Name, err)
				}
			}
		} else {
			// entry existed, so override
			if config[n].Existing == "replace" || config[n].Existing == "override" {
				// do i need to do this?
				logManager.byLogName.Set(targetname, unsafe.Pointer(&config[n]))
				//write this out to the DB
				err := logManager.logConfigDB.Put(targetname, config[n])
				if err != nil {
					log.MaestroDebugf("LogManager: Problem (2) storing config for interface [%s]: %s\n", config[n].Name, err)
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
				logManager.byLogName.Set(logname, unsafe.Pointer(&logdata))
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

//Constants used in the logic for connecting to devicedb
/////////////////////////////////////////////////////////

//MaxDeviceDBWaitTimeInSecs is 24 hours
const MaxDeviceDBWaitTimeInSecs int = (24 * 60 * 60)

//LoopWaitTimeIncrementWindow is 6 minutes which is the exponential retry backoff window
const LoopWaitTimeIncrementWindow int = (6 * 60)

//InitialDeviceDBStatusCheckIntervalInSecs is 5 secs
const InitialDeviceDBStatusCheckIntervalInSecs int = 5

//IncreasedDeviceDBStatusCheckIntervalInSecs is Exponential retry backoff interval
const IncreasedDeviceDBStatusCheckIntervalInSecs int = 120

//initDeviceDBConfig This function is called during bootup. It waits for devicedb to be up and running to connect to it, once connected it calls
//SetupDeviceDBConfig
func (logManager *logManagerInstance) initDeviceDBConfig() {
	var totalWaitTime int = 0
	var loopWaitTime int = InitialDeviceDBStatusCheckIntervalInSecs
	var err error
	log.MaestroInfof("initDeviceDBConfig: connecting to devicedb\n")
	err = logManager.SetupDeviceDBConfig()

	//After 24 hours just assume its never going to come up stop waiting for it and break the loop
	for (err != nil) && (totalWaitTime < MaxDeviceDBWaitTimeInSecs) {
		log.MaestroDebugf("initDeviceDBConfig: Waiting for devicedb to connect\n")
		time.Sleep(time.Second * time.Duration(loopWaitTime))
		totalWaitTime += loopWaitTime
		//If we cant connect in first 6 minutes, check much less frequently for next 24 hours hoping that devicedb may come up later.
		if totalWaitTime > LoopWaitTimeIncrementWindow {
			loopWaitTime = IncreasedDeviceDBStatusCheckIntervalInSecs
		}
		err = logManager.SetupDeviceDBConfig()
	}

	if totalWaitTime >= MaxDeviceDBWaitTimeInSecs {
		log.MaestroDebugf("initDeviceDBConfig: devicedb is not connected, cannot fetch config from devicedb")
	}
	if err == nil {
		log.MaestroInfof("initDeviceDBConfig: successfully connected to devicedb\n")
	}
}

//SetupDeviceDBConfig reads the config from devicedb and if its new it applies the new config.
//It also sets up the config update handlers for all the tags/groups.
func (logManager *logManagerInstance) SetupDeviceDBConfig() (err error) {
	//TLS config to connect to devicedb
	var tlsConfig *tls.Config

	if logManager.ddbConnConfig != nil {
		log.MaestroInfof("LogManager: Found valid devicedb connection config, try connecting and fetching the config from devicedb: uri:%s prefix: %s bucket:%s id:%s cert:%s\n",
			logManager.ddbConnConfig.DeviceDBUri, logManager.ddbConnConfig.DeviceDBPrefix, logManager.ddbConnConfig.DeviceDBBucket, logManager.ddbConnConfig.RelayId, logManager.ddbConnConfig.CaChainCert)
		//Device DB config uses deviceid as the relay_id, so uset that to set the hostname
		log.MaestroInfof("LogManager: Setting hostname: %s\n", logManager.ddbConnConfig.RelayId)
		syscall.Sethostname([]byte(logManager.ddbConnConfig.RelayId))

		relayCaChain, err := ioutil.ReadFile(logManager.ddbConnConfig.CaChainCert)
		if err != nil {
			log.MaestroDebugf("LogManager: Unable to access ca-chain-cert file at: %s\n", logManager.ddbConnConfig.CaChainCert)
			errUpdated := errors.New(fmt.Sprintf("LogManager: Unable to access ca-chain-cert file at: %s, err = %v\n", logManager.ddbConnConfig.CaChainCert, err))
			return errUpdated
		}

		caCerts := x509.NewCertPool()

		if !caCerts.AppendCertsFromPEM(relayCaChain) {
			log.MaestroDebugf("CA chain loaded from %s is not valid: %v\n", logManager.ddbConnConfig.CaChainCert, err)
			errUpdated := errors.New(fmt.Sprintf("CA chain loaded from %s is not valid\n", logManager.ddbConnConfig.CaChainCert))
			return errUpdated
		}

		tlsConfig = &tls.Config{
			RootCAs: caCerts,
		}

		//Config for log
		var ddbLogConfig maestroSpecs.LogConfigPayload

		//Create a config analyzer object, required for registering the config change hook and diff the config objects.
		configLogAna := maestroSpecs.NewConfigAnalyzer(DDBLogConfigGroupID)
		if configLogAna == nil {
			log.MaestroDebugf("LogManager: Failed to create config analyzer object, unable to fetch config from devicedb")
			errUpdated := errors.New("Failed to create config analyzer object, unable to fetch config from devicedb")
			return errUpdated
		} else {
			logManager.ddbConfigClient = maestroConfig.NewDDBRelayConfigClient(tlsConfig, logManager.ddbConnConfig.DeviceDBUri, logManager.ddbConnConfig.RelayId, logManager.ddbConnConfig.DeviceDBPrefix, logManager.ddbConnConfig.DeviceDBBucket)
			err = logManager.ddbConfigClient.Config(DDBLogConfigName).Get(&ddbLogConfig)
			if err != nil {
				for _, target := range logManager.logConfig {
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
				for _, target := range logManager.logConfig {
					target := target
					temp.Targets = append(temp.Targets, &target)
				}
				identical, _, _, err := configLogAna.DiffChanges(temp, ddbLogConfig)
				if !identical && (err == nil) {
					//The configs are different, lets go ahead reconfigure the intfs
					log.MaestroInfof("LogManager: New log config found from devicedb, reconfigure nework using new config\n")
					logManager.logConfig = make([]maestroSpecs.LogTarget, len(ddbLogConfig.Targets))
					for i, target := range ddbLogConfig.Targets {
						logManager.logConfig[i] = *target
					}
					logManager.submitConfigAndSync(logManager.logConfig)
					// //Setup the intfs using new config
					// this.setupTargets()
				} else {
					log.MaestroInfof("LogManager: New log config found from devicedb, but its same as boot config, no need to re-configure\n")
				}
			}

			//Now start a monitor for the log config in devicedb
			err, logManager.ddbConfigMonitor = maestroConfig.NewDeviceDBMonitor(logManager.ddbConnConfig)
			if err != nil {
				log.MaestroDebugf("LogManager: Unable to create config monitor: %v\n", err)
			} else {
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
				//directly updating the running config(this.logConfig).
				for _, target := range logManager.logConfig {
					target := target
					origLogConfig.Targets = append(origLogConfig.Targets, &target)
				}

				log.MaestroInfof("adding log monitor config\n")

				//Adding monitor config
				logManager.ddbConfigMonitor.AddMonitorConfig(&origLogConfig, &updatedLogConfig, DDBLogConfigName, configLogAna)
			}
		}
	} else {
		log.MaestroDebugf("LogManager: No devicedb connection config available, configuration will not be fetched from devicedb\n")
	}

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
	inst.logConfig = config.Targets
	inst.ddbConnConfig = config.DDBConnConfig

	log.MaestroInfof("LogManager: Initializing %v %v\n", inst.logConfig, inst.ddbConnConfig)

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
