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
	"fmt"
	"time"
             
	"github.com/armPelionEdge/maestro/log"
	wifi "github.com/armPelionEdge/wpa-connect"
)

func ConnectToWifi(ifname string, wifissid string, wifipassword string) (status string, ip string, errout error) {

	wifi.ConnectManager.NetInterface = ifname
	log.MaestroInfof("WifiManager interface:%v SSID:%v password:%v \n", ifname, wifissid, wifipassword)
	conn, err := wifi.ConnectManager.Connect( wifissid, wifipassword, time.Second * 60) 
	if err == nil {
		log.MaestroInfof(" WifiManager: Connected Interface:%v SSID:%v IPV4:%v IPV6:%v \n", conn.NetInterface, conn.SSID, conn.IP4.String(), conn.IP6.String())
		return "connected", conn.IP4.String(), err
	} else if err.Error() == "connection_failed, reason=-3" {
		return err.Error(), "", err
		
	} else if err.Error() == "connection_failed, reason=0" {
		return err.Error(), "", err
	} else if err.Error() == "address_not_allocated"{
		log.MaestroDebugf("WifiManager: Connected Still waiting to get an IP") 
		return "address_not_allocated", "", err
	} else {
		fmt.Printf(" WifiManager %v\n", err.Error())
		return err.Error(), "", err
	}
	
}

func DisconnectWifi(ifname string) (status string, errout error) {
	wifi.ConnectManager.NetInterface = ifname
	err := wifi.ConnectManager.Disconnect()
	if err == nil {
		log.MaestroInfof("WifiManager interface: %v Wifi disconnected from all networks \n", ifname)
		return "success", err
	} else {
		return "Wifi not disconnected", err
	}

}