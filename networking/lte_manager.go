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
	"github.com/PelionIoT/maestro/log"
	"fmt"
	"strings"
	"strconv"
	"os/exec"
)

// Parses the index string from the modem path name returned by mmcli -L
// or portion of that path name
// e.g. all of the following return "0"
// indexFromPath("    /org/freedesktop/ModemManager1/Modem/0 [VENDOR] ACME [1234:5678]")
// indexFromPath("/org/freedesktop/ModemManager1/Modem/0 [VENDOR] ACME [1234:5678]")
// indexFromPath("/org/freedesktop/ModemManager1/Modem/0")
// indexFromPath("0")
func indexFromPath(path string) string{
	filepath := strings.Fields(path)[0]
	nodes := strings.Split(filepath, "/")
	return nodes[len(nodes) - 1]
}

// AvailableModems returns array of string names for all available modems
func AvailableModems() ([]string, error) {
	mmcli_out, err := exec.Command("mmcli", "-L").Output()
	if err != nil {
		return make([]string, 0), err
	}
	lines := strings.Split(string(mmcli_out), "\n")
	for n, line := range lines {
		if strings.HasPrefix(line, "Found") {
			num_modems, errconv := strconv.Atoi(strings.Fields(line)[1])
			if errconv != nil {
				log.MaestroErrorf("Unexpected output format from mmcli:\n%s\n", mmcli_out)
				return make([]string, 0), fmt.Errorf("Unable to parse mmcli output")
			}
			return lines[n+1:n+num_modems+1], nil
		}
	} 
	return make([]string, 0), nil

}

// IsModemRegistered returns whether SIM is recognized in the modem and it is registered to the network
func IsModemRegistered(index string) bool {
	err := exec.Command("mmcli", "-m", index).Run()
	return (err == nil)
}

// AddLTEInterface adds a network interface for the modem
func AddLTEInterface(ifName string, connectionName string, apn string) error {
	return exec.Command("nmcli", "con", "add", "type", "gsm", "ifname", ifName, "con-name", connectionName, "apn", apn).Run()
}

func BringUpModem(connectionName string) error {
	return exec.Command("nmcli", "con", "up", connectionName).Run()
}

func ConnectModem(index string, serial string, connectionName string, apn string) error {
	log.MaestroInfof("Connecting modem %s on serial interface %s with name %s to APN %s\n",
		index, serial, connectionName, apn)

	indexnum := indexFromPath(index)

	avail_modems, err:= AvailableModems()
	modemfound := false
	if err == nil {
		for _, idx := range avail_modems {
		   if indexFromPath(idx) == indexnum {
			  modemfound = true
			  break
		   }
		}
	}
	if !modemfound {
		return fmt.Errorf("Modem %s not available", index)
	}

	if !IsModemRegistered(indexnum) {
		return fmt.Errorf("Modem %s SIM not present or registered", index)
	}

	err = AddLTEInterface(serial, connectionName, apn)

	if err != nil {
		log.MaestroErrorf("Unable to add LTE interface ( serial %s connection %s, apn %s): %s\n",
			serial, connectionName, apn, err.Error())
		return fmt.Errorf("Iterface not added")
	}

	err = BringUpModem(connectionName)

	if err != nil {
		log.MaestroErrorf("Unable to bring up modem (connection %s): %s\n",
			connectionName, err.Error())
		return fmt.Errorf("Modem not up")
	}

	return nil
}
