package wwrmi

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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/armPelionEdge/maestro/debugging"
	"github.com/armPelionEdge/maestro/events"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/utils"
)

func _dummy() { fmt.Printf("") }

type statSender struct {
	client      *Client
	readOngoing bool
	// we don't need to hold "empty" stats struct, so just these two
	// stats to send
	willSendStatsFifo *statsFIFO
	// stats which are bing sent (but might fail and go back into willSendFifo)
	sendingStatsFifo *statsFIFO

	sentBytes uint32
	// max amount of time to go by before we send stats
	sendTimeThreshold uint32
	countThreshold    uint32
	cmdChan           chan int
	waitStart         *sync.Cond
	locker            sync.Mutex
	running           bool
	backoff           time.Duration
	backingOff        bool
	statsChannel      chan *events.MaestroEvent
}

func newStatSender(c *Client) (ret *statSender) {
	ret = &statSender{
		client: c,
		// sysstats FIFOs. We don't need "available" FIFO, b/c
		// stats just use the normal heap
		willSendStatsFifo: New_statsFIFO(c.sysStatsMaxQueued),
		sendingStatsFifo:  New_statsFIFO(c.sysStatsMaxQueued),
		sendTimeThreshold: c.sysStatsTimeThreshold,
		countThreshold:    c.sysStatsCountThreshold,
		cmdChan:           make(chan int, 1),
	}
	ret.waitStart = sync.NewCond(&ret.locker)
	return
}

func (sndr *statSender) Read(p []byte) (copied int, err error) {
	remain := len(p)
	buflen := 0
	if remain < 1 {
		err = io.EOF
		return
	}
	firstitem := true
	if !sndr.readOngoing {
		copy(p[copied:], []byte("["))
		copied++
	} else {
		firstitem = false
	}
	sndr.readOngoing = true
	stat := sndr.willSendStatsFifo.Peek()
	for stat != nil {
		debugging.DEBUG_OUT("RMI Read() (stats) to of loop\n")
		// get the JSON encoding of the stat in a []byte - or use
		// a cached version if we already did this
		var jsonbuf []byte
		var err2 error
		if len(stat.jsonEncode) < 1 {
			jsonbuf, err2 = json.Marshal(stat)
			if err2 != nil {
				sndr.willSendStatsFifo.Pop()
				stat = sndr.willSendStatsFifo.Peek()
				log.MaestroErrorf("RMI - could not json encode stat. dropping. [%s]", err2.Error())
				continue
			}
			stat.jsonEncode = jsonbuf
		} else {
			// used cached value (a resend)
			jsonbuf = stat.jsonEncode
		}

		buflen = len(jsonbuf) // need room for ending ']'
		if buflen > remain {
			// no more room in this Read's p[]
			sndr.sentBytes += uint32(copied - 1)
			// note, the stat remains in willSendStatsFifo
			return
		}

		stat = sndr.willSendStatsFifo.Pop()
		// keep the stat in case of failure in other FIFO
		sndr.sendingStatsFifo.Push(stat)
		if !firstitem {
			copy(p[copied:], []byte(","))
			copied++
		}
		copy(p[copied:], jsonbuf)
		copied += buflen
		remain -= buflen
		stat = sndr.willSendStatsFifo.Peek()
		firstitem = false
	}
	// we ran out of buffers, so say EOF

	debugging.DEBUG_OUT("RMI Read() (stats) EOF\n")
	copy(p[copied:], []byte("]")) // replace last ',' with ']'
	debugging.DEBUG_OUT("RMI stat sending: %s\n", string(p))
	copied++
	sndr.sentBytes += uint32(copied)
	sndr.readOngoing = false
	err = io.EOF
	return
}

func (sndr *statSender) submitStat(s *statWrapper) (dropped bool) {
	dropped, _ = sndr.willSendStatsFifo.Push(s)
	//  check to see if we are at the threshold to send stats to the cloud
	debugging.DEBUG_OUT("RMI (statsender) wilLSendStatsFifo.Len() = %d\n", sndr.willSendStatsFifo.Len())
	if sndr.willSendStatsFifo.Len() > int(sndr.countThreshold) {
		// time to send stats - tell our owning Client to do it
		debugging.DEBUG_OUT("RMI (statsender) hit count threshold - cmdChan <- sndrSendNow\n")
		sndr.cmdChan <- sndrSendNow
		debugging.DEBUG_OUT("RMI (statsender) past cmdChan <- sndrSendNow\n")
	}

	return
}

