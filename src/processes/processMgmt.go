package processes

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

/*
#cgo amd64 LDFLAGS: -L/usr/lib/x86_64-linux-gnu 
#cgo LDFLAGS: -L${SRCDIR}/../vendor/github.com/WigWagCo/greasego/deps/lib
#cgo LDFLAGS: -lgrease -luv -lTW  -lstdc++ -lm -ltcmalloc_minimal -lm
#cgo CFLAGS: -I${SRCDIR}/../vendor/github.com/WigWagCo/greasego/deps/include DEBUG(-DDEBUG_BINDINGS) -I${SRCDIR}/processes
#define GREASE_IS_LOCAL 1
#include <stdio.h>
#include "process_utils.h"
#include "grease_lib.h"

*/
import "C"

import (
	"bytes"
	"github.com/WigWagCo/greasego"
	"unsafe"
	"sync"
	"regexp"
	"sync/atomic"
	"encoding/json"
	"github.com/WigWagCo/hashmap"  // thread-safe, fast hashmaps
//	"github.com/WigWagCo/gopsutil/mem"
	"github.com/WigWagCo/maestroSpecs"
	"github.com/WigWagCo/maestro/storage"
	"github.com/WigWagCo/maestro/configMgr"
	"github.com/WigWagCo/maestro/configs"	
	"github.com/kardianos/osext"  // not needed in go1.8 - see their README
	"os"
	"golang.org/x/sys/unix"
	"syscall"
	"time"
	"strconv"
	"fmt"

//	"unsafe"
)

type ProcessStatsConfig interface {
	GetInterval() uint32
	GetConfig_CheckMem() (uint32, bool)
}

const (
	STDIN_CLOSE = iota
	STDIN_TO_MANAGER = iota
)

const (
	STDOUT_CLOSE = iota
	STDOUT_TO_LOGGER = iota
	STDOUT_TO_MANAGER = iota
)

const (
	STDERR_CLOSE = iota
	STDERR_TO_LOGGER = iota
	STDERR_TO_MANAGER = iota
)

const (
	PROCESS_STARTED = 0x1
	PROCESS_READY = 0x2
	PROCESS_DIED = 0x4
)

var PROCESS_STATES = map[uint32]string{
  0: "unknown",
  PROCESS_STARTED: "started",
  PROCESS_READY : "ready",
  PROCESS_DIED : "died",
}

type ExecFileOpts struct {
	jobName string
	compId string
	// stdoutDirection int
	// stderrDirection int
	// stdinDirection int
	internal C.childOpts
}

// The magic message sent from a process 
// which is following the maestro startup procedures.
// when it writes this to stdout, the process is finished 
// with startup.
const process_OK_message = maestroSpecs.JOB_OK_STRING

func NewExecFileOpts(jobname string, compid string) (opts *ExecFileOpts) {
	opts = new(ExecFileOpts)
	opts.jobName = "[no job name]"
	if len(jobname) > 0 {
		opts.jobName = jobname
	}
	opts.compId = "[no composite id]"
	if len(compid) > 0 {
		opts.compId = compid
	}	
	opts.internal.pgid = 0
	opts.internal.flags = 0
	opts.internal.chroot = nil
	opts.internal.message = nil	
	opts.internal.ok_string = 0
	opts.internal.jobname = nil
	opts.internal.stdout_fd = 0
	opts.internal.stderr_fd = 0
	opts.internal.die_on_parent_sig = 0
	opts.internal.env_GREASE_ORIGIN_ID = 0
	return
}

func (opts *ExecFileOpts) SetNewPgid() {
	opts.internal.flags &= ^C.uint32_t(C.PROCESS_USE_PGID)
}

func (opts *ExecFileOpts) SetUsePgid(pid int) {
	opts.internal.flags |= C.uint32_t(C.PROCESS_USE_PGID)
	opts.internal.pgid = C.pid_t(pid)
}

func (opts *ExecFileOpts) SetJobName(name string) {
	if len(name) > 0 {
		opts.internal.jobname = C.CString(name)
	} else {
		opts.internal.jobname = nil
	}	
}

func (opts *ExecFileOpts) GetOriginLabelId() uint32 {
	return uint32(opts.internal.originLabel)
}

func (opts *ExecFileOpts) SetNewSid() {
	opts.internal.flags |= C.uint32_t(C.PROCESS_NEW_SID)
}

func (opts *ExecFileOpts) SetMessageString(msg string) {
	if len(msg) > 0 {
		opts.internal.message = C.CString(msg)
	} else {
		opts.internal.message = nil
	}
}

func (opts *ExecFileOpts) SetOkString(ok bool) {
	if ok {
		opts.internal.ok_string = 1
	} else {
		opts.internal.ok_string = 0
	}
}

func (opts *ExecFileOpts) GetStdoutFd() (ret int) {
	ret = int(opts.internal.stdout_fd)
	return
}
func (opts *ExecFileOpts) GetStderrFd() (ret int) {
	ret = int(opts.internal.stderr_fd)
	return
}


// a record of start and/or stop operation. basically a 
// single life of a process (not a Job)
type ProcessRecord struct {
	Pid int
	// the Path to start the process executable
	Path string
	// serves as a generic identifier for a particular job
	// jobs maybe restarted, etc. A Job should be a unique name to the 
	// instance of maestro on this gateway
	Job string
	CompId string
	Started time.Time
	Stopped time.Time
	State uint32  // see PROCESS_STATES
	ExitValue int
	// cgroup
	// pgid
}

// returns a ProcessRecord by its PID (process ID)
func GetTrackedProcessByPid(pid int) (ret *ProcessRecord, ok bool) {
	var val unsafe.Pointer
	val, ok = processIndexByPid.GetHashedKey(uintptr(pid))
	if ok {
		ret = (*ProcessRecord)(val)
	}
//	ret, ok = processIndexByPid[pid]
	return
}

// creates a new ProcessRecord, stores it in the tracking map, and returns this new one
// the old one - if for some rare reason a process with the same PID was already there
func TrackNewProcessByPid(pid int) (ret *ProcessRecord, old *ProcessRecord) {
	oldval, ok := processIndexByPid.GetHashedKey(uintptr(pid))
	if ok {
		old = (*ProcessRecord)(oldval)
	}
	ret = new(ProcessRecord)
	ret.Pid = pid
	ret.Path = ""
	processIndexByPid.SetHashedKey(uintptr(pid),unsafe.Pointer(ret))
	// old = processIndexByPid[pid]
	// processIndexByPid[pid] = ret
	return
}

func RemoveTrackedProcessByPid(pid int) (ret *ProcessRecord) {
//	var ok bool
//	ret, ok = processIndexByPid[pid]
	oldval, ok := processIndexByPid.GetHashedKey(uintptr(pid))
	if ok {
		ret = (*ProcessRecord)(oldval)
		processIndexByPid.DelHashedKey(uintptr(pid))
//		delete(processIndexByPid,pid)		
	}
	return
}

func ForEachTrackedProcess(cb func(int, *ProcessRecord)) {
	for keyval := range processIndexByPid.Iter() {
		key, ok := keyval.Key.(uintptr)
		if ok {
		 	cb(int(key),(*ProcessRecord)(keyval.Value))			
		}
	}
	// for pid, record := range processIndexByPid {
	// 	cb(pid, record)
	// }
}

func NumberOfTrackedProcesses() (int) {
//	return len(processIndexByPid)
	return processIndexByPid.Len()
}



//var processIndexByPid map[int]*ProcessRecord
var processIndexByPid *hashmap.HashMap

// composite processes by their compositeId
var compositeProcessesById *hashmap.HashMap

// enum JOB_STATUS
const ( 
//	JOB_STATUS_UNSET = iota  // zero should be the unset 
	NEVER_RAN = iota
	RUNNING = iota
	STOPPED = iota
	WAITING_FOR_DEPS = iota
	STARTING = iota
	STOPPING = iota
	RESTARTING = iota
	FAILED = iota
)

var stringOfStatus map[int]string

type processStatus struct {
	// the original JobDefinition which create this process. This is 'nil' if this process
	// is a master process
	job maestroSpecs.JobDefinition

	// at some point, the Job is initially started.
	// We keep this around so we can reference on restarts, etc.
	originatingEvent *processEvent
	// for jobs which are waiting on the OK start screen, used if we get a partial messages
	startMessageBuf []rune
	status int  // JOB_STATUS
	restarts uint32
	pid int
	errno int    // last error if tried to exec() and failed    
	// composite data, formed from template & job start request
	execPath string
	execArgs []string
	execEnv []string
	usesOKString bool
	deferRunningStatus uint32
	originLabelId uint32 // used by greasego to identify this process to the logger

	// If this process is part of a composite process this is set
	// See masterComposite for more.
	compositeId string

	// attempt to make this process die if maestro dies
	dieOnParentDeath bool 

	// The following keys are for Composite Processes only:
	// 
	// masterComposite & compositeId 
	// A composite ID allows grouping of multiple 'jobs' together under a single process
	// If masterComposite is true, then this process is a master 'holding' process for one or more
	// Jobs (and their processStatus records)
	// This should always hold true: masterComposite == true && len(compositeID) > 0
	// for a master composite process
	masterComposite bool
	// all the 'GetDependsOn()' of all Jobs which this composite process holds
	// as a map
	combinedDeps map[string]bool
	// if true, we bundle up all the job data as JSON and send to the process
	// once started to it's standard input
	send_composite_jobs_to_stdin bool  
	// make a GREASE_ORIGIN_ID env var, and then when execvpe the new process, attach this env var
	// which will have the generated origin ID for the process. Useful for deviceJS
	send_GREASE_ORIGIN_ID bool
	// string of Job Names this composite process holds
	childNames map[string]int  // int is JOB_STATUS
	mutex sync.Mutex 
	compositeContainerTemplate string
}

