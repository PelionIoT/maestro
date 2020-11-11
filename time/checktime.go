package time

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
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"

	"github.com/PelionIoT/maestro/debugging"
	"github.com/PelionIoT/maestro/log"
)

// Simple sub package to handle getting the time from a server, using a
// a very basic /time GET.

const (
	defaultCheckTimeInterval = time.Duration(time.Hour * 24)
	initialBackoff           = time.Duration(time.Second * 10)
	maxBackoff               = time.Duration(5) * time.Minute
	// a sanity check value. A time value below this is crazy.
	// Jan 1, 2018:
	recentTime = int64(1514786400000)
	// SetTimeOk means the time was set Correctly
	SetTimeOk = 1
	// TimedOut means there was no response from server
	TimedOut = 2
	// BadResponse means the response from server was not formatted correctly
	BadResponse = 3
	// InsaneResponse means the server provided a time value which is crazy
	InsaneResponse = 4
	// SycallFailed means the syscall failed to work to set the time
	SycallFailed = 5
)

// ClientConfig for getting time.
// /time is tricky, b/c if the time is not sane on the system
// SSL validation can break. So, even if we are using SSL, we will get
// time value with validation disabled on SSL.
// We also do some sanity checks to make sure the time value makes sense.
type ClientConfig struct {
	// // The RootCA option should be a PEM encoded root ca chain
	// // Use this if the server's TLS certificate is not signed
	// // by a certificate authority in the default list. If the
	// // server is signed by a certificate authority in the default
	// // list it can be omitted.
	// RootCA []byte	// will be converted to byte array
	// RootCAString string `yaml:"root_ca"`

	// The ServerName is also only required if the root ca chain
	// is not in the default list. This option should be omitted
	// if RootCA is not specified. It should match the common name
	// of the server's certificate.
	// ServerName string `yaml:"server_name"`
	// This option can be used in place of the RootCA and ServerName
	// for servers that are not signed by a well known certificate
	// authority. It will skip the authentication for the server. It
	// is not recommended outside of a test environment.
	NoValidate bool `yaml:"no_validate"`
	// This option turns off encryption entirely
	// it is only for testing
	NoTLS bool `yaml:"no_tls"`
	// Pretend true is run, but don't actually set the time
	Pretend bool `yaml:"pretend"`
	// This is the PEM encoded SSL client certificate. This is required
	// for all https based client connections. It provides the relay identity
	// to the server
	// ClientCertificate []byte
	// ClientCertificateString string `yaml:"client_cert"`
	// // This is the PEM encoded SSL client private key. This is required
	// // for all https based client connections.
	// ClientKey []byte
	// ClientKeyString string `yaml:"client_key"`
	// // This is the hostname or IP address of the relaymq server
	Host string `yaml:"host"`
	// This is the port of the relaymq server
	Port int `yaml:"port"`
	// CheckTimeInterval in seconds
	CheckTimeInterval int `yaml:"check_time_interval"`

	// If this flag is set, client library logging will be printed
	//EnableLogging bool
	// number of buffers to hold. Remember, grease lib also holds its own buffers, so this sould be minimal
	// (optional)
	//NumBuffers uint32 `yaml:"num_buffers"`
	// MaxBuffers is the max number of the said buffers
	// MaxBuffers uint32 `yaml:"max_buffers"`
	// // BufferSize is the size of each of these buffers in bytes
	// //BufferSize uint32 `yaml:"buffer_size"`
	// // SendSizeThreshold is the amount of bytes being held before the
	// // worker will start sending
	// SendSizeThreshold uint32 `yaml:"send_size_threshold"`
	// // SendTimeThreshold is the amount of time in milliseconds before the worker
	// // will start sending logs
	// SendTimeThreshold uint32 `yaml:"send_time_threshold"`
}

type TimeClient struct {
	client        *http.Client
	tlsconfig     tls.Config
	host          string
	port          int
	url           string
	statusChannel chan int
	// used to shutdown the client only
	stopChannel       chan struct{}
	running           bool
	locker            sync.Mutex
	checkTimeInterval time.Duration
	pretend           bool

	// if true, the sender backs off "backoff" time
	backingOff bool
	backoff    time.Duration
}

type ClientError struct {
	StatusCode int
	Status     string
}

func (err *ClientError) Error() string {
	return fmt.Sprintf("TIME Client Error: %d - %s", err.StatusCode, err.Status)
}

func newClientError(resp *http.Response) (ret *ClientError) {
	ret = new(ClientError)
	ret.StatusCode = resp.StatusCode
	ret.Status = resp.Status
	return
}

// StatusChannel returns the status channel which can be used to know if time is set
// if nothing reads the channel, the time will be set anyway, and a simple log message is
// printed out.
func (client *TimeClient) StatusChannel() (ok bool, status chan int) {
	client.locker.Lock()
	ok = client.running
	status = client.statusChannel
	client.locker.Unlock()
	return
}

// Run starts the client
func (client *TimeClient) Run() {
	go client.worker()
}

// Stop the current client's worker
func (client *TimeClient) Stop() {
	client.locker.Lock()
	if client.running {
		close(client.stopChannel)
	}
	client.locker.Unlock()
}

// NewClient creates a new TimeClient and validates the config
func NewClient(config *ClientConfig) (ok bool, ret *TimeClient, err error) {
	ret = new(TimeClient)
	ret.statusChannel = make(chan int)
	err = ret.Reconfigure(config)
	if err == nil {
		ok = true
	}
	return
}

