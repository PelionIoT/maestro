package sysstats

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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/armPelionEdge/maestro/debugging"
	"github.com/armPelionEdge/maestro/events"
	"github.com/armPelionEdge/maestro/log"
)

const (
	// SmallestAccpetableInterval is the smallest interval at which a
	// periodic stat can be generated
	SmallestAccpetableInterval time.Duration = 400 * time.Millisecond
	logPx                                    = "sysstats: "
	// used for ctlChan in SysStats
	ctlWakeup   = 1
	ctlShutdown = 2
	// for events channel
	statEventsChannelName = "sysstats"
)

func dummy() {
	// What happens when your compiler developers insist on warnings being errors.
	// Thanks golang team.
	fmt.Sprintf("")
}

// the channel name for events
var statEventsChannel string

type statConfig struct {
	// Name is a name used to identify the stat
	Name string `yaml:"name" json:"name"`
	// Every is a string representation of the interval this stat should be gathered at.
	// The string should be formatted to be compatiable with time.ParseDuration's
	// format. So "1m" for every minute, "30s", "1m45s", "250ms"
	// It can't be below
	Every string `yaml:"every" json:"every"`
	// Disable the stat entirely
	Disable bool `yaml:"disable" json:"disable"`
	// this is Every represented as time.Duration
	interval time.Duration
	// used internally to track the number of times we have
	// updated this stat
	count int64
	// pace is computed based on the manager's "masterInverval"
	// if pace is 1, then the stat is calculated on every turn of the master interval
	// if 3, then every third turn. If 7, then every 7th turn
	pace int64
	// configVer is just incremented if the config changes
	// not currently used
	configVer int32
}

// type DiskConfig struct {
// 	statConfig
// }

// StatsConfig is used to configure the sysstats (system stats) subsystem
type StatsConfig struct {
	//	loopInterval int64       `yaml:"loop_interval" json:"loop_interval"`
	VMStats   *VMConfig   `yaml:"vm_stats" json:"vm_stats"`
	DiskStats *DiskConfig `yaml:"disk_stats" json:"disk_stats"`
}

func SubscribeToEvents() (ret events.EventSubscription, err error) {
	ret, err = events.SubscribeToChannel(statEventsChannel, 0)
	return
}

func (stat *statConfig) Validate() (ok bool, err error) {
	ok = true
	debugging.DEBUG_OUT("ParseDuration:<%s>\n", stat.Every)
	if len(stat.Every) < 1 {
		ok = false
		stat.Disable = true
		err = fmt.Errorf("No 'every' field for stat config %s", stat.Name)
		return
	}
	stat.interval, err = time.ParseDuration(stat.Every)
	if err == nil {
		if stat.interval < SmallestAccpetableInterval {
			stat.interval = SmallestAccpetableInterval
		}
	} else {
		ok = false
		stat.Disable = true
	}
	if stat.Disable {
		ok = false
	}
	return
}

// type ProcessStatsConfig interface {
// 	GetInterval() uint32
// 	GetConfig_CheckMem() (uint32, bool)
// }

type SysStats struct {
	masterInterval time.Duration
	counter        int64
	statsConfig    *StatsConfig

	innerMut sync.Mutex
	running  bool

	ctlChan chan int

	statCallers sync.Map
}

// type event struct {
// 	Name string `json:"name"`
// 	When int64  `json:"when"`
// }

// type MemStatEvent struct {
// 	event                        // base class
// 	stat  *mem.VirtualMemoryStat // straigh outta here: https://github.com/armPelionEdge/gopsutil/blob/master/mem/mem.go
// }

// func NewVirtualMemEvent(stats *mem.VirtualMemoryStat) (ret *MemStatEvent) {
// 	ret = new(MemStatEvent)
// 	ret.Name = "VirtualMemory"
// 	ret.When = time.Now().UnixNano()
// 	ret.stat = stats
// 	return
// }

// func NewVirtualMemEvent(stats *mem.VirtualMemoryStat) (ret *MemStatEvent) {
// 	ret = new(MemStatEvent)
// 	ret.Name = "VirtualMemory"
// 	ret.When = time.Now().UnixNano()
// 	ret.stat = stats
// 	return
// }

