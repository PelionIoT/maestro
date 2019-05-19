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
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/armPelionEdge/maestro/storage"
	"github.com/boltdb/bolt"
)

type testStorageManager struct {
	db     *bolt.DB
	dbname string
}

const (
	TEST_DB_NAME = "testMdnsServer.db"
)

// func (this *testStorageManager) start() {
// 	this.dbname = TEST_DB_NAME
// 	db, err := bolt.Open(TEST_DB_NAME, 0600, nil)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	this.db = db
// }

// func (this *testStorageManager) GetDb() *bolt.DB {
// 	return this.db
// }

// func (this *testStorageManager) GetDbName() string {
// 	return this.dbname
// }

// func (this *testStorageManager) RegisterStorageUser(user storage.StorageUser) {
// 	if this.db != nil {
// 		user.StorageInit(this)
// 		user.StorageReady(this)
// 	} else {
// 		log.Fatal("test DB failed.")
// 	}
// }

// func (this *testStorageManager) shutdown(user storage.StorageUser) {
// 	user.StorageClosed(this)
// }

func TestMain(m *testing.M) {
	storage.InitStorage(TEST_DB_NAME)
	m.Run()
}

func TestPublishAllInterfacesFromConfig(t *testing.T) {
	config := &ConfigEntry{
		Name:    "TestMdns1",
		Service: "_ssh._tcp",
		Domain:  "local",
		Port:    22,
		Text:    []string{"txtv=0", "lo=1", "la=2"},
		//		NotPersistent: true,
	}

	mdns := GetInstance()

	entries := []*ConfigEntry{config}

	ok, errs := mdns.LoadFromConfigFile(entries)

	if !ok {
		fmt.Print("LoadFromConfigFile failed.")
		for n, err := range errs {
			if err != nil {
				log.Fatalf("Entry %d failed: %s", n, err.Error())
			}
		}
	}

	fmt.Printf("Serving mdns record(s) Sleeping 30 seconds...")
	time.Sleep(time.Second * 600)

}

// ***DEBUG_GO:mdns: record: &{Name:TestMdns1 Service:_ssh._tcp Domain:local Port:22 TTL:0 Text:[txtv=0 lo=1 la=2] Hostname: Ips: Interfaces: NotInterfaces: NotPersistent:true servers:[]}

// ***DEBUG_GO:mdns: record: &{Name:MaestroStaticRecordTest Service:_wwservices._tcp Domain: Port:313 TTL:0 Text:[] Hostname: Ips: Interfaces: NotInterfaces: NotPersistent:true servers:[]}
func TestArbitraryIPServer(t *testing.T) {
	config := &ConfigEntry{
		Name:    "TestArbitraryIPServer",
		Service: "_specialsauce._tcp",
		Domain:  "local", // most Apple Bonjour browsers can only see "local"
		Port:    22,
		Text:    []string{"this=is_a_test"},
		Ips:     "1.1.1.1,2.2.2.2",
		//		NotPersistent: true,
	}

	mdns := GetInstance()

	entries := []*ConfigEntry{config}

	ok, errs := mdns.LoadFromConfigFile(entries)

	if !ok {
		fmt.Print("LoadFromConfigFile failed.")
		for n, err := range errs {
			if err != nil {
				log.Fatalf("Entry %d failed: %s", n, err.Error())
			}
		}
	}

	fmt.Printf("Serving mdns record(s) Sleeping 30 seconds...")
	time.Sleep(time.Second * 600)

}

func TestSpecificInterface(t *testing.T) {
	config := &ConfigEntry{
		Name:       "TestSpecificInterface",
		Service:    "_specialsauce2._tcp",
		Domain:     "local", // most Apple Bonjour browsers can only see "local"
		Port:       22,
		Text:       []string{"this=is_a_test"},
		Interfaces: "eth1,wlan1",
		//		Ips:     "1.1.1.1,2.2.2.2",
		//		NotPersistent: true,
	}

	mdns := GetInstance()

	entries := []*ConfigEntry{config}

	ok, errs := mdns.LoadFromConfigFile(entries)

	if !ok {
		fmt.Print("LoadFromConfigFile failed.")
		for n, err := range errs {
			if err != nil {
				log.Fatalf("Entry %d failed: %s", n, err.Error())
			}
		}
	}

	fmt.Printf("Serving mdns record(s) Sleeping 30 seconds...")
	time.Sleep(time.Second * 600)

}

// To run test:
// sub out for your directories
// sudo GOROOT=/opt/go PATH="$PATH:/opt/go/bin" GOPATH=/home/ed/work/gostuff LD_LIBRARY_PATH=../../greasego/deps/lib /opt/go/bin/go test
// to try a specific test add:
// -run TestArbitraryIPServer
