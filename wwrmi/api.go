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
	"net/url"
	"strings"
	"io"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
	"github.com/armPelionEdge/maestro/debugging"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/events"
	"github.com/armPelionEdge/maestro/sysstats"
	"github.com/armPelionEdge/maestro/utils"
	"github.com/armPelionEdge/greasego"
)

const (
	// the maximum amount of times we will try to send a log entry
	maxLogTries = 2
	// in milliseconds
	maxBackoff time.Duration = 240000 * time.Millisecond
	// max stats to hold
	maxQueuedStats = 100
	// amount of stats until we send some
	statThreshold = 10
	// the maximum amount of time we will wait until threshold is hit in ms
//	maxWaitTime = 2500
	// the threshold of bytes until we send (unless maxWaitTime passes)
	byteThreshold = 50 * 20 // say N lines, at Y bytes each
	// number of buffers to hold logs. Each buffer will hold one callback worth of log entries 
	// from the greasego/greaselib subsystem
	// Bear in mind greaselib has its own internal rotating buffers
	// this defaults to NUM_BANKS (4) which is defined in greaseLib/logger.h
	// and is overridden by the defaults.NUMBER_BANKS_WEBLOG (20) (see main.go)
	// but it can be changed using the GreaseLibTargetOpts field .num_banks
	// when the target is created
	// this should be at least 1 less than what NUM_BANKs is set to
	// so that the internal thread of the logger does not get jammed up
	// (if it does it will also start to drop logs, since it 
	// will not have any buffers left)
	defaultMaxBuffers = 10   // max amount of outboundBuffers
	// defaultNumBuffers = 3    // starting amount of outboundBuffer
	defaultSendTimeThreshold uint32 = 2500 // ms
	defaultSendSizeThreshold uint32 = 4096 // bytes
	// defaultBufferSize = 8192 // 8k - outboundBufferSize
	cmdShutdown = 1
	cmdSendLogs = 2
	cmdSendStats = 3
	// commands used by senders
	sndrSendNow = 1
	sndrShutdown = 0xFF
	// SysStatsCountThresholdDefault default is 10
	SysStatsCountThresholdDefault = 10
	// SysStatsTimeThresholdDefault default is every minute
	SysStatsTimeThresholdDefault = 60 * 1000 
	// SyStatsMaxSysStatsMaxBufferDefault default is 100
	SysStatsMaxBufferDefault = 100
)

// used for internal control
type ctrlToken struct {
	code int
	// for future aux use
}

type ClientError struct {
	StatusCode int
	Status string
}

func (err *ClientError) Error() string {
	return fmt.Sprintf("RMI Client Error: %d - %s",err.StatusCode,err.Status)
}

func newClientError(resp *http.Response) (ret *ClientError) {
	ret = new(ClientError)
	ret.StatusCode = resp.StatusCode
	ret.Status = resp.Status
	return
}

// WigWag Remote Management Interface (Symphony)
// client APIs

