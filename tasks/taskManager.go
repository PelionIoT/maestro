package tasks

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
	"encoding/gob"
	"errors"
	"sync"
	"time"

	"github.com/armPelionEdge/maestro/debugging"
	"github.com/armPelionEdge/maestro/defaults"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/storage"
	"github.com/armPelionEdge/maestroSpecs"
	"github.com/satori/go.uuid"
	"github.com/armPelionEdge/stow"
	"github.com/boltdb/bolt"
)

const TASK_COMPLETE_STEP = ^uint32(0)
const MAX_TASK_LIFE = maestroSpecs.MAX_TASK_LIFE

const RUNAWAY_THRESHOLD = 20

type MaestroTask struct {
	Id   string
	Src  string                 // identifies where the task came from
	Op   maestroSpecs.Operation // a generic Operation
	Step uint32                 // A step of 0 means the task has never been started
	// Steps after this are proprietary to the Task at hand
	TotalSteps   uint32 // the last step in the Task (so '2' if there are 2 steps, 0 is the first)
	StepProgress uint32 // 0-100 % of Step complete
	StepName     string
	enqueTime    int64 // seconds since Unix Epoch
	CompleteTime int64
	persistent   bool
	batchTask    bool // true if this is the parent Task of a batch operation / set of batch tasks
	// and this task has no parent
	batchOrder uint32 // if len(parentTaskId) > 0 then this is order of tasks. 0 based number
	//	waitingOnIds map[string]bool  // if this Task is waiting on other task, here are their IDs
	// when a subtask is complete, it should be removed from here
	childrenOrder []string               // an ordered list of the IDs of the children.
	parentTaskId  string                 // if len > 0, then this is a subtask
	Error         *maestroSpecs.APIError // if the Task has failed, this is the associate error
	// A Task is considered failed if 'failed' is true
	failFast bool // if there is an error in any subtask, fail everything
}

const c_TASK_QUEUE = "tasks"

// initialized in init()
type taskManagerInstance struct {
	db        *bolt.DB
	taskQueue *stow.Store
}

var instance *taskManagerInstance

// impl storage.StorageUser interface
func (this *taskManagerInstance) StorageInit(instance storage.MaestroDBStorageInterface) {
	gob.Register(&MaestroTask{})
	this.taskQueue = stow.NewStore(instance.GetDb(), []byte(c_TASK_QUEUE))
}

func (this *taskManagerInstance) StorageReady(instance storage.MaestroDBStorageInterface) {
	this.db = instance.GetDb()
}

func (this *taskManagerInstance) StorageClosed(instance storage.MaestroDBStorageInterface) {
}

const (
	op_next_task     = iota
	op_next_finished = iota
	op_shutdown      = iota
)

// An AckHandler is another module which can Ack a particular Task
// When you EnqueTask you must provide a AckHandler for the task
type AckHandler interface {
	SendFinishedAck(task *MaestroTask) (err error)
	SendFailedAck(task *MaestroTask) (err error)
}

// A TaskHandler is another module which can handle a certain type of Task
// Examples are: imageManager.go and jobManager.go
type TaskHandler interface {
	// Submits a Task into the Handler. It's work is b
	SubmitTask(task *MaestroTask) error
	ValidateTask(task *MaestroTask) error
}

type controlToken struct {
	op   uint32
	task *string // if the op was driven by a new task, this is it's ID
}

func newControlTokenNewTask(id *string) (ret *controlToken) {
	ret = new(controlToken)
	ret.op = op_next_task
	if id != nil {
		ret.task = new(string)
		*ret.task = *id
	}
	return
}

// a token to get to the next task, or next step of the current task
func newControlTokenNextTask(id *string) (ret *controlToken) {
	ret = new(controlToken)
	ret.op = op_next_task
	if id != nil {
		ret.task = new(string)
		*ret.task = *id
	}
	return
}

func newControlTokenFinishedTask(id *string) (ret *controlToken) {
	ret = new(controlToken)
	ret.op = op_next_finished
	if id != nil {
		ret.task = new(string)
		*ret.task = *id
	}
	return
}

var controlChan chan *controlToken
var internalTicker *time.Ticker

type metaTaskData struct {
	failed      bool
	running     bool // if the Task has started. It could be between steps here.
	runningStep uint32
	// used to check to see if the task has continue to repeat a Step too many time
	// if Step does not change, then this number is incremented. If it gets past
	// RUNAWAY_THRESHOLD, the Task is considered hung, and is killed.
	runawayCheck uint32
	// only true if a Step is in progress. Only needs to be set if the Task defers the step,
	// starts a new thread / go routine which is handling the step
	// Marked with MarkTaskAsExecuting()
	stepExecuting bool
	ackSent       bool
	failSent      bool
	successSent   bool
	ackHandler    AckHandler
	mutex         sync.Mutex
}

func doesMetaExist(id string) (ret bool) {
	_, ret = metaTasks.Load(id)
	return
}

// returns metaData with Lock!!
func getMetaTaskData(id string) (ret *metaTaskData) {
	pmeta, ok := metaTasks.Load(id)
	if ok {
		ret = pmeta.(*metaTaskData)
		ret.mutex.Lock()
	} else {
		ret = new(metaTaskData)
		ret.mutex.Lock()
		metaTasks.Store(id, ret)
	}
	return
}
func (this *metaTaskData) release() {
	this.mutex.Unlock()
}
func (this *metaTaskData) reset() {
	this.running = false
	this.ackSent = false
	this.failSent = false
	this.successSent = false
}