// StatPayload carries the stat along with a 'name'
type StatPayload struct {
	Name     string
	StatData interface{}
}

// a convenience functions. returns true if we keep going
func continueCheckContext(ctx context.Context) (keepgoing bool, err error) {
	keepgoing = true
	select {
	case <-ctx.Done():
		err = ctx.Err()
		keepgoing = false
		log.MaestroInfo("Stats: ctx.Done")
	default:
	}
	return
}

// MarshalJSON for StatPayload
// Proper format is: "{<Name>": { <JSON STAT DATA> }}
func (pl *StatPayload) MarshalJSON() (ret []byte, err error) {
	var buffer bytes.Buffer
	debugging.DEBUG_OUT("RMI calling pl.MarshalJSON()\n")
	// write the "timestamp" as ms since epoch
	buffer.WriteString(fmt.Sprintf(`{%s:`, strconv.QuoteToASCII(pl.Name)))
	if pl.StatData != nil {
		byts, err2 := json.Marshal(pl.StatData)
		if err2 != nil {
			err = err2
			return
		}
		if len(byts) < 1 {
			err = errors.New("bad encode of sysstat data")
			return
		}
		// encoding the MaestroEvent will result in an object with {"data":<DATA>}
		// so, to speed things up, we use this, and just clip the first '{'
		buffer.Write(byts) // add in the data, but not the enclosing '{'
		buffer.Write([]byte("}"))
		ret = buffer.Bytes()
		debugging.DEBUG_OUT("StatPayload RMI encode %s\n", string(ret))
	} else {
		ret = nil
		err = errors.New("StatData field of sysstat was nil")
	}
	return
}

// RunnableStat is a way for us to generically run the stats
type RunnableStat interface {
	// RunStat get the latest stat for this stat. The returned interface{}
	// should be a json encodable
	RunStat(ctx context.Context, configChanged bool) (*StatPayload, error)
	GetPace() int64
}

// tell manager to get a single stat by name, one time. This will block
// until the stat is returned. No event will be generated
func (mgr *SysStats) OneTimeRun(name string) (stat *StatPayload, err error) {
	rstatP, ok := mgr.statCallers.Load(name)

	if ok {
		rstat, ok := rstatP.(RunnableStat)
		if ok {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Millisecond)
			defer cancel()
			stat, err = rstat.RunStat(ctx, true)
		} else {
			err = errors.New("corruption in internal stat runner map")
		}
	} else {
		err = errors.New("No stat exists")
	}
	return
}

func (mgr *SysStats) periodicRunner() {
	configChanged := true

	var _callStat = func(key, stat interface{}) bool {
		debugging.DEBUG_OUT("sysstats - in _callStat()\n")
		name, ok2 := key.(string)
		rstat, ok := stat.(RunnableStat)
		if ok && ok2 {
			log.MaestroDebugf(logPx+"running stat %s\n", name)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Millisecond)
			defer cancel()
			data, err := rstat.RunStat(ctx, configChanged)
			// get stat - put in events channel queue
			if err == nil {
				if data != nil {
					// TODO - flatten event data
					debugging.DEBUG_OUT("sysstats - stat data: %+v\n", data)
					ev := &events.MaestroEvent{Data: data}
					events.SubmitEvent([]string{statEventsChannel}, ev)
				} else {
					log.MaestroErrorf(logPx+"Error running stat \"%s\" - data nil!\n", name)
				}
			} else {
				log.MaestroErrorf(logPx+"Error running stat \"%s - %s\n", name, err.Error())
			}
		}
		return true
	}

	mgr.innerMut.Lock()
	if mgr.running {
		mgr.innerMut.Unlock()
		return
	}
	mgr.running = true
	mgr.innerMut.Unlock()

	pacer := time.NewTicker(mgr.masterInterval)

