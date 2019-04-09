package mdns

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
	"encoding/gob"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/WigWagCo/maestro/log"
	"github.com/WigWagCo/maestro/storage"
	"github.com/WigWagCo/stow"
	"github.com/WigWagCo/zeroconf"
	"github.com/boltdb/bolt"
)

// This package implements a small mdns server which is programatically controllable
// over the maestro API

const (
	cDBNAME   = "mdnsConfig"
	logPrefix = "mdns service: "
)

// ConfigEntry is the config entry for a single published MDNS record
type ConfigEntry struct {
	Name    string   `yaml:"name" json:"name"` // AKA "instance name"
	Service string   `yaml:"service" json:"service"`
	Domain  string   `yaml:"domain" json:"domain"`
	Port    int      `yaml:"port" json:"port"`
	TTL     uint32   `yaml:"ttl" json:"ttl"`
	Text    []string `yaml:"text" json:"text"`
	// Hostname is a string representing the host to lookup
	// This is its DNS name. If left blank, then this subsystem will
	// use os.Hostname()
	Hostname string `yaml:"hostname" json:"hostname"`
	// Ips is a comma separate string of one or more IP address
	// if blank, then then the hostname will be looked up for interface
	// being publish on.
	Ips string `yaml:"ips" json:"ips"`
	// Interfaces should be a comma separate string stating all interfaces
	// to publish the record on. If left empty it will publish on all interfaces
	Interfaces string `yaml:"interfaces" json:"interfaces"`
	// NotInterfaces black lists certain interfaces from being published on, even if
	// Interfaces names them, or is set to empty
	NotInterfaces string `yaml:"not_interfaces" json:"not_interfaces"`
	// NotPersistent if true means the record will not be stored in the maestro
	// config database
	NotPersistent bool `yaml:"not_persistent" json:"not_persistent"`
	// actual server when published
	servers []*zeroconf.Server
}

func (conf *ConfigEntry) dupNoServer(orig *ConfigEntry) (ret *ConfigEntry) {
	ret = &ConfigEntry{
		Name:          orig.Name,
		Service:       orig.Service,
		Domain:        orig.Domain,
		Port:          orig.Port,
		Text:          orig.Text,
		Hostname:      orig.Hostname,
		Ips:           orig.Ips,
		Interfaces:    orig.Interfaces,
		NotInterfaces: orig.NotInterfaces,
		NotPersistent: orig.NotPersistent,
	}
	return
}

// Manager is a singleton which manage all MDNS publication for maestro
type Manager struct {
	db           *bolt.DB
	mdnsConfigDB *stow.Store
	// published interfaces
	// hash : *ConfigEntry
	published sync.Map
}

// StorageInit implements the storage.StorageUser interface
// implements the storage.StorageUser interface
func (mgr *Manager) StorageInit(instance storage.MaestroDBStorageInterface) {
	gob.Register(&ConfigEntry{})
	mgr.mdnsConfigDB = stow.NewStore(instance.GetDb(), []byte(cDBNAME))
}

// // called when the DB is open and ready
// StorageReady(instance MaestroDBStorageInterface)
// // called when the DB is initialized
// StorageInit(instance MaestroDBStorageInterface)
// // called when storage is closed
// StorageClosed(instance MaestroDBStorageInterface)

// StorageReady is called when the DB is initialized
// implements the storage.StorageUser interface
func (mgr *Manager) StorageReady(instance storage.MaestroDBStorageInterface) {
	mgr.db = instance.GetDb()
	err := mgr.loadAllData()
	if err != nil {
		log.MaestroErrorf("mdns manager: Failed to load existing network interface settings: %s", err.Error())
	}
}

// StorageClosed is called when storage is closed
// implements the storage.StorageUser interface
func (mgr *Manager) StorageClosed(instance storage.MaestroDBStorageInterface) {

}

// returns a hash string representing the published record.
// if hash strings match, then the record is considered the same
func hashConfig(entry *ConfigEntry) (ret string) {
	if entry != nil {
		ret = fmt.Sprintf("%s.%s:%d:%s=%s:%s", entry.Hostname, entry.Domain, entry.Port, entry.Name, entry.Ips, entry.Interfaces)
		// ret = entry.Hostname + "." + entry.Domain + ":" + strconv.FormatInt(int64(entry.Port), 10)
		// ret += ":" + entry.Name + "=" + entry.Ips + ":" + entry.Interfaces
	}
	return
}

func makeServiceEntry(entry *ConfigEntry) (ret *zeroconf.ServiceEntry) {
	if len(entry.Domain) < 1 {
		entry.Domain = "local"
	}
	ret = zeroconf.NewServiceEntry(entry.Name, entry.Service, entry.Domain)
	ret.HostName = entry.Hostname
	ret.Port = entry.Port
	ret.Text = entry.Text
	ret.TTL = entry.TTL
	return
}

func splitCommasAndTrim(s string) (ret []string) {
	if len(s) > 0 {
		ret = strings.Split(s, ",")
	outerSplit:
		for {
			for n, v := range ret {
				ret[n] = strings.TrimSpace(v)
				if len(ret[n]) < 1 {
					ret = append(ret[:n], ret[n+1:]...)
					continue outerSplit
				}
			}
			break outerSplit
		}
	}
	return
}

// ShutdownAllPublished TODO - shutsdown all mdns servers
func (mgr *Manager) ShutdownAllPublished() (err error) {
	return
}