func (this *processStatus) lock() {
	this.mutex.Lock()
}

func (this *processStatus) unlock() {
	this.mutex.Unlock()
}

// locks the mutex - returns .status
func (this *processStatus) lockingGetStatus() (ret int) {
	this.mutex.Lock()
	ret = this.status
	this.mutex.Unlock()
	return
}

// locks the mutex - returns .status
func (this *processStatus) lockingSetStatus(v int) {
	this.mutex.Lock()
	this.status = v
	this.mutex.Unlock()
}

// sets the status on jobs referenced in the childNames slice
func (this *processStatus) setAllChildJobStatus(status int) {
	this.mutex.Lock()
	for child, _ := range this.childNames {
		this.childNames[child] = status
		val, ok := jobsByName.GetStringKey(child) 
		if ok {
			child_proc := (*processStatus)(val)
			child_proc.status = status
//			child_proc.setAllChildJobStatus(status)
		} else {
			DEBUG_OUT("WARNING - could not find child referenced in .childNames in the jobsByName map!")
		}
	}
	this.mutex.Unlock()
}

// sends an event for all child processes - NOTE: needs a lock on processStatus first
func (status *processStatus) sendEventAndSetStatusForCompProcess(statuscode int, evcode int) {
	for child, _ := range status.childNames {
		status.childNames[child] = statuscode
		val, ok := jobsByName.GetStringKey(child) 
		if ok {
			child_proc := (*processStatus)(val)
			child_proc.status = statuscode
			jname := child_proc.job.GetJobName()
			internalEv := newProcessEvent(jname, evcode)
			controlChan <- internalEv
		} else {
			DEBUG_OUT("WARNING - could not find child referenced in .childNames in the jobsByName map!")
		}
	}
	status.status = statuscode
}


type containerStatus struct {
	uses int
	container maestroSpecs.ContainerTemplate
}

type JobError struct {
	JobName string
	Code int
	Aux string
}

type ContainerError struct {
	ContainerName string
	Code int
	Aux string
}

const (
	CONTAINERERROR_UNKNOWN = iota
	CONTAINERERROR_DUPLICATE = iota
)

const (
	JOBERROR_UNKNOWN = iota
	JOBERROR_NOT_FOUND = iota  // don't have a job by that name
	JOBERROR_DUPLICATE_JOB = iota
	JOBERROR_MISSING_DEPENDENCY = iota
	JOBERROR_DEPENDENCY_FAILURE = iota
	JOBERROR_MISSING_CONTAINER_TEMPLATE = iota
	JOBERROR_INVALID_CONFIG = iota  // config options don't make sense
	JOBERROR_INTERNAL_ERROR = iota  // catch all for things which should just not happen
)

var job_error_map = map[int]string{
  0: "unknown",
  JOBERROR_NOT_FOUND: "JOBERROR_NOT_FOUND",
  JOBERROR_MISSING_CONTAINER_TEMPLATE: "JOBERROR_MISSING_CONTAINER_TEMPLATE",  
}

var container_error_map = map[int]string{
	0: "unknown",
	CONTAINERERROR_DUPLICATE: "CONTAINERERROR_DUPLICATE",

}

func (this *JobError) Error() string {
	s := job_error_map[this.Code]
	return "JobError: " + this.JobName + ":"+s+" (" + strconv.Itoa(this.Code) +") " + this.Aux
}

func (this *ContainerError) Error() string {
	s := container_error_map[this.Code]
	return "JobError: " + this.ContainerName + ":"+s+" (" + strconv.Itoa(this.Code) +") " + this.Aux
}

const (
	internalEvent_unknown = iota     // not set!
	internalEvent_shutdown = iota     // shutdown the processEventWork thread
	internalEvent_start_job = iota    // start a process
	internalEvent_stop_job = iota     // tell loop to stop a process
	internalEvent_job_starting = iota  // event happens when the process is 'starting'
	internalEvent_job_running = iota  // event happens when the process is 'running'
	internalEvent_job_stopped = iota  // event happens when the process stopped
	internalEvent_job_died = iota     // an external event cause a process to die (we did not stop it)
	internalEvent_job_controlledExit = iota  // event happens when the process exited normally or killed explicitly
	internalEvent_job_failed = iota   // a job failed to start
)

// the internal event structure
// Note: not to be confused with ProcessEvent which is the external
// event, which is in processEvents.go
type processEvent struct {
	code int // codes above ^
	errno int
	// these interface pointers used as needed:
	jobname string
	request maestroSpecs.JobDefinition
}

// new internal process Event
func newProcessEvent(jobname string,code int) (ret *processEvent) {
	//ret := new(processEvent)
	ret = &processEvent{code,0,jobname,nil}
	return
}

func newProcessEventFromExisting(ev *processEvent,code int) (ret *processEvent) {
	//ret := new(processEvent)
	ret = &processEvent{code,0,ev.jobname,ev.request}
	return
}

func newShutdownEvent() (ret *processEvent) {
	ret = &processEvent{internalEvent_shutdown,0,"",nil}
	return
}

// var containerTemplatesByName map[string]*ContainerTemplate
// var jobsByName map[string]*processStatus
var containerTemplatesByName *hashmap.HashMap
var jobsByName *hashmap.HashMap

var controlChan chan *processEvent
var internalTicker *time.Ticker
var statsConfig ProcessStatsConfig
var monitorWorkerRunning int32

var globalDepGraph dependencyGraph  // defined in depGraph.go
var graphValidated = bool(true)     // the graph is empty to begin with, which is validated

const (
	worker_stopped = iota
	worker_started = iota
)

var re_this_dir *regexp.Regexp  // {{thisdir}}
var re_cwd *regexp.Regexp // {{cwd}}
var THIS_DIR string
var CWD string

func updateMacroVars() {
	cwd, err := os.Getwd()
	if err == nil {
		CWD = cwd
	}	
}

const (
	EPOLLET        = 1 << 31
	MaxEpollEvents = 32
)	


type epollListener struct {
	shutdown_messageListener int // 1 == shutdown, atomic	
	wakeup_fd_messageListener *WakeupFd
	epfd int // the epoll() fd
//	event syscall.EpollEvent
	events [MaxEpollEvents]syscall.EpollEvent
	fdToJob *hashmap.HashMap
}

var mainEpollListener *epollListener

func newEpollListener() (ret *epollListener, e error) {	
	if epfd, e := syscall.EpollCreate1(0); e == nil {
		ret = new(epollListener)
		ret.epfd = epfd	
		ret.fdToJob = hashmap.New(10)
	} else {
		DEBUG_OUT("!!!!!!!!!!! Error creating epoll FD: %s\n",e.Error())
	}
	return
}


