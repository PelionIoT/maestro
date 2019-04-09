package rp200_edge

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
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/WigWagCo/maestroSpecs"
	"github.com/WigWagCo/maestroSpecs/templates"
	"github.com/mholt/archiver"
)

func _dummy() {
	var s string
	_ = fmt.Sprintf(s)
}

// func check(err error) {
// 	if err != nil {
// 		log.Errorf("Platform variable check failed %s\n", err.Error())
// 	}
// }

type ssl_certs struct {
	Key         string `json:"key"`
	Certificate string `json:"certificate"`
}

type ssl_ca struct {
	Ca           string `json:"ca"`
	Intermediate string `json:"intermediate"`
}

type ssl_obj struct {
	Client       ssl_certs `json:"client"`
	Server       ssl_certs `json:"server"`
	Ca           ssl_ca    `json:"ca"`
	Intermediate ssl_certs `json:"intermediate"`
}

type eeprom_anatomy struct {
	SERIAL_NUMBER           string  `json:"serialNumber" dict:"SERIAL_NUMBER"`
	DEVICE_ID               string  `json:"deviceID" dict:"DEVICE_ID"`
	CATEGORY                string  `json:"deviceID" dict:"CATEGORY"`
	HARDWARE_VERSION        string  `json:"hardwareVersion" dict:"HARDWARE_VERSION"`
	WW_PLATFORM             string  `dict:"WW_PLATFORM"`
	RADIO_CONFIG            string  `json:"radioConfig" dict:"RADIO_CONFIG"`
	CREATED_AT              string  `json:"createdAt" dict:"EEPROM_CREATED_AT"`
	ETHERNET_MAC            string  `dict:"ETHERNET_MAC"`
	SIXLBR_MAC              string  `dict:"SIXLBR_MAC"`
	ETHERNET_MAC_ARRAY      []byte  `json:"ethernetMAC" dict:"ETHERNET_MAC_ARRAY"`
	SIXLBR_MAC_ARRAY        []byte  `json:"sixBMAC" dict:"SIXLBR_MAC_ARRAY"`
	LED_CONFIG              string  `json:"ledConfig" dict:"LED_CONFIG"`
	LED_COLOR_PROFILE       string  `json:"ledConfig" dict:"LED_COLOR_PROFILE"`
	CLOUD_ADDRESS           string  `json:"cloudAddress" dict:"CLOUD_ADDRESS"`
	RELAY_SERVICES_HOST     string  `json:"gatewayServicesAddress" dict:"RELAY_SERVICES_HOST"`
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

	eepromDecoded.WW_PLATFORM = "wwgateway_v" + eepromDecoded.HARDWARE_VERSION
	if eepromDecoded.LED_CONFIG == "02" {
		eepromDecoded.LED_COLOR_PROFILE = "RBG"
	} else {
		eepromDecoded.LED_COLOR_PROFILE = "RGB"
	}
	// TODO - these should use a regex, not this technique
	if len(eepromDecoded.RELAY_SERVICES_HOST) < 8 {
		log.Errorf("RELAY_SERVICES_HOST_RES in eeprom appears corrupt")
	} else {
		eepromDecoded.RELAY_SERVICES_HOST_RES = eepromDecoded.RELAY_SERVICES_HOST[8:len(eepromDecoded.RELAY_SERVICES_HOST)]
	}

	if len(eepromDecoded.ETHERNET_MAC_ARRAY) > 0 {
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
	}

	return eepromDecoded, err
}

