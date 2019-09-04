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
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	
	"github.com/armPelionEdge/dhcp4"
	"github.com/armPelionEdge/dhcp4client"
	"github.com/armPelionEdge/maestro/events"
	"github.com/armPelionEdge/maestro/storage"
	"github.com/armPelionEdge/maestro/tasks"
	"github.com/armPelionEdge/maestro/maestroConfig"
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

const DHCP_TEST_INTERFACE_NAME = "wlan1"

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

	success, newleaseinfo, err := RequestOrRenewDhcpLease(DHCP_TEST_INTERFACE_NAME, &leaseinfo, requestopts)

	if success != dhcp4client.Success {
		log.Fatalf("Failed to renew DHCP lease: %+v\n", success)
	}

	if newleaseinfo != &leaseinfo {
		log.Fatalf("Passed in a valid DhcpLeaseInfo pointer pointing to a struct but failed. %+v\n", err)
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
		log.Fatalf("Failed to GetInterfaceMacAddress - %s\n", err.Error())
	} else {
		if index < 1 {
			log.Fatal("index is not sane.")
			return
		}
		macstring := mac.String()
		err = SetInterfaceMacAddress("eth2", 0, macstring)
		if err != nil {
			log.Fatalf("Failed to SetInterfaceMacAddress - %s\n", err.Error())
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
		log.Fatalf("NetworkManager: CRITICAL - error creating global channel \"events\" %s\n", err.Error())
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
		log.Fatalf("Error on Network manager SubmitTask() %s %+v\n", err.Error(), err)
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
		log.Fatalf("Error on Network manager SubmitTask() %s %+v\n", err.Error(), err)
	}

	timeout := 60 * 60 * 12
	fmt.Printf("Waiting on threads (%d seconds)...\n", timeout)
	manager.waitForActiveInterfaceThreads(timeout)

	fmt.Printf("Now we will try SetupExistingInterface(nil) ...\n")

	manager.SetupExistingInterfaces()

	storage.shutdown(manager)

}

func TestRestartNetworkManagerDhcp(t *testing.T) {

	storage := new(testStorageManager)
	storage.start()
	manager := getNetworkManagerInstance(storage)

	manager.enableThreadCount() // enable block on thread count

	fmt.Printf("Now we will try SetupExistingInterface(nil) ...\n")

	manager.SetupExistingInterfaces()

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
		log.Fatalf("Error on Network manager SubmitTask() %s %+v\n", err.Error(), err)
	}

	timeout := 60 * 60 * 12
	fmt.Printf("Waiting on threads (%d seconds)...\n", timeout)
	manager.waitForActiveInterfaceThreads(timeout)

	fmt.Printf("Now we will try SetupExistingInterface() ...\n")

	manager.enableThreadCount() // enable block on thread count
	manager.SetupExistingInterfaces()

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
		log.Fatalf("Error on Network manager SubmitTask() %s %+v\n", err.Error(), err)
	}

	timeout := 60 * 60 * 12
	fmt.Printf("Waiting on threads (%d seconds)...\n", timeout)
	manager.waitForActiveInterfaceThreads(timeout)

	fmt.Printf("Now we will try SetupExistingInterface() ...\n")

	manager.enableThreadCount() // enable block on thread count
	manager.SetupExistingInterfaces()

	storage.shutdown(manager)

}