type ClientConfig struct {
	// The RootCA option should be a PEM encoded root ca chain
	// Use this if the server's TLS certificate is not signed
	// by a certificate authority in the default list. If the
	// server is signed by a certificate authority in the default
	// list it can be omitted.
	RootCA []byte	// will be converted to byte array
	RootCAString string `yaml:"root_ca"`

	// The ServerName is also only required if the root ca chain
	// is not in the default list. This option should be omitted
	// if RootCA is not specified. It should match the common name
	// of the server's certificate.
	ServerName string `yaml:"server_name"`
	// This option can be used in place of the RootCA and ServerName
	// for servers that are not signed by a well known certificate
	// authority. It will skip the authentication for the server. It
	// is not recommended outside of a test environment.
	NoValidate bool `yaml:"no_validate"`
	// This option turns off encryption entirely
	// it is only for testing
	NoTLS bool `yaml:"no_tls"`
	// This is the PEM encoded SSL client certificate. This is required
	// for all https based client connections. It provides the relay identity
	// to the server
	ClientCertificate []byte
	ClientCertificateString string `yaml:"client_cert"`
	// This is the PEM encoded SSL client private key. This is required
	// for all https based client connections.
	ClientKey []byte
	ClientKeyString string `yaml:"client_key"`
	// This is the hostname or IP address of the Symphony server
	Host string `yaml:"host"`
	// This is the port of the Symphony server
	Port int `yaml:"port"`
	// UrlLogEndpoint will override the settings of host and port if set
	// for the remote logging endpoints
	// the default is https://[Host]/relay-logs/logs
	UrlLogsEndpoint string `yaml:"url_logs"`
	// UrlStatsEndpoint will override the settings of host and port if set
	// for the remote system stats history
	// the default is https://[Host]/relay-stats/stats_obj
	UrlStatsEndpoint string `yaml:"url_stats"`

	// If this flag is set, client library logging will be printed
	//EnableLogging bool
	// number of buffers to hold. Remember, grease lib also holds its own buffers, so this sould be minimal
	// (optional)
	//NumBuffers uint32 `yaml:"num_buffers"`
	// MaxBuffers is the max number of the said buffers
	MaxBuffers uint32 `yaml:"max_buffers"`
	// If true, then system stats are not sent to Symphony
	DisableSysStats bool `yaml:"disable_sys_stats"`
	// BufferSize is the size of each of these buffers in bytes
	//BufferSize uint32 `yaml:"buffer_size"`
	// SendSizeThreshold is the amount of bytes being held before the 
	// worker will start sending
	SendSizeThreshold uint32 `yaml:"send_size_threshold"`
	// SendTimeThreshold is the amount of time in milliseconds before the worker 
	// will start sending logs
	SendTimeThreshold uint32 `yaml:"send_time_threshold"`
	// SysStatsCountThreshold is the threshold where we will send stats
	SysStatsCountThreshold uint32 `yaml:"sys_stats_count_threshold"`
	// SysStatsTimeThreshold is the max amount of time which will go by before
	// we will send stats to the API. This is in ms
	SysStatsTimeThreshold uint32 `yaml:"sys_stats_time_threshold"`
	// SysStatsMaxBuffer is the max amount of sys_stats we will hold in memory before dropping
	SysStatsMaxBuffer uint32 `yaml:"sys_stats_max_buffer"`
}

// for general guidelines see: https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779
// and https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/

// Client is primary connection object for connecting to the RMI API server
// This is a system wide singleton
type Client struct {
	config    *ClientConfig
	client    *http.Client
	tlsconfig tls.Config
	// empty logBuffer's to place stuff
	availableFifo     *logBufferFIFO
	// logBuffer's to send
	willSendFifo      *logBufferFIFO
	// logBuffers which are bing sent (but might fail and go back into willSendFifo)
	sendingFifo       *logBufferFIFO
	ctrlChan chan *ctrlToken

	host      string
	port      int
	url       string
	postLogsUrl string
	postStatsUrl string
	// numBuffers uint32
	maxBuffers uint32
	sendTimeThreshold uint32
	sendSizeThreshold uint32
	// bufferSize uint32
	notValidConfig bool
	// amount of bytes held in all buffers in willSendFifo
	sendableBytes uint32
	// num bytes that were just sent
	sentBytes uint32

	connected        bool
	logWorkerRunning bool

	// if true, then the sendSizeThreshold is ignored
	// and the sender backs off "backoff" time 
	backingOff bool
	backoff          time.Duration
	// TODO callback for when client has failed
	locker sync.Mutex
	waitStart *sync.Cond

	buflocker sync.Mutex
	readOngoing bool
	// for sending + queing stats
	statsSender *statSender
	// our subscription for sysstats events
	sysStatsEventSubscription events.EventSubscription
	sysStatsMaxQueued uint32
	sysStatsTimeThreshold uint32
	sysStatsCountThreshold uint32
}

