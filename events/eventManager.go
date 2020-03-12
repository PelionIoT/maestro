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
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/armPelionEdge/maestro/debugging"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/storage"
	"github.com/boltdb/bolt"
)

/******************************

An event manager, with a different methodology than
the 'go' channels. By default every event will go to all
subscribers of that event (so "fan out")

Peristent events will survive Maestro restarts.

******************************/

// initialized in init()
type eventManagerInstance struct {
	db *bolt.DB
}

var instance *eventManagerInstance

// impl storage.StorageUser interface
func (this *eventManagerInstance) StorageInit(instance storage.MaestroDBStorageInterface) {
	// any gob.Register ?

}

func (this *eventManagerInstance) StorageReady(instance storage.MaestroDBStorageInterface) {
	this.db = instance.GetDb()
}

func (this *eventManagerInstance) StorageClosed(instance storage.MaestroDBStorageInterface) {
}

// persisteEventChannels always use bolt for storage
// they do not use a FIFO queue ever
type persistentEventChannel struct {
	id string
	// each key simply uses a UnixNano timestamp number encoded as a string
	bucketName *bolt.Bucket
	// is restored when the channel is reloaded from disk
	// slaves have a prefix like "slave1:191818717119"
	//	slaves *hashmap.HashMap
}

type persistentSlaveChannelMap map[string]int64 // a map of channel name to time create in time.UnixNano()

type MaestroEvent struct {
	// subscriber ID that the event belongs to (when sent to a slave channel / subscribing channel)
	subID string
	// seconds since Unix Epoch
	enqueTime int64
	// some data, usually JSON encodable
	Data interface{} `json:"data"`
}

func (ev *MaestroEvent) GetEnqueTime() (ret int64) {
	ret = ev.enqueTime
	return
}

type MaestroEventError struct {
	s           string
	channelName string
	data        interface{}
}

func unixNanoNowString(offset int64) (ret string) {
	now := time.Now().UnixNano()
	now += offset
	ret = fmt.Sprintf("%019d", now)
	return
}

func (this *MaestroEventError) Error() string {
	return this.s
}

func newMaestroEventError(s string) (ret *MaestroEventError) {
	ret = &MaestroEventError{s: s}
	return
}

// used??
type EventHandler interface {
	// a simple function which is called when new Events
	// are available on 'channel' eventChannel
	HandleEventNotify(channel string)

	// called when a channel is dropping events
	// May do something with the events, or may just
	// ignore them
	HandleEventOverflow(channel string, events []*MaestroEvent)
}

var maxChannelQueue uint32

const channelBucketPrefix = "channel#"
const channelBucketPrefix_len = len(channelBucketPrefix)

const channelCreatedDateKey = "##created"

var channelCreatedDateKey_bytes []byte

// this key holds a list of list slaves for the channel as a gob-encoded map[string]bool
const slavesKey = "##slaves"

var slavesKey_bytes []byte

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// should be pretty fast: https://stackoverflow.com/questions/1760757/how-to-efficiently-concatenate-strings-in-go
// single byte chars only!
func bucketNameFromChannel(channel string) (ret []byte) {
	ret = make([]byte, len(channel)+channelBucketPrefix_len)
	copy(ret, channelBucketPrefix)
	copy(ret[channelBucketPrefix_len:], channel)
	return
}

// single byte chars only!
func stringToBytes(s string) (ret []byte) {
	ret = make([]byte, len(s))
	copy(ret, s)
	return
}

// func slaveBucketNameFromChannel(master string, slave string) (ret []byte) {
// 	master_len := len(master)
// 	ret = make([]byte,master_len + len(slave) + channelBucketPrefix_len + 1)
// 	copy(ret,channelBucketPrefix)
// 	copy(ret[channelBucketPrefix_len:],master)
// 	copy(ret[channelBucketPrefix_len+master_len:],"#")  // a second '#'
// 	copy(ret[channelBucketPrefix_len+master_len+1:],slave)
// 	return
// }