// Must Lock!
func markAsRunning(id string) {
	meta := getMetaTaskData(id)
	meta.running = true
	meta.release()
}

// Must Lock!
func unmarkAsRunning(id string) {
	meta := getMetaTaskData(id)
	meta.running = false
	meta.release()
}

// must lock
func markAsFailed(id string) {
	meta := getMetaTaskData(id)
	meta.running = false
	meta.failed = true
	meta.release()
}

// Must Lock!
func isRunning(id string) (ret bool) {
	meta := getMetaTaskData(id)
	ret = meta.running
	meta.release()
	return
}

func getInstance() *taskManagerInstance {
	if instance == nil {
		instance = new(taskManagerInstance)
		storage.RegisterStorageUser(instance)
		internalTicker = time.NewTicker(time.Second * time.Duration(defaults.TASK_MANAGER_CLEAR_INTERVAL))
		controlChan = make(chan *controlToken, 100) // buffer up to 100 events
	}
	return instance
}

func InitTaskManager() {
	getInstance()
	// needed for storage of MaestroTasks to work
}

///////////////////////
/// Storage of Tasks
///////////////////////

func (this *taskManagerInstance) upsertTask(obj *MaestroTask) error {
	// key := obj.GetJobName()
	if this.db != nil {
		this.taskQueue.Put(obj.Id, obj)
		return nil
	} else {
		return errors.New("no database")
	}
}

type UpdateTaskCB func(task *MaestroTask)
type IfTaskCB func(task *MaestroTask) bool

func (this *taskManagerInstance) updateTaskById(id string, cb UpdateTaskCB) (err error, updated *MaestroTask) {
	if this.db != nil {
		temp := new(MaestroTask)
		err = this.taskQueue.Update(id, temp, func(val interface{}) {
			if val != nil {
				task, ok := val.(*MaestroTask)
				if ok {
					cb(task)
				} else {
					// should be impossible
				}
			} else {
				// should be impossible, if 'id' did not exist
				// then callback will not be called
			}
		})
		updated = temp
		return
	} else {
		err = errors.New("no database")
		return
	}
}

func (this *taskManagerInstance) getTaskById(id string) (obj *MaestroTask, err error) {
	if this.db != nil {
		this.taskQueue.Get(id, &obj)
	} else {
		err = errors.New("no database")
	}
	return
}

func (this *taskManagerInstance) removeTaskById(id string) error {
	if this.db != nil {
		this.taskQueue.Delete(id)
		return nil
	} else {
		return errors.New("no database")
	}
}

func (this *taskManagerInstance) forEachIfTask(cb IfTaskCB) error {
	var temp MaestroTask
	return this.taskQueue.IterateIf(func(key []byte, val interface{}) bool {
		ty, ok := val.(*MaestroTask)
		if ok {
			return cb(ty) // true, then we should keep iterating
		} else {
			log.MaestroWarn("Possible DB corruption - not castable to MaestroTask")
			return false
		}
	}, &temp)
}

// If the callback returns true, then the MaestroTask will be removed
// from the database
func (this *taskManagerInstance) deleteIfTask(cb IfTaskCB) error {
	var temp MaestroTask
	return this.taskQueue.DeleteIf(func(key []byte, val interface{}) bool {
		ty, ok := val.(*MaestroTask)
		if ok {
			return cb(ty) // true, then it should be deleted
		} else {
			log.MaestroWarn("Possible DB corruption - not castable to MaestroTask")
			return false
		}
	}, &temp)
}

////////////////////////////////////////////////////////
/// End Storage ////////////////////////////////////////
////////////////////////////////////////////////////////

// Ensures the Task can be run by the TaskManager
// No error is good
func ValidateTask(task *MaestroTask) (err error) {
	if len(task.Id) > 0 {
		// batch tasks don't have an Op
		if !task.batchTask {
			// check to see if there is a handler for the Op type:
			_handler, ok := handlers.Load(task.Op.GetType())
			if !ok {
				apierror := new(maestroSpecs.APIError)
				apierror.ErrorString = "no task handler"
				apierror.Detail = "Unknown task type:" + task.Op.GetType()
				apierror.HttpStatusCode = 406 // not acceptable
				err = apierror
				return
			} else {
				handler := _handler.(*TaskHandler)
				err = (*handler).ValidateTask(task)
				if err != nil {
					return
				}
			}
		}
		// other checks

	} else {
		apierr := new(maestroSpecs.APIError)
		apierr.HttpStatusCode = 500
		apierr.ErrorString = "invalid task ID"
		apierr.Detail = "in ValidateTask()"
		err = apierr
	}
	return
}

//func CreateNewBatchTask(childtasks []*MaestroTask, masterid string, src string) (ret *MaestroTask, err error) {
func CreateNewBatchTasks(ops []maestroSpecs.Operation, masterid string, src string) (ret []*MaestroTask, err error) {
	ret = make([]*MaestroTask, 0, len(ops)+1)
	var master *MaestroTask
	master, err = internalCreateNewTask(masterid, src)
	if err == nil {
		master.batchTask = true
		ret = append(ret, master)
		for _, op := range ops {
			if op != nil {
				var task *MaestroTask
				task, err = CreateNewTask(op, src)
				if err == nil {
					//			master.waitingOnIds[task.Id] = true
					master.childrenOrder = append(master.childrenOrder, task.Id)
					task.parentTaskId = master.Id
					ret = append(ret, task)
				} else {
					// return the error
					return
				}
			} else {
				debugging.DEBUG_OUT("Corrupted Task passed to CreateNewBatchTask() - had nil\n")
			}
		}

		// for _, t := range childtasks {
		// 	if t != nil && len(t.Id) > 0 {
		// 		ret.waitingOnIds[t.Id] = true
		// 		t.parentTaskId = ret.Id
		// 	} else {
		// 		debugging.DEBUG_OUT("Corrupted Task passed to CreateNewBatchTask() - had no Id\n")
		// 	}
		// }
	}
	return
}

