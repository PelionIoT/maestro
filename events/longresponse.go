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
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/armPelionEdge/httprouter"
)

//type wakeupType *MaestroEvent

type ResponderData struct {
	Writer  http.ResponseWriter
	Request *http.Request
	Params  httprouter.Params
}

const (
	// the max amount of queued up http request we will hold without blocking
	// for any given subscription ID (typically only one subscription ID should be used per http caller)
	maxQueuedHttpReqPerSubId = 10
)

// DeferredResponder is anything which might respond later
// must have a golang chan
type DeferredResponder struct {
	manager *DeferredResponseManager
	dat     ResponderData
	sub     EventSubscription
	// called when new events are ready to handle
	responseCallback EventCallback
	closedCallback   ResponderSubscriptionClosedCallback
	id               string
	evchan           chan *MaestroEvent
	rcase            reflect.SelectCase
	// the index of the Responder in the arrays in DeferredResponseManager
	index int
	// protects all data above:
	mutex sync.Mutex
}

type DeferredResponseManager struct {
	// element 0 is always the wakeup channel (does nothing - just to wakeup the block)
	evchans []chan *MaestroEvent
	cases   []reflect.SelectCase
	//	responders []*DeferredResponder
	// a slice of Subscription IDs which is maintained exaclty in the same order as above
	evIDs             []string
	waitingOnChannels bool
	running           bool
	requestStop       bool
	wakeupChan        chan *MaestroEvent
	wakeupCase        reflect.SelectCase

	// protects above data
	chanDatMutex sync.Mutex
	// all subscriptions that are managed
	// Key: Subscription-ID Value: *DeferredResponder
	responders sync.Map
	// http handlers by pathid: *DeferredResponderHandlerChalk
	httpChalks sync.Map
	// subHttpHandler that having waiting connections are tracked here
	// sub ID : *handlerList
	waitingHandlers sync.Map
}

// DeferredResponsePreHandler callback is called by DeferredResponseManager when acting as an
// http.Handler. The function should return the subscription id, or an error if the subscription id
// cannot be determined
type DeferredResponsePreHandler func(http.ResponseWriter, *http.Request) (error, string)

// DeferredResponsePreHandler3 callback is called by DeferredResponseManager when acting as an
// httprouter router handler. The function should return the subscription id, or an error if the subscription id
// cannot be determined
type DeferredResponsePreHandler3 func(http.ResponseWriter, *http.Request, httprouter.Params) (error, string)

type HttpHandler func(http.ResponseWriter, *http.Request)
type HttpHandler3 func(http.ResponseWriter, *http.Request, httprouter.Params)

type DeferredResponderHandlerChalk struct {
	pathid        string
	manager       *DeferredResponseManager
	preHandler    DeferredResponsePreHandler
	handler       HttpHandler
	customHandler HttpHandler
	errorHandler  HttpHandler
	// for use with httprouter
	preHttprouterHandler DeferredResponsePreHandler3
	handler3             HttpHandler3
	//	waitingReqs []*subHttpHandler
	// sub ID: subHttpHandler
	//	waitingReqs sync.Map
	//	disabled     bool // true when the http handler has closed, etc. We leave it in the map for efficiency reasons
	//	evchan       chan *MaestroEvent
}

type handlerList struct {
	// a thread-safe FIFO of subHttpHandler(s)
	fifo *subHttpHandlerFIFO
}

// every time a caller comes in, and we wait, one of these is created
type subHttpHandler struct {
	//	subid string
	parent   *DeferredResponderHandlerChalk
	timedout bool
	// a channel which takes slices of events
	evChan chan []*MaestroEvent
}

// EventCallback is the callback definition to implements when you want to send / process events when they are ready
// asynchronously via a callback. Return remove at true, to remove the Subscription and stop being called. Return stop as true
// if you want a pause in the callback being called
type EventCallback func(dat *ResponderData, id string, events []*MaestroEvent) //(stop bool, remove bool)

// ResponderSubscriptionClosedCallback is called when the DeferredResponseManager can no longer access the EventSubscription b/c it was closed
type ResponderSubscriptionClosedCallback func(dat *ResponderData, id string)