const listener_shutdown_magic_num = 99
// this worker deals specifically with listening for processes
// which are using the process_OK_message feedback for maestro.
// 
func (this *epollListener) listener() {
	var event syscall.EpollEvent
	ok_message_len := len(process_OK_message)
	ok_first_char := []rune(process_OK_message)[0]
//	var local_buf [ok_message_len*4]byte
	var local_buf = make([]byte,ok_message_len*4,ok_message_len*4)

	wakeup_fd, err := NewWakeupFd()
	if err != nil {
		// bad news!
		// 
		DEBUG_OUT("CRITICAL - could not create a FD for wakingup processMessageListener() thread. FAILING\n")
		return

	}
	this.wakeup_fd_messageListener = wakeup_fd
	// add in our wakeup FD
	event.Events = syscall.EPOLLIN
	event.Fd = int32(wakeup_fd.fd_read)	
	DEBUG_OUT("add FD from event_fd %d to epoll FD %d\n", wakeup_fd.fd_read,this.epfd)
	if e := syscall.EpollCtl(this.epfd, syscall.EPOLL_CTL_ADD, this.wakeup_fd_messageListener.fd_read, &event); e != nil {
		DEBUG_OUT("CRITICAL - could not add FD for waking up epoll - epoll_ctl: ", e)
		return
	}	
	// loop forever waiting for stuff
	DEBUG_OUT("epollListener FD %d - listener() loop starting\n",this.epfd)
	for {
		nevents, e := syscall.EpollWait(this.epfd, this.events[:], -1)
		if e != nil {
			DEBUG_OUT("ERROR - epoll_wait: ", e)
		}
		if this.shutdown_messageListener > 0 {
			break;
		}
		
		for ev := 0; ev < nevents; ev++ {
			if int(this.events[ev].Fd) == this.wakeup_fd_messageListener.fd_read {
				val, err2 := this.wakeup_fd_messageListener.ReadWakeup() // clear the wakeup
				if err2 == nil {
					if val == listener_shutdown_magic_num {
						DEBUG_OUT("epollListener listener() got shutdown magic num\n")
						this.shutdown_messageListener = 1
					} else {
						DEBUG_OUT("epollListener listener() got wakeup\n")
					}
				} else {
					DEBUG_OUT("ERROR on WakeupFd ReadWakeup() %s\n",err2.Error())
				}
			} else {
				// it must be a process we are waiting for it's OK string
				val, ok := this.fdToJob.GetHashedKey(uintptr(this.events[ev].Fd))
				if ok {
					process_ready := false
					job := (*processStatus)(val)
					var _p0 unsafe.Pointer
					currlen := len(job.startMessageBuf)
					remain := ok_message_len - currlen
					_p0 = unsafe.Pointer(&local_buf[0])
					r0, _, e1 := syscall.RawSyscall(syscall.SYS_READ, uintptr(this.events[ev].Fd), uintptr(_p0), uintptr(remain))
					n := int(r0)
					if e1 != 0 {
//						errno = e1
					} else {
						if n > 0 {
							offset := -1
							stringified := string(local_buf[:n])
							// fast path:
							if currlen == 0 && len(stringified) >= remain {								
								if stringified[:ok_message_len] == process_OK_message {	
									process_ready = true
								}
							} else {
							// longer path
								a := []rune(stringified) // convert to array of codepoints
								if currlen == 0 {
									for i, c := range a {
										if c == ok_first_char {
											offset = i
											break
										}
 									}
 									if offset == -1 {
 										// nothing, the output had no 'process_OK_message'
 										continue
 									}
 								} else {
 									offset = 0
 								}
								take := len(a)
								if take > remain {
									take = remain
								}
								job.startMessageBuf = append(job.startMessageBuf,a[offset:take]...)
 								if (len(job.startMessageBuf) == ok_message_len) && (string(job.startMessageBuf) == process_OK_message) {
 									process_ready = true
 								}
							}

							if process_ready {
								this.removeProcessFDListen(int(this.events[ev].Fd))
								// switch it over to stdout log capture and report ready:
								var originid uint32
								if job.originLabelId > 0 {
									originid = job.originLabelId
								} else {
									originid = greasego.GetUnusedOriginId()
								}
								greasego.AddFDForStdout(int(this.events[ev].Fd),originid)
								greasego.AddOriginLabel(job.originLabelId,job.job.GetJobName())
								job.status = RUNNING
								// submit an event that process is RUNNING
								DEBUG_OUT("Got process_ready (OK string) for %s\n",job.job.GetJobName())
								controlChan <- newProcessEvent(job.job.GetJobName(), internalEvent_job_running)
//								
							}

						}
					}
				} // if ok



			}

		}

	} // big for{}	
}


func (this *epollListener) removeProcessFDListen(fd int) (e error) {
	var event syscall.EpollEvent
	event.Events = syscall.EPOLLIN
	event.Fd = int32(fd)	
	this.fdToJob.DelHashedKey(uintptr(fd))
	if e = syscall.EpollCtl(this.epfd, syscall.EPOLL_CTL_DEL, fd, &event); e != nil {
		DEBUG_OUT("ERROR: epoll_ctl DEL: ", e)
	}
	// wakeup listener
	this.wakeup_fd_messageListener.Wakeup(1)
	return	
}

func (this *epollListener) addProcessFDListen(fd int, job *processStatus) (e error){
	var event syscall.EpollEvent
	event.Events = syscall.EPOLLIN
	event.Fd = int32(fd)	
	job.startMessageBuf = make([]rune,0,len(process_OK_message))
	this.fdToJob.SetHashedKey(uintptr(fd),unsafe.Pointer(job))
	if e = syscall.EpollCtl(this.epfd, syscall.EPOLL_CTL_ADD, fd, &event); e != nil {
		DEBUG_OUT("ERROR: epoll_ctl: ", e)
	}	
	// wakeup listener
	this.wakeup_fd_messageListener.Wakeup(1)
	return
}

func (this *epollListener) shutdownListener() {
	// wakeup listener
	this.wakeup_fd_messageListener.Wakeup(listener_shutdown_magic_num)

}

func init() {
	processIndexByPid = hashmap.New(10)
	compositeProcessesById = hashmap.New(10)

	atomic.StoreInt32(&monitorWorkerRunning,worker_stopped)

	containerTemplatesByName = hashmap.New(10)

	jobsByName = hashmap.New(10)

	controlChan = make(chan *processEvent, 100) // buffer up to 100 events
	re_this_dir = regexp.MustCompile(`\{\{thisdir\}\}`)
	re_cwd = regexp.MustCompile(`\{\{cwd\}\}`)
	
	folderPath, err2 := osext.ExecutableFolder()
	if err2 == nil {
		THIS_DIR = folderPath
	} else {
		fmt.Printf("ERROR - could not get maestro executable folder. {{thisdir}} macro will fail.")
	}
	updateMacroVars()
	// NEVER_RAN = iota
	// RUNNING = iota
	// STOPPED = iota
	// WAITING_FOR_DEPS = iota
	// STARTING = iota
	// STOPPING = iota
	// RESTARTING = iota
	// FAILED = iota	
	stringOfStatus = map[int]string{
		NEVER_RAN: "NEVER_RAN",
		RUNNING: "RUNNING",
		STARTING: "STARTING",
		STOPPING: "STOPPING",
		STOPPED: "STOPPED",
		WAITING_FOR_DEPS: "WAITING_FOR_DEPS",
		RESTARTING: "RESTARTING",
		FAILED: "FAILED",		
	}

}

// Should be called when a job successfully starts, named jobstarted
// to see if any other jobs were waiting on it as a dependency
// If there are jobs - then it sends events to start those.
func startStartableJobs(jobstarted string) {
	DEBUG_OUT2("startStartableJobs()--->\n")
	for job := range jobsByName.Iter() {
		val := job.Value
		DEBUG_OUT("ITER %+v\n",val)
		if val != nil {
			j := (*processStatus)(val)
			DEBUG_OUT2("ITER processStatus %+v\n",j)
			if(j.status == WAITING_FOR_DEPS) {
				StartJob(j.job.GetJobName())
			}
		}
	}


}



func _internalStartProcess(proc_status *processStatus, orig_event *processEvent, sendthis []byte, procIdentifier string) { //ev *processEvent,
	var opts *ExecFileOpts 
	var originatingJob maestroSpecs.JobDefinition

	// FIXME / TODO - note, this entire merging of options between container, job and composite ID / process
	// is a bit of a mess. We need to have a more organized approach here.

	originatingJob = proc_status.job
	procName := procIdentifier
	if originatingJob == nil {
		DEBUG_OUT("NOTE: don't have originating job\n")
		if orig_event != nil {
			originatingJob = orig_event.request
		} else {
			procLogErrorf("Malformed process in _internalStartProcess() - be advised. Don't have an originating Job.")
		}
	} else {
		procName = originatingJob.GetJobName()
	}
	// }
	if len(proc_status.compositeId) > 0  {
		opts = NewExecFileOpts("",proc_status.compositeId)
	} else {
		opts = NewExecFileOpts(procName,"")
	}
	DEBUG_OUT2("_internalStartProcess 1.1\n")
	if originatingJob == nil {
		opts.SetNewSid()
	} else {
		if originatingJob.IsDaemonize() {
			opts.SetNewSid()
		} else {
			DEBUG_OUT2("_internalStartProcess 2\n")
			if originatingJob.GetPgid() < 1 {
				opts.SetNewPgid()
			} else {
				opts.SetUsePgid(originatingJob.GetPgid())
			}
		}
	}
	if (proc_status.job != nil && proc_status.job.IsDieOnParentDeath()) || proc_status.dieOnParentDeath {
		proc_status.dieOnParentDeath = true
		opts.internal.die_on_parent_sig = 1
	}
	if proc_status.send_GREASE_ORIGIN_ID {
		opts.internal.env_GREASE_ORIGIN_ID = 1
	}
	env := proc_status.execEnv
	if originatingJob != nil && originatingJob.IsInheritEnv() {
		env = append(env,os.Environ()...)
	}
	DEBUG_OUT("_internalStartProcess 3\n")				
	var errno int
	var pid int
	args := proc_status.execArgs
	path := proc_status.execPath

	// there can not be both a message and the sendthis data.
	// sendthis is usually used for a composite process, and sending its configuration.
	// If this happens, then we will use sendthis, but throw a warning that message was set
	// 
	// setup stdin message if one is provided (otherwise just set to NULL)
	var msg string
	if originatingJob != nil {
		msg = originatingJob.GetMessageForProcess()
		if len(msg) > 0 {
			if len(sendthis) > 0 {
				procLogWarnf("NOTE: the .message option was configured for Job %s - but this Job is part of a composite process. The config object will take precendence.\n",proc_status.job.GetJobName())
			} else {
				opts.SetMessageString(msg)			
			}
		}
	}
	if len(sendthis) > 0 {
		opts.SetMessageString(string(sendthis))
	}
	// if it is a composite Job, use the composite Job ID, otherwise use the JobName
	if len(proc_status.compositeId) > 0 {
		opts.SetJobName(proc_status.compositeId)
	} else {
		opts.SetJobName(procName)
	}

	opts.SetOkString(proc_status.usesOKString)
	if(len(args) > 0) {
		pid, errno = ExecFile(path,args,env,opts)
	} else {
		pid, errno = ExecFile(path,nil,env,opts)
	}
	proc_status.originLabelId = opts.GetOriginLabelId()

	if errno != 0 {
		proc_status.status = FAILED
		proc_status.setAllChildJobStatus(FAILED)
		next_ev := newProcessEventFromExisting(orig_event,internalEvent_job_failed)
		controlChan <- next_ev
	} else {
		// status RUNNING or 
		proc_status.pid = pid
		if proc_status.usesOKString {
			// if there was a message to send, then we are waiting on an "!!OK!!"
			// from the process
			
			DEBUG_OUT("   Job: %s is STARTING\n",procName)
			// TODO add listener to mainEpollListener
			stdoutfd := opts.GetStdoutFd()
			if stdoutfd == 0 {
				DEBUG_OUT("CRITICAL - stdout FD for new process is zero!\n")
			} else {
				mainEpollListener.addProcessFDListen(stdoutfd, proc_status)
			}
			proc_status.status = STARTING
			proc_status.setAllChildJobStatus(STARTING)			

			ev:= new(ProcessEvent)
			ev.Name = "job_started"
			ev.Pid = pid
			ev.When = time.Now().UnixNano()
			if proc_status.job != nil {
				ev.Job = proc_status.job.GetJobName()				
			}
			// TODO make Job name for master process
			SubmitEvent(ev)
		} else {
			if (proc_status.deferRunningStatus > 0) {
				DEBUG_OUT("Deferring RUNNING status change for %s\n",procName)
				procLogDebugf("Deferring RUNNING status change for %s for %d ms\n",procName,proc_status.deferRunningStatus)
				go func(jobname string,pid int,proc_status *processStatus){
					time.Sleep(time.Millisecond * time.Duration(proc_status.deferRunningStatus))
					_setJobToRunning(jobname,pid,proc_status)
					procLogDebugf("Now setting RUNNING status change for %s for %d ms\n",procName,proc_status.deferRunningStatus)
					proc_status.setAllChildJobStatus(RUNNING)
					controlChan <- newProcessEvent(procName, internalEvent_job_running)
				}(procName,pid,proc_status)
			} else {
				_setJobToRunning(procName,pid,proc_status)
			}
		}
	}
}