// Enqueu's the task into disk storage
func EnqueTask(task *MaestroTask, ackHandler AckHandler, persistent bool) (err error) {
	if len(task.Id) > 0 && !doesMetaExist(task.Id) {
		task.enqueTime = time.Now().Unix()
		meta := getMetaTaskData(task.Id)
		meta.ackHandler = ackHandler
		meta.release()

		if persistent {
			task.persistent = true
			err = getInstance().upsertTask(task)
		}
		tasks.Store(task.Id, task)
		if err == nil {
			controlChan <- newControlTokenNewTask(&task.Id)
		}
	} else {
		err := new(maestroSpecs.APIError)
		err.HttpStatusCode = 500
		err.ErrorString = "invalid task ID"
		err.Detail = "in EnqueTask()"
	}
	return
}

// enques a Batch of Tasks.
func EnqueTasks(thesetasks []*MaestroTask, persistent bool) (err error) {
	now := time.Now().Unix()

	for _, task := range thesetasks {
		unmarkAsRunning(task.Id)
		if persistent {
			task.persistent = true
			task.enqueTime = now
			err = getInstance().upsertTask(task)
		}
		tasks.Store(task.Id, task)
		if err != nil {
			return
		}
	}
	controlChan <- newControlTokenNewTask(nil)
	return
}

func internalCreateNewTask(id string, src string) (ret *MaestroTask, err error) {
	ret = new(MaestroTask)
	if len(id) > 0 {
		ret.Id = id
	} else {
		var id uuid.UUID
		id, err = uuid.NewV4() // use a random number UUID
		if err == nil {
			ret.Id = id.String()
		}
	}
	ret.Src = src
	//	ret.waitingOnIds = make(map[string]bool)
	ret.childrenOrder = make([]string, 0, 10)
	return
}

func CreateNewTask(op maestroSpecs.Operation, src string) (ret *MaestroTask, err error) {
	ret = new(MaestroTask)
	id := op.GetTaskId()
	if len(id) > 0 {
		ret.Id = id
	} else {
		var id uuid.UUID
		id, err = uuid.NewV4() // use a random number UUID
		if err == nil {
			ret.Id = id.String()
		}
	}
	ret.Src = src
	ret.Op = op
	//	ret.waitingOnIds = make(map[string]bool)
	ret.childrenOrder = make([]string, 0, 10)
	return
}

// func ReturnTask(id string, cb UpdateTaskCB) (err error) {
// 	err, _ = UpdateTaskById(id, func(task *MaestroTask){
// 		if cb != nil {
// 			cb(task)
// 		}
// 		task.running = false
// 	})
// 	return
// }

// marks the current Step as Executing
// Used if a Task starts a new goroutine, or defers an operation
// This is automatically cleared by IterateTask
func MarkTaskStepAsExecuting(id string) {
	debugging.DEBUG_OUT(" TASK>>>>>> MarkTaskStepAsExecuting(\"%s\")\n", id)
	meta := getMetaTaskData(id)
	meta.stepExecuting = true
	meta.release()
}

// When a TaskHandler has finished a step of a task,
// the handler should call this. This will execute a call back
// which makes changes to the MaestroTask and automatically iterates
// the Step field
func IterateTask(id string, cb UpdateTaskCB) (err error) {
	debugging.DEBUG_OUT(" TASK>>>>>> IterateTask(\"%s\")\n", id)
	ptask, ok := tasks.Load(id)
	if instance == nil || instance.db == nil {
		err = errors.New("task db not ready")
		return
	}
	if ok {
		task := ptask.(*MaestroTask)
		meta := getMetaTaskData(id)
		meta.stepExecuting = false
		meta.runawayCheck = 0
		meta.release()
		if task.persistent {
			err, newtask := instance.updateTaskById(id, func(t *MaestroTask) {
				if cb != nil {
					cb(t)
				}
				t.Step++
			})
			if err == nil {
				// update in memory map
				tasks.Store(task.Id, newtask)
			}
		} else {
			if cb != nil {
				cb(task)
			}
			task.Step++
		}
		controlChan <- newControlTokenNextTask(&task.Id) // wake up thread
		return
	} else {
		err = errors.New("no task")
		return
	}
}

func CompleteTask(id string) (err error) {
	ptask, ok := tasks.Load(id)
	if ok {
		task := ptask.(*MaestroTask)
		if task.persistent {
			err, newtask := getInstance().updateTaskById(id, func(t *MaestroTask) {
				unmarkAsRunning(t.Id)
				t.Step = TASK_COMPLETE_STEP
				t.CompleteTime = time.Now().Unix()
			})
			if err == nil {
				// update in memory map
				tasks.Store(task.Id, newtask)
			}
		} else {
			unmarkAsRunning(task.Id)
			task.Step = TASK_COMPLETE_STEP
			task.CompleteTime = time.Now().Unix()
		}
		controlChan <- newControlTokenFinishedTask(&task.Id) // wake up thread
		return
	} else {
		err = errors.New("no task")
		return
	}
}

