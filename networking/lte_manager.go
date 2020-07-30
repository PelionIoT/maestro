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
	"github.com/armPelionEdge/maestro/log"
)

// Returns array of string names for all available modems
func AvailableModems() ([]string, error) {
    log.MaestroInfof("[STUB] would call\n>mmcli -L\nor equivalent\n")
    //Implementation TBD
    //This routine should either return all modems returned by mmcli -L, i.e.
    //all physical modems
    //or
    //return all interface names of type gsm which exist in the current config
    modems := make([]string, 0)
    return modems, nil
}

// Returns whether SIM is recognized in the modem and it is registered to the network
func IsModemRegistered(index string) bool {
    log.MaestroInfof("[STUB]  would call\n>mmcli -m %s\nor equivalent\n", index)
    //Implementation TBD

    return true
}

func AddLTEInterface(serial string, connection-name string, apn string) error {
    log.MaestroInfof("[STUB]  would call\n")
    log.MaestroInfof(">nmcli con add type gsm ifname %s con-name %s apn %s\n", serial, connection-name, apn)
    log.MaestroInfof("or equivalent\n")
    //Implementation TBD

    return nil
}

func BringUpModem(connection-name string) error {
    log.MaestroInfof("[STUB] would call\n>nmcli con up %s\nor equivalent\n", connection-name)
    //Implementation TBD

    return nil
}


func ConnectModem(index string, serial string, connection-name string, apn string) error {
    log.MaestroInfof("Connecting modem %s on srial interface %s with name % to APN %s\n",
        index, serial, connection-name, apn)

    //TODO - once Available modems is registered, verify index is valid
    //modemfound := false
    //for _, idx := range AvailableModems() {
    //   if idx == index {
    //      modemfound = true
    //      break
    //   }
    //}
    //if !modemfound {
    //   return fmt.Errorf("Modem %s not available", index")
    //}

    if !IsModemRegistered(index) {
        return fmt.Errorf("Modem %s SIM not present or registered")
    }

    err := AddLTEInterface(serial, connection-name, apn)

    if err != nil {
        log.MaestroErrorf("Unable to add LTE interface ( serial %s connection %s, apn %s): %s\n",
                          serial, connection-name, apn, err.Error())
        return fmt.Errorf("Iterface not added")
    }

    err = BringUpModem(connection-name)

    if err != nil {
        log.MaestroErrorf("Unable to bring up modem (connection %s): %s\n",
                          connection-name, err.Error())
        return fmt.Errorf("Modem not up")
    }

    return nil
}