func TestNetworkConfigInDDB(t *testing.T) {
	var devicedbUri string = "https://WWRL000000:9090" //The URI of the relay's local DeviceDB instan*client.e
	var devicedbPrefix string = "wigwag.configs.relay" //The prefix where keys related to configuration are stored
	var devicedbBucket string = "local" //"The devicedb bucket where configurations are stored
	var relay string = "WWRL000000" //The ID of the relay whose configuration should be monitored
	var relayCaChainFile string = "../test-assets/ca-chain.cert.pem" //The file path to a PEM encoded CA chain used to validate the server certificate used by the DeviceDB instance
	var tlsConfig *tls.Config

	// fresh start - no database
	os.Remove(TEST_DB_NAME)
	
	relayCaChain, err := ioutil.ReadFile(relayCaChainFile)
	if err != nil {
		fmt.Printf("Unable to load CA chain from %s: %v\n", relayCaChainFile, err)
		t.FailNow()
	}

	caCerts := x509.NewCertPool()

	if !caCerts.AppendCertsFromPEM(relayCaChain) {
		fmt.Printf("CA chain loaded from %s is not valid: %v\n", relayCaChainFile, err)
		t.FailNow()
	}

	tlsConfig = &tls.Config{
		RootCAs: caCerts,
	}

	// assumes TestNetworkManagerSetupExisting was ran previously
	storage := new(testStorageManager)
	storage.start()
	// manager := getNetworkManagerInstance(storage)

	//Delete any object that many be in devicedb for testing
	var ddbNetworkConfig maestroSpecs.NetworkConfigPayload
	configClient := maestroConfig.NewDDBRelayConfigClient(tlsConfig, devicedbUri, relay, devicedbPrefix, devicedbBucket)
	err = configClient.Config(DDB_NETWORK_CONFIG_NAME).Delete()
	if(err != nil) {
		log.Fatalf("Failed to delete config object from devicedb err: %v.\n", err)
		t.FailNow()
	} else {
		fmt.Printf("\nDeleted network config in devicedb\n")
	}

	//Wait for sometime for delete to commit
	time.Sleep(time.Second * 2)

	err = configClient.Config(DDB_NETWORK_CONFIG_NAME).Get(&ddbNetworkConfig)
	if(err == nil) {
		log.Fatalf("Unable to delete object from devicedb\n",)
		t.FailNow()
	}

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
	
	err = testInitNetworkManager(config, storage)
	if err != nil {
		log.Fatalf("Failed to setup test instance of network manager: %+v\n", err)
		t.FailNow()
	}
	manager := testGetInstance(storage)
	manager.waitForDeviceDB = false
	manager.ddbConnConfig = &maestroConfig.DeviceDBConnConfig {devicedbUri, devicedbPrefix, devicedbBucket, relay, relayCaChainFile}

	timeout := 60 * 60 * 12
	fmt.Printf("Waiting on threads (%d seconds)...\n", timeout)
	manager.waitForActiveInterfaceThreads(timeout)

	fmt.Printf("Now we will try SetupExistingInterface() ...\n")

	manager.enableThreadCount() // enable block on thread count
	manager.SetupExistingInterfaces()
	
	//Wait for sometime for everything to come up on Network manager
	time.Sleep(time.Second * 2)

	err = configClient.Config(DDB_NETWORK_CONFIG_NAME).Get(&ddbNetworkConfig)
	if(err != nil) {
		log.Fatalf("No network config found in devicedb or unable to connect to devicedb err: %v. Let's try to put the current config we have from config file.\n", err)
		t.FailNow()
	} else {
		fmt.Printf("\nFound a network config in devicedb: %v:%v\n", ddbNetworkConfig, *ddbNetworkConfig.Interfaces[0])
	}

	storage.shutdown(manager)
}

