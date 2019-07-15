package maestroConfig

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
	"strings"
	"testing"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"reflect"
	"sync"
	"time"
	"github.com/armPelionEdge/maestroSpecs"
)

func TestMain(m *testing.M) {
	m.Run()
}

func TestLoadFromFile(t *testing.T) {
	config_loader := new(YAMLMaestroConfig)
	err := config_loader.LoadFromFile("../test-assets/wwrelay-config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
		t.FailNow()
	} else {
		fmt.Printf("Loaded the config = %+v\n", config_loader)
	}
}

func TestConfigVarRead(t *testing.T) {
	config_loader := new(YAMLMaestroConfig)
	err := config_loader.LoadFromFile("../test-assets/wwrelay-config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
		t.FailNow()
	} else {
		if strings.Compare(config_loader.ClientId, "WWRL000001") != 0 {
			fmt.Printf("clientId from config = %+v expected:WWRL000001\n", config_loader.ClientId)
			t.FailNow()
		} else {
			fmt.Printf("clientId from config = %+v\n", config_loader.ClientId)
		}
	}
}

type ConfigChangeHook struct{
	//struct implementing ConfigChangeHook intf
}

type MyConfig struct {
	Property1 int    `json:"property1" configGroup:"property"`
	Property2 string `json:"property2" configGroup:"property"`
	Property3 bool   `json:"property3" configGroup:"property"`
	Property4 [5]int `json:"property4" configGroup:"property"`
}

type NestedConfigType struct {
	NestedProperty1 int    `json:"property1" configGroup:"diffproperty"`
	NestedProperty2 string `json:"property2" configGroup:"diffproperty"`
	NestedProperty3 bool   `json:"property3" configGroup:"diffproperty"`
	NestedProperty4 [5]int `json:"property4" configGroup:"diffproperty"`
}

type MyDiffConfig struct {
	Name string `json:"property1" configGroup:"diffproperty"`
	NestedStruct NestedConfigType `json:"property2" configGroup:"diffproperty"`
}

type MyConfigMultipleGroup struct {
	Property1 int   `json:"property1" configGroup:"someproperty"`
	Property2 int 	`json:"property2" configGroup:"someproperty"`
	Property3 int   `json:"property3" configGroup:"anotherproperty"`
	Property4 int 	`json:"property4" configGroup:"anotherproperty"`
}

// ChangesStart is called before reporting any changes via multiple calls to SawChange. It will only be called
// if there is at least one change to report
func (cfgHook ConfigChangeHook) ChangesStart(configgroup string) {
	fmt.Printf("\nConfigChangeHook:ChangesStart: %s\n", configgroup)
}

// SawChange is called whenever a field changes. It will be called only once for each field which is changed.
// It will always be called after ChangesStart is called
// If SawChange return true, then the value of futvalue will replace the value of current value
func (cfgHook ConfigChangeHook) SawChange(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}) (acceptchange bool) {
	fmt.Printf("\nConfigChangeHook:SawChange: %s:%s old:%v new:%v\n", configgroup, fieldchanged, curvalue, futvalue)
	if(configgroup == "property") {
		reflect.ValueOf(&myConfigUpdateFromHook).Elem().FieldByName(fieldchanged).Set(reflect.ValueOf(futvalue))
	}
	if(configgroup == "someproperty" || configgroup == "anotherproperty") {
		reflect.ValueOf(&myConfigMultipleGroup).Elem().FieldByName(fieldchanged).Set(reflect.ValueOf(futvalue))
	}
	if(configgroup == "diffproperty") {
		if(strings.HasPrefix(fieldchanged, "Nested")) {
			reflect.ValueOf(&myDiffConfigUpdateFromHook.NestedStruct).Elem().FieldByName(strings.Split(fieldchanged,".")[1]).Set(reflect.ValueOf(futvalue))	
		} else {
			reflect.ValueOf(&myDiffConfigUpdateFromHook).Elem().FieldByName(fieldchanged).Set(reflect.ValueOf(futvalue))
		}
	}
	return true;
}