// NewClient creates a new client object to the Symphony API server
func NewClient(config *ClientConfig) (ret *Client, err error) {
	var clientTLSConfig *tls.Config
	rootCAs := x509.NewCertPool()

	// if strings were used to populate config, convert them...
	if len(config.RootCA) == 0 {
		if len(config.RootCAString) > 0 {
			config.RootCA = []byte(config.RootCAString)
		}
	}
	if len(config.ClientCertificate) == 0 {
		if len(config.ClientCertificateString) > 0 {
			config.ClientCertificate = []byte(config.ClientCertificateString)
		}
	}
	if len(config.ClientKey) == 0 {
		if len(config.ClientKeyString) > 0 {
			config.ClientKey = []byte(config.ClientKeyString)
		}
	}

	if config.RootCA != nil {
		if !rootCAs.AppendCertsFromPEM(config.RootCA) {
			return nil, errors.New("Could not append root CA to chain")
		}
	} else {
		rootCAs = nil
	}

	if config.ClientCertificate != nil || config.ClientKey != nil {
		clientCertificate, err := tls.X509KeyPair(config.ClientCertificate, config.ClientKey)

		if err != nil {
			return nil, err
		}

		if config.ServerName == "" {
			config.ServerName = config.Host
		}

		clientTLSConfig = &tls.Config{
			Certificates:       []tls.Certificate{clientCertificate},
			RootCAs:            rootCAs,
			InsecureSkipVerify: config.NoValidate,
			ServerName:         config.ServerName,
			GetClientCertificate: func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				return &clientCertificate, nil
			},
		}
	}

	ret = new(Client)
	ret.ctrlChan = make(chan *ctrlToken)
	ret.host = config.Host
	ret.port = config.Port

	if !config.NoTLS {
		ret.client = &http.Client{
			Timeout: 35 * time.Second,
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).Dial,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				TLSClientConfig:       clientTLSConfig,
			},
		}
		if len(config.Host) > 0 {
			ret.url = "https://" + config.Host
		} else {
			ret.notValidConfig = true
			err = errors.New("No Host field specified")
			return
		}
	} else {
		ret.client = &http.Client{
			Timeout: 35 * time.Second,
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).Dial,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		}
		if len(config.Host) > 0 {
			ret.url = "http://" + config.Host
		} else {
			ret.notValidConfig = true
			err = errors.New("No Host field specified")
			return
		}
	}

	if config.MaxBuffers > 0 {
		ret.maxBuffers = config.MaxBuffers
	} else {
		ret.maxBuffers = defaultMaxBuffers
	}

	if config.SendSizeThreshold > 0 {
		ret.sendSizeThreshold = config.SendSizeThreshold
	} else {
		ret.sendSizeThreshold = defaultSendSizeThreshold
	}
	if config.SendTimeThreshold > 0 {
		ret.sendTimeThreshold = config.SendTimeThreshold
	} else {
		ret.sendTimeThreshold = defaultSendTimeThreshold
	}
	// log buffer FIFOs. 
	ret.availableFifo = New_logBufferFIFO(ret.maxBuffers)
	ret.willSendFifo = New_logBufferFIFO(ret.maxBuffers-1)
	ret.sendingFifo = New_logBufferFIFO(ret.maxBuffers-1)	

	// pre-create all the structs. 
	var n uint32
	for n=0; n < ret.maxBuffers; n++ {
		buf := new(logBuffer)
		ret.availableFifo.Push(buf)		
	}

	if config.Port == 0 {
		config.Port = 443
	}
	if config.Port != 443 {
		ret.url = fmt.Sprintf("%s:%d", ret.url, config.Port)
	}

	ret.postLogsUrl = ret.url + "/relay-logs/logs"
	ret.postStatsUrl = ret.url + "/relay-stats/stats_obj"
	if len(config.UrlLogsEndpoint) > 0 {
		_, err2 := url.ParseRequestURI(config.UrlLogsEndpoint)
		if err2 != nil {
			err = fmt.Errorf("bad parse 'url_logs' config: %s",err2.Error())
			return
		}
		ret.postLogsUrl = config.UrlLogsEndpoint
		log.MaestroDebugf("RMI: using logs endpoint of %s",ret.postLogsUrl)		
	}
	if len(config.UrlStatsEndpoint) > 0 {
		_, err2 := url.ParseRequestURI(config.UrlStatsEndpoint)
		if err2 != nil {
			err = fmt.Errorf("bad parse 'url_stats' config: %s",err2.Error())
			return
		}
		ret.postStatsUrl = config.UrlStatsEndpoint
		log.MaestroDebugf("RMI: using stats endpoint of %s",ret.postStatsUrl)
	}

	ret.waitStart = sync.NewCond(&ret.locker)
	ret.sysStatsCountThreshold = config.SysStatsCountThreshold
	if ret.sysStatsCountThreshold == 0 {
		ret.sysStatsCountThreshold = SysStatsCountThresholdDefault
	}
	ret.sysStatsTimeThreshold = config.SysStatsTimeThreshold
	if ret.sysStatsTimeThreshold == 0 {
		ret.sysStatsTimeThreshold = SysStatsTimeThresholdDefault
	}
	ret.sysStatsMaxQueued = config.SysStatsMaxBuffer
	if ret.sysStatsMaxQueued == 0 {
		ret.sysStatsMaxQueued = SysStatsMaxBufferDefault
	}

	ret.statsSender = newStatSender(ret)
	if !config.DisableSysStats {
		ret.sysStatsEventSubscription, err = sysstats.SubscribeToEvents()
	}
	return
}


