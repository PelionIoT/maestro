package rp200

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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/WigWagCo/maestro/log"
	"github.com/WigWagCo/maestroSpecs"
	"github.com/WigWagCo/maestroSpecs/templates"
	"github.com/mholt/archiver"
)

func _dummy() {
	var s string
	_ = fmt.Sprintf(s)
}

func check(err error) {
	if err != nil {
		log.MaestroErrorf("Platform variable check failed %s\n", err.Error())
	}
}

type ssl_certs struct {
	Key         string `json:"key"`
	Certificate string `json:"certificate"`
}

type ssl_ca struct {
	Ca           string `json:"ca"`
	Intermediate string `json:"intermediate"`
}

type ssl_obj struct {
	Client ssl_certs `json:"client"`
	Server ssl_certs `json:"server"`
	Ca     ssl_ca    `json:"ca"`
}

type eeprom_anatomy struct {
	BRAND                   string  `dict:"BRAND"`
	DEVICE                  string  `dict:"DEVICE"`
	UUID                    string  `dict:"BRAND"`
	RELAYID                 string  `json:"relayID" dict:"RELAYID"`
	HARDWARE_VERSION        string  `json:"hardwareVersion" dict:"HARDWARE_VERSION"`
	WW_PLATFORM             string  `dict:"WW_PLATFORM"`
	FIRMWARE_VERSION        string  `dict:"FIRMWARE_VERSION"`
	RADIO_CONFIG            string  `json:"radioConfig" dict:"RADIO_CONFIG"`
	YEAR                    string  `json:"year" dict:"YEAR"`
	MONTH                   string  `json:"month" dict:"MONTH"`
	BATCH                   string  `json:"batch" dict:"BATCH"`
	ETHERNET_MAC            string  `dict:"ETHERNET_MAC"`
	SIXLBR_MAC              string  `dict:"SIXLBR_MAC"`
	ETHERNET_MAC_ARRAY      []byte  `json:"ethernetMAC" dict:"ETHERNET_MAC_ARRAY"`
	SIXLBR_MAC_ARRAY        []byte  `json:"sixBMAC" dict:"SIXLBR_MAC_ARRAY"`
	RELAY_SECRET            string  `json:"relaySecret" dict:"RELAY_SECRET"`
	PAIRING_CODE            string  `json:"pairingCode" dict:"PAIRING_CODE"`
	LED_CONFIG              string  `json:"ledConfig" dict:"LED_CONFIG"`
	LED_COLOR_PROFILE       string  `json:"ledConfig" dict:"LED_COLOR_PROFILE"`
	CLOUD_URL               string  `json:"cloudURL" dict:"CLOUD_URL"`
	CLOUD_DEVJS_URL         string  `json:"devicejsCloudURL" dict:"CLOUD_DEVJS_URL"`
	CLOUD_DDB_URL           string  `json:"devicedbCloudURL" dict:"CLOUD_DDB_URL"`
	CLOUD_DDB_URL_RES       string  `dict:"CLOUD_DDB_URL_RES"`
	RELAY_HISTORY_HOST      string  `json:"relayHistoryHost" dict:"RELAY_HISTORY_HOST"`
	RELAY_HISTORY_HOST_RES  string  `dict:"RELAY_HISTORY_HOST_RES"`
	RELAY_SERVICES_HOST     string  `json:"relayServicesHost" dict:"RELAY_SERVICES_HOST"`
	RELAY_SERVICES_HOST_RES string  `dict:"RELAY_SERVICES_HOST_RES"`
	MCC_CONFIG              string  `json:"mcc_config" dict:"MCC_CONFIG"`
	SSL                     ssl_obj `json:"ssl"`
}

var mcc_config_file string = "/mnt/.boot/.ssl/mcc_config.tar.gz"