// ChangesComplete is called when all changes for a specific configgroup tagname
// If ChangesComplete returns true, then all changes in that group will be assigned to the current struct
func (cfgHook ConfigChangeHook) ChangesComplete(configgroup string) (acceptallchanges bool) {
	sawChangeCount++
	fmt.Printf("\nConfigChangeHook:ChangesComplete: %s Change count=%d\n", configgroup, sawChangeCount)
	return true;
}

var myConfigUpdateFromHook MyConfig;
var myDiffConfigUpdateFromHook MyDiffConfig;
var myConfigMultipleGroup MyConfigMultipleGroup
func TestConfigMonitorSimple(t *testing.T) {
	var devicedbUri string = "https://WWRL000000:9090" //The URI of the relay's local DeviceDB instan*client.e
	var devicedbPrefix string = "wigwag.configs.relay" //The prefix where keys related to configuration are stored
	var devicedbBucket string = "local" //"The devicedb bucket where configurations are stored
	var relay string = "WWRL000000" //The ID of the relay whose configuration should be monitored
	var configName string = "myConfig" //The name of the configuration object that should be monitored
	var relayCaChainFile string = "../test-assets/ca-chain.cert.pem" //The file path to a PEM encoded CA chain used to validate the server certificate used by the DeviceDB instance
	var tlsConfig *tls.Config
	
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

	configClient := NewDDBRelayConfigClient(tlsConfig, devicedbUri, relay, devicedbPrefix, devicedbBucket)
	
	var config, updatedConfig MyConfig
	config.Property1 = 1
	config.Property2 = "a"
	config.Property3 = true
	config.Property4 = [...]int{0,0,0,0,0}
	err = configClient.Config(configName).Put(config)
	if err != nil {
		fmt.Printf("\nUnable to put config: %v\n", err)
		t.FailNow()
	}

	//Create a DDB connection config
	ddbConnConfig := new(DeviceDBConnConfig)
	ddbConnConfig.DeviceDBUri = devicedbUri
	ddbConnConfig.DeviceDBPrefix = devicedbPrefix
	ddbConnConfig.DeviceDBBucket = devicedbBucket
	ddbConnConfig.RelayId = relay
	ddbConnConfig.CaChainCert = relayCaChainFile

	err, ddbConfigMon := NewDeviceDBMonitor(ddbConnConfig)
	if(err != nil) {
		fmt.Printf("\nUnable to create config monitor: %v\n", err)
		t.FailNow()
	}

	//Add a config change monitor
	configAna := maestroSpecs.NewConfigAnalyzer("configGroup")
	if configAna == nil {
		fmt.Printf("Failed to create config analyzer object")
		t.FailNow()
	}

	var configChangeHook ConfigChangeHook
	configAna.AddHook("property", &configChangeHook)

	//Add monitor for this config
	ddbConfigMon.AddMonitorConfig(&config, &updatedConfig, configName, configAna)

	updatedConfig.Property1 = 100
	updatedConfig.Property2 = "asdf"
	updatedConfig.Property3 = false
	updatedConfig.Property4 = [...]int{1,2,3,4,5}
	fmt.Printf("\nPutting updated config: %v\n", config)
	err = configClient.Config(configName).Put(updatedConfig)
	if err != nil {
		fmt.Printf("\nUnable to put config: %v\n", err)
		t.FailNow()
	}	

	//Wait for few seconds for the callbacks to complete
	time.Sleep(time.Second * 3)
	
	//Now verify if the changes are updated
	if ( myConfigUpdateFromHook.Property1 != updatedConfig.Property1) { t.FailNow() }
	if ( myConfigUpdateFromHook.Property2 != updatedConfig.Property2) { t.FailNow() }
	if ( myConfigUpdateFromHook.Property3 != updatedConfig.Property3) { t.FailNow() }
	if ( myConfigUpdateFromHook.Property4 != updatedConfig.Property4) { t.FailNow() }

	//Remove monitor for this config
	ddbConfigMon.RemoveMonitorConfig(configName)
}

