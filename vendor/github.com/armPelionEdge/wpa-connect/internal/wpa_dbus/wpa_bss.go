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
package wpa_dbus

import (
	"encoding/hex"
	"fmt"

	"github.com/godbus/dbus"
	"github.com/armPelionEdge/wpa-connect/internal/log"
)

type BSSWPA struct {
	Interface     *InterfaceWPA
	Object        dbus.BusObject
	BSSID         string
	SSID          string
	WPAKeyMgmt    []string
	RSNKeyMgmt    []string
	WPS           string
	Frequency     uint16
	Signal        int16
	Age           uint32
	Mode          string
	Privacy       bool
	SignalChannel chan *dbus.Signal
	Error         error
}

func (self *BSSWPA) ReadWPA() *BSSWPA {
	if self.Error == nil {
		if value, err := self.Interface.WPA.get("fi.w1.wpa_supplicant1.BSS.WPA", self.Object); err == nil {
			if value, ok := value.(map[string]dbus.Variant); ok {
				for key, variant := range value {
					if key == "KeyMgmt" {
						self.WPAKeyMgmt = variant.Value().([]string)
					}
				}
			}
		} else {
			self.Error = err
		}
	}
	return self
}

func (self *BSSWPA) ReadRSN() *BSSWPA {
	if self.Error == nil {
		if value, err := self.Interface.WPA.get("fi.w1.wpa_supplicant1.BSS.RSN", self.Object); err == nil {
			if value, ok := value.(map[string]dbus.Variant); ok {
				for key, variant := range value {
					if key == "KeyMgmt" {
						self.RSNKeyMgmt = variant.Value().([]string)
					}
				}
			}
		} else {
			self.Error = err
		}
	}
	return self
}

func (self *BSSWPA) ReadWPS() *BSSWPA {
	if self.Error == nil {
		if value, err := self.Interface.WPA.get("fi.w1.wpa_supplicant1.BSS.WPS", self.Object); err == nil {
			if value, ok := value.(map[string]dbus.Variant); ok {
				for key, variant := range value {
					if key == "Type" {
						self.WPS = variant.Value().(string)
					}
				}
			}
		} else {
			self.Error = err
		}
	}
	return self
}

func (self *BSSWPA) ReadBSSID() *BSSWPA {
	if self.Error == nil {
		if value, err := self.Interface.WPA.get("fi.w1.wpa_supplicant1.BSS.BSSID", self.Object); err == nil {
			self.BSSID = hex.EncodeToString(value.([]byte))
		} else {
			self.Error = err
		}
	}
	return self
}

func (self *BSSWPA) ReadSSID() *BSSWPA {
	if self.Error == nil {
		if value, err := self.Interface.WPA.get("fi.w1.wpa_supplicant1.BSS.SSID", self.Object); err == nil {
			self.SSID = string(value.([]byte))
		} else {
			self.Error = err
		}
	}
	return self
}

func (self *BSSWPA) ReadFrequency() *BSSWPA {
	if self.Error == nil {
		if value, err := self.Interface.WPA.get("fi.w1.wpa_supplicant1.BSS.Frequency", self.Object); err == nil {
			self.Frequency = value.(uint16)
		} else {
			self.Error = err
		}
	}
	return self
}

func (self *BSSWPA) ReadSignal() *BSSWPA {
	if self.Error == nil {
		if value, err := self.Interface.WPA.get("fi.w1.wpa_supplicant1.BSS.Signal", self.Object); err == nil {
			self.Signal = value.(int16)
		} else {
			self.Error = err
		}
	}
	return self
}

func (self *BSSWPA) ReadAge() *BSSWPA {
	if self.Error == nil {
		if value, err := self.Interface.WPA.get("fi.w1.wpa_supplicant1.BSS.Age", self.Object); err == nil {
			self.Age = value.(uint32)
		} else {
			self.Error = err
		}
	}
	return self
}

func (self *BSSWPA) ReadMode() *BSSWPA {
	if self.Error == nil {
		if value, err := self.Interface.WPA.get("fi.w1.wpa_supplicant1.BSS.Mode", self.Object); err == nil {
			self.Mode = value.(string)
		} else {
			self.Error = err
		}
	}
	return self
}

func (self *BSSWPA) ReadPrivacy() *BSSWPA {
	if self.Error == nil {
		if value, err := self.Interface.WPA.get("fi.w1.wpa_supplicant1.BSS.Privacy", self.Object); err == nil {
			self.Privacy = value.(bool)
		} else {
			self.Error = err
		}
	}
	return self
}

func (self *BSSWPA) AddSignalsObserver() *BSSWPA {
	log.Log.Debug("AddSignalsObserver.BSS")
	match := fmt.Sprintf("type='signal',interface='fi.w1.wpa_supplicant1.BSS',path='%s'", self.Object.Path())
	if call := self.Interface.WPA.Connection.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, match); call.Err == nil {
	} else {
		self.Error = call.Err
	}
	return self
}

func (self *BSSWPA) RemoveSignalsObserver() *BSSWPA {
	log.Log.Debug("RemoveSignalsObserver.BSS")
	match := fmt.Sprintf("type='signal',interface='fi.w1.wpa_supplicant1.BSS',path='%s'", self.Object.Path())
	if call := self.Interface.WPA.Connection.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, match); call.Err == nil {
	} else {
		self.Error = call.Err
	}
	return self
}