// RefreshInterfaces when called, will walk though all published records,
// and look and see if the interfaces IP addresses changed. If they did
// it will use new IP, and republish the record. This only matters for records
// which don't have a stated IP. Such case is the usual case.
func (mgr *Manager) RefreshInterfaces() (err error) {
	// TODO
	return
}

// PublishEntry publishes one or multiple mDNS records to the network
// on the interfaces specifed (or all interfaces) - if NotPersistent is not set
// publications will be published again on restart of maestro when they are
// re-read from the database
func (mgr *Manager) PublishEntry(entry *ConfigEntry) (err error) {
	if entry != nil {
		hash := hashConfig(entry)
		// validate entry first...

		actual, loaded := mgr.published.LoadOrStore(hash, entry)
		if loaded {
			existing, ok := actual.(*ConfigEntry)
			if ok {
				// ok - there is an existing entry...
				log.MaestroInfof(logPrefix+"existing mdns record published. stopping and removing. (%s)", hash)
				if len(existing.servers) > 0 {
					for _, server := range existing.servers {
						server.Shutdown()
					}
				}
				existing.servers = nil
				mgr.published.Store(hash, entry)
			} else {
				log.MaestroErrorf(logPrefix + "Possible internal map corruption.")
			}
		}
		// publish the said new entry
		svcentry := makeServiceEntry(entry)
		ips := splitCommasAndTrim(entry.Ips)
		// if len(ips) < 1 {
		// ok - this means we should figure out the IP. We will need to do a
		// separate publish for each interface, as each interface will
		// have a different IP address.
		// } else {
		ifs := splitCommasAndTrim(entry.Interfaces)
		notIfs := splitCommasAndTrim(entry.NotInterfaces)
		var server *zeroconf.Server
		if len(ips) < 1 {
			var servers []*zeroconf.Server
			servers, err = zeroconf.RegisterServiceEntryEachInterfaceIP(svcentry, ifs, notIfs)
			if err == nil {
				entry.servers = servers
			}
		} else {
			server, err = zeroconf.RegisterServiceEntry(svcentry, ips, ifs, notIfs)
			if err == nil {
				entry.servers = append(entry.servers, server)
			}
		}
	} else {
		err = errors.New("null entry")
	}
	return
}

func (mgr *Manager) loadAllData() (err error) {
	var temp ConfigEntry
	mgr.mdnsConfigDB.IterateIf(func(key []byte, val interface{}) bool {
		DEBUG(hash := string(key[:]))
		entry, ok := val.(*ConfigEntry)
		if ok {
			mgr.PublishEntry(entry)
			// err2 := resetIfDataFromStoredConfig(ifdata)
			// if err2 == nil {
			// 	this.newInterfaceMutex.Lock()
			// 	// if there is an existing in-memory entry, overwrite it
			// 	this.byInterfaceName.Set(ifname, unsafe.Pointer(ifdata))
			// 	this.newInterfaceMutex.Unlock()
			// 	DEBUG_OUT("loadAllInterfaceData() see if: %s --> %+v\n", ifname, ifdata)
			// } else {
			// 	log.MaestroErrorf("networkManager: Critical problem with interface [%s] config. Not loading config.", ifname)
			// }
		} else {
			err = errors.New("Internal DB corruption")
			DEBUG_OUT(logPrefix+"internal DB corruption - @ %s\n", hash)
			log.MaestroErrorf(logPrefix + "internal DB corruption")
		}
		return true
	}, &temp)
	return
}

// LoadFromConfigFile loads up ConfigEntry which were published in the config file.
// The big difference here is that we mark these as NotPersistent
func (mgr *Manager) LoadFromConfigFile(entries []*ConfigEntry) (allok bool, errs []error) {
	allok = true
	for _, entry := range entries {
		entry.NotPersistent = true
		DEBUG_OUT("mdns: record: %+v\n", entry)
		err := mgr.PublishEntry(entry)
		if err != nil {
			log.MaestroErrorf(logPrefix+"Failed to publish mdns record: %s %s %s - %s", entry.Name, entry.Service, entry.Text, err.Error())
			allok = false
			errs = append(errs, err)
		} else {
			errs = append(errs, nil)
		}
	}
	return
}

var instance *Manager

func newManager() *Manager {
	return new(Manager)
}

// GetInstance returns and creates if needed the singleton instance of the MDNS manager
func GetInstance() *Manager {
	logger := log.NewPrefixedLogger("mdns-zeroconf")
	zeroconf.OverrideLogging(logger.Error, logger.Info, logger.Debug)
	// FIXME: this is a very odd situation:
	// for some reasons, the prefix logger does not recognize that maestro's main.go called SetGoLoggerReady()
	// which sets an internal var in the "log" package. My guess is there may be separate instance of the Go
	// portion of the "log" package - some behavior of the Go run time (but "greasego" or libgrease.so). In
	// any case, this hack up seems to work. And the grease logger should always be up and running at this point typically
	log.SetGoLoggerReady()
	if instance == nil {
		instance = newManager()
		storage.RegisterStorageUser(instance)
		// internalTicker = time.NewTicker(time.Second * time.Duration(defaults.TASK_MANAGER_CLEAR_INTERVAL))
		// controlChan = make(chan *controlToken, 100) // buffer up to 100 events
	}
	return instance
}
