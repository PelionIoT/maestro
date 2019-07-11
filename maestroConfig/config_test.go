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
	fmt.Printf("\nConfigChangeHook:SawChange: %s:%s old:%s new:%s\n", configgroup, fieldchanged, curvalue, futvalue)
	return true;
}

// ChangesComplete is called when all changes for a specific configgroup tagname
// If ChangesComplete returns true, then all changes in that group will be assigned to the current struct
func (cfgHook ConfigChangeHook) ChangesComplete(configgroup string) (acceptallchanges bool) {
	fmt.Printf("\nConfigChangeHook:ChangesComplete: %s\n", configgroup)
	return true;
}


func TestConfigMonitorSimple(t *testing.T) {
	var devicedbUri string = "https://WWRL000000:9090" //The URI of the relay's local DeviceDB instance
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
	
	var config MyConfig
	config.Property1 = 100
	config.Property2 = "asdf"
	config.Property3 = false
	fmt.Printf("\nPutting config: %v\n", config)
	err = configClient.Config(configName).Put(&config)
	if err != nil {
		fmt.Printf("\nUnable to put config: %v\n", err)
		t.FailNow()
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