func (client *Client) SubmitLogs(data *greasego.TargetCallbackData, godata []byte) (err error) {
	// client.buflocker.Lock()
	// if client.activeBuffer == nil {
	// 	err = errors.New("no-buffers")
	// 	client.buflocker.Unlock()
	// 	return
	// }
	// lenactive := len(client.activeBuffer.data)
	// capactive := cap(client.activeBuffer.data)
	// if (capactive-lenactive) < len(godata) {
	// 	// we don't have enough room in the active buffer
	// 	// so let's close out this buffer
	// 	client.activeBuffer.closeout()
	// 	client.willSendFifo.Push(client.activeBuffer)
	// 	client.activeBuffer = nil
	// 	client.buflocker.Unlock()
	// 	buf := client.availableFifo.PopOrWait()
	// 	lenactive = buf.clear()
	// 	if buf == nil {
	// 		err = errors.New("no-buffers-2")
	// 		return
	// 	} else {
	// 		client.activeBuffer = buf
	// 	}
	// }

	// copy(client.activeBuffer.data[lenactive:],godata)
	// return
	buflen := len(godata)
	buf := client.availableFifo.PopOrWait()
	if buf == nil {
		err = errors.New("shutdown")
		return
	}
	buf.godata = godata
	buf.data = data

	// move to FIFO to send out

	dropped, droppedbuf := client.willSendFifo.Push(buf)
	if dropped {
		debugging.DEBUG_OUT("RMI: FIFO full - Dropped some old log entries!!!!!!\n\n")
		log.MaestroErrorf("RMI: FIFO full - Dropped some old log entries!")
		if droppedbuf != nil {
			client.locker.Lock()
			client.sendableBytes = client.sendableBytes - uint32(len(droppedbuf.godata))
			client.locker.Unlock()
			greasego.RetireCallbackData(droppedbuf.data)
			droppedbuf.clear()
			client.availableFifo.Push(droppedbuf)
		}
	}
	client.locker.Lock()
	client.sendableBytes = client.sendableBytes + uint32(buflen)
	if client.sendableBytes > client.sendSizeThreshold {
		client.locker.Unlock()
		cmd := new(ctrlToken)
		cmd.code = cmdSendLogs	
		client.ctrlChan <- cmd
	} else {
		client.locker.Unlock()
	}

	return
}

var _count int

// This is a target callback to assign to the grease subsystem.
func TargetCB(err *greasego.GreaseError, data *greasego.TargetCallbackData){

	_count++
	debugging.DEBUG_OUT("}}}}}}}}}}}} TargetCB_count called %d times\n",_count);
	if(err != nil) {
		fmt.Printf("ERROR in toCloud target CB %s\n", err.Str)
	} else {
		buf := data.GetBufferAsSlice()
		debugging.DEBUG_OUT("CALLBACK %+v ---->%s<----\n\n",data,string(buf));
		client, err2 := GetMainClient(nil)
		if err2 == nil {
			client.locker.Lock()
			if !client.logWorkerRunning {
				greasego.RetireCallbackData(data)
				client.locker.Unlock()
				log.MaestroErrorf("RMI client not ready! log worker not running.")
				return
			}
			client.locker.Unlock()
			client.SubmitLogs(data,buf)
		} else {
			log.MaestroErrorf("RMI client not ready! logs will be dropped. details: %s",err2.Error())
			greasego.RetireCallbackData(data)
		}
	}
}

var mainClient *Client