// Make a new channel. If 'id' is an empty string then a random
// name for the channel is made. If 'fanout' is true, then each subscriber
// will get their own buffered channel. If 'fanout' is false, if a subscriber's go chan will block
// then it will miss the event.
func MakeEventChannel(id string, fanout bool, persistent bool) (ok bool, finalId string, err error) {
	finalId = id

	if !persistent {

		if len(id) < 1 {
			ok2 := true
			for ok2 {
				finalId = randStringBytes(5)
				_, ok2 = channels.Load(finalId)
			}
		} else {
			_, ok = channels.Load(id)
			if ok {
				// it's already made
				return
			}
		}

		channel := newEventChannel(finalId, fanout)
		ok = true

		channels.Store(finalId, channel)
	} else {
		if instance != nil && instance.db != nil {
			errouter := instance.db.Update(func(tx *bolt.Tx) error {
				_, err2 := tx.CreateBucketIfNotExists(bucketNameFromChannel(id))
				if err2 != nil {
					errout := newMaestroEventError(fmt.Sprintf("persistent channel %s - create bucket error: %s", id, err))
					return errout
				} else {
					return nil
				}
			})
			if errouter != nil {
				ok = false
				err = errouter
				return
			}
		} else {
			ok = false
			errout := newMaestroEventError("storage not setup")
			err = errout
			return
		}
	}

	return
}

func makeSlaveChannel(master string, timeout int64) (finalId string, err error, subscription EventSubscription) {

	masterp, ok2 := channels.Load(master)

	if ok2 {
		masterc := masterp.(*eventChannel)

		channel := newSlaveChannel(finalId, masterc, timeout)

		ok3 := true
		for ok3 {
			finalId = randStringBytes(8)
			channel.id = finalId
			_, ok3 = masterc.slaves.LoadOrStore(finalId, channel)
		}
		subscription = channel
		logEvent("Made slave channel %s for master %s", finalId, master)
	} else {
		if instance.db != nil {
			errouter := instance.db.Update(func(tx *bolt.Tx) error {
				masterb := tx.Bucket(bucketNameFromChannel(master))

				if masterb == nil {
					errout := newMaestroEventError(fmt.Sprintf("no persistent master channel named: %s", master))
					err = errout
					return err
				}

				var slaveb *bolt.Bucket

				finalId = randStringBytes(5)
				slavebname := bucketNameFromChannel(finalId)
				slaveb = masterb.Bucket(slavebname)
				// loop, if per chance we made a name which was already used
				for slaveb != nil {
					finalId = randStringBytes(5)
					slavebname = bucketNameFromChannel(finalId)
					slaveb = masterb.Bucket(bucketNameFromChannel(finalId))
				}

				slaveb, err = masterb.CreateBucket(slavebname)

				if err != nil {
					return err
				} else {
					// todo: add to slave list

					slavesval := masterb.Get(slavesKey_bytes)
					var slaves persistentSlaveChannelMap
					if slavesval != nil {
						buffer := bytes.NewBuffer(slavesval)
						decoder := gob.NewDecoder(buffer)
						err = decoder.Decode(&slaves)
						if err == nil {
							// assume corrupted, use empty map
							// print error
							slaves = persistentSlaveChannelMap{}
						}
					}
					// add new name of slave channel to map
					// encode, and write back to DB
					slaves[finalId] = time.Now().UnixNano()
					outbuffer := new(bytes.Buffer)
					encoder := gob.NewEncoder(outbuffer)
					err = encoder.Encode(slaves)
					if err != nil {
						return err
					}

					err = masterb.Put(slavesKey_bytes, outbuffer.Bytes())
					if err != nil {
						return err
					}

					return nil
				}
			})
			if errouter != nil {
				err = errouter
				return
			}
		} else {
			errout := newMaestroEventError("storage not setup")
			err = errout
			return
		}
	}

	return
}

func GetSubscription(channelname string, subscriptionid string) (ok bool, ret EventSubscription) {
	mcp, ok2 := channels.Load(channelname)
	if ok2 {
		mc := mcp.(*eventChannel)
		ret, ok = mc.getSlaveByID(subscriptionid)
		if !ok {
			ret = nil
		}
		// if ret != EventSubscription(nil) {
		// 	fmt.Printf("Here////////////////// ret:%v\n",ret)
		// 	ok = true
		// }
	}
	return
}