func read_eeprom(log maestroSpecs.Logger) (*eeprom_anatomy, error) {
	dat, err := ioutil.ReadFile("/sys/bus/i2c/devices/1-0050/eeprom")
	if err != nil {
		return nil, err
	}

	eepromDecoded := &eeprom_anatomy{}
	err = json.Unmarshal(dat, eepromDecoded)
	if err != nil {
		return nil, err
	}

	if len(eepromDecoded.RELAYID) < 5 {
		log.Errorf("CRITICAL - RELAYID in eeprom appears corrupt.")
	} else {
		eepromDecoded.BRAND = eepromDecoded.RELAYID[0:2]
		eepromDecoded.DEVICE = eepromDecoded.RELAYID[2:4]
		eepromDecoded.UUID = eepromDecoded.RELAYID[4:len(eepromDecoded.RELAYID)]
	}
	eepromDecoded.WW_PLATFORM = "wwgateway_v" + eepromDecoded.HARDWARE_VERSION
	eepromDecoded.FIRMWARE_VERSION = "-----"
	if eepromDecoded.LED_CONFIG == "02" {
		eepromDecoded.LED_COLOR_PROFILE = "RBG"
	} else {
		eepromDecoded.LED_COLOR_PROFILE = "RGB"
	}
	// TODO - these should use a regex, not this technique
	if len(eepromDecoded.CLOUD_DDB_URL) < 8 {
		log.Errorf("CLOUD_DDB_URL in eeprom appears corrupt")
	} else {
		eepromDecoded.CLOUD_DDB_URL_RES = eepromDecoded.CLOUD_DDB_URL[8:len(eepromDecoded.CLOUD_DDB_URL)]
	}
	if len(eepromDecoded.RELAY_HISTORY_HOST) < 8 {
		log.Errorf("RELAY_HISTORY_HOST_RES in eeprom appears corrupt")
	} else {
		eepromDecoded.RELAY_HISTORY_HOST_RES = eepromDecoded.RELAY_HISTORY_HOST[8:len(eepromDecoded.RELAY_HISTORY_HOST)]
	}
	if len(eepromDecoded.RELAY_SERVICES_HOST) < 8 {
		log.Errorf("RELAY_SERVICES_HOST_RES in eeprom appears corrupt")
	} else {
		eepromDecoded.RELAY_SERVICES_HOST_RES = eepromDecoded.RELAY_SERVICES_HOST[8:len(eepromDecoded.RELAY_SERVICES_HOST)]
	}

	eepromDecoded.ETHERNET_MAC = hex.EncodeToString(eepromDecoded.ETHERNET_MAC_ARRAY)
	if len(eepromDecoded.ETHERNET_MAC) < 12 {
		log.Errorf("ETHERNET_MAC_ARRAY in eeprom appears corrupt")
	} else {
		eepromDecoded.ETHERNET_MAC = eepromDecoded.ETHERNET_MAC[:2] + ":" + eepromDecoded.ETHERNET_MAC[2:4] + ":" +
			eepromDecoded.ETHERNET_MAC[4:6] + ":" + eepromDecoded.ETHERNET_MAC[6:8] + ":" +
			eepromDecoded.ETHERNET_MAC[8:10] + ":" + eepromDecoded.ETHERNET_MAC[10:12]
	}
	if len(eepromDecoded.SIXLBR_MAC_ARRAY) < 1 {
		log.Errorf("SIXLBR_MAC in eeprom appears corrupt")
	} else {
		eepromDecoded.SIXLBR_MAC = hex.EncodeToString(eepromDecoded.SIXLBR_MAC_ARRAY)
	}

	return eepromDecoded, err
}