// its either a matter of maintaining another map for N to string,
// which would have to be fixed everytime there was a delete in list of responders,
// or just look for the ID when we need to. This is not a common call
// so this seems more efficient.
// func (manager *DeferredResponseManager) findId(searchid string) (found bool, n int) {
// 	n = 0
// 	var check = func(key, val interface{}) bool {
// 		id, ok := key.(string)
// 		val, ok2 := val.(*DeferredResponder)
// 		if ok && ok2 {
// 			if id == searchid {
// 				found = true
// 				return false
// 			}
// 		}
// 		n++
// 		return true
// 	}
// 	manager.responders.Range(check)
// 	return
// }

// Requires a chantDatMutex.Lock() before a call
func (manager *DeferredResponseManager) rebuildIndex() {
	n := int(0)
	manager.cases = make([]reflect.SelectCase, 0, 10)
	manager.evchans = make([]chan *MaestroEvent, 0, 10)
	manager.evIDs = make([]string, 0, 10)
	// add wakeup channel - always 0
	manager.cases = append(manager.cases, manager.wakeupCase)
	manager.evchans = append(manager.evchans, manager.wakeupChan)
	manager.evIDs = append(manager.evIDs, "??")

	n++
	var doeach = func(key, val interface{}) bool {
		_, ok := key.(string)
		resp, ok2 := val.(*DeferredResponder)
		if ok && ok2 {
			resp.mutex.Lock()
			manager.cases = append(manager.cases, resp.rcase)
			manager.evchans = append(manager.evchans, resp.evchan)
			manager.evIDs = append(manager.evIDs, resp.id)
			resp.index = n
			resp.mutex.Unlock()
			n++
		}
		return true
	}
	manager.responders.Range(doeach)
	debug_out("rebuildIndex() added in %d entries", n-1)
	return
}

func (manager *DeferredResponseManager) GetResponder(id string) (responder *DeferredResponder, ok bool) {
	var rp interface{}
	rp, ok = manager.responders.Load(id)
	if ok {
		responder, ok = rp.(*DeferredResponder)
	}
	return
}

// Requires a chantDatMutex.Lock() before a call
func (manager *DeferredResponseManager) removeByN(n int) (ret *DeferredResponder) {
	id := manager.evIDs[n]

	retp, ok := manager.responders.Load(id)
	if ok {
		ret, ok = retp.(*DeferredResponder)
		manager.responders.Delete(id)
		manager.rebuildIndex()
	}
	return
}

// Requires a chantDatMutex.Lock() before a call
func (manager *DeferredResponseManager) removeById(id string) (ret *DeferredResponder) {
	retp, ok := manager.responders.Load(id)
	if ok {
		ret, ok = retp.(*DeferredResponder)
		manager.responders.Delete(id)
		manager.rebuildIndex()
	}
	return
}

func (manager *DeferredResponseManager) RemoveSubscriptionById(id string) {
	_, ok := manager.responders.Load(id)
	if ok {
		manager.chanDatMutex.Lock()
		manager.responders.Delete(id)
		manager.rebuildIndex()
		manager.chanDatMutex.Unlock()
	}
	return
}