func FailTask(id string, cause *maestroSpecs.APIError) (err error) {
	ptask, ok := tasks.Load(id)
	if ok {
		task := ptask.(*MaestroTask)
		if task.persistent {
			err, newtask := instance.updateTaskById(id, func(t *MaestroTask) {
				//				t.Step = TASK_COMPLETE_STEP
				markAsFailed(t.Id)
				t.Error = cause
				t.CompleteTime = time.Now().Unix()
			})
			if err == nil {
				// update in memory map
				tasks.Store(task.Id, newtask)
			}
		} else {
			markAsFailed(task.Id)
			task.Error = cause
			task.CompleteTime = time.Now().Unix()
		}
		debugging.DEBUG_OUT(" FAILED TASK >>>>>>  %s\n", id)

		// ok - sucks - but let's see if we have a failed Ack handler
		meta := getMetaTaskData(task.Id)
		handler := meta.ackHandler
		meta.release()
		if handler != nil {
			go runFailedAck(task, handler)
		}

		controlChan <- newControlTokenNewTask(nil) // wake up thread
		return
	} else {
		err = errors.New("no task")
		return
	}
}

func GetTaskStatus(id string) (step uint32, ok bool) {
	ptask, ok2 := tasks.Load(id)
	if ok2 {
		task := ptask.(*MaestroTask)
		ok = true
		step = task.Step
	} else {
		// try storage, might an old task which was in db, but not in memory
		task, err := instance.getTaskById(id)
		if err == nil {
			ok = true
			step = task.Step
		}
	}
	return
}

// Informs the TaskManager that a request ACK for specific Task
// failed to send or be recieved by the original caller of the Task
// This is typically a remote server.
// In this case, the TaskManager will again ask for an ACK at some point
// in the future
func TaskAckFailed(taskId string, err error) {

}

// Informs the TaskManager that a request ACK for specific Task
// was successful sent and recieved by the original caller of the Task
// This is typically a remote server
func TaskAckOK(taskid string) {

}

func RegisterHandler(optype string, handler TaskHandler) {
	handlers.Store(optype, &handler)
}

func RemoveTask(id string) error {
	tasks.Delete(id)
	return getInstance().removeTaskById(id)
}

// LoadFromStorage
// Loads all tasks which were queued into storage
// to be called on startup. This
func RestartStoredTasks() {
	// first pull out anything which is really old
	instance := getInstance()
	cleanOutStaleTasks()

	instance.forEachIfTask(func(task *MaestroTask) bool {
		newtask := new(MaestroTask)
		*newtask = *task
		// add to our in-memory hashmap
		tasks.Store(newtask.Id, newtask)
		return true // iterate through all tasks in DB
	})
	controlChan <- newControlTokenNewTask(nil) // wake up thread
}

var taskManagerRunning = false

func StartTaskManager() {
	if !taskManagerRunning {
		taskManagerRunning = true
		go taskRunner()
	}
}

func cleanOutStaleTasks() {
	debugging.DEBUG_OUT("TASK>>>>>>>>>  cleanOutStaleTasks()\n")
	oldest := time.Now().Unix() - MAX_TASK_LIFE
	getInstance().deleteIfTask(func(task *MaestroTask) bool {
		if task.enqueTime < oldest {
			return true
		} else {
			return false
		}
	})
	var removeids []string

	tasks.Range(func(key, value interface{}) bool {
		if value == nil {
			debugging.DEBUG_OUT("CORRUPTION in hashmap - have null value pointer\n")
			return true
		}
		if value != nil {
			taskitem := value.(*MaestroTask)
			if taskitem.enqueTime < oldest {
				removeids = append(removeids, taskitem.Id)
			}
		}
		return true
	})
	// remove stale Tasks
	for id := range removeids {
		debugging.DEBUG_OUT("TASK>>>>>>>>>  removing Task %s\n", id)
		tasks.Delete(id)
	}

}

func newAPIErrorNoOpTypeHandler(s string) (err *maestroSpecs.APIError) {
	return &maestroSpecs.APIError{
		HttpStatusCode: 501, // not implemented
		ErrorString:    "No Handler for Op Type",
		Detail:         s,
	}
}
func newAPIErrorFailedBatch(taskid string) (err *maestroSpecs.APIError) {
	return &maestroSpecs.APIError{
		HttpStatusCode: 424, // "Failed dependency"
		ErrorString:    "Batch failed",
		Detail:         "{\"taskId\":\"" + taskid + "\"}",
	}
}
func newAPIErrorRunawayTask(taskid string) (err *maestroSpecs.APIError) {
	return &maestroSpecs.APIError{
		HttpStatusCode: 508, // "loop detected"
		ErrorString:    "Runaway Task",
		Detail:         "{\"taskId\":\"" + taskid + "\"}",
	}
}

func runFinishAck(task *MaestroTask, handler AckHandler) {
	err := handler.SendFinishedAck(task)
	if err != nil {
		debugging.DEBUG_OUT("Task %s ackHandler.SendFinishedAck() failed %s\n", task.Id, err.Error())
		log.MaestroErrorf("Task %s ackHandler.SendFinishedAck() failed %s\n", task.Id, err.Error())
	}
}

func runFailedAck(task *MaestroTask, handler AckHandler) {
	err := handler.SendFailedAck(task)
	if err != nil {
		debugging.DEBUG_OUT("Task %s ackHandler.SendFailedAck() failed %s\n", task.Id, err.Error())
		log.MaestroErrorf("Task %s ackHandler.SendFailedAck() failed %s\n", task.Id, err.Error())
	}
}