func _setJobToRunning(jobname string,pid int,proc_status *processStatus) {
	proc_status.lockingSetStatus(RUNNING)
	DEBUG_OUT("   Job: %s is RUNNING\n",jobname)
	proc_status.setAllChildJobStatus(RUNNING)
	ev := new(ProcessEvent)
	ev.Name = "job_running"
	ev.Pid = pid
	if proc_status.job != nil {
		ev.Job = proc_status.job.GetJobName()				
	}
	ev.When = time.Now().UnixNano()
//			ProcessEvents.Push(ev)
	SubmitEvent(ev)
	if proc_status.job != nil {						
		startStartableJobs(proc_status.job.GetJobName())
	} else {
		startStartableJobs("") // FIXME - this works for now
	}

}


func _startCompositeProcessJobs(proc *processStatus, ev *processEvent) (err error) {
	templName := proc.job.GetContainerTemplate()
	var templStatus *containerStatus
	var masterP *processStatus
	childList := []*processStatus{}

	_innerStart := func() (err error) {

		configOut := []byte{}
		config := configMgr.NewCompositeProcessConfigPayload()

		if masterP.send_composite_jobs_to_stdin {

			for _, child := range childList {
				config.Moduleconfigs[child.job.GetJobName()] = child.job
			}

			config.ProcessConfig = templStatus.container.GetCompositeConfig()

			configOut, err = json.Marshal(config) 
			if err != nil {
				return
			}
		}

		if templStatus.container.IsDieOnParentDeath() {
			masterP.dieOnParentDeath = true
		}

		_internalStartProcess(masterP, ev, configOut, proc.compositeId)

		return
		// if send_composite_jobs_to_stdin, then JSON-ify and ready string
		// 
		// start process
		// emit start event for all Jobs it has

	}

	if len(templName) > 0 {

		val, ok := containerTemplatesByName.GetStringKey(templName)
		if ok {
			templStatus = (*containerStatus)(val)
		} else {
			joberr := new(JobError)
			joberr.Code = JOBERROR_MISSING_CONTAINER_TEMPLATE
			joberr.JobName = proc.job.GetJobName()
			joberr.Aux = "_startCompositeProcessJobs can't find container template: "+templName 
			err = joberr			
		}

		// see if the master process is already running...
		unsafep, ok := compositeProcessesById.GetStringKey(proc.compositeId)

		if ok {

			masterP = (*processStatus)(unsafep)
			if masterP.compositeContainerTemplate != templName {
				joberr := new(JobError)
				joberr.Code = JOBERROR_INVALID_CONFIG
				joberr.JobName = proc.job.GetJobName()
				joberr.Aux = "_startCompositeProcessJobs attempting to start jobs with differing container templates" 
				err = joberr
			}

			// find all jobs which use this composite ID
			for el := range jobsByName.Iter() {
				if el.Value != nil {
					child := (*processStatus)(el.Value)
					if child.compositeId == proc.compositeId {
						childList = append(childList,child)
					}
				}
			}

			_innerStart()
			// if it is running, then shut it down.
			// emit event for all jobs that were shutdown
			// upon succesful shutdown, start composite process again

			// _innerStart()


		} else {
			joberr := new(JobError)
			joberr.Code = JOBERROR_INTERNAL_ERROR
			joberr.JobName = proc.job.GetJobName()
			joberr.Aux = "Could not find master process for given compositeId - _startCompositeProcessJobs()." 
			err = joberr			
		}

		

	} else {
		joberr := new(JobError)
		joberr.Code = JOBERROR_INVALID_CONFIG
		joberr.JobName = proc.job.GetJobName()
		joberr.Aux = "Existing composite process record's container template does not match this child job." 
		err = joberr
	}
	return
}

