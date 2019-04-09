package events

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
	"sync"
	DEBUG("fmt")
)


// Generics follow
////////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////////

//  begin generic
// m4_define({{*NODE*}},{{*subHttpHandler*}})  m4_define({{*FIFO*}},{{*subHttpHandlerFIFO*}})
type FIFO struct {
	q []*NODE	
	mutex *sync.Mutex
	condWait *sync.Cond
	condFull *sync.Cond
	maxSize uint32
	drops int
	shutdown bool
	wakeupIter int  // this is to deal with the fact that go developers
	                // decided not to implement pthread_cond_timedwait()
	                // So we use this as a work around to temporarily wakeup
	                // (but not shutdown) the queue. Bring your own timer.
}


func GENERIC_New(FIFO)(maxsize uint32) (ret *FIFO) {
	ret = new(FIFO)
	ret.mutex =  new(sync.Mutex)
	ret.condWait = sync.NewCond(ret.mutex)
	ret.condFull = sync.NewCond(ret.mutex)
	ret.maxSize = maxsize
	ret.drops = 0
	ret.shutdown = false
	ret.wakeupIter = 0
	return
}
func (fifo *FIFO) Push(n *NODE) (drop bool, dropped *NODE) {
	drop = false
	DEBUG_OUT2(" >>>>>>>>>>>> [ FIFO ] >>> In Push\n")
	fifo.mutex.Lock()
	DEBUG_OUT2(" ------------ In Push (past Lock)\n")
    if int(fifo.maxSize) > 0 && len(fifo.q)+1 > int(fifo.maxSize) {
    	// drop off the queue
    	dropped = (fifo.q)[0]
    	fifo.q = (fifo.q)[1:]
    	fifo.drops++
    	DEBUG_OUT("!!! Dropping NODE in FIFO \n")
    	drop = true
    }
    fifo.q = append(fifo.q, n)
	DEBUG_OUT2(" ------------ In Push (@ Unlock)\n")
    fifo.mutex.Unlock()
    fifo.condWait.Signal()
	DEBUG_OUT2(" <<<<<<<<<<< Return Push\n")
    return
}

// Pushes a batch of NODE. Drops older NODE to make room
func (fifo *FIFO) PushBatch(n []*NODE) (drop bool, dropped []*NODE) {
	drop = false
	DEBUG_OUT2(" >>>>>>>>>>>> [ FIFO ] >>> In PushBatch\n")
	fifo.mutex.Lock()
	DEBUG_OUT2(" ------------ In PushBatch (past Lock)\n")
	_len := uint32(len(fifo.q))
	_inlen := uint32(len(n))
	if fifo.maxSize > 0 && _inlen > fifo.maxSize {
		_inlen = fifo.maxSize
	}
    if fifo.maxSize > 0 && _len+_inlen > fifo.maxSize {
    	needdrop := _inlen+_len - fifo.maxSize 
    	if needdrop >= fifo.maxSize {
	    	drop = true
    		dropped = fifo.q
	    	fifo.q = nil
    	} else if needdrop > 0 {
	    	drop = true
	    	dropped = (fifo.q)[0:needdrop]
	    	fifo.q=(fifo.q)[needdrop:]
	    }
    	// // drop off the queue
    	// dropped = (fifo.q)[0]
    	// fifo.q = (fifo.q)[1:]
    	// fifo.drops++
    	DEBUG_OUT2(" ----------- PushBatch() !!! Dropping %d NODE in FIFO \n", len(dropped))
    }
    DEBUG_OUT2(" ----------- In PushBatch (pushed %d)\n",_inlen)
    fifo.q = append(fifo.q, n[0:int(_inlen)]...)
	DEBUG_OUT2(" ------------ In PushBatch (@ Unlock)\n")
    fifo.mutex.Unlock()
    fifo.condWait.Signal()
	DEBUG_OUT2(" <<<<<<<<<<< Return PushBatch\n")
    return
}


func (fifo *FIFO) Pop() (n *NODE) {
	fifo.mutex.Lock()
	if len(fifo.q) > 0 {
	    n = (fifo.q)[0]
	    fifo.q = (fifo.q)[1:]		
		fifo.condFull.Signal()
	} 
	fifo.mutex.Unlock()
    return
}

func (fifo *FIFO) Peek() (n *NODE) {
	fifo.mutex.Lock()
	if len(fifo.q) > 0 {
	    n = (fifo.q)[0]
	} 
	fifo.mutex.Unlock()
    return
}