func GetPlatformVars(dict *templates.TemplateVarDictionary, log maestroSpecs.Logger) (err error) {
	//Read eeprom
	eepromData, err := read_eeprom(log)
	if err != nil {
		return
	}

	if dict == nil {
		err = errors.New("no dictionary")
		return
	}

	//Write certs
	//	pathErr := os.Mkdir("/wigwag/devicejs-core-modules/Runner/.ssl", os.ModePerm)

	// err = ioutil.WriteFile("/wigwag/devicejs-core-modules/Runner/.ssl/ca.cert.pem", []byte(eepromData.SSL.Ca.Ca), 0644)
	// if err != nil {
	// 	return
	// }
	dict.AddArch("CA_PEM", eepromData.SSL.Ca.Ca)
	// err = ioutil.WriteFile("/wigwag/devicejs-core-modules/Runner/.ssl/intermediate.cert.pem", []byte(eepromData.SSL.Ca.Intermediate), 0644)
	// if err != nil {
	// 	return
	// }
	dict.AddArch("CA_INTERMEDIATE_PEM", eepromData.SSL.Ca.Intermediate)
	// err = ioutil.WriteFile("/wigwag/devicejs-core-modules/Runner/.ssl/client.key.pem", []byte(eepromData.SSL.Client.Key), 0644)
	// if err != nil {
	// 	return
	// }
	dict.AddArch("CLIENT_KEY_PEM", eepromData.SSL.Client.Key)
	// err = ioutil.WriteFile("/wigwag/devicejs-core-modules/Runner/.ssl/client.cert.pem", []byte(eepromData.SSL.Client.Certificate), 0644)
	// if err != nil {
	// 	return
	// }
	dict.AddArch("CLIENT_CERT_PEM", eepromData.SSL.Client.Certificate)
	// err = ioutil.WriteFile("/wigwag/devicejs-core-modules/Runner/.ssl/server.key.pem", []byte(eepromData.SSL.Server.Key), 0644)
	// if err != nil {
	// 	return
	// }
	dict.AddArch("SERVER_KEY_PEM", eepromData.SSL.Server.Key)
	// err = ioutil.WriteFile("/wigwag/devicejs-core-modules/Runner/.ssl/server.cert.pem", []byte(eepromData.SSL.Server.Certificate), 0644)
	// if err != nil {
	// 	return
	// }
	dict.AddArch("SERVER_CERT_PEM", eepromData.SSL.Server.Certificate)
	// err = ioutil.WriteFile("/wigwag/devicejs-core-modules/Runner/.ssl/ca-chain.cert.pem", []byte(eepromData.SSL.Ca.Ca+eepromData.SSL.Ca.Intermediate), 0644)
	// if err != nil {
	// 	return
	// }
	dict.AddArch("CA_CHAIN_PEM", eepromData.SSL.Ca.Ca+eepromData.SSL.Ca.Intermediate)

	if strings.Contains(eepromData.CLOUD_URL, "mbed") {
		mcc, err := hex.DecodeString(eepromData.MCC_CONFIG)
		if err != nil {
			log.Errorf("Failed to decode mcc, error- %s", err)
		} else {
			err = ioutil.WriteFile(mcc_config_file, []byte(mcc), 0644)
			if err != nil {
				log.Errorf("Failed to write %s, Error- %s", mcc_config_file, err)
			}

			arch := archiver.NewTarGz()
			err = arch.Unarchive(mcc_config_file, "/userdata/mbed/")
			if err != nil {
				log.Errorf("Failed to untar mcc_config- %s", err)
			}
		}
	}

	var n int
	// this adds in all the rest of the struct fields with tags 'dict'
	n, err = dict.AddTaggedStructArch(eepromData)

	fmt.Printf("PLATFORM_RP200: %d values\n", n)

	if err != nil {
		log.Errorf("Failed to add struct value: %s", err.Error())
	}

	// put all these found vars into the dictionary
	// for _, eeprom_entry := range eepromData {
	// 	dict.AddArch(eeprom_entry.name, eeprom_entry.data)
	// }
	return
}

// static_file_generators:
//   - name: "ca_pem"
//     template: "{{ARCH_SSL_CA_PEM}}"
//     output_file: "/wigwag/devicejs-core-modules/Runner/.ssl/ca.cert.pem"
//   - name: "intermediate_ca_pem"
//     template: "{{ARCH_SSL_CA_INTERMEDIATE_PEM}}"
//     output_file: "/wigwag/devicejs-core-modules/Runner/.ssl/intermediate.cert.pem"
//   - name: "client_key"
//     template: "{{ARCH_SSL_CLIENT_KEY_PEM}}"
//     output_file: "/wigwag/devicejs-core-modules/Runner/.ssl/client.key.pem"
//   - name: "client_cert"
//     template: "{{ARCH_SSL_CLIENT_CERT_PEM}}"
//     output_file: "/wigwag/devicejs-core-modules/Runner/.ssl/client.cert.pem"
//   - name: "server_key"
//     template: "{{ARCH_SSL_SERVER_KEY_PEM}}"
//     output_file: "/wigwag/devicejs-core-modules/Runner/.ssl/server.key.pem"
//   - name: "client_cert"
//     template: "{{ARCH_SSL_SERVER_CERT_PEM}}"
//     output_file: "/wigwag/devicejs-core-modules/Runner/.ssl/server.cert.pem"
//   - name: "ca_chain"
//     template: "{{ARCH_SSL_CA_CHAIN_PEM}}"
//     output_file: "/wigwag/devicejs-core-modules/Runner/.ssl/ca-chain.cert.pem"

//

// func main() {
//     GetPlatformVars(nil)
// }

// PlatformReader is a required export for a platform module
var PlatformReader maestroSpecs.PlatformReader

// PlatformKeyWriter is a required export for a platform key writing
//var PlatformKeyWriter maestroSpecs.PlatformKeyWriter

type platformInstance struct {
}

func (reader *platformInstance) GetPlatformVars(dict *templates.TemplateVarDictionary, log maestroSpecs.Logger) (err error) {
	err = GetPlatformVars(dict, log)
	return
}
func (reader *platformInstance) SetOptsPlatform(map[string]interface{}) (err error) {
	return
}

func init() {
	inst := new(platformInstance)
	PlatformReader = inst
	//	PlatformKeyWriter = inst
}