func GetMainClient(config *ClientConfig) (client *Client, err error) {
	if mainClient == nil {
		if config != nil {
			mainClient, err = NewClient(config)
		} else {
			err = errors.New("RMI client not configured")
		}
	}
	client = mainClient
	return
}

func (client *Client) Valid() bool {
	return !client.notValidConfig
}

// StartWorkers starts any IO workers, and will not return until all workers 
// are confirmed as started
func (client *Client) StartWorkers() (err error) {
	if client.notValidConfig {
		err = errors.New("not valid config")
	} else {
		client.locker.Lock()
		go client.clientLogWorker()
		client.waitStart.Wait()
		client.locker.Unlock()
	}
	client.statsSender.start()
	return
}




// func (client *Client) IsLogWorkerRunning() (ret bool) {
// 	client.locker.Lock()
// 	ret = client.logWorkerRunning
// 	client.locker.Unlock()
// 	return
// }

// func (client *Client) startTicker() {
// 	client.ticker = time.NewTicker(client.interval)
// 	go func() {
// 	// only dump ticker info when in debug build:
//         DEBUG(for t := range client.ticker.C { )
//             debugging.DEBUG_OUT("Tick at", t)
//         DEBUG(    DumpMemStats() )
// 	        DEBUG(	client.fifo.WakeupAll() )
//         DEBUG(} )
//     }()
// }

// Read implements the io.Reader interface for a FIFO storing
// the logBuffer objects
func (client *Client) Read(p []byte) (copied int, err error) {
	remain := len(p)
	buflen := 0
	if !client.readOngoing {
		copy(p[copied:],[]byte("["))
		copied++
	}
	client.readOngoing = true
	buf := client.willSendFifo.Peek()
	for buf != nil {
		debugging.DEBUG_OUT("RMI Read() (logs) to of loop\n")
		buflen = len(buf.godata) 
		if copied >= len(p) {
			client.sentBytes += uint32(copied-1)
			return
		}
		if buflen+1 > remain { // need room for ending ']'
			// no more room in this Read's p[]
			client.sentBytes += uint32(copied-1)
			return
		}
		buf = client.willSendFifo.Pop()		
		// write data to p 
		copy(p[copied:],buf.godata)
		copied += buflen
		remain -= buflen
		// release the memory it referenced back to the greaseLib
		buf.tries++
		client.sendingFifo.Push(buf)
		buf = client.willSendFifo.Peek()
	}
	debugging.DEBUG_OUT("RMI Read() (logs) EOF\n")
	if copied > 1 {
		// we ran out of buffers, so say EOF
		client.sentBytes += uint32(copied-1)
		copy(p[copied-2:],[]byte("]")) // replace last ',' with ']'
	} else {
		// it was en empty read. Not what we want (this should not happen)
		// but let's make it valid JSON
		if remain > 1 {
			copy(p[1:],[]byte("]")) // replace last ',' with ']'
			err = errors.New("no data")
			client.readOngoing = false
			return
		}
	}
//	debugging.DEBUG_OUT("RMI - READ EOF >>%s<<\n",string(p))
	client.readOngoing = false
	err = io.EOF
	return
}



