package wwrelayA10

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
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/armPelionEdge/maestroSpecs"
	"github.com/armPelionEdge/maestroSpecs/templates"
	"github.com/mholt/archiver"
	I2C "golang.org/x/exp/io/i2c"
)

func _dummy() {
	var s string
	_ = fmt.Sprintf(s)
}

func check(log maestroSpecs.Logger, err error) {
	if err != nil {
		log.Errorf("Platform variable check failed %s\n", err.Error())
	}
}

type eeprom_anatomy struct {
	name     string
	pageaddr int
	memaddr  byte
	length   int
}

type eeData struct {
	name string
	data string
}

//Need to define two file names because one cert needs to be concatenated
type certs_anatomy struct {
	metavariable string
	fname1       string
	fname2       string
}

// ARCH will be added on automatically  by maestro
var sslCerts = []certs_anatomy{
	certs_anatomy{"CLIENT_KEY_PEM", "/mnt/.boot/.ssl/client.key.pem", ""},
	certs_anatomy{"CLIENT_CERT_PEM", "/mnt/.boot/.ssl/client.cert.pem", ""},
	certs_anatomy{"SERVER_KEY_PEM", "/mnt/.boot/.ssl/server.key.pem", ""},
	certs_anatomy{"SERVER_CERT_PEM", "/mnt/.boot/.ssl/server.cert.pem", ""},
	certs_anatomy{"CA_PEM", "/mnt/.boot/.ssl/ca.cert.pem", ""},
	certs_anatomy{"CA_INTERMEDIATE_PEM", "/mnt/.boot/.ssl/intermediate.cert.pem", ""},
	certs_anatomy{"CA_CHAIN_PEM", "/mnt/.boot/.ssl/ca.cert.pem", "/mnt/.boot/.ssl/intermediate.cert.pem"},
}

var metadata = []eeprom_anatomy{
	eeprom_anatomy{"BRAND", 0x50, 0, 2},
	eeprom_anatomy{"DEVICE", 0x50, 2, 2},
	eeprom_anatomy{"UUID", 0x50, 4, 6},
	eeprom_anatomy{"RELAYID", 0x50, 0, 10},
	eeprom_anatomy{"HARDWARE_VERSION", 0x50, 10, 5},
	eeprom_anatomy{"WW_PLATFORM", 0x50, 10, 5},
	eeprom_anatomy{"FIRMWARE_VERSION", 0x50, 15, 5},
	eeprom_anatomy{"RADIO_CONFIG", 0x50, 20, 2},
	eeprom_anatomy{"YEAR", 0x50, 22, 1},
	eeprom_anatomy{"MONTH", 0x50, 23, 1},
	eeprom_anatomy{"BATCH", 0x50, 24, 1},
	eeprom_anatomy{"ETHERNET_MAC", 0x50, 25, 6},
	eeprom_anatomy{"SIXLBR_MAC", 0x50, 31, 8},
	eeprom_anatomy{"RELAY_SECRET", 0x50, 39, 32},
	eeprom_anatomy{"PAIRING_CODE", 0x50, 71, 25},
	eeprom_anatomy{"LED_CONFIG", 0x50, 96, 2},
	eeprom_anatomy{"LED_COLOR_PROFILE", 0x50, 96, 2},
	eeprom_anatomy{"CLOUD_URL", 0x51, 0, 250},
	eeprom_anatomy{"CLOUD_DEVJS_URL", 0x52, 0, 250},
	eeprom_anatomy{"CLOUD_DDB_URL", 0x53, 0, 250},
	eeprom_anatomy{"CLOUD_DDB_URL_RES", 0x53, 0, 250},
	eeprom_anatomy{"RELAY_SERVICES_HOST", 0x51, 0, 250},
	eeprom_anatomy{"RELAY_SERVICES_HOST_RES", 0x51, 0, 250},
}

var mcc_config string = "/mnt/.boot/.ssl/mcc_config.tar.gz"
var regex = "[^a-zA-Z0-9.:/-]"
var cloudurl string