// go routing which send stats when told to
// backs off, when it has problems, trying again later
func (sndr *statSender) sender() {
	var timeout time.Duration
	// var startBackoff time.Time
	var statsChannel chan *events.MaestroEvent

	if sndr.client.sysStatsEventSubscription == nil {
		statsChannel = events.GetDummyEventChannel()
	} else {
		var ok bool
		ok, statsChannel = sndr.client.sysStatsEventSubscription.GetChannel()
		if !ok {
			log.MaestroErrorf("RMI - could not get the events channel for sysstats. Will not report.")
			statsChannel = events.GetDummyEventChannel()
		}
	}

	sndr.locker.Lock()
	if sndr.running {
		sndr.waitStart.Broadcast()
		sndr.locker.Unlock()
		return
	}
	sndr.running = true
	sndr.waitStart.Broadcast()
	timeout = time.Duration(sndr.sendTimeThreshold) * time.Millisecond
	sndr.locker.Unlock()

	var elapsed time.Duration
	var start time.Time
	timedout := false

	var handleErr = func(clienterr error) {
		debugging.DEBUG_OUT("RMI statsender.handleErr(%+v)\n", clienterr)
		timedout = false
		if clienterr != nil {
			// transfer failed, so...
			// move all the buffers back into the willSendFifo
			buf := sndr.sendingStatsFifo.Pop()
			for buf != nil {
				sndr.willSendStatsFifo.Push(buf)
				buf = sndr.sendingStatsFifo.Pop()
			}
			// do a backoff
			debugging.DEBUG_OUT("RMI (statsender) @lock 1\n")
			sndr.locker.Lock()
			debugging.DEBUG_OUT("RMI (statsender) @pastlock 1\n")
			// startBackoff = time.Now()
			// if already backing off...
			if sndr.backingOff && sndr.backoff > 0 {
				sndr.backoff = sndr.backoff * 2
			} else {
				sndr.backoff = time.Duration(sndr.sendTimeThreshold) * time.Millisecond
			}
			if sndr.backoff > maxBackoff {
				sndr.backoff = time.Duration(sndr.sendTimeThreshold) * time.Millisecond
			}
			sndr.backingOff = true
			timedout = true
			sndr.locker.Unlock()
		} else {
			// success. reset backoff
			timeout = time.Duration(sndr.sendTimeThreshold) * time.Millisecond
			sndr.locker.Lock()
			sndr.backoff = 0
			sndr.backingOff = false
			// client.sendableBytes = client.sendableBytes - client.sentBytes
			//sndr.sentBytes = 0
			sndr.locker.Unlock()
		}
	}

sndrLoop:
	for {
		start = time.Now()
		debugging.DEBUG_OUT("RMI top of loop (statsender) %d\n", timeout)
		select {
		case <-time.After(timeout):
			debugging.DEBUG_OUT("RMI (statsender)- sysstats loop got timeout. try to send?\n")
			timedout = true
			if sndr.willSendStatsFifo.Len() > 0 {
				err := sndr.postStats()
				handleErr(err)
			}
		case stat := <-statsChannel:
			wrapper, ok := convertStatEventToWrapper(stat)
			if ok {
				dropped := sndr.submitStat(wrapper)
				if dropped {
					log.MaestroWarnf("RMI - dropping systems stats. Queue is full.")
				}
			}
		case code := <-sndr.cmdChan:
			debugging.DEBUG_OUT("RMI (statsender) - code = %d\n", code)
			switch code {
			case sndrSendNow:
				debugging.DEBUG_OUT("RMI (statsender) @lock 2\n")
				sndr.locker.Lock()
				debugging.DEBUG_OUT("RMI (statsender) @pastlock 2\n")
				if sndr.backingOff {
					sndr.locker.Unlock()
					elapsed = time.Since(start)
					timeout = timeout - elapsed
					if timeout > 10000 {
						debugging.DEBUG_OUT("RMI (statsender) ignoring cmd sndrSendNow (%d)\n", timeout)
						continue sndrLoop
					}
					// here to see how well this is working
					debugging.DEBUG_OUT("RMI (statsender) triggered - (ALMOST IGNORED) cmdSendLogs\n")
				} else {
					sndr.locker.Unlock()
				}
				debugging.DEBUG_OUT("RMI triggered - statSender: sndrSendNow\n")
				err := sndr.postStats()
				handleErr(err)
			case sndrShutdown:
				debugging.DEBUG_OUT("RMI - statsender saw shutdown\n")
				break sndrLoop
			}
		}

		// set up the next timeout.
		// elapsed = time.Since(start)
		// timeout = timeout - elapsed
		// if timeout < 10000 {
		// 	timeout = 10000
		// }

		sndr.locker.Lock()
		// if sndr.backingOff {
		// 	// elapsed = time.Since(startBackoff)
		// 	// timeout = timeout - elapsed
		// 	// if timeout < 10000 {
		// 	// 	timeout = 10000
		// 	// 	sndr.locker.Unlock()
		// 	// }
		// 	// if timedout {
		// 	// if sndr.backoff < 2 {
		// 	// 	sndr.backoff = time.Duration(sndr.sendTimeThreshold) * time.Millisecond
		// 	// } else {
		// 	// 	sndr.backoff = sndr.backoff * 2
		// 	// }
		// 	// if sndr.backoff > maxBackoff {
		// 	// 	sndr.backoff = time.Duration(sndr.sendTimeThreshold) * time.Millisecond
		// 	// }
		// 	timeout = sndr.backoff
		// 	// } else {
		// 	// 	elapsed = time.Since(start)
		// 	// 	timeout = timeout - elapsed
		// 	// 	if timeout < 10000 {
		// 	// 		timeout = 10000
		// 	// 	}
		// 	// }
		// 	debugging.DEBUG_OUT("RMI (statsender) send stats is backing off %d ns\n", timeout)
		// } else {
		if timedout {
			if sndr.backingOff {
				timeout = sndr.backoff
				debugging.DEBUG_OUT("RMI (statsender) send stats is backing off %d ns\n", timeout)
			} else {
				timeout = time.Duration(sndr.sendTimeThreshold) * time.Millisecond
			}
		} else {
			elapsed = time.Since(start)
			timeout = timeout - elapsed
			if timeout < 10000 {
				timeout = 10000
			}
		}
		// }
		sndr.locker.Unlock()
		timedout = false
	}
	debugging.DEBUG_OUT("RMI - statsender sender() routing ending\n")
}