func TestConfigMonitorMultipleGroup(t *testing.T) {
	var devicedbUri string = "https://WWRL000000:9090" //The URI of the relay's local DeviceDB instan*client.e
	var devicedbPrefix string = "wigwag.configs.relay" //The prefix where keys related to configuration are stored
	var devicedbBucket string = "local" //"The devicedb bucket where configurations are stored
	var relay string = "WWRL000000" //The ID of the relay whose configuration should be monitored
	var configName string = "myConfigMultiGroup" //The name of the configuration object that should be monitored
	var relayCaChainFile string = "../test-assets/ca-chain.cert.pem" //The file path to a PEM encoded CA chain used to validate the server certificate used by the DeviceDB instance
	var tlsConfig *tls.Config
	
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

	configClient := NewDDBRelayConfigClient(tlsConfig, devicedbUri, relay, devicedbPrefix, devicedbBucket)
	
	var config, updatedConfig MyConfigMultipleGroup
	config.Property1 = 1
	config.Property2 = 2
	config.Property3 = 3
	config.Property4 = 4
	err = configClient.Config(configName).Put(config)
	if err != nil {
		fmt.Printf("\nUnable to put config: %v\n", err)
		t.FailNow()
	}

	//Create a DDB connection config
	ddbConnConfig := new(DeviceDBConnConfig)
	ddbConnConfig.DeviceDBUri = devicedbUri
	ddbConnConfig.DeviceDBPrefix = devicedbPrefix
	ddbConnConfig.DeviceDBBucket = devicedbBucket
	ddbConnConfig.RelayId = relay
	ddbConnConfig.CaChainCert = relayCaChainFile

	err, ddbConfigMon := NewDeviceDBMonitor(ddbConnConfig)
	if(err != nil) {
		fmt.Printf("\nUnable to create config monitor: %v\n", err)
		t.FailNow()
	}

	//Add a config change monitor
	configAna := maestroSpecs.NewConfigAnalyzer("configGroup")
	if configAna == nil {
		fmt.Printf("Failed to create config analyzer object")
		t.FailNow()
	}

	var configChangeHook1 ConfigChangeHook
	configAna.AddHook("someproperty", &configChangeHook1)

	var configChangeHook2 ConfigChangeHook
	configAna.AddHook("anotherproperty", &configChangeHook2)

	//Add monitor for this config
	ddbConfigMon.AddMonitorConfig(&config, &updatedConfig, configName, configAna)

	updatedConfig.Property1 = 100
	updatedConfig.Property2 = 200
	updatedConfig.Property3 = 300
	updatedConfig.Property4 = 400
	fmt.Printf("\nPutting updated config: %v\n", config)
	err = configClient.Config(configName).Put(updatedConfig)
	if err != nil {
		fmt.Printf("\nUnable to put config: %v\n", err)
		t.FailNow()
	}	

	//Wait for few seconds for the callbacks to complete
	time.Sleep(time.Second * 3)
	
	//Now verify if the changes are updated
	if ( myConfigMultipleGroup.Property1 != updatedConfig.Property1) { t.FailNow() }
	if ( myConfigMultipleGroup.Property2 != updatedConfig.Property2) { t.FailNow() }
	if ( myConfigMultipleGroup.Property3 != updatedConfig.Property3) { t.FailNow() }
	if ( myConfigMultipleGroup.Property4 != updatedConfig.Property4) { t.FailNow() }

	//Remove monitor for this config
	ddbConfigMon.RemoveMonitorConfig(configName)
}