// GeneratePlatformDeviceKeyNCert generates the platforms key and certificate. Takes in
// the "device id" and "account id" which are two items which must be stored in the certificate
func GeneratePlatformDeviceKeyNCert(deviceid string, accountid string, log maestroSpecs.Logger) (key string, cert string, err error) {

	publicKey := func(priv interface{}) interface{} {
		switch k := priv.(type) {
		case *rsa.PrivateKey:
			return &k.PublicKey
		case *ecdsa.PrivateKey:
			return &k.PublicKey
		default:
			return nil
		}
	}

	pemBlockForKey := func(priv interface{}) *pem.Block {
		switch k := priv.(type) {
		case *rsa.PrivateKey:
			return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
		case *ecdsa.PrivateKey:
			b, err := x509.MarshalECPrivateKey(k)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
				os.Exit(2)
			}
			return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
		default:
			return nil
		}
	}

	// priv, err := rsa.GenerateKey(rand.Reader, *rsaBits)
	//	priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Errorf(err.Error())
		return
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"012345678901234567890123456789ab"}, // OU is the account ID
			CommonName:   "012345678901234567890123456789ab",           // CN is the device ID
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 35),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		SignatureAlgorithm:    x509.ECDSAWithSHA256,
		PublicKeyAlgorithm:    x509.ECDSA,
	}

	/*
		hosts := strings.Split(*host, ",")
		for _, h := range hosts {
			if ip := net.ParseIP(h); ip != nil {
				template.IPAddresses = append(template.IPAddresses, ip)
			} else {
				template.DNSNames = append(template.DNSNames, h)
			}
		}

		if *isCA {
			template.IsCA = true
			template.KeyUsage |= x509.KeyUsageCertSign
		}
	*/

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		log.Errorf("Failed to create certificate: %s", err)
		return
	}
	out := &bytes.Buffer{}
	pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	//	fmt.Println(out.String())
	cert = out.String()
	out.Reset()
	pem.Encode(out, pemBlockForKey(priv))
	key = out.String()
	//	fmt.Println(out.String())

	return
}

// WritePlatformDeviceKeyNCert writes out the device key and cert to the eeprom
func WritePlatformDeviceKeyNCert(key string, cert string, log maestroSpecs.Logger) (err error) {
	eepromData, err := read_eeprom(log)
	if err != nil {
		log.Errorf("Failed to read eeprom before writing, error- %s", err)
		return
	}

	eepromData.SSL.Intermediate.Key = key
	eepromData.SSL.Intermediate.Certificate = cert

	eepromJson, err := json.Marshal(eepromData)
	if err != nil {
		log.Errorf("Failed to marshal eepromData, error- %s", err)
		return
	}
	err = ioutil.WriteFile("/sys/bus/i2c/devices/1-0050/eeprom", eepromJson, 0600)
	if err != nil {
		log.Errorf("Failed to write eeprom, error- %s", err)
		return
	}

	return
}

func GetPlatformVars(dict *templates.TemplateVarDictionary, log maestroSpecs.Logger) (err error) {
	//Read eeprom
	eepromData, err := read_eeprom(log)
	if err != nil {
		log.Errorf("Failed to read eeprom, error- %s", err)
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
	dict.AddArch("INTERMEDIATE_KEY", eepromData.SSL.Intermediate.Key)
	dict.AddArch("INTERMEDIATE_CERT", eepromData.SSL.Intermediate.Certificate)

	if strings.Contains(eepromData.CLOUD_ADDRESS, "mbed") {
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
var PlatformKeyWriter maestroSpecs.PlatformKeyWriter

type platformInstance struct {
}

func (reader *platformInstance) GetPlatformVars(dict *templates.TemplateVarDictionary, log maestroSpecs.Logger) (err error) {
	err = GetPlatformVars(dict, log)
	return
}
func (reader *platformInstance) SetOptsPlatform(map[string]interface{}) (err error) {
	return
}

func (reader *platformInstance) WritePlatformDeviceKeyNCert(dict *templates.TemplateVarDictionary, key string, cert string, log maestroSpecs.Logger) (err error) {
	err = WritePlatformDeviceKeyNCert(key, cert, log)
	return
}

func (reader *platformInstance) GeneratePlatformDeviceKeyNCert(dict *templates.TemplateVarDictionary, deviceid string, accountid string, log maestroSpecs.Logger) (key string, cert string, err error) {
	key, cert, err = GeneratePlatformDeviceKeyNCert(deviceid, accountid, log)
	return
}

func init() {
	inst := new(platformInstance)
	PlatformReader = inst
	PlatformKeyWriter = inst
}