// to understand this insanity, please refer to this:
// https://gist.github.com/timstclair/dd6df610c7c372de66a9#file-dynamic_select_test-go-L115
// and https://stackoverflow.com/questions/19992334/how-to-listen-to-n-channels-dynamic-select-statement
func (manager *DeferredResponseManager) waitOnChannels() {

	var responseCallWrapper = func(dat *ResponderData, responder *DeferredResponder, events []*MaestroEvent) {
		responder.mutex.Lock()
		if responder.responseCallback != nil {
			responder.responseCallback(dat, responder.id, events)
		}
		responder.mutex.Unlock()
	}

	// called for each handler in the manager.httpHandlers map
	// var submitToEachHandler(pathidI, chalkI interface{}) bool {
	// 	pathid, ok := pathidI.(string)
	// 	handlerChalk, ok2 := chalkI.(*DeferredResponderHandlerChalk)
	// 	if ok && ok2 {

	// 	} else {
	// 		debug_out("OUCH - SHOULD NOT HAPPEN - bad cast from chalk map\n")
	// 	}
	// }

	// sent is true if events has any events and callback was called
	// closed is true if the channel has closed and should be removed
	var checkSub = func(id string, events []*MaestroEvent) (sent bool, closed bool, responder *DeferredResponder) {
		// we know the Subscription index which has new data,
		// some sanity checks
		responderp, ok := manager.responders.Load(id)
		if ok {
			// ok, we have a valid EventSubscription
			responder = responderp.(*DeferredResponder)
			if responder == nil || responder.sub == nil {
				closed = true
				return
			}
			ok2, ch := responder.sub.GetChannel()
			if ok2 {
			readloop:
				for {
					debug_out("top of readloop:")
					select {
					case ev, ok3 := <-ch:
						if !ok3 {
							// channel gone / closed.
							// Should be unreachable - GetChannel()
							closed = true
							break readloop
						} else {
							debug_out("sub got events")
							// we know the Subscription index which has new data,
							// some sanity checks
							events = append(events, ev)
						}
					default:
						// non-blocking, we are just checking if they are they have any data
						// okay, no more events. We should spawn a callback
						if len(events) > 0 {
							debug_out("checkSub() have events")
							// first handle callback if there is one
							if responder.responseCallback != nil {
								debug_out("have responseCallback - calling it")
								// do a callback in another goroutine
								sent = true
								dat := new(ResponderData)
								*dat = responder.dat
								go responseCallWrapper(dat, responder, events)
							}
							// next handle all http handlers, if there are any
							listI, ok3 := manager.waitingHandlers.Load(id)
							if ok3 {
								debug_out("got a waitingHandlers list for this id:%s", id)
								list, ok4 := listI.(*handlerList)
								if ok4 {
									for {
										handler := list.fifo.Pop()
										if handler != nil {
											if !handler.timedout {
												select {
												// don't block. If the handler is gone or can't take a
												// an event, forget it
												case handler.evChan <- events:
													debug_out("handler.evChan <- events")
												default:
													debug_out("dropping timed out http handler + events (would block)")
												}
											} else {
												debug_out("dropping timed out http handler")
											}
											// done, this channel and subHttpHandler won't be used again
											close(handler.evChan)
										} else {
											break
										}
									}
								}
							} else {
								debug_out("waitingHandlers for this id:%s", id)
							}
						}
						break readloop
					}
				}
				responder.sub.ReleaseChannel()
			} else {
				debug_out("Note - channel closed %s", responder.sub.GetID())
				// TODO remove channel - its been closed
				closed = true
				// removeChan(n)
			}
		}
		return
	}

	manager.chanDatMutex.Lock()
	manager.running = true
	manager.chanDatMutex.Unlock()

mainLoop:
	for {
		debug_out("DeferredResponseManager.waitOnChannels - top of mainLoop")
		// First try to pull from each channel.
		manager.chanDatMutex.Lock()
		for i, id := range manager.evIDs {
			events := make([]*MaestroEvent, 0)
			_, closed, resp := checkSub(id, events)
			if closed {
				if resp.closedCallback != nil {
					dat := new(ResponderData)
					*dat = resp.dat
					go resp.closedCallback(dat, resp.id)
				}
				manager.removeByN(i)
				manager.chanDatMutex.Unlock()
				// we just rebuilt the index, so start over
				continue mainLoop
			}
		}
		manager.chanDatMutex.Unlock()

		// using reflectin we can wait until one of the channels does something
		i, value, ok := reflect.Select(manager.cases)
		manager.chanDatMutex.Lock()
		id := manager.evIDs[i]
		manager.chanDatMutex.Unlock()
		debug_out("DeferredResponseManager.waitOnChannels woke up %d %s %v", i, id, ok)
		if !ok {
			manager.chanDatMutex.Lock()
			manager.removeByN(i)
			manager.chanDatMutex.Unlock()
			// we just rebuilt the index, so start over
			continue mainLoop
		} else {
			events := make([]*MaestroEvent, 0)
			// if i == 0 - then this was just a wakeup  - skip it
			if i != 0 {
				// look weird? read (5) here https://golang.org/pkg/unsafe/#Pointer
				// doing gymanstics b/c go hates real pointers
				ev := (*MaestroEvent)(unsafe.Pointer(value.Pointer()))
				// append the first event we got through select
				events = append(events, ev)
				// FIXME actually we should use the ID on the event object, which has no likelyhood of race problem
				// versus the evIDs array
				_, closed, resp := checkSub(id, events)
				if closed {
					if resp.closedCallback != nil {
						dat := new(ResponderData)
						*dat = resp.dat
						go resp.closedCallback(dat, resp.id)
					}
					manager.removeByN(i)
				}
				//				manager.chanDatMutex.Unlock()
			} else {
				// n == 0 is the wakeup channel. Just clear out the channel
				for {
					select {
					case _, ok := <-manager.wakeupChan: //.evchans[0]:
						if !ok {
							debug_out("wakeup channel was !ok - error")
							return
						}
						continue
					default:
						debug_out("wakeup channel default")
						continue mainLoop
					}
				}
			}
		}

		manager.chanDatMutex.Lock()
		if manager.requestStop {
			manager.requestStop = false
			manager.chanDatMutex.Unlock()
			break
		}
		manager.chanDatMutex.Unlock()
	}

	manager.chanDatMutex.Lock()
	manager.running = false
	manager.chanDatMutex.Unlock()

}

