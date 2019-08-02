package networking

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
	"reflect"
	"strings"
	"io/ioutil"
	"github.com/armPelionEdge/maestro/log"
)

type NetworkConfigChangeHook struct{
	//struct implementing ConfigChangeHook intf
}

type ConfigChangeInfo struct{
	//struct capturing the job args to be carried out when a config change happens
	configgroup string
	fieldchanged string
	futvalue interface{}
	curvalue interface{}
	index int
}

var configChangeRequestChan chan ConfigChangeInfo = nil

func ConfigChangeHandler(jobConfigChangeChan <-chan ConfigChangeInfo) {
	
	instance = GetInstance();
	for configChange := range jobConfigChangeChan {
		log.MaestroInfof("ConfigChangeHandler:: group:%s field:%s old:%v new:%v\n", configChange.configgroup, configChange.fieldchanged, configChange.curvalue, configChange.futvalue)
        switch(configChange.configgroup) {
		case "dhcp":
			instance.ProcessDhcpConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index);
		case "if":
			instance.ProcessIfConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index);			
		case "ipv4":
			instance.ProcessIpv4ConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index);			
		case "ipv6":
			instance.ProcessIpv6ConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index);			
		case "mac":
			instance.ProcessMacConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index);			
		case "wifi":
			instance.ProcessWifiConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index);			
		case "IEEE8021x":
			instance.Process8021xConfigChange(configChange.fieldchanged,configChange. futvalue, configChange.curvalue, configChange.index);			
		case "route":
			instance.ProcessRouteConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index);			
		case "http":
			instance.ProcessHttpConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index);			
		case "nameserver":
			instance.ProcessNameserverConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index);			
		case "gateway":
			instance.ProcessGatewayConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index);			
		case "dns":
			instance.ProcessDnsConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index);			
		case "config_netif":
			instance.ProcessConfNetIfConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index);			
		case "config_network":
			instance.ProcessConfNetworkConfigChange(configChange.fieldchanged, configChange.futvalue, configChange.curvalue, configChange.index);			
		default:
			log.MaestroWarnf("\nConfigChangeHook:Unknown field or group: %s:%s old:%v new:%v\n", configChange.configgroup, configChange.fieldchanged, configChange.curvalue, configChange.futvalue)
		}
    }
}

// ChangesStart is called before reporting any changes via multiple calls to SawChange. It will only be called
// if there is at least one change to report
func (cfgHook NetworkConfigChangeHook) ChangesStart(configgroup string) {
	log.MaestroInfof("ConfigChangeHook:ChangesStart: %s\n", configgroup)
	if(configChangeRequestChan == nil) {
		configChangeRequestChan = make(chan ConfigChangeInfo, 100)
		go ConfigChangeHandler(configChangeRequestChan)
	}
}

// SawChange is called whenever a field changes. It will be called only once for each field which is changed.
// It will always be called after ChangesStart is called
// If SawChange return true, then the value of futvalue will replace the value of current value
func (cfgHook NetworkConfigChangeHook) SawChange(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) (acceptchange bool) {
	log.MaestroInfof("ConfigChangeHook:SawChange: %s:%s old:%v new:%v index:%d\n", configgroup, fieldchanged, curvalue, futvalue, index)
	if(configChangeRequestChan != nil) {
		fieldnames := strings.Split(fieldchanged,".")
		configChangeRequestChan <- ConfigChangeInfo{ configgroup, fieldnames[len(fieldnames)-1], futvalue, curvalue, index }
	} else {
		log.MaestroErrorf("ConfigChangeHook:Config change chan is nil, unable to process change")
	}

	return false;//return false as we would apply only those we successfully processed
}

// ChangesComplete is called when all changes for a specific configgroup tagname
// If ChangesComplete returns true, then all changes in that group will be assigned to the current struct
func (cfgHook NetworkConfigChangeHook) ChangesComplete(configgroup string) (acceptallchanges bool) {
	log.MaestroInfof("ConfigChangeHook:ChangesComplete: %s\n", configgroup)
	return false; //return false as we would apply only those we successfully processed
}

//Function to process Dhcp config change
func (inst *networkManagerInstance) ProcessDhcpConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessDhcpConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
}

