package common

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
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/PelionIoT/maestro/debugging"
	"github.com/PelionIoT/maestroSpecs"
	"github.com/PelionIoT/maestroSpecs/templates"
	"github.com/mholt/archiver"
)

// format of file is:
/*
{
    "serialNumber": "SOFT0005T9",
    "OU": "00ced56faf434f3ab7cf49470977e801",
    "deviceID": "2f9aad184e404d8bbafd5ad949552cf0",
    "hardwareVersion": "SOFT_GW",
    "radioConfig": "00",
    "ledConfig": "01",
    "category": "development",
    "ethernetMAC": [
        0,
        165,
        9,
        106,
        110,
        148
    ],
    "sixBMAC": [
        0,
        165,
        9,
        0,
        1,
        106,
        110,
        148
    ],
    "hash": [],
    "gatewayServicesAddress": "https://gw.stuff.com",
    "apiServerAddress": "https://api.stuff.com",
    "cloudAddress": "https://gw.stuff.com",
    "ssl": {
        "client": {
            "key": "-----BEGIN EC PARAMETERS-----\nBggqhkjOPQMBBw==\n-----END EC PARAMETERS-----\n-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIOX0N9QQ38cIIbwOtI+NhMxfIo+M8u0b/N3IiV8cBa0ToAoGCCqGSM49\nAwEHoUQDQgAEIS8z+tTTuI0FCaOXigC5ER7iQUknFI2GB5Sv0zG+82KID1VaIKYF\nAVQzMN38O0Ix9xpAS38Y8oyrsDAn8nUKgA==\n-----END EC PRIVATE KEY-----\n",
            "certificate": "-----BEGIN CERTIFICATE-----\nMIIB8DCCAZUCCQCHsqepYIG2jzAKBggqhkjOPQQDAjBsMQswCQYDVQQGEwJVUzEO\nMAwGA1UECAwFVGV4YXMxDzANBgNVBAcMBkF1c3RpbjEMMAoGA1UECgwDQVJNMS4w\nLAYDVQQDDCVyZWxheXNfYXJtLmlvX2dhdGV3YXlfY2FfaW50ZXJtZWRpYXRlMB4X\nDTE5MDMwMTE5MDgxNloXDTM5MDIyNDE5MDgxNlowgZIxCzAJBgNVBAYTAlVTMQ4w\nDAYDVQQIDAVUZXhhczEPMA0GA1UEBwwGQXVzdGluMQwwCgYDVQQKDANBUk0xKTAn\nBgNVBAsMIDAwY2VkNTZmYWY0MzRmM2FiN2NmNDk0NzA5NzdlODAxMSkwJwYDVQQD\nDCAyZjlhYWQxODRlNDA0ZDhiYmFmZDVhZDk0OTU1MmNmMDBZMBMGByqGSM49AgEG\nCCqGSM49AwEHA0IABCEvM/rU07iNBQmjl4oAuREe4kFJJxSNhgeUr9MxvvNiiA9V\nWiCmBQFUMzDd/DtCMfcaQEt/GPKMq7AwJ/J1CoAwCgYIKoZIzj0EAwIDSQAwRgIh\nANT+XJC5CxOIHlZQAToRFB+dNbrtMrS5+XxZihdxVxUBAiEApW1OlJu8gMKD/zIG\nhBxTbp3kfPglfQExahzNWzCCvnw=\n-----END CERTIFICATE-----\n"
        },
        "server": {
            "key": "-----BEGIN EC PARAMETERS-----\nBggqhkjOPQMBBw==\n-----END EC PARAMETERS-----\n-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIOX0N9QQ38cIIbwOtI+NhMxfIo+M8u0b/N3IiV8cBa0ToAoGCCqGSM49\nAwEHoUQDQgAEIS8z+tTTuI0FCaOXigC5ER7iQUknFI2GB5Sv0zG+82KID1VaIKYF\nAVQzMN38O0Ix9xpAS38Y8oyrsDAn8nUKgA==\n-----END EC PRIVATE KEY-----\n",
            "certificate": "-----BEGIN CERTIFICATE-----\nMIIB8DCCAZUCCQCHsqepYIG2jzAKBggqhkjOPQQDAjBsMQswCQYDVQQGEwJVUzEO\nMAwGA1UECAwFVGV4YXMxDzANBgNVBAcMBkF1c3RpbjEMMAoGA1UECgwDQVJNMS4w\nLAYDVQQDDCVyZWxheXNfYXJtLmlvX2dhdGV3YXlfY2FfaW50ZXJtZWRpYXRlMB4X\nDTE5MDMwMTE5MDgxNloXDTM5MDIyNDE5MDgxNlowgZIxCzAJBgNVBAYTAlVTMQ4w\nDAYDVQQIDAVUZXhhczEPMA0GA1UEBwwGQXVzdGluMQwwCgYDVQQKDANBUk0xKTAn\nBgNVBAsMIDAwY2VkNTZmYWY0MzRmM2FiN2NmNDk0NzA5NzdlODAxMSkwJwYDVQQD\nDCAyZjlhYWQxODRlNDA0ZDhiYmFmZDVhZDk0OTU1MmNmMDBZMBMGByqGSM49AgEG\nCCqGSM49AwEHA0IABCEvM/rU07iNBQmjl4oAuREe4kFJJxSNhgeUr9MxvvNiiA9V\nWiCmBQFUMzDd/DtCMfcaQEt/GPKMq7AwJ/J1CoAwCgYIKoZIzj0EAwIDSQAwRgIh\nANT+XJC5CxOIHlZQAToRFB+dNbrtMrS5+XxZihdxVxUBAiEApW1OlJu8gMKD/zIG\nhBxTbp3kfPglfQExahzNWzCCvnw=\n-----END CERTIFICATE-----\n"
        },
        "ca": {
            "ca": "-----BEGIN CERTIFICATE-----\nMIIB1DCCAXqgAwIBAgIJAKPyrSjzqa81MAoGCCqGSM49BAMCMF8xCzAJBgNVBAYT\nAlVTMQ4wDAYDVQQIDAVUZXhhczEPMA0GA1UEBwwGQXVzdGluMQwwCgYDVQQKDANB\nUk0xITAfBgNVBAMMGHJlbGF5c19hcm0uaW9fZ2F0ZXdheV9jYTAgFw0xOTAzMDEx\nOTA4MTZaGA8yMDU0MDIyMDE5MDgxNlowXzELMAkGA1UEBhMCVVMxDjAMBgNVBAgM\nBVRleGFzMQ8wDQYDVQQHDAZBdXN0aW4xDDAKBgNVBAoMA0FSTTEhMB8GA1UEAwwY\ncmVsYXlzX2FybS5pb19nYXRld2F5X2NhMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcD\nQgAEdIxJ3NVkgk4uzaheIpEQ1JN8TllkmICsxREOD34Qb7TaLxanW0XxZ3o+Y8Q0\nIiM178Sm1G7LwadRXveiPSe52KMdMBswDAYDVR0TBAUwAwEB/zALBgNVHQ8EBAMC\nAYYwCgYIKoZIzj0EAwIDSAAwRQIhAOkKElimdoxNEOjXLOoIZzcK5TrWDq/KuB3m\nzbL9psYLAiByhUd/B9AxiuUWX8OP6Z8zTBsz7NBiIhYAdov33+753w==\n-----END CERTIFICATE-----\n",
            "intermediate": "-----BEGIN CERTIFICATE-----\nMIIB3zCCAYWgAwIBAgIJAK6X1ZPEgdN4MAoGCCqGSM49BAMCMF8xCzAJBgNVBAYT\nAlVTMQ4wDAYDVQQIDAVUZXhhczEPMA0GA1UEBwwGQXVzdGluMQwwCgYDVQQKDANB\nUk0xITAfBgNVBAMMGHJlbGF5c19hcm0uaW9fZ2F0ZXdheV9jYTAeFw0xOTAzMDEx\nOTA4MTZaFw0zOTAyMjQxOTA4MTZaMGwxCzAJBgNVBAYTAlVTMQ4wDAYDVQQIDAVU\nZXhhczEPMA0GA1UEBwwGQXVzdGluMQwwCgYDVQQKDANBUk0xLjAsBgNVBAMMJXJl\nbGF5c19hcm0uaW9fZ2F0ZXdheV9jYV9pbnRlcm1lZGlhdGUwWTATBgcqhkjOPQIB\nBggqhkjOPQMBBwNCAATOK0BXQmhntge2QSbu8yqev4LBtNmlgsO/CnXKdCVEI2ST\nqY+wvO9YwccFyuxCU/+nnOq8kzPaSXx5hkzSNYRjox0wGzAMBgNVHRMEBTADAQH/\nMAsGA1UdDwQEAwIBhjAKBggqhkjOPQQDAgNIADBFAiEAjX78Ilpcc35y4KWohP53\nOMy19wx/4hKwqLfyKbPiXOcCIB+eTYT/H4YIslzPUtsnPhLyaHN/2QG+mzSzfvcc\nv2Rc\n-----END CERTIFICATE-----\n"
        }
    }
}

*/
// This should be passed through the maestro config
var mcc_config_file string = "/userdata/mcc_config.tar.gz"
var mcc_untar_dir string = "/userdata/mbed/"
var mcc_config_dir string = "/userdata/mbed/mcc_config"
var mcc_config_working_dir string = "/userdata/mbed/mcc_config/WORKING"