// return back a golang chan, which will be a channel which is
// subscribed to a specific Event Channel
func SubscribeToChannel(name string, timeout int64) (ret EventSubscription, err error) {
	// find the master channel:
	_, ok2 := channels.Load(name)
	if ok2 {
		//		channel := (*eventChannel)(c)
		_, _, ret = makeSlaveChannel(name, timeout)
	} else {
		errout := newMaestroEventError("No channel named: " + name)
		err = errout
	}

	return
}

func dupEvent(ev *MaestroEvent) (ret *MaestroEvent) {
	ret = new(MaestroEvent)
	ret.enqueTime = ev.enqueTime
	ret.Data = ev.Data
	return
}

func dupEventForSlave(id string, ev *MaestroEvent) (ret *MaestroEvent) {
	ret = new(MaestroEvent)
	ret.subID = id
	ret.enqueTime = ev.enqueTime
	ret.Data = ev.Data
	return
}

func (channel *eventChannel) submitEvent(ev *MaestroEvent) (dropped bool, droppedEv *MaestroEvent) {
	var nextev *MaestroEvent
	if !channel.fanout {
		dropped, droppedEv = channel.fifo.Push(ev)
		nextev = channel.fifo.Pop()
	}
	if dropped {
		log.MaestroWarnf("EventManager - dropping an event on master channel %s\n", channel.id)
	}
	// channel.slaveReadIterLock.Lock()
	funcSubmitEv := func(key, val interface{}) bool {
		id, ok := key.(string)
		slave, ok2 := val.(*slaveChannel)
		if ok && ok2 && slave.ifOpenLock() {
			// ifOpenLock is true if the slave channel is not closed, and then
			// locks the channel from being closed
			if channel.fanout {
				dup_ev := dupEventForSlave(slave.GetID(), ev)
				// no queue in master channel, all slaves use buffered standard channels
				// slaves will get dropped events, only if their (individual) buffer fills up
				select {

				case slave.output <- dup_ev:
					logEvent("submitEvent() (fanout) slave --> %s  %+v\n", channel.id, ev)
				default:
					// if ok {
					log.MaestroWarnf("EventManager - dropping an event on slave(fanout) [%s] of channel %s\n", id, channel.id)
					// } else {
					// 	log.MaestroError("EventManager - seems to have corruption in event map")
					// }
				}
			} else {
				// in non-fanout, there is a single buffer. Slave channels are not buffered, so if
				// they can't accept they event then, when given to them,, it will be dropped.
				dup_ev := dupEvent(nextev)
				select {
				case slave.output <- dup_ev:
					logEvent("submitEvent() slave --> %s   %+v\n", channel.id, ev)
				default:

					// if ok {
					log.MaestroWarnf("EventManager - dropping an event on slave [%s] of channel %s\n", id, channel.id)
					// } else {
					// 	log.MaestroError("EventManager - seems to have corruption in event map")
					// }
				}
			}
			// allow slave to be closed
			slave.openedUnlock()
		}
		return true
	}

	logEvent("submitEvent() %s  %+v\n", channel.id, ev)
	channel.slaves.Range(funcSubmitEv)

	// channel.slaveReadIterLock.Unlock()
	// TODO need to add support for persistent channels
	return
}

// Submit an event to one or more channels
func SubmitEvent(channelNames []string, data interface{}) (dropped bool, err error) {

	// TODO need to add support for persisten channels
	var ev *MaestroEvent

	for _, name := range channelNames {
		if ev == nil {
			ev = &MaestroEvent{Data: data, enqueTime: time.Now().UnixNano()}
		}
		c, ok2 := channels.Load(name)
		if ok2 {
			channel := c.(*eventChannel)
			drop, _ := channel.submitEvent(ev)
			if drop {
				dropped = true
			}
			if dropped {
				log.MaestroWarnf("EventManager - dropping an event on channel %s\n", name)
			}
		} else {
			errout := newMaestroEventError("No channel named: " + name)
			errout.data = data
			err = errout
			return
		}
	}

	return
}

// This submits an event to every channel, which means every subscriber
// for every channel name will get the events
func BroadcastEvent(data interface{}) (dropped bool, err error) {
	var ev *MaestroEvent
	channels.Range(func(key, value interface{}) bool {
		if ev == nil {
			ev = &MaestroEvent{Data: data, enqueTime: time.Now().UnixNano()}
		}
		channel := value.(*eventChannel)
		drop, _ := channel.fifo.Push(ev)
		if drop {
			dropped = true
		}
		return true
	})

	return
}