func TestNetworkConfigSimpleUpdateInDDB(t *testing.T) {
	var devicedbUri string = "https://WWRL000000:9090" //The URI of the relay's local DeviceDB instan*client.e
	var devicedbPrefix string = "wigwag.configs.relay" //The prefix where keys related to configuration are stored
	var devicedbBucket string = "local" //"The devicedb bucket where configurations are stored
	var relay string = "WWRL000000" //The ID of the relay whose configuration should be monitored
	var relayCaChainFile string = "../test-assets/ca-chain.cert.pem" //The file path to a PEM encoded CA chain used to validate the server certificate used by the DeviceDB instance
	var tlsConfig *tls.Config

	// fresh start - no database
	os.Remove(TEST_DB_NAME)
	
	relayCaChain, err := ioutil.ReadFile(relayCaChainFile)
	if err != nil {
		fmt.Printf("Unable to load CA chain from %s: %v\n", relayCaChainFile, err)
		t.FailNow()
	}

	caCerts := x509.NewCertPool()

	if !caCerts.AppendCertsFromPEM(relayCaChain) {
		fmt.Printf("CA chain loaded from %s is not valid: %v\n", relayCaChainFile, err)
		t.FailNow()
	}

	tlsConfig = &tls.Config{
		RootCAs: caCerts,
	}

	// assumes TestNetworkManagerSetupExisting was ran previously
	storage := new(testStorageManager)
	storage.start()

	//Delete any object that many be in devicedb for testing
	var ddbNetworkConfig maestroSpecs.NetworkConfigPayload
	configClient := maestroConfig.NewDDBRelayConfigClient(tlsConfig, devicedbUri, relay, devicedbPrefix, devicedbBucket)
	err = configClient.Config(DDB_NETWORK_CONFIG_NAME).Delete()
	if(err != nil) {
		log.Fatalf("Failed to delete config object from devicedb err: %v.\n", err)
		t.FailNow()
	} else {
		fmt.Printf("\nDeleted network config in devicedb\n")
	}

	//Wait for sometime for delete to commit
	time.Sleep(time.Second * 2)

	err = configClient.Config(DDB_NETWORK_CONFIG_NAME).Get(&ddbNetworkConfig)
	if(err == nil) {
		log.Fatalf("Unable to delete object from devicedb\n",)
		t.FailNow()
	}
	
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

	err = testInitNetworkManager(config, storage)
	if err != nil {
		log.Fatalf("Failed to setup test instance of network manager: %+v\n", err)
		t.FailNow()
	}
	manager := testGetInstance(storage)
	manager.waitForDeviceDB = false
	manager.ddbConnConfig = &maestroConfig.DeviceDBConnConfig {devicedbUri, devicedbPrefix, devicedbBucket, relay, relayCaChainFile}

	timeout := 60 * 60 * 12
	fmt.Printf("Waiting on threads (%d seconds)...\n", timeout)
	manager.waitForActiveInterfaceThreads(timeout)

	manager.enableThreadCount() // enable block on thread count
	manager.SetupExistingInterfaces()
	
	//Wait for sometime for everything to come up on Network manager
	time.Sleep(time.Second * 3)

	//Now change something in devicedb
	err = configClient.Config(DDB_NETWORK_CONFIG_NAME).Get(&ddbNetworkConfig)
	if(err == nil) {
		fmt.Printf("Current DnsIgnoreDhcp value in network manager: %v\n", ddbNetworkConfig.DnsIgnoreDhcp)
		ddbNetworkConfig.DnsIgnoreDhcp = true;
	} else {
		log.Fatalf("Unable to read Network config object from devicedb\n")
		t.FailNow()
	}

	//Now write out the updated config
	err = configClient.Config(DDB_NETWORK_CONFIG_NAME).Put(&ddbNetworkConfig)
	if(err == nil) {
		fmt.Printf("devicedb update succeeded.\n")
	} else {
		log.Fatalf("Unable to update Network config object to devicedb\n")
		t.FailNow()
	}

	//Wait for updates to propagate and processed by maestro
	time.Sleep(time.Second * 2)

	fmt.Printf("Validate DnsIgnoreDhcp value in network manager\n")
	if(manager.networkConfig.DnsIgnoreDhcp != true) {
		log.Fatalf("Test failed, values are different for DnsIgnoreDhcp expected:true actual:%v\n",manager.networkConfig.DnsIgnoreDhcp)
		t.FailNow()
	} else {
		fmt.Printf("DnsIgnoreDhcp value in network manager: %v\n", manager.networkConfig.DnsIgnoreDhcp)
	}

	storage.shutdown(manager)
}

