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
	"github.com/armPelionEdge/maestroSpecs" 
	"github.com/armPelionEdge/maestro/processes"
	"github.com/armPelionEdge/maestro/storage"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/tasks"
	"github.com/armPelionEdge/hashmap"  // thread-safe, fast hashmaps
//	"errors"
	"net/http"
	IFDEBUG("fmt")
)

type jobManagerInstance struct {
	// ticker *time.Ticker
	// controlChan chan uint32
//	aggregateChan chan *imageop
	jobops *hashmap.HashMap // taskId : imageop
}

var JobManagerInstance *jobManagerInstance

func init() {
	JobManagerInstance = new(jobManagerInstance)	
	JobManagerInstance.jobops = hashmap.New(10)
}

// TODO - jobManager needs to be a TaskHandler... and Jobs should be started like any other Task
// 
// Note: JobOperation in maestroSpecs
// 
func (this *jobManagerInstance) SubmitTask(task *tasks.MaestroTask) (errout error) {
	// if task.Op.GetType() == maestroSpecs.OP_TYPE_JOB {
	// 	requestedOp, ok := task.Op.(*maestroSpecs.JobOpPayload)
	// } else {
	 	DEBUG_OUT("IMAGE>>>JobManagerInstance.SubmitTask() not correct Op type\n")
	// 	err := new(maestroSpecs.APIError)
	// 	err.HttpStatusCode = 500
	// 	err.ErrorString = "op was not an ImageOperation"
	// 	err.Detail = "in SubmitTask()"		
	// 	log.MaestroErrorf("JobManagerInstance got missplaced task %+v\n",task)
	// 	errout = err
	// }
	return nil
}
func (this *jobManagerInstance) ValidateTask(task *tasks.MaestroTask) error {
	return nil
}


func ProcessJobOp(DB *storage.MaestroDBInstance, msg maestroSpecs.JobOperation) (joberr *maestroSpecs.APIError, taskid string) {
	switch msg.GetOp() {
	case maestroSpecs.OP_ADD: 
		job := msg.GetJobPayload()
		if job != nil {
			err := DB.UpsertJob( job )
			if err != nil {
				log.MaestroErrorf("Error on saving job in DB: %s\n", err.Error())
			}
			err = processes.RegisterJob(job)
			if err != nil {
				log.MaestroErrorf("Job [%s] failed to register job: %s\n",job.GetJobName(),err.Error())
			}

		} else {
			joberr = &maestroSpecs.APIError{http.StatusNotAcceptable,"Badly formed","no job data"}
			// w.WriteHeader(http.StatusNotAcceptable)
			// w.Write([]byte("{\"error\":\"Badly formed\", \"details\":\"no job data\"}"))								
		}

	case maestroSpecs.OP_REMOVE:
		params := msg.GetParams()
		if len(params) > 0 {
			// TODO: removal of a job requires shutdown
			// task will need to be queued, and task ID given
		} else {
			joberr = &maestroSpecs.APIError{http.StatusNotAcceptable,"Badly formed","no job name(s)"}
			// w.WriteHeader(http.StatusNotAcceptable)
			// w.Write([]byte("{\"error\":\"Badly formed\", \"details\":\"no job name(s)\"}"))												
		}
	case maestroSpecs.OP_UPDATE:
		job := msg.GetJobPayload()
		if job != nil {
			err := DB.UpsertJob( job )
			if err != nil {
				log.MaestroErrorf("Error on saving job in DB: %s\n", err.Error())
			}
			err = processes.RegisterJob(job)
			if err != nil {
				log.MaestroErrorf("Job [%s] failed to register job: %s\n",job.GetJobName(), err.Error())
			} 				
		} else {
			joberr = &maestroSpecs.APIError{http.StatusNotAcceptable,"Badly formed","no job data"}
			// w.WriteHeader(http.StatusNotAcceptable)
			// w.Write([]byte("{\"error\":\"Badly formed\", \"details\":\"no job data\"}"))				
		}
	default:
		joberr = &maestroSpecs.APIError{http.StatusNotAcceptable,"Badly formed","unknown command"}
		// w.WriteHeader(http.StatusNotAcceptable)
		// w.Write([]byte("{\"error\":\"Badly formed\", \"details\":\"unknown command\"}"))
	}
	return
}
