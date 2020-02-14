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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/armPelionEdge/greasego"
	"github.com/armPelionEdge/httprouter"
	"github.com/armPelionEdge/maestro/configMgr"
	"github.com/armPelionEdge/maestro/debugging"
	"github.com/armPelionEdge/maestro/defaults"
	. "github.com/armPelionEdge/maestro/defaults"
	"github.com/armPelionEdge/maestro/events"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/maestroConfig"
	"github.com/armPelionEdge/maestro/networking"
	"github.com/armPelionEdge/maestro/processes"
	"github.com/armPelionEdge/maestro/storage"
	"github.com/armPelionEdge/maestro/tasks"
	"github.com/armPelionEdge/maestroSpecs"
)

func AddProcessRoutes(router *httprouter.Router) {
	router.POST("/process", handleStartProcess)
	router.GET("/runningProcesses", handleGetRunningProcesses)
	router.GET("/events", handleGetLastEvents)

	router.GET("/processStarted", handleProcessUpAlert)
	router.POST("/processFailed", handleProcessFailedAlert)

	// generic operation to start a new Task
	router.POST("/image", handleImageOp)

	router.POST("/ops", handleBatchOps)

	router.POST("/job", handleJobOp) // start one or more jobs (and define jobs if not there)
	router.GET("/jobs", handleAllJobStatus)
	router.GET("/job/:jobname", handleJobStatus)
	router.DELETE("/job/:jobname", handleDeleteJob) // remove the Job from the job list, and also stops it
	router.DELETE("/job", handleDeleteJobs)         // remove the Job from the job list, and also stops it
	router.PUT("/job/:jobname", handleSignalJob)    // used to stop job or 'kill' job, or restart job

	router.POST("/containerTemplate", handleAddContainerTemplate) // add one or more container templates
	router.DELETE("/containerTemplate/:name", handleDelContainerTemplate)
	router.GET("/containerTemplate", handleGetAllContainerTemplates)
	router.GET("/containerTemplate/:name", handleGetContainerTemplate)
	router.PUT("/containerTemplate/:name", handleChangeContainerTemplate)

	router.POST("/statsConfig", handleSetStatsConfig)
	router.GET("/statsConfig", handleGetStatsConfig)

	router.GET("/jobConfig/:jobname/:confname", handleGetJobConfig)
	router.GET("/jobConfig/:jobname", handleGetAllConfigsForJob)
	router.GET("/jobConfig", handleGetAllJobConfigs)
	router.POST("/jobConfig/:jobname/:confname", handlePostJobConfig)
	router.DELETE("/jobConfig/:jobname/:confname", handleDeleteJobConfig)

	router.GET("/status/:taskid", handleGetTaskStatus)

	router.GET("/net/interfaces", handleGetNetworkInterfaces)
	router.PUT("/net/interfaces", handlePutNetworkInterfaces)
	router.GET("/net/events", handleSubscribeNetworkEvents)
	router.GET("/net/events/:subscription", handleGetLatestNetworkEvents)

	router.PUT("/log/filter", handlePutLogFilter)

	router.GET("/alive", handleAlive)

	router.POST("/reboot", handleReboot)
	router.POST("/shutdown", handleShutdown)
}

var startTime time.Time
var netEventTimeout int64
var longPollManager *events.DeferredResponseManager

var netEventsHandler httprouter.Handle

func init() {
	startTime = time.Now()
	// network event subscriptions will go stale in 45 seconds, if they are not polled
	netEventTimeout = int64(time.Second * 45)
	events.OnEventManagerReady(func() error {
		// see test cases in events/events_test.go TestResponseManagerXXXX
		longPollManager = events.NewDeferredResponseManager()
		longPollManager.Start()
		netEventsHandler = longPollManager.MakeSubscriptionHttprouterHandler("netEventsHandler", func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (err error, id string) {
			id = ps.ByName("subscription")
			return
			// timeout is 20 seconds --> In 20 seconds, if no events, the handler will return StatusNoContent
		}, func(w http.ResponseWriter, req *http.Request) {
			log.MaestroError("Error ocurred in longpoll handler for network events.")
		}, nil, 20*time.Second)

		return nil
	})
}

func handleAlive(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	buffer := bytes.NewBufferString("")
	uptime := time.Since(startTime)
	buffer.WriteString(fmt.Sprintf("{\"ok\":true, \"uptime\":%d}", uptime.Nanoseconds()))
	w.WriteHeader(http.StatusOK)
	w.Write(buffer.Bytes())
	defer r.Body.Close()
}

