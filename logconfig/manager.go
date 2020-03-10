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
	LOG_PREFIX = "LogManager: "
)

// internal wrapper, used to store log settings
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
	DHCP_NETWORK_FAILURE_TIMEOUT_DURATION time.Duration = 30 * time.Second
	// used when an interface goes down, then back up, etc.
	DHCP_NET_TRANSITION_TIMEOUT time.Duration = 2 * time.Second
	c_DB_NAME                                 = "logConfig"

	// public event names
	INTERFACE_READY = 0x10F2
	INTERFACE_DOWN  = 0x10F3

	// internal event status
	state_LOWER_DOWN = 0x1001
	state_LOWER_UP   = 0x1002

	// for dhcpWorkControl / watcherWorkChannel
	dhcp_get_lease      = 0x2001
	dhcp_renew_lease    = 0x2002
	dhcp_rebind_lease   = 0x2003
	thread_shutdown     = 0xFF
	stop_and_release_IP = 0x00F1

	start_watch_interface = 0x3001
	stop_watch_interface  = 0x3002

	// for dhcpWaitOnShutdown
	shutdown_complete = 0x0001

	no_interface_threads = 0x3001
)

type logThreadMessage struct {
	cmd     int
	logname string // sometimes used
}

type ConfigCommit struct {
	// Set this flag to true for the changes to commit, if this flag is false
	// the changes to configuration on these structs will not acted upon
	// by log manager. For exmaple, this flag will be initially false
	// so that user can change the config object in DeviceDB and verify that the
	// intented changes are captured correctly. Once verified set this flag to true
	// so that the changes will be applied by maestro. Once maestro complete the
	// changes the flag will be set to false by maestro.
	ConfigCommitFlag bool `yaml:"config_commit" json:"config_commit" netgroup:"config_commit"`
	//Datetime of last update
	LastUpdateTimestamp string `yaml:"config_commit" json:"last_commit_timestamp" netgroup:"config_commit"`
	//Total number of updates from boot
	TotalCommitCountFromBoot int `yaml:"config_commit" json:"total_commit_count_from_boot" netgroup:"config_commit"`
}

// initialized in InitLogManager()
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
	CurrConfigCommit ConfigCommit
}

const DDB_LOG_CONFIG_NAME string = "MAESTRO_LOG_CONFIG_ID"
const DDB_LOG_CONFIG_COMMIT_FLAG string = "MAESTRO_LOG_CONFIG_COMMIT_FLAG"
const DDB_LOG_CONFIG_CONFIG_GROUP_ID string = "log_group"

var instance *logManagerInstance

func (this *logManagerInstance) enableThreadCount() {
	this.threadCountEnable = true
}
func (this *logManagerInstance) incLogThreadCount() {
	if this.threadCountEnable {
		this.interfaceThreadCountMutex.Lock()
		this.interfaceThreadCount++
		this.interfaceThreadCountMutex.Unlock()
	}
}
func (this *logManagerInstance) decLogThreadCount() {
	if this.threadCountEnable {
		this.interfaceThreadCountMutex.Lock()
		this.interfaceThreadCount--
		if this.interfaceThreadCount < 1 {
			debugging.DEBUG_OUT("submit to threadCountChan no_interface_threads\n")
			this.threadCountChan <- logThreadMessage{cmd: no_interface_threads}
		}
		this.interfaceThreadCountMutex.Unlock()
	}
}
func (this *logManagerInstance) waitForActiveLogThreads(timeout_seconds int) (wastimeout bool) {
	if !this.threadCountEnable {
		log.MaestroError("LogManager: Called waitForActiveLogThreads - threadCountEnable is false!")
		return
	}
	timeout := time.Second * time.Duration(timeout_seconds)
	// CountLoop:
	// for {
	select {
	case cmd := <-this.threadCountChan:
		if cmd.cmd != no_interface_threads {
			debugging.DEBUG_OUT("Uknown command entered threadCountChan channel\n")
		}
		// break CountLoop
	case <-time.After(timeout):
		debugging.DEBUG_OUT("TIMEOUT in waitForNowActiveInterfaceThreads\n")
		wastimeout = true
		// break CountLoop
	}
	// }
	return
}