type MaestroEventBaton struct {
	uid         uint32
	channelName string
	events      []*MaestroEvent
}

func (this *MaestroEventBaton) Close() (err error) {
	c, ok := channels.Load(this.channelName)
	if ok {
		channel := c.(*eventChannel)
		channel.fifo.RemovePeeked(this.events, this.uid)
	} else {
		errout := newMaestroEventError("MaestroEventBaton has unknown channel: " + this.channelName)
		err = errout
	}
	return
}

func (this *MaestroEventBaton) Events() []*MaestroEvent {
	return this.events
}

func PullEvents(name string, max uint32) (err error, baton *MaestroEventBaton) {
	c, ok := channels.Load(name)
	if ok {
		channel := c.(*eventChannel)
		baton = &MaestroEventBaton{channelName: name}
		baton.events, baton.uid = channel.fifo.PeekBatch(max)
	} else {
		errout := newMaestroEventError("No channel named: " + name)
		err = errout
	}
	return
}

type EventManagerReadyCB func() error

var evMgrReady bool

var readyCBs []EventManagerReadyCB

func OnEventManagerReady(cb EventManagerReadyCB) {
	if evMgrReady {
		if cb != nil {
			cb()
		}
	} else {
		fmt.Printf("SHOULD BE UNREACHABLE.")
		readyCBs = append(readyCBs, cb)
	}
}

func init() {
	readyCBs = make([]EventManagerReadyCB, 5, 5)
	instance = new(eventManagerInstance)
	maxChannelQueue = EVENT_CHANNEL_QUEUE_SIZE
	slavesKey_bytes = stringToBytes(slavesKey)
	channelCreatedDateKey_bytes = stringToBytes(channelCreatedDateKey)
	storage.RegisterStorageUser(instance)
	evMgrReady = true
}

// Generics follow
//////////////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////

//  begin generic
// m4_define({{*NODE*}},{{*MaestroEvent*}})  m4_define({{*FIFO*}},{{*eventFIFO*}})
// Thread safe queue for LogBuffer
type eventFIFO struct {
	nextUidOut uint32 // the 'uid' of the next
	nextUid    uint32
	q          []*MaestroEvent
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

func New_eventFIFO(maxsize uint32) (ret *eventFIFO) {
	ret = new(eventFIFO)
	ret.mutex = new(sync.Mutex)
	ret.condWait = sync.NewCond(ret.mutex)
	ret.condFull = sync.NewCond(ret.mutex)
	ret.maxSize = maxsize
	ret.drops = 0
	ret.shutdown = false
	ret.wakeupIter = 0
	ret.nextUid = 0
	ret.nextUidOut = 0
	return
}

// returns a duplicate of the eventFIFO
// the eventFIFO's queue will be a new queue,
// but, the queue's elements will still be pointing at the same MaestroEvent elements
// as the original (since eventFIFO always uses pointers to MaestroEvent)
// The new eventFIFO's drop count will be zero
func (this *eventFIFO) Duplicate() (ret *eventFIFO) {
	ret = new(eventFIFO)
	ret.mutex = new(sync.Mutex)
	ret.condWait = sync.NewCond(ret.mutex)
	ret.condFull = sync.NewCond(ret.mutex)
	ret.maxSize = this.maxSize
	ret.drops = 0
	ret.shutdown = false
	ret.wakeupIter = 0
	ret.nextUid = this.nextUid
	ret.nextUidOut = this.nextUidOut
	ret.q = make([]*MaestroEvent, len(this.q), cap(this.q))
	copy(ret.q, this.q)
	return
}
func (fifo *eventFIFO) Push(n *MaestroEvent) (drop bool, dropped *MaestroEvent) {
	drop = false
	debugging.DEBUG_OUT(" >>>>>>>>>>>> In Push\n")
	fifo.mutex.Lock()
	debugging.DEBUG_OUT(" ------------ In Push (past Lock)\n")
	if int(fifo.maxSize) > 0 && len(fifo.q)+1 > int(fifo.maxSize) {
		// drop off the queue
		dropped = (fifo.q)[0]
		fifo.q = (fifo.q)[1:]
		fifo.drops++
		fifo.nextUidOut++
		debugging.DEBUG_OUT("!!! Dropping MaestroEvent in eventFIFO \n")
		drop = true
	}
	fifo.q = append(fifo.q, n)
	fifo.nextUid++
	debugging.DEBUG_OUT(" ------------ In Push (@ Unlock)\n")
	fifo.mutex.Unlock()
	fifo.condWait.Signal()
	debugging.DEBUG_OUT(" <<<<<<<<<<< Return Push\n")
	return
}
func (fifo *eventFIFO) PushBatch(n []*MaestroEvent) (drop bool, dropped []*MaestroEvent) {
	drop = false
	debugging.DEBUG_OUT(" >>>>>>>>>>>> In PushBatch\n")
	fifo.mutex.Lock()
	debugging.DEBUG_OUT(" ------------ In PushBatch (past Lock)\n")
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
			fifo.nextUidOut += fifo.maxSize
		} else if needdrop > 0 {
			drop = true
			dropped = (fifo.q)[0:needdrop]
			fifo.q = (fifo.q)[needdrop:]
			fifo.nextUidOut += needdrop
		}
		// // drop off the queue
		// dropped = (fifo.q)[0]
		// fifo.q = (fifo.q)[1:]
		// fifo.drops++
		debugging.DEBUG_OUT(" ----------- PushBatch() !!! Dropping %d MaestroEvent in eventFIFO \n", len(dropped))
	}
	debugging.DEBUG_OUT(" ----------- In PushBatch (pushed %d)\n", _inlen)
	fifo.q = append(fifo.q, n[0:int(_inlen)]...)
	fifo.nextUid += _inlen
	debugging.DEBUG_OUT(" ------------ In PushBatch (@ Unlock)\n")
	fifo.mutex.Unlock()
	fifo.condWait.Signal()
	debugging.DEBUG_OUT(" <<<<<<<<<<< Return PushBatch\n")
	return
}

