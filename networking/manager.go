package networking

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
 * This is the network manager.
 *
 * It is responsible for
 * - setting up the host's network interface
 * - monitoring the state of network interfaces
 * - doing specific things on network up / down
 * - running an mDNS server, to broadcast certain things about the gateway
 * - maintaining the state of interfaces in the storage subsystem
 */

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/armPelionEdge/maestro/debugging"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/maestroConfig"
	"github.com/armPelionEdge/maestro/networking/arp"
	"github.com/armPelionEdge/maestro/storage"
	"github.com/armPelionEdge/maestro/tasks"
	"github.com/armPelionEdge/maestroSpecs"
	"github.com/armPelionEdge/maestroSpecs/netevents"
	"github.com/armPelionEdge/netlink"
	"github.com/armPelionEdge/stow"
	"github.com/armPelionEdge/structmutate"
	"github.com/boltdb/bolt"
	"golang.org/x/sys/unix"

	"github.com/armPelionEdge/dhcp4"
	"github.com/armPelionEdge/dhcp4client"
)

const (
	LOG_PREFIX = "NetworkManager: "
)

type NetlinkAddr netlink.Addr

// internal wrapper, used to store network settings
type NetworkInterfaceData struct {
	IfName string `json:"name"`
	// current set Address
	//	PrimaryAddress NetlinkAddr `json:"primary_addr"`
	CurrentIPv4Addr net.IP
	// not used yet
	CurrentIPv6Addr net.IP
	// the settings as they should be
	StoredIfconfig *maestroSpecs.NetIfConfigPayload
	// the settings as they are right now
	RunningIfconfig *maestroSpecs.NetIfConfigPayload
	// true if we are waiting on a response
	// i.e. in mid-action on getting a lease
	waitingForLease bool
	// the last lease information
	DhcpLease *DhcpLeaseInfo

	dhcpRunning bool

	dhcpWaitOnShutdown chan networkThreadMessage
	dhcpWorkerControl  chan networkThreadMessage
	interfaceChange    chan networkThreadMessage
	// used to stop the netlink.LinkSubscribe() when we no longer
	// want to monitor the link:
	stopInterfaceMonitor chan struct{}
	// true if this interface had associated DNS entries
	hadDNS bool
	// last time the interface config was changed
	// ms since Unix epoch
	LastUpdated int64
	// updated everytime we see a netlink message about the interface
	lastFlags uint32
	flagsSet  bool
}

func (addr *NetlinkAddr) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("")
	debugging.DEBUG_OUT("NetlinkAddr1\n")
	if addr != nil && addr.IPNet != nil {
		debugging.DEBUG_OUT("NetlinkAddr2\n")
		buffer.WriteString(fmt.Sprintf("\"%s\"", addr.String()))
	} else {
		buffer.WriteString(fmt.Sprintf("null"))
	}
	return buffer.Bytes(), nil
}

func (ifdata *NetworkInterfaceData) MarshalJSON() ([]byte, error) {
	type Alias NetworkInterfaceData
	return json.Marshal(&struct {
		//		LastSeen int64 `json:"lastSeen"`
		*Alias
	}{
		//		LastSeen: u.LastSeen.Unix(),
		Alias: (*Alias)(ifdata),
	})
}

func (ifdata *NetworkInterfaceData) hasFlags() bool {
	return ifdata.flagsSet
}

func (ifdata *NetworkInterfaceData) setFlags(v uint32) {
	ifdata.flagsSet = true
	ifdata.lastFlags = v
}

