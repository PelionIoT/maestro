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

import (
	"fmt"
	"log"
	"testing"

	//    "github.com/armPelionEdge/hashmap"
	"net"
	"os"
	"time"

	"github.com/armPelionEdge/dhcp4"
	"github.com/armPelionEdge/dhcp4client"
	"github.com/armPelionEdge/maestro/events"
	"github.com/armPelionEdge/maestro/storage"
	"github.com/armPelionEdge/maestro/tasks"
	"github.com/armPelionEdge/maestroSpecs"
	"github.com/armPelionEdge/netlink"
	"github.com/boltdb/bolt"
	"golang.org/x/sys/unix"
	// "time"
)

func TestMain(m *testing.M) {
	m.Run()
}

func TestSetupInterfaces(t *testing.T) {
	fmt.Printf("Test SetupInterfaces\n")

	// let's do something with eth2 which is a USB plugin

	conf1 := new(maestroSpecs.NetIfConfigPayload)
	conf1.IfName = "eth2"
	conf1.IPv4Addr = "192.168.78.31"
	conf1.IPv4Mask = 24
	conf1.IPv4BCast = "192.168.78.255"
	conf1.ClearAddresses = true

	confs := []*maestroSpecs.NetIfConfigPayload{}
	confs = append(confs, conf1)
	results, err := SetupStaticInterfaces(confs)

	if len(results) < 1 {
		log.Fatal("No results!")
	}

	fmt.Printf("SetupInterfaces results:\n")
	for n, result := range results {
		fmt.Printf("SetupInterfaces() results[%d] = %+v\n", n, result)
	}

	if err != nil {
		log.Fatal("Error:" + err.Error())
	}
}

func TestInterfaceDownEvent(t *testing.T) {

	ch := make(chan netlink.LinkUpdate)
	done := make(chan struct{})
	//    defer close(done)

	netlink.LinkSubscribe(ch, done)

	fmt.Printf("Waiting... This test requires you to unplug ethernet cable or some net interface.\n")

Outer:
	for {
		select {
		case update := <-ch:
			fmt.Printf("Got link update: %+v\n", update)

			//            fmt.Printf("Waiting and then reading if status\n")
			//            time.Sleep(time.Millisecond * 5000)

			// attrs, err := GetLinkStatusByName(update.Attrs().Name)

			// if err == nil {
			state := "down"
			//update.IfInfomsg.Flags&unix.IFF_UP != 0
			if (update.IfInfomsg.Flags & unix.IFF_LOWER_UP) != 0 {
				state = "up"
			}
			// if (update.Attrs().Flags & net.FlagUp) > 0 {
			//     state = "up"
			// }

			fmt.Printf("Saw change on: %s - is now %s\n", update.Attrs().Name, state)

			if state == "down" {
				close(done)
				break Outer
			}

		}

	}

}

func showDhcpRequestProgress(state int, addinfo string) (keepgoing bool) {
	fmt.Printf("showDhcpRequestProgress: state: %v addinfo: %v\n", state, addinfo)
	keepgoing = true
	return
}

func TestDhcpRequest(t *testing.T) {

	var leaseinfo DhcpLeaseInfo
	requestopts := new(dhcp4client.DhcpRequestOptions)
	requestopts.ProgressCB = showDhcpRequestProgress
	requestopts.StepTimeout = 1 * time.Minute
	requestopts.AddRequestParam(dhcp4.OptionRouter)
	requestopts.AddRequestParam(dhcp4.OptionSubnetMask)
	requestopts.AddRequestParam(dhcp4.OptionDomainNameServer)
	requestopts.AddRequestParam(dhcp4.OptionHostName)
	requestopts.AddRequestParam(dhcp4.OptionDomainName)
	requestopts.AddRequestParam(dhcp4.OptionBroadcastAddress)
	requestopts.AddRequestParam(dhcp4.OptionNetworkTimeProtocolServers)

	success, newleaseinfo, err := RequestOrRenewDhcpLease("enp0s3", &leaseinfo, requestopts)

	if success != dhcp4client.Success {
		log.Fatalf("Failed to renew DHCP lease: %+v\n", success)
	}

	if newleaseinfo != &leaseinfo {
		log.Fatalf("Passed in a valid DhcpLeaseInfo pointer pointing to a struct but failed. %+v", err)
	}

	if err != nil {
		log.Fatalf("Error on RequestOrRenewDhcpLease: %s\n", err.Error())
	} else {
		fmt.Printf("Lease: %+v\n", leaseinfo)
		fmt.Printf("   Address setting mask:%s\n", leaseinfo.CurrentMask.String())
		ok, ip := leaseinfo.GetRouter()
		if ok {
			fmt.Printf("   Router: %s\n", ip.String())
		} else {
			fmt.Printf("   Router: NOT SET\n")
		}
		ok, ips := leaseinfo.GetDNS()
		if ok {
			for _, dns := range ips {
				fmt.Printf("   DNS: %s\n", dns.String())
			}
		} else {
			fmt.Printf("   DNS: NOT SET\n")
		}
	}
}