func TestConfigMonitorMultipleStructs(t *testing.T) {
	var devicedbUri string = "https://WWRL000000:9090" //The URI of the relay's local DeviceDB instan*client.e
	var devicedbPrefix string = "wigwag.configs.relay" //The prefix where keys related to configuration are stored
	var devicedbBucket string = "local" //"The devicedb bucket where configurations are stored
	var relay string = "WWRL000000" //The ID of the relay whose configuration should be monitored
	var configName string = "myConfig" //The name of the configuration object that should be monitored
	var diffConfigName string = "myDiffConfig" //The name of the configuration object that should be monitored
	var relayCaChainFile string = "../test-assets/ca-chain.cert.pem" //The file path to a PEM encoded CA chain used to validate the server certificate used by the DeviceDB instance
	var tlsConfig *tls.Config
	
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

	configClient := NewDDBRelayConfigClient(tlsConfig, devicedbUri, relay, devicedbPrefix, devicedbBucket)
	
	var config, updatedConfig MyConfig
	var diffConfig, diffUpdatedConfig MyDiffConfig
	config.Property1 = 1
	config.Property2 = "a"
	config.Property3 = true
	config.Property4 = [...]int{0,0,0,0,0}
	err = configClient.Config(configName).Put(config)
	if err != nil {
		fmt.Printf("\nUnable to put config: %v\n", err)
		t.FailNow()
	}
	//Put diff config
	diffConfig.Name = "unset"
	diffConfig.NestedStruct = NestedConfigType{ -1, "--", false, [...]int{5,4,3,2,1} }
	err = configClient.Config(diffConfigName).Put(diffConfig)
	if err != nil {
		fmt.Printf("\nUnable to put diffConfig: %v\n", err)
		t.FailNow()
	}

	//Create a DDB connection config
	ddbConnConfig := new(DeviceDBConnConfig)
	ddbConnConfig.DeviceDBUri = devicedbUri
	ddbConnConfig.DeviceDBPrefix = devicedbPrefix
	ddbConnConfig.DeviceDBBucket = devicedbBucket
	ddbConnConfig.RelayId = relay
	ddbConnConfig.CaChainCert = relayCaChainFile

	err, ddbConfigMon := NewDeviceDBMonitor(ddbConnConfig)
	if(err != nil) {
		fmt.Printf("\nUnable to create config monitor: %v\n", err)
		t.FailNow()
	}

	////////////////////////////////////////////////////////////////////////////
	//Add a config change monitor for first struct
	configAna := maestroSpecs.NewConfigAnalyzer("configGroup")
	if configAna == nil {
		fmt.Printf("Failed to create config analyzer object")
		t.FailNow()
	}

	var configChangeHook ConfigChangeHook
	configAna.AddHook("property", &configChangeHook)

	//Add monitor for this config
	ddbConfigMon.AddMonitorConfig(&config, &updatedConfig, configName, configAna)
	////////////////////////////////////////////////////////////////////////////

	//Add a config change monitor for second struct
	configAna2 := maestroSpecs.NewConfigAnalyzer("configGroup")
	if configAna2 == nil {
		fmt.Printf("Failed to create config analyzer object(diffconfig)")
		t.FailNow()
	}

	var configChangeHook2 ConfigChangeHook
	configAna2.AddHook("diffproperty", &configChangeHook2)

	//Add monitor for this config
	ddbConfigMon.AddMonitorConfig(&diffConfig, &diffUpdatedConfig, diffConfigName, configAna2)
	////////////////////////////////////////////////////////////////////////////
	//Update MyConfig
	updatedConfig.Property1 = 100
	updatedConfig.Property2 = "asdf"
	updatedConfig.Property3 = false
	updatedConfig.Property4 = [...]int{1,2,3,4,5}
	fmt.Printf("\nPutting updated config: %v\n", updatedConfig)
	err = configClient.Config(configName).Put(updatedConfig)
	if err != nil {
		fmt.Printf("\nUnable to put config: %v\n", err)
		t.FailNow()
	}
	////////////////////////////////////////////////////////////////////////////
	//Update DiffConfig
	diffUpdatedConfig.Name = "diffconfig"
	nestedStruct := NestedConfigType{ 1000, "xyz", false, [...]int{11,22,33,44,55} }
	diffUpdatedConfig.NestedStruct = nestedStruct
	fmt.Printf("\nPutting updated diff config: %v\n", diffUpdatedConfig)
	err = configClient.Config(diffConfigName).Put(diffUpdatedConfig)
	if err != nil {
		fmt.Printf("\nUnable to put config: %v\n", err)
		t.FailNow()
	}
	/////////////////////////////////////////////////////////////////////////////

	//Wait for few seconds for the callbacks to complete
	time.Sleep(time.Second * 3)
	
	//Now verify if the changes are updated
	if ( myConfigUpdateFromHook.Property1 != updatedConfig.Property1) { t.FailNow() }
	if ( myConfigUpdateFromHook.Property2 != updatedConfig.Property2) { t.FailNow() }
	if ( myConfigUpdateFromHook.Property3 != updatedConfig.Property3) { t.FailNow() }
	if ( myConfigUpdateFromHook.Property4 != updatedConfig.Property4) { t.FailNow() }

	if ( myDiffConfigUpdateFromHook.Name != diffUpdatedConfig.Name) { t.FailNow() }
	if ( myDiffConfigUpdateFromHook.NestedStruct != nestedStruct) { t.FailNow() }
	
	//Remove monitor for this config
	ddbConfigMon.RemoveMonitorConfig(configName)
}

