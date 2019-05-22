// Generated source file. 
// Edit files in 'src' folder    
package maestro

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
	"github.com/edhemphill/relaymq"
	relaymqClient "github.com/edhemphill/relaymq/client"
	"github.com/armPelionEdge/maestro/dcsAPI"
	"github.com/armPelionEdge/hashmap"  // thread-safe, fast hashmaps
	"encoding/json"
	"github.com/armPelionEdge/maestro/debugging"
	. "github.com/armPelionEdge/maestro/defaults"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/tasks"
	"github.com/armPelionEdge/maestro/processes"	
	"github.com/armPelionEdge/maestroSpecs"
	"github.com/armPelionEdge/maestro/maestroConfig"	
	"time"
	"unsafe"
	"strconv"
)

const RELAYMQ_DEFAULT_QUEUE_NAME = "upgrades"
const RELAYMQ_DEFAULT_PORT = 443

const (
	stop_driver = 0
)

type relayMQDriver struct {
	client *relaymqClient.RelayMQClient
	clientConfig *relaymqClient.RelayMQClientConfig
	config *maestroConfig.RelayMQDriverConfig
	listenerRunning bool
	controlChan chan int
}


func NewRelayMQDriver(config *maestroConfig.RelayMQDriverConfig) (driver *relayMQDriver, err error) {

	clientConfig := new(relaymqClient.RelayMQClientConfig)
	if(len(config.RootCA)>0) {
		clientConfig.RootCA = []byte(config.RootCA)
	}
	if(len(config.ClientCertificate)>0) {
		clientConfig.ClientCertificate = []byte(config.ClientCertificate)
	}
	if(len(config.ClientKey)>0) {
		clientConfig.ClientKey = []byte(config.ClientKey)
	}
	clientConfig.ServerName = config.ServerName
	if config.NoValidate == "true" {
		clientConfig.NoValidate = true
	}
	clientConfig.Host = config.Host
	if len(config.Port) > 0 {
		port, err2 := strconv.Atoi(config.Port)
		if err2 != nil {
			log.MaestroErrorf("RelayMQDriver>> Invalid port number: \"%s\". Using default.", config.Port)		
			clientConfig.Port = RELAYMQ_DEFAULT_PORT
		} else {
			clientConfig.Port = port
			if(port == 0) {
				clientConfig.Port = RELAYMQ_DEFAULT_PORT
			}
		}		
	} else {
		clientConfig.Port = RELAYMQ_DEFAULT_PORT		
	}
	if config.EnableLogging == "true" {
		clientConfig.EnableLogging = true
	}	
	clientConfig.PrefetchLimit = config.PrefetchLimit
	if(len(config.QueueName)>0) {
		clientConfig.QueueName = config.QueueName		
	} else {
		clientConfig.QueueName = RELAYMQ_DEFAULT_QUEUE_NAME
	}

	driver = new(relayMQDriver)
	driver.clientConfig = clientConfig
	driver.config = config
	driver.controlChan = make(chan int)
	driver.client, err = relaymqClient.New(*driver.clientConfig)

	return
}

func (this *relayMQDriver) Connect() (err error) {
	err = this.client.Connect()
	if err == nil {
		if !this.listenerRunning {
			go this.listener()			
		}
	}
	return
}

func (this *relayMQDriver) Disconnect() {
	this.client.Disconnect()
	this.controlChan <- stop_driver
}

// messages should be kept in these while they are being processed
// and after they are succesfully processed
// but not if they fail
var messagesProcessed *hashmap.HashMap  // by Message.ID to *heldMessage
var messagesProcessedBySignature *hashmap.HashMap

type heldMessage struct {
	msg *relaymq.Message
	decodedIn *dcsAPI.GatewayConfigMessage
	recieved int64  // time as Unix epoch in nanoseconds
	signature string 
	client *relaymqClient.RelayMQClient
}

func (this *relayMQDriver) newHeldMessage() (ret *heldMessage) {
	ret = new(heldMessage)
	ret.client = this.client
	return
}

func init() {
	messagesProcessed = hashmap.New(10)
	messagesProcessedBySignature = hashmap.New(10)
}

type relayMQ_OpAckHandler struct {
	op string  // add_image is first, then start_job (when we use JobManager to handle task launch)
	task *tasks.MaestroTask  // the Task which will - hopefully - be ACKed
	mqMsg *heldMessage       // originating relayMQ message
}

func transposeDCSToMaestroJob(dcs *dcsAPI.GatewayConfigMessage, job *maestroSpecs.JobDefinitionPayload) {
	if len(dcs.Jobs)>0 {
		job.Job = dcs.Jobs[0].Job
		job.CompositeId = dcs.Jobs[0].CompositeId
	} else {
		job.Job = "MISSING_NAME"
	}
	if len(dcs.Configs) > 0 {
		appname := dcs.Configs[0].Name
		ret, ok := ImageManagerGetInstance().LookupImage(appname)
		if ok {
			job.ExecCmd = ret.location  // the command to start for now is just the directory of the deviceJS script
		}
	}
	if len(job.ExecCmd) < 1 {
		log.MaestroErrorf("RelayMQDriver>> Can't start process b/c don't know where image is.")
	}
	job.ContainerTemplate = "deviceJS_process"


	// FIXME FIXME deal with config, composite ID, etc.
	
}