func (manager *DeferredResponseManager) wakeup() {
	manager.chanDatMutex.Lock()
	if manager.running {
		p := &MaestroEvent{}
		// send something to the wakeup channel
		manager.evchans[0] <- p
	}
	manager.chanDatMutex.Unlock()
}

// refAllSubs references each subscription using the GetChannel call
func (manager *DeferredResponseManager) refAllSubs() {
	var doeach = func(key, val interface{}) bool {
		id, ok := key.(string)
		resp, ok2 := val.(*DeferredResponder)
		if ok && ok2 {
			resp.mutex.Lock()
			ok, _ = resp.sub.GetChannel()
			if !ok {
				debug_out("!! GetChannel() failed in refAllSubs() (%s)", id)
			}
			resp.mutex.Unlock()
		}
		return true
	}
	manager.responders.Range(doeach)
}

// releaseAllSubs unreferences each subscription using the ReleaseChannel call
func (manager *DeferredResponseManager) releaseAllSubs() {
	var doeach = func(key, val interface{}) bool {
		_, ok := key.(string)
		resp, ok2 := val.(*DeferredResponder)
		if ok && ok2 {
			resp.mutex.Lock()
			resp.sub.ReleaseChannel()
			resp.mutex.Unlock()
		}
		return true
	}
	manager.responders.Range(doeach)
}

// AddEventSubscription adds a new EventSubscription object, along with a callback which
// should be called when
func (manager *DeferredResponseManager) AddEventSubscription(sub EventSubscription) (id string, responder *DeferredResponder, err error) {
	ok, subchan := sub.GetChannel() // _ = subchan
	if ok {
		id = sub.GetID()
		responder = new(DeferredResponder)
		responder.id = id
		responder.sub = sub
		responder.manager = manager
		responder.rcase = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(subchan)}

		manager.responders.Store(id, responder)
		debug_out("AddEventSubscription - adding id %s", id)
		// we assigned a new subscription / responder - so call wakeup watcher
		manager.chanDatMutex.Lock()
		manager.rebuildIndex()
		manager.chanDatMutex.Unlock()
		responder.manager.wakeup()
		debug_out("AddEventSubscription - rebuilt index and wokeup manager (%s)", id)
		sub.ReleaseChannel()
	}

	return
}

func (responder *DeferredResponder) AssignHttpCallback(cb EventCallback, w http.ResponseWriter, r *http.Request, ps httprouter.Params) (err error) {
	responder.mutex.Lock()
	responder.responseCallback = cb
	responder.dat.Writer = w
	responder.dat.Request = r
	responder.dat.Params = ps
	responder.mutex.Unlock()
	return
}