func get_eeprom(prop eeprom_anatomy, log maestroSpecs.Logger) (eeData, error) {

	bus, err := I2C.Open(&I2C.Devfs{Dev: "/dev/i2c-1"}, prop.pageaddr)
	if err != nil {
		return eeData{"", ""}, err
	}

	data := make([]byte, prop.length)

	err = bus.ReadReg(prop.memaddr, data)
	if err != nil {
		return eeData{"", ""}, err
	}

	dataStr := string(data)

	r, _ := regexp.Compile(regex)
	dataStr = r.ReplaceAllString(string(data), "")

	if prop.name == "WW_PLATFORM" {
		dataStr = "wwgateway_v" + dataStr
	}

	if prop.name == "ETHERNET_MAC" || prop.name == "SIXLBR_MAC" {
		dataStr = hex.EncodeToString(data)
	}

	if prop.name == "ETHERNET_MAC" {
		if len(dataStr) < 12 {
			log.Error("property ETHERNET_MAC looks corrupt. skipping.")
		} else {
			dataStr = dataStr[:2] + ":" + dataStr[2:4] + ":" +
				dataStr[4:6] + ":" + dataStr[6:8] + ":" +
				dataStr[8:10] + ":" + dataStr[10:12]
		}
	}

	if prop.name == "LED_COLOR_PROFILE" {
		if dataStr == "02" {
			dataStr = "RBG"
		} else {
			dataStr = "RGB"
		}
	}

	if prop.name == "CLOUD_URL" {
		cloudurl = dataStr
	}

	//remove "https://"
	if prop.name == "CLOUD_DDB_URL_RES" {
		if len(dataStr) < 8 {
			log.Error("property CLOUD_DDB_URL_RES looks corrupt. skipping.")
		} else {
			dataStr = dataStr[8:len(dataStr)]
		}
	}

	if prop.name == "RELAY_SERVICES_HOST" {
		dataStr = strings.Replace(dataStr, ".wigwag.io", "-relays.wigwag.io", 17)
	}

	if prop.name == "RELAY_SERVICES_HOST_RES" {
		if len(dataStr) < 8 {
			log.Error("property RELAY_SERVICES_HOST_RES looks corrupt. skipping.")
		} else {
			dataStr = strings.Replace(dataStr, ".wigwag.io", "-relays.wigwag.io", 17)
			dataStr = dataStr[8:len(dataStr)]
		}
	}

	bus.Close()

	return eeData{prop.name, dataStr}, err
}

func read_file(cert certs_anatomy) (eeData, error) {

	data, err := ioutil.ReadFile(cert.fname1)
	if err != nil {
		return eeData{"", ""}, err
	}

	dataStr := string(data)

	if cert.fname2 != "" {
		data2, err := ioutil.ReadFile(cert.fname2)
		if err != nil {
			return eeData{"", ""}, err
		}

		dataStr = dataStr + string(data2)
	}

	return eeData{cert.metavariable, dataStr}, err
}

func GetPlatformVars(dict *templates.TemplateVarDictionary, log maestroSpecs.Logger) (err error) {
	eepromData := make([]eeData, len(metadata)+len(sslCerts))

	for i := 0; i < len(metadata); i++ {
		eepromData[i], err = get_eeprom(metadata[i], log)
		if err != nil {
			return
		}

		DEBUG_OUT("wwrelayA10 %s --> %s\n", eepromData[i].name, eepromData[i].data)
	}

	cmd := exec.Command("mount", "/dev/mmcblk0p1", "/mnt/.boot/")

	err = cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		}
		log.Infof("command result, exitCode: %v\n", exitCode)
		//if exitCode is 32 that means it is already mounted
		if exitCode == 32 {
			err = nil
			log.Error("Already mounted... continuing reading certs")
		} else {
			return
		}
	}

	for i := len(metadata); i < (len(metadata) + len(sslCerts)); i++ {
		eepromData[i], err = read_file(sslCerts[i-len(metadata)])
		if err != nil {
			return
		}

		DEBUG_OUT("wwrelayA10 %s --> %s\n", eepromData[i].name, eepromData[i].data)
	}

	if strings.Contains(cloudurl, "mbed") {
		err = archiver.TarGz.Open(mcc_config, "/userdata/mbed/")
		if err != nil {
			log.Errorf("Failed to untar mcc_config- %s\n", err)
		}
	}

	// put all these found vars into the dictionary
	if dict != nil {
		for _, eeprom_entry := range eepromData {
			dict.AddArch(eeprom_entry.name, eeprom_entry.data)
		}
	}
	return
}

// func main() {
//     GetPlatformVars(nil)
// }