func TestGetAndSetInterfaceMacAddress(t *testing.T) {
	mac, index, err := GetInterfaceMacAddress("eth2")
	if err != nil {
		log.Fatalf("Failed to GetInterfaceMacAddress - %s", err.Error())
	} else {
		if index < 1 {
			log.Fatalf("index is not sane.")
			return
		}
		macstring := mac.String()
		err = SetInterfaceMacAddress("eth2", 0, macstring)
		if err != nil {
			log.Fatalf("Failed to SetInterfaceMacAddress - %s", err.Error())
		} else {
			fmt.Printf("get and set interface to %s\n", macstring)
		}
	}
}

func TestGetInterfaceIndexAndName(t *testing.T) {
	name, index, err := GetInterfaceIndexAndName("saldjaskdj", 0)

	if err == nil {
		log.Fatal("Unknown interface. Should be an error")
	} else {
		fmt.Printf("Expected failure: %+v\n", err.Error())
	}

	name, index, err = GetInterfaceIndexAndName("lo", 0)

	if err != nil {
		log.Fatal("Failed. Should pass (unless this box has no if named 'lo')")
	} else {
		if index < 1 {
			log.Fatal("Invalid value for index (?)")
		}
		if len(name) < 1 {
			log.Fatal("Index 'name' was left blank (?)")
		} else {
			fmt.Printf("Found interface \"%s\" index:%d ", name, index)
		}
	}
}

func TestIsIPv4AddressSet(t *testing.T) {
	_, ipv4Net, err := net.ParseCIDR("127.0.0.1/8")
	if err != nil {
		log.Fatal("Error on test setup " + err.Error())
	}
	addrtest := &netlink.Addr{
		IPNet: ipv4Net, // IPNet.IP is now 127.0.0.0
		// IPNet.Mask is now /8
	}
	addrtest.IPNet.IP = net.ParseIP("127.0.0.1")
	yes, err := IsIPv4AddressSet("lo", addrtest)
	if err != nil {
		log.Fatal("Error on test IsIPv4AddressSet " + err.Error())
	}
	if !yes {
		log.Fatal("Loopback address not set? Looks like a failure.")
	}
}

type testStorageManager struct {
	db     *bolt.DB
	dbname string
}

const (
	TEST_DB_NAME = "testStorageManager.db"
)