func handlePutLogFilter(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
		return
	}

	var filterConfig maestroSpecs.LogFilter
	err = json.Unmarshal(body, &filterConfig)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
		return
	}

	targId := greasego.GetTargetId(filterConfig.Target)
	if targId == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
		return
	}

	filter := greasego.NewGreaseLibFilter()
	greasego.AssignFromStruct(filter, filterConfig)
	filter.Target = targId
	greasego.SetFilterValue(filter, greasego.GREASE_LIB_SET_FILTER_TARGET, filter.Target)
	if len(filterConfig.Levels) > 0 {
		mask := maestroConfig.ConvertLevelStringToUint32Mask(filterConfig.Levels)
		greasego.SetFilterValue(filter, greasego.GREASE_LIB_SET_FILTER_MASK, mask)
	}
	if len(filterConfig.Tag) > 0 {
		tag := maestroConfig.ConvertTagStringToUint32(filterConfig.Tag)
		greasego.SetFilterValue(filter, greasego.GREASE_LIB_SET_FILTER_MASK, tag)
	}
	debugging.DEBUG_OUT("PUT Filter -----------> %+v\n", filter)
	modified := greasego.ModifyFilter(filter)
	if modified != 0 {
		debugging.DEBUG_OUT("Failed to modify filter: %d\n", modified)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error\":\"filter does not exist\"}"))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handlePutNetworkInterfaces(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	inst := networking.GetInstance()
	if inst != nil {
		body, err := ioutil.ReadAll(r.Body)
		if err == nil {
			err = inst.SetInterfacesAsJson(body)
			if err == nil {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("{\"error\":\"no network manager\"}"))
	}
	defer r.Body.Close()

}

func handleGetNetworkInterfaces(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	inst := networking.GetInstance()
	if inst != nil {
		bytes, err := inst.GetInterfacesAsJson(false, false)
		if err == nil {
			w.WriteHeader(http.StatusOK)
			w.Write(bytes)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("{\"error\":\"no network manager\"}"))
	}
	defer r.Body.Close()
}

func jsonEscape(s string) (ret string) {
	byts, err := json.Marshal(s)
	if err != nil {
		ret = fmt.Sprintf("encode error")
		log.MaestroErrorf("error encoding JSON: %s\n", err.Error())
	} else {
		ret = string(byts)
	}
	return
}

// handler := manager.MakeSubscriptionHttpHandler("test", func(w http.ResponseWriter, req *http.Request) (err error, id string) {
// 	parts := strings.Split(req.URL.Path, "/")
// 	id = parts[len(parts)-1]
// 	fmt.Sprintf("parse path: %+v  id=%s\n", parts, id)
// 	return
// }, func(w http.ResponseWriter, req *http.Request) {}, nil, 2500*time.Millisecond)

func handleSubscribeNetworkEvents(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	inst := networking.GetInstance()
	if inst != nil {
		//		id, err := networking.SubscribeToNetEvents(netEventTimeout)
		ok, id := networking.GetNeteventsID()
		if ok {
			sub, err := events.SubscribeToChannel(id, netEventTimeout)
			if err == nil {
				// let the channel shutdown on itmeout
				sub.ReleaseChannel()
				if longPollManager != nil {
					subid, _, err2 := longPollManager.AddEventSubscription(sub)
					if err2 == nil {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(fmt.Sprintf("{\"id\":\"%s\"}", subid)))
					} else {
						w.WriteHeader(http.StatusInternalServerError)
						w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", jsonEscape(err2.Error()))))
					}
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(fmt.Sprintf("{\"error\":\"not ready\"}")))
				}
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", jsonEscape(err.Error()))))
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("{\"error\":\"server not ready\"}")))
		}
		// add to longpoll manager somehow

		// bytes, err := inst.GetInterfacesAsJson(false, false)
		// if err == nil {
		// 	w.WriteHeader(http.StatusOK)
		// 	w.Write(bytes)
		// } else {
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
		// }
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("{\"error\":\"no network manager\"}"))
	}
	defer r.Body.Close()
}

func handleGetLatestNetworkEvents(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if netEventsHandler != nil {
		// call the long poll handler setup in init()
		netEventsHandler(w, r, ps)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("{\"error\":\"server-not-ready\"}")))
	}
	defer r.Body.Close()
}

