/*
 Copyright (c) 2020 ARM Limited and affiliates.
 Copyright (c) 2016 Mark Berner
 SPDX-License-Identifier: MIT
 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to
 deal in the Software without restriction, including without limitation the
 rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
 sell copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:
 The above copyright notice and this permission notice shall be included in all
 copies or substantial portions of the Software.
 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 SOFTWARE.
*/
package main

import (
	"fmt"
	"os"

	wifi "github.com/armPelionEdge/wpa-connect"
	"time"
)

func main() {

	args := os.Args[1:]
	if len(args) < 2 {
		fmt.Println("Insufficient arguments")
		return
	}
	ssid := args[0]
	password := args[1]
	wifi.SetDebugMode()
	if conn, err := wifi.ConnectManager.Connect(ssid, password, time.Second * 60); err == nil {
		fmt.Println("Connected", conn.NetInterface, conn.SSID, conn.IP4.String(), conn.IP6.String())
	} else {
		fmt.Println(err)
	}
}