// task handle thread
func taskRunner() {

	anyran := false

	getTask := func(id string) (ret *MaestroTask, ok bool) {
		retp, ok := tasks.Load(id)
		if ok {
			ret = retp.(*MaestroTask)
		}
		return
	}

	handleTask := func(taskitem *MaestroTask) {
		meta := getMetaTaskData(taskitem.Id)
		if taskitem.Step < TASK_COMPLETE_STEP && !meta.failed && !meta.stepExecuting { // && !isRunning(taskitem.Id)
			meta.runawayCheck++
			if meta.runawayCheck > RUNAWAY_THRESHOLD {
				meta.failed = true
				log.MaestroErrorf(" TASK>>>>>> Task looks like a runaway. Failing. id: %s\n", taskitem.Id)
				meta.release()
				FailTask(taskitem.Id, newAPIErrorRunawayTask(taskitem.Id))
				return
			}
			meta.release()
			if taskitem.Op == nil {
				if len(taskitem.childrenOrder) > 0 {
					for n, child := range taskitem.childrenOrder {
						task, ok := getTask(child)
						if ok {
							// check steps

							if !isRunning(child) {
								debugging.DEBUG_OUT(" TASK>>>>>> batch Task %s - found child %d  - %s\n", taskitem.Id, n, task.Id)
								_handler, ok := handlers.Load(task.Op.GetType())
								if ok && _handler != nil {
									markAsRunning(task.Id)
									debugging.DEBUG_OUT(" TASK>>>>>> Executing task %s on step %d (%s) %+v\n", task.Id, task.Step, task.Op.GetType(), isRunning(task.Id))
									handler := _handler.(*TaskHandler)
									taskcopy := new(MaestroTask)
									*taskcopy = *task
									(*handler).SubmitTask(taskcopy)
									anyran = true
									return
								} else {
									debugging.DEBUG_OUT(" TASK>>>>>> (batch) No TaskHandler for task type %s\n", task.Op.GetType())
									log.MaestroErrorf(" TASK>>>>>> (batch) No TaskHandler for task type %s\n", task.Op.GetType())
									FailTask(taskitem.Id, newAPIErrorNoOpTypeHandler("@ taskRunner()::handleTask() (batch)"))
								}
							} else {
								debugging.DEBUG_OUT(" TASK>>>>>> batch Task %s - child %d is already running\n", taskitem.Id, n)
								return
							}
						} else {
							debugging.DEBUG_OUT(" TASK>>>>>> ERROR - nil task in children of %s\n", taskitem.Id)
							return
						}
					}
				} else {
					debugging.DEBUG_OUT(" TASK>>>>>> ERROR You have what looks to be a batch task, but has no children, task:%s\n", taskitem.Id)
					FailTask(taskitem.Id, newAPIErrorNoOpTypeHandler("@ taskRunner()::handleTask() - broke Batch task? - no children\n"))
					return
				}
			} else {
				// Looks to be a standard task
				// find a handler, and attempt to execute this Task
				_handler, ok := handlers.Load(taskitem.Op.GetType())
				if ok && _handler != nil {
					markAsRunning(taskitem.Id)
					debugging.DEBUG_OUT(" TASK>>>>>> Executing task %s on step %d (%s) %+v\n", taskitem.Id, taskitem.Step, taskitem.Op.GetType(), isRunning(taskitem.Id))
					handler := _handler.(*TaskHandler)
					taskitemcopy := new(MaestroTask)
					*taskitemcopy = *taskitem
					(*handler).SubmitTask(taskitemcopy)
					anyran = true
				} else {
					debugging.DEBUG_OUT(" TASK>>>>>> No TaskHandler for task type %s\n", taskitem.Op.GetType())
					log.MaestroErrorf(" TASK>>>>>> No TaskHandler for task type %s\n", taskitem.Op.GetType())
					FailTask(taskitem.Id, newAPIErrorNoOpTypeHandler("@ taskRunner()::handleTask()"))
				}
			}
		} else if !meta.failed {
			// Task is complete or is already running a step
			meta.release()
		} else {
			// Task has failed
			meta.release()
		}

	}

	// Find a task which can be ran
	findTask := func() (ret *MaestroTask) {
		tasks.Range(func(key, value interface{}) bool {
			if value == nil {
				debugging.DEBUG_OUT(" TASK>>>>>> CORRUPTION in hashmap - have null value pointer\n")
				return true
			}
			if value != nil {
				taskitem := value.(*MaestroTask)
				debugging.DEBUG_OUT(" TASK>>>>>> Looking at task %s on step %d  {%+v}\n", taskitem.Id, taskitem.Step, *taskitem)

				meta := getMetaTaskData(taskitem.Id)
				// failed tasks are kept around until an ACK
				// for the Task is sent successfully
				// Just skip them
				if meta.failed {
					debugging.DEBUG_OUT2(" TASK>>>>>> Task %s is marked failed. next...\n", taskitem.Id)
					meta.release()
					return true
				}

				// the Task is already executing a step
				// so leave it alone
				if meta.stepExecuting {
					debugging.DEBUG_OUT2(" TASK>>>>>> Task %s is marked executing now. next...\n", taskitem.Id)
					meta.release()
					return true
				}

				// If this is a batchTask - then it will have subtasks
				// Each subtask must be done in order
				// If it is a subtask, then it will have a parentTaskId
				if len(taskitem.childrenOrder) > 0 {
					//markAsRunning(taskitem.Id) // mark the batch task as running
					meta.running = true
					meta.release()
					debugging.DEBUG_OUT(" TASK>>>>>> Task %s is a batch task.\n", taskitem.Id)
				innerBatchLoop:
					for n, child := range taskitem.childrenOrder {
						childtask, ok := getTask(child)
						if ok {
							// check steps

							// if !isRunning(child) {
							childmeta := getMetaTaskData(childtask.Id)
							if childmeta.failed {
								childmeta.release()
								// if a Task has failed, then if the Batch task is marked 'failFast' the entire
								// Batch should be marked failed
								FailTask(taskitem.Id, newAPIErrorFailedBatch(taskitem.Id))
								return true
							} else {
								childmeta.release()
								debugging.DEBUG_OUT(" TASK>>>>>> batch Task %s - found child %d  - %s\n", taskitem.Id, n, childtask.Id)
								ret = childtask
							}
							return false
							// } else {
							// 	debugging.DEBUG_OUT(" TASK>>>>>> batch Task %s - child %d is already running\n",taskitem.Id,n)
							// 	// TODO - check timeout

							// 	continue outerTaskLoop
							// }
						} else {
							debugging.DEBUG_OUT(" TASK>>>>>> ERROR - nil task in children of %s\n", taskitem.Id)
							continue innerBatchLoop
						}
					}
				} else {
					meta.release()
				}

				if taskitem.Step < TASK_COMPLETE_STEP { // && !isRunning(taskitem.Id)
					// if this task is a child, then skip it
					// Task children must be done in order.
					if len(taskitem.parentTaskId) > 0 {
						// skip it

					} else {
						if taskitem.Op != nil {
							debugging.DEBUG_OUT(" TASK>>>>>> Eligible task %s on step %d (%s) %+v\n", taskitem.Id, taskitem.Step, taskitem.Op.GetType(), isRunning(taskitem.Id))
						} else {
							debugging.DEBUG_OUT(" TASK>>>>>> Eligible task %s on step %d (nil) %+v\n", taskitem.Id, taskitem.Step, isRunning(taskitem.Id))
						}

						ret = taskitem
						//						handleTask(taskitem)
					}
				} else {
					debugging.DEBUG_OUT2(" TASK>>>>>> Task %s is at TASK_COMPLETE_STEP (%d)\n", taskitem.Id, taskitem.Step)
					// ok cool, now check to see if we have an ackHandler
					meta := getMetaTaskData(taskitem.Id)
					handler := meta.ackHandler
					meta.release()
					if handler != nil {
						go runFinishAck(taskitem, handler)
					}

				}

			}
			return true
		})
		return
	}

	n := 0

taskLooper:
	for true {
		anyran = false

		debugging.DEBUG_OUT("********************* top of taskRunner %d ***********************\n", n)
		n++
		select {
		case <-internalTicker.C:
			cleanOutStaleTasks()
		case token := <-controlChan:
			if token.op == op_shutdown {
				break taskLooper
			}

			if token.op == op_next_finished {
				debugging.DEBUG_OUT(" TASK>>>>>>>> op_next_finished\n")
				// if a task has completed, then let's see if it was part of a Batch?
				//
				if token.task != nil {
					task, ok := getTask(*token.task)
					if ok {
						if isRunning(*token.task) {
							debugging.DEBUG_OUT(" TASK>>>>>>>> ERROR - got a op_next_finished for task %s but it is still running?\n", *token.task)
						} else {
							if len(task.parentTaskId) > 0 {
								parenttask, ok := getTask(task.parentTaskId)
								if ok {
									// sanity check
									if parenttask.childrenOrder[task.batchOrder] == task.Id {
										// if this is not the last subtask
										if len(parenttask.childrenOrder) > (int(task.batchOrder) + 1) {
											nexttask, ok := getTask(parenttask.childrenOrder[int(task.batchOrder)+1])
											if ok {
												handleTask(nexttask)
											} else {
												debugging.DEBUG_OUT(" TASK>>>>>>>> ERROR missing next task - %s parent was %s\n", parenttask.childrenOrder[int(task.batchOrder)+1], parenttask.Id)
											}
										} else {
											// Batch complete!
											debugging.DEBUG_OUT(" TASK>>>>>>>> OK. Batch task is complete: %s\n", parenttask.Id)
											CompleteTask(parenttask.Id)
										}
									} else {
										debugging.DEBUG_OUT(" TASK>>>>>>>> ERROR mismatched order of parent / child task - %s\n", task.Id)
									}
								} else {
									debugging.DEBUG_OUT(" TASK>>>>>>>> ERROR Can't find parent task - %s\n", task.parentTaskId)
								}
							}
						}
						//						handleTask(task)
					} else {
						debugging.DEBUG_OUT(" TASK>>>>>>>> ERROR - could not find referenced task %s\n", *token.task)
					}
				} else {

				}

			}

			if token.op == op_next_task {
				debugging.DEBUG_OUT(" TASK>>>>>>>> op_next_task\n")
				// if a task has completed, then let's see if it was part of a
				if token.task != nil {
					nexttask, ok := getTask(*token.task)
					debugging.DEBUG_OUT(" TASK>>>>>>>> op_next_task task -> %s\n", *token.task)
					if ok {
						handleTask(nexttask)

					} else {

					}
				} else {
					nexttask := findTask()
					if nexttask != nil {
						handleTask(nexttask)
					}
				}

				// opportunistically look for new tasks
				// look at all tasks, and check for one at stage 0

				if anyran {
					controlChan <- newControlTokenNewTask(nil) // ask for another run
				}
			}

		}

	}

}