func (fifo *eventFIFO) Pop() (n *MaestroEvent) {
	fifo.mutex.Lock()
	if len(fifo.q) > 0 {
		n = (fifo.q)[0]
		fifo.q = (fifo.q)[1:]
		fifo.nextUidOut++
		fifo.condFull.Signal()
	}
	fifo.mutex.Unlock()
	return
}

// func (fifo *eventFIFO) PopBatch(max uint32) (n *MaestroEvent) {
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
func (fifo *eventFIFO) PopBatch(max uint32) (slice []*MaestroEvent) {
	debugging.DEBUG_OUT(" >>>>>>>>>>>> In PopOrWaitBatch (Lock)\n")
	fifo.mutex.Lock()
	//	_wakeupIter := fifo.wakeupIter
	if fifo.shutdown {
		fifo.mutex.Unlock()
		debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 1)\n")
		return
	}
	_len := uint32(len(fifo.q))
	if _len > 0 {
		if max >= _len {
			slice = fifo.q
			fifo.nextUidOut += _len
			fifo.q = nil // http://stackoverflow.com/questions/29164375/golang-correct-way-to-initialize-empty-slice
		} else {
			fifo.nextUidOut += max
			slice = (fifo.q)[0:max]
			fifo.q = (fifo.q)[max:]
		}
		fifo.mutex.Unlock()
		fifo.condFull.Signal()
		debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 2)\n")
		return
	}
	fifo.mutex.Unlock()
	return
}