// the client worker goroutine
// does the sending of data to the server
func (client *Client) clientLogWorker() {
	// closeit := func(r *http.Response, buf *logBuffer) {
	// 	r.Body.Close()
	// 	greasego.RetireCallbackData(buf.data)
	// 	debugging.DEBUG_OUT("  OKOKOKOKOKOKOKOKOK -----> retired callback data\n\n")
	// }
	var timeout time.Duration
	client.locker.Lock()
	if client.logWorkerRunning {
		client.waitStart.Broadcast()
		client.locker.Unlock()
		return
	}
	client.logWorkerRunning = true
	client.waitStart.Broadcast()
	timeout = time.Duration(client.sendTimeThreshold) * time.Millisecond
	client.locker.Unlock()

	var handleErr = func(clienterr error) {
		debugging.DEBUG_OUT("RMI clientLogWorker.handleErr(%+v)\n",clienterr)
		if clienterr != nil {
			errs := clienterr.Error()
			if strings.HasSuffix(errs,"no data") {
				debugging.DEBUG_OUT("handleErr(no data) - NOOP\n")
				// do nothing. No data was sent.
				return
			}
			// transfer failed, so...
			// move all the buffers back into the willSendFifo
			buf := client.sendingFifo.Pop()
			for buf != nil {
				if buf.tries > maxLogTries {
					debugging.DEBUG_OUT("RMI ERROR: dropping a log buffer (max tries)\n")
					greasego.RetireCallbackData(buf.data)
					buf.clear()
					client.availableFifo.Push(buf)
				} else {
					debugging.DEBUG_OUT("RMI clientLogWorker.handleErr() returning buf for another try.\n")
					client.willSendFifo.Push(buf)
				}
				buf = client.sendingFifo.Pop()
			}
			// do a backoff
			debugging.DEBUG_OUT("RMI @lock 1\n")
			client.locker.Lock()
			debugging.DEBUG_OUT("RMI @pastlock 1\n")
			client.backingOff = true
			client.locker.Unlock()
		} else {
			// success. reset backoff
			client.locker.Lock()
			client.backoff = 0
			client.backingOff = false
			client.sendableBytes = client.sendableBytes - client.sentBytes
			client.sentBytes = 0
			client.locker.Unlock()
			// buffers were sent. So let's retire all those greasego buffer's back
			// to the library
			buf := client.sendingFifo.Pop()
			for buf != nil {
				greasego.RetireCallbackData(buf.data)
				buf.clear()
				client.availableFifo.Push(buf)
				buf = client.sendingFifo.Pop()
			}
		}
	}

	var start time.Time
	var elapsed time.Duration

	// var statsChannel chan *events.MaestroEvent

	// if client.sysStatsEventSubscription == nil {
	// 	statsChannel = events.GetDummyEventChannel()
	// } else {
	// 	var ok bool
	// 	ok, statsChannel = client.sysStatsEventSubscription.GetChannel()
	// 	if !ok {
	// 		log.MaestroErrorf("RMI - could not get the events channel for sysstats. Will not report.")
	// 		statsChannel = events.GetDummyEventChannel()
	// 	}
	// }

	commandLoop:
	for {		
		debugging.DEBUG_OUT("RMI clientLogWorker top of for{} (%d)\n",int(timeout))
		start = time.Now()
		select {
		case <-time.After(timeout):
			debugging.DEBUG_OUT("RMI triggered - timeout after %d\n",timeout)
			err := client.postLogs()
			handleErr(err)

		// case stat := <-statsChannel:
		// 	wrapper, ok := convertStatEventToWrapper(stat)
		// 	if ok {
		// 		dropped, _ := client.statsSender.submitStat(wrapper)
		// 		if dropped {
		// 			log.MaestroWarnf("RMI - dropping systems stats. Queue is full.")
		// 		}
		// 	}
		// 	// TODO - hold events, until read to send
		// 	// then JSON encode.
		case cmd := <-client.ctrlChan:
			// got a command
			switch cmd.code {
			case cmdSendLogs:
				client.locker.Lock()
				if client.backingOff {
					client.locker.Unlock()
					elapsed = time.Since(start)
					timeout = timeout - elapsed
					if timeout > 10000 {
						debugging.DEBUG_OUT("RMI ignoring cmdSendLogs\n")
						continue
					}	
					// here to see how well this is working		
					debugging.DEBUG_OUT("RMI triggered - (ALMOST IGNORED) cmdSendLogs\n")
				} else {
					client.locker.Unlock()
				}
				debugging.DEBUG_OUT("RMI triggered - cmdSendLogs\n")
				err := client.postLogs()
				handleErr(err)
			case cmdShutdown:
				break commandLoop
			}
		}
		// set up the next timeout.
		client.locker.Lock()
		timeout = time.Duration(client.sendTimeThreshold) * time.Millisecond
		if client.backingOff {
			if client.backoff < 2 {
				client.backoff = timeout
			} else {
				client.backoff = client.backoff * 2
			}
			if client.backoff > maxBackoff {
				client.backoff = timeout
			}
			timeout = client.backoff
			debugging.DEBUG_OUT("RMI send logs is backing off %d ms\n",client.backoff)
		}
		client.locker.Unlock()
	}

	// for true {
	// 	next = client.willSendFifo.PopOrWait()
	// 	if next == nil {
	// 		if client.fifo.IsShutdown() {
	// 			debugging.DEBUG_OUT("clientWorker @shutdown - via FIFO")
	// 			break
	// 		} else {
	// 			// was woken, but no new data
	// 			continue
	// 		}
	// 	}
	// 	// send data to server0
	// 	//		req, err := http.NewRequest("POST", client.url, bytes.NewReader(next.data.GetBufferAsSlice()))
	// 	// req, err := http.NewRequest("POST", client.url, bytes.NewReader(next.godata))
	// 	//	    req.Header.Set("X-Custom-Header", "myvalue")
	// 	// req.Header.Set("Content-Type", "application/json")
	// 	// req.Header.Add("X-Symphony-ClientId", client.clientId)

	// 	// resp, err := client.httpClient.Do(req)
	// 	// if err != nil {
	// 	// 	debugging.DEBUG_OUT("XXXXXXXXXXXXXXXXXXXXXXX error on sending request %+v\n", err)
	// 	// 	greasego.RetireCallbackData(next.data)
	// 	// } else {
	// 	// 	fmt.Println("response Status:", resp.Status)
	// 	// 	fmt.Println("response Headers:", resp.Header)
	// 	// 	body, _ := ioutil.ReadAll(resp.Body)
	// 	// 	fmt.Println("response Body:", string(body))

	// 	// 	debugging.DEBUG_OUT("CALLING closeit()\n")
	// 	// 	closeit(resp, next)
	// 	// }

	// 	err := client.postLogs(next.data.GetBufferAsSlice())
	// 	if err != nil {
	// 		debugging.DEBUG_OUT("RMI API error - could not push logs: %s\n", err.Error())
	// 		if next.tries < maxLogTries {
	// 			next.tries++
	// 			drop, dropped := client.fifo.Push(next)
	// 			if drop {
	// 				debugging.DEBUG_OUT("Causing us to drop a log entry!!\n")
	// 				log.MaestroErrorf("RMI - log entries dropped.")
	// 				if dropped != nil {
	// 					greasego.RetireCallbackData(dropped.data)
	// 				}
	// 			}
	// 		} else {
	// 			log.MaestroErrorf("RMI - failed to send logs. past max retry. dropping some logs.")
	// 			greasego.RetireCallbackData(next.data)
	// 		}
	// 	} else {
	// 		debugging.DEBUG_OUT("RMI --> Pushed log entry.\n")
	// 		greasego.RetireCallbackData(next.data)
	// 	}
	// }

	client.locker.Lock()
	client.logWorkerRunning = false
	client.locker.Unlock()
}