//Function to process if config change
func (inst *networkManagerInstance) ProcessIfConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessIfConfigChange: %s old:%v new:%v index:%d\n", fieldchanged, curvalue, futvalue, index)
	switch(fieldchanged) {
	case "IfName":
		log.MaestroInfof("ProcessIfConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].IfName, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].IfName = reflect.ValueOf(futvalue).String();
		
	default:
		log.MaestroWarnf("\nProcessDnsConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//Function to process ipv4 config change
func (inst *networkManagerInstance) ProcessIpv4ConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessIpv4ConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	switch(fieldchanged) {
	case "IPv4Addr":
		log.MaestroInfof("ProcessIpv4ConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].IPv4Addr, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].IPv4Addr = reflect.ValueOf(futvalue).String();
		
	default:
		log.MaestroWarnf("\nProcessIpv4ConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//Function to process ipv6 config change
func (inst *networkManagerInstance) ProcessIpv6ConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessIpv6ConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
}

//Function to process Mac config change
func (inst *networkManagerInstance) ProcessMacConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessMacConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
}

//Function to process Wifi config change
func (inst *networkManagerInstance) ProcessWifiConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessWifiConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
}

//Function to process IEEE8021x config change
func (inst *networkManagerInstance) Process8021xConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("Process8021xConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
}

//Function to process Route config change
func (inst *networkManagerInstance) ProcessRouteConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessRouteConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
}

//Function to process Http config change
func (inst *networkManagerInstance) ProcessHttpConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessHttpConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
}

//Function to process Nameserver config change
func (inst *networkManagerInstance) ProcessNameserverConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("\nProcessNameserverConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	switch(fieldchanged) {
	case "AltResolvConf":
		log.MaestroInfof("ProcessNameserverConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.AltResolvConf, reflect.ValueOf(futvalue))
		inst.networkConfig.AltResolvConf = reflect.ValueOf(futvalue).String();
	default:
		log.MaestroWarnf("ProcessNameserverConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//Function to process Gateway config change
func (inst *networkManagerInstance) ProcessGatewayConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessGatewayConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
}

//Function to process Dns config change
func (inst *networkManagerInstance) ProcessDnsConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessDnsConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	switch(fieldchanged) {
	case "DnsIgnoreDhcp":
		log.MaestroInfof("ProcessDnsConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.DnsIgnoreDhcp, reflect.ValueOf(futvalue))
		inst.networkConfig.DnsIgnoreDhcp = reflect.ValueOf(futvalue).Bool();
	case "DnsRunLocalCaching":
		log.MaestroInfof("ProcessDnsConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.DnsRunLocalCaching, reflect.ValueOf(futvalue))
		inst.networkConfig.DnsRunLocalCaching = reflect.ValueOf(futvalue).Bool();
	case "DnsRunRootLookup":
		log.MaestroInfof("ProcessDnsConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.DnsRunRootLookup, reflect.ValueOf(futvalue))
		inst.networkConfig.DnsRunRootLookup = reflect.ValueOf(futvalue).Bool();
	case "DnsForwardTo":
		log.MaestroInfof("ProcessDnsConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.DnsForwardTo, reflect.ValueOf(futvalue))
		inst.networkConfig.DnsForwardTo = reflect.ValueOf(futvalue).String();
	case "DnsHostsData":
		log.MaestroInfof("ProcessDnsConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.DnsHostsData, reflect.ValueOf(futvalue))
		inst.networkConfig.DnsHostsData = reflect.ValueOf(futvalue).String();
		//Write the contents to /etc/hosts
		err := ioutil.WriteFile("/etc/hosts", []byte(inst.networkConfig.DnsHostsData), 0644)
		if(err != nil) {
			log.MaestroErrorf("ProcessDnsConfigChange: unable to updtae /etc/hosts err:%v", err)
		}
		
	default:
		log.MaestroWarnf("\nProcessDnsConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//Function to process config_netif config change
func (inst *networkManagerInstance) ProcessConfNetIfConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessConfNetIfConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	for idx:=0; idx < len(inst.networkConfig.Interfaces); idx++ {
		inst.networkConfig.Interfaces[idx].Existing = reflect.ValueOf(futvalue).String();
	}
}

//Function to process config_network config change
func (inst *networkManagerInstance) ProcessConfNetworkConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessConfNetworkConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	inst.networkConfig.Existing = reflect.ValueOf(futvalue).String();
}