const (
	DHCP_NETWORK_FAILURE_TIMEOUT_DURATION time.Duration = 30 * time.Second
	// used when an interface goes down, then back up, etc.
	DHCP_NET_TRANSITION_TIMEOUT time.Duration = 2 * time.Second
	c_DB_NAME                                 = "networkConfig"

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

type networkThreadMessage struct {
	cmd    int
	ifname string // sometimes used
}

// a wrapper around a bytes.Buffer to provide thread safety
type dnsBuf struct {
	ifname   string
	buf      bytes.Buffer
	lock     sync.Mutex
	disabled bool
}

func (dnsbuf *dnsBuf) disable() {
	dnsbuf.lock.Lock()
	dnsbuf.disabled = true
	dnsbuf.lock.Unlock()
}

func (dnsbuf *dnsBuf) enable() {
	dnsbuf.lock.Lock()
	dnsbuf.disabled = false
	dnsbuf.lock.Unlock()
}

func init() {
	arp.SetupLog(nmLogDebugf, nmLogErrorf, nmLogSuccessf)
}

type ConfigCommit struct {
	// Set this flag to true for the changes to commit, if this flag is false
	// the changes to configuration on these structs will not acted upon
	// by network manager. For exmaple, this flag will be initially false
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

// initialized in init()
type networkManagerInstance struct {
	// for persistence
	db              *bolt.DB
	networkConfigDB *stow.Store
	// internal structures - thread safe hashmaps
	byInterfaceName    sync.Map // map of identifier to struct -> 'eth2':&networkInterfaceData{}
	indexToName        sync.Map
	watcherWorkChannel chan networkThreadMessage

	newInterfaceMutex sync.Mutex

	// mostly used for testing
	threadCountEnable         bool
	interfaceThreadCount      int
	interfaceThreadCountMutex sync.Mutex
	threadCountChan           chan networkThreadMessage
	networkConfig             *maestroSpecs.NetworkConfigPayload
	waitForDeviceDB           bool

	//Configs to be used for connecting to devicedb
	ddbConnConfig    *maestroConfig.DeviceDBConnConfig
	ddbConfigMonitor *maestroConfig.DDBMonitor
	ddbConfigClient  *maestroConfig.DDBRelayConfigClient
	CurrConfigCommit ConfigCommit

	// DNS related
	writeDNS bool // if true, then DNS will be written out once interfaces are processed
	// a buffer used to amalgamate all the DNS servers provided from various sources
	// (typically a single interface however)
	// a map of interface name to dnsBufs
	dnsPerInterface sync.Map
	//	dnsResolvConf *bytes.Buffer
	// this is the interface where we have set a default route
	// for the primary (default) routing table
	primaryTable routingTable
}

const DDB_NETWORK_CONFIG_NAME string = "MAESTRO_NETWORK_CONFIG_ID"
const DDB_NETWORK_CONFIG_COMMIT_FLAG string = "MAESTRO_NETWORK_CONFIG_COMMIT_FLAG"
const DDB_NETWORK_CONFIG_CONFIG_GROUP_ID string = "netgroup"

func (inst *networkManagerInstance) getNewDnsBufferForInterface(ifname string) (ret *dnsBuf) {
	ret = new(dnsBuf)
	ret.ifname = ifname
	inst.dnsPerInterface.Store(ifname, ret)
	return
}

func (inst *networkManagerInstance) getDnsBufferForInterface(ifname string) (ret *dnsBuf) {
	retI, ok := inst.dnsPerInterface.Load(ifname)
	if ok {
		ret, ok = retI.(*dnsBuf)
		if !ok {
			log.MaestroError("NetworkManager: internal error @ getDnsBufferForInterface()")
		}
	}
	return
}

func (inst *networkManagerInstance) removeDnsBufferForInterface(ifname string) {
	inst.dnsPerInterface.Delete(ifname)
}

// this write out the final DNS file to either the default location
// or the alternate (AltResolvConf) if set
func (inst *networkManagerInstance) finalizeDns() (err error) {
	path := resolvConfPath
	if len(inst.networkConfig.AltResolvConf) > 0 {
		path = inst.networkConfig.AltResolvConf
	}

	var file *os.File

	// overwrite and erase contents of old file.
	file, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, dnsFilePerms)
	if err != nil {
		return
	}

	defer file.Close()

	writer := bufio.NewWriter(file)

	now := time.Now()
	s := fmt.Sprintf(dnsTemplateFile, now.Format("Mon Jan 2 15:04:05 MST 2006"), now.Unix())
	// write the file header
	writer.WriteString(s)

	// if the config had any stated name servers, add them first
	if len(inst.networkConfig.Nameservers) > 0 {
		for _, ipS := range inst.networkConfig.Nameservers {
			// check to make sure its a valid IP, by parsing it using the go net package
			ip := net.ParseIP(ipS)
			if ip != nil {
				log.MaestroDebugf("NetworkManager: adding static DNS entry %s\n", ipS)
				writer.WriteString(fmt.Sprintf("nameserver %s\n", ipS))
			} else {
				log.MaestroErrorf("NetworkManager: top level config of static 'nameserver' %s is not a valid address.\n", ipS)
			}
		}

	}

	writeIf := func(key, val interface{}) bool {
		buf, ok := val.(*dnsBuf)
		if ok {
			buf.lock.Lock()
			if !buf.disabled {
				writer.Write(buf.buf.Bytes())
			} else {
				writer.Write([]byte(fmt.Sprintf("# skipping interface %s\n", buf.ifname)))
			}
			buf.lock.Unlock()
		} else {
			log.MaestroError("NetworkManager: internal error @ finalizeDNS")
		}
		return true
	}

	inst.dnsPerInterface.Range(writeIf)

	err = writer.Flush()
	if err == nil {
		log.MaestroSuccessf("NetworkManager: wrote out DNS resolv.conf (%s) ok\n", path)
	}
	debugging.DEBUG_OUT("finalizeDNS() wrote %s ok\n", path)
	return
}

var instance *networkManagerInstance

func (this *networkManagerInstance) enableThreadCount() {
	this.threadCountEnable = true
}
func (this *networkManagerInstance) incIfThreadCount() {
	if this.threadCountEnable {
		this.interfaceThreadCountMutex.Lock()
		this.interfaceThreadCount++
		this.interfaceThreadCountMutex.Unlock()
	}
}
func (this *networkManagerInstance) decIfThreadCount() {
	if this.threadCountEnable {
		this.interfaceThreadCountMutex.Lock()
		this.interfaceThreadCount--
		if this.interfaceThreadCount < 1 {
			debugging.DEBUG_OUT("submit to threadCountChan no_interface_threads\n")
			this.threadCountChan <- networkThreadMessage{cmd: no_interface_threads}
		}
		this.interfaceThreadCountMutex.Unlock()
	}
}
func (this *networkManagerInstance) waitForActiveInterfaceThreads(timeout_seconds int) (wastimeout bool) {
	if !this.threadCountEnable {
		log.MaestroError("NetworkManager: Called waitForActiveInterfaceThreads - threadCountEnable is false!")
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
func (this *networkManagerInstance) StorageInit(instance storage.MaestroDBStorageInterface) {
	gob.Register(&maestroSpecs.NetIfConfigPayload{})
	gob.Register(&NetworkInterfaceData{})
	gob.Register(&DhcpLeaseInfo{})
	gob.Register(&netlink.Addr{})
	gob.Register(&net.IPMask{})
	gob.Register(&net.IP{})
	gob.Register(&net.IPNet{})
	gob.Register(&dhcp4.Packet{})
	this.networkConfigDB = stow.NewStore(instance.GetDb(), []byte(c_DB_NAME))
}

// // called when the DB is open and ready
// StorageReady(instance MaestroDBStorageInterface)
// // called when the DB is initialized
// StorageInit(instance MaestroDBStorageInterface)
// // called when storage is closed
// StorageClosed(instance MaestroDBStorageInterface)

func (this *networkManagerInstance) StorageReady(instance storage.MaestroDBStorageInterface) {
	this.db = instance.GetDb()
	err := this.loadAllInterfaceData()
	if err != nil {
		log.MaestroErrorf("NetworkManager: Failed to load existing network interface settings: %s\n", err.Error())
	}
}

func (this *networkManagerInstance) StorageClosed(instance storage.MaestroDBStorageInterface) {

}

func (this *networkManagerInstance) SetInterfacesAsJson(data []byte) error {

	var err error
	configs := make([]maestroSpecs.NetIfConfigPayload, 0)

	err = json.Unmarshal(data, &configs)
	if err != nil {
		return err
	}

	dirty := false
	for _, config := range configs {
		ok, problem := validateIfConfig(&config)
		if !ok {
			log.MaestroErrorf("NetworkManager: Interface config problem: \"%s\"  Skipping interface config.\n", problem)
			return errors.New("Invalid config")
		}
		log.MaestroDebugf("NetworkManager: SetInterfacesAsJson: Searching for iface IfName=%s\n", config.IfName)
		var iface = this.getInterfaceData(config.IfName)
		if iface == nil {
			return fmt.Errorf("Interface not found: %s", config.IfName)
		}
		log.MaestroDebugf("NetworkManager: SetInterfacesAsJson: Config iface %s\n", iface.IfName)
		err = this.SetInterfaceConfigByName(config.IfName, &config)
		if err != nil {
			return err
		}
		dirty = true
	}

	if dirty {
		return this.setupInterfaces()
	} else {
		return nil
	}
}

func (this *networkManagerInstance) GetInterfacesAsJson(enabled_only bool, up_only bool) ([]byte, error) {
	ifs := make([]*NetworkInterfaceData, 0)

	this.byInterfaceName.Range(func(key, value interface{}) bool {
		if value == nil {
			debugging.DEBUG_OUT("CORRUPTION in hashmap - have null value pointer\n")
			return true
		}
		// ifname := "<interface name>"
		// ifname, _ = item.Key.(string)

		ifdata := value.(*NetworkInterfaceData)
		ifs = append(ifs, ifdata)
		log.MaestroDebugf("NetworkManager: GetInterfacesAsJson: found if [%s]\n", ifdata.IfName)
		return true
	})

	return json.Marshal(ifs)
}

// func (inst *networkManagerInstance) resetDNSBuffer() {
// 	inst.dnsResolvConf = bytes.NewBufferString(dnsTemplateFile)
// }

func newNetworkManagerInstance() (ret *networkManagerInstance) {
	ret = new(networkManagerInstance)
	ret.watcherWorkChannel = make(chan networkThreadMessage, 10) // use a buffered channel
	ret.threadCountChan = make(chan networkThreadMessage)
	ret.waitForDeviceDB = true
	// ret.resetDNSBuffer()
	go ret.watchInterfaces()
	return
}

func GetInstance() *networkManagerInstance {
	if instance == nil {
		instance = newNetworkManagerInstance()
		// provide a blank network config for now (until one is provided)
		instance.networkConfig = new(maestroSpecs.NetworkConfigPayload)
		storage.RegisterStorageUser(instance)
		// internalTicker = time.NewTicker(time.Second * time.Duration(defaults.TASK_MANAGER_CLEAR_INTERVAL))
		// controlChan = make(chan *controlToken, 100) // buffer up to 100 events
	}
	return instance
}

func newIfData(ifname string, ifconf *maestroSpecs.NetIfConfigPayload) (ret *NetworkInterfaceData) {
	ret = new(NetworkInterfaceData)
	ret.IfName = ifname
	// should be set when the config is applied
	// ret.RunningIfconfig = new(maestroSpecs.NetIfConfigPayload)
	// ret.RunningIfconfig.IfName = ifname
	ret.StoredIfconfig = new(maestroSpecs.NetIfConfigPayload)
	*ret.StoredIfconfig = *ifconf
	return
}

func (this *NetworkInterfaceData) setRunningConfig(ifconfig *maestroSpecs.NetIfConfigPayload) (ret *NetworkInterfaceData) {
	ret.RunningIfconfig = new(maestroSpecs.NetIfConfigPayload)
	*ret.RunningIfconfig = *ifconfig // copy that entire struct
	return
}

func validateIfConfig(ifconf *maestroSpecs.NetIfConfigPayload) (ok bool, problem string) {
	ok = true
	if len(ifconf.IfName) < 1 {
		ok = false
		problem = "Interface is missing IfName / if_name field"
	}
	return
}

// to be ran on startup, or new interface creation
//
func resetIfDataFromStoredConfig(netdata *NetworkInterfaceData) error {
	var ifconf *maestroSpecs.NetIfConfigPayload
	if netdata.StoredIfconfig != nil {
		ifconf = netdata.StoredIfconfig
	} else {
		log.MaestroWarnf("Interface [%s] does not have a StoredConfig. Why?\n", netdata.IfName)
		if netdata.RunningIfconfig != nil {
			ifconf = netdata.RunningIfconfig
		} else {
			return errors.New("No interface configuration")
		}
	}

	if !ifconf.DhcpV4Enabled {
		netdata.dhcpRunning = false
		netdata.DhcpLease = nil
	}

	if ifconf.IfName != netdata.IfName {
		return errors.New("Corruption in configuration")
	}

	return nil
}

func updateIfConfigFromStored(netdata *NetworkInterfaceData) (ok bool, err error) {
	if netdata.StoredIfconfig != nil {
		ok = true
	} else {
		log.MaestroError("NetworkManager: CRITICAL updateIfConfigFromStored encountered nil StoredIfconfig")
		ok = false
		err = errors.New("No StoredIfconfig")
	}
	return
}

func (this *networkManagerInstance) resetLinks() {
	log.MaestroDebugf("NetworkManager: resetLinks: reset all network links\n")
	//This function brings all the intfs down and puts everything back in pristine state
	this.byInterfaceName.Range(func(key, value interface{}) bool {
		if value == nil {
			debugging.DEBUG_OUT("NetworkManager: resetLinks: Invalid Entry in hashmap - have null value pointer\n")
			return true
		}
		ifname := "<interface name>"
		ifname, _ = key.(string)
		log.MaestroDebugf("NetworkManager: found existing key for if [%s]\n", ifname)
		if value != nil {
			ifdata := value.(*NetworkInterfaceData)
			//First kill the DHCP routine if its running
			if ifdata.dhcpRunning {
				log.MaestroDebugf("NetworkManager: resetLinks: Stopping DHCP routine for if %s\n", ifname)
				ifdata.dhcpWorkerControl <- networkThreadMessage{cmd: stop_and_release_IP}
				// wait on that shutdown
				<-ifdata.dhcpWaitOnShutdown
				//Set the flag
				ifdata.dhcpRunning = false
			}
			//ifdata := (*NetworkInterfaceData)(item.Value)
			log.MaestroDebugf("NetworkManager: Remove the interface config/settings from hashmap for if [%s]\n", ifname)
			//Clear/Remove the interface config/settings from hashmap
			link, err := GetInterfaceLink(ifname, -1)
			if err == nil && link != nil {
				ifname = link.Attrs().Name
				// ok - need to bring interface down to set Mac
				log.MaestroDebugf("NetworkManager: resetLinks: bringing if %s down\n", ifname)
				err2 := netlink.LinkSetDown(link)
				if err2 != nil {
					log.MaestroErrorf("NetworkManager: resetLinks: failed to bring if %s down - %s\n", ifname, err2.Error())
				}
				//Now we can remove the entry from hashmap
				this.byInterfaceName.Delete(key)
			}
		}
		return true
	})
}

func (this *networkManagerInstance) submitConfig(config *maestroSpecs.NetworkConfigPayload) {
	this.networkConfig = config

	//reset all current config if this function is called again
	this.resetLinks()

	// NOTE: this is only for the config file initial start
	// NOTE: the database must already be loaded and read
	// should only be called once. Is only called by InitNetworkManager()
	for _, ifconf := range config.Interfaces {
		log.MaestroInfof("NetworkManager: submitConfig: if %s\n", ifconf.IfName)
		// look in database
		conf_ok, problem := validateIfConfig(ifconf)
		if !conf_ok {
			log.MaestroErrorf("NetworkManager: Interface config problem: \"%s\"  Skipping interface config.\n", problem)
			continue
		}
		ifname := ifconf.IfName
		debugging.DEBUG_OUT("------> ifname: [%s]\n", ifname)
		var storedifconfig NetworkInterfaceData
		err := this.networkConfigDB.Get(ifname, &storedifconfig)
		if err != nil {
			log.MaestroInfof("NetworkManager: submitConfig: Get failed: %v\n", err)
			if err != stow.ErrNotFound {
				log.MaestroErrorf("NetworkManager: problem with database on if %s - Get: %s\n", ifname, err.Error())
			} else {
				// ret = new(NetworkInterfaceData)
				// ret.IfName = ifconf.IfName
				newconf := newIfData(ifconf.IfName, ifconf)
				this.byInterfaceName.Store(ifname, newconf)
				err = this.commitInterfaceData(ifname)
				if err != nil {
					log.MaestroErrorf("NetworkManager: Problem (1) storing config for interface [%s]: %s\n", ifconf.IfName, err)
				}
				// // ret.RunningIfconfig = new(maestroSpecs.NetIfConfigPayload)
				// // ret.RunningIfconfig.IfName = ifname
				// ret.StoredIfconfig = new(maestroSpecs.NetIfConfigPayload)
				// ret.StoredIfconfig.IfName = ifname

				// this.byInterfaceName.Set(ifname,unsafe.Pointer(ret))
				// this.commitInterfaceData(ifname)
				// // this.byInterfaceName.Set(ifname,unsafe.Pointer(ret))
			}
		} else {
			// entry existed, so override
			if ifconf.Existing == "replace" {
				// same as above, brand new
				newconf := newIfData(ifconf.IfName, ifconf)
				this.byInterfaceName.Store(ifname, newconf)
				err = this.commitInterfaceData(ifname)
				if err != nil {
					log.MaestroErrorf("NetworkManager: Problem (2) storing config for interface [%s]: %s\n", ifconf.IfName, err)
				} else {
					log.MaestroInfof("NetworkManager: interface [%s] - setting \"replace\"\n", ifconf.IfName)
				}

			} else if ifconf.Existing == "override" {
				debugging.DEBUG_OUT("ifconfig: %+v\n", ifconf)
				// override fields where the incoming config has data
				if storedifconfig.StoredIfconfig != nil {
					debugging.DEBUG_OUT("merging...\n")
					// structmutate.SetupUtilsLogs(func (format string, a ...interface{}) {
					//     s := fmt.Sprintf(format, a...)
					//     fmt.Printf("[debug-typeutils]  %s\n", s)
					// },nil)
					err = structmutate.MergeInStruct(storedifconfig.StoredIfconfig, ifconf)
					if err != nil {
						// should not happen
						log.MaestroErrorf("NetworkManager: Failed to merge in config file data. %s\n", err.Error())
					}
				} else {
					storedifconfig.StoredIfconfig = ifconf
				}
				debugging.DEBUG_OUT("storedifconfig: %+v\n", storedifconfig.StoredIfconfig)
				this.byInterfaceName.Store(ifname, &storedifconfig)
				err = this.commitInterfaceData(ifname)
				if err != nil {
					log.MaestroErrorf("NetworkManager: Problem (3) storing config for interface [%s]: %s\n", ifconf.IfName, err)
				} else {
					log.MaestroInfof("NetworkManager: interface [%s] - setting \"override\"\n", ifconf.IfName)
				}

			} else { // else do nothingm db has priority
				this.byInterfaceName.Store(ifname, &storedifconfig)
			}
		}

	}
}

func (this *networkManagerInstance) getIfFromDb(ifname string) (ret *NetworkInterfaceData) {
	err := this.networkConfigDB.Get(ifname, ret)
	if err != nil {
		//            debugging.DEBUG_OUT("getOrNewInterfaceData: %s - error %s\n",ifname,err.Error())
		if err != stow.ErrNotFound {
			log.MaestroErrorf("NetworkManager: problem with database Get: %s\n", err.Error())
		}
	}
	return
}

func (this *networkManagerInstance) getOrNewInterfaceData(ifname string) (ret *NetworkInterfaceData) {
	this.newInterfaceMutex.Lock()
	pdata, ok := this.byInterfaceName.Load(ifname)
	if ok {
		if pdata == nil {
			ret = this.getIfFromDb(ifname)
			if ret == nil {
				ret = new(NetworkInterfaceData)
				ret.IfName = ifname
				ret.RunningIfconfig = new(maestroSpecs.NetIfConfigPayload)
				ret.RunningIfconfig.IfName = ifname
				ret.StoredIfconfig = new(maestroSpecs.NetIfConfigPayload)
				ret.StoredIfconfig.IfName = ifname
			}
		} else {
			ret = pdata.(*NetworkInterfaceData)
		}
		this.byInterfaceName.Store(ifname, ret)
		this.commitInterfaceData(ifname)
		//        debugging.DEBUG_OUT("HERE getOrNewInterfaceData ---------*********************-------------- %s: %+v\n",ifname,ret)
	} else {
		// ok, let's try the database
		ret = this.getIfFromDb(ifname)
		if ret == nil {
			ret = new(NetworkInterfaceData)
			ret.IfName = ifname
			ret.RunningIfconfig = new(maestroSpecs.NetIfConfigPayload)
			ret.RunningIfconfig.IfName = ifname
			ret.StoredIfconfig = new(maestroSpecs.NetIfConfigPayload)
			ret.StoredIfconfig.IfName = ifname
			this.byInterfaceName.Store(ifname, ret)
			this.commitInterfaceData(ifname)
		} else {
			//            debugging.DEBUG_OUT("HERE(2) getOrNewInterfaceData ---------*********************-------------- %s: %+v\n",ifname,ret)
			// store in in-memory map
			this.byInterfaceName.Store(ifname, ret)
		}
	}
	this.newInterfaceMutex.Unlock()
	return
}

func (this *networkManagerInstance) getInterfaceData(ifname string) (ret *NetworkInterfaceData) {
	pdata, ok := this.byInterfaceName.Load(ifname)
	if ok {
		ret = pdata.(*NetworkInterfaceData)
	}
	return
}

// loads all existing data from the DB, for an interface
// If one or more interfaces data have problems, it will keep loading
// If reading the DB fails completely, it will error out
func (this *networkManagerInstance) loadAllInterfaceData() (err error) {
	var temp NetworkInterfaceData
	this.networkConfigDB.IterateIf(func(key []byte, val interface{}) bool {
		ifname := string(key[:])
		ifdata, ok := val.(*NetworkInterfaceData)
		if ok {
			err2 := resetIfDataFromStoredConfig(ifdata)
			if err2 == nil {
				this.newInterfaceMutex.Lock()
				// if there is an existing in-memory entry, overwrite it
				log.MaestroInfof("NetworkManager:loadAllInterfaceData: Loading config for: %s.\n", ifname)
				this.byInterfaceName.Store(ifname, ifdata)
				this.newInterfaceMutex.Unlock()
				debugging.DEBUG_OUT("loadAllInterfaceData() see if: %s --> %+v\n", ifname, ifdata)
			} else {
				log.MaestroErrorf("NetworkManager: Critical problem with interface [%s] config. Not loading config.\n", ifname)
			}
		} else {
			err = errors.New("Internal DB corruption")
			debugging.DEBUG_OUT("NetworkManager: internal DB corruption - @if %s\n", ifname)
			log.MaestroError("NetworkManager: internal DB corruption")
		}
		return true
	}, &temp)
	return
}

func (this *networkManagerInstance) DoesInterfaceHaveValidConfig(ifname string) (err error, ok bool, ifconfig *maestroSpecs.NetIfConfigPayload) {
	ifdata := this.getInterfaceData(ifname)
	if ifdata != nil && ifdata.StoredIfconfig != nil {
		ifconfig = ifdata.StoredIfconfig
		ok, s := validateIfConfig(ifconfig)
		if !ok {
			err = errors.New("Config failed validation: " + s)
		}
	} else {
		err = errors.New("No config")
	}
	return
}

func (this *networkManagerInstance) SetInterfaceConfigByName(ifname string, ifconfig *maestroSpecs.NetIfConfigPayload) (err error) {
	ifdata := this.getOrNewInterfaceData(ifname)
	ifdata.StoredIfconfig = ifconfig
	this.commitInterfaceData(ifname)
	return
}

func (this *networkManagerInstance) setupInterfaces() (err error) {

	this.byInterfaceName.Range(func(key, value interface{}) bool {
		if value == nil {
			debugging.DEBUG_OUT("CORRUPTION in hashmap - have null value pointer\n")
			return true
		}
		ifname := "<interface name>"
		//ifname, _ = item.Key.(string)
		ifname, _ = key.(string)
		log.MaestroDebugf("NetworkManager: see existing setup for if [%s]\n", ifname)
		if value != nil {
			ifdata := value.(*NetworkInterfaceData)
			if ifdata == nil {
				log.MaestroErrorf("NetworkManager: Interface [%s] does not have an interface data structure.\n", ifname)
				return true
			}

			var ifconfig *maestroSpecs.NetIfConfigPayload

			if ifdata.StoredIfconfig != nil {
				ifconfig = ifdata.StoredIfconfig
			} else if ifdata.RunningIfconfig != nil {
				log.MaestroWarnf("NetworkManager: unusual, StoredIfconfig for if [%s] is nil. using RunningIfconfig\n", ifname)
				ifconfig = ifdata.RunningIfconfig
			} else {
				log.MaestroErrorf("NetworkManager: unusual, StoredIfconfig & RunningIfconfig for if [%s] is nil. skipping interface setup\n", ifname)
				return true
			}

			// if ifconfig == nil {
			//     log.MaestroErrorf("Interface [%s] did not have a valid ifconfig data structure.\n",ifdata.IfName)
			//     continue
			// }

			if ifconfig.DhcpV4Enabled {
				log.MaestroInfof(LOG_PREFIX+"Interface [%s] has DHCP enabled.\n", ifconfig.IfName)
			} else {
				log.MaestroInfof(LOG_PREFIX+"Interface [%s] has a static IP config.\n", ifconfig.IfName)
			}

			// create an Internal task and submit it
			conf1 := new(maestroSpecs.NetIfConfigPayload)
			*conf1 = *ifconfig

			op := new(maestroSpecs.NetInterfaceOpPayload)
			op.Type = maestroSpecs.OP_TYPE_NET_INTERFACE
			op.Op = maestroSpecs.OP_UPDATE_ADDRESS
			op.IfConfig = conf1
			op.TaskId = "setup_existing_" + ifconfig.IfName

			task := new(tasks.MaestroTask)

			task.Id = "setup_existing_" + ifconfig.IfName
			task.Src = "SetupExistingInterfaces"
			task.Op = op

			this.SubmitTask(task)

		}
		return true
	})

	return
}

//This function calls the setupInterfaces with the current config and then start connection with devicedb by calling initDeviceDBConfig
//Note that call to initDeviceDBConfig is done as a go routine.
func (this *networkManagerInstance) SetupExistingInterfaces() (err error) {
	log.MaestroInfof("NetworkManager: Setup the intfs using initial boot config first: %v:%v\n", this.networkConfig, this.networkConfig.Interfaces)

	if this.networkConfig.Disable == true {
		log.MaestroInfo("NetworkManager: Network management is disabled.  No interfaces to bring up.\n")
		return
	}

	//Setup the intfs using initial boot config first
	this.setupInterfaces()

	//Try setup the device using DeviceDB config now
	go this.initDeviceDBConfig()

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
func (this *networkManagerInstance) initDeviceDBConfig() error {
	var totalWaitTime int = 0
	var loopWaitTime int = INITIAL_DEVICEDB_STATUS_CHECK_INTERVAL_IN_SECS
	var err error
	//TLS config to connect to devicedb
	var tlsConfig *tls.Config

	if this.ddbConnConfig == nil {
		log.MaestroErrorf("NetworkManager: No devicedb connection config available, configuration will not be fetched from devicedb\n")
		return errors.New("No devicedb connection config available, configuration will not be fetched from devicedb\n")
	}

	log.MaestroInfof("NetworkManager: Found valid devicedb connection config, try connecting and fetching the config from devicedb: uri:%s prefix: %s bucket:%s id:%s cert:%s\n",
		this.ddbConnConfig.DeviceDBUri, this.ddbConnConfig.DeviceDBPrefix, this.ddbConnConfig.DeviceDBBucket, this.ddbConnConfig.RelayId, this.ddbConnConfig.CaChainCert)

	//Device DB config uses deviceid as the relay_id, so uset that to set the hostname
	log.MaestroWarnf("NetworkManager: Setting hostname: %s\n", this.ddbConnConfig.RelayId)
	err = syscall.Sethostname([]byte(this.ddbConnConfig.RelayId))
	if err != nil {
		log.MaestroErrorf("NetworkManager: Failed to set hostname (ignoring...): %v\n", err)
	}

	relayCaChain, err := ioutil.ReadFile(this.ddbConnConfig.CaChainCert)
	if err != nil {
		log.MaestroErrorf("NetworkManager: Unable to access ca-chain-cert file at: %s\n", this.ddbConnConfig.CaChainCert)
		return errors.New(fmt.Sprintf("NetworkManager: Unable to access ca-chain-cert file at: %s, err = %v\n", this.ddbConnConfig.CaChainCert, err))
	}

	caCerts := x509.NewCertPool()

	if !caCerts.AppendCertsFromPEM(relayCaChain) {
		log.MaestroErrorf("CA chain loaded from %s is not valid: %v\n", this.ddbConnConfig.CaChainCert, err)
		return errors.New(fmt.Sprintf("CA chain loaded from %s is not valid\n", this.ddbConnConfig.CaChainCert))
	}

	tlsConfig = &tls.Config{
		RootCAs: caCerts,
	}

	this.ddbConfigClient = maestroConfig.NewDDBRelayConfigClient(
		tlsConfig,
		this.ddbConnConfig.DeviceDBUri,
		this.ddbConnConfig.RelayId,
		this.ddbConnConfig.DeviceDBPrefix,
		this.ddbConnConfig.DeviceDBBucket)

	for totalWaitTime < MAX_DEVICEDB_WAIT_TIME_IN_SECS {
		log.MaestroInfof("initDeviceDBConfig: checking devicedb availability\n")
		if this.ddbConfigClient.IsAvailable() || !this.waitForDeviceDB {
			break
		} else {
			log.MaestroWarnf("initDeviceDBConfig: devicedb is not running. retrying in %d seconds", loopWaitTime)
			time.Sleep(time.Second * time.Duration(loopWaitTime))
			totalWaitTime += loopWaitTime
			//If we cant connect in first 6 minutes, check much less frequently for next 24 hours hoping that devicedb may come up later.
			if totalWaitTime > LOOP_WAIT_TIME_INCREMENT_WINDOW {
				loopWaitTime = INCREASED_DEVICEDB_STATUS_CHECK_INTERVAL_IN_SECS
			}
		}

		//After 24 hours just assume its never going to come up stop waiting for it and break the loop
		if totalWaitTime >= MAX_DEVICEDB_WAIT_TIME_IN_SECS {
			log.MaestroErrorf("initDeviceDBConfig: devicedb is not running, cannot fetch config from devicedb")
			return errors.New("devicedb is not running, cannot fetch config from devicedb")
		}
	}
	log.MaestroInfof("initDeviceDBConfig: successfully connected to devicedb\n")

	err = this.SetupDeviceDBConfig()
	if err != nil {
		log.MaestroErrorf("initDeviceDBConfig: error setting up config using devicedb: %v", err)
	} else {
		log.MaestroInfof("initDeviceDBConfig: successfully read config from devicedb\n")
	}

	return err
}

//SetupDeviceDBConfig reads the config from devicedb and if its new it applies the new config.
//It also sets up the config update handlers for all the tags/groups.
func (this *networkManagerInstance) SetupDeviceDBConfig() error {

	var err error
	var ddbNetworkConfig maestroSpecs.NetworkConfigPayload

	//Create a config analyzer object, required for registering the config change hook and diff the config objects.
	configAna := maestroSpecs.NewConfigAnalyzer(DDB_NETWORK_CONFIG_CONFIG_GROUP_ID)
	if configAna == nil {
		log.MaestroErrorf("NetworkManager: Failed to create config analyzer object, unable to fetch config from devicedb")
		return errors.New("Failed to create config analyzer object, unable to fetch config from devicedb")
	}

	err = this.ddbConfigClient.Config(DDB_NETWORK_CONFIG_NAME).Get(&ddbNetworkConfig)
	if err != nil {
		log.MaestroWarnf("NetworkManager: No network config found in devicedb or unable to connect to devicedb err: %v. Let's put the current running config: %v.\n", err, *this.networkConfig)
		err = this.ddbConfigClient.Config(DDB_NETWORK_CONFIG_NAME).Put(this.networkConfig)
		if err != nil {
			log.MaestroErrorf("NetworkManager: Unable to put network config in devicedb err:%v, config will not be monitored from devicedb\n", err)
			return errors.New(fmt.Sprintf("\nUnable to put network config in devicedb err:%v, config will not be monitored from devicedb\n", err))
		}
	} else {
		//We found a config in devicedb, lets try to use and reconfigure network if its an updated one
		log.MaestroInfof("NetworkManager: Found a valid config in devicedb [%v], will try to use and reconfigure network if its an updated one\n", ddbNetworkConfig)
		identical, _, _, err := configAna.DiffChanges(this.networkConfig, ddbNetworkConfig)
		if !identical && (err == nil) {
			//The configs are different, lets go ahead reconfigure the intfs
			log.MaestroDebugf("NetworkManager: New network config found from devicedb, reconfigure nework using new config\n")
			this.networkConfig = &ddbNetworkConfig
			this.submitConfig(this.networkConfig)
			//Setup the intfs using new config
			this.setupInterfaces()
			//Set the hostname again as we reconfigured the network
			log.MaestroWarnf("NetworkManager: Again setting hostname: %s\n", this.ddbConnConfig.RelayId)
			err = syscall.Sethostname([]byte(this.ddbConnConfig.RelayId))
			if err != nil {
				log.MaestroErrorf("NetworkManager: Failed to set hostname (ignoring...): %v\n", err)
			}
		} else {
			log.MaestroInfof("NetworkManager: New network config found from devicedb, but its same as boot config, no need to re-configure\n")
		}
	}

	//Since we are booting set the Network config commit flag to false
	log.MaestroWarnf("NetworkManager: Setting Network config commit flag to false\n")
	this.CurrConfigCommit.ConfigCommitFlag = false
	this.CurrConfigCommit.LastUpdateTimestamp = ""
	this.CurrConfigCommit.TotalCommitCountFromBoot = 0
	err = this.ddbConfigClient.Config(DDB_NETWORK_CONFIG_COMMIT_FLAG).Put(&this.CurrConfigCommit)
	if err != nil {
		log.MaestroErrorf("NetworkManager: Unable to put network commit flag in devicedb err:%v, config will not be monitored from devicedb\n", err)
		return errors.New(fmt.Sprintf("\nUnable to put network commit flag in devicedb err:%v, config will not be monitored from devicedb\n", err))
	}

	//Now start a monitor for the network config in devicedb
	err, this.ddbConfigMonitor = maestroConfig.NewDeviceDBMonitor(this.ddbConnConfig)
	if err != nil {
		log.MaestroErrorf("NetworkManager: Unable to create config monitor: %v\n", err)
		return errors.New(fmt.Sprintf("\n Unable to create config monitor: %v\n", err))
	}

	//Add config change hook for all property groups, we can use the same interface
	var networkConfigChangeHook NetworkConfigChangeHook

	configAna.AddHook("dhcp", networkConfigChangeHook)
	configAna.AddHook("if", networkConfigChangeHook)
	configAna.AddHook("ipv4", networkConfigChangeHook)
	configAna.AddHook("ipv6", networkConfigChangeHook)
	configAna.AddHook("mac", networkConfigChangeHook)
	configAna.AddHook("wifi", networkConfigChangeHook)
	configAna.AddHook("IEEE8021x", networkConfigChangeHook)
	configAna.AddHook("route", networkConfigChangeHook)
	configAna.AddHook("http", networkConfigChangeHook)
	configAna.AddHook("nameserver", networkConfigChangeHook)
	configAna.AddHook("gateway", networkConfigChangeHook)
	configAna.AddHook("dns", networkConfigChangeHook)
	configAna.AddHook("config_netif", networkConfigChangeHook)
	configAna.AddHook("config_network", networkConfigChangeHook)

	//Add monitor for this config
	var origNetworkConfig, updatedNetworkConfig maestroSpecs.NetworkConfigPayload
	//Provide a copy of current network config monitor to Config monitor, not the actual config we use, this would prevent config monitor
	//directly updating the running config(this.networkConfig).
	origNetworkConfig = *this.networkConfig

	//Adding monitor config
	this.ddbConfigMonitor.AddMonitorConfig(&origNetworkConfig, &updatedNetworkConfig, DDB_NETWORK_CONFIG_NAME, configAna)

	//Add config change hook for all property groups, we can use the same interface
	var commitConfigChangeHook CommitConfigChangeHook
	configAna.AddHook("config_commit", commitConfigChangeHook)

	//Add monitor for this object
	var updatedConfigCommit ConfigCommit
	log.MaestroInfof("NetworkManager: Adding monitor for config commit object\n")
	this.ddbConfigMonitor.AddMonitorConfig(&this.CurrConfigCommit, &updatedConfigCommit, DDB_NETWORK_CONFIG_COMMIT_FLAG, configAna)

	return nil
}

// ErrNoInterface is the error when no interface can be found by the reference data
var ErrNoInterface = errors.New("no interface")

// writes the interface data back to the database
func (mgr *networkManagerInstance) commitInterfaceData(ifname string) error {
	pdata, ok := mgr.byInterfaceName.Load(ifname)
	if ok {
		ifdata := pdata.(*NetworkInterfaceData)
		err := mgr.networkConfigDB.Put(ifname, ifdata)
		debugging.DEBUG_OUT("NetworkManager: --> commitInterfaceData() for [%s]\n", ifname)
		return err
	}
	return ErrNoInterface
}

// MUST be run as a go routine, with "go doDhcp()"
// the a simplestic approach - but works fine with the way go routines work.
// The Dhcp thread will wait until the lease ends, and then keep renewing.
// So this go routine may never end.  If this routine should be stopped,
// the dhcpWorkerControl channel is used to stop it, and release the IP
func (this *networkManagerInstance) doDhcp(ifname string, op maestroSpecs.NetInterfaceOperation) {
	var err error
	var newlease *DhcpLeaseInfo
	var ifdata *NetworkInterfaceData
	var timeout time.Duration

	this.incIfThreadCount()

	var showProgress = func(state int, addinfo string) bool {
		switch state {
		case dhcp4client.AtGetOffer:
			log.MaestroInfof("NetworkManager: DHCP if %s - @GetOffer", ifname)
		case dhcp4client.AtGetOfferUnicast:
			log.MaestroInfof("NetworkManager: DHCP if %s - @GetOfferUnicast", ifname)
		case dhcp4client.AtSendRequest:
			log.MaestroInfof("NetworkManager: DHCP if %s - @SendRequest", ifname)
		case dhcp4client.AtSendRenewalRequest:
			log.MaestroInfof("NetworkManager: DHCP if %s - @SendRenewalRequest", ifname)
		case dhcp4client.AtGetAcknowledgement:
			log.MaestroInfof("NetworkManager: DHCP if %s - @GetAcknowledgement", ifname)
		case dhcp4client.AtGetAckLoop:
			log.MaestroInfof("NetworkManager: DHCP if %s - @GetAckLoop", ifname)
		case dhcp4client.AtGetOfferLoop:
			log.MaestroInfof("NetworkManager: DHCP if %s - @GetOfferLoop", ifname)
		case dhcp4client.AtEndOfRequest:
			log.MaestroInfof("NetworkManager: DHCP if %s - @EndOfRequest", ifname)
		case dhcp4client.AtEndOfRenewal:
			log.MaestroInfof("NetworkManager: DHCP if %s - @EndOfRenewal", ifname)
		case dhcp4client.AtSendRenewalRequestInitReboot:
			log.MaestroInfof("NetworkManager: DHCP if %s - @AtSendRenewalRequestInitReboot", ifname)
		case dhcp4client.AtGetOfferLoopTimedOut:
			log.MaestroInfof("NetworkManager: DHCP if %s - @AtGetOfferLoopTimedOut", ifname)
		case dhcp4client.AtGetOfferError:
			log.MaestroInfof("NetworkManager: DHCP if %s - @AtGetOfferError - %s", ifname, addinfo)
		case dhcp4client.AtGetAckError:
			log.MaestroInfof("NetworkManager: DHCP if %s - @AtGetAckError - %s", ifname, addinfo)
		case dhcp4client.SyscallFailed:
			log.MaestroErrorf("NetworkManager: DHCP if %s - Sycall failed: %s", ifname, addinfo)
		}
		return true
	}

	requestopts := new(dhcp4client.DhcpRequestOptions)
	requestopts.AddRequestParam(dhcp4.OptionSubnetMask)
	requestopts.AddRequestParam(dhcp4.OptionRouter)
	requestopts.AddRequestParam(dhcp4.OptionDomainNameServer)
	requestopts.AddRequestParam(dhcp4.OptionHostName)
	requestopts.AddRequestParam(dhcp4.OptionDomainName)
	requestopts.AddRequestParam(dhcp4.OptionBroadcastAddress)
	requestopts.AddRequestParam(dhcp4.OptionNetworkTimeProtocolServers)
	requestopts.ProgressCB = showProgress

	timeout = 5

	ifdata = this.getInterfaceData(ifname)
	if ifdata != nil {
		ifdata.dhcpRunning = true
		// the channel can hold one next command
		ifdata.dhcpWorkerControl = make(chan networkThreadMessage, 1)
		ifdata.dhcpWaitOnShutdown = make(chan networkThreadMessage, 1)
	} else {
		log.MaestroWarnf("NetworkManager: Can't start DHCP lease routine, as interface '%s' is no longer managed!\n", ifname)
		this.decIfThreadCount()
		return
	}

	ifconfig := op.GetIfConfig()
	if ifconfig == nil {
		log.MaestroErrorf("NetworkManager: No interface config in operation Can't start DHCP thread for If %s\n", ifname)
		this.decIfThreadCount()
		return
	}

	if ifconfig.DhcpStepTimeout > 0 {
		requestopts.StepTimeout = time.Second * time.Duration(ifconfig.DhcpStepTimeout)
	} else {
		requestopts.StepTimeout = time.Second * time.Duration(defaultDhcpStepTimeout)
	}

	var remainrenewal int64
	now := time.Now().Unix()
	var workstate networkThreadMessage
	var nextstate *networkThreadMessage

	leaseinfo := ifdata.DhcpLease

	if leaseinfo != nil && leaseinfo.IsValid() {
		//		debugging.DEBUG_OUT("DHCP goDhcp() LEASE IS VALID \n")
		// Step #1
		// if still have a lease, and the lease is not out yet, then just set the IP
		// and engage the watcher for the lease
		//            if leaseinfo.LeaseEndTime
		remainrenewal = leaseinfo.renewalTime - now
		if remainrenewal < 30 {
			remainrenewal = leaseinfo.rebindTime - now
		}
		if remainrenewal >= 30 {
			log.MaestroInfo("DHCP lease is still valid. Setting up address, and waiting until renew.")
			// ifresult, err :=
			_, err := SetupInterfaceFromLease(ifconfig, leaseinfo)
			if err != nil {
				log.MaestroError("NetworkManager: Problem setting up from old lease. Will try new lease.")
				leaseinfo = nil
				ifdata.dhcpWorkerControl <- networkThreadMessage{cmd: dhcp_get_lease} // rebind?
			} else {
				// setup the interface fine.
				// So wait until timeout, then do a renew
				timeout = time.Duration(remainrenewal) * time.Second
				nextstate = &networkThreadMessage{cmd: dhcp_renew_lease}
			}
		} else {
			log.MaestroInfo("NetworkManager: DHCP: looks like lease is expired or almost expired. Let's get a new one.")
			ifdata.dhcpWorkerControl <- networkThreadMessage{cmd: dhcp_get_lease} // rebind?
		}
	} else {
		debugging.DEBUG_OUT("NetworkManager: doDhcp() DHCP: no lease or lease invalid! Will get new lease.\n")
		// ok - get a completely fresh lease then.
		ifdata.dhcpWorkerControl <- networkThreadMessage{cmd: dhcp_get_lease}
	}

	debugging.DEBUG_OUT("@DhcpLoop\n")
	// yeap, this is gonna run forever, or until it's shutdown
DhcpLoop:
	for {
		debugging.DEBUG_OUT("DhcpLoop to - if %s\n", ifname)
		select {
		case statechange := <-ifdata.interfaceChange:
			// happens if the interface goes up or down
			switch statechange.cmd {
			case state_LOWER_DOWN:
				// the interface has gone down. So we want to re-run our
				// default route out and DNS setup, to choose another interface if one
				// is available.

				ok, ifpref, r := this.primaryTable.findPreferredRoute()
				if ok {
					log.MaestroWarnf("NetworkManager:(DhcpLoop) preferred default route is on if %s - %+v\n", ifpref, r)
					err = this.primaryTable.setPreferredRoute(!this.networkConfig.DontOverrideDefaultRoute)
					if err == nil {
						log.MaestroWarnf("NetworkManager: set default route to %s %+v\n", ifpref, r)
					} else {
						log.MaestroErrorf("NetworkManager: error setting preferred route: %s\n", err.Error())
					}
				}
			case state_LOWER_UP:
				// the interface and it's old route have already been brought up, in the
				// handler below, for update := <-ch - however, we now need to see
				// if we should use an alternate network, by asking for DHCP
				// we call this without just sending ourselves an event, b/c we want the timeouts to be
				// very fast in this case.
				debugging.DEBUG_OUT("DhcpLoop - Saw LOWER_UP change - if %s. Checking for new DHCP?\n", ifname)
				ifdata = this.getInterfaceData(ifname)
				if ifdata != nil {

					if ifdata.CurrentIPv4Addr != nil {
						success, newlease, err := InitRebootDhcpLease(ifname, ifdata.CurrentIPv4Addr, requestopts)
						if err != nil {
							log.MaestroErrorf("NetworkManager: DHCP (InitReboot) for '%s' failed: %s\n", ifname, err.Error())
							timeout = DHCP_NET_TRANSITION_TIMEOUT
							nextstate = &networkThreadMessage{cmd: dhcp_get_lease}
							continue DhcpLoop
						}
						if success == dhcp4client.Rejected {
							nmLogWarnf("DHCP server rejected request. Will try new lease.\n")
							timeout = DHCP_NET_TRANSITION_TIMEOUT
							nextstate = &networkThreadMessage{cmd: dhcp_get_lease}
							continue DhcpLoop
						}
						if success == dhcp4client.Success && newlease != nil {
							nextstate = &networkThreadMessage{cmd: dhcp_renew_lease}
							timeout = time.Duration(newlease.renewalTime) * time.Second
						} else {
							log.MaestroErrorf("NetworkManager: DHCP for '%s' failed - no lease?\n", ifname)
							timeout = DHCP_NET_TRANSITION_TIMEOUT
							nextstate = &networkThreadMessage{cmd: dhcp_get_lease}
							continue DhcpLoop
						}
					} else {
						log.MaestroWarn("NetworkManager: DHCP (LOWER_UP) had no current IP! Bumping to discovery.")
						timeout = DHCP_NET_TRANSITION_TIMEOUT
						ifdata.dhcpWorkerControl <- networkThreadMessage{cmd: dhcp_get_lease}
						continue DhcpLoop
					}

					// ok, setup the interface.
					result, err := SetupInterfaceFromLease(ifdata.RunningIfconfig, newlease)
					if err != nil {
						log.MaestroErrorf("NetworkManager: Problem setting up from old lease. Will try new lease. Details: %s\n", err.Error())
						leaseinfo = nil
						ifdata.dhcpWorkerControl <- networkThreadMessage{cmd: dhcp_get_lease} // rebind?
					} else {
						// after we have a lease, and we managed to set it up, save it
						ifdata = this.getInterfaceData(ifname)
						if ifdata != nil {
							ifdata.DhcpLease = newlease
							ifdata.CurrentIPv4Addr = result.ipv4
							leaseinfo = newlease
							this.commitInterfaceData(ifname)
						}
						nextstate = &networkThreadMessage{cmd: dhcp_renew_lease}

						// // setup the interface fine.
						// // So wait until timeout, then do a renew
						// timeout = time.Duration(remainrenewal) * time.Second
						// nextstate = &networkThreadMessage{cmd:dhcp_renew_lease}
						timeout = time.Duration(newlease.renewalTime) * time.Second
					}

					// setup default route
					routeset, gw, err := setupDefaultRouteInPrimaryTable(this, ifdata.RunningIfconfig, leaseinfo)
					if routeset {
						log.MaestroInfof("NetworkManager: default route from DHCP: %s - recorded in primary table\n", gw)
					}
					if err != nil {
						log.MaestroErrorf("NetworkManager: error setting adding default route to primaryTable via DHCP: %s\n", err.Error())
					}
					// ok, now finalize the default route
					this.finalizePrimaryRoutes()
					// setup DNS
					dnsset, primarydns, err := AppendDNSResolver(ifdata.RunningIfconfig, leaseinfo, this.getNewDnsBufferForInterface(ifdata.IfName), this.networkConfig.DnsIgnoreDhcp)
					if dnsset {
						log.MaestroInfof("NetworkManager: adding DNS nameserver %s as primary\n", primarydns)
						ifdata.hadDNS = true
						err = this.finalizeDns()
					}
					if err != nil {
						log.MaestroErrorf("NetworkManager: error getting / setting DNS from lease / ifconfig: %s\n", err.Error())
					}

				} else {
					log.MaestroWarnf("NetworkManager: Stopping DHCP lease renewal, as interface '%s' is no longer managed!\n", ifname)
					break DhcpLoop
				}

			}
		case workstate = <-ifdata.dhcpWorkerControl:
			switch workstate.cmd {
			case dhcp_get_lease:
				debugging.DEBUG_OUT("DhcpLoop to - @dhcp_get_lease [%s]\n", ifname)
				ifdata = this.getInterfaceData(ifname)
				var success int
				if ifdata != nil {
					if ifdata.DhcpLease != nil {
						success, newlease, err = RequestOrRenewDhcpLease(ifname, ifdata.DhcpLease, requestopts)
						if err != nil {
							log.MaestroErrorf("NetworkManager: DHCP for '%s' failed: %s\n", ifname, err.Error())
							if err.Error() == "timeout" {
								// if this was indeed a timeout, let's try a fresh lease.
								nmLogDebugf("if: %s - Trying to get a fresh lease...\n", ifname)
								success, newlease, err = GetFreshDhcpLease(ifname, requestopts)
								if err != nil {
									log.MaestroErrorf("NetworkManager: DHCP (fresh lease) for '%s' failed: %s\n", ifname, err.Error())
									timeout = DHCP_NETWORK_FAILURE_TIMEOUT_DURATION
									nextstate = &networkThreadMessage{cmd: dhcp_get_lease}
									continue DhcpLoop
								}
							} else {
								timeout = DHCP_NETWORK_FAILURE_TIMEOUT_DURATION
								nextstate = &networkThreadMessage{cmd: dhcp_get_lease}
								continue DhcpLoop
							}
						}
						if success == dhcp4client.Rejected {
							nmLogWarnf("DHCP server rejected request. Will try new lease.\n")
							timeout = DHCP_NETWORK_FAILURE_TIMEOUT_DURATION
							ifdata.DhcpLease = nil
							nextstate = &networkThreadMessage{cmd: dhcp_get_lease}
							continue DhcpLoop
						}
					} else {
						// New lease. Easy, just ask for one
						nmLogDebugf("No existing lease for if %s. Requesting now.\n", ifname)
						success, newlease, err = RequestOrRenewDhcpLease(ifname, nil, requestopts)
						if err != nil {
							log.MaestroErrorf("NetworkManager: DHCP for '%s' failed: %s\n", ifname, err.Error())
							timeout = DHCP_NETWORK_FAILURE_TIMEOUT_DURATION
							nextstate = &networkThreadMessage{cmd: dhcp_get_lease}
							continue DhcpLoop
						}
						if success == dhcp4client.Rejected {
							nmLogWarnf("DHCP server rejected request. Will try new lease.\n")
							timeout = DHCP_NETWORK_FAILURE_TIMEOUT_DURATION
							ifdata.DhcpLease = nil
							nextstate = &networkThreadMessage{cmd: dhcp_get_lease}
							continue DhcpLoop
						}
					}

					if success != dhcp4client.Success {
						nmLogWarnf("DHCP did not provide an ACK. Will try new lease.\n")
						timeout = DHCP_NETWORK_FAILURE_TIMEOUT_DURATION
						ifdata.DhcpLease = nil
						nextstate = &networkThreadMessage{cmd: dhcp_get_lease}
						continue DhcpLoop
					}

					// ok, setup the interface.
					debugging.DEBUG_OUT("@DhcpLoop - setup interface\n")
					if ifdata.RunningIfconfig == nil {
						if ifconfig != nil {
							ifdata.RunningIfconfig = ifconfig
						} else {
							ifdata.RunningIfconfig = new(maestroSpecs.NetIfConfigPayload)
						}
						ifdata.RunningIfconfig.DhcpV4Enabled = true
					}

					result, err := SetupInterfaceFromLease(ifdata.RunningIfconfig, newlease)
					if err != nil {
						log.MaestroErrorf("NetworkManager: Problem setting up from old lease. Will try new lease. Details: %s\n", err.Error())
						leaseinfo = nil
						ifdata.dhcpWorkerControl <- networkThreadMessage{cmd: dhcp_get_lease} // rebind?
					} else {
						// after we have a lease, and we managed to set it up, save it
						ifdata = this.getInterfaceData(ifname)
						if ifdata != nil {
							ifdata.CurrentIPv4Addr = result.ipv4
							ifdata.DhcpLease = newlease
							leaseinfo = newlease
							log.MaestroDebugf("NetworkManager: Committing if data %s\n", ifname)
							this.commitInterfaceData(ifname)
						}
						nextstate = &networkThreadMessage{cmd: dhcp_renew_lease}

						// // setup the interface fine.
						// // So wait until timeout, then do a renew
						// timeout = time.Duration(remainrenewal) * time.Second
						// nextstate = &networkThreadMessage{cmd:dhcp_renew_lease}
						timeout = time.Duration(newlease.renewalTime) * time.Second

						// send gratuitous ARP to let people conforming to this
						// know we are around (helps with routers, etc.)
						if result.ipv4 != nil {
							arperr := arp.SendGratuitous(&arp.Gratuitous{
								IfaceName: ifconfig.IfName,
								IP:        result.ipv4,
							})
							if arperr != nil {
								nmLogErrorf("Erroring sending gratuitous ARP: %s\n", arperr.Error())
							} else {
								nmLogDebugf("Gratuitous ARP sent for IP %s\n", result.IPV4)
							}
						} else {
							nmLogErrorf("SetupInterfaceFromLease() has invalid results array or no address for if %s", ifname)
						}

					}

					if leaseinfo != nil {
						debugging.DEBUG_OUT("@DhcpLoop - setup gw and dns\n")
						// setup default route
						routeset, gw, err := setupDefaultRouteInPrimaryTable(this, ifdata.RunningIfconfig, leaseinfo)
						if routeset {
							log.MaestroDebugf("NetworkManager: default route from DHCP: %s - recorded in primary table\n", gw)
						}
						if err != nil {
							log.MaestroErrorf("NetworkManager: error setting adding default route to primaryTable via DHCP: %s\n", err.Error())
						}
						// ok, now finalize the default route
						this.finalizePrimaryRoutes()
						// ok, ifpref, r := this.primaryTable.findPreferredRoute()
						// if ok {
						// 	log.MaestroDebugf("NetworkManager: preferred default route is on if %s - %+v\n", ifpref, r)
						// 	err = this.primaryTable.setPreferredRoute()
						// 	if err == nil {
						// 		log.MaestroInfof("NetworkManager: set default route to %s %+v\n", ifpref, r)
						// 	} else {
						// 		log.MaestroErrorf("NetworkManager: error setting preferred route: %s", err.Error())
						// 	}
						// }
						//setupDefaultRouteInPrimaryTable(this,ifdata.RunningIfconfig,)
						// setup DNS
						dnsset, primarydns, err := AppendDNSResolver(ifdata.RunningIfconfig, leaseinfo, this.getNewDnsBufferForInterface(ifdata.IfName), this.networkConfig.DnsIgnoreDhcp)
						if dnsset {
							log.MaestroInfof("NetworkManager: adding DNS nameserver %s as primary\n", primarydns)
							ifdata.hadDNS = true
							err = this.finalizeDns()
						}
						if err != nil {
							log.MaestroErrorf("NetworkManager: error getting / setting DNS from lease / ifconfig: %s\n", err.Error())
						}
					}

				} else {
					log.MaestroWarnf("NetworkManager: Stopping DHCP lease request, as interface '%s' is no longer managed!\n", ifname)
					break DhcpLoop
				}
			case dhcp_renew_lease:
				debugging.DEBUG_OUT("DhcpLoop to - @dhcp_renew_lease [%s]\n", ifname)
				ifdata = this.getInterfaceData(ifname)
				var success int
				if ifdata != nil {

					if ifdata.DhcpLease != nil {
						success, newlease, err = RenewFromServer(ifname, ifdata.DhcpLease, requestopts)
						if err != nil {
							log.MaestroErrorf("NetworkManager: DHCP for '%s' failed: %s\n", ifname, err.Error())
							timeout = DHCP_NETWORK_FAILURE_TIMEOUT_DURATION
							nextstate = &networkThreadMessage{cmd: dhcp_get_lease}
							continue DhcpLoop
						}
						if success == dhcp4client.Rejected {
							nmLogWarnf("DHCP server rejected renewal request. Will try new lease.\n")
							timeout = DHCP_NET_TRANSITION_TIMEOUT
							ifdata.DhcpLease = nil
							nextstate = &networkThreadMessage{cmd: dhcp_get_lease}
							continue DhcpLoop
						}
						if newlease != nil {
							nextstate = &networkThreadMessage{cmd: dhcp_renew_lease}
							timeout = time.Duration(newlease.renewalTime) * time.Second
						} else {
							log.MaestroErrorf("NetworkManager: DHCP for '%s' failed - no lease?\n", ifname)
							timeout = DHCP_NETWORK_FAILURE_TIMEOUT_DURATION
							nextstate = &networkThreadMessage{cmd: dhcp_get_lease}
							continue DhcpLoop
						}

					} else {
						// No DhcpLease on file? we probably should not have gotten here,
						// but just immediately go back to Discovery
						log.MaestroWarn("NetworkManager: DHCP tried renew lease, but had no existing lease. Bumping to discovery.")
						ifdata.dhcpWorkerControl <- networkThreadMessage{cmd: dhcp_get_lease}
						continue DhcpLoop
					}

					if success != dhcp4client.Success {
						nmLogWarnf("DHCP renewal did not provide an ACK. Will try new lease.\n")
						timeout = DHCP_NET_TRANSITION_TIMEOUT
						ifdata.DhcpLease = nil
						nextstate = &networkThreadMessage{cmd: dhcp_get_lease}
						continue DhcpLoop
					}

					// ok, setup the interface.
					result, err := SetupInterfaceFromLease(ifdata.RunningIfconfig, newlease)
					if err != nil {
						log.MaestroErrorf("NetworkManager: Problem setting up from old lease. Will try new lease. Details: %s\n", err.Error())
						leaseinfo = nil
						ifdata.dhcpWorkerControl <- networkThreadMessage{cmd: dhcp_get_lease} // rebind?
					} else {
						// after we have a lease, and we managed to set it up, save it
						ifdata = this.getInterfaceData(ifname)
						if ifdata != nil {
							ifdata.DhcpLease = newlease
							ifdata.CurrentIPv4Addr = result.ipv4
							leaseinfo = newlease
							this.commitInterfaceData(ifname)
						}
						nextstate = &networkThreadMessage{cmd: dhcp_renew_lease}

						// // setup the interface fine.
						// // So wait until timeout, then do a renew
						// timeout = time.Duration(remainrenewal) * time.Second
						// nextstate = &networkThreadMessage{cmd:dhcp_renew_lease}
						timeout = time.Duration(newlease.renewalTime) * time.Second
					}
					// // after we have a lease save it
					// ifdata = this.getInterfaceData(ifname)
					// if ifdata != nil {
					//     ifdata.DhcpLease = newlease
					//     leaseinfo = newlease
					//     this.commitInterfaceData(ifname)
					// }
					// nextstate = &networkThreadMessage{cmd:dhcp_renew_lease}

					// setup default route
					routeset, gw, err := setupDefaultRouteInPrimaryTable(this, ifdata.RunningIfconfig, leaseinfo)
					if routeset {
						log.MaestroInfof("NetworkManager: default route from DHCP: %s - recorded in primary table\n", gw)
					}
					if err != nil {
						log.MaestroErrorf("NetworkManager: error setting adding default route to primaryTable via DHCP: %s\n", err.Error())
					}
					// ok, now finalize the default route
					this.finalizePrimaryRoutes()
					// ok, ifpref, r := this.primaryTable.findPreferredRoute()
					// if ok {
					// 	log.MaestroDebugf("NetworkManager: preferred default route is on if %s - %+v\n", ifpref, r)
					// 	err = this.primaryTable.setPreferredRoute()
					// 	if err == nil {
					// 		log.MaestroInfof("NetworkManager: set default route to %s %+v\n", ifpref, r)
					// 	} else {
					// 		log.MaestroErrorf("NetworkManager: error setting preferred route: %s\n", err.Error())
					// 	}
					// }
					// setup DNS
					dnsset, primarydns, err := AppendDNSResolver(ifdata.RunningIfconfig, leaseinfo, this.getNewDnsBufferForInterface(ifdata.IfName), this.networkConfig.DnsIgnoreDhcp)
					if dnsset {
						log.MaestroInfof("NetworkManager: adding DNS nameserver %s as primary\n", primarydns)
						ifdata.hadDNS = true
						err = this.finalizeDns()
					}
					if err != nil {
						log.MaestroErrorf("NetworkManager: error getting / setting DNS from lease / ifconfig: %s\n", err.Error())
					}

				} else {
					log.MaestroWarnf("NetworkManager: Stopping DHCP lease renewal, as interface '%s' is no longer managed!\n", ifname)
					break DhcpLoop
				}
			case dhcp_rebind_lease:
				debugging.DEBUG_OUT("DhcpLoop to - @dhcp_rebind_lease [%s]\n", ifname)
				// the same as get lease for now
				ifdata.dhcpWorkerControl <- networkThreadMessage{cmd: dhcp_get_lease}
				continue DhcpLoop

			case stop_and_release_IP:
				debugging.DEBUG_OUT("DhcpLoop to - @stop_and_release_IP [%s]\n", ifname)
				ifdata = this.getInterfaceData(ifname)
				if ifdata != nil {
					if ifdata.DhcpLease != nil {
						log.MaestroInfof("NetworkManager: DHCP: Releasing IP for interface %s\n", ifname)
						ReleaseFromServer(ifname, ifdata.DhcpLease)
						if ifdata.hadDNS {
							this.removeDnsBufferForInterface(ifname)
							this.finalizeDns()
							ifdata.hadDNS = false
						}
						break DhcpLoop
					}
				}
				log.MaestroWarnf("NetworkManager: Stopping DHCP lease (release requested), as interface '%s' is no longer managed!\n", ifname)
				break DhcpLoop
				// release the IP - to the server TODO
				// stop this thread

			case thread_shutdown:
				debugging.DEBUG_OUT("DhcpLoop to - @thread_shutdown [%s]\n", ifname)
				log.MaestroInfof("NetworkManager: DHCP worker for if %s got shutdown\n", ifname)
				break DhcpLoop
			}

		case <-time.After(timeout):
			if nextstate != nil {
				debugging.DEBUG_OUT("DHCP worker *timeout* for if %s - nextstate %+v\n", ifname, nextstate)
				log.MaestroWarnf("NetworkManager: DHCP worker *timeout* for if %s - nextstate %+v\n", ifname, nextstate)
				ifdata.dhcpWorkerControl <- *nextstate
				nextstate = nil
			}

		}

	}
	this.decIfThreadCount()

	ifdata = this.getInterfaceData(ifname)
	if ifdata != nil {
		ifdata.dhcpRunning = false
		ifdata.dhcpWorkerControl = nil
		// tell caller that this thread has stopped
		ifdata.dhcpWaitOnShutdown <- networkThreadMessage{cmd: shutdown_complete}
	}

}

func (mgr *networkManagerInstance) finalizePrimaryRoutes() {
	ok, ifpref, r := mgr.primaryTable.findPreferredRoute()
	log.MaestroDebugf("NetworkManager: finalizePrimaryRoutes: if %s - %+v (ok=%v)\n", ifpref, r, ok)
	if ok {
		log.MaestroDebugf("NetworkManager: preferred default route is on if %s - %+v\n", ifpref, r)
		err := mgr.primaryTable.setPreferredRoute(!mgr.networkConfig.DontOverrideDefaultRoute)
		if err == nil {
			log.MaestroDebugf("NetworkManager: set default route to %s %+v\n", ifpref, r)
		} else {
			log.MaestroErrorf("NetworkManager: error setting preferred route: %s\n", err.Error())
		}
	}
}

func (mgr *networkManagerInstance) watchInterface(ifname string) {
	mgr.watcherWorkChannel <- networkThreadMessage{
		cmd:    start_watch_interface,
		ifname: ifname,
	}
}

func (mgr *networkManagerInstance) stopWatchInterface(ifname string) {
	mgr.watcherWorkChannel <- networkThreadMessage{
		cmd:    stop_watch_interface,
		ifname: ifname,
	}
}

// a go routing to watch interfaces.
func (mgr *networkManagerInstance) watchInterfaces() {
	ch := make(chan netlink.LinkUpdate, 10)

Outer:
	for {
		select {
		case work := <-mgr.watcherWorkChannel:
			if work.cmd == thread_shutdown {
				// TODO: loop through and close all interface.stopInterfaceMonitor channels to
				// stop the subscriptions
				// close(done)
				break Outer
			}
			// add an interface to the watch list
			if work.cmd == start_watch_interface {
				pdata, ok := mgr.byInterfaceName.Load(work.ifname)
				if ok {
					if pdata == nil {
						nmLogErrorf("watchInterfaces() - got nil on lookup of <%s>\n", work.ifname)
					} else {
						ifdata := pdata.(*NetworkInterfaceData)
						if ifdata.interfaceChange == nil {
							ifdata.stopInterfaceMonitor = make(chan struct{})
							ifdata.interfaceChange = make(chan networkThreadMessage)
							err := netlink.LinkSubscribe(ch, ifdata.stopInterfaceMonitor)
							if err != nil {
								nmLogErrorf("watchInterfaces() - Error calling stopInterfaceMonitor(LinkSubscribe) %s - details: %s\n", work.ifname, err.Error())
							} else {
								nmLogSuccessf("watchInterfaces() - Succeeded subscribing interface monitors for %s\n", work.ifname)
							}
						} else {
							nmLogWarnf("watchInterfaces() - interface link %s appear to already be watched", work.ifname)
						}
					}
				} else {
					nmLogErrorf("watchInterfaces() - no interface in table named <%s>\n", work.ifname)
				}
			}
			if work.cmd == stop_watch_interface {
				pdata, ok := mgr.byInterfaceName.Load(work.ifname)
				if ok {
					if pdata == nil {
						nmLogErrorf("watchInterfaces() (stop_watch_interface) - got nil on lookup of <%s>\n", work.ifname)
					} else {
						ifdata := pdata.(*NetworkInterfaceData)
						if ifdata.stopInterfaceMonitor != nil {
							close(ifdata.stopInterfaceMonitor)
							ifdata.stopInterfaceMonitor = nil
							ifdata.interfaceChange = nil
						} else {
							nmLogWarnf("watchInterfaces() (stop_watch_interface) - interface link %s appears to not be watched\n", work.ifname)
						}
					}
				} else {
					nmLogErrorf("watchInterfaces() (stop_watch_interface) - no interface in table named <%s>\n", work.ifname)
				}
			}
		case update := <-ch:
			nmLogDebugf("watchInterfaces() - link state change for if <%s>: %+v\n", update.Attrs().Name, update)
			debugging.DEBUG_OUT("NetworkManager>>> saw link update: %+v\n", update)
			ifname := update.Attrs().Name
			ifdata := mgr.getInterfaceData(ifname)
			if ifdata != nil {
				if ifdata.lastFlags == 0 {
					ifdata.lastFlags = update.IfInfomsg.Flags
				}
				// NOTE: see man pages rtnetlink for details on this
				// structure. It's ultimately just defined in syscall as a
				// copy of the C Linux header files
				if (update.IfInfomsg.Flags & unix.IFF_LOWER_UP) != 0 {
					// was the interface previously down?
					up, err := mgr.primaryTable.isIfUp(ifname)
					if err != nil {
						nmLogErrorf("watchInterfaces() - internal route table does not have if %s - %s\n", ifname, err.Error())
					}
					if !up {
						// ok this is a change. The interface was down, is now up
						nmLogDebugf("interface %s now LOWER_UP\n", ifname)
						debugging.DEBUG_OUT("NetworkManager>>> Interface %s LOWER_UP now\n", ifname)
						// mark as up, and check for preferred default route
						err := netlink.LinkSetUp(update.Link)
						if err != nil {
							nmLogErrorf("watchInterfaces() - error brining up interface %s - details: %s\n", ifname, err.Error())
						}
						mgr.primaryTable.markIfAsUp(ifname)
						mgr.finalizePrimaryRoutes()
						dnsbuf := mgr.getDnsBufferForInterface(ifname)
						if dnsbuf != nil {
							dnsbuf.enable()
							mgr.finalizeDns()
						}
						// send out events
						if ifdata.interfaceChange != nil {
							select {
							case ifdata.interfaceChange <- networkThreadMessage{
								cmd:    state_LOWER_UP,
								ifname: ifname,
							}:
							default:
								nmLogWarnf("if %s state_LOWER_UP event dropped\n", ifname)
							}
						} else {
							nmLogErrorf("if %s - saw state change but interfaceChange chan is nil!\n", ifname)
						}
						addrstring := ""
						addr6string := ""
						if ifdata.CurrentIPv4Addr != nil {
							addrstring = ifdata.CurrentIPv4Addr.String()
						}
						if ifdata.CurrentIPv6Addr != nil {
							addr6string = ifdata.CurrentIPv6Addr.String()
						}
						submitNetEventData(&netevents.NetEventData{
							Type: netevents.InterfaceStateUp,
							Interface: &netevents.InterfaceEventData{
								ID:        ifname,
								Index:     int(update.IfInfomsg.Index),
								LinkState: "LOWER_UP",
								Address:   addrstring,
								AddressV6: addr6string,
							},
						})
					} else {
						nmLogDebugf("watchInterfaces() - ignoring link change for if %s. already marked as up.\n", ifname)
					}
				} else {
					// ok this is a change. The interface was up, is now down
					if (update.IfInfomsg.Flags & unix.IFF_LOWER_UP) == 0 {
						up, err := mgr.primaryTable.isIfUp(ifname)
						if err != nil {
							nmLogErrorf("watchInterfaces() - internal route table does not have if %s - %s\n", ifname, err.Error())
						}
						if up {
							nmLogWarnf("watchInterfaces() - interface %s now DOWN - (LOWER_UP false)\n", ifname)
							debugging.DEBUG_OUT("NetworkManager>>> Interface %s is now DOWN (LOWER_UP off)\n", ifname)
							err := netlink.LinkSetDown(update.Link)
							if err != nil {
								nmLogErrorf("watchInterfaces() - error bringing down interface %s - details: %s\n", ifname, err.Error())
							}
							// mark as down, and check for new preferred default route
							mgr.primaryTable.markIfAsDown(ifname)
							mgr.finalizePrimaryRoutes()
							dnsbuf := mgr.getDnsBufferForInterface(ifname)
							if dnsbuf != nil {
								dnsbuf.disable()
								mgr.finalizeDns()
							}
							// send events
							// apparently we don't get updates without this?
							err = netlink.LinkSetUp(update.Link)
							if err != nil {
								nmLogErrorf("watchInterfaces() - error bringing up interface (2) %s - details: %s\n", ifname, err.Error())
							}
							err = netlink.LinkSubscribe(ch, ifdata.stopInterfaceMonitor)
							if err != nil {
								nmLogErrorf("watchInterfaces() - Error calling LinkSubscribe %s - details: %s\n", ifname, err.Error())
							} else {
								nmLogSuccessf("watchInterfaces() - Calling LinkSubscribe succeeded: %s\n", ifname)
							}

							if ifdata.interfaceChange != nil {
								select {
								case ifdata.interfaceChange <- networkThreadMessage{
									cmd:    state_LOWER_DOWN,
									ifname: ifname,
								}:
								default:
									nmLogWarnf("watchInterfaces() - if %s state_LOWER_UP event dropped\n", ifname)
								}
							} else {
								nmLogErrorf("watchInterfaces() - if %s - saw state change but interfaceChange chan is nil!\n", ifname)
							}
							submitNetEventData(&netevents.NetEventData{
								Type: netevents.InterfaceStateDown,
								Interface: &netevents.InterfaceEventData{
									ID:        ifname,
									Index:     int(update.IfInfomsg.Index),
									LinkState: "LOWER_DOWN",
								},
							})
						} else {
							nmLogDebugf("watchInterfaces() - ignoring link change for if %s. already marked as down.\n", ifname)
						}
					}
				}
				// update the flags
				ifdata.lastFlags = update.IfInfomsg.Flags
			} else {
				nmLogDebugf("interface state change %s - ignoring. Not managed\n", ifname)
				debugging.DEBUG_OUT("NetworkManager>>> ignoring interface %s\n", ifname)
			}

		}

	}
}

// TODO - networkManagerInstance needs to be a TaskHandler... and Jobs should be started like any other Task
//
func (mgr *networkManagerInstance) SubmitTask(task *tasks.MaestroTask) (errout error) {
	var err error
	switch task.Op.GetType() {
	case maestroSpecs.OP_TYPE_NET_INTERFACE:
		requestedOp, ok := task.Op.(maestroSpecs.NetInterfaceOperation)
		if ok {
			ifconfig := requestedOp.GetIfConfig()
			if ifconfig != nil {
				switch task.Op.GetOp() {
				// NOTE: OP_ADD_ADDRESS and OP_REMOVE_ADDRESS change / manipulate the
				// address interfaces in some way or delta, where as OP_UPDATE_ADDRESS
				// essentially takes a given state, and sets up the interface exactly
				// as such
				// OP_ADD_ADDRESS and OP_REMOVE_ADDRESS need to update the interface's RunningIfconfig
				// on completion
				case maestroSpecs.OP_ADD_ADDRESS:
					log.MaestroWarnf("NetworkManager: OP_ADD_ADDRESS not implemented yet\n")
				case maestroSpecs.OP_REMOVE_ADDRESS:
					log.MaestroWarnf("NetworkManager: network manager>> OP_REMOVE_ADDRESS not implemented yet\n")
					// remove address
				case maestroSpecs.OP_UPDATE_ADDRESS:
					// TODO !! - need to validate config

					var ifdata *NetworkInterfaceData
					// first, save this record. Even if the interface does not exist, if it does
					// come up then we will set it up.
					ifdata = mgr.getOrNewInterfaceData(ifconfig.IfName)
					debugging.DEBUG_OUT("past getOrNewInterfaceData(%s) - %+v\n", ifconfig.IfName, ifdata)
					ifdata.StoredIfconfig = ifconfig
					err = mgr.commitInterfaceData(ifconfig.IfName)
					if err != nil {
						log.MaestroErrorf("NetworkManager: Problem storing interface [%s] data in DB: %s. Skipping config.\n", ifconfig.IfName, err.Error())
						errout = err
						return
					}
					// second, determine if that interface exists, and get it's index and name (one is required to be known)
					//					ifname, ifindex, err := GetInterfaceIndexAndName(ifconfig.IfName, ifconfig.IfIndex)
					var ifname string
					var ifindex int
					link, err := GetInterfaceLink(ifconfig.IfName, ifconfig.IfIndex)
					if err == nil && link != nil {
						ifname = link.Attrs().Name
						ifindex = link.Attrs().Index
						// first see if a hardware address should be set
						if len(ifconfig.HwAddr) > 0 {
							currentHwAddr := link.Attrs().HardwareAddr
							if len(currentHwAddr) < 1 || (currentHwAddr.String() != ifconfig.HwAddr) {
								log.MaestroDebugf("NetworkManager: looks like mac address for if %s is new or different than set. changing.\n", ifname)
								newHwAddr, err2 := net.ParseMAC(ifconfig.HwAddr)
								if err2 == nil {
									// ok - need to bring interface down to set Mac
									log.MaestroInfof("NetworkManager: brining if %s down\n", ifname)
									err2 = netlink.LinkSetDown(link)
									if err2 != nil {
										log.MaestroErrorf("NetworkManager: failed to bring if %s down - %s\n", ifname, err2.Error())
									}
									log.MaestroDebugf("NetworkManager: setting if %s MAC address to %s\n", ifname, ifconfig.HwAddr)
									err2 = netlink.LinkSetHardwareAddr(link, newHwAddr)
									if err2 != nil {
										log.MaestroErrorf("NetworkManager: failed to set MAC address on if %s - %s\n", ifname, err2.Error())
									}
									err2 = netlink.LinkSetUp(link)
									if err2 != nil {
										log.MaestroErrorf("NetworkManager: failed to bring if %s up - %s\n", ifname, err2.Error())
									} else {
										log.MaestroInfof("NetworkManager:  if %s is up\n", ifname)
									}
								} else {
									log.MaestroErrorf("NetworkManager: Failed to parse MAC address \"%s\" for interface %s (%d). Skipping other setup.\n", ifconfig.HwAddr, ifname, ifindex)
									errout = err2
									return
								}

							}
						}

						// takes the state of the NetIfConfigPayload
						// and sets up the interface as such
						if ifconfig.DhcpV4Enabled {

							// do dchp, based on current database status

							// is their a .RunningConfig?

							// var ifdata *NetworkInterfaceData
							// // first, retrieve the interface
							// mgr.networkConfigDB.Get(ifname,&ifdata)
							// // we do DHCP with a new go routine, as it may take some time
							// if ifdata != nil {
							debugging.DEBUG_OUT("ok, running goDhcp for if %s\n", ifname)
							mgr.watchInterface(ifconfig.IfName)
							if !ifdata.dhcpRunning {
								log.MaestroInfof("Starting DhcpLoop for %s", ifname)
								go mgr.doDhcp(ifname, requestedOp)
							} else {
								log.MaestroWarnf("goDhcp for if %s\n already running, skipping new instance", ifname)
							}
						} else {
							// assign static IP

							// we do DHCP with a new go routine, as it may take some time
							if ifdata != nil && ifdata.dhcpRunning {
								debugging.DEBUG_OUT("waiting on DHCP thread shutdown for if %s\n", ifname)
								ifdata.dhcpWorkerControl <- networkThreadMessage{cmd: stop_and_release_IP}
								// wait on that shutdown
								<-ifdata.dhcpWaitOnShutdown
							}

							debugging.DEBUG_OUT("Setting up static IP for if %s\n", ifname)

							confs := []*maestroSpecs.NetIfConfigPayload{}
							confs = append(confs, ifconfig)
							// results, err :=
							results, err := SetupStaticInterfaces(confs)
							if err != nil {
								errout = err
								log.MaestroErrorf("NetworkManager: Failed to setup static address on interface %s - %s\n", ifname, err.Error())
								debugging.DEBUG_OUT("NetworkManager: Failed to setup static address on interface %s - %s\n", ifname, err.Error())
							} else {
								log.MaestroSuccessf("Network Manager: Static address set on %s of %s\n", ifname, confs[0].IPv4Addr)
								//Start watching the interface
								mgr.watchInterface(ifconfig.IfName)
								//Set the link up if its down
								err := netlink.LinkSetUp(link)
								if err != nil {
									log.MaestroErrorf("NetworkManager: failed to bring if %s up while doing static config - %s\n", ifname, err.Error())
								} else {
									log.MaestroInfof("NetworkManager:  Link set up for if %s\n", ifname)
								}
								log.MaestroInfof("NetworkManager:  Adding primary routes %v\n", confs)
								_, err = addDefaultRoutesToPrimaryTable(mgr, confs)
								if err != nil {
									log.MaestroErrorf("NetworkManager: Failed to add default route for %s - %s\n", ifname, err.Error())
								} else {
									log.MaestroErrorf("NetworkManager: Finalize default route for %s\n", ifname)
									//Finalize the primary routes
									mgr.finalizePrimaryRoutes()
								}
								if ifdata != nil {
									ifdata.CurrentIPv4Addr = results[0].ipv4
								}
								if len(results) > 0 && results[0].ipv4 != nil {
									arperr := arp.SendGratuitous(&arp.Gratuitous{
										IfaceName: ifconfig.IfName,
										IP:        results[0].ipv4,
									})
									if arperr != nil {
										nmLogErrorf("Erroring sending gratuitous ARP: %s\n", arperr.Error())
									} else {
										nmLogDebugf("Gratuitous ARP sent for IP %s\n", results[0].IPV4)
									}
								} else {
									nmLogErrorf("SetupStaticInterfaces() has invalid results array or no address for if %s\n", ifname)
								}
							}
						}
					} else {
						// Interface could not be found!
						debugging.DEBUG_OUT("NetworkManager: Inteface could not be found %s (%d) - %+v\n", ifconfig.IfName, ifconfig.IfIndex, err.Error())
						err2 := new(maestroSpecs.APIError)
						err2.HttpStatusCode = 500
						err2.ErrorString = "Interface could not be found"
						err2.Detail = "in SubmitTask(): " + err.Error()
						log.MaestroErrorf("NetworkManager: Inteface could not be found %s (%d) - %+v\n", ifconfig.IfName, ifconfig.IfIndex, err.Error())
						errout = err2
					}
				case maestroSpecs.OP_RENEW_DHCP:
					var ifdata *NetworkInterfaceData
					// first, save this record. Even if the interface does not exist, if it does
					// come up then we will set it up.
					ifdata = mgr.getOrNewInterfaceData(ifconfig.IfName)
					if ifdata.dhcpRunning {

					} else {
						// dhcp is not running. So, that indicates that it is not configured either.
						// throw an error
						err2 := new(maestroSpecs.APIError)
						err2.HttpStatusCode = http.StatusBadRequest
						err2.ErrorString = "Asked for DHCP renew, but DHCP not running on interface"
						log.MaestroErrorf("NetworkManager: Asked for DHCP renew, but DHCP not running on interface - %s\n", ifconfig.IfName)
						errout = err2
					}

					// debugging.DEBUG_OUT("past getOrNewInterfaceData(%s) - %+v\n", ifconfig.IfName, ifdata)
					// ifdata.StoredIfconfig = ifconfig
					// err = mgr.commitInterfaceData(ifconfig.IfName)
					// if err != nil {
					// 	log.MaestroErrorf("network manager: Problem storing interface [%s] data in DB: %s. Skipping config.\n", ifconfig.IfName, err.Error())
					// 	errout = err
					// 	return
					// }
					// // second, determine if that interface exists, and get it's index and name (one is required to be known)
					// ifname, ifindex, err := GetInterfaceIndexAndName(ifconfig.IfName, ifconfig.IfIndex)
					// if err == nil && ifindex > 0 {
					// 	// takes the state of the NetIfConfigPayload
					// 	// and sets up the interface as such
					// 	if ifconfig.DhcpV4Enabled {

					// 	}
					// }

				case maestroSpecs.OP_RELEASE_DHCP:

				default:
					debugging.DEBUG_OUT("NETWORK>>>networkManagerInstance.SubmitTask() unknown Op\n")
					err := new(maestroSpecs.APIError)
					err.HttpStatusCode = 500
					err.ErrorString = "Uknown op"
					err.Detail = "in SubmitTask()"
					log.MaestroErrorf("NetworkManager: got missplaced task %+v\n", task)
					errout = err
				}
			} else {
				debugging.DEBUG_OUT("NETWORK>>>networkManagerInstance.SubmitTask() unknown Op\n")
				err := new(maestroSpecs.APIError)
				err.HttpStatusCode = 500
				err.ErrorString = "No config"
				err.Detail = "in SubmitTask()"
				log.MaestroErrorf("NetworkManager: got null config\n")
				errout = err
			}
		} else {
			debugging.DEBUG_OUT("NETWORK>>>networkManagerInstance.SubmitTask() not correct Op type\n")
			err := new(maestroSpecs.APIError)
			err.HttpStatusCode = 500
			err.ErrorString = "op was not an network operation"
			err.Detail = "in SubmitTask() - switch{}"
			log.MaestroErrorf("NetworkManager: got missplaced task %+v\n", task)
			errout = err

		}

	case maestroSpecs.OP_TYPE_NET_CONFIG:
		requestedOp, ok := task.Op.(maestroSpecs.NetConfigOperation)
		if ok {
			ifconfig := requestedOp.GetNetConfig()
			if ifconfig != nil {
				switch task.Op.GetOp() {

				}
			}
		}

	default:
		debugging.DEBUG_OUT("NETWORK>>>networkManagerInstance.SubmitTask() not correct Op type\n")
		err := new(maestroSpecs.APIError)
		err.HttpStatusCode = 500
		err.ErrorString = "op was not an network operation"
		err.Detail = "in SubmitTask()"
		log.MaestroErrorf("NetworkManager: got missplaced task %+v\n", task)
		errout = err
	}

	return nil
}

// ValidateTask maks sure the network task is properly formatted. TODO
func (mgr *networkManagerInstance) ValidateTask(task *tasks.MaestroTask) error {
	return nil
}

// InitNetworkManager be called on startup.
// NetworkConfigPayload will come from config file
// Storage should be started already.
func InitNetworkManager(networkconfig *maestroSpecs.NetworkConfigPayload, ddbconfig *maestroConfig.DeviceDBConnConfig) error {
	log.MaestroInfof("NetworkManager: Initializing %v %v\n", networkconfig, ddbconfig)
	inst := GetInstance()
	inst.networkConfig = networkconfig
	inst.ddbConnConfig = ddbconfig

	//Setup the config with the given network config
	if inst.networkConfig != nil {
		if inst.networkConfig.Disable == false {
			log.MaestroInfof("NetworkManager: Submit config read from config file\n")
			inst.submitConfig(inst.networkConfig)
		} else {
			log.MaestroWarnf("NetworkManager: Network configuration Disable flag set to true\n")
			return nil
		}
	} else {
		return errors.New("NetworkManager: No network configuration set, unable to cofigure network")
	}

	return nil
}
