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
    "testing"
    "log"
    "strings"
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
		if(strings.Compare(config_loader.ClientId, "WWRL000001") != 0) {
			fmt.Printf("clientId from config = %+v expected:WWRL000001\n", config_loader.ClientId)
			t.FailNow()
		} else {
			fmt.Printf("clientId from config = %+v\n", config_loader.ClientId)
		}
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