// this threads does the following:
// - handles internal events as they arrive in the controlChan channel
// - start / stops jobs
// - process periodic stats
// - submit outbound events for these things
// - accepts a shutdown command for shutdown
// 
// It is built as one thread, b/c in a some cases, we need ordered execution
// Secondly, we don't want to consume too many resources on process startup
func processEventsWorker() {
//	var counter uint32
//	counter = 0
	var err error
	mainEpollListener, err = newEpollListener()
	if err != nil {
		DEBUG_OUT("CRITICAL: could not create epoll for epollListener: %s\n",err.Error())
		DEBUG_OUT("processEventsWorker() WILL FAIL\n")
		return
	}
	go mainEpollListener.listener()

	eventLoop:
	for true {
		DEBUG_OUT("************************************* maestro processEventsWorker() ********************************\n")
		select {
		// MOVED TO sysstats package
		// case <-internalTicker.C:
		// 	counter++
		// 	DEBUG_OUT("COUNTER ---------------------------> : %d\n",counter)
		// 	// check intervals
		// 	// 
		// 	if statsConfig != nil {
		// 		pace, ok := statsConfig.GetConfig_CheckMem()
		// 		if ok && (counter % pace == 0) {
		// 			DEBUG_OUT("   **** Check mem stats")
					
		// 			stats, err := mem.VirtualMemory()
		// 			var ev *MemStatEvent
		// 			if err == nil {
		// 				ev = NewVirtualMemEvent(stats)						
		// 			} else {
		// 				ev = NewVirtualMemEvent(nil)						
		// 			}
		// 			SubmitEvent(*ev)
		// 		}

		// 	}
		case ev := <-controlChan:
			DEBUG_OUT("************************>>>>>>>>>> Got internal event: %+v\n", ev)
			if ev.code == internalEvent_shutdown {
//				continue
				break eventLoop
			}
			if ev.code == internalEvent_start_job {
				DEBUG_OUT("************************>>>>>>>>>> Got internal event: internalEvent_start_job\n")
				var proc_status *processStatus
				val, ok := jobsByName.GetStringKey(ev.jobname)
				if ok {
					proc_status = (*processStatus)(val)
				DEBUG_OUT2("internal event 0\n")
				DEBUG_OUT("        *** for job: %s\n",proc_status.job.GetJobName())

				// TODO - need a version / uuid for a particular Job start. if it changes
				// afterwards, then it shold be modified

				// check for job already trying to start...
					proc_status.lock()
					if proc_status.status == RUNNING || proc_status.status == STARTING || proc_status.status == STOPPING {
						proc_status.unlock()
						DEBUG_OUT("    Oop. Job %s is starting/running/stopping\n",proc_status.job.GetJobName())
						procLogWarnf("Maestro is ignoring a start job request [%s], b/c the job is current starting/stopping/running",proc_status.job.GetJobName())
						startStartableJobs(proc_status.job.GetJobName())
						continue eventLoop
					}
					proc_status.unlock()

				} else {
					proc_status = new(processStatus)
					jobsByName.Set(ev.jobname,unsafe.Pointer(proc_status))
				}

				DEBUG_OUT2("internal event 1\n")

				// update status with the latest job request
				// if the new start process event has a request
				// (if its a restart, it will not)
				if ev.request != nil {
					proc_status.job = ev.request					
				}

				// mark it as STARTING now
				// in case we do any deferred work
				proc_status.lockingSetStatus(STARTING)
				if len(proc_status.compositeId) > 0 {
					DEBUG_OUT("proc_status here---> %+v %+v\n", proc_status, proc_status.job)
					// this is a composite process. So multiple jobs per process.
					_startCompositeProcessJobs(proc_status, ev)

				} else {
					_internalStartProcess(proc_status, ev, nil, "")
				} // end if len(proc_status.compositeId) > 0

			}

			if ev.code == internalEvent_stop_job {
				// TODO:
				// -> mark the job as being stopped
				// -> kill Job based on existing PID
				// -> Kill any dependencies of the process
				// 


			}

			if ev.code == internalEvent_job_controlledExit {
				procLogErrorf("Job %s saw expected exit.",ev.jobname)
			}

			if ev.code == internalEvent_job_died {
				name := ev.jobname
				DEBUG_OUT("************************>>>>>>>>>> Got internal event - job_died: %s\n", name)
				procLogErrorf("Job %s died unexpectedly.",name)
				// check Job settings to see if a restart should be done 
				// and how many restarts remain
				val, ok := jobsByName.GetStringKey(ev.jobname)
				if ok {
					proc_status := (*processStatus)(val)
					if(proc_status.job.IsRestart()) {
						DEBUG_OUT("job was restart-able: %s\n",name)
						if proc_status.job.GetRestartLimit() == 0 {
							proc_status.restarts++
							// no limit on amount of restarts
							if proc_status.job.GetRestartPause() > 0 {
								go func(jobname string,proc_status *processStatus){
									procLogDebugf("Waiting %dms until asking for restart of Job %s",proc_status.job.GetRestartPause(),name)
									time.Sleep(time.Millisecond * time.Duration(proc_status.job.GetRestartPause()))
									procLogSuccessf("Asking for restart on Job %s (restarts: %d - no limit)",name,proc_status.restarts)									
									DEBUG_OUT("sending _start_job event for Job: %s\n",jobname)
									controlChan <- newProcessEvent(jobname, internalEvent_start_job)
								}(name,proc_status)
							} else {
								procLogSuccessf("Asking for restart on Job %s (restarts: %d - no limit)",name,proc_status.restarts)
								controlChan <- newProcessEvent(name, internalEvent_start_job)
							}
						} else {
							proc_status.restarts++
							if proc_status.restarts > proc_status.job.GetRestartLimit() {
								procLogWarnf("Job %s has hit its quota on restarts. All restart requests will be ignored.",name)
							} else {
								if proc_status.job.GetRestartPause() > 0 {
									go func(jobname string,proc_status *processStatus){
										procLogDebugf("Waiting %dms until asking for restart of Job %s",proc_status.job.GetRestartPause(),name)
										time.Sleep(time.Millisecond * time.Duration(proc_status.job.GetRestartPause()))
										procLogSuccessf("Asking for restart on Job %s (restarts: %d - limit:%d)",name,proc_status.restarts,proc_status.job.GetRestartLimit())
										DEBUG_OUT("sending _start_job event for Job: %s\n",jobname)
										controlChan <- newProcessEvent(jobname, internalEvent_start_job)
									}(name,proc_status)
								} else {
									procLogSuccessf("Asking for restart on Job %s (restarts: %d - limit:%d)",name,proc_status.restarts,proc_status.job.GetRestartLimit())
									controlChan <- newProcessEvent(name, internalEvent_start_job)
								}
							}
						}
					} else {
						procLogWarnf("Job %s is not configured for restart.",name)
					}
				} else {
					procLogErrorf("Internal error: Job %s is not in Maestro's job table but previously was.",name)
				}	
			}

			if ev.code == internalEvent_job_running {
				val, ok := jobsByName.GetStringKey(ev.jobname)
				if ok {
					proc_status := (*processStatus)(val)
					proc_status.status = RUNNING

					ev2 := new(ProcessEvent)
					ev2.Name = "job_running"
					ev2.Pid = proc_status.pid
					ev2.Job = proc_status.job.GetJobName()
					ev2.When = time.Now().UnixNano()
		//			ProcessEvents.Push(ev)
					SubmitEvent(ev2)
				}				
				startStartableJobs(ev.jobname)	
				DEBUG_OUT2("---------post startStartableJobs()")
				continue
			}

		}
	}

	DEBUG_OUT("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! LEFT processEventsWorker")
	atomic.StoreInt32(&monitorWorkerRunning,worker_stopped)
}

// only ever gets a close() - a close will be broadcast to
// all listeners (unlike a normal channel message)
var processMgmtShutdownNotifier chan interface{}

func periodicReaper() {	
	shutdown := false
	for !shutdown {
		select {
			case <-time.After(time.Millisecond * time.Duration(globalConfig.GetReaperInterval())):
				DEBUG_OUT("processMgmt: periodicReaper() looped\n")
				pid, status :=  ReapChildren()
				if pid > 0 {
					procLogSuccessf("periodicReaper() -- Saw closed process: %d %d\n", pid, status)		
				}			
			case _ = <- processMgmtShutdownNotifier:
				shutdown = true
		}
	}
}

func StartEventsMonitor() {
	ok := atomic.CompareAndSwapInt32(&monitorWorkerRunning,worker_stopped,worker_started)
	if ok {
		go processEventsWorker()		
	}
}

func StopEventsMonitor() {
	controlChan <- newShutdownEvent()
}

var globalConfig *configs.ProcessesConfig

func InitProcessMgmt(config *configs.ProcessesConfig) {
	if config != nil {
		globalConfig = config
	} else {
		globalConfig = configs.ProcessesDefaults
	}
	processMgmtShutdownNotifier = make(chan interface{})
	go periodicReaper()
}

// called after config file is read
func InitProcessEvents(config maestroSpecs.StatsConfig) {
	// fmt.Printf("Interval: %d\n",config.GetInterval())
	// interval, ok := config.GetConfig_CheckMem()
	// if ok {
	// 	fmt.Printf("check mem pace: %d\n",interval)			
	// } else {
	// 	fmt.Printf("don't check mem\n")
	// }

	statsConfig = config

	internalTicker = time.NewTicker(time.Second * time.Duration(config.GetInterval()))

	StartEventsMonitor()
}

// these are defined in waitflags.h normally
const const_WNOHANG = 1
const const_WUNTRACED = 2
const const_WCONTINUED = 8


func convertToCStrings(args []string) (out **C.char) {
//	out = make([]*C.char, len(args))
	if len(args) < 1 {
		return nil
	}
	out = C.makeCStringArray(C.int(len(args)+3)) // why 3? we leave two extra in case we need to add one or two 
	                                             // in the createChild function
	for n, s := range args {
		if len(s) < 1 {
			C.setCStringInArray(out,nil,C.int(n))
			break
		} else {
			C.setCStringInArray(out,C.CString(s),C.int(n))
		}
	}
	// always set the final string to nil, to mark the end of the array
	// of strings for the C code
	C.setCStringInArray(out,nil,C.int(len(args)))
	return
}

func freeCStringArray(in []*C.char) {

}

// does the work of replacing:
// {{thisdir}}  - directory of the maestro executable file
// ... perhaps more added later
func replaceMacroVars(s string) (ret string) {
	ret = re_this_dir.ReplaceAllString(s, THIS_DIR)
	ret = re_cwd.ReplaceAllString(ret, CWD)	
	return
}

// 
func replaceMacroVarsSlice(s []string, out *[]string) {
	for _, v := range s {
//		DEBUG_OUT("replace: %s\n",v)
//		DEBUG_OUT("replace: %s\n",replaceMacroVars(v))
		*out = append(*out,replaceMacroVars(v))
	}
}

