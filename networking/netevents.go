package networking

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

	"github.com/armPelionEdge/maestro/events"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestroSpecs/netevents"
)

const (
	netEventChannelName = "netevents"
	// well known event names are in maestroSpecs
)

var netEventChannel string

func submitNetEventData(data *netevents.NetEventData) {
	ev := &events.MaestroEvent{Data: data}
	events.SubmitEvent([]string{netEventChannel}, ev)
}

// SubscribeToNetEvents let's you subscribe to all core network event. A timeout should be provided if your
// channel listener may stop listening to events in the futurer. timeout is in nanoseconds, and is the
// equivalent to time.Duration
func SubscribeToNetEvents(timeout int64) (id string, err error) {
	if len(netEventChannel) > 0 {
		sub, err2 := events.SubscribeToChannel(netEventChannel, timeout)
		if err2 != nil {
			id = sub.GetID()
			sub.ReleaseChannel()
		} else {
			err = err2
		}
	} else {
		err = errors.New("Channel not ready")
	}
	return
}

// GetLastEventsForSubscriber returns the latest events available for the subscriber ID
// if valid is false, then the subscriber is expired
func GetLastEventsForSubscriber(id string) (output []*netevents.NetEventData, valid bool, err error) {
	ok, sub := events.GetSubscription(netEventChannel, id)
	if ok {
		// subscription is still valid
		ok2, subchan := sub.GetChannel()
		if ok2 {
			valid = true
		evloop1:
			for {
				select {
				case ev := <-subchan:
					netev, ok := ev.Data.(*netevents.NetEventData)
					if ok {
						output = append(output, netev)
					} else {
						// wrong
					}
				default:
					// nothing in the channel, break
					break evloop1
				}
			}
			sub.ReleaseChannel() // releases GetChannel() call
		}
	}
	return
}

// GetNeteventsID returns the string ID used to reference network events
// in the event manager,
func GetNeteventsID() (ok bool, ret string) {
	if len(netEventChannel) > 0 {
		ok = true
		ret = netEventChannel
	}
	return
}

// sets up the master network event channel
func init() {
	events.OnEventManagerReady(func() error {
		// The main NetEventChannel is a fanout (separate queues for each subscriber) and is non-persistent
		var ok bool
		var err error
		ok, netEventChannel, err = events.MakeEventChannel(netEventChannelName, true, false)
		if err != nil || !ok {
			if err != nil {
				log.MaestroErrorf("NetworkManager: Can't create network event channel. Not good. --> %s\n", err.Error())
			} else {
				log.MaestroErrorf("NetworkManager: Can't create network event channel. Not good. Unknown Error\n")
			}
		}
		return err
	})
}