//Global wait group
var wg sync.WaitGroup
var sawChangeCount int = 0
func TestConfigMonitorMultipleUpdates(t *testing.T) {
	var devicedbUri string = "https://WWRL000000:9090" //The URI of the relay's local DeviceDB instan*client.e
	var devicedbPrefix string = "wigwag.configs.relay" //The prefix where keys related to configuration are stored
	var devicedbBucket string = "local" //"The devicedb bucket where configurations are stored
	var relay string = "WWRL000000" //The ID of the relay whose configuration should be monitored
	var configName string = "myConfig" //The name of the configuration object that should be monitored
	var relayCaChainFile string = "../test-assets/ca-chain.cert.pem" //The file path to a PEM encoded CA chain used to validate the server certificate used by the DeviceDB instance
	var tlsConfig *tls.Config
	
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

	configClient := NewDDBRelayConfigClient(tlsConfig, devicedbUri, relay, devicedbPrefix, devicedbBucket)
	
	var config, updatedConfig MyConfig
	updatedConfig.Property1 = 100
	updatedConfig.Property2 = "asdf"
	updatedConfig.Property3 = false
	updatedConfig.Property4 = [...]int{1,2,3,4,5}
	fmt.Printf("\nPutting config: %v\n", updatedConfig)
	err = configClient.Config(configName).Put(updatedConfig)
	if err != nil {
		fmt.Printf("\nUnable to put config: %v\n", err)
		t.FailNow()
	}

	//Create a DDB connection config
	ddbConnConfig := new(DeviceDBConnConfig)
	ddbConnConfig.DeviceDBUri = devicedbUri
	ddbConnConfig.DeviceDBPrefix = devicedbPrefix
	ddbConnConfig.DeviceDBBucket = devicedbBucket
	ddbConnConfig.RelayId = relay
	ddbConnConfig.CaChainCert = relayCaChainFile

	err, ddbConfigMon := NewDeviceDBMonitor(ddbConnConfig)
	if(err != nil) {
		fmt.Printf("\nUnable to create config monitor: %v\n", err)
		t.FailNow()
	}

	//Add a config change monitor
	configAna := maestroSpecs.NewConfigAnalyzer("configGroup")
	if configAna == nil {
		fmt.Printf("Failed to create config analyzer object")
		t.FailNow()
	}

	var configChangeHook ConfigChangeHook
	configAna.AddHook("property", &configChangeHook)

	//Set the change cnt to 0 before adding monitor
	sawChangeCount = 0

	//Add monitor for this config
	ddbConfigMon.AddMonitorConfig(&config, &updatedConfig, configName, configAna)
	//Wait for couple of seconds before starting the updater
	time.Sleep(time.Second * 2)
	wg.Add(1)
	go ConfigUpdater(configClient)
	if( true == waitTimeout(&wg, time.Second * 60)) {
		//Timeout waiting for loop to exit, so fail
		t.FailNow()
	}

	if(sawChangeCount < 5) {
		fmt.Printf("Didn't see all the changes, test failed")
		t.FailNow()
	}
	//Remove monitor for this config
	ddbConfigMon.RemoveMonitorConfig(configName)
}