// Reconfigure allows you to reconfigure the client
func (client *TimeClient) Reconfigure(config *ClientConfig) (err error) {
	client.pretend = config.Pretend
	client.host = config.Host
	client.port = config.Port
	client.checkTimeInterval = time.Duration(config.CheckTimeInterval) * time.Second

	if client.checkTimeInterval < 5 {
		client.checkTimeInterval = defaultCheckTimeInterval
	}

	if !config.NoTLS {
		client.client = &http.Client{
			Timeout: 35 * time.Second,
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).Dial,
				MaxIdleConnsPerHost:   100,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				//				TLSClientConfig:       clientTLSConfig,
			},
		}
		if len(config.Host) > 0 {
			client.url = "https://" + config.Host + "/api/time"
		} else {
			// client.notValidConfig = true
			err = errors.New("No Host field specified")
		}
	} else {
		client.client = &http.Client{
			Timeout: 35 * time.Second,
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).Dial,
				MaxIdleConnsPerHost:   100,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		}
		if len(config.Host) > 0 {
			client.url = "http://" + config.Host
		} else {
			// client.notValidConfig = true
			err = errors.New("No Host field specified")
		}
	}
	return
}

// // this is actually from the golang source - but is really knew - 1.10 only:
// // TimeToTimespec converts t into a Timespec.
// // On some 32-bit systems the range of valid Timespec values are smaller
// // than that of time.Time values.  So if t is out of the valid range of
// // Timespec, it returns a zero Timespec and EINVAL.
// func TimeToTimespec(t time.Time) (Timespec, error) {
// 	sec := t.Unix()
// 	nsec := int64(t.Nanosecond())
// 	ts := setTimespec(sec, nsec)

// 	// Currently all targets have either int32 or int64 for Timespec.Sec.
// 	// If there were a new target with floating point type for it, we have
// 	// to consider the rounding error.
// 	if int64(ts.Sec) != sec {
// 		return Timespec{}, EINVAL
// 	}
// 	return ts, nil
// }

type timeResponse struct {
	Time int64 `json:"time"`
}

func (client *TimeClient) getTime() (err error, errcode int, ret *timeResponse) {
	// adds log to the fifo

	var req *http.Request
	var resp *http.Response
	debugging.DEBUG_OUT("TIME GET %s >>>\n", client.url)
	// Client implements io.Reader's Read(), so we do this
	req, err = http.NewRequest("GET", client.url, nil)
	//	req.Cancel = c

	if err == nil {
		resp, err = client.client.Do(req)
		if err != nil {
			if resp != nil {
				defer resp.Body.Close()
			}
		}
		debugging.DEBUG_OUT("TIME --> response +%v\n", resp)
		if err == nil {
			if resp != nil {
				if resp.StatusCode != 200 {
					debugging.DEBUG_OUT("TIME bad response - creating error object\n")
					err = newClientError(resp)
					errcode = BadResponse
					return
				} else {
					ret = new(timeResponse)
					dec := json.NewDecoder(resp.Body)
					if dec != nil {
						err = dec.Decode(ret)
						if err != nil {
							err = errors.New("Bad response")
							errcode = BadResponse
							ret = nil
						}
					} else {
						err = errors.New("Failed to create decoder")
						errcode = BadResponse
					}
				}
			} else {
				err = errors.New("No response")
				errcode = TimedOut
			}
		}
	} else {
		log.MaestroErrorf("Error on GET request: %s\n", err.Error())
		debugging.DEBUG_OUT("TIME ERROR: %s\n", err.Error())
		err = errors.New("Failed to create request")
		errcode = BadResponse
	}
	return
}

func (client *TimeClient) sendToStatusChannel(val int) {
	select {
	case client.statusChannel <- val:
	default:
		log.MaestroWarn("time status channel is blocking.")
	}
}

// periodically asks for the time
func (client *TimeClient) worker() {
	client.locker.Lock()
	if client.running {
		client.locker.Unlock()
		return
	}
	client.running = true
	client.locker.Unlock()
	timeout := time.Duration(1)

	for {
		select {
		case <-client.stopChannel:
			break
		case <-time.After(timeout):
			// run the client
			err, errcode, timeresp := client.getTime()
			timeout = client.checkTimeInterval
			if err != nil {
				if client.backingOff {
					client.backoff = client.backoff * time.Duration(2)
				} else {
					client.backoff = initialBackoff
				}
				if client.backoff > maxBackoff {
					client.backoff = maxBackoff
				}
				client.backingOff = true
				timeout = client.backoff
				// analyze errors and send to status channel
				client.sendToStatusChannel(errcode)
			} else {
				client.backingOff = false
				client.backoff = 0
				client.locker.Lock()
				timeout = client.checkTimeInterval
				client.locker.Unlock()

				// set the time
				if timeresp.Time > recentTime {
					timespec := timeToTimeval(timeresp.Time)
					now := time.Now().UnixNano() / 1000000
					log.MaestroInfof("Time: time being adjusted. Skew is %d ms\n", now-timeresp.Time)
					if client.pretend {
						log.MaestroInfof("Time: time would be set to %d s %d us - but prentending only.\n", timespec.Sec, timespec.Usec)
					} else {
						errno := syscall.Settimeofday(&timespec)
						if errno != nil {
							log.MaestroErrorf("Time: settimeofday failed: %s\n", errno.Error())
							client.sendToStatusChannel(SycallFailed)
						} else {
							log.MaestroSuccess("Time: time of day updated.")
							client.sendToStatusChannel(SetTimeOk)
						}
					}
				} else {
					log.MaestroErrorf("Time server reported INSANE time value (%ld) - ignoring.\n", timeresp.Time)
					client.sendToStatusChannel(InsaneResponse)
				}
				// send to status channel
			}
		}
	}

	client.locker.Lock()
	client.running = false
	client.locker.Unlock()
}
