
//
// Overview:
// Setups a go 'channel' - and uses this channel to transfer log data to symphonyd
// This same on-going connection can be used for the server to 'ride back' for commands
// 

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
	"net/http"
	"time"
	"sync"
	"fmt"
	"github.com/armPelionEdge/greasego"
	"bytes"
	"io/ioutil"
//	DEBUG("runtime")
)


type logBuffer struct {
	data *greasego.TargetCallbackData
	godata []byte
}

//  begin generic
// m4_define({{*NODE*}},{{*logBuffer*}})  m4_define({{*FIFO*}},{{*logBufferFifo*}})
// Thread safe queue for LogBuffer
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
	DEBUG_OUT2(" >>>>>>>>>>>> In Push\n")
	fifo.mutex.Lock()
	DEBUG_OUT2(" ------------ In Push (past Lock)\n")
    if int(fifo.maxSize) > 0 && len(fifo.q)+1 > int(fifo.maxSize) {
    	// drop off the queue
    	dropped = (fifo.q)[0]
    	fifo.q = (fifo.q)[1:]
    	fifo.drops++
    	DEBUG_OUT2("!!! Dropping NODE in FIFO \n")
    	drop = true
    }
    fifo.q = append(fifo.q, n)
	DEBUG_OUT2(" ------------ In Push (@ Unlock)\n")
    fifo.mutex.Unlock()
    fifo.condWait.Signal()
	DEBUG_OUT2(" <<<<<<<<<<< Return Push\n")
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

const TIMEOUT = time.Second * 10

type Client struct {
	url string
	clientId string
	httpClient *http.Client
	fifo *FIFO  // see fifo.go
	ticker *time.Ticker
	interval time.Duration
}




// maxBuffers: this number represents the amount of stored log buffers we will hold 
// before dropping them. This can be from 1 to [max amount of bytes from greasego callback]
// In effect, this should be close to the same number as NumBanks is set in the target options
// for the greasego target
func NewSymphonyClient(url string, clientid string, maxBuffers uint32, heartbeatInterval time.Duration) *Client {
	client := new(Client)
	client.url = url
	client.clientId = clientid
	client.httpClient = &http.Client{
		Timeout: TIMEOUT,
	}
	client.fifo = New_logBufferFifo(maxBuffers)
	client.interval = heartbeatInterval
	fmt.Printf("")
	return client
}


func (client *Client) Start() {
	go client.clientWorker()
	client.startTicker()
	DEBUG(fmt.Printf("client started: %s\n",client.url))
}

func (client *Client) SubmitLogs(data *greasego.TargetCallbackData, godata []byte) {
	buf := new(logBuffer)
	buf.data = data
	buf.godata = godata
	dropped, _ := client.fifo.Push(buf)
	if dropped {
		DEBUG_OUT("Dropped some log entries!!!!!!\n\n")
	}
}

func (client *Client) startTicker() {
	client.ticker = time.NewTicker(client.interval)
	go func() {
	// only dump ticker info when in debug build:
        DEBUG(for t := range client.ticker.C { )
            DEBUG_OUT("Tick at %d", t.Unix())
        DEBUG(    DumpMemStats() )
	        DEBUG(	client.fifo.WakeupAll() )
        DEBUG(} )
    }()
}


// the client worker goroutine
// does the sending of data to the server
func (client *Client) clientWorker() {
	closeit := func(r *http.Response, buf *logBuffer) {
		r.Body.Close()
		greasego.RetireCallbackData(buf.data)
		DEBUG_OUT("  OKOKOKOKOKOKOKOKOK -----> retired callback data\n\n")
	}

	var next *logBuffer
	for true {
		next = client.fifo.PopOrWait()
		if next == nil {
			if client.fifo.IsShutdown() {
				DEBUG_OUT("clientWorker @shutdown - via FIFO")
				break;
			} else {
				// SEND HEARTBEAT or whatever
				continue;				
			}
		}
		// send data to server0
//		req, err := http.NewRequest("POST", client.url, bytes.NewReader(next.data.GetBufferAsSlice()))
		req, err := http.NewRequest("POST", client.url, bytes.NewReader(next.godata))
//	    req.Header.Set("X-Custom-Header", "myvalue")
	    req.Header.Set("Content-Type", "application/json")
	    // req.Header.Add("X-Symphony-ClientId", client.clientId)

	    // client := &http.Client{}
	    resp, err := client.httpClient.Do(req)
	    if err != nil {
	    	DEBUG_OUT("XXXXXXXXXXXXXXXXXXXXXXX error on sending request %+v\n",err)
			greasego.RetireCallbackData(next.data)
	    } else {
		    fmt.Println("response Status:", resp.Status)
		    fmt.Println("response Headers:", resp.Header)
		    body, _ := ioutil.ReadAll(resp.Body)
		    fmt.Println("response Body:", string(body))

	    	DEBUG_OUT("CALLING closeit()\n")
		    closeit(resp,next)
	    }

	}
}

func (client *Client) Shutdown() {
	client.ticker.Stop()
	client.fifo.Shutdown()
	// all go routines should end
}









// Test for our FIFO above

// func TestFifo() {
// 	buffer := New_logBufferFifo(5)
// 	exits := 0
// 	// addsome := func(z int, name string){
// 	// 	for n :=0;n < z;n++ {
// 	// 		dropped, _ := buffer.Push(new(logBuffer))
// 	// 		if(dropped) {
// 	// 			fmt.Printf("[%s] Added and Dropped a buffer!: %d\n",name,n)
// 	// 		} else {
// 	// 			fmt.Printf("[%s] added a buffer: %d\n",name,n)
// 	// 		}
// 	// 	}		
// 	// }

// 	addsome := func(z int, name string){
// 		for n :=0;n < z;n++ {
// 			ok := buffer.PushOrWait(new(logBuffer))
// 			if(ok) {
// 				fmt.Printf("[%s] Added a buffer!: %d\n",name,n)
// 			} else {
// 				fmt.Printf("[%s] (add buffer) must be shutdown: %d\n",name,n)
// 				break
// 			}
// 		}		
// 	}

// 	removesome := func(z int, name string){
// 		for n :=0;n < z;n++ {
// 			fmt.Printf("[%s] PopOrWait()...\n",name)
// 			outbuf := buffer.PopOrWait()
// 			if(outbuf != nil) {
// 				fmt.Printf("[%s] Got a buffer\n",name)
// 			} else {
// 				fmt.Printf("[%s] Got nil - must be shutdown\n",name)
// 				break
// 			}
// 		}
// 		fmt.Printf("[%s] removesome Done :)\n",name)
// 		exits++;
// 	}

// 	shutdown_in := func(s int) {
// 		time.Sleep(time.Duration(s)*time.Second)
// 		fmt.Printf("Shutting down FIFO\n")
// 		buffer.Shutdown()
// 		fmt.Printf("Shutdown FIFO complete\n")
// 	}

// 	go addsome(10,"one")
// 	go removesome(10,"remove_one")
// 	go addsome(10,"two")
// 	go removesome(10,"remove_two")
// 	go addsome(10,"three")

// 	go removesome(11,"remove_three")

// 	shutdown_in(5)
// 	time.Sleep(time.Duration(2)*time.Second)
// 	if exits != 3 {
// 		fmt.Printf("exits: %d\n",exits)
// 		panic("Not all exited")
// 	} 
// }


// var netClient = &http.Client{
//   Timeout: time.Second * 10,
// }

