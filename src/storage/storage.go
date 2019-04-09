package storage

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
	"errors"
//	"encoding/json"
	"encoding/gob"
	"github.com/WigWagCo/maestroSpecs"
	//"github.com/syndtr/goleveldb/leveldb"
	"github.com/boltdb/bolt"
	"github.com/WigWagCo/stow"
)

const c_CONFIG_PREFIX = "CONFIG."

const c_JOB_STORE = "jobs"
const c_TEMPLATE_STORE = "contTempls"
const c_CONFIG_STORE = "config"


type ForEachJobCB func (job maestroSpecs.JobDefinition)
type ForEachContainerTemplateCB func (job maestroSpecs.ContainerTemplate)

type MaestroDBInstance struct {
//	db *leveldb.DB
	Db *bolt.DB
	dbName string

	// bolt / stow specific:
	jobStore *stow.Store
	containerTemplateStore *stow.Store
	configStore *stow.Store
	// taskQueue *stow.Store
}

type MaestroDBStorageInterface interface {
	GetDb() *bolt.DB
	GetDbName() string
}

func (this *MaestroDBInstance) GetDb() *bolt.DB {
	return this.Db
}

func (this *MaestroDBInstance) GetDbName() string {
	return this.dbName
}


var isReady bool
var needsInit bool
var singleton *MaestroDBInstance

func GetStorage() (ret *MaestroDBInstance) {
	if isReady {
		ret = singleton
		return
	} else {
		return
	}
}


// type StorageUser interface {
// 	// called when the DB is open and ready
// 	StorageReady(instance *MaestroDBInstance)
// 	// called when the DB is initialized
// 	StorageInit(instance *MaestroDBInstance)
// 	// called when storage is closed
// 	StorageClosed(instance *MaestroDBInstance)
// }
type StorageUser interface {
	// called when the DB is open and ready
	StorageReady(instance MaestroDBStorageInterface)
	// called when the DB is initialized
	StorageInit(instance MaestroDBStorageInterface)
	// called when storage is closed
	StorageClosed(instance MaestroDBStorageInterface)
}

type StorageReadyCB func (instance *MaestroDBInstance)

var storageUsers []StorageUser

func RegisterStorageUser(user StorageUser) {
	storageUsers = append(storageUsers,user)
	if isReady {
		// if needsInit {
			user.StorageInit(singleton)
		// }
		user.StorageReady(singleton)
	}
}

// func OnStorageReady(cb StorageReadyCB) {
// 	if isReady {
// 		cb(singleton)
// 	} else {
// 		onReadyCBs = append(onReadyCBs,cb)		
// 	}
// }

// func OnStorageInit(cb StorageInitCB) {

// }

func InitStorage(path string) (ret *MaestroDBInstance, err error) {
	if isReady {
		ret = singleton
		return
	}
	// //leveldb:
	// ret = &MaestroDBInstance{ db: nil, dbName: "" }
	// var err error
	// ret.db, err = leveldb.OpenFile(path, nil)
	// if err != nil {
	// 	ret = nil
	// } else {
	// 	// return
	// }

	// won't store stuff if we don't register the types with the gob encoder
	gob.Register(&maestroSpecs.JobDefinitionPayload{})
	gob.Register(&maestroSpecs.ContainerTemplatePayload{})
	gob.Register(&maestroSpecs.StatsConfigPayload{})
//	gob.Register(&maestroTasks.MaestroTask{})
	gob.Register(&maestroSpecs.JobOpPayload{})
	gob.Register(&maestroSpecs.ImageOpPayload{})
	gob.Register(&maestroSpecs.JobDefinitionPayload{})
	gob.Register(&maestroSpecs.ImageDefinitionPayload{})

	ret = &MaestroDBInstance{ Db: nil, dbName: "" }
	ret.Db, err = bolt.Open(path,0600,nil)
	if err != nil {
		
	} else {
		needsInit = true
		ret.jobStore = stow.NewStore(ret.Db, []byte(c_JOB_STORE))
		ret.containerTemplateStore = stow.NewStore(ret.Db, []byte(c_TEMPLATE_STORE))
		ret.configStore = stow.NewStore(ret.Db, []byte(c_CONFIG_STORE))
//		ret.taskQueue = stow.NewStore(ret.Db, []byte(c_TASK_QUEUE))

		singleton = ret

		for _, user := range storageUsers {
			user.StorageInit(singleton)
		}

		// call all onReady callbacks if any are waiting
		// for _, cb := range onReadyCBs {
		// 	cb(singleton)
		// }
		isReady = true
		for _, user := range storageUsers {
			user.StorageReady(singleton)
		}


	}
	return
}

func (this *MaestroDBInstance) Close() {
	// works for either bolt or leveldb
	if this.Db != nil {
		this.Db.Close()
		this.Db = nil
	}
}


func (this *MaestroDBInstance) ForEachJob(cb ForEachJobCB) error {
	if this.Db != nil {
		this.jobStore.ForEach(cb)
		return nil
	} else {
		return errors.New("no database")
	}
}

func (this *MaestroDBInstance) ForEachContainerTemplate(cb ForEachContainerTemplateCB) error {
	if this.Db != nil {
		this.containerTemplateStore.ForEach(cb)
		return nil
	} else {
		return errors.New("no database")
	}
}

