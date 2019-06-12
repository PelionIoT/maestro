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
	"fmt"
	"sync"
	"time"

	"github.com/armPelionEdge/hashmap" // thread-safe, fast hashmaps
)

func debug_out(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Printf("[debug-events]  %s\n", s)
}

type eventChannel struct {
	fifo *eventFIFO
	id   string
	// if fanout is true, then this eventChannel will not buffer,
	// but will have all slaves buffer. If fanout is false (default)
	// then the master channel buffers, and if a slave misses an event
	// (i.e. its receiving channel would have blocked) then it misses
	// the event entirely.
	// So it depends on the application. If events should be delivered in real-time
	// and are time sensitive, the use fanout false (default)
	// If events should always be recieved, but are not real-time, meaning recievers
	// can deal with getting such events at entirely different times, then use
	// fanout true.
	// Fanout true - is similar to the node.js event emitter, but with the fact
	// that events can buffer up for each subscriber
	// Persistent channels: persistent channles are always fanout true.
	fanout bool
	// if this channel has slave channels,
	// then anything going to this channel will also
	// go to the slave channels
	slaves sync.Map
	// if destroyTimeout (nanoseconds) is greater than zero, then the channel will be torndown after
	// destroyTimeout seconds passby with no PullEvents()
	destroyTimeout int64
	// These are for dealing with slave channel timeouts:
	// interval to check for timed out slaves (nanoseconds)
	timeoutCheckInterval int64
	// when this chane is close()d it will stop the interval checker go-routine
	stopTimeoutCheckerCh chan struct{}
	// true if the interval checker is running
	stoppedTimeoutCheckerLatch sync.Mutex
	timeoutCheckerRunning      bool
	// this is a work around to avoid keys which were delete, but are still in the Iteration of the 'slaves' map
	closedIds *hashmap.HashMap
}

func (evchan *eventChannel) stopTimeoutChecker() {
	evchan.stoppedTimeoutCheckerLatch.Lock()
	close(evchan.stopTimeoutCheckerCh)
}

// should only be called after stopTimeoutChecker()
func (evchan *eventChannel) waitForStoppedTimeoutChecker() {
	evchan.stoppedTimeoutCheckerLatch.Lock()
	evchan.timeoutCheckerRunning = false
	evchan.stoppedTimeoutCheckerLatch.Unlock()
}

func (evchan *eventChannel) isSlaveDeleted(id string) (yes bool) {
	if evchan.closedIds != nil {
		_, yes = evchan.closedIds.GetStringKey(id)
	}
	return
}

func (evchan *eventChannel) slaveTimeoutChecker() {
	var now int64
	evchan.stopTimeoutCheckerCh = make(chan struct{})

	checkFunc := func(key, val interface{}) bool {
		id, ok := key.(string)
		slave, ok2 := val.(*slaveChannel)
		if ok && ok2 {
			debug_out("checkFunc: %s\n", id)
			if slave.destroyTimeout > 0 {
				refs := slave.refs.lock()
				// if no one is using / referencing this channel,
				// and it's timeout has expired, close it down

				if refs < 1 && slave.destroyTimeout < now {
					debug_out("Channel timedout: %s", slave.id)
					slave.Close()
					// _, deleted := evchan.closedIds.GetStringKey(id)
					// if !deleted {
					// 	debug_out("doing real delete on: %s\n", id)
					// 	slave.Close()
					// 	slave.refs.unlock()
					// 	continue innerloop
					// } else {
					// 	debug_out("skipping timedout: %s\n", id)
					// }
				}
				//  else {
				// 	slave.destroyTimeout = now + slave.destroyTime
				// }
				slave.refs.unlock()
			}
		}
		return true
	}

	if evchan.timeoutCheckInterval > 0 {
		evchan.timeoutCheckerRunning = true
		ticker := time.NewTicker(time.Duration(evchan.timeoutCheckInterval) * time.Nanosecond)
		for {
			debug_out("top of for (slaveTimeoutChecker):\n")
			select {
			case <-ticker.C:
				// this closedIds is used to mark slaves as closed
				// for when a iterations (Range) call may be going on while
				// we are also iterating (Range) here checking ideas.
				// See: https://golang.org/pkg/sync/#Map.Range
				evchan.closedIds = hashmap.New(10)
				// look at slaves. see if they are due for expiration
				now = time.Now().UnixNano()
				// innerloop:
				evchan.slaves.Range(checkFunc)
				// for {
				// 	len := evchan.slaves.Len()
				// 	debug_out("top of innerloop (%d):\n", len)
				// evchan.slaveReadIterLock.Lock()
				// if evchan.slaves.Len() < 1 {
				// 	evchan.timeoutCheckerRunning = false
				// 	evchan.slaveReadIterLock.Unlock()
				// 	debug_out("breaking at len() check\n")
				// 	return
				// }

				// for kv := range evchan.slaves.Iter() {
				// 	slave := (*slaveChannel)(kv.Value)
				// 	if slave.destroyTimeout > 0 {
				// 		refs := slave.refs.lock()
				// 		// if no one is using / referencing this channel,
				// 		// and it's timeout has expired, close it down
				// 		if refs < 1 && slave.destroyTimeout < now {
				// 			id := kv.Key.(string)
				// 			debug_out("Channel timedout: %s\n", id)
				// 			_, deleted := evchan.closedIds.GetStringKey(id)
				// 			if !deleted {
				// 				debug_out("doing real delete on: %s\n", id)
				// 				evchan.closedIds.Set(id, unsafe.Pointer(&struct{}{}))
				// 				slave.Close()
				// 				evchan.slaveReadIterLock.Unlock()
				// 				slave.refs.unlock()
				// 				continue innerloop
				// 			} else {
				// 				debug_out("skipping timedout: %s\n", id)
				// 			}
				// 		} else {
				// 			slave.destroyTimeout = now + slave.destroyTime
				// 		}
				// 		slave.refs.unlock()
				// 	}
				// }
				// 	evchan.slaveReadIterLock.Unlock()
				// 	break
				// }
			case <-evchan.stopTimeoutCheckerCh:
				break
			}
		}
	}
	// latch the stop event (so we can safely restart)
	evchan.stoppedTimeoutCheckerLatch.Unlock()
	debug_out("ending slaveTimeoutChecker!!\n")
}