// impl storage.StorageUser interface
func (this *logManagerInstance) StorageInit(instance storage.MaestroDBStorageInterface) {
	gob.Register(&maestroSpecs.LogTarget{})
	gob.Register(&LogData{})
	this.logConfigDB = stow.NewStore(instance.GetDb(), []byte(c_DB_NAME))
}

func (this *logManagerInstance) StorageReady(instance storage.MaestroDBStorageInterface) {
	this.db = instance.GetDb()
	err := this.loadAllLogData()
	if err != nil {
		log.MaestroErrorf("LogManager: Failed to load existing log interface settings: %s\n", err.Error())
	}
}

func (this *logManagerInstance) StorageClosed(instance storage.MaestroDBStorageInterface) {

}

func newLogManagerInstance() (ret *logManagerInstance) {
	ret = new(logManagerInstance)
	ret.watcherWorkChannel = make(chan logThreadMessage, 10) // use a buffered channel
	ret.threadCountChan = make(chan logThreadMessage)

	return
}

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

func (this *LogData) setRunningConfig(logconfig []maestroSpecs.LogTarget) (ret *LogData) {
	//ret.RunningLogconfig = new(maestroSpecs.LogFilter)
	ret.RunningLogconfig = logconfig // copy that entire struct
	return
}

func validateLogConfig(logconf maestroSpecs.LogTarget) (ok bool, problem string) {
	ok = true
	/*if len(logconf.IfName) < 1 {
		ok = false
		problem = "Interface is missing IfName / if_name field"
	}*/
	return
}

// to be ran on startup, or new interface creation
//
func resetLogDataFromStoredConfig(logtarget maestroSpecs.LogTarget) error {
	/*var logconf []maestroSpecs.LogTarget

	logconf = nil

	if logdata.StoredLogconfig != nil {
		logconf = logdata.StoredLogconfig
	} else {
		log.MaestroWarnf("Logging does not have a StoredConfig. Why?\n")
		if logdata.RunningLogconfig != nil {
			logconf = logdata.RunningLogconfig
		} else {
			return errors.New("No interface configuration")
		}
	}
	*/
	return nil
}

func (this *logManagerInstance) submitConfigAndSync(config []maestroSpecs.LogTarget) {
	this.submitConfig(config)
	var ddbLogConfig maestroSpecs.LogConfigPayload

	for _, target := range config {
		target := target
		ddbLogConfig.Targets = append(ddbLogConfig.Targets, &target)
	}
	log.MaestroWarnf("LogManager: Updating devicedb config.\n")
	err := this.ddbConfigClient.Config(DDB_LOG_CONFIG_NAME).Put(&ddbLogConfig)
	if err != nil {
		log.MaestroErrorf("LogManager: Unable to put log config in devicedb err: %v\n", err)
	}
}