func ExecFile(path string, args []string, env []string, opts *ExecFileOpts) (pid int, errno int) {
	var err C.execErr
	err._errno = 0

	DEBUG_OUT("ExecFile: %+v %+v %+v\n",replaceMacroVars(path),args,env)


	updateMacroVars()
	ready_path := replaceMacroVars(path)
	c_path := C.CString(ready_path)

	ready_args := []string{ready_path}
	var ready_env []string
	// arg 0 should always be the file name we are executing.
	// See: https://linux.die.net/man/2/execve
	replaceMacroVarsSlice(args,&ready_args)
	replaceMacroVarsSlice(env,&ready_env)

	c_args := convertToCStrings(ready_args)
	c_env := convertToCStrings(ready_env)

	DEBUG_OUT("READY ExecFile: %+v %+v %+v\n",replaceMacroVars(path),ready_args,ready_env)

	// NOTE: createChild will automically free the strings handed to it.
	if opts != nil {
		C.createChild(c_path,c_args,c_env,opts.internal.message,&opts.internal,&err)
	} else {
		C.createChild(c_path,c_args,c_env,nil,nil,&err)
	}

	pid = int(err.pid)
	errno = int(err._errno)

	// TODO if errno is 0, add to tracking table
	if errno != 0 {
		// See: https://golang.org/src/syscall/syscall_unix.go?s=2532:2550#L91
		_errno := syscall.Errno(errno) // which is a uintptr
		procLogErrorf("Process failed to start: %d %s - Job: %s",errno, _errno.Error(), opts.jobName)
		DEBUG_OUT("ERROR ExecFile: %d %s - Job: %s",errno, _errno.Error(), opts.jobName)
		// ok - if the error is something which will continue to happen, then we should 
		// not attempt to restart the process, even if the restart param is true

		// TODO - do something with errors
		// FIXME need to submit error events


	} else {
		record, _ := TrackNewProcessByPid(pid)
		record.Job = opts.jobName
		record.CompId = opts.compId
		record.Path = path
		record.Started = time.Now()
		procLogSuccessf("Process started. PID:%d Job:%s CompId:%s",pid,record.Job,record.CompId)
		ev:= new(ProcessEvent)
		ev.Name = "process_started"
		ev.Pid = pid
		ev.When = record.Started.UnixNano()
		SubmitEvent(ev)
	}
	return
}


//export sawClosedRedirectedFD
func sawClosedRedirectedFD() {
	IFDEBUG({{*pid, status := *}},) ReapChildren()

	IFDEBUG(if(pid > 0) {,)
		DEBUG_OUT("Saw closed process: %d %d\n", pid, status)		
	IFDEBUG(},)


}

// attempts to reap a child, but does not block
// a valid PID is returned if a reap happened, otherwise 0
func ReapChildren() (ret int, status unix.WaitStatus) {
	ret = 0
//	usage := new(unix.Rusage)

	stop := 2

	mainReapLoop:
	for stop > 0  {
		pid, err := unix.Wait4(-1, &status,const_WNOHANG|const_WUNTRACED|const_WCONTINUED,nil) //usage)
		DEBUG_OUT("ReapChildren() = %d %d\n", pid,err)
		if err == nil || err == syscall.Errno(0) {
			ret = pid
			if pid > 0 {
				now := time.Now()
				record, ok := GetTrackedProcessByPid(pid)
				if ok {
					if status.Exited() {
						ret = pid
						ev := new(ProcessEvent)
						record.ExitValue = status.ExitStatus()
						ev.ExitVal = record.ExitValue
						procLogDebugf("Process was reaped. A tracked process shutdown. PID %d - Job [%s] - exit val %d",pid,record.Job,record.ExitValue)
						ev.Name = "process_ended"
						ev.Pid = pid
						ev.When = now.UnixNano()
						record.Stopped = now
			//			ProcessEvents.Push(ev)
						SubmitEvent(ev)
					}
					DEBUG_OUT("ReapChildren(): saw signal: %d\n",status.Signal())				
					// For exit status defs, see: https://golang.org/src/syscall/syscall_linux.go?s=5302:5335#L203
					exited := status.Exited()

					// Bear in mind, a process could die - from say a 'kill' command, in which 
					// case we would be signaled, but Exited() would not return true.
					// Confusing, yes. In any case here are the gophers discussing this:
					// https://github.com/golang/go/issues/19798

					if status.Signaled() {
						switch status.Signal() {
						case syscall.SIGTERM:
							procLogWarnf("Process %d saw signal SIGTERM",pid)
							DEBUG_OUT("Process %d saw signal SIGTERM\n",pid)
							stop = 0
							exited = true
						case syscall.SIGKILL:
							procLogErrorf("Process %d saw signal SIGKILL",pid)
							DEBUG_OUT("Process %d saw signal SIGKILL\n",pid)
							stop = 0
							exited = true
						default:
							// some other signal, so whatever
							// May be SIGCHLD, or may be other values
							// NOTE: on Arm kernels, the expected value seems to change
							procLogWarnf("Process %d saw signal 0x%x",pid,status.Signal())
							DEBUG_OUT("Process %d saw signal %x\n",pid,status.Signal()&0xFF)
//							continue mainReapLoop
						}
					}					
					// if exited || status.Signaled() { //   ?? -->  || status.CoreDump() {


					// only do any of this, if there was a real exit
					// not just that we saw a signal
					if exited {
						status_p, ok2 := jobsByName.GetStringKey(record.Job)
						if ok2 {
							status := (*processStatus)(status_p)
							status.lock()
							var internalEv *processEvent
							if status.status == STOPPING {
								// expected death
								internalEv = newProcessEvent(record.Job, internalEvent_job_controlledExit)
							} else {
								// unexpected death
								internalEv = newProcessEvent(record.Job, internalEvent_job_died)
							}
							status.status = STOPPED
							status.unlock()
							// delete ProcessRecord. Process is gone.
							RemoveTrackedProcessByPid(pid)			
							controlChan <- internalEv
						} else {
							// ok - maybe its a composite process instead:
							status_p, ok2 := compositeProcessesById.GetStringKey(record.CompId)							
							if ok2 {
								DEBUG_OUT("Process PID %d was a composite process. Notifying all internal jobs.\n",pid)
								procLogWarnf("Process PID %d was a composite process. Notifying all internal jobs.\n",pid)
								//  Lookup in Composite table instead
								status := (*processStatus)(status_p)
								DEBUG_OUT("AT LOCK - comp process\n")
								status.lock()
								DEBUG_OUT("PAST LOCK - comp process\n")
								//								var internalEv *processEvent
								if status.status == STOPPING {
									// expected death
									status.sendEventAndSetStatusForCompProcess(STOPPED,internalEvent_job_controlledExit)
//									internalEv = newProcessEvent(record.Job, internalEvent_job_controlledExit)
								} else {
									// unexpected death
									status.sendEventAndSetStatusForCompProcess(STOPPED,internalEvent_job_died)
									// status.sendEventForAllChildren(internalEvent_job_died)
//									internalEv = newProcessEvent(record.Job, internalEvent_job_died)
								}
								status.unlock()
								DEBUG_OUT("PAST UNLOCK - comp process\n")
								// delete ProcessRecord. Process is gone.
								RemoveTrackedProcessByPid(pid)
//								controlChan <- internalEv
							} else {
								stop = 0
								procLogWarnf("Process was reaped PID %d -> A record was found but no Job / Compid entry was found in the processManager for %s",pid,record.Job)
								DEBUG_OUT("ReapChildren(): Process was reaped PID %d -> A record was found but no Job / Compid entry was found in the processManager for %s\n",pid,record.Job)
							}
						}
					} else {
						procLogDebugf("Ignoring process PID %d - signal %d - child did not exit.", pid, status.Signal())
						continue mainReapLoop
					}
					// } else {
					// 	DEBUG_OUT("ReapChildren(): Process was reaped PID %d -> Status was something else")
					// 	stop--
					// }
				} else {
					procLogWarnf("Process was reaped. A un-tracked process shutdown. PID %d w/ exit val %d",pid,status.ExitStatus())
					DEBUG_OUT("ReapChildren(): Process was reaped. A un-tracked process shutdown. PID %d w/ exit val %d\n",pid,status.ExitStatus())
					stop--
				}
			} else {
				stop--
			}			
		} else {
			stop--
			DEBUG_OUT("ReapChildren(): Skipped main ReapChildren handler %s\n",err.Error())
			ret = 0
		}
	}
	return
}

// returns the equivalent of a job.GetDependsOn() but for a 
// 'master process' which is identified by a composite ID vs. 
// a Job name
func getMasterProcessDepends( compositeid string ) (ok2 bool, ret []string) {
	DEBUG_OUT("StartJob getMasterProcessDepends %s\n",compositeid)
	if len(compositeid) > 0 {
		existing, ok := compositeProcessesById.GetStringKey(compositeid)
		if ok {
			masterP := (*processStatus)(existing)
			for dep, _ := range masterP.combinedDeps {
				ret = append(ret,dep)
			}
			ok2 = true
			DEBUG_OUT("StartJob getMasterProcessDepends ret = %+v\n",ret)
		} else {
			DEBUG_OUT("StartJob getMasterProcessDepends not in table!  %s\n",compositeid)
		}
	}
	return
}

