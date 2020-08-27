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
	"fmt"

	"github.com/godbus/dbus"
	"github.com/armPelionEdge/wpa-connect/internal/log"
)

type NetworkWPA struct {
	Interface     *InterfaceWPA
	Object        dbus.BusObject
	SSID          string
	KeyMgmt       string
	SignalChannel chan *dbus.Signal
	Error         error
}

func (self *NetworkWPA) ReadProperties() *NetworkWPA {
	log.Log.Debug("ReadProperties")
	if self.Error == nil {
		if properties, err := self.Interface.WPA.get("fi.w1.wpa_supplicant1.Network.Properties", self.Object); err == nil {
			for key, value := range properties.(map[string]dbus.Variant) {
				switch key {
				case "ssid":
					self.SSID = value.Value().(string)
				case "key_mgmt":
					self.KeyMgmt = value.Value().(string)
				}
			}
		} else {
			self.Error = err
		}
	}
	return self
}

func (self *NetworkWPA) Select() *NetworkWPA {
	log.Log.Debug("Select")
	if self.Error == nil {
		if call := self.Interface.Object.Call("fi.w1.wpa_supplicant1.Interface.SelectNetwork", 0, self.Object.Path()); call.Err == nil {
		} else {
			self.Error = call.Err
		}
	}
	return self
}

func (self *NetworkWPA) AddSignalsObserver() *NetworkWPA {
	log.Log.Debug("AddSignalsObserver.Network")
	match := fmt.Sprintf("type='signal',interface='fi.w1.wpa_supplicant1.Network',path='%s'", self.Object.Path())
	if call := self.Interface.WPA.Connection.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, match); call.Err == nil {
	} else {
		self.Error = call.Err
	}
	return self
}

func (self *NetworkWPA) RemoveSignalsObserver() *NetworkWPA {
	log.Log.Debug("RemoveSignalsObserver.Network")
	match := fmt.Sprintf("type='signal',interface='fi.w1.wpa_supplicant1.Network',path='%s'", self.Object.Path())
	if call := self.Interface.WPA.Connection.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, match); call.Err == nil {
	} else {
		self.Error = call.Err
	}
	return self
}