// MakeSubscriptionHttpHandler returns a http.Handler which will send out events
// based on the subscrition ID which connects to it. The DeferredResponsePreHandler will
// determine the subscription ID. This function should not block.
// The mainhandler is optional, and may be nil. If nil, then each Data element of every event
// will be JSON encoded as an array and sent when available.
func (mymanager *DeferredResponseManager) MakeSubscriptionHttpHandler(pathid string, mypreHandler DeferredResponsePreHandler, myerrorhandler HttpHandler, mainhandler HttpHandler, timeout time.Duration) (ret http.Handler) {
	chalk := &DeferredResponderHandlerChalk{
		pathid:        pathid,
		manager:       mymanager,
		preHandler:    mypreHandler,
		errorHandler:  myerrorhandler,
		handler:       nil,
		customHandler: nil,
	}
	chalk.handler = func(w http.ResponseWriter, req *http.Request) {
		debug_out("In chalk handler - %s", chalk.pathid)
		// this is the core of the http handler
		err, id := chalk.preHandler(w, req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			errbytes, err := json.Marshal(err.Error())
			if err != nil {
				s := fmt.Sprintf("\"error\":%s", string(errbytes))
				w.Write([]byte(s))
			} else {
				w.Write([]byte("\"error\":\"encode error\""))
			}
			return
		}
		debug_out("preHandler result, id=%s", id)
		// got subscription ID - let's wait for an event with this ID
		//		var responder *DeferredResponder
		_, ok := mymanager.responders.Load(id)
		if !ok {
			// unknown subscription ID
			debug_out("statusNotFound b/c subscription ID %s unknown", id)
			w.WriteHeader(http.StatusNotFound)
			return
		} else {
			//			responder, ok = rp.(*DeferredResponder)
		}
		debug_out("found valid responder")
		// its a valid responder, lookup or create a
		// subHttpHandler
		subhttphandler := &subHttpHandler{
			//			subid: id,
			evChan: make(chan []*MaestroEvent),
			parent: chalk,
		}
		//		subhttphandler, _ = baton.subs.LoadOrStore(id,subhttphandler)
		subhttphandler.timedout = false

		// lookup handlerList or make if not existant and then
		// put the subhttphandler there
		var loaded bool
		var handlerlistI interface{}
		handlerlist := &handlerList{}
		handlerlistI, loaded = mymanager.waitingHandlers.LoadOrStore(id, handlerlist)
		if !loaded {
			debug_out("new handlerList")
			handlerlist.fifo = New_subHttpHandlerFIFO(maxQueuedHttpReqPerSubId)
		} else {
			handlerlist, loaded = handlerlistI.(*handlerList)
			if !loaded {
				debug_out("Type assertion failed at handleListI - dropping out")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		queued := handlerlist.fifo.PushOrWait(subhttphandler)
		if queued {
			// ok - now we will wait
			debug_out("at select{}")
			select {
			// waiting for an event specific to this subscription ID
			case <-time.After(timeout):
				debug_out("select{} timed out")
				subhttphandler.timedout = true
				w.WriteHeader(http.StatusNoContent)
			case events := <-subhttphandler.evChan:
				debug_out("select{} got events")
				// need to define this. should send multiple events
				// optional 'writer' callback, or just do a smart default
				// we could keep continue-ing here until we have enough or enough time has gone by
				// if err == nil {
				if chalk.customHandler != nil {
					debug_out("have handler - using it")
					chalk.customHandler(w, req)
				} else {
					var buf bytes.Buffer
					buf.Write([]byte("["))
					// encode the .Data for each event
					for n, ev := range events {
						if n > 0 {
							buf.Write([]byte(","))
						}
						b, err2 := json.Marshal(ev)
						if err2 == nil {
							buf.Write(b)
						} else {
							debug_out("Error encoding event!!\n")
						}
					}
					buf.Write([]byte("]"))
					debug_out("wrote: %s", buf.String())
					w.Write(buf.Bytes())
					w.WriteHeader(http.StatusOK)
				}
			}
			// done. this subhttphandler won't be used again
			// } else {
			// 	if ret.errorHandler != nil {
			// 		ret.errorHandler(w, req)
			// 	} else {
			// 		// need default error
			// 		w.WriteHeader(http.StatusInternalServerError)
			// 	}
			// }
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	mymanager.httpChalks.Store(pathid, chalk)
	ret = chalk
	return
}

func (mymanager *DeferredResponseManager) MakeSubscriptionHttprouterHandler(pathid string, mypreHandler DeferredResponsePreHandler3, myerrorhandler HttpHandler, mainhandler HttpHandler, timeout time.Duration) (ret httprouter.Handle) {
	chalk := &DeferredResponderHandlerChalk{
		pathid:               pathid,
		manager:              mymanager,
		preHttprouterHandler: mypreHandler,
		errorHandler:         myerrorhandler,
		handler3:             nil,
		customHandler:        nil,
	}
	chalk.handler3 = func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		debug_out("In chalk handler - %s", chalk.pathid)
		// this is the core of the http handler
		err, id := chalk.preHttprouterHandler(w, req, ps)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			errbytes, err := json.Marshal(err.Error())
			if err != nil {
				s := fmt.Sprintf("\"error\":%s", string(errbytes))
				w.Write([]byte(s))
			} else {
				w.Write([]byte("\"error\":\"encode error\""))
			}
			return
		}
		debug_out("preHandler result, id=%s", id)
		// got subscription ID - let's wait for an event with this ID
		//		var responder *DeferredResponder
		_, ok := mymanager.responders.Load(id)
		if !ok {
			// unknown subscription ID
			debug_out("statusNotFound b/c subscription ID %s unknown", id)
			w.WriteHeader(http.StatusNotFound)
			return
		} else {
			//			responder, ok = rp.(*DeferredResponder)
		}
		debug_out("found valid responder")
		// its a valid responder, lookup or create a
		// subHttpHandler
		subhttphandler := &subHttpHandler{
			//			subid: id,
			evChan: make(chan []*MaestroEvent),
			parent: chalk,
		}
		//		subhttphandler, _ = baton.subs.LoadOrStore(id,subhttphandler)
		subhttphandler.timedout = false

		// lookup handlerList or make if not existant and then
		// put the subhttphandler there
		var loaded bool
		var handlerlistI interface{}
		handlerlist := &handlerList{}
		handlerlistI, loaded = mymanager.waitingHandlers.LoadOrStore(id, handlerlist)
		if !loaded {
			debug_out("new handlerList")
			handlerlist.fifo = New_subHttpHandlerFIFO(maxQueuedHttpReqPerSubId)
		} else {
			handlerlist, loaded = handlerlistI.(*handlerList)
			if !loaded {
				debug_out("Type assertion failed at handleListI - dropping out")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		mymanager.refAllSubs()

		queued := handlerlist.fifo.PushOrWait(subhttphandler)
		if queued {
			// ok - now we will wait
			debug_out("at select{}")
			select {
			// waiting for an event specific to this subscription ID
			case <-time.After(timeout):
				debug_out("select{} timed out")
				subhttphandler.timedout = true
				w.WriteHeader(http.StatusNoContent)
			case events := <-subhttphandler.evChan:
				// need to define this. should send multiple events
				// optional 'writer' callback, or just do a smart default
				// we could keep continue-ing here until we have enough or enough time has gone by
				// if err == nil {
				if chalk.customHandler != nil {
					debug_out("have handler - using it")
					chalk.customHandler(w, req)
				} else {
					var buf bytes.Buffer
					buf.Write([]byte("["))
					// encode the .Data for each event
					for n, ev := range events {
						if n > 0 {
							buf.Write([]byte(","))
						}
						b, err2 := json.Marshal(ev)
						if err2 == nil {
							buf.Write(b)
						} else {
							debug_out("Error encoding event!!\n")
						}
					}
					buf.Write([]byte("]"))
					debug_out("wrote: %s", buf.String())
					w.Write(buf.Bytes())
					w.WriteHeader(http.StatusOK)
				}
			}
			// done. this subhttphandler won't be used again
			// } else {
			// 	if ret.errorHandler != nil {
			// 		ret.errorHandler(w, req)
			// 	} else {
			// 		// need default error
			// 		w.WriteHeader(http.StatusInternalServerError)
			// 	}
			// }
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		mymanager.releaseAllSubs()
	}
	mymanager.httpChalks.Store(pathid, chalk)
	ret = httprouter.Handle(chalk.handler3)
	return
}

// ServeHTTP implements the http.Handler interface, so the DeferredResponder can itself
// be the http.Handler for a path / route on the server
func (chalk *DeferredResponderHandlerChalk) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	debug_out("in ServeHTTP()")
	// bear in mind, this call will be made in another goroutine, by the http library
	chalk.handler(w, r)
}

// // Returns a function to use with httprouter as a route handler
// func (responder *DeferredResponder) GetHttpRouteHandler() (ret func(http.ResponseWriter, *http.Request, httprouter.Params)) {
// 	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
// 		// bear in mind, this call will be made in another goroutine, by the http library
// 		responder.ServeHTTP(w, r)
// 	}
// }

func (responder *DeferredResponder) AssignCloseCallback(cb ResponderSubscriptionClosedCallback) (err error) {
	responder.mutex.Lock()
	responder.closedCallback = cb
	responder.mutex.Unlock()
	return
}

func NewDeferredResponseManager() (ret *DeferredResponseManager) {
	ret = new(DeferredResponseManager)
	ret.wakeupChan = make(chan *MaestroEvent)
	//	wakeupid := "??"
	ret.wakeupCase = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ret.wakeupChan)}
	ret.rebuildIndex()
	return
}

func (manager *DeferredResponseManager) Start() {
	manager.chanDatMutex.Lock()
	if !manager.waitingOnChannels {
		go manager.waitOnChannels()
	}
	manager.chanDatMutex.Unlock()
}

func (manager *DeferredResponseManager) Stop() {
	manager.chanDatMutex.Lock()
	if manager.waitingOnChannels {
		manager.requestStop = true
		// should trigger the reflect.Select()
		close(manager.evchans[0])
	}
	manager.chanDatMutex.Unlock()
}