// Generics follow
////////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////////

//  begin generic
// m4_define({{*NODE*}},{{*MaestroTask*}})  m4_define({{*FIFO*}},{{*taskFIFO*}})
// Thread safe queue for LogBuffer
type taskFIFO struct {
	q          []*MaestroTask
	mutex      *sync.Mutex
	condWait   *sync.Cond
	condFull   *sync.Cond
	maxSize    uint32
	drops      int
	shutdown   bool
	wakeupIter int // this is to deal with the fact that go developers
	// decided not to implement pthread_cond_timedwait()
	// So we use this as a work around to temporarily wakeup
	// (but not shutdown) the queue. Bring your own timer.
}

func New_taskFIFO(maxsize uint32) (ret *taskFIFO) {
	ret = new(taskFIFO)
	ret.mutex = new(sync.Mutex)
	ret.condWait = sync.NewCond(ret.mutex)
	ret.condFull = sync.NewCond(ret.mutex)
	ret.maxSize = maxsize
	ret.drops = 0
	ret.shutdown = false
	ret.wakeupIter = 0
	return
}
func (fifo *taskFIFO) Push(n *MaestroTask) (drop bool, dropped *MaestroTask) {
	drop = false
	debugging.DEBUG_OUT2(" >>>>>>>>>>>> [ taskFIFO ] >>> In Push\n")
	fifo.mutex.Lock()
	debugging.DEBUG_OUT2(" ------------ In Push (past Lock)\n")
	if int(fifo.maxSize) > 0 && len(fifo.q)+1 > int(fifo.maxSize) {
		// drop off the queue
		dropped = (fifo.q)[0]
		fifo.q = (fifo.q)[1:]
		fifo.drops++
		debugging.DEBUG_OUT("!!! Dropping MaestroTask in taskFIFO \n")
		drop = true
	}
	fifo.q = append(fifo.q, n)
	debugging.DEBUG_OUT2(" ------------ In Push (@ Unlock)\n")
	fifo.mutex.Unlock()
	fifo.condWait.Signal()
	debugging.DEBUG_OUT2(" <<<<<<<<<<< Return Push\n")
	return
}