// Works by copying a batch of queued values to a slice.
// If the caller decided it really want to remove that which
// it earlier Peek(ed) then it will use the 'uid' value to tell
// the eventFIFO which Peek it was. If that batch is not already gone, it will
// then be removed.
func (fifo *eventFIFO) PeekBatch(max uint32) (slice []*MaestroEvent, uid uint32) {
	debugging.DEBUG_OUT(" >>>>>>>>>>>> In PeekBatch (Lock)\n")
	fifo.mutex.Lock()
	//	_wakeupIter := fifo.wakeupIter
	if fifo.shutdown {
		fifo.mutex.Unlock()
		debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PeekBatch (Unlock 1 - shutdown)\n")
		return
	}
	_len := uint32(len(fifo.q))
	debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PeekBatch %d\n", _len)
	if _len > 0 {
		uid = fifo.nextUidOut
		debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PeekBatch uid %d\n", uid)
		// we make a copy of the slice, so that all elements will be available, regardless if
		// the eventFIFO itself bumps these off the queue later
		if max >= _len {
			slice = make([]*MaestroEvent, _len)
			copy(slice, fifo.q)
			slice = fifo.q
			//	    	fifo.nextUidOut+= _len
			//	    	fifo.q = nil  // http://stackoverflow.com/questions/29164375/golang-correct-way-to-initialize-empty-slice
		} else {
			//			fifo.nextUidOut += max
			slice = make([]*MaestroEvent, max)
			copy(slice, fifo.q)
			//			slice = (fifo.q)[0:max]
			//			fifo.q = (fifo.q)[max:]
		}
		fifo.mutex.Unlock()
		//		fifo.condFull.Signal()
		debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PeekBatch (Unlock 2)\n")
		return
	}
	fifo.mutex.Unlock()
	return
}

// Removes the stated 'peek', which is the slice of nodes
// provided by the PeekBatch function, and the 'uid' that
// was returned by that function
func (fifo *eventFIFO) RemovePeeked(slice []*MaestroEvent, uid uint32) {
	// if for some reason the nextUidOut is less than
	// the uid provided, it means the uid value was high it flowed over
	// and just ignore this call
	fifo.mutex.Lock()
	if slice != nil && uid <= fifo.nextUidOut {
		_len := uint32(len(slice))
		if _len > 0 {
			_offset := (fifo.nextUidOut - uid)
			_fifolen := uint32(len(fifo.q))
			if _len > _offset {
				_removelen := _len - _offset
				//				debugging.DEBUG_OUT("RemovePeeked nextUidOut %d %d %d %d %d %d\n",fifo.nextUidOut, uid, _offset, _len, _fifolen, _removelen)
				debugging.DEBUG_OUT("RemovePeeked _removelen %d \n", _removelen)
				if _removelen >= _fifolen {
					debugging.DEBUG_OUT("RemovePeeked (1) nil\n")
					fifo.nextUidOut += _removelen
					fifo.q = nil // http://stackoverflow.com/questions/29164375/golang-correct-way-to-initialize-empty-slice
				} else {
					debugging.DEBUG_OUT("RemovePeeked (2) %d\n", _removelen)
					fifo.nextUidOut += _removelen
					fifo.q = (fifo.q)[_removelen:]
				}
				fifo.mutex.Unlock()
				fifo.condFull.Signal()
				return
			}
		}
	}
	debugging.DEBUG_OUT("RemovePeeked (3) noop\n")
	fifo.mutex.Unlock()
	return
}