mainloop:
	for {
		debugging.DEBUG_OUT("sysstats - top of main loop\n")
		select {
		case code := <-mgr.ctlChan:
			switch code {
			case ctlShutdown:
				break mainloop
			case ctlWakeup:
				continue
			}
		case <-pacer.C:
			mgr.counter++

			log.MaestroDebug(logPx + "Running all stats.")
			mgr.statCallers.Range(_callStat)
			log.MaestroDebug(logPx + "stat runs complete.")
		}
		// right now we don't support changing the stats configs on the fly.
		// so just call them with configChanged == true, only the first time.
		if configChanged {
			configChanged = false
		}
	}

}

var instance *SysStats

// GetManager returns the singleton manager for Sysstats subsystem
func GetManager() *SysStats {
	if instance == nil {
		instance = new(SysStats)
		instance.ctlChan = make(chan int)
		// create our event channel
		events.OnEventManagerReady(func() error {
			// The main NetEventChannel is a fanout (separate queues for each subscriber) and is non-persistent
			var ok bool
			var err error
			ok, statEventsChannel, err = events.MakeEventChannel(statEventsChannelName, true, false)
			if err != nil || !ok {
				if err != nil {
					log.MaestroErrorf("NetworkManager: Can't create stat event channel. Not good. --> %s\n", err.Error())
				} else {
					log.MaestroErrorf("NetworkManager: Can't create stat event channel. Not good. Unknown Error\n")
				}
			}
			return err
		})
	}
	return instance
}

func (mgr *SysStats) ReadConfig(config *StatsConfig) (ok bool, err error) {
	// validate the config
	ok = true

	var checkStat = func(stat *statConfig) {
		if mgr.masterInterval == 0 {
			mgr.masterInterval = stat.interval
		}
		if stat.interval < mgr.masterInterval {
			mgr.masterInterval = stat.interval
		}
	}
	var computePace = func(stat *statConfig) {
		if stat.interval > 0 {
			stat.pace = int64(stat.interval / mgr.masterInterval)
			if stat.pace < 1 {
				stat.pace = 1
			}
		}
	}

	// find the smallest interval
	var statok bool
	if config.DiskStats != nil {
		debugging.DEBUG_OUT("sysstats - DiskStats: %+v\n", config.DiskStats)
		statok, err = config.DiskStats.Validate()
		debugging.DEBUG_OUT("sysstats - DiskStats out: %+v\n", config.DiskStats)
		if err != nil {
			ok = false
			log.MaestroErrorf(logPx+"disk stats are misconfigured: %s\n", err.Error())
		} else {
			if ok {
				ok = statok
			}
			checkStat(&config.DiskStats.statConfig)
		}
	}
	if config.VMStats != nil {
		debugging.DEBUG_OUT("sysstats - VMStats: %+v\n", config.VMStats)
		statok, err = config.VMStats.Validate()
		debugging.DEBUG_OUT("sysstats - VMStats out: %+v\n", config.VMStats)
		if err != nil {
			ok = false
			log.MaestroErrorf(logPx+"VM stats are misconfigured: %s\n", err.Error())
		} else {
			if ok {
				ok = statok
			}
			checkStat(&config.VMStats.statConfig)
		}
	}
	if mgr.masterInterval < SmallestAccpetableInterval {
		mgr.masterInterval = SmallestAccpetableInterval
	}
	mgr.statCallers = sync.Map{}
	// ok, now compute the pace for each stat
	if config.DiskStats != nil {
		computePace(&config.DiskStats.statConfig)
		debugging.DEBUG_OUT("sysstats - DiskStats out: %+v\n", config.DiskStats)
		mgr.statCallers.Store(config.DiskStats.statConfig.Name, RunnableStat(config.DiskStats))
	}
	if config.VMStats != nil {
		computePace(&config.VMStats.statConfig)
		debugging.DEBUG_OUT("sysstats - VMStats out2: %+v\n", config.VMStats)
		mgr.statCallers.Store(config.VMStats.statConfig.Name, RunnableStat(config.VMStats))
	}
	return
}

func (mgr *SysStats) Start() (err error) {
	go mgr.periodicRunner()
	return
}