// this finds or creates a master process processStatus object and stores it in the 
// compositeProcessesById table. It then adds the child to the list of Jobs in the master Process It does not start the master process.
func _findMasterProcessAddChild( child maestroSpecs.JobDefinition ) (masterP *processStatus, reterr error) {
	compId := child.GetCompositeId()
	childTemplate := child.GetContainerTemplate()

	addDeps := func(child maestroSpecs.JobDefinition, templ maestroSpecs.ContainerTemplate, masterP *processStatus) {
		if masterP.combinedDeps == nil {
			masterP.combinedDeps = make(map[string]bool)			
		}
		if templ != nil {
			deps := templ.GetDependsOn()
			for _, dep := range deps {
				masterP.combinedDeps[dep] = true
			}
		}
		deps := child.GetDependsOn()
		for _, dep := range deps {
			masterP.combinedDeps[dep] = true
		}
	}

	if(len(compId) > 0 && len(childTemplate) > 0) {

		existing, ok := compositeProcessesById.GetStringKey(compId)
		if ok {
			masterP = (*processStatus)(existing)

			templName := masterP.compositeContainerTemplate
			templP, ok2 := containerTemplatesByName.GetStringKey(templName)
			
			if !ok2 {
				err := new(JobError)
				err.Code = JOBERROR_INTERNAL_ERROR
				err.JobName = child.GetJobName()
				err.Aux = "Composite proces record exists, but it's container template does not?? Can't attach child job: " + child.GetJobName()
				reterr = err
				return
			}

			templ := (*containerStatus)(templP)	
			if templName != childTemplate {
				err := new(JobError)
				err.Code = JOBERROR_INVALID_CONFIG
				err.JobName = child.GetJobName()
				err.Aux = "Existing composite process record's container template does not match this child job." 
				reterr = err
				return
			}

			addDeps(child,templ.container,masterP)

			templ.uses++

			masterP.lock()
			_, inlist := masterP.childNames[child.GetJobName()]
			if !inlist {
				masterP.childNames[child.GetJobName()] = NEVER_RAN			
			}
			// error if already there?
			masterP.unlock()
		} else {
			// in this case, no Job using this particular compositeID has been seen yet
			
			val, ok := containerTemplatesByName.GetStringKey(childTemplate)
			if !ok {
				err := new(JobError)
				err.Code = JOBERROR_MISSING_CONTAINER_TEMPLATE
				err.JobName = child.GetJobName()
				err.Aux = "Composite process + child Job needed container '"+childTemplate+"'"
				reterr = err
				return
			} else {	
				templ := (*containerStatus)(val)
				templ.uses++

				masterP = new(processStatus)
				// the master process get's the container template the child says its using
				masterP.compositeContainerTemplate = childTemplate
				masterP.status = NEVER_RAN
				masterP.compositeId = compId
				masterP.childNames = make(map[string]int)
				masterP.send_composite_jobs_to_stdin = templ.container.IsSendCompositeJobsToStdin()
				masterP.send_GREASE_ORIGIN_ID = templ.container.IsSendGreaseOriginIdEnv()

				addDeps(child,templ.container,masterP)

				templ.uses++
				exec_cmd := templ.container.GetExecCmd()
				if len(exec_cmd) > 0 {
					masterP.execArgs = append([]string{child.GetExecCmd()}, child.GetArgs()...)
					masterP.execPath = exec_cmd
				} else {
					err := new(JobError)
					err.Code = JOBERROR_INVALID_CONFIG
					err.JobName = child.GetJobName()
					err.Aux = "ContainerTemplate used by Job's with compositeId must have complete exec commands."
					reterr = err
					return
				}
				masterP.execArgs = append(templ.container.GetPreArgs(),masterP.execArgs...)
				masterP.execArgs = append(masterP.execArgs,templ.container.GetPostArgs()...)
				masterP.execEnv = append(masterP.execEnv,templ.container.GetEnv()...)
				masterP.usesOKString = templ.container.UsesOkString()
				masterP.deferRunningStatus = templ.container.GetDeferRunningStatus()
				masterP.dieOnParentDeath = templ.container.IsDieOnParentDeath()
				// record that this JobName is a child
				masterP.childNames[child.GetJobName()] = NEVER_RAN
				// add to our master table
				compositeProcessesById.Set(compId,unsafe.Pointer(masterP))
				
				// have not seen this composite ID before...
				DEBUG_OUT("New composited ID for process seen: %s\n",child.GetCompositeId())
			}
		}	
		return
	} else {
		err := new(JobError)
		err.Code = JOBERROR_INVALID_CONFIG
		err.JobName = child.GetJobName()
		err.Aux = "Composite Process: A CompositeID was stated for the process, but it has no ContainerTemplate. Template required."
		reterr = err
		return	
	}
}

func RegisterMutableJob( job maestroSpecs.JobDefinition ) (err error) {
	if job.IsMutable() {
		DB := storage.GetStorage()
		err = DB.UpsertJob( job )
		if err != nil {
			DEBUG_OUT("ERROR Error on saving job in DB: %s\n", err.Error())
			procLogErrorf("Error on saving job in DB: %s\n", err.Error())
		}			
	} else {
		DEBUG_OUT("Immutable: Not storing config file Job: %s\n",job.GetJobName())			
	}
	err = RegisterJobOverwrite( job )
	return
}

func _getJobDeps(job maestroSpecs.JobDefinition) (ret []string) {
	depmap := make(map[string]bool)
	templname := job.GetContainerTemplate()
	val, ok := containerTemplatesByName.GetStringKey(templname)	
	if ok {
		templ := (*containerStatus)(val)
		if templ != nil {
			deps := templ.container.GetDependsOn()
			for _, dep := range deps {
				depmap[dep] = true
			}
		}
	}
	deps := job.GetDependsOn()
	for _, dep := range deps {
		depmap[dep] = true
	}
	for dep, _ := range depmap {
		ret = append(ret,dep)
	}
	return
}

func RegisterJobOverwrite( job maestroSpecs.JobDefinition ) (err error) {
	entry := new(processStatus)
	entry.job = job
	entry.status = NEVER_RAN
	name := job.GetJobName()

	DEBUG_OUT("RegisterJobOverwrite(%s)\n",name)
	jobsByName.Set(name,unsafe.Pointer(entry))

	node := newDependencyNode(name,_getJobDeps(job)...)
	globalDepGraph = append(globalDepGraph,node)
	graphValidated = false
	return
}


func RegisterJob( job maestroSpecs.JobDefinition ) (err error) {
	entry := new(processStatus)
	entry.job = job
	entry.status = NEVER_RAN
	name := job.GetJobName()
	_, ok := jobsByName.GetStringKey(name)
//	_, ok := jobsByName[name]
	if ok {
		joberr := new(JobError)
		joberr.Code = JOBERROR_DUPLICATE_JOB
		joberr.JobName = name
		err = joberr
	} else {
		DEBUG_OUT("RegisterJob(%s)\n",name)
		jobsByName.Set(name,unsafe.Pointer(entry))
//		jobsByName[name] = entry		
	}
	node := newDependencyNode(name,job.GetDependsOn()...)
	globalDepGraph = append(globalDepGraph,node)
	graphValidated = false
	return
}


func RegisterContainer( cont maestroSpecs.ContainerTemplate ) (err error) {
	entry := new(containerStatus)
	entry.container = cont
	name := cont.GetName()
	_, ok := containerTemplatesByName.GetStringKey(name)
	if ok {
		cerr := new(ContainerError)
		cerr.Code = CONTAINERERROR_DUPLICATE
		cerr.ContainerName = name
		err = cerr
	} else {
		containerTemplatesByName.Set(name,unsafe.Pointer(entry))
	}
	return
}

type _jobStatusData struct {
	jobsByName *hashmap.HashMap
}

type JobStatusData interface {
	MarshalJSON() ([]byte, error)
}

func (data *_jobStatusData) MarshalJSON() (outjson []byte, err error)  {
	buffer := bytes.NewBufferString("{")

	// loop through all entries.
	// output format is:
	// { 
	//    "someJobName" : {
	//    { 
	//	     // all the job data fields here...
	//    },
	//    // each job...		
	// }
	n := 0
	for job := range data.jobsByName.Iter() {
		val := job.Value
		key := job.Key
		DEBUG_OUT("ITER %+v\n",val)
		if n>0 {
			buffer.WriteString(",")
		}
		if val != nil {
			j := (*processStatus)(val)
			name, ok2 := key.(string) 
			if ok2 {
				buffer.WriteString(fmt.Sprintf("\"%s\":{",name))
				statusS := stringifyJobStatus(j.status)
				buffer.WriteString(fmt.Sprintf("\"status\":\"%s\",\"def\":",statusS))
				jobDef, ok := j.job.(*maestroSpecs.JobDefinitionPayload)
				if ok {
					jsonV, err := json.Marshal(jobDef)
					if err != nil {
						buffer.WriteString(fmt.Sprintf("\"error decoding\","))
					} else {
						buffer.WriteString(fmt.Sprintf("%s,",string(jsonV)))
					}
				} else {
					buffer.WriteString(fmt.Sprintf("null,"))
				}
				buffer.WriteString(fmt.Sprintf("\"restarts\":\"%d\",",j.restarts))
				buffer.WriteString(fmt.Sprintf("\"pid\":\"%d\",",j.pid))
				buffer.WriteString(fmt.Sprintf("\"errno\":\"%d\",",j.errno))
				if len(j.compositeId) > 0 {
					buffer.WriteString(fmt.Sprintf("\"compositeId\":\"%s\",",j.compositeId))
				}
				if len(j.compositeContainerTemplate) > 0 {
					buffer.WriteString(fmt.Sprintf("\"compositeContainerTemplate\":\"%s\",",j.compositeContainerTemplate))
				}
	
				buffer.WriteString(fmt.Sprintf("\"execPath\":\"%s\"",j.execPath))
				// if(j.status == WAITING_FOR_DEPS) {
						
				// }
				buffer.WriteString("}")	
			}
		}
		n++
	}

	buffer.WriteString("}")
	outjson = buffer.Bytes()
	return
}