// TODO cancel http requests
// see: https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/

// PostLogs sends logs to the API
func (client *Client) postLogs() (err error) {
	var req *http.Request
	var resp *http.Response
	debugging.DEBUG_OUT("RMI POST %s >>>\n",client.postLogsUrl)
	// Client implements io.Reader's Read(), so we do this
	client.sentBytes = 0
	req, err = http.NewRequest("POST", client.postLogsUrl, client)
	//	req.Cancel = c

	if err == nil {
		resp, err = client.client.Do(req)
		// close response body in case of success and errors, not required if no response
		if resp != nil {
                        defer resp.Body.Close()
                }
		debugging.DEBUG_OUT("RMI --> response +%v",resp)
		if err == nil && resp != nil && resp.StatusCode != 200 {
			debugging.DEBUG_OUT("RMI bad response - creating error object\n")
			bodystring, _ := utils.StringifyReaderWithLimit(resp.Body,300)
			log.MaestroErrorf("RMI: Error on POST request for logs: Response was %d (Body <%s>)",resp.StatusCode,bodystring)
			err = newClientError(resp)
		}
	} else {
		log.MaestroErrorf("Error on POST request: %s\n",err.Error())
		debugging.DEBUG_OUT("RMI ERROR: %s\n",err.Error())
	}
	return
}



// StopWorkers stops any worker routines of the client. The client cane effectively be lost
// garabage collected afterwards
func (client *Client) StopWorkers() {
	client.availableFifo.Shutdown()
	client.sendingFifo.Shutdown()
	client.willSendFifo.Shutdown()
}