func ConfigUpdater(ddbClient *DDBRelayConfigClient) {
	var updateIntVal int = 0
	var configName string = "myConfig"
	
	defer wg.Done()

	for i := 0; i < 5; i++ {
		var newConfig MyConfig
		updateIntVal++
		newConfig.Property1 = updateIntVal
		newConfig.Property2 = "xyz"
		newConfig.Property4[0] = updateIntVal * 1
		newConfig.Property4[1] = updateIntVal * 2
		newConfig.Property4[2] = updateIntVal * 3
		err := ddbClient.Config(configName).Put(&newConfig)
		if(err != nil) {
			fmt.Printf("\nUpdating(Put) config failed: %v %v", newConfig, err)
		}
		time.Sleep(time.Second * 3)
	}
}

// waitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
    c := make(chan struct{})
    go func() {
        defer close(c)
        wg.Wait()
    }()
    select {
    case <-c:
        return false // completed normally
    case <-time.After(timeout):
        return true // timed out
    }
}

// package main

// import (
//  "fmt"
//  "regexp"
//  "strings"
// )

// const _re_findEnvSubs="\\${(.*)}"
// const _re_innerEnv = "([^|]+)\\|?(.*)"

// var re_findEnvSubs *regexp.Regexp
// var re_innerEnv *regexp.Regexp

// func subInEnvVars(in string, varmap map[string]string) (out string) {
//  m := re_findEnvSubs.FindAllStringSubmatch(in,-1)
//  if m != nil {
//      for _, match := range m {
//          inner := re_innerEnv.FindStringSubmatch(match[1])
//          if inner != nil {
//              dfault := inner[2]
//              subvar := inner[1]
//              fmt.Printf(">>%s<< >>%s<<  ))%s((\n",match,subvar,dfault);
//              sub, ok := varmap[subvar]
//              if ok {
//                  in = strings.Replace(in,match[0],sub,-1)
//              } else {
//                  if len(dfault) > 0 {
//                      in = strings.Replace(in,match[0],dfault,-1)
//                  } else {
//                      in = strings.Replace(in,match[0],"",-1)
//                  }
//              }
//          } else {
//              // no match? then it is not a valid expression, just leave alone
//              //out = in //string.replace(in,match,"",-1)
//          }
//      }

//  }
// //    else {
// //       out = in
// //   }
//  out = in
//  return
// }

// func main() {
//  fmt.Println("Hello, playground")

//  re_findEnvSubs = regexp.MustCompile(_re_findEnvSubs)
//  re_innerEnv = regexp.MustCompile(_re_innerEnv)

//  envvars := map[string]string{"PORT": "9001", "MOLY": "moly"}

//  var test1 = "https://cloud.com:${PORT|443}"

//  var test2 = "stuff"
//  var test3 = "holy${MOLY}"
//  var test4 = "https://cloud.com:${PORT2|1111}"
//  var test5 = "something{{}}${ELSE}!"

//  fmt.Printf("%s --> %s\n",test1,subInEnvVars(test1,envvars));
//  fmt.Printf("%s --> %s\n",test2,subInEnvVars(test2,envvars));
//  fmt.Printf("%s --> %s\n",test3,subInEnvVars(test3,envvars));
//  fmt.Printf("%s --> %s\n",test4,subInEnvVars(test4,envvars));
//  fmt.Printf("%s --> %s\n",test5,subInEnvVars(test5,envvars));

// }