func stringifyJobStatus(val int) (ret string) {
	ret = "UNKNOWN"
	ret, _ = stringOfStatus[val]
	return
}

func GetJobStatus() (data JobStatusData, err error) {
	data = &_jobStatusData{
		jobsByName:jobsByName,
	}
	return
}



func ValidateJobs() error {
	// validate that dependencies exist for all jobs which are registered
	outGraph, err := resolveDependencyGraph(globalDepGraph, func(name string) bool {
		_, ok := jobsByName.GetStringKey(name)
		return ok
	})
	if err == nil {
		globalDepGraph = outGraph
	} else {
		return err
	}
	// validate that ContainerTemplates exist for all jobs which reference on
	for job := range jobsByName.Iter() {
		val := job.Value
		if val != nil {
			j := (*processStatus)(val)
			template := j.job.GetContainerTemplate()
			j.compositeId = j.job.GetCompositeId()
			confname := j.job.GetConfigName()
			conf := j.job.GetStaticConfig()
			if len(conf) > 0 {
				confobj := new(maestroSpecs.ConfigDefinitionPayload)
				confobj.Name = confname
				confobj.Job = j.job.GetJobName()
				confobj.Data = conf
				confobj.Encoding = "utf8" // if the config comes in this way, its always 'utf8' (for now)
				// store the config
				configMgr.JobConfigMgr().SetConfig(confname,j.job.GetJobName(),confobj)
			}
			msg := j.job.GetMessageForProcess()
			if(len(j.compositeId) > 0 && len(msg) > 0) {
				err := new(JobError)
				err.Code = JOBERROR_INVALID_CONFIG
				err.JobName = j.job.GetJobName()
				err.Aux = "can't set both .compositeId (config composite ID) and .message (message for process) at the same time"
				return err				
			}

			if(len(j.compositeId) > 0) {
				// if this belongs to a composite ID, then it shares a processStatus with
				// multiple jobs
				if(len(template) > 0) {
					// check to make sure container type matches existing master Process container type
					_, err := _findMasterProcessAddChild(j.job)
					if err != nil {
						return err
					}

				} else {
					err := new(JobError)
					err.Code = JOBERROR_INVALID_CONFIG
					err.JobName = j.job.GetJobName()
					err.Aux = "A CompositeID was stated for the process, but it has no ContainerTemplate. Template required."
					return err
				}
			} else {
				// otherwise, this is a single process.
				j.execPath = j.job.GetExecCmd()
				// for arg := range j.job.GetArgs() {
				// 	j.execArgs = append(j.execArgs,arg)
				// }
				j.execArgs = append(j.execArgs,j.job.GetArgs()...)
				// for env := range j.job.GetEnv() {
				j.execEnv = append(j.execEnv,j.job.GetEnv()...)
				// }
				j.dieOnParentDeath = j.job.IsDieOnParentDeath()
				j.usesOKString = j.job.UsesOkString()
				j.deferRunningStatus = j.job.GetDeferRunningStatus()
				if len(template) > 0 {
					val, ok := containerTemplatesByName.GetStringKey(template)
					if !ok {
						err := new(JobError)
						err.Code = JOBERROR_MISSING_CONTAINER_TEMPLATE
						err.JobName = j.job.GetJobName()
						err.Aux = "needed container '"+template+"'"
						return err
					} else {	
						templ := (*containerStatus)(val)
						templ.uses++
						exec_cmd := templ.container.GetExecCmd()
						if len(exec_cmd) > 0 {
							j.execArgs = append([]string{j.execPath}, j.execArgs...)
							j.execPath = exec_cmd
						}
						j.execArgs = append(templ.container.GetPreArgs(),j.execArgs...)
						j.execArgs = append(j.execArgs,templ.container.GetPostArgs()...)
						j.execEnv = append(j.execEnv,templ.container.GetEnv()...)
						j.usesOKString = templ.container.UsesOkString()
						if templ.container.GetDeferRunningStatus() > 0 {
							j.deferRunningStatus = templ.container.GetDeferRunningStatus()							
						}
					}
				}
			}

			if (j.deferRunningStatus > 0 && j.usesOKString) {
				procLogWarnf("Note - defer_running_status and uses_ok_string are incompatible. Only one will be used.\n")
			}

			// JobDefinitionPayload.Config versus .Message
			// If a JobDefinition has a Config, then it must:
			// - It's .Message becomes [CONFIG]ENDCONFIG
			// where "[CONFIG]" is the actual byte stream of the .Config
			// It can not also have a .Message

			// check for composite Processes - the only supported Composite Process
			// is one which uses a container type of 'deviceJS_process' or CONTAINER_DEVICEJS_PROCESS_NODE8

			// if(len(j.job.GetCompositeId()) > 0 && j.job.GetContainerTemplate() != maestroSpecs.CONTAINER_DEVICEJS_PROCESS) {
			// 	err := new(JobError)
			// 	err.Code = JOBERROR_INVALID_CONFIG
			// 	err.JobName = j.job.GetJobName()
			// 	err.Aux = "CompositeId for Job is only supported for Jobs with container type " + maestroSpecs.CONTAINER_DEVICEJS_PROCESS
			// 	return err								
			// }
		}
	}
	return nil
}

func RestartJob( name string ) (errout error) {
	//
	// FIXME FIXME - needs to stop Job!!
	//
	errout = StartJob(name)
	return
}

func StartJob( name string ) (errout error) {
//	job, ok := jobsByName[name]
DEBUG_OUT2("StartJob("+name+")\n")
	jobval, ok := jobsByName.GetStringKey(name)
	if ok {
DEBUG_OUT2("StartJob("+name+") 1\n")
		job := (*processStatus)(jobval)
		if job.status == RUNNING || job.status == STARTING {
			DEBUG_OUT2("StartJob("+name+") already running %d\n",job.status)
			return
		}

DEBUG_OUT2("StartJob("+name+") 1.1\n")
		deps := job.job.GetDependsOn()
		// if we have a Job in a composite process, check the dependencies of the
		// entire composite process, not just this Job within it
		hasmaster, masterdeps := getMasterProcessDepends(job.job.GetCompositeId())
		if hasmaster {
DEBUG_OUT2("StartJob("+name+") 1.1a - using master process deps\n")			
			deps = masterdeps
		}
DEBUG_OUT2("StartJob("+name+") 1.2\n")
		needdeps := false
DEBUG_OUT2("StartJob("+name+") 1.3\n")
		for _, dep := range deps {	
			dep_status_val, ok := jobsByName.GetStringKey(dep)
DEBUG_OUT2("StartJob("+name+") 1.3.1\n")
			if !ok {
				err := new(JobError)
				err.Code = JOBERROR_MISSING_DEPENDENCY
				DEBUG_OUT("StartJob("+name+") JOBERROR_MISSING_DEPENDENCY\n")
				err.JobName = name
				errout = err
				procLogErrorf("Job %s is missing dependency: %s",job.job.GetJobName(),dep)
				return			
			} else {
DEBUG_OUT2("StartJob("+name+") 1.3.2\n")
				dep_status := (*processStatus)(dep_status_val)
				// if this dependency is not running, then we need to 
				// request a start first
				if dep_status.status != RUNNING {
					job.status = WAITING_FOR_DEPS
DEBUG_OUT2("StartJob("+name+") 1.3.2.1 %+v\n",dep_status)
					err2 := StartJob(dep)
					if err2 != nil {
						err := new(JobError)
						DEBUG_OUT("StartJob("+name+") JOBERROR_DEPENDENCY_FAILURE\n")
						err.Code = JOBERROR_DEPENDENCY_FAILURE
						err.JobName = name
						errout = err
						return									
					} else {
						DEBUG_OUT("StartJob("+name+") needs dep %s\n",dep)
						needdeps = true
					}
				}
DEBUG_OUT2("StartJob("+name+") 1.3.3\n")				
			}
		}
DEBUG_OUT2("StartJob("+name+") 1.4\n")
		if needdeps {
			procLogDebugf("Job %s is waiting for dependencies",job.job.GetJobName())
			DEBUG_OUT(" Job: "+name+" marked WAITING_FOR_DEPS\n")
			job.status = WAITING_FOR_DEPS
		} else {
DEBUG_OUT2("StartJob("+name+") 2  -> submit start\n")
			procLogDebugf("Job %s has all dependencies. Asking for start.",job.job.GetJobName())
			ev := new(processEvent)
			ev.code = internalEvent_start_job
			ev.jobname = name
			ev.request = job.job
			controlChan <- ev
		}
	} else {
		err := new(JobError)
		procLogErrorf("processManager - StartJob() did not find Job [%s]",name)
		DEBUG_OUT("StartJob("+name+") JOBERROR_NOT_FOUND\n")
		err.Code = JOBERROR_NOT_FOUND
		err.JobName = name
		errout = err
	}
	return
}

func StartAllAutoStartJobs() {
	for job := range jobsByName.Iter() {
		val := job.Value
		DEBUG_OUT("ITER %+v\n",val)
		if val != nil {
			j := (*processStatus)(val)
			DEBUG_OUT("ITER processStatus %+v\n",j)
			if(j.status == NEVER_RAN) {
				StartJob(j.job.GetJobName())
			}
		}
	}
}