// func (fifo *FIFO) PopBatch(max uint32) (n *NODE) {
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
func (fifo *FIFO) Len() int {
	fifo.mutex.Lock()
	ret := len(fifo.q)
	fifo.mutex.Unlock()
    return ret
}
func (fifo *FIFO) PopOrWait() (n *NODE) {
	n = nil
	DEBUG_OUT2(" >>>>>>>>>>>> In PopOrWait (Lock)\n")
	fifo.mutex.Lock()
	_wakeupIter := fifo.wakeupIter
	if(fifo.shutdown) {
		fifo.mutex.Unlock()
		DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWait (Unlock 1)\n")
		return
	}
	if len(fifo.q) > 0 {
	    n = (fifo.q)[0]
	    fifo.q = (fifo.q)[1:]		
		fifo.mutex.Unlock()
		fifo.condFull.Signal()
		DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWait (Unlock 2)\n")
		return
	}
	// nothing there, let's wait
	for !fifo.shutdown && fifo.wakeupIter == _wakeupIter {
//		fmt.Printf(" --entering wait %+v\n",*fifo);
	DEBUG_OUT2(" ----------- In PopOrWait (Wait / Unlock 1)\n")
		fifo.condWait.Wait() // will unlock it's "Locker" - which is fifo.mutex
//		Wait returns with Lock
//		fmt.Printf(" --out of wait %+v\n",*fifo);
		if fifo.shutdown { 
			fifo.mutex.Unlock()
	DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWait (Unlock 4)\n")
			return 
		}
		if len(fifo.q) > 0 {
		    n = (fifo.q)[0]
		    fifo.q = (fifo.q)[1:]		
			fifo.mutex.Unlock()
			fifo.condFull.Signal()
		DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWait (Unlock 3)\n")
			return
		}
	}
	DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWait (Unlock 5)\n")
	fifo.mutex.Unlock()
	return
}
func (fifo *FIFO) PopOrWaitBatch(max uint32) (slice []*NODE) {
	DEBUG_OUT2(" >>>>>>>>>>>> In PopOrWaitBatch (Lock)\n")
	fifo.mutex.Lock()
	_wakeupIter := fifo.wakeupIter
	if(fifo.shutdown) {
		fifo.mutex.Unlock()
		DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 1)\n")
		return
	}
	_len := uint32(len(fifo.q))
	if _len > 0 {
		if  max >= _len {
	    	slice = fifo.q
	    	fifo.q = nil  // http://stackoverflow.com/questions/29164375/golang-correct-way-to-initialize-empty-slice
		} else {
			slice = (fifo.q)[0:max]
			fifo.q = (fifo.q)[max:]		
		}
		fifo.mutex.Unlock()
		fifo.condFull.Signal()
		DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 2)\n")
		return
	}
	// nothing there, let's wait
	for !fifo.shutdown && fifo.wakeupIter == _wakeupIter {
//		fmt.Printf(" --entering wait %+v\n",*fifo);
	DEBUG_OUT2(" ----------- In PopOrWaitBatch (Wait / Unlock 1)\n")
		fifo.condWait.Wait() // will unlock it's "Locker" - which is fifo.mutex
//		Wait returns with Lock
//		fmt.Printf(" --out of wait %+v\n",*fifo);
		if fifo.shutdown { 
			fifo.mutex.Unlock()
	DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 4)\n")
			return 
		}
		_len = uint32(len(fifo.q))
		if _len > 0 {
			if max >= _len {
		    	slice = fifo.q
		    	fifo.q = nil  // http://stackoverflow.com/questions/29164375/golang-correct-way-to-initialize-empty-slice
			} else {
				slice = (fifo.q)[0:max]
				fifo.q = (fifo.q)[max:]			
			}
			fifo.mutex.Unlock()
			fifo.condFull.Signal()
		DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 3)\n")
			return
		}
	}
	DEBUG_OUT2(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 5)\n")
	fifo.mutex.Unlock()
	return
}
func (fifo *FIFO) PushOrWait(n *NODE) (ret bool) {
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
func (fifo *FIFO) Shutdown() {
	fifo.mutex.Lock()
	fifo.shutdown = true
	fifo.mutex.Unlock()
	fifo.condWait.Broadcast()
	fifo.condFull.Broadcast()
}
func (fifo *FIFO) WakeupAll() {
	DEBUG_OUT2(" >>>>>>>>>>> in WakeupAll @Lock\n")
	fifo.mutex.Lock()
	DEBUG_OUT2(" +++++++++++ in WakeupAll\n")
	fifo.wakeupIter++
	fifo.mutex.Unlock()
	DEBUG_OUT2(" +++++++++++ in WakeupAll @Unlock\n")
	fifo.condWait.Broadcast()
	fifo.condFull.Broadcast()
	DEBUG_OUT2(" <<<<<<<<<<< in WakeupAll past @Broadcast\n")
}
func (fifo *FIFO) IsShutdown() (ret bool) {
	fifo.mutex.Lock()
	ret = fifo.shutdown
	fifo.mutex.Unlock()
	return
}
// end generic
// 
// 