// IdentitySSLKeySet is used to hold a set of SSL key pair/ca/intermediate data
type IdentitySSLKeySet struct {
	// not all fields always used
	Key          string `json:"key"`
	Cert         string `json:"certificate"`
	CA           string `json:"ca"`
	Intermediate string `json:"intermediate"`
}

// IdentitySSLData is a structure to hold SSL information for this gateway node
type IdentitySSLData struct {
	Client *IdentitySSLKeySet `json:"client"`
	Server *IdentitySSLKeySet `json:"server"`
	CA     *IdentitySSLKeySet `json:"ca"`
}

// IdentityJSONFile is a structure which holds data which is read from identity.json - a file generated by the Pelion
// Edge Gateway provisioning tools.
type IdentityJSONFile struct {
	SerialNumber string `json:"serialNumber" dict:"SERIAL_NUMBER"`
	OU           string `json:"OU" dict:"OU"`
	DeviceID     string `json:"DeviceID" dict:"DEVICE_ID"`
	// HardwareVersion is a string identitying the type of hardware and version of hardware
	HardwareVersion string `json:"hardwareVersion" dict:"HARDWARE_VERSION"`
	EdgePlatform    string `dict:"EDGE_PLATFORM"`
	RadioConfig     string `json:"radioConfig" dict:"RADIO_CONFIG"`
	LEDConfig       string `json:"ledConfig" dict:"LED_CONFIG"`
	LEDProfile      string `json:"ledConfig" dict:"LED_COLOR_PROFILE"`
	ConfigCategory  string `json:"category" dict:"CONFIG_CATEGORY"`
	EthernetMAC     []byte `json:"ethernetMAC"`
	SixMAC          []byte `json:"sixBMAC"`
	// hash []string  // ??
	GatewayServicesAddress  string           `json:"gatewayServicesAddress" dict:"GW_SERVICES_URL"`
	GatewayServicesResource string           `dict:"GW_SERVICES_RESRC"`
	APIServerAddress        string           `json:"apiServerAddress" dict:"API_SERVICES_URL"`
	MccConfig               string           `json:"mcc_config" dict:"MCC_CONFIG"`
	SSL                     *IdentitySSLData `json:"ssl"`
}

