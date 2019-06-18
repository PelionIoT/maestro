package testplatform

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
	"errors"
	"fmt"

	"github.com/armPelionEdge/maestroSpecs"
	"github.com/armPelionEdge/maestroSpecs/templates"
)

type eeprom struct {
	BRAND              string `dict:"BRAND"`
	DEVICE             string `dict:"DEVICE"`
	UUID               string `dict:"BRAND"`
	RELAYID            string `json:"relayID" dict:"RELAYID"`
	HARDWARE_VERSION   string `json:"hardwareVersion" dict:"HARDWARE_VERSION"`
	WW_PLATFORM        string `dict:"WW_PLATFORM"`
	FIRMWARE_VERSION   string `dict:"FIRMWARE_VERSION"`
	RADIO_CONFIG       string `json:"radioConfig" dict:"RADIO_CONFIG"`
	YEAR               string `json:"year" dict:"YEAR"`
	MONTH              string `json:"month" dict:"MONTH"`
	BATCH              string `json:"batch" dict:"BATCH"`
	ETHERNET_MAC       string `dict:"ETHERNET_MAC"`
	SIXLBR_MAC         string `dict:"SIXLBR_MAC"`
	ETHERNET_MAC_ARRAY []byte `json:"ethernetMAC" dict:"ETHERNET_MAC_ARRAY"`
	SIXLBR_MAC_ARRAY   []byte `json:"sixBMAC" dict:"SIXLBR_MAC_ARRAY"`
	RELAY_SECRET       string `json:"relaySecret" dict:"RELAY_SECRET"`
	PAIRING_CODE       string `json:"pairingCode" dict:"PAIRING_CODE"`
	LED_CONFIG         string `json:"ledConfig" dict:"LED_CONFIG"`
	LED_COLOR_PROFILE  string `json:"ledConfig" dict:"LED_COLOR_PROFILE"`
	CLOUD_URL          string `json:"cloudURL" dict:"CLOUD_URL"`
	CLOUD_DEVJS_URL    string `json:"devicejsCloudURL" dict:"CLOUD_DEVJS_URL"`
	CLOUD_DDB_URL      string `json:"devicedbCloudURL" dict:"CLOUD_DDB_URL"`
	CLOUD_DDB_URL_RES  string `dict:"CLOUD_DDB_URL_RES"`
}

func GetPlatformVars(dict *templates.TemplateVarDictionary, log maestroSpecs.Logger) (err error) {
	//Read eeprom
	// eepromData, err := read_eeprom()
	// if err != nil {
	// 	return
	// }

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
	dict.AddArch("SSL_CA_PEM", "SSL-CA-DATA-HERE")
	// err = ioutil.WriteFile("/wigwag/devicejs-core-modules/Runner/.ssl/intermediate.cert.pem", []byte(eepromData.SSL.Ca.Intermediate), 0644)
	// if err != nil {
	// 	return
	// }
	dict.AddArch("SSL_CA_INTERMEDIATE_PEM", "SSL-CA-INTERMEDIATE-DATA-HERE")
	// err = ioutil.WriteFile("/wigwag/devicejs-core-modules/Runner/.ssl/client.key.pem", []byte(eepromData.SSL.Client.Key), 0644)
	// if err != nil {
	// 	return
	// }
	dict.AddArch("SSL_CLIENT_KEY_PEM", "SSL-CLIENT-KEY-DATA-HERE")
	// err = ioutil.WriteFile("/wigwag/devicejs-core-modules/Runner/.ssl/client.cert.pem", []byte(eepromData.SSL.Client.Certificate), 0644)
	// if err != nil {
	// 	return
	// }
	dict.AddArch("SSL_CLIENT_CERT_PEM", "SSL-CLIENT-CERT-DATA-HERE")
	// err = ioutil.WriteFile("/wigwag/devicejs-core-modules/Runner/.ssl/server.key.pem", []byte(eepromData.SSL.Server.Key), 0644)
	// if err != nil {
	// 	return
	// }
	dict.AddArch("SSL_SERVER_KEY_PEM", "SSL-SERVER-KEY-DATA-HERE")
	// err = ioutil.WriteFile("/wigwag/devicejs-core-modules/Runner/.ssl/server.cert.pem", []byte(eepromData.SSL.Server.Certificate), 0644)
	// if err != nil {
	// 	return
	// }
	dict.AddArch("SSL_SERVER_CERT_PEM", "SSL-SERVER-CERT-DATA-HERE")
	// err = ioutil.WriteFile("/wigwag/devicejs-core-modules/Runner/.ssl/ca-chain.cert.pem", []byte(eepromData.SSL.Ca.Ca+eepromData.SSL.Ca.Intermediate), 0644)
	// if err != nil {
	// 	return
	// }
	dict.AddArch("SSL_CA_CHAIN_PEM", "SSL-CA-CHAIN-DATA-HERE")

	// just test data
	eepromData := &eeprom{
		CLOUD_DDB_URL:     "https://devcloud-devicedb.wigwag.io",
		CLOUD_DDB_URL_RES: "https://devcloud-devicedb.wigwag.io",
		CLOUD_DEVJS_URL:   "https://devcloud-devicejs.wigwag.io",
		RELAYID:           "WWTEST001",
		WW_PLATFORM:       "test-platform",
		PAIRING_CODE:      "123",
		RELAY_SECRET:      "verysecret",
	}

	var n int
	// this adds in all the rest of the struct fields with tags 'dict'
	// you can pass a struct or a pointer to a struct
	n, err = dict.AddTaggedStructArch(eepromData)

	fmt.Printf("PLATFORM_RP200: %d values\n", n)

	if err != nil {
		log.Errorf("Failed to add struct value: %s\n", err.Error())
	}

	// to test errors being reported.
	// err = errors.New("OUCH!!")

	// put all these found vars into the dictionary
	// for _, eeprom_entry := range eepromData {
	// 	dict.AddArch(eeprom_entry.name, eeprom_entry.data)
	// }
	return
}

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