// postStats send sysstats events to the API
func (sndr *statSender) postStats() (err error) {
	var req *http.Request
	var resp *http.Response
	debugging.DEBUG_OUT("RMI POST %s >>>\n", sndr.client.postStatsUrl)
	// Client implements io.Reader's Read(), so we do this
	//client.sentBytes = 0
	req, err = http.NewRequest("POST", sndr.client.postStatsUrl, sndr)
	if err != nil {
		log.MaestroErrorf("Error on POST request: %s\n", err.Error())
		debugging.DEBUG_OUT("RMI ERROR: %s\n", err.Error())
		return
	}
	resp, err = sndr.client.client.Do(req)
	if err != nil {
		log.MaestroErrorf("Error on POST request: %s\n", err.Error())
		debugging.DEBUG_OUT("RMI ERROR: %s\n", err.Error())
		return
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	debugging.DEBUG_OUT("RMI --> response +%v\n", resp)
	if resp != nil && resp.StatusCode != 200 {
		bodystring, _ := utils.StringifyReaderWithLimit(resp.Body, 300)
		log.MaestroErrorf("RMI: Error on POST request for stats: Response was %d (Body <%s>)", resp.StatusCode, bodystring)
		debugging.DEBUG_OUT("RMI bad response - creating error object\n")
		err = newClientError(resp)
		return
	}
	// Read and discard response body
	_, err = io.Copy(ioutil.Discard, resp.Body)

	return
}

func (sndr *statSender) start() {
	sndr.locker.Lock()
	go sndr.sender()
	sndr.waitStart.Wait()
	sndr.locker.Unlock()
}