func sendAPIErrorOrAlt(w http.ResponseWriter, err error, alt string, altcode int) {
	// see if it's a maestroSpec.APIError
	//	var apierror *maestroSpecs.APIError

	apierror, ok := err.(maestroSpecs.Error)
	if ok {
		w.WriteHeader(apierror.GetHttpStatusCode())
		w.Write([]byte("{\"error\":\"" + apierror.GetErrorString() + "\",\"details:\":\"" + apierror.GetDetail() + "\"}"))
	} else {
		// otherwise
		w.WriteHeader(altcode)
		if err != nil {
			w.Write([]byte("{\"error\":\"" + alt + "\",\"details:\":\"" + err.Error() + "\"}"))
		} else {
			w.Write([]byte("{\"error\":\"" + alt + "\"}"))
		}
	}
}

func handleBatchOps(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	decoder := json.NewDecoder(r.Body)
	var msg maestroSpecs.BatchOpPayload
	err := decoder.Decode(&msg)
	msgp := &msg

	if err == nil {
		ops, err := msgp.GetOps() // decodes each task from JSON
		if err == nil {
			debugging.DEBUG_OUT(" ----- Batch Task ---- Out:%+v\n", ops[0])

			taskz, err := tasks.CreateNewBatchTasks(ops, msgp.GetTaskId(), TASK_SRC_LOCAL)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("{\"error\":\"Failed to enqueue:" + err.Error() + "\"}"))
			} else {
				for _, t := range taskz {
					err2 := tasks.ValidateTask(t)
					if err2 != nil {
						sendAPIErrorOrAlt(w, err2, "Error in validating tasks", http.StatusBadRequest)
						return
					}
				}
				// ok, tasks are acceptable...

				debugging.DEBUG_OUT("    Tasks look OK. total: %d\n", len(taskz))
				if len(taskz) < 2 { // sanity check
					sendAPIErrorOrAlt(w, nil, "Internal error at handleBatchOps()", http.StatusBadRequest)
					return
				}

				tasks.EnqueTasks(taskz, false)

			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("{\"error\":\"Badly formed batch request\"}"))
			debugging.DEBUG_OUT("Error: %s\n", err.Error())
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error\":\"Badly formed\"}"))
	}
	defer r.Body.Close()

}

type imageOpAckHandler struct {
	task *tasks.MaestroTask
}

func (this *imageOpAckHandler) SendFinishedAck(task *tasks.MaestroTask) (err error) {

	return
}

func (this *imageOpAckHandler) SendFailedAck(task *tasks.MaestroTask) (err error) {

	return
}

func handleImageOp(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	decoder := json.NewDecoder(r.Body)
	var msg maestroSpecs.ImageOpPayload
	err := decoder.Decode(&msg)
	msgp := &msg

	if err == nil {

		debugging.DEBUG_OUT("GOT: %+v\n", msg)
		debugging.DEBUG_OUT("     Is messge type:%s\n", msgp.GetType())
		if msgp.GetType() == maestroSpecs.OP_TYPE_IMAGE {
			task, err := tasks.CreateNewTask(msgp, TASK_SRC_LOCAL)
			// task.Op = msgp
			// task.Src = TASK_SRC_LOCAL
			if err == nil {
				handler := &imageOpAckHandler{task: task}
				err = tasks.EnqueTask(task, handler, false)
			}

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("{\"error\":\"Failed to enqueue:" + err.Error() + "\"}"))
			} else {
				w.Write([]byte("{\"task_id\":\"" + task.Id + "\"}"))
			}
		} else {
			w.WriteHeader(http.StatusNotAcceptable)
			w.Write([]byte("{\"error\":\"Only accept image as .type for this operation\"}"))
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error\":\"Badly formed\"}"))
	}
	defer r.Body.Close()
}