func (this *testStorageManager) start() {
	this.dbname = TEST_DB_NAME
	db, err := bolt.Open(TEST_DB_NAME, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	this.db = db
}

func (this *testStorageManager) GetDb() *bolt.DB {
	return this.db
}

func (this *testStorageManager) GetDbName() string {
	return this.dbname
}

func (this *testStorageManager) RegisterStorageUser(user storage.StorageUser) {
	if this.db != nil {
		user.StorageInit(this)
		user.StorageReady(this)
	} else {
		log.Fatal("test DB failed.")
	}
}

// like GetInstance but for testing
func testGetInstance(storage *testStorageManager) *networkManagerInstance {
	if instance == nil {
		instance = newNetworkManagerInstance()
		storage.RegisterStorageUser(instance)
		// internalTicker = time.NewTicker(time.Second * time.Duration(defaults.TASK_MANAGER_CLEAR_INTERVAL))
		// controlChan = make(chan *controlToken, 100) // buffer up to 100 events
	}
	return instance
}

// like InitNetworkManager but for testing
func testInitNetworkManager(config *maestroSpecs.NetworkConfigPayload, storage *testStorageManager) (err error) {
	inst := testGetInstance(storage)
	inst.submitConfig(config)
	// make the global events channel. 'false' - not a persisten event (won't use storage)
	_, _, err = events.MakeEventChannel("events", false, false)
	if err != nil {
		log.Fatalf("NetworkManager: CRITICAL - error creating global channel \"events\" %s", err.Error())
	}
	return
}

func (this *testStorageManager) shutdown(user storage.StorageUser) {
	user.StorageClosed(this)
}

// like networking/manager.go:getInstance()
func getNetworkManagerInstance(storage *testStorageManager) *networkManagerInstance {
	if instance == nil {
		instance = newNetworkManagerInstance()
		storage.RegisterStorageUser(instance)
		// internalTicker = time.NewTicker(time.Second * time.Duration(defaults.TASK_MANAGER_CLEAR_INTERVAL))
		// controlChan = make(chan *controlToken, 100) // buffer up to 100 events
	}
	return instance
}

func TestClearAllAddressesByLinkName(t *testing.T) {
	err := ClearAllAddressesByLinkName("eth2", nil)

	if err != nil {
		log.Fatal("ClearAllAddressesByLinkName failed: " + err.Error())
	}

}

func TestNetworkManagerDhcpLoop(t *testing.T) {
	// fresh start - no database
	os.Remove(TEST_DB_NAME)

	storage := new(testStorageManager)
	storage.start()
	manager := getNetworkManagerInstance(storage)

	manager.enableThreadCount() // enable block on thread count

	conf1 := new(maestroSpecs.NetIfConfigPayload)
	conf1.IfName = "wlan2"
	// conf1.IPv4Addr = "192.168.78.31"
	// conf1.IPv4Mask = 24
	// conf1.IPv4BCast = "192.168.78.255"
	conf1.DhcpV4Enabled = true
	conf1.ClearAddresses = true

	op := new(maestroSpecs.NetInterfaceOpPayload)
	op.Type = maestroSpecs.OP_TYPE_NET_INTERFACE
	op.Op = maestroSpecs.OP_UPDATE_ADDRESS
	op.IfConfig = conf1
	op.TaskId = "TestId"

	task := new(tasks.MaestroTask)

	task.Id = "Example"
	task.Src = "Test"
	task.Op = op

	fmt.Printf("Submitting task %+v\n", task)
	err := manager.SubmitTask(task)

	if err != nil {
		log.Fatalf("Error on Network manager SubmitTask() %s %+v", err.Error(), err)
	}

	timeout := 60 * 60 * 12
	fmt.Printf("Waiting on threads (%d seconds)...\n", timeout)
	manager.waitForActiveInterfaceThreads(timeout)

	storage.shutdown(manager)

}

func TestNetworkManagerNewDbDhcp(t *testing.T) {
	// fresh start - no database
	os.Remove(TEST_DB_NAME)

	storage := new(testStorageManager)
	storage.start()
	manager := getNetworkManagerInstance(storage)

	manager.enableThreadCount() // enable block on thread count

	conf1 := new(maestroSpecs.NetIfConfigPayload)
	conf1.IfName = "wlx00b08c05ed58" // like 'eth2'
	// conf1.IPv4Addr = "192.168.78.31"
	// conf1.IPv4Mask = 24
	// conf1.IPv4BCast = "192.168.78.255"
	conf1.DhcpV4Enabled = true
	conf1.ClearAddresses = true

	op := new(maestroSpecs.NetInterfaceOpPayload)
	op.Type = maestroSpecs.OP_TYPE_NET_INTERFACE
	op.Op = maestroSpecs.OP_UPDATE_ADDRESS
	op.IfConfig = conf1
	op.TaskId = "TestId"

	task := new(tasks.MaestroTask)

	task.Id = "Example"
	task.Src = "Test"
	task.Op = op

	fmt.Printf("Submitting task %+v\n", task)
	err := manager.SubmitTask(task)

	if err != nil {
		log.Fatalf("Error on Network manager SubmitTask() %s %+v", err.Error(), err)
	}

	timeout := 60 * 60 * 12
	fmt.Printf("Waiting on threads (%d seconds)...\n", timeout)
	manager.waitForActiveInterfaceThreads(timeout)

	fmt.Printf("Now we will try SetupExistingInterface(nil) ...\n")

	manager.SetupExistingInterfaces(nil)

	storage.shutdown(manager)

}

func TestRestartNetworkManagerDhcp(t *testing.T) {

	storage := new(testStorageManager)
	storage.start()
	manager := getNetworkManagerInstance(storage)

	manager.enableThreadCount() // enable block on thread count

	fmt.Printf("Now we will try SetupExistingInterface(nil) ...\n")

	manager.SetupExistingInterfaces(nil)

	timeout := 60 * 60 * 12
	fmt.Printf("Waiting on threads (%d seconds)...\n", timeout)
	manager.waitForActiveInterfaceThreads(timeout)

	storage.shutdown(manager)
}

// This test assumes an existing entry for the interface, and then replaces it
// with the new config, with Existing as "override"
func TestRestartWithConfigOverride(t *testing.T) {
	// assumes TestNetworkManagerSetupExisting was ran previously
	storage := new(testStorageManager)
	storage.start()
	// manager := getNetworkManagerInstance(storage)

	config := &maestroSpecs.NetworkConfigPayload{
		DnsIgnoreDhcp: false,
	}

	ifconfig := &maestroSpecs.NetIfConfigPayload{
		IfName:         "eth2",
		ClearAddresses: true,
		DhcpV4Enabled:  false,
		IPv4Addr:       "192.168.78.31",
		IPv4Mask:       24,
		IPv4BCast:      "192.168.78.255",
		Existing:       "override",
	}

	config.Interfaces = append(config.Interfaces, ifconfig)

	err := testInitNetworkManager(config, storage)
	if err != nil {
		log.Fatalf("Failed to setup test instance of network manager: %+v\n", err)
	}
	manager := testGetInstance(storage)

	// conf1 := new(maestroSpecs.NetIfConfigPayload)
	// conf1.IfName = "eth2"  // like 'eth2'
	// // conf1.IPv4Addr = "192.168.78.31"
	// // conf1.IPv4Mask = 24
	// // conf1.IPv4BCast = "192.168.78.255"
	// conf1.DhcpV4Enabled = true
	// conf1.ClearAddresses = true

	// op := new (maestroSpecs.NetInterfaceOpPayload)
	// op.Type = maestroSpecs.OP_TYPE_NET_INTERFACE
	// op.Op = maestroSpecs.OP_UPDATE_ADDRESS
	// op.IfConfig = conf1
	// op.TaskId = "TestId"

	// task := new(tasks.MaestroTask)

	// task.Id = "Example"
	// task.Src = "Test"
	// task.Op = op

	// fmt.Printf("Submitting task %+v\n",task)
	// err := manager.SubmitTask(task)

	if err != nil {
		log.Fatalf("Error on Network manager SubmitTask() %s %+v", err.Error(), err)
	}

	timeout := 60 * 60 * 12
	fmt.Printf("Waiting on threads (%d seconds)...\n", timeout)
	manager.waitForActiveInterfaceThreads(timeout)

	fmt.Printf("Now we will try SetupExistingInterface() ...\n")

	manager.enableThreadCount() // enable block on thread count
	manager.SetupExistingInterfaces(nil)

	storage.shutdown(manager)

}

// This test assumes an existing entry for the interface, and then replaces it
// with the new config, with Existing as "override"
func TestRestartWithConfigOverride2(t *testing.T) {
	// assumes TestNetworkManagerSetupExisting was ran previously
	storage := new(testStorageManager)
	storage.start()
	// manager := getNetworkManagerInstance(storage)

	config := &maestroSpecs.NetworkConfigPayload{
		DnsIgnoreDhcp: false,
	}

	ifconfig := &maestroSpecs.NetIfConfigPayload{
		IfName:         "eth2",
		ClearAddresses: true,
		DhcpV4Enabled:  false,
		HwAddr:         "f4:f9:51:00:01:02", // orig f4:f9:51:f2:2d:b3
		IPv4Addr:       "192.168.78.31",
		IPv4Mask:       24,
		IPv4BCast:      "192.168.78.255",
		Existing:       "override",
	}

	config.Interfaces = append(config.Interfaces, ifconfig)

	err := testInitNetworkManager(config, storage)
	if err != nil {
		log.Fatalf("Failed to setup test instance of network manager: %+v\n", err)
	}
	manager := testGetInstance(storage)

	// conf1 := new(maestroSpecs.NetIfConfigPayload)
	// conf1.IfName = "eth2"  // like 'eth2'
	// // conf1.IPv4Addr = "192.168.78.31"
	// // conf1.IPv4Mask = 24
	// // conf1.IPv4BCast = "192.168.78.255"
	// conf1.DhcpV4Enabled = true
	// conf1.ClearAddresses = true

	// op := new (maestroSpecs.NetInterfaceOpPayload)
	// op.Type = maestroSpecs.OP_TYPE_NET_INTERFACE
	// op.Op = maestroSpecs.OP_UPDATE_ADDRESS
	// op.IfConfig = conf1
	// op.TaskId = "TestId"

	// task := new(tasks.MaestroTask)

	// task.Id = "Example"
	// task.Src = "Test"
	// task.Op = op

	// fmt.Printf("Submitting task %+v\n",task)
	// err := manager.SubmitTask(task)

	if err != nil {
		log.Fatalf("Error on Network manager SubmitTask() %s %+v", err.Error(), err)
	}

	timeout := 60 * 60 * 12
	fmt.Printf("Waiting on threads (%d seconds)...\n", timeout)
	manager.waitForActiveInterfaceThreads(timeout)

	fmt.Printf("Now we will try SetupExistingInterface() ...\n")

	manager.enableThreadCount() // enable block on thread count
	manager.SetupExistingInterfaces(nil)

	storage.shutdown(manager)

}