type slaveChannel struct {
	parent *eventChannel
	output chan *MaestroEvent
	id     string
	// if non-zero, then the slave channel will be torn down after this time has passed
	// any time the channel is read from, this will be updated with destoryTimeout = now + destroyTime
	destroyTimeout int64
	// the nano seconds until this channel should time.
	destroyTime  int64
	closed       bool
	refs         refCounter
	closingMutex sync.Mutex
}

type EventSubscription interface {
	GetChannel() (bool, chan *MaestroEvent)
	// call when the channel is not being waiting on
	ReleaseChannel()
	// close the subscription entirely
	Close()
	GetID() string
	IsClosed() bool
}

// GetDummyEventChannel just returns a chan *MaestroEvent,
// which will never have any events. Useful for select{}
// statements where the subscription is not yet valid.
func GetDummyEventChannel() (ret chan *MaestroEvent) {
	ret = make(chan *MaestroEvent)
	return
}

func (channel *eventChannel) getSlaveByID(id string) (ret *slaveChannel, ok bool) {
	var ok2 bool
	var retp interface{}
	retp, ok = channel.slaves.Load(id)
	debug_out("Slaves map @ %v %+v, %+v\n", channel.slaves, retp, ok)
	if ok {
		ret, ok2 = retp.(*slaveChannel)
		if !ok2 {
			ret = nil
			ok = false
		}
	}
	return
}

func (slave *slaveChannel) GetChannel() (ok bool, ret chan *MaestroEvent) {
	slave.refs.lock()
	// ok = !slave.closed
	ok = slave.ifOpenLock()
	if ok {
		ret = slave.output
		slave.openedUnlock()
		slave.refs.refAndUnlock()
	} else {
		slave.refs.unlock()
	}
	return
}

func (slave *slaveChannel) ReleaseChannel() {
	if slave.destroyTime > 0 {
		// if a destroyTime is set for the slave
		// then reset it, since the slave is still being up until the time of release.
		now := time.Now().UnixNano()
		slave.destroyTimeout = now + slave.destroyTime
	}
	// FIXME should we openedUnlock() ?
	// now unref it. The timeout is effective active now
	n := slave.refs.unref()
	debug_out("ReleaseChannel() %d\n", n)
}

// Close may block until a channel calls ReleaseChannel() - be aware
func (slave *slaveChannel) Close() {
	slave.closingMutex.Lock()
	if !slave.closed {
		slave.closed = true
		// remove from parent
		debug_out("Slaves map @ (close) %v - id %s\n", slave.parent.slaves, slave.id)
		slave.parent.slaves.Delete(slave.id)
		close(slave.output)
	} else {
		debug_out("Slaves map @ (closed already) %v - id %s\n", slave.parent.slaves, slave.id)
	}
	slave.closingMutex.Unlock()
}

func (slave *slaveChannel) IsClosed() bool {
	var ret bool
	slave.closingMutex.Lock()
	ret = slave.closed
	slave.closingMutex.Unlock()
	return ret
}

func (slave *slaveChannel) ifOpenLock() bool {
	var ret bool
	slave.closingMutex.Lock()
	ret = slave.closed
	if ret {
		slave.closingMutex.Unlock()
	}
	return !ret
}

func (slave *slaveChannel) openedUnlock() {
	slave.closingMutex.Unlock()
}

func (this *slaveChannel) GetID() string {
	return this.id
}

var channels *hashmap.HashMap

func newEventChannel(name string, fanout bool) (ret *eventChannel) {
	ret = &eventChannel{fifo: nil, id: name}
	ret.fanout = fanout
	if !fanout {
		ret.fifo = New_eventFIFO(EVENT_CHANNEL_QUEUE_SIZE)
	}
	return
}

func newSlaveChannel(name string, parent *eventChannel, timeout int64) (ret *slaveChannel) {
	ret = &slaveChannel{id: name, parent: parent, destroyTime: timeout}
	if timeout > 0 {
		ret.destroyTimeout = time.Now().UnixNano() + ret.destroyTime
		if !parent.timeoutCheckerRunning || parent.timeoutCheckInterval > ret.destroyTime {
			// stop the current checker, if it's interval is too slow
			if parent.timeoutCheckerRunning {
				parent.stopTimeoutChecker()
				parent.waitForStoppedTimeoutChecker()
			}
			parent.timeoutCheckInterval = ret.destroyTime
			go parent.slaveTimeoutChecker()
		}
	}
	if parent.fanout {
		ret.output = make(chan *MaestroEvent, EVENT_CHANNEL_QUEUE_SIZE)
	} else {
		ret.output = make(chan *MaestroEvent)
	}
	ret.refs.ref()
	return
}

func init() {
	channels = hashmap.New(10)
}