func (this *logManagerInstance) submitConfig(config []maestroSpecs.LogTarget) {
	this.logConfig = config

	debugging.DEBUG_OUT("targets:", len(config))
	for n := 0; n < len(config); n++ {
		conf_ok, problem := validateLogConfig(config[n])
		if !conf_ok {
			log.MaestroErrorf("LogManager: Target config problem: \"%s\"  Skipping interface config.\n", problem)
			continue
		}
		targetname := config[n].Name

		var storedconfig maestroSpecs.LogTarget
		err := this.logConfigDB.Get(targetname, &storedconfig)
		if err != nil {
			log.MaestroInfof("LogManager: sumbitConfig: Get failed: %v\n", err)
			if err != stow.ErrNotFound {
				log.MaestroErrorf("LogManager: problem with database on if %s - Get: %s\n", targetname, err.Error())
			} else {
				//cache our local copy. Do we need to do this anymore?
				this.byLogName.Set(targetname, unsafe.Pointer(&config[n]))
				//write this out to the DB
				err := this.logConfigDB.Put(targetname, config[n])
				if err != nil {
					log.MaestroErrorf("LogManager: Problem (1) storing config for interface [%s]: %s\n", config[n].Name, err)
				}
			}
		} else {
			// entry existed, so override
			if config[n].Existing == "replace" || config[n].Existing == "override" {
				// do i need to do this?
				this.byLogName.Set(targetname, unsafe.Pointer(&config[n]))
				//write this out to the DB
				err := this.logConfigDB.Put(targetname, config[n])
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
func (this *logManagerInstance) loadAllLogData() (err error) {
	var temp LogData
	this.logConfigDB.IterateIf(func(key []byte, val interface{}) bool {
		logname := string(key[:])
		logdata, ok := val.(maestroSpecs.LogTarget)
		if ok {
			//this probably isn't even remotely necessary
			err2 := resetLogDataFromStoredConfig(logdata)
			if err2 == nil {
				//this.newInterfaceMutex.Lock()
				// if there is an existing in-memory entry, overwrite it
				log.MaestroInfof("LogManager:loadAllLogData: Loading config for: %s.\n", logname)
				this.byLogName.Set(logname, unsafe.Pointer(&logdata))
				//this.newInterfaceMutex.Unlock()
				debugging.DEBUG_OUT("loadAllLogData() see if: %s --> %+v\n", logname, logdata)
			} else {
				log.MaestroErrorf("LogManager: Critical problem with interface [%s] config. Not loading config.\n", logname)
			}
		} else {
			err = errors.New("Internal DB corruption")
			debugging.DEBUG_OUT("LogManager: internal DB corruption - @if %s\n", logname)
			log.MaestroError("LogManager: internal DB corruption")
		}
		return true
	}, &temp)
	return
}

func (this *logManagerInstance) SetTargetConfigByName(targetname string, logconfig []maestroSpecs.LogTarget) (err error) {
	/*targetdata := this.getOrNewInterfaceData(targetname)
	targetdata.StoredLogconfig = logconfig
	this.commitInterfaceData(targetname)*/
	return
}

//Constants used in the logic for connecting to devicedb
const MAX_DEVICEDB_WAIT_TIME_IN_SECS int = (24 * 60 * 60)        //24 hours
const LOOP_WAIT_TIME_INCREMENT_WINDOW int = (6 * 60)             //6 minutes which is the exponential retry backoff window
const INITIAL_DEVICEDB_STATUS_CHECK_INTERVAL_IN_SECS int = 5     //5 secs
const INCREASED_DEVICEDB_STATUS_CHECK_INTERVAL_IN_SECS int = 120 //Exponential retry backoff interval
const DEVICEDB_JOB_NAME string = "devicedb"

//This function is called during bootup. It waits for devicedb to be up and running to connect to it, once connected it calls
//SetupDeviceDBConfig
func (this *logManagerInstance) initDeviceDBConfig() {
	var totalWaitTime int = 0
	var loopWaitTime int = INITIAL_DEVICEDB_STATUS_CHECK_INTERVAL_IN_SECS
	var err error
	log.MaestroWarnf("initDeviceDBConfig: connecting to devicedb\n")
	err = this.SetupDeviceDBConfig()

	//After 24 hours just assume its never going to come up stop waiting for it and break the loop
	for (err != nil) && (totalWaitTime < MAX_DEVICEDB_WAIT_TIME_IN_SECS) {
		log.MaestroInfof("initDeviceDBConfig: Waiting for devicedb to connect\n")
		time.Sleep(time.Second * time.Duration(loopWaitTime))
		totalWaitTime += loopWaitTime
		//If we cant connect in first 6 minutes, check much less frequently for next 24 hours hoping that devicedb may come up later.
		if totalWaitTime > LOOP_WAIT_TIME_INCREMENT_WINDOW {
			loopWaitTime = INCREASED_DEVICEDB_STATUS_CHECK_INTERVAL_IN_SECS
		}
		err = this.SetupDeviceDBConfig()
	}

	if totalWaitTime >= MAX_DEVICEDB_WAIT_TIME_IN_SECS {
		log.MaestroErrorf("initDeviceDBConfig: devicedb is not connected, cannot fetch config from devicedb")
	}
	if err == nil {
		log.MaestroWarnf("initDeviceDBConfig: successfully connected to devicedb\n")
	}
}

//SetupDeviceDBConfig reads the config from devicedb and if its new it applies the new config.
//It also sets up the config update handlers for all the tags/groups.
func (this *logManagerInstance) SetupDeviceDBConfig() (err error) {
	//TLS config to connect to devicedb
	var tlsConfig *tls.Config

	if this.ddbConnConfig != nil {
		log.MaestroInfof("LogManager: Found valid devicedb connection config, try connecting and fetching the config from devicedb: uri:%s prefix: %s bucket:%s id:%s cert:%s\n",
			this.ddbConnConfig.DeviceDBUri, this.ddbConnConfig.DeviceDBPrefix, this.ddbConnConfig.DeviceDBBucket, this.ddbConnConfig.RelayId, this.ddbConnConfig.CaChainCert)
		//Device DB config uses deviceid as the relay_id, so uset that to set the hostname
		log.MaestroWarnf("LogManager: Setting hostname: %s\n", this.ddbConnConfig.RelayId)
		syscall.Sethostname([]byte(this.ddbConnConfig.RelayId))

		relayCaChain, err := ioutil.ReadFile(this.ddbConnConfig.CaChainCert)
		if err != nil {
			log.MaestroErrorf("LogManager: Unable to access ca-chain-cert file at: %s\n", this.ddbConnConfig.CaChainCert)
			err_updated := errors.New(fmt.Sprintf("LogManager: Unable to access ca-chain-cert file at: %s, err = %v\n", this.ddbConnConfig.CaChainCert, err))
			return err_updated
		}

		caCerts := x509.NewCertPool()

		if !caCerts.AppendCertsFromPEM(relayCaChain) {
			log.MaestroErrorf("CA chain loaded from %s is not valid: %v\n", this.ddbConnConfig.CaChainCert, err)
			err_updated := errors.New(fmt.Sprintf("CA chain loaded from %s is not valid\n", this.ddbConnConfig.CaChainCert))
			return err_updated
		}

		tlsConfig = &tls.Config{
			RootCAs: caCerts,
		}

		//Config for log
		var ddbLogConfig maestroSpecs.LogConfigPayload

		//Create a config analyzer object, required for registering the config change hook and diff the config objects.
		configLogAna := maestroSpecs.NewConfigAnalyzer(DDB_LOG_CONFIG_CONFIG_GROUP_ID)
		if configLogAna == nil {
			log.MaestroErrorf("LogManager: Failed to create config analyzer object, unable to fetch config from devicedb")
			err_updated := errors.New("Failed to create config analyzer object, unable to fetch config from devicedb")
			return err_updated
		} else {
			this.ddbConfigClient = maestroConfig.NewDDBRelayConfigClient(tlsConfig, this.ddbConnConfig.DeviceDBUri, this.ddbConnConfig.RelayId, this.ddbConnConfig.DeviceDBPrefix, this.ddbConnConfig.DeviceDBBucket)
			err = this.ddbConfigClient.Config(DDB_LOG_CONFIG_NAME).Get(&ddbLogConfig)
			if err != nil {
				for _, target := range this.logConfig {
					target := target
					ddbLogConfig.Targets = append(ddbLogConfig.Targets, &target)
				}
				log.MaestroWarnf("LogManager: No log config found in devicedb or unable to connect to devicedb err: %v. Let's put the current running config.\n", err)
				err = this.ddbConfigClient.Config(DDB_LOG_CONFIG_NAME).Put(&ddbLogConfig)
				if err != nil {
					log.MaestroErrorf("LogManager: Unable to put log config in devicedb err:%v, config will not be monitored from devicedb\n", err)
					err_updated := errors.New(fmt.Sprintf("\nUnable to put log config in devicedb err:%v, config will not be monitored from devicedb\n", err))
					return err_updated
				}
			} else {
				//We found a config in devicedb, lets try to use and reconfigure log if its an updated one
				log.MaestroInfof("LogManager: Found a valid config in devicedb [%v], will try to use and reconfigure log if its an updated one\n", ddbLogConfig.Targets)
				log.MaestroInfof("MRAY DiffChanges being called\n")
				fmt.Printf("MRAY DiffChanges being called\n")
				var temp maestroSpecs.LogConfigPayload
				for _, target := range this.logConfig {
					target := target
					temp.Targets = append(temp.Targets, &target)
				}
				identical, _, _, err := configLogAna.DiffChanges(temp, ddbLogConfig)
				if !identical && (err == nil) {
					//The configs are different, lets go ahead reconfigure the intfs
					log.MaestroDebugf("LogManager: New log config found from devicedb, reconfigure nework using new config\n")
					this.logConfig = make([]maestroSpecs.LogTarget, len(ddbLogConfig.Targets))
					for i, target := range ddbLogConfig.Targets {
						this.logConfig[i] = *target
					}
					this.submitConfigAndSync(this.logConfig)
					// //Setup the intfs using new config
					// this.setupTargets()
				} else {
					log.MaestroInfof("LogManager: New log config found from devicedb, but its same as boot config, no need to re-configure\n")
				}
			}

			//Now start a monitor for the log config in devicedb
			err, this.ddbConfigMonitor = maestroConfig.NewDeviceDBMonitor(this.ddbConnConfig)
			if err != nil {
				log.MaestroErrorf("LogManager: Unable to create config monitor: %v\n", err)
			} else {
				//Add config change hook for all property groups, we can use the same interface
				var logConfigChangeHook LogConfigChangeHook

				//log target keys
				configLogAna.AddHook("name", logConfigChangeHook)
				configLogAna.AddHook("filters", logConfigChangeHook)
				configLogAna.AddHook("format", logConfigChangeHook)
				configLogAna.AddHook("opts", logConfigChangeHook)

				//Add monitor for this config
				var origLogConfig, updatedLogConfig maestroSpecs.LogConfigPayload
				//Provide a copy of current log config monitor to Config monitor, not the actual config we use, this would prevent config monitor
				//directly updating the running config(this.logConfig).
				for _, target := range this.logConfig {
					target := target
					origLogConfig.Targets = append(origLogConfig.Targets, &target)
				}

				//Adding monitor config
				this.ddbConfigMonitor.AddMonitorConfig(&origLogConfig, &updatedLogConfig, DDB_LOG_CONFIG_NAME, configLogAna)
			}
		}
	} else {
		log.MaestroErrorf("LogManager: No devicedb connection config available, configuration will not be fetched from devicedb\n")
	}

	return
}

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
	var symphony_client *wwrmi.Client
	var symphony_err error
	if config.Symphony != nil {
		symphony_client, symphony_err = wwrmi.GetMainClient(config.Symphony)
	} else {
		fmt.Printf("Symphony / RMI API server not configured.\n")
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
			fmt.Printf("\nFound toCloud target-------->\n")
			opts.NumBanks = defaults.NUMBER_BANKS_WEBLOG
			//			DEBUG(_count := 0)
			if config.Symphony != nil && symphony_client != nil && symphony_err == nil {
				opts.TargetCB = wwrmi.TargetCB
			} else {
				log.MaestroError("Log: 'toCloud' target is enabled, but Symphony API is not configured. Will not work.")
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
					fmt.Printf("ERROR on creating target: %s\n", err.Str)
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
		if symphony_err != nil {
			log.MaestroErrorf("Symphony / RMI client is not configured correctly or has failed: %s\n", symphony_err.Error())
		} else {
			symphony_client.StartWorkers()
			log.MaestroSuccess("Maestro RMI workers started")
			log.MaestroInfo("Symphony / RMI client workers started.")
		}
	}

	client := log.NewSymphonyClient("http://127.0.0.1:9443/submitLog/1", config.ClientId, defaults.NUMBER_BANKS_WEBLOG, 30*time.Second)
	client.Start()

	go inst.initDeviceDBConfig()

	return
}