func convertByteSliceToHexString(bytes []byte) (ret string) {
	ret = hex.EncodeToString(bytes)
	return
}

func IsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	n, err := f.Readdirnames(1)
	if len(n) == 0 {
		return true, nil
	}
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func ReadIdentityFile(path string, dict *templates.TemplateVarDictionary, log maestroSpecs.Logger) (identityData *IdentityJSONFile, err error) {
	data, err := ioutil.ReadFile(path)

	identityData = &IdentityJSONFile{}

	if err == nil {
		err = json.Unmarshal(data, &identityData)
		debugging.DEBUG_OUT("Unmarshaled identity data from file %s ok.\n", path)
		if err == nil {
			// add all the fields with a 'dict' tag (takes care of all the top level string values)
			dict.AddTaggedStructArch(identityData)

			if len(identityData.EthernetMAC) > 0 {
				if len(identityData.EthernetMAC) != 6 {
					log.Errorf("Identity file %s has invalid length data for ethernetMAC address\n", path)
				} else {
					var eMac string = convertByteSliceToHexString(identityData.EthernetMAC)
					if len(eMac) < 12 {
						log.Error("EthernetMAC in identity appears corrupt")
					} else {
						eMac = eMac[:2] + ":" + eMac[2:4] + ":" + eMac[4:6] + ":" + eMac[6:8] + ":" + eMac[8:10] + ":" + eMac[10:12]
					}
					dict.AddArch("ETHERNET_MAC", eMac)
				}
			} else {
				log.Warnf("Identity file %s has no ethernetMAC address\n", path)
			}
			if len(identityData.SixMAC) > 0 && len(identityData.SixMAC) != 8 {
				log.Errorf("Identity file %s has invalid length data for ethernetMAC address\n", path)
			} else {
				dict.AddArch("SIXLBR_MAC", convertByteSliceToHexString(identityData.SixMAC))
			}

			// This is now redundant, in future we should remove it
			identityData.EdgePlatform = "wwgateway_v" + identityData.HardwareVersion
			dict.AddArch("EDGE_PLATFORM", identityData.EdgePlatform)

			// This is now redundant, in future we should remove it
			if identityData.LEDConfig == "02" {
				identityData.LEDProfile = "RBG"
			} else {
				identityData.LEDProfile = "RGB"
			}
			dict.AddArch("LED_COLOR_PROFILE", identityData.LEDProfile)

			// Need the Gateway services resources (without protocol), required by devicejs and devicedb configuration
			if len(identityData.GatewayServicesAddress) < 8 {
				log.Error("GatewayServicesAddress in identity appears corrupt")
			} else {
				identityData.GatewayServicesResource = identityData.GatewayServicesAddress[8:len(identityData.GatewayServicesAddress)]
			}
			dict.AddArch("GW_SERVICES_RESRC", identityData.GatewayServicesResource)

			// Validate if mcc_config directory is empty or not. If empty, then delete it
			res, _ := IsEmpty(mcc_config_working_dir)
			if res {
				// mcc_config is empty, delete it as unarchiver will not populate it otherwise
				os.RemoveAll(mcc_config_dir)
			}

			// Untar the mcc_config and save it in userdata
			mcc, err := hex.DecodeString(identityData.MccConfig)
			if err != nil {
				log.Errorf("Failed to decode mcc, error- %s\n", err)
			} else {
				err = ioutil.WriteFile(mcc_config_file, []byte(mcc), 0644)
				if err != nil {
					log.Errorf("Failed to write %s, Error- %s\n", mcc_config_file, err)
				}

				arch := archiver.NewTarGz()
				err = arch.Unarchive(mcc_config_file, mcc_untar_dir)
				if err != nil {
					log.Errorf("Failed to untar mcc_config- %s\n", err)
				}
			}

			if identityData.SSL != nil {
				if identityData.SSL.Client != nil {
					s := fmt.Sprintf("%s", identityData.SSL.Client.Key)
					dict.AddArch("CLIENT_KEY_PEM", s)
					s = fmt.Sprintf("%s", identityData.SSL.Client.Cert)
					dict.AddArch("CLIENT_CERT_PEM", s)
				} else {
					log.Warnf("Identity file %s has no ssl.client info\n", path)
				}
				if identityData.SSL.Server != nil {
					s := fmt.Sprintf("%s", identityData.SSL.Server.Key)
					dict.AddArch("SERVER_KEY_PEM", s)
					s = fmt.Sprintf("%s", identityData.SSL.Server.Cert)
					dict.AddArch("SERVER_CERT_PEM", s)
				} else {
					log.Warnf("Identity file %s has no ssl.server info\n", path)
				}
				if identityData.SSL.CA != nil {
					s := fmt.Sprintf("%s", identityData.SSL.CA.Intermediate)
					dict.AddArch("INTERMEDIATE_CERT_PEM", s)
					s2 := fmt.Sprintf("%s", identityData.SSL.CA.CA)
					dict.AddArch("CA_CERT_PEM", s2)
					dict.AddArch("CA_CHAIN_CERT_PEM", s2+s)
				} else {
					log.Warnf("Identity file %s has no ssl.ca info\n", path)
				}

			}
		}
	}

	return
}