func (this *relayMQ_OpAckHandler) SendFinishedAck(task *tasks.MaestroTask) (err error) {
	debugging.DEBUG_OUT("RELAYMQ>>>>>>>>>>>>>>>>>>> GOT SendFinishedAck() call -> %s\n",this.op)
	log.MaestroInfof("RelayMQDriver - submitted Task finished: %s",task.Id)

	job := new(maestroSpecs.JobDefinitionPayload)
	transposeDCSToMaestroJob(this.mqMsg.decodedIn,job)

	debugging.DEBUG_OUT("RELAYMQ>>>>>>>>>>>>>>>>>>>  transposed Job to register %+v\n",job)

	// Register the Job and start it

	err = processes.RegisterMutableJob(job)

	if err != nil {
		log.MaestroErrorf("RelayMQDriver --> Failed to register Job: %s\n",err.Error())
	} else {
		// start the Job
		err = processes.ValidateJobs()
		if err == nil {
			processes.RestartJob(job.GetJobName())			
		} else {
			log.MaestroErrorf("RelayMQDriver --> Problem validating jobs: %s\n",err.Error())			
		}
	}
	this.mqMsg.client.Ack(this.mqMsg.msg.ID)
	return
}

func (this *relayMQ_OpAckHandler) SendFailedAck(task *tasks.MaestroTask) (err error) {
	debugging.DEBUG_OUT("RELAYMQ>>>>>>>>>>>>>>>>>>> [FAIL] GOT SendFailedAck() call -> %s\n",this.op)
	log.MaestroErrorf("RelayMQDriver - submitted Task failed: %s",task.Id)
	return
}

// main thread for handling inbound RelayMQ messages.
// The main use case for these messages today are:
//  - Installing images
//  - Starting / Stopping Jobs
func (this *relayMQDriver) listener() {
	messageCount := 0

	messageChan := this.client.Messages()
	mainLoop:
	for {
		select {
		case control := <-this.controlChan:
			if control == stop_driver {
				debugging.DEBUG_OUT("RelayMQDriver.listener() got shutdown\n")
				break mainLoop
			}

		case message := <-messageChan:
			debugging.DEBUG_OUT("RELAYMQ>>>>>>>>>>>>>>>>> Got relayMQ message %+v\n",message)
			dcsUpdate := new(dcsAPI.GatewayConfigMessage)
			err := json.Unmarshal([]byte(message.Body),dcsUpdate)
			if err != nil {
				log.MaestroErrorf("Can't decode inbound RelayMQ message: %s\n",err.Error())
			} else {
				debugging.DEBUG_OUT("RELAYMQ>>>>>>>>>>>>>>>>> [unique seen: %d] decoded %+v\n",messageCount,dcsUpdate)
				_, ok := messagesProcessed.GetStringKey(message.ID)
				if ok {
					debugging.DEBUG_OUT("RELAYMQ>>>>>>>>>>>>>>>>> Message seen and processed already.\n")
				} else {
					messageCount++
					debugging.DEBUG_OUT("RELAYMQ>>>>>>>>>>>>>>>>> Handle message\n")

					if len(dcsUpdate.Images) > 0 && len(dcsUpdate.Configs) > 0 {
						sig := dcsUpdate.Images[0].ComputeSignature()
						_, ok := messagesProcessedBySignature.GetStringKey(sig)

						if ok {
							debugging.DEBUG_OUT("RELAYMQ>>>>>>>>>>>>>>>>> Duplicate message by signature\n")
							log.MaestroWarnf("RelayMQDriver: Saw duplicate message based on signature")
						} else {

							holder := this.newHeldMessage()
							holder.msg = &message
							holder.decodedIn = dcsUpdate
							holder.recieved = time.Now().UnixNano()
							messagesProcessed.Set(message.ID,unsafe.Pointer(holder))
							messagesProcessedBySignature.Set(sig,unsafe.Pointer(holder))

							// For now we will make a very unsafe assumption, that all
							// we need to do is install the App

							// TODO:
							// - See if a job exists with this name
							//   - Shutdown if it does and remove
							// 
							// - Install new image
							op := maestroSpecs.NewImageOpPayload()
							op.Op = "add"
							op.Image = &holder.decodedIn.Images[0]
							op.AppName = holder.decodedIn.Configs[0].Name

							debugging.DEBUG_OUT("RELAYMQ>>>>>>>>>>>>>>>>> formed payload - sending to TaskManager %+v\n",op)
							log.MaestroDebugf("RelayMQDriver: Have new recognized inbound payload - sending to TaskManager")
							task, err := tasks.CreateNewTask(op, TASK_SRC_RELAYMQ)
							if err == nil {
								handler := &relayMQ_OpAckHandler{op:"add_image",task:task,mqMsg:holder}
								err = tasks.EnqueTask(task,handler,false)
							}

							// 
							// - Start Job

						}
					} else {
						log.MaestroErrorf(" Error in relayMQDriver: badly formed inbound 'update' - had no valid image")
						// TODO - ack this message - and fail it
					}
				}

			}
		}
	}
	this.listenerRunning = false
}