// Pushes a batch of MaestroTask. Drops older MaestroTask to make room
func (fifo *taskFIFO) PushBatch(n []*MaestroTask) (drop bool, dropped []*MaestroTask) {
	drop = false
	debugging.DEBUG_OUT2(" >>>>>>>>>>>> [ taskFIFO ] >>> In PushBatch\n")
	fifo.mutex.Lock()
	debugging.DEBUG_OUT2(" ------------ In PushBatch (past Lock)\n")
	_len := uint32(len(fifo.q))
	_inlen := uint32(len(n))
	if fifo.maxSize > 0 && _inlen > fifo.maxSize {
		_inlen = fifo.maxSize
	}
	if fifo.maxSize > 0 && _len+_inlen > fifo.maxSize {
		needdrop := _inlen + _len - fifo.maxSize
		if needdrop >= fifo.maxSize {
			drop = true
			dropped = fifo.q
			fifo.q = nil
		} else if needdrop > 0 {
			drop = true
			dropped = (fifo.q)[0:needdrop]
			fifo.q = (fifo.q)[needdrop:]
		}
		// // drop off the queue
		// dropped = (fifo.q)[0]
		// fifo.q = (fifo.q)[1:]
		// fifo.drops++
		debugging.DEBUG_OUT2(" ----------- PushBatch() !!! Dropping %d MaestroTask in taskFIFO \n", len(dropped))
	}
	debugging.DEBUG_OUT2(" ----------- In PushBatch (pushed %d)\n", _inlen)
	fifo.q = append(fifo.q, n[0:int(_inlen)]...)
	debugging.DEBUG_OUT2(" ------------ In PushBatch (@ Unlock)\n")
	fifo.mutex.Unlock()
	fifo.condWait.Signal()
	debugging.DEBUG_OUT2(" <<<<<<<<<<< Return PushBatch\n")
	return
}

func (fifo *taskFIFO) Pop() (n *MaestroTask) {
	fifo.mutex.Lock()
	if len(fifo.q) > 0 {
		n = (fifo.q)[0]
		fifo.q = (fifo.q)[1:]
		fifo.condFull.Signal()
	}
	fifo.mutex.Unlock()
	return
}

// func (fifo *taskFIFO) PopBatch(max uint32) (n *MaestroTask) {
// 	fifo.mutex.Lock()
// 	_len := len(fifo.q)
// 	if _len > 0 {
// 		if _len >= max {

// 		} else {