func (fifo *eventFIFO) Len() int {
	fifo.mutex.Lock()
	ret := len(fifo.q)
	fifo.mutex.Unlock()
	return ret
}
func (fifo *eventFIFO) PopOrWait() (n *MaestroEvent) {
	n = nil
	debugging.DEBUG_OUT(" >>>>>>>>>>>> In PopOrWait (Lock)\n")
	fifo.mutex.Lock()
	_wakeupIter := fifo.wakeupIter
	if fifo.shutdown {
		fifo.mutex.Unlock()
		debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PopOrWait (Unlock 1)\n")
		return
	}
	if len(fifo.q) > 0 {
		n = (fifo.q)[0]
		fifo.q = (fifo.q)[1:]
		fifo.nextUidOut++
		fifo.mutex.Unlock()
		fifo.condFull.Signal()
		debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PopOrWait (Unlock 2)\n")
		return
	}
	// nothing there, let's wait
	for !fifo.shutdown && fifo.wakeupIter == _wakeupIter {
		//		fmt.Printf(" --entering wait %+v\n",*fifo);
		debugging.DEBUG_OUT(" ----------- In PopOrWait (Wait / Unlock 1)\n")
		fifo.condWait.Wait() // will unlock it's "Locker" - which is fifo.mutex
		//		Wait returns with Lock
		//		fmt.Printf(" --out of wait %+v\n",*fifo);
		if fifo.shutdown {
			fifo.mutex.Unlock()
			debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PopOrWait (Unlock 4)\n")
			return
		}
		if len(fifo.q) > 0 {
			n = (fifo.q)[0]
			fifo.q = (fifo.q)[1:]
			fifo.nextUidOut++
			fifo.mutex.Unlock()
			fifo.condFull.Signal()
			debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PopOrWait (Unlock 3)\n")
			return
		}
	}
	debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PopOrWait (Unlock 5)\n")
	fifo.mutex.Unlock()
	return
}
func (fifo *eventFIFO) PopOrWaitBatch(max uint32) (slice []*MaestroEvent) {
	debugging.DEBUG_OUT(" >>>>>>>>>>>> In PopOrWaitBatch (Lock)\n")
	fifo.mutex.Lock()
	_wakeupIter := fifo.wakeupIter
	if fifo.shutdown {
		fifo.mutex.Unlock()
		debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 1)\n")
		return
	}
	_len := uint32(len(fifo.q))
	if _len > 0 {
		if max >= _len {
			fifo.nextUidOut += _len
			slice = fifo.q
			fifo.q = nil // http://stackoverflow.com/questions/29164375/golang-correct-way-to-initialize-empty-slice
		} else {
			fifo.nextUidOut += max
			slice = (fifo.q)[0:max]
			fifo.q = (fifo.q)[max:]
		}
		fifo.mutex.Unlock()
		fifo.condFull.Signal()
		debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 2)\n")
		return
	}
	// nothing there, let's wait
	for !fifo.shutdown && fifo.wakeupIter == _wakeupIter {
		//		fmt.Printf(" --entering wait %+v\n",*fifo);
		debugging.DEBUG_OUT(" ----------- In PopOrWaitBatch (Wait / Unlock 1)\n")
		fifo.condWait.Wait() // will unlock it's "Locker" - which is fifo.mutex
		//		Wait returns with Lock
		//		fmt.Printf(" --out of wait %+v\n",*fifo);
		if fifo.shutdown {
			fifo.mutex.Unlock()
			debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 4)\n")
			return
		}
		_len = uint32(len(fifo.q))
		if _len > 0 {
			if max >= _len {
				fifo.nextUidOut += _len
				slice = fifo.q
				fifo.q = nil // http://stackoverflow.com/questions/29164375/golang-correct-way-to-initialize-empty-slice
			} else {
				fifo.nextUidOut += max
				slice = (fifo.q)[0:max]
				fifo.q = (fifo.q)[max:]
			}
			fifo.mutex.Unlock()
			fifo.condFull.Signal()
			debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 3)\n")
			return
		}
	}
	debugging.DEBUG_OUT(" <<<<<<<<<<<<< In PopOrWaitBatch (Unlock 5)\n")
	fifo.mutex.Unlock()
	return
}
func (fifo *eventFIFO) PushOrWait(n *MaestroEvent) (ret bool) {
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
	fifo.nextUid++
	fifo.mutex.Unlock()
	fifo.condWait.Signal()
	return
}
func (fifo *eventFIFO) DumpDebug() {
	debugging.DEBUG_OUT("[DUMP eventFIFO]>>>> ")
	for _, item := range fifo.q {
		debugging.DEBUG_OUT("[%+v] ", item)
		item = item
	}
	debugging.DEBUG_OUT("\n")
}
func (fifo *eventFIFO) Shutdown() {
	fifo.mutex.Lock()
	fifo.shutdown = true
	fifo.mutex.Unlock()
	fifo.condWait.Broadcast()
	fifo.condFull.Broadcast()
}
func (fifo *eventFIFO) WakeupAll() {
	debugging.DEBUG_OUT(" >>>>>>>>>>> in WakeupAll @Lock\n")
	fifo.mutex.Lock()
	debugging.DEBUG_OUT(" +++++++++++ in WakeupAll\n")
	fifo.wakeupIter++
	fifo.mutex.Unlock()
	debugging.DEBUG_OUT(" +++++++++++ in WakeupAll @Unlock\n")
	fifo.condWait.Broadcast()
	fifo.condFull.Broadcast()
	debugging.DEBUG_OUT(" <<<<<<<<<<< in WakeupAll past @Broadcast\n")
}
func (fifo *eventFIFO) IsShutdown() (ret bool) {
	fifo.mutex.Lock()
	ret = fifo.shutdown
	fifo.mutex.Unlock()
	return
}

// end generic
//////////////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////
