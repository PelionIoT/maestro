/*
Copyright (c) 2020, Arm Limited and affiliates.
Copyright (c) 2023 Izuma Networks

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
package gcd

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"

	"github.com/PelionIoT/maestro/log"
	"github.com/PelionIoT/maestro/maestroConfig"
)

type gwResourceManagerRegisterArgs struct {
	Name string `json:"name"`
}

type addResouceArgs struct {
	Lwm2mObjects []lwm2mObject `json:"objects"`
}

type lwm2mObject struct {
	ObjectId        int              `json:"objectId"`
	ObjectInstances []objectInstance `json:"objectInstances"`
}

type objectInstance struct {
	ObjectInstanceId int        `json:"objectInstanceId"`
	Resources        []resource `json:"resources"`
}

type resource struct {
	ResourceId int    `json:"resourceId"`
	Operations int    `json:"operations"`
	Type       string `json:"type"`
	Value      string `json:"value"`
}

var grm_client *Client = nil            // Gateway Resource Manager Client
var gcd_config *maestroConfig.GcdConfig // Gateway Capability Discovery config

func Gcd_init(config *maestroConfig.GcdConfig) {

	if config == nil {
		log.MaestroWarnf("maestroGCD: No config Provided")
		return
	}
	if config.EdgeCoreSocketPath == "" {
		log.MaestroWarnf("maestroGCD: edge_core_socketpath not provided in config file. Using default socketpath '/tmp/edge.sock'")
		config.EdgeCoreSocketPath = "/tmp/edge.sock"
	}
	if config.GatewayResources == nil {
		log.MaestroErrorf("maestroGCD: No gateway resources provided")
		return
	}
	if config.ConfigObjectId <= 0 {
		log.MaestroErrorf("maestroGCD: lwm2m objectid not provided")
		return
	}
	var err error
	gcd_config = config
	log.MaestroInfof("maestroGCD: connecting to edge-core")

	grm_client, err = grm_connect(config.EdgeCoreSocketPath) //connect to edge-core
	if err == nil {
		defer grm_client.Close()
	} else {
		log.MaestroErrorf("maestroGCD: could not connect to edge-core")
		return
	}
	log.MaestroInfof("maestroGCD: successfully connected to edge-core\n")
	objectinstanceid := 0
	for _, gatewayResource := range config.GatewayResources {
		err = grm_add_resource(config.ConfigObjectId, objectinstanceid, 1, 3, "string", gatewayResource.Name)
		if err == nil {
			grm_write_resource(config.ConfigObjectId, objectinstanceid, 1, 3, "string", b64.StdEncoding.EncodeToString([]byte(gatewayResource.Name)))
		}
		enable := "0"
		if gatewayResource.Enable == true {
			enable = "1"
		}
		err = grm_add_resource(config.ConfigObjectId, objectinstanceid, 2, 3, "string", string(enable))
		if err == nil {
			grm_write_resource(config.ConfigObjectId, objectinstanceid, 2, 3, "string", b64.StdEncoding.EncodeToString([]byte(enable)))
		}
		err = grm_add_resource(config.ConfigObjectId, objectinstanceid, 3, 3, "string", "Config")
		if err == nil {
			b64_config, err := read_config_file(gatewayResource.ConfigFilePath)
			if err == nil {
				// Known issue - to be fixed in next release. An object of size more than 4096 is being
				// truncated by the gorilla/websocket library and thus edge-core closes the connection on
				// receiving the invalid data
				// Hack: To avoid protocol error, shrink the config resource value to a smaller size
				if len(*b64_config) > 3800 {
					*b64_config = (*b64_config)[:3800]
				}
				grm_write_resource(config.ConfigObjectId, objectinstanceid, 3, 3, "string", *b64_config)
			} else {
				log.MaestroErrorf("maestroGCD: could not read %s config file %s, err: %v", gatewayResource.Name, gatewayResource.ConfigFilePath, err.Error())
			}
		}
		objectinstanceid = objectinstanceid + 1
	}
	grm_update_resource_loop()
}

func grm_update_resource_loop() {
	var c clientRequest
	srvreqchannel, err := grm_client.RegisterRequestReceiver()
	if err == nil {
		for {
			select {
			case srvreq := <-srvreqchannel:
				json.Unmarshal(srvreq, &c)
				log.MaestroDebugf("maestroGCD: Got request from edge-core: %v", c)
				if c.Method == "write" {
					params := c.Params.(map[string]interface{})
					uri := params["uri"].(map[string]interface{})
					objectinstanceid := 0
					for _, gatewayResource := range gcd_config.GatewayResources {
						if uri != nil && uri["objectId"].(float64) == float64(gcd_config.ConfigObjectId) && uri["objectInstanceId"].(float64) == float64(objectinstanceid) && uri["resourceId"].(float64) == 3 {
							log.MaestroDebugf("maestroGCD: Writing %s config file", gatewayResource.Name)
							if gatewayResource.ConfigFilePath != "" && write_config_file(gatewayResource.ConfigFilePath, params["value"].(string)) == nil {
								okresult := json.RawMessage(`"ok"`)
								grm_client.Respond(c.ID, &okresult, nil)
							}
						}
						if uri != nil && uri["objectId"].(float64) == float64(gcd_config.ConfigObjectId) && uri["objectInstanceId"].(float64) == float64(objectinstanceid) && uri["resourceId"].(float64) == 2 {
							if params["value"].(string) == "1" {
								//enable the service
							} else {
								//disable the service
							}
						}
						objectinstanceid = objectinstanceid + 1
					}
				}
			}
		}
	} else {
		log.MaestroErrorf("maestroGCD: Could not register request receiver: %s", err.Error())
	}
}

func read_config_file(filepath string) (*string, error) {
	config, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	b64_config := b64.StdEncoding.EncodeToString(config)
	return &b64_config, nil
}

func write_config_file(filepath string, config string) error {
	file, err := os.Create(filepath)
	if err != nil {
		log.MaestroErrorf("maestroGCD: could not create config file: %v", err)
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

	log.MaestroDebugf("maestroGCD: gw_resource_manager_register response:%v ", res)
	if err != nil {
		log.MaestroErrorf("maestroGCD: Failed to connect to edge-core %s", err.Error())
		return nil, err
	} else {
		log.MaestroInfof("maestroGCD: Websocket connection established with edge-core")
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
								Type:       resource_type,
								Value:      resource_value,
							},
						},
					},
				},
			},
		},
	}

	if grm_client == nil {
		log.MaestroErrorf("maestroGCD: grm_client is nil")
		return errors.New("grm_client is nil")
	}

	err := grm_client.Call("add_resource", add_resource, &res)

	log.MaestroDebugf("maestroGCD: add_resource response:%v ", res)
	if err != nil {
		log.MaestroErrorf("maestroGCD: Failed to add resource %s", err.Error())
		return err
	} else {
		log.MaestroInfof("maestroGCD: Lwm2m resource %d/%d/%d added ", objectid, objectinstanceid, resourceid)
	}
	return nil

}

func grm_write_resource(objectid int, objectinstanceid int, resourceid int, operations_allowed int, resource_type string, resource_value string) error {

	var res string
	write_resource := addResouceArgs{
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
								Type:       resource_type,
								Value:      resource_value,
							},
						},
					},
				},
			},
		},
	}

	if grm_client == nil {
		log.MaestroErrorf("maestroGCD: grm_client is nil")
		return errors.New("grm_client is nil")
	}

	err := grm_client.Call("write_resource_value", write_resource, &res)

	log.MaestroDebugf("maestroGCD: write response:%v ", res)
	if err != nil {
		log.MaestroErrorf("maestroGCD: Failed to write resource %s", err.Error())
		return err
	} else {
		log.MaestroInfof("maestroGCD: Lwm2m resource %d/%d/%d added ", objectid, objectinstanceid, resourceid)
	}
	return nil

}