func TestConfigCommitUpdateInDDB(t *testing.T) {
	var devicedbUri string = "https://WWRL000000:9090" //The URI of the relay's local DeviceDB instan*client.e
	var devicedbPrefix string = "wigwag.configs.relay" //The prefix where keys related to configuration are stored
	var devicedbBucket string = "local" //"The devicedb bucket where configurations are stored
	var relay string = "WWRL000000" //The ID of the relay whose configuration should be monitored
	var relayCaChainFile string = "../test-assets/ca-chain.cert.pem" //The file path to a PEM encoded CA chain used to validate the server certificate used by the DeviceDB instance
	var tlsConfig *tls.Config

	// fresh start - no database
	os.Remove(TEST_DB_NAME)
	
	relayCaChain, err := ioutil.ReadFile(relayCaChainFile)
	if err != nil {
		fmt.Printf("Unable to load CA chain from %s: %v\n", relayCaChainFile, err)
		t.FailNow()
	}

	caCerts := x509.NewCertPool()

	if !caCerts.AppendCertsFromPEM(relayCaChain) {
		fmt.Printf("CA chain loaded from %s is not valid: %v\n", relayCaChainFile, err)
		t.FailNow()
	}

	tlsConfig = &tls.Config{
		RootCAs: caCerts,
	}

	// assumes TestNetworkManagerSetupExisting was ran previously
	storage := new(testStorageManager)
	storage.start()

	//Delete any object that many be in devicedb for testing
	var ddbCommitConfig ConfigCommit
	configClient := maestroConfig.NewDDBRelayConfigClient(tlsConfig, devicedbUri, relay, devicedbPrefix, devicedbBucket)
	err = configClient.Config(DDB_NETWORK_CONFIG_COMMIT_FLAG).Delete()
	if(err != nil) {
		log.Fatalf("Failed to delete commit config object from devicedb err: %v.\n", err)
		t.FailNow()
	} else {
		fmt.Printf("\nDeleted commit config in devicedb\n")
	}

	//Wait for sometime for delete to commit
	time.Sleep(time.Second * 2)

	err = configClient.Config(DDB_NETWORK_CONFIG_COMMIT_FLAG).Get(&ddbCommitConfig)
	if(err == nil) {
		log.Fatalf("Unable to delete commit object from devicedb\n",)
		t.FailNow()
	}
	
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

	err = testInitNetworkManager(config, storage)
	if err != nil {
		log.Fatalf("Failed to setup test instance of network manager: %+v\n", err)
		t.FailNow()
	}
	manager := testGetInstance(storage)
	manager.waitForDeviceDB = false
	manager.ddbConnConfig = &maestroConfig.DeviceDBConnConfig {devicedbUri, devicedbPrefix, devicedbBucket, relay, relayCaChainFile}

	timeout := 60 * 60 * 12
	fmt.Printf("Waiting on threads (%d seconds)...\n", timeout)
	manager.waitForActiveInterfaceThreads(timeout)

	manager.enableThreadCount() // enable block on thread count
	manager.SetupExistingInterfaces()
	
	//Wait for sometime for everything to come up on Network manager
	time.Sleep(time.Second * 3)

	//Now change something in devicedb
	err = configClient.Config(DDB_NETWORK_CONFIG_COMMIT_FLAG).Get(&ddbCommitConfig)
	if(err == nil) {
		fmt.Printf("Current commit flag value in devicedb: %v\n", ddbCommitConfig.ConfigCommitFlag)
		ddbCommitConfig.ConfigCommitFlag = true;
	} else {
		log.Fatalf("Unable to read Network config object from devicedb\n")
		t.FailNow()
	}

	//Now write out the updated config
	err = configClient.Config(DDB_NETWORK_CONFIG_COMMIT_FLAG).Put(&ddbCommitConfig)
	if(err == nil) {
		fmt.Printf("Devicedb update succeeded.\n")
	} else {
		log.Fatalf("Unable to update commit config object to devicedb\n")
		t.FailNow()
	}

	//Wait for updates to propagate and processed by maestro
	time.Sleep(time.Second * 2)

	//mt.Printf("Validate commit config object, it should be false as the flag should revert to false after commtting the updates\n")
	if((manager.configCommit.ConfigCommitFlag != false) || (manager.configCommit.TotalCommitCountFromBoot < 0) || (len(instance.configCommit.LastUpdateTimestamp) <= 0)) {
		log.Fatalf("Test failed, values are different:%v\n",manager.configCommit)
		t.FailNow()
	} else {
		fmt.Printf("ConfigCommitFlag value in network manager: %v\n", manager.configCommit.ConfigCommitFlag)
	}

	storage.shutdown(manager)
}

