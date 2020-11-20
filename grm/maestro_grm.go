/*
Copyright (c) 2020, Arm Limited and affiliates.
SPDX-License-Identifier: Apache-2.0
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package grm


import (
	
	"encoding/json"
	"os"
	"errors"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/maestroConfig"
	b64 "encoding/base64"
)

type gwResourceManagerRegisterArgs struct {
	Name string `json:"name"`
}

type addResouceArgs struct {
	Lwm2mObjects []lwm2mObject `json:"objects"`
}

type lwm2mObject struct {
	ObjectId int 					 `json:"objectId"`
	ObjectInstances []objectInstance `json:"objectInstances"`
}

type objectInstance struct {
	ObjectInstanceId int `json:"objectInstanceId"`
	Resources []resource `json:"resources"`
}

type resource struct {
	ResourceId int  `json:"resourceId"`
	Operations int  `json:"operations"`
	Type string `json:"type"`
	Value string    `json:"value"`
}

var grm_client *Client = nil
var fluentbitconfigfilepath string
var fluentbitconfigobjectid int


func Grm_init(config *maestroConfig.GrmConfig) {

	if(config == nil) {
		return
	}
	if (config.EdgeCoreSocketPath == "") {
		log.MaestroWarnf("maestroGRM: edge_core_socketpath not provided in config file. Using default socketpath '/tmp/edge.sock'")
		config.EdgeCoreSocketPath = "/tmp/edge.sock"
	}

	if(config.FluentbitConfigFilePath == "") {
		log.MaestroErrorf("maestroGRM: fluentbit_config_filepath not provided in config file")
		return
	}

	if(config.FluentbitConfigObjectId == 0) {
		log.MaestroErrorf("maestroGRM: fluentbit_config_lwm2mobjectid not provided in config file")
		return
	}

	var err error
	fluentbitconfigfilepath = config.FluentbitConfigFilePath
	fluentbitconfigobjectid = config.FluentbitConfigObjectId
	log.MaestroInfof("maestroGRM: connecting to edge-core")

	grm_client, err = grm_connect(config.EdgeCoreSocketPath)   //connect to edge-core
	if (err == nil) {
		defer grm_client.Close()
	} else {
		log.MaestroErrorf("maestroGRM: could not connect to edge-core")
		return	
	}
	log.MaestroInfof("maestroGRM: successfully connected to edge-core\n")

	err = grm_add_resource(fluentbitconfigobjectid, 0, 1, 3, "string", "#Fluentbit Config file")
	
	if(err!=nil) {
		log.MaestroErrorf("maestroGRM: could not add resource with ")
		return
	} else {
		grm_update_resource_loop()
	}

}

func grm_update_resource_loop() {
	var c clientRequest
	srvreqchannel, err := grm_client.RegisterRequestReceiver()
	if(err == nil) {
		for {
			select {
				case srvreq := <-srvreqchannel:
					json.Unmarshal(srvreq, &c)
					log.MaestroDebugf("maestroGRM: Got request from edge-core: %v", c)
					if (c.Method == "write") {
						params := c.Params.(map[string]interface{})
						uri := params["uri"].(map[string]interface{})
						if (uri != nil && uri["objectId"].(float64) == float64(fluentbitconfigobjectid) && uri["objectInstanceId"].(float64) == 0 && uri["resourceId"].(float64) == 1) {
							log.MaestroDebugf("maestroGRM: Writing fluentbit config file")
							if(write_config_file(fluentbitconfigfilepath, params["value"].(string))  == nil) {
								okresult := json.RawMessage(`"ok"`)
								grm_client.Respond(c.ID, &okresult, nil)
							}
						}
					}	
			}
		}
	} else {
		log.MaestroErrorf("maestroGRM: Could not register request receiver: %s", err.Error())
	}
}

func write_config_file( filepath string, config string) error{
	file, err := os.Create(filepath)
	if err != nil {
		log.MaestroErrorf("maestroGRM: could not create config file: %v", err)
		return err
	} else {
		b64_config, _ := b64.StdEncoding.DecodeString(config)
		file.WriteString(string(b64_config[:]))
	}
	file.Close()
	return nil
}

func grm_connect(edgecorescoketpath string) (*Client, error) {
	
	client := Dial(edgecorescoketpath, "/1/grm", nil)

	var res string
	err := client.Call("gw_resource_manager_register", gwResourceManagerRegisterArgs{Name: "maestro_grm"}, &res)

	log.MaestroDebugf("maestroGRM: gw_resource_manager_register response:%v ",res)
	if (err != nil) {
		log.MaestroErrorf("maestroGRM: Failed to connect to edge-core %s", err.Error()) 
		return nil, err
	} else {
		log.MaestroInfof("maestroGRM: Websocket connection established with edge-core")
	}

	return client, nil
}

func grm_add_resource(objectid int, objectinstanceid int, resourceid int, operations_allowed int, resource_type string, resource_value string) error {

	var res string
	add_resource := addResouceArgs{
		Lwm2mObjects: []lwm2mObject{
			lwm2mObject{
				ObjectId: objectid,
				ObjectInstances: []objectInstance{
					objectInstance{
						ObjectInstanceId: objectinstanceid,
						Resources: []resource{
							resource{
								ResourceId: resourceid,
								Operations: operations_allowed,
								Type: resource_type,
								Value: resource_value,
							},
						},
					},
				},
			},
		},
	}

	if(grm_client == nil) {
		log.MaestroErrorf("maestroGRM: grm_client is nil")
		return errors.New("grm_client is nil")
	}

	err := grm_client.Call("add_resource", add_resource, &res)

	log.MaestroDebugf("maestroGRM: add_resource response:%v ",res)
	if (err != nil) {
		log.MaestroErrorf("maestroGRM: Failed to add resource %s", err.Error()) 
		return err
	} else {
		log.MaestroInfof("maestroGRM: Lwm2m resource %d/%d/%d added ",objectid, objectinstanceid, resourceid)
	}
	return nil

}