// 		}
// 	    n = (fifo.q)[0]
// 	    fifo.q = (fifo.q)[1:]
// 		fifo.condFull.Signal()
// 	}
// 	fifo.mutex.Unlock()
//     return
// }
func (fifo *taskFIFO) Len() int {
	fifo.mutex.Lock()
	ret := len(fifo.q)
	fifo.mutex.Unlock()
	return ret
}
func (fifo *taskFIFO) PopOrWait() (n *MaestroTask) {
	n = nil
	debugging.DEBUG_OUT2(" >>>>>>>>>>>> In PopOrWait (Lock)\n")
	fifo.mutex.Lock()
	_wakeupIter := fifo.wakeupIter
	if fifo.shutdown {
		fifo.mutex.Unlock()
		debugging.DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWait (Unlock 1)\n")
		return
	}
	if len(fifo.q) > 0 {
		n = (fifo.q)[0]
		fifo.q = (fifo.q)[1:]
		fifo.mutex.Unlock()
		fifo.condFull.Signal()
		debugging.DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWait (Unlock 2)\n")
		return
	}
	// nothing there, let's wait
	for !fifo.shutdown && fifo.wakeupIter == _wakeupIter {
		//		fmt.Printf(" --entering wait %+v\n",*fifo);
		debugging.DEBUG_OUT2(" ----------- In PopOrWait (Wait / Unlock 1)\n")
		fifo.condWait.Wait() // will unlock it's "Locker" - which is fifo.mutex
		//		Wait returns with Lock
		//		fmt.Printf(" --out of wait %+v\n",*fifo);
		if fifo.shutdown {
			fifo.mutex.Unlock()
			debugging.DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWait (Unlock 4)\n")
			return
		}
		if len(fifo.q) > 0 {
			n = (fifo.q)[0]
			fifo.q = (fifo.q)[1:]
			fifo.mutex.Unlock()
			fifo.condFull.Signal()
			debugging.DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWait (Unlock 3)\n")
			return
		}
	}
	debugging.DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWait (Unlock 5)\n")
	fifo.mutex.Unlock()
	return
}
func (fifo *taskFIFO) PopOrWaitBatch(max uint32) (slice []*MaestroTask) {
	debugging.DEBUG_OUT2(" >>>>>>>>>>>> In PopOrWaitBatch (Lock)\n")
	fifo.mutex.Lock()
	_wakeupIter := fifo.wakeupIter
	if fifo.shutdown {
		fifo.mutex.Unlock()
		debugging.DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 1)\n")
		return
	}
	_len := uint32(len(fifo.q))
	if _len > 0 {
		if max >= _len {
			slice = fifo.q
			fifo.q = nil // http://stackoverflow.com/questions/29164375/golang-correct-way-to-initialize-empty-slice
		} else {
			slice = (fifo.q)[0:max]
			fifo.q = (fifo.q)[max:]
		}
		fifo.mutex.Unlock()
		fifo.condFull.Signal()
		debugging.DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 2)\n")
		return
	}
	// nothing there, let's wait
	for !fifo.shutdown && fifo.wakeupIter == _wakeupIter {
		//		fmt.Printf(" --entering wait %+v\n",*fifo);
		debugging.DEBUG_OUT2(" ----------- In PopOrWaitBatch (Wait / Unlock 1)\n")
		fifo.condWait.Wait() // will unlock it's "Locker" - which is fifo.mutex
		//		Wait returns with Lock
		//		fmt.Printf(" --out of wait %+v\n",*fifo);
		if fifo.shutdown {
			fifo.mutex.Unlock()
			debugging.DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 4)\n")
			return
		}
		_len = uint32(len(fifo.q))
		if _len > 0 {
			if max >= _len {
				slice = fifo.q
				fifo.q = nil // http://stackoverflow.com/questions/29164375/golang-correct-way-to-initialize-empty-slice
			} else {
				slice = (fifo.q)[0:max]
				fifo.q = (fifo.q)[max:]
			}
			fifo.mutex.Unlock()
			fifo.condFull.Signal()
			debugging.DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 3)\n")
			return
		}
	}
	debugging.DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 5)\n")
	fifo.mutex.Unlock()
	return
}
func (fifo *taskFIFO) PushOrWait(n *MaestroTask) (ret bool) {
	ret = true
	fifo.mutex.Lock()
	_wakeupIter := fifo.wakeupIter
	for int(fifo.maxSize) > 0 && (len(fifo.q)+1 > int(fifo.maxSize)) && !fifo.shutdown && (fifo.wakeupIter == _wakeupIter) {
		//		fmt.Printf(" --entering push wait %+v\n",*fifo);
		fifo.condFull.Wait()
		if fifo.shutdown {
			fifo.mutex.Unlock()
			ret = false
			return
		}
		//		fmt.Printf(" --exiting push wait %+v\n",*fifo);
	}
	fifo.q = append(fifo.q, n)
	fifo.mutex.Unlock()
	fifo.condWait.Signal()
	return
}
func (fifo *taskFIFO) Shutdown() {
	fifo.mutex.Lock()
	fifo.shutdown = true
	fifo.mutex.Unlock()
	fifo.condWait.Broadcast()
	fifo.condFull.Broadcast()
}
func (fifo *taskFIFO) WakeupAll() {
	debugging.DEBUG_OUT2(" >>>>>>>>>>> in WakeupAll @Lock\n")
	fifo.mutex.Lock()
	debugging.DEBUG_OUT2(" +++++++++++ in WakeupAll\n")
	fifo.wakeupIter++
	fifo.mutex.Unlock()
	debugging.DEBUG_OUT2(" +++++++++++ in WakeupAll @Unlock\n")
	fifo.condWait.Broadcast()
	fifo.condFull.Broadcast()
	debugging.DEBUG_OUT2(" <<<<<<<<<<< in WakeupAll past @Broadcast\n")
}
func (fifo *taskFIFO) IsShutdown() (ret bool) {
	fifo.mutex.Lock()
	ret = fifo.shutdown
	fifo.mutex.Unlock()
	return
}

// end generic
//
//
