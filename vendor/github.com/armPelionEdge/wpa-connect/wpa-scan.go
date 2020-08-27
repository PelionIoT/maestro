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
package wpaconnect

import (
	"github.com/godbus/dbus"
	"github.com/armPelionEdge/wpa-connect/internal/log"
	"github.com/armPelionEdge/wpa-connect/internal/wpa_dbus"
)

func (self *scanManager) Scan() (bssList []BSS, e error) {
	self.scanContext = &scanContext{}
	self.scanContext.scanDone = make(chan bool)
	if wpa, err := wpa_dbus.NewWPA(); err == nil {
		wpa.WaitForSignals(self.onScanSignal)
		if wpa.ReadInterface(self.NetInterface); wpa.Error == nil {
			iface := wpa.Interface
			iface.AddSignalsObserver()
			self.scanContext.phaseWaitForScanDone = true
			if iface.Scan(); iface.Error == nil {
				// Wait for scan_example done
				<-self.scanContext.scanDone
				if iface.ReadBSSList(); iface.Error == nil {
					for _, bss := range iface.BSSs {
						if bss.ReadBSSID().ReadSSID().ReadRSN().ReadMode().ReadSignal().
							ReadFrequency().ReadPrivacy().ReadAge().ReadWPS().ReadWPA(); bss.Error == nil {
							bssList = append(bssList, BSS{BSSID: bss.BSSID, SSID: bss.SSID, KeyMgmt: bss.RSNKeyMgmt, WPS: bss.WPS,
								Frequency: bss.Frequency, Privacy: bss.Privacy, Age: bss.Age, Mode: bss.Mode, Signal: bss.Signal})
						}
					}
				}
			} else {
				e = iface.Error
			}
			iface.RemoveSignalsObserver()
		} else {
			e = wpa.Error
		}
		wpa.StopWaitForSignals()
	} else {
		e = err
	}
	return
}

func (self *scanManager) onScanSignal(wpa *wpa_dbus.WPA, signal *dbus.Signal) {
	log.Log.Debug(signal.Name, signal.Path)
	switch signal.Name {
	case "fi.w1.wpa_supplicant1.Interface.BSSAdded":
	case "fi.w1.wpa_supplicant1.Interface.BSSRemoved":
	case "fi.w1.wpa_supplicant1.Interface.PropertiesChanged":
		break
	case "fi.w1.wpa_supplicant1.Interface.ScanDone":
		self.processScanDone(wpa, signal)
	default:
		log.Log.Debug(signal.Name, signal.Path, signal.Body)
	}
}

func (self *scanManager) processScanDone(wpa *wpa_dbus.WPA, signal *dbus.Signal) {
	log.Log.Debug("processScanDone")
	if self.scanContext.phaseWaitForScanDone {
		self.scanContext.phaseWaitForScanDone = false
		self.scanContext.scanDone <- true
	}
}

type BSS struct {
	BSSID     string
	SSID      string
	KeyMgmt   []string
	WPS       string
	Frequency uint16
	Signal    int16
	Age       uint32
	Mode      string
	Privacy   bool
}

type scanContext struct {
	phaseWaitForScanDone bool
	scanDone             chan bool
}

type scanManager struct {
	scanContext  *scanContext
	NetInterface string
}

var (
	ScanManager = &scanManager{NetInterface: "wlan0"}
)
