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
	"net"
	"net/http"
	"sync"
	"syscall"

	"github.com/PelionIoT/httprouter"
	"github.com/PelionIoT/maestro/log"
)

type UnixHttpEndpoint struct {
	SocketPath string
	proper     bool
	server     *http.Server
	WG         *sync.WaitGroup
	//	locker sync.Mutex    // protect internal data, threaded data follows:
	//	conn *net.UnixConn
	running bool
	//	closing bool
}

func (sink *UnixHttpEndpoint) Init(path string) error { //, wg *sync.WaitGroup

	sink.SocketPath = path
	sink.proper = false
	sink.running = false
	sink.server = nil

	if len(path) > 0 {
		sink.proper = true
	}

	err := syscall.Unlink(sink.SocketPath)
	if err != nil {
		log.MaestroWarn("Unlink()", err)
	}

	return nil

}

func (sink *UnixHttpEndpoint) Start(router *httprouter.Router, wg *sync.WaitGroup) error {

	//		Handler: http.FileServer(http.Dir(root)),
	sink.server = &http.Server{Handler: router}
	sink.WG = wg
	unixListener, err := net.Listen("unix", sink.SocketPath)
	if err != nil {
		return err
	}
	sink.server.Serve(unixListener)
	sink.running = true

	//	sink.proper = false
	//	sink.SocketPath = path
	//	sink.locker = sync.Mutex{}  // init the mutex
	//	sink.WG = wg
	//	if(len(sink.SocketPath) < 1) {
	//		return errors.New("No socket path");
	//	}
	//	if sink.WG == nil {
	//		return errors.New("No WaitGroup assigned")
	//	}
	//
	//    addr, err := net.ResolveUnixAddr("unixgram", sink.SocketPath)
	//    if err != nil {
	//    	return err;
	//    }
	//
	//   	err = syscall.Unlink(sink.SocketPath)
	//   	if err != nil {
	//   		log.Warning("Unlink()",err)
	//   	}
	//
	//	conn, err := net.ListenUnixgram("unixgram", addr);
	//	if err != nil {
	//		return err;
	//	}
	//
	//	conn.SetReadBuffer(1024*8)
	//
	//	log.Info("Socket bound:",sink.SocketPath)
	//
	//	sink.conn = conn;
	//	sink.proper = true;

	return nil
}

//func main() {
//	if len(os.Args) < 2 {
//		fmt.Fprintln(os.Stderr, "usage:", os.Args[0], "/path.sock [wwwroot]")
//		return
//	}
//
//	fmt.Println("Unix HTTP server")
//
//	root := "."
//	if len(os.Args) > 2 {
//		root = os.Args[2]
//	}
//
//	os.Remove(os.Args[1])
//
//	server := http.Server{
//		Handler: http.FileServer(http.Dir(root)),
//	}
//
//	unixListener, err := net.Listen("unix", os.Args[1])
//	if err != nil {
//		panic(err)
//	}
//	server.Serve(unixListener)
//}