func TestNetworkConfigUpdateInDDBMultipleInterfaces(t *testing.T) {
	var devicedbUri string = "https://WWRL000000:9090" //The URI of the relay's local DeviceDB instan*client.e
	var devicedbPrefix string = "wigwag.configs.relay" //The prefix where keys related to configuration are stored
	var devicedbBucket string = "local" //"The devicedb bucket where configurations are stored
	var relay string = "WWRL000000" //The ID of the relay whose configuration should be monitored
	var relayCaChainFile string = "../test-assets/ca-chain.cert.pem" //The file path to a PEM encoded CA chain used to validate the server certificate used by the DeviceDB instance
	var tlsConfig *tls.Config

	// fresh start - no database
	os.Remove(TEST_DB_NAME)
	
	relayCaChain, err := ioutil.ReadFile(relayCaChainFile)
	if err != nil {
		fmt.Printf("Unable to load CA chain from %s: %v\n", relayCaChainFile, err)
		t.FailNow()
	}

	caCerts := x509.NewCertPool()

	if !caCerts.AppendCertsFromPEM(relayCaChain) {
		fmt.Printf("CA chain loaded from %s is not valid: %v\n", relayCaChainFile, err)
		t.FailNow()
	}

	tlsConfig = &tls.Config{
		RootCAs: caCerts,
	}

	// assumes TestNetworkManagerSetupExisting was ran previously
	storage := new(testStorageManager)
	storage.start()
	// manager := getNetworkManagerInstance(storage)

	//Delete any object that many be in devicedb for testing
	var ddbNetworkConfig maestroSpecs.NetworkConfigPayload
	configClient := maestroConfig.NewDDBRelayConfigClient(tlsConfig, devicedbUri, relay, devicedbPrefix, devicedbBucket)
	err = configClient.Config(DDB_NETWORK_CONFIG_NAME).Delete()
	if(err != nil) {
		log.Fatalf("Failed to delete config object from devicedb err: %v.\n", err)
		t.FailNow()
	} else {
		fmt.Printf("\nDeleted network config in devicedb\n")
	}

	//Wait for sometime for delete to commit
	time.Sleep(time.Second * 2)

	err = configClient.Config(DDB_NETWORK_CONFIG_NAME).Get(&ddbNetworkConfig)
	if(err == nil) {
		log.Fatalf("Unable to delete object from devicedb\n",)
		t.FailNow()
	}

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
	ifconfig2 := &maestroSpecs.NetIfConfigPayload{
		IfName:         "eth3",
		ClearAddresses: true,
		DhcpV4Enabled:  false,
		HwAddr:         "f4:f9:51:00:01:03", // orig f4:f9:51:f2:2d:b3
		IPv4Addr:       "192.168.78.32",
		IPv4Mask:       24,
		IPv4BCast:      "192.168.78.255",
		Existing:       "override",
	}

	config.Interfaces = append(config.Interfaces, ifconfig)
	config.Interfaces = append(config.Interfaces, ifconfig2)
	
	err = testInitNetworkManager(config, storage)
	if err != nil {
		log.Fatalf("Failed to setup test instance of network manager: %+v\n", err)
		t.FailNow()
	}
	manager := testGetInstance(storage)
	manager.waitForDeviceDB = false
	manager.ddbConnConfig = &maestroConfig.DeviceDBConnConfig {devicedbUri, devicedbPrefix, devicedbBucket, relay, relayCaChainFile}

	timeout := 60 * 60 * 12
	fmt.Printf("Waiting on threads (%d seconds)...\n", timeout)
	manager.waitForActiveInterfaceThreads(timeout)

	fmt.Printf("Now we will try SetupExistingInterface() ...\n")

	manager.enableThreadCount() // enable block on thread count
	manager.SetupExistingInterfaces()
	
	//Wait for sometime for everything to come up on Network manager
	time.Sleep(time.Second * 3)

	err = configClient.Config(DDB_NETWORK_CONFIG_NAME).Get(&ddbNetworkConfig)
	if(err != nil) {
		log.Fatalf("No network config found in devicedb or unable to connect to devicedb err: %v. Let's try to put the current config we have from config file.\n", err)
		t.FailNow()
	} else {
		fmt.Printf("Test: Found a network config in devicedb: %v:%v\n", ddbNetworkConfig, ddbNetworkConfig.Interfaces)
	}

	updatedConfig := &maestroSpecs.NetworkConfigPayload{
		DnsIgnoreDhcp: true,
	}

	ifconfig3 := &maestroSpecs.NetIfConfigPayload{
		IfName:         "wifi1",
		ClearAddresses: true,
		DhcpV4Enabled:  false,
		HwAddr:         "f4:f9:51:00:01:02", // orig f4:f9:51:f2:2d:b3
		IPv4Addr:       "178.168.78.41",
		IPv4Mask:       24,
		IPv4BCast:      "192.168.78.255",
		Existing:       "override",
	}
	ifconfig4 := &maestroSpecs.NetIfConfigPayload{
		IfName:         "wifi2",
		ClearAddresses: true,
		DhcpV4Enabled:  false,
		HwAddr:         "f4:f9:51:00:01:03", // orig f4:f9:51:f2:2d:b3
		IPv4Addr:       "178.168.78.42",
		IPv4Mask:       24,
		IPv4BCast:      "192.168.78.255",
		Existing:       "override",
	}
	updatedConfig.Interfaces = append(updatedConfig.Interfaces, ifconfig3)
	updatedConfig.Interfaces = append(updatedConfig.Interfaces, ifconfig4)
	
	err = configClient.Config(DDB_NETWORK_CONFIG_NAME).Put(&updatedConfig)
	if(err != nil) {
		log.Fatalf("Unable to put updated config: %v\n", err)
		t.FailNow()
	} else {
		fmt.Printf("\nUpdated the network config in devicedb: %v:%v:%v\n", updatedConfig, *updatedConfig.Interfaces[0], *updatedConfig.Interfaces[1])
	}
	
	//Wait for sometime for everything to come up on Network manager
	time.Sleep(time.Second * 2)

	if(manager.networkConfig.Interfaces[0].IfName != "wifi1") {
		log.Fatalf("Test failed, values are different for Interfaces[0].IfName expected:wifi1 actual:%v\n",manager.networkConfig.Interfaces[0].IfName)
		t.FailNow()
	}
	if(manager.networkConfig.Interfaces[1].IfName != "wifi2") {
		log.Fatalf("Test failed, values are different for Interfaces[0].IfName expected:wifi2 actual:%v\n",manager.networkConfig.Interfaces[1].IfName)
		t.FailNow()
	}

	//Add another set of changes
	updatedConfig2 := &maestroSpecs.NetworkConfigPayload{
		DnsIgnoreDhcp: true,
	}

	ifconfig5 := &maestroSpecs.NetIfConfigPayload{
		IfName:         "eth5",
		ClearAddresses: true,
		DhcpV4Enabled:  false,
		HwAddr:         "f4:f9:51:00:01:02", // orig f4:f9:51:f2:2d:b3
		IPv4Addr:       "178.10.8.1",
		IPv4Mask:       24,
		IPv4BCast:      "192.168.78.255",
		AliasAddrV4:	[]maestroSpecs.AliasAddressV4{{"178.10.10.10", "178.255.255.255", "178.10.255.255"}, {"178.10.11.10", "178.255.255.255", "178.10.255.255"}},
		Existing:       "override",
	}
	ifconfig6 := &maestroSpecs.NetIfConfigPayload{
		IfName:         "eth6",
		ClearAddresses: true,
		DhcpV4Enabled:  false,
		HwAddr:         "f4:f9:51:00:01:03", // orig f4:f9:51:f2:2d:b3
		IPv4Addr:       "178.10.8.2",
		IPv4Mask:       24,
		IPv4BCast:      "192.168.78.255",
		AliasAddrV4:	[]maestroSpecs.AliasAddressV4{{"178.10.10.11", "178.255.255.255", "178.255.255.255"}, {"178.10.11.11", "178.255.255.255", "178.10.255.255"}},
		Existing:       "override",
	}
	updatedConfig2.Interfaces = append(updatedConfig2.Interfaces, ifconfig5)
	updatedConfig2.Interfaces = append(updatedConfig2.Interfaces, ifconfig6)
	
	err = configClient.Config(DDB_NETWORK_CONFIG_NAME).Put(&updatedConfig2)
	if(err != nil) {
		log.Fatalf("Unable to put updated config: %v\n", err)
		t.FailNow()
	} else {
		fmt.Printf("\nUpdated new network config in devicedb: %v:%v:%v\n", updatedConfig2, *updatedConfig2.Interfaces[0], *updatedConfig2.Interfaces[1])
	}
	
	//Wait for sometime for everything to come up on Network manager
	time.Sleep(time.Second * 2)

	if(manager.networkConfig.Interfaces[0].IfName != "eth5") {
		log.Fatalf("Test failed, values are different for Interfaces[0].IfName expected:eth5 actual:%v\n",manager.networkConfig.Interfaces[0].IfName)
		t.FailNow()
	}
	if(manager.networkConfig.Interfaces[1].IfName != "eth6") {
		log.Fatalf("Test failed, values are different for Interfaces[0].IfName expected:eth6 actual:%v\n",manager.networkConfig.Interfaces[1].IfName)
		t.FailNow()
	}

	storage.shutdown(manager)
}