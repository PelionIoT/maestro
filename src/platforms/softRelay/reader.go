package softRelay

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
	"errors"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/armPelionEdge/maestroSpecs"
	"github.com/armPelionEdge/maestroSpecs/templates"

	// "encoding/hex"
	"encoding/json"
	"encoding/pem"
	"io/ioutil"

	"github.com/armPelionEdge/maestro/log"
)

// SOFTRELAY_CONFIG_PATH is a path to
// where the particular soft relay's instance data is stored
const SOFTRELAY_CONFIG_PATH = "/config/.softRelaySetup.json"

func check(err error) {
	if err != nil {
		log.MaestroError("softrelay platform reader: check failed:", err)
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

var sslCerts = []certs_anatomy{
	certs_anatomy{"CLIENT_KEY_PEM", "/wigwag/outputConfigFiles/ssl/client.key.pem", ""},
	certs_anatomy{"CLIENT_CERT_PEM", "/wigwag/outputConfigFiles/ssl/client.cert.pem", ""},
	certs_anatomy{"SERVER_KEY_PEM", "/wigwag/outputConfigFiles/ssl/server.key.pem", ""},
	certs_anatomy{"SERVER_CERT_PEM", "/wigwag/outputConfigFiles/ssl/server.cert.pem", ""},
	certs_anatomy{"CA_CERT_PEM", "/wigwag/outputConfigFiles/ssl/ca.cert.pem", ""},
	certs_anatomy{"INTERMEDIATE_CERT_PEM", "/wigwag/outputConfigFiles/ssl/intermediate.cert.pem", ""},
	certs_anatomy{"CA_CHAIN_CERT_PEM", "/wigwag/outputConfigFiles/ssl/ca.cert.pem", "/wigwag/outputConfigFiles/ssl/intermediate.cert.pem"},
}

// var metadata = []eeprom_anatomy {
//     eeprom_anatomy {"ARCH_BRAND",             0x50, 0, 2   },
//     eeprom_anatomy {"ARCH_DEVICE",            0x50, 2, 2   },
//     eeprom_anatomy {"ARCH_UUID",              0x50, 4, 6   },
//     eeprom_anatomy {"ARCH_RELAYID",           0x50, 0, 10  },
//     eeprom_anatomy {"ARCH_HARDWARE_VERSION",  0x50, 10, 5  },
//     eeprom_anatomy {"ARCH_WW_PLATFORM",       0x50, 10, 5  },
//     eeprom_anatomy {"ARCH_FIRMWARE_VERSION",  0x50, 15, 5  },
//     eeprom_anatomy {"ARCH_RADIO_CONFIG",      0x50, 20, 2  },
//     eeprom_anatomy {"ARCH_YEAR",              0x50, 22, 1  },
//     eeprom_anatomy {"ARCH_MONTH",             0x50, 23, 1  },
//     eeprom_anatomy {"ARCH_BATCH",             0x50, 24, 1  },
//     eeprom_anatomy {"ARCH_ETHERNET_MAC",      0x50, 25, 6  },
//     eeprom_anatomy {"ARCH_SIXLBR_MAC",        0x50, 31, 8  },
//     eeprom_anatomy {"ARCH_RELAY_SECRET",      0x50, 39, 32 },
//     eeprom_anatomy {"ARCH_PAIRING_CODE",      0x50, 71, 25 },
//     eeprom_anatomy {"ARCH_LED_CONFIG",        0x50, 96, 2  },
//     eeprom_anatomy {"ARCH_LED_COLOR_PROFILE", 0x50, 96, 2  },
//     eeprom_anatomy {"ARCH_CLOUD_URL",         0x51, 0, 250 },
//     eeprom_anatomy {"ARCH_CLOUD_DEVJS_URL",   0x52, 0, 250 },
//     eeprom_anatomy {"ARCH_CLOUD_DDB_URL",     0x53, 0, 250 },
// }

var regex = "[^a-zA-Z0-9.:/-]"

// func get_eeprom(prop eeprom_anatomy) (eeData, error) {

//     bus, err := I2C.Open(&I2C.Devfs{Dev: "/dev/i2c-1"}, prop.pageaddr)
//     if err != nil {
//         return eeData{"", ""}, err
//     }

//     data := make([]byte, prop.length)

//     err = bus.ReadReg(prop.memaddr, data)
//     if err != nil {
//         return eeData{"", ""}, err
//     }

//     dataStr := string(data)

//     r, _ := regexp.Compile(regex)
//     dataStr = r.ReplaceAllString(string(data), "")

//     if prop.name == "WW_PLATFORM" {
//         dataStr = "wwrelay_v" + dataStr
//     }

//     if prop.name == "ETHERNET_MAC" || prop.name == "SIXLBR_MAC" {
//         dataStr = hex.EncodeToString(data)
//     }

//     if prop.name == "LED_COLOR_PROFILE" {
//         if dataStr == "02" {
//             dataStr = "RBG"
//         } else {
//             dataStr = "RGB"
//         }
//     }

//     bus.Close()

//     return eeData{prop.name, dataStr}, err
// }

func read_file(cert certs_anatomy) (eeData, error) {

	data, err := ioutil.ReadFile(cert.fname1)
	if err != nil {
		log.MaestroError("softrelay platform reader: Error when reading cert data:", err.Error())
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

var softrelay_config map[string]interface{}

/**
 *
 *{
    "cloudURL": "https://devcloud.wigwag.io",
    "devicejsCloudURL": "https://devcloud-devicejs.wigwag.io",
    "devicedbCloudURL": "https://devcloud-devicedb.wigwag.io",
    "relayTemplateFilePath": "/wigwag/runner.config.json",
    "relaytermTemplateFilePath": "./testConfigFiles/relayterm_template.config.json",
    "relaytermConfigFilePath": "/wigwag/wigwag-core-modules/relay-term/config/config.json",
    "relayConfigFilePath": "/wigwag/outputConfigFiles/relay.config.json",
    "rsmiTemplateFilePath": "./testConfigFiles/radioProfile.template.json",
    "rsmiConfigFilePath": "/wigwag/outputConfigFiles/radioProfile.config.json",
    "devicejsTemplateFilePath": "./testConfigFiles/template.devicejs.conf",
    "devicejsConfigFilePath": "/wigwag/outputConfigFiles/softrelaydevicejs.conf",
    "devicedbTemplateFilePath": "./testConfigFiles/template.devicedb.conf",
    "devicedbConfigFilePath": "/wigwag/outputConfigFiles/softrelaydevicedb.yaml",
    "devicedbLocalPort": 9000,
    "overwriteConfig": true,
    "overwriteSSL": true,
    "eepromFile": "/config/WWSR00001F-config.json",
    "certsMountPoint": "/mnt/.boot/",
    "certsSourcePoint": ".ssl",
    "certsMemoryBlock": "/dev/mmcblk0p1",
    "certsOutputDirectory": "/wigwag/outputConfigFiles/ssl",
    "localDatabaseDirectory": "/userdata/etc/devicejs/db",
    "relayFirmwareVersionFile": "/wigwag/etc/versions.json"
}
 *   We will read a file like the above.
*/
func read_softrelay_config(path string) (err error) {
	data, err := ioutil.ReadFile(path)

	softrelay_config = make(map[string]interface{})

	if err == nil {
		err = json.Unmarshal(data, &softrelay_config)
	}

	return
}

func read_eepromfile(path string, dict *templates.TemplateVarDictionary) (eeprom *eepromData, err error) {
	data, err := ioutil.ReadFile(path)

	eeprom = &eepromData{}

	if err == nil {
		err = json.Unmarshal(data, &eeprom)
		DEBUG_OUT("Unmarshaled eeprom data from file %s ok.\n", path)
		if err == nil {
			dict.AddArch("relaySecret", eeprom.RelaySecret)
			dict.AddArch("pairingCode", eeprom.PairingCode)
			dict.AddArch("relayID", eeprom.RelayID)
			if eeprom.SSL != nil {
				if eeprom.SSL.Client != nil {
					s := fmt.Sprintf("%s", eeprom.SSL.Client.Key)
					dict.AddArch("CLIENT_KEY_PEM", s)
					s = fmt.Sprintf("%s", eeprom.SSL.Client.Cert)
					dict.AddArch("CLIENT_CERT_PEM", s)
				} else {
					log.MaestroErrorf("Error reading eeprom data file: Missing Client key data\n")
				}
				if eeprom.SSL.Server != nil {
					s := fmt.Sprintf("%s", eeprom.SSL.Server.Key)
					dict.AddArch("SERVER_KEY_PEM", s)
					s = fmt.Sprintf("%s", eeprom.SSL.Server.Cert)
					dict.AddArch("SERVER_CERT_PEM", s)
				} else {
					log.MaestroErrorf("Error reading eeprom data file: Missing Server key data\n")
				}
				if eeprom.SSL.CA != nil {
					s := fmt.Sprintf("%s", eeprom.SSL.CA.CA)
					dict.AddArch("CA_CERT_PEM", s)
					s2 := fmt.Sprintf("%s", eeprom.SSL.CA.Intermediate)
					//                    DEBUG_OUT("Found intermediate:%s %+v\n",s2,eeprom.SSL.CA)
					dict.AddArch("INTERMEDIATE_CERT_PEM", s2)
					dict.AddArch("CA_CHAIN_CERT_PEM", s+s2)
				} else {
					log.MaestroErrorf("Error reading eeprom data file: Missing CA key data\n")
				}
			}
		} else {
			log.MaestroErrorf("Error reading eeprom data file: %s\n", err.Error())
		}

		// if err == nil {
		//     for key, val := range datamap {
		//         switch v := val.(type) {
		//          // case int: */
		//          // v2 := fmt.Sprintf("%d",v)
		//          // dict.AddArch(key,v2)
		//         case string:
		//             if key=="relaySecret" || key=="pairingCode" || key=="relayID" {
		//                 dict.AddArch(key,v)
		//             }
		//         }

		//     }
		// }
	}

	return
}

var currentEepromFile *eepromData

func GetPlatformVars(dict *templates.TemplateVarDictionary, log maestroSpecs.Logger) (err error) {
	eepromData := make([]eeData, len(sslCerts)) //len(metadata) +

	reGetUrl := regexp.MustCompile("http[s]?://([^/]+)")

	err = read_softrelay_config(SOFTRELAY_CONFIG_PATH)
	if err == nil {
		for key, val := range softrelay_config {
			switch v := val.(type) {
			case int:
				v2 := fmt.Sprintf("%d", v)
				dict.AddArch(key, v2)
			case string:
				var err2 error
				if key == "eepromFile" {
					currentEepromFile, err2 = read_eepromfile(v, dict)
					if err2 != nil {
						log.Errorf("Problem reading eepromFile: %s - %s\n", v, err2.Error())
					}
				}
				if key == "devicedbCloudURL" {
					res := reGetUrl.FindStringSubmatch(v)
					if len(res) > 1 {
						dict.AddArch("devicedbHost", res[1])
					} else {
						log.Error("devicedbCloudURL did not match regex.")
					}
				}
				if key == "relaymqCloudURL" {
					res := reGetUrl.FindStringSubmatch(v)
					if len(res) > 1 {
						dict.AddArch("relaymqHost", res[1])
					} else {
						log.Error("relaymqCloudURL did not match regex.")
					}
				}
				if key == "devicejsCloudURL" {
					res := reGetUrl.FindStringSubmatch(v)
					if len(res) > 1 {
						dict.AddArch("devicejsHost", res[1])
					} else {
						log.Error("devicejsCloudURL did not match regex.")
					}
				}
				if key == "historyCloudURL" {
					res := reGetUrl.FindStringSubmatch(v)
					if len(res) > 1 {
						dict.AddArch("historyHost", res[1])
						// create the historyID field for historical data, which should be a wildcard the domain
						// portion of the history host
						subs := strings.Split(res[1], ".")
						if len(subs) > 1 {
							s := "*"
							for i := 1; i < len(subs); i++ {
								s += "." + subs[i]
							}
							dict.AddArch("cloudHistoryWildcard", s)
						} else {
							log.Error("historyCloudURL is too short. Must have a domain - not just host. historyHost is \"%s\"", res[1])
						}
					} else {
						log.Error("historyCloudURL did not match regex.")
					}
				}
				dict.AddArch(key, v)
			}
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

// GeneratePlatformDeviceKeyNCert generates the platforms key and certificate. Takes in
// the "device id" and "account id" which are two items which must be stored in the certificate
func GeneratePlatformDeviceKeyNCert(dict *templates.TemplateVarDictionary, deviceid string, accountid string, log maestroSpecs.Logger) (key string, cert string, err error) {

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
func WritePlatformDeviceKeyNCert(dict *templates.TemplateVarDictionary, key string, cert string, log maestroSpecs.Logger) (err error) {
	// eepromData, err := read_eeprom(log)
	// if err != nil {
	// 	log.Errorf("Failed to read eeprom before writing, error- %s", err)
	// 	return
	// }

	// dict.Add("DEVICE_KEY", key)
	// dict.Add("DEVICE_CERT", cert)

	path, ok := dict.Get("eepromFile")

	if ok {
		if currentEepromFile == nil {
			currentEepromFile, err = read_eepromfile(path, dict)
			if err != nil {
				return
			}
		}
		if currentEepromFile.SSL == nil {
			currentEepromFile.SSL = new(sslStuff)
		}
		if currentEepromFile.SSL.Intermediate == nil {
			currentEepromFile.SSL.Intermediate = new(keyPair)
		}
		currentEepromFile.SSL.Intermediate.Key = key
		currentEepromFile.SSL.Intermediate.Cert = cert
		s := fmt.Sprintf("%s", currentEepromFile.SSL.Intermediate.Key)
		dict.AddArch("CLIENT_KEY_PEM", s)
		s = fmt.Sprintf("%s", currentEepromFile.SSL.Intermediate.Cert)
		dict.AddArch("CLIENT_CERT_PEM", s)
		var eepromJSON []byte
		eepromJSON, err = json.Marshal(currentEepromFile)
		if err != nil {
			// write out file
			err = ioutil.WriteFile(path, eepromJSON, 0600)
		}
	} else {
		err = errors.New("No path to eeprom file")
	}

	// if err != nil {
	// 	log.Errorf("Failed to marshal eepromData, error- %s", err)
	// 	return
	// }
	// err = ioutil.WriteFile("/sys/bus/i2c/devices/1-0050/eeprom", eepromJson, 0600)
	// if err != nil {
	// 	log.Errorf("Failed to write eeprom, error- %s", err)
	// 	return
	// }

	return
}

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

func (reader *platformInstance) WritePlatformDeviceKeyNCert(dict *templates.TemplateVarDictionary, key string, cert string, log maestroSpecs.Logger) (err error) {
	err = WritePlatformDeviceKeyNCert(dict, key, cert, log)
	return
}

func (reader *platformInstance) GeneratePlatformDeviceKeyNCert(dict *templates.TemplateVarDictionary, deviceid string, accountid string, log maestroSpecs.Logger) (key string, cert string, err error) {
	key, cert, err = GeneratePlatformDeviceKeyNCert(dict, deviceid, accountid, log)
	return
}

func (reader *platformInstance) SetOptsPlatform(map[string]interface{}) (err error) {
	return
}

func init() {
	inst := new(platformInstance)
	PlatformReader = inst
	PlatformKeyWriter = inst
}