// stubbed out handlers
func handleJobOp(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	decoder := json.NewDecoder(r.Body)
	var msg maestroSpecs.JobOpPayload
	err := decoder.Decode(&msg)

	DB := storage.GetStorage()
	if DB == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("{\"error\":\"internal error\", \"details\":\"storage driver failed\"}"))
	} else {
		if err != nil {
			debugging.DEBUG_OUT("malformed!\n")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("{\"error\":\"Badly formed\", \"details\":\"" + err.Error() + "\"}"))

			// send an error
		} else {
			err, taskid := ProcessJobOp(DB, &msg)

			if err != nil {
				w.WriteHeader(err.HttpStatusCode)
				w.Write([]byte("{\"error\":\"" + err.ErrorString + "\", \"details\":\"" + err.Detail + "\"}"))
			} else {
				// hand back a taskid for follow up
				w.Write([]byte("{\"task_id\":\"" + taskid + "\"}"))
			}
			// switch msg.Op {
			// case maestroSpecs.OP_ADD:
			// 	job := msg.GetJobPayload()
			// 	if job != nil {
			// 		err := DB.UpsertJob( job )
			// 		if err != nil {
			// 			log.Errorf("Error on saving job in DB: %s\n", err.Error())
			// 		}
			// 		err = processes.RegisterJob(job)
			// 		if err != nil {
			// 			log.Errorf("Job [%s] failed to register job: %s\n",job.GetJobName(),err.Error())
			// 		}

			// 	} else {
			// 		w.WriteHeader(http.StatusNotAcceptable)
			// 		w.Write([]byte("{\"error\":\"Badly formed\", \"details\":\"no job data\"}"))
			// 	}

			// case maestroSpecs.OP_REMOVE:
			// 	params := msg.GetParams()
			// 	if len(params) > 0 {
			// 		// TODO: removal of a job requires shutdown
			// 		// task will need to be queued, and task ID given
			// 	} else {
			// 		w.WriteHeader(http.StatusNotAcceptable)
			// 		w.Write([]byte("{\"error\":\"Badly formed\", \"details\":\"no job name(s)\"}"))
			// 	}
			// case maestroSpecs.OP_UPDATE:
			// 	job := msg.GetJobPayload()
			// 	if job != nil {
			// 		err := DB.UpsertJob( job )
			// 		if err != nil {
			// 			log.Errorf("Error on saving job in DB: %s\n", err.Error())
			// 		}
			// 		err = processes.RegisterJob(job)
			// 		if err != nil {
			// 			log.Errorf("Job [%s] failed to register job: %s\n",job.GetJobName(), err.Error())
			// 		}
			// 	} else {
			// 		w.WriteHeader(http.StatusNotAcceptable)
			// 		w.Write([]byte("{\"error\":\"Badly formed\", \"details\":\"no job data\"}"))
			// 	}
			// default:
			// 	w.WriteHeader(http.StatusNotAcceptable)
			// 	w.Write([]byte("{\"error\":\"Badly formed\", \"details\":\"unknown command\"}"))

			// }
		}
	}
	defer r.Body.Close()
}

func handleGetTaskStatus(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()

}

func handleAllJobStatus(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	buffer := bytes.NewBufferString("")
	dat, err := processes.GetJobStatus()
	if err == nil {
		out, err := json.Marshal(dat)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errorS, _ := json.Marshal(err.Error())
			buffer.WriteString("{\"error\":\"Can't encode response\":\",\"details\":\"")
			buffer.Write(errorS)
		} else {
			w.Write(out)
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		errorS, _ := json.Marshal(err.Error())
		buffer.WriteString("{\"error\":\"Can't get response\":\",\"details\":\"")
		buffer.Write(errorS)
	}
	w.Write(buffer.Bytes())
	defer r.Body.Close()
}

func handleJobStatus(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}
func handleDeleteJob(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}
func handleDeleteJobs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}
func handleSignalJob(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}
func handleAddContainerTemplate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}
func handleDelContainerTemplate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}
func handleGetAllContainerTemplates(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}
func handleGetContainerTemplate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}
func handleChangeContainerTemplate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}
func handleSetStatsConfig(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}
func handleGetStatsConfig(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}
func handleReboot(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}
func handleShutdown(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}

type configResponse struct {
	Status          string                          `json:"status,omitempty"`
	Error           string                          `json:"error,omitempty"`
	ErrorDetails    string                          `json:"error_details,omitempty"`
	Configs         []maestroSpecs.ConfigDefinition `json:"configs,omitempty"`
	ReplacedConfigs []maestroSpecs.ConfigDefinition `json:"replaced_configs,omitempty"`
}

// See http://gregtrowbridge.com/golang-json-serialization-with-interfaces/
// For overview on handling dynamic JSON encoded objects

func handleGetJobConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	buffer := bytes.NewBufferString("")
	resp := new(configResponse)
	conf, err := configMgr.JobConfigMgr().GetConfig(ps.ByName("confname"), ps.ByName("jobname"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp.Error = "Lookup Failure"
		resp.ErrorDetails = err.Error()
		//		buffer.Write(errorS)
	} else {
		if conf != nil {
			resp.Configs = append(resp.Configs, conf)
		}
	}

	jsonval, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorS, _ := json.Marshal(err.Error())
		buffer.WriteString("{\"error\":\"Can't encode response\":\",\"details\":\"")
		buffer.Write(errorS)
	} else {
		if conf == nil {
			w.WriteHeader(http.StatusNoContent)
		}
		buffer.Write(jsonval)
	}
	//	buffer.WriteString("}")
	w.Write(buffer.Bytes())
	defer r.Body.Close()
}

func handleDeleteJobConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	buffer := bytes.NewBufferString("")
	resp := new(configResponse)
	conf, err := configMgr.JobConfigMgr().GetConfig(ps.ByName("confname"), ps.ByName("jobname"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp.Error = "Lookup Failure"
		resp.ErrorDetails = err.Error()
		//		buffer.Write(errorS)
	} else {
		if conf != nil {
			resp.ReplacedConfigs = append(resp.ReplacedConfigs, conf)
		}
	}

	if conf != nil {
		configMgr.JobConfigMgr().DelConfig(ps.ByName("confname"), ps.ByName("jobname"))
	}

	jsonval, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorS, _ := json.Marshal(err.Error())
		buffer.WriteString("{\"error\":\"Can't encode response\":\",\"details\":\"")
		buffer.Write(errorS)
	} else {
		buffer.Write(jsonval)
	}
	//	buffer.WriteString("}")
	w.Write(buffer.Bytes())
	defer r.Body.Close()
}

func handleGetAllConfigsForJob(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	buffer := bytes.NewBufferString("")
	resp := new(configResponse)
	confs, err := configMgr.JobConfigMgr().GetConfigsByJob(ps.ByName("jobname"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp.Error = "Lookup Failure"
		resp.ErrorDetails = err.Error()
		//		buffer.Write(errorS)
	}
	if len(confs) > 0 {
		resp.Configs = confs
	}
	jsonval, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorS, _ := json.Marshal(err.Error())
		buffer.WriteString("{\"error\":\"Can't encode response\":\",\"details\":\"")
		buffer.Write(errorS)
	} else {
		if len(resp.Configs) < 1 {
			w.WriteHeader(http.StatusNoContent)
		} else {
			buffer.Write(jsonval)
		}
	}
	//	buffer.WriteString("}")
	w.Write(buffer.Bytes())
	defer r.Body.Close()
}

func handleGetAllJobConfigs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	buffer := bytes.NewBufferString("")
	resp := new(configResponse)
	confs, err := configMgr.JobConfigMgr().GetAllConfigs()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp.Error = "Lookup Failure"
		resp.ErrorDetails = err.Error()
		//		buffer.Write(errorS)
	}
	if len(confs) > 0 {
		resp.Configs = confs
	}
	jsonval, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorS, _ := json.Marshal(err.Error())
		buffer.WriteString("{\"error\":\"Can't encode response\":\",\"details\":\"")
		buffer.Write(errorS)
	} else {
		if len(resp.Configs) < 1 {
			w.WriteHeader(http.StatusNoContent)
		} else {
			buffer.Write(jsonval)
		}
	}
	//	buffer.WriteString("}")
	w.Write(buffer.Bytes())
	defer r.Body.Close()
}

func handlePostJobConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	buffer := bytes.NewBufferString("")
	decoder := json.NewDecoder(r.Body)
	var config maestroSpecs.ConfigDefinitionPayload
	err := decoder.Decode(&config)
	sentHeader := false
	var oldconf maestroSpecs.ConfigDefinition

	resp := new(configResponse)
	if err == nil {
		oldconf, err = configMgr.JobConfigMgr().GetConfig(ps.ByName("confname"), ps.ByName("jobname"))

		if err == nil {
			if oldconf != nil {
				resp.ReplacedConfigs = append(resp.ReplacedConfigs, oldconf)
			}

			debugging.DEBUG_OUT("GOT: %+v\n", config)

			if len(config.Name) < 1 {
				config.Name = ps.ByName("confname")
				log.MaestroWarnf("Poorly formed POST /jobConfig. Normalizing Name to %s\n", config.Name)
			}
			if len(config.Job) < 1 {
				config.Job = ps.ByName("jobname")
				log.MaestroWarnf("Poorly formed POST /jobConfig. Normalizing Job to %s\n", config.Job)
			}

			err = configMgr.JobConfigMgr().SetConfig(ps.ByName("confname"), ps.ByName("jobname"), &config)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				sentHeader = true
				resp.Error = "Invalid request"
				resp.ErrorDetails = err.Error()
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			sentHeader = true
			resp.Error = "Lookup Failure"
			resp.ErrorDetails = err.Error()
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		sentHeader = true
		resp.Error = "Bad Payload - could not parse JSON"
		resp.ErrorDetails = err.Error()
	}

	if err != nil {
		if !sentHeader {
			w.WriteHeader(http.StatusInternalServerError)
			sentHeader = true
		}
		errorS, _ := json.Marshal(err.Error())
		buffer.WriteString("{\"error\":\"Can't encode response\":\",\"details\":\"")
		buffer.Write(errorS)
		buffer.WriteString("\"}")
	}

	jsonval, err := json.Marshal(resp)
	if err != nil {
		if !sentHeader {
			w.WriteHeader(http.StatusInternalServerError)
			sentHeader = true
		}
		errorS, _ := json.Marshal(err.Error())
		buffer.WriteString("{\"error\":\"Can't encode response\":\",\"details\":\"")
		buffer.Write(errorS)
		buffer.WriteString("\"}")
	} else {
		if !sentHeader {
			if oldconf == nil {
				w.WriteHeader(http.StatusCreated)
			} else {
				w.WriteHeader(http.StatusAccepted)
			}
		}
		buffer.Write(jsonval)
	}
	w.Write(buffer.Bytes())
	defer r.Body.Close()
}

func init() {
	processes.InitProcessSubsystem(defaults.MAX_QUEUED_PROCESS_EVENTS)
}

// Route for: POST /startProcess
func handleStartProcess(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// var args [5]string
	// args[0] = "/home/ed/work/gostuff/bin/devicedb"
	// args[1] = "start"
	// args[2] = "-conf=/home/ed/work/gostuff/config.yaml"  //"test-re.cc";
	// args[3] = ""

	// processes.ExecFile(args[0],args[:],nil,nil)

	log.MaestroInfo("GOT START PROCESS")

	decoder := json.NewDecoder(r.Body)
	var msg Msg_StartProcess
	err := decoder.Decode(&msg)
	debugging.DEBUG_OUT("got POST /process: %+v\n", msg)
	if err != nil {
		debugging.DEBUG_OUT("malformed!\n")
		// send an error
	} else {
		if len(msg.Path) > 0 {
			debugging.DEBUG_OUT("ExecFile: %s\n", msg.Path)
			opts := processes.NewExecFileOpts("[none]", "")
			if msg.Daemonize {
				opts.SetNewSid()
			} else {
				if msg.Pgid < 1 {
					opts.SetNewPgid()
				} else {
					opts.SetUsePgid(msg.Pgid)
				}
			}
			env := msg.Environment[:]
			if msg.InheritEnv {
				env = append(env, os.Environ()...)
			}
			var errno int
			var pid int
			if len(msg.Arguments) > 0 {
				pid, errno = processes.ExecFile(msg.Path, msg.Arguments[:], env, opts)
			} else {
				pid, errno = processes.ExecFile(msg.Path, nil, env, opts)
			}

			if errno != 0 {
				w.WriteHeader(http.StatusNotAcceptable)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"exec failed\",\"errno\":%d}", errno)))
			} else {
				w.Write([]byte(fmt.Sprintf("{\"pid\":%d}", pid)))
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("{\"error\":\"Badly formed\"}"))
		}
	}
	defer r.Body.Close()

}

// Router for: GET /processStarted/:uuid
func handleProcessUpAlert(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}

// Route for: POST /processFailed/:uuid
func handleProcessFailedAlert(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}

// Router for: GET /getRunningProcesses
func handleGetRunningProcesses(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
}

// Router for: GET /lastEvents
func handleGetLastEvents(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	events := new(processes.EventList)
	events.Events = processes.GetLastLocalEvents(defaults.MAX_QUEUED_PROCESS_EVENTS)
	var output []byte
	var err error
	if events.Events != nil {
		output, err = json.Marshal(events)
	} else {
		output, err = json.Marshal(&processes.EventList{
			Events: nil})
	}

	if err != nil {
		fmt.Println("ERROR ON JSON MARSHAL: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("{\"error\":%+v}", err)))
	} else {
		w.Write(output)
	}
	defer r.Body.Close()
}
