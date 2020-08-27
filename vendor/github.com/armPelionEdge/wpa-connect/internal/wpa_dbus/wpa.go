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
	"errors"
	"fmt"

	"github.com/godbus/dbus"
	"github.com/armPelionEdge/wpa-connect/internal/log"
)

type WPA struct {
	Connection    *dbus.Conn
	Object        dbus.BusObject
	Interfaces    []InterfaceWPA
	Interface     *InterfaceWPA
	SignalChannel chan *dbus.Signal
	Error         error
}

func NewWPA() (wpa *WPA, e error) {
	if conn, err := dbus.SystemBus(); err == nil {
		if obj := conn.Object("fi.w1.wpa_supplicant1", "/fi/w1/wpa_supplicant1"); obj != nil {
			wpa = &WPA{Connection: conn, Object: obj}
		} else {
			conn.Close()
			err = errors.New("Can't create WPA object")
		}
	} else {
		e = err
	}

	return
}

func (self *WPA) ReadInterface(ifname string) *WPA {
	if self.Error == nil {
		if call := self.Object.Call("fi.w1.wpa_supplicant1.GetInterface", 0, ifname); call.Err == nil {
			var objectPath = dbus.ObjectPath(call.Body[0].(dbus.ObjectPath))
			self.Interface = &InterfaceWPA{WPA: self, Object: self.Connection.Object("fi.w1.wpa_supplicant1", objectPath)}
		} else {
			self.Error = call.Err
		}
	}
	return self
}

func (self *WPA) ReadInterfaceList() *WPA {
	if self.Error == nil {
		if interfaces, err := self.get("fi.w1.wpa_supplicant1.Interfaces", self.Object); err == nil {
			newInterfaces := []InterfaceWPA{}
			for _, interfaceObjectPath := range interfaces.([]dbus.ObjectPath) {
				iface := InterfaceWPA{WPA: self, Object: self.Connection.Object("fi.w1.wpa_supplicant1", interfaceObjectPath)}
				newInterfaces = append(newInterfaces, iface)
			}
			self.Interfaces = newInterfaces
		} else {
			self.Error = err
		}
	}
	return self
}

func (self *WPA) get(name string, target dbus.BusObject) (value interface{}, e error) {
	if variant, err := target.GetProperty(name); err == nil {
		value = variant.Value()
	} else {
		e = err
	}
	return
}

func (self *WPA) WaitForSignals(callBack func(*WPA, *dbus.Signal)) *WPA {
	log.Log.Debug("WaitForSignals")
	self.SignalChannel = make(chan *dbus.Signal, 10)
	self.Connection.Signal(self.SignalChannel)
	go func() {
		for ch := range self.SignalChannel {
			callBack(self, ch)
		}
	}()
	return self
}

func (self *WPA) StopWaitForSignals() *WPA {
	log.Log.Debug("StopWaitForSignals")
	self.Connection.RemoveSignal(self.SignalChannel)
	return self
}

func (self *WPA) AddSignalsObserver() *WPA {
	log.Log.Debug("AddSignalsObserver.WPA")
	match := fmt.Sprintf("type='signal',interface='fi.w1.wpa_supplicant1',path='%s'", self.Object.Path())
	if call := self.Connection.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, match); call.Err == nil {
	} else {
		self.Error = call.Err
	}
	return self
}

func (self *WPA) RemoveSignalsObserver() *WPA {
	log.Log.Debug("RemoveSignalsObserver.WPA")
	match := fmt.Sprintf("type='signal',interface='fi.w1.wpa_supplicant1',path='%s'", self.Object.Path())
	if call := self.Connection.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, match); call.Err == nil {
	} else {
		self.Error = call.Err
	}
	return self
}