func (this *MaestroDBInstance) GetStatsConfig() (ret maestroSpecs.StatsConfig, err error) {
	if this.Db != nil {
		this.configStore.Get("statsConfig",&ret)
	} else {
		err = errors.New("no database")
	}
	return
}

func (this *MaestroDBInstance) SetStatsConfig(obj maestroSpecs.StatsConfig) error {
	if this.Db != nil {
		this.configStore.Put("statsConfig",&obj)
		return nil
	} else {
		return errors.New("no database")
	}
}







// func (this *MaestroDBInstance) UpsertTask(obj *MaestroTask) error {
// 	// key := obj.GetJobName()
// 	if this.Db != nil {
// 		this.taskQueue.Put(obj.Id, obj)
// 		return nil
// 	} else {
// 		return errors.New("no database")		
// 	}
// }

// type UpdateTaskCB func(task *MaestroTask)
// type IfTaskCB func(task *MaestroTask) bool

// func (this *MaestroDBInstance) UpdateTaskById(id string, cb UpdateTaskCB) (err error, updated *MaestroTask) {
// 	if this.Db != nil {
// 		temp := new(MaestroTask)
// 		err = this.taskQueue.Update(id,temp, func(val interface{}){
// 			if val != nil {
// 				task, ok := val.(*MaestroTask)
// 				if ok {
// 					cb(task)
// 				} else {
// 					// should be impossible		
// 				}
// 			} else {
// 				// should be impossible, if 'id' did not exist 
// 				// then callback will not be called
// 			}
// 		})
// 		updated = temp
// 		return
// 	} else {
// 		err = errors.New("no database")
// 		return
// 	}
// }

// func (this *MaestroDBInstance) GetTaskById(id string) (obj *MaestroTask, err error) {
// 	if this.Db != nil {
// 		this.taskQueue.Get(id,&obj)
// 	} else {
// 		err = errors.New("no database")
// 	}
// 	return
// }

// func (this *MaestroDBInstance) RemoveTaskById(id string) error {
// 	if this.Db != nil {
// 		this.taskQueue.Delete(id)
// 		return nil
// 	} else {
// 		return errors.New("no database")
// 	}
// }

// func (this *MaestroDBInstance) ForEachIfTask(cb IfTaskCB) error {
// 	var temp MaestroTask
// 	return this.taskQueue.IterateIf(func(key []byte, val interface{}) bool {
// 		ty, ok := val.(*MaestroTask)
// 		if ok {
// 			return cb(ty) // true, then we should keep iterating
// 		} else {
// 			MaestroWarn("Possible DB corruption - not castable to MaestroTask\n")
// 			return false
// 		}
// 	},&temp)	
// }

// // If the callback returns true, then the MaestroTask will be removed
// // from the database
// func (this *MaestroDBInstance) DeleteIfTask(cb IfTaskCB) error {
// 	var temp MaestroTask
// 	return this.taskQueue.DeleteIf(func(key []byte, val interface{}) bool {
// 		ty, ok := val.(*MaestroTask)
// 		if ok {
// 			return cb(ty) // true, then it should be deleted
// 		} else {
// 			MaestroWarn("Possible DB corruption - not castable to MaestroTask\n")
// 			return false
// 		}
// 	},&temp)
// }


// JOBS: storage for Maestro jobs, known processes / containers which Maestro starts and stops

func (this *MaestroDBInstance) UpsertJob(obj maestroSpecs.JobDefinition) error {
	// key := obj.GetJobName()
	if this.Db != nil {
		this.jobStore.Put(obj.GetJobName(), &obj)
		return nil
	} else {
		return errors.New("no database")		
	}
}

func (this *MaestroDBInstance) UpsertJobAsPayload(obj *maestroSpecs.JobDefinitionPayload) error {
	if this.Db != nil {
		this.jobStore.Put(obj.Job, obj)
		return nil
	} else {
		return errors.New("no database")		
	}
	// leveldb:
	// key := []byte(c_CONFIG_PREFIX+obj.Job)

	// val, err := json.Marshal(obj)
	// err = this.db.Put(key,val,nil)

	// return err
	// return nil
}



func (this *MaestroDBInstance) DeleteJob(byname string) error {
	if this.Db != nil {
		this.jobStore.Delete(byname)
		return nil
	} else {
		return errors.New("no database")
	}
}

func (this *MaestroDBInstance) GetJobByName(byname string) (job maestroSpecs.JobDefinition, err error) {
	if this.Db != nil {
		this.jobStore.Get(byname,&job)
	} else {
		err = errors.New("no database")
	}
	return
}

func (this *MaestroDBInstance) UpsertContainerTemplate(obj maestroSpecs.ContainerTemplate) error {
	if this.Db != nil {
		this.containerTemplateStore.Put(obj.GetName(), &obj)
		return nil
	} else {
		return errors.New("no database")		
	}
}

func (this *MaestroDBInstance) DeleteContainerTemplate(byname string) error {
	if this.Db != nil {
		this.containerTemplateStore.Delete(byname)
		return nil
	} else {
		return errors.New("no database")
	}
}

func (this *MaestroDBInstance) GetContainerTemplateByName(byname string) (templ maestroSpecs.ContainerTemplate, err error) {
	if this.Db != nil {
		this.containerTemplateStore.Get(byname,&templ)
	} else {
		err = errors.New("no database")
	}
	return
}



