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
	"time"
	"io/ioutil"
	"github.com/armPelionEdge/maestroSpecs"
	"github.com/armPelionEdge/maestro/log"
)

type NetworkConfigChangeHook struct{
	//struct implementing ConfigChangeHook intf
}

type ConfigChangeInfo struct{
	//struct capturing the job args to be carried out when a config change happens
	configgroup string
	fieldchanged string
	canonicalfieldname string
	futvalue interface{}
	curvalue interface{}
	index int
}

var configChangeRequestChan chan ConfigChangeInfo = nil

func ConfigChangeHandler(jobConfigChangeChan <-chan ConfigChangeInfo) {
	
	instance = GetInstance();
	for configChange := range jobConfigChangeChan {
		log.MaestroWarnf("ConfigChangeHandler:: group:%s field:%s old:%v new:%v\n", configChange.configgroup, configChange.fieldchanged, configChange.curvalue, configChange.futvalue)
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
	log.MaestroWarnf("ConfigChangeHook:SawChange: %s:%s old:%v new:%v index:%d\n", configgroup, fieldchanged, curvalue, futvalue, index)
	if(configChangeRequestChan != nil) {
		fieldnames := strings.Split(fieldchanged,".")
		log.MaestroInfof("ConfigChangeHook:fieldnames: %v\n", fieldnames)
		configChangeRequestChan <- ConfigChangeInfo{ configgroup, fieldnames[len(fieldnames)-1], fieldchanged, futvalue, curvalue, index }
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

////////////////////////////////////////////////////////////////////////////////////////////////////
// Functions to process the parameters which are changed
////////////////////////////////////////////////////////////////////////////////////////////////////

//Function to process Dhcp config change
func (inst *networkManagerInstance) ProcessDhcpConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessDhcpConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	
	switch(fieldchanged) {
	case "DhcpDisableClearAddresses":
		log.MaestroInfof("ProcessDhcpConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].DhcpDisableClearAddresses, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].DhcpDisableClearAddresses = reflect.ValueOf(futvalue).Bool();
	case "DhcpStepTimeout":
		log.MaestroInfof("ProcessDhcpConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].DhcpStepTimeout, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].DhcpStepTimeout = int(reflect.ValueOf(futvalue).Int());
		
	default:
		log.MaestroWarnf("\nProcessDnsConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//Function to process if config change
func (inst *networkManagerInstance) ProcessIfConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessIfConfigChange: %s old:%v new:%v index:%d\n", fieldchanged, curvalue, futvalue, index)
	switch(fieldchanged) {
	case "IfName":
		log.MaestroInfof("ProcessIfConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].IfName, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].IfName = reflect.ValueOf(futvalue).String();
	case "IfIndex":
		log.MaestroInfof("ProcessIfConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].IfIndex, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].IfIndex = int(reflect.ValueOf(futvalue).Int());
		
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
	case "IPv4Mask":
		log.MaestroInfof("ProcessIpv4ConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].IPv4Mask, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].IPv4Mask = int(reflect.ValueOf(futvalue).Int());
	case "IPv4BCast":
		log.MaestroInfof("ProcessIpv4ConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].IPv4BCast, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].IPv4BCast = reflect.ValueOf(futvalue).String();
	case "DhcpV4Enabled":
		log.MaestroInfof("ProcessIpv4ConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].DhcpV4Enabled, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].DhcpV4Enabled = reflect.ValueOf(futvalue).Bool();
	case "AliasAddrV4":
		log.MaestroInfof("ProcessIpv4ConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].AliasAddrV4, reflect.ValueOf(futvalue))
		aliaslen := reflect.ValueOf(futvalue).Len()
		if(len(inst.networkConfig.Interfaces[index].AliasAddrV4) < aliaslen) {
			inst.networkConfig.Interfaces[index].AliasAddrV4 = make([]maestroSpecs.AliasAddressV4, aliaslen)
		}
		for idx := 0; idx < aliaslen; idx++ {
			inst.networkConfig.Interfaces[index].AliasAddrV4[idx].IPv4Addr = reflect.ValueOf(futvalue).Index(idx).FieldByName("IPv4Addr").String()
			inst.networkConfig.Interfaces[index].AliasAddrV4[idx].IPv4Mask = reflect.ValueOf(futvalue).Index(idx).FieldByName("IPv4Mask").String()
			inst.networkConfig.Interfaces[index].AliasAddrV4[idx].IPv4BCast = reflect.ValueOf(futvalue).Index(idx).FieldByName("IPv4BCast").String()
		}
	case "TestICMPv4EchoOut":
		log.MaestroInfof("ProcessIpv4ConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].TestICMPv4EchoOut, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].TestICMPv4EchoOut = reflect.ValueOf(futvalue).String();
	default:
		log.MaestroWarnf("ProcessIpv4ConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//Function to process ipv6 config change
func (inst *networkManagerInstance) ProcessIpv6ConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessIpv6ConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	switch(fieldchanged) {
	case "IPv6Addr":
		log.MaestroInfof("ProcessIpv6ConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].IPv6Addr, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].IPv6Addr = reflect.ValueOf(futvalue).String();
		
	default:
		log.MaestroWarnf("\nProcessIpv6ConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//Function to process Mac config change
func (inst *networkManagerInstance) ProcessMacConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessMacConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	switch(fieldchanged) {
	case "HwAddr":
		log.MaestroInfof("ProcessMacConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].HwAddr, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].HwAddr = reflect.ValueOf(futvalue).String();
	case "ReplaceAddress":
		log.MaestroInfof("ProcessMacConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].ReplaceAddress, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].ReplaceAddress = reflect.ValueOf(futvalue).String();
	case "ClearAddresses":
		log.MaestroInfof("ProcessMacConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].ClearAddresses, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].ClearAddresses = reflect.ValueOf(futvalue).Bool();
	case "Aux":
		log.MaestroInfof("ProcessMacConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].Aux, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].Aux = reflect.ValueOf(futvalue).Bool();		
		
	default:
		log.MaestroWarnf("\nProcessMacConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//Function to process Wifi config change
func (inst *networkManagerInstance) ProcessWifiConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessWifiConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	//TODO: No WiFi supports exists as of now
}

//Function to process IEEE8021x config change
func (inst *networkManagerInstance) Process8021xConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("Process8021xConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	//TODO: No 8021x supports exists as of now
}

//Function to process Route config change
func (inst *networkManagerInstance) ProcessRouteConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessRouteConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	switch(fieldchanged) {
	case "RoutePriority":
		inst.networkConfig.Interfaces[index].RoutePriority = int(reflect.ValueOf(futvalue).Int());
	case "Routes":
		inst.networkConfig.Interfaces[index].Routes = reflect.ValueOf(futvalue).Interface().([]string);
	case "DontOverrideDefaultRoute":
		inst.networkConfig.DontOverrideDefaultRoute = reflect.ValueOf(futvalue).Bool();
	default:
		log.MaestroWarnf("\nProcessRouteConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//Function to process Gateway config change
func (inst *networkManagerInstance) ProcessGatewayConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessGatewayConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	switch(fieldchanged) {
	case "DefaultGateway":
		inst.networkConfig.Interfaces[index].DefaultGateway = reflect.ValueOf(futvalue).String();
	case "FallbackDefaultGateway":
		inst.networkConfig.Interfaces[index].FallbackDefaultGateway = reflect.ValueOf(futvalue).String();
	default:
		log.MaestroWarnf("\nProcessGatewayConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//Function to process Http config change
func (inst *networkManagerInstance) ProcessHttpConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessHttpConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	switch(fieldchanged) {
	case "TestHttpsRouteOut":
		inst.networkConfig.Interfaces[index].TestHttpsRouteOut = reflect.ValueOf(futvalue).String();
	default:
		log.MaestroWarnf("\nProcessHttpConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//Function to process Nameserver config change
func (inst *networkManagerInstance) ProcessNameserverConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("\nProcessNameserverConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	switch(fieldchanged) {
	case "NameserverOverrides":
		log.MaestroInfof("ProcessNameserverConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Interfaces[index].NameserverOverrides, reflect.ValueOf(futvalue))
		inst.networkConfig.Interfaces[index].NameserverOverrides = reflect.ValueOf(futvalue).String();
	case "AltResolvConf":
		log.MaestroInfof("ProcessNameserverConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.AltResolvConf, reflect.ValueOf(futvalue))
		inst.networkConfig.AltResolvConf = reflect.ValueOf(futvalue).String();
	case "Nameservers":
		log.MaestroInfof("ProcessNameserverConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.Nameservers, reflect.ValueOf(futvalue))
		inst.networkConfig.Nameservers = reflect.ValueOf(futvalue).Interface().([]string);
	case "FallbackNameservers":
		log.MaestroInfof("ProcessNameserverConfigChange: current value %s:%v new:%v\n", fieldchanged, inst.networkConfig.FallbackNameservers, reflect.ValueOf(futvalue))
		inst.networkConfig.FallbackNameservers = reflect.ValueOf(futvalue).String();
	default:
		log.MaestroWarnf("ProcessNameserverConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
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
	switch(fieldchanged) {
	case "Existing":
		inst.networkConfig.Interfaces[index].Existing = reflect.ValueOf(futvalue).String();
	default:
		log.MaestroWarnf("\nProcessConfNetIfConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}
}

//Function to process config_network config change
func (inst *networkManagerInstance) ProcessConfNetworkConfigChange(fieldchanged string, futvalue interface{}, curvalue interface{}, index int) {
	log.MaestroInfof("ProcessConfNetworkConfigChange: %s old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	switch(fieldchanged) {
	case "Existing":
		inst.networkConfig.Existing = reflect.ValueOf(futvalue).String();
	default:
		log.MaestroWarnf("\nProcessConfNetworkConfigChange:Unknown field: %s: old:%v new:%v\n", fieldchanged, curvalue, futvalue)
	}	
}

//////////////////////////////////////////////////////////////////////////////////////////
// Monitor for ConfigChangeHook
//////////////////////////////////////////////////////////////////////////////////////////
type CommitConfigChangeHook struct{
	//struct implementing CommitConfigChangeHook intf
}

var configApplyRequestChan chan bool = nil

func ConfigApplyHandler(jobConfigApplyRequestChan <-chan bool) {
	instance = GetInstance();
	for applyChange := range jobConfigApplyRequestChan {
		log.MaestroWarnf("ConfigApplyHandler::Received a apply change message: %v\n", applyChange)
		if(applyChange) {
			log.MaestroWarnf("ConfigApplyHandler::Processing apply change: %v\n", instance.configCommit.ConfigCommitFlag)
			instance.submitConfig(instance.networkConfig)
			//Setup the intfs using new config
			instance.setupInterfaces();
			instance.configCommit.ConfigCommitFlag = false
			instance.configCommit.LastUpdateTimestamp = time.Now().Format(time.RFC850)
			instance.configCommit.TotalCommitCountFromBoot = instance.configCommit.TotalCommitCountFromBoot + 1
			//Now write out the updated commit config
			err := instance.ddbConfigClient.Config(DDB_NETWORK_CONFIG_COMMIT_FLAG).Put(&instance.configCommit)
			if(err == nil) {
				log.MaestroInfof("Updating commit config object to devicedb succeeded.\n")
			} else {
				log.MaestroErrorf("Unable to update commit config object to devicedb\n")
			}
		} else {
			log.MaestroWarnf("ConfigApplyHandler::Commit flag is false: %v\n", instance.configCommit.ConfigCommitFlag)
		}
    }
}

// ChangesStart is called before reporting any changes via multiple calls to SawChange. It will only be called
// if there is at least one change to report
func (cfgHook CommitConfigChangeHook) ChangesStart(configgroup string) {
	log.MaestroInfof("CommitChangeHook:ChangesStart: %s\n", configgroup)
	if(configApplyRequestChan == nil) {
		configApplyRequestChan = make(chan bool, 10)
		go ConfigApplyHandler(configApplyRequestChan)
	}
}

// SawChange is called whenever a field changes. It will be called only once for each field which is changed.
// It will always be called after ChangesStart is called
// If SawChange return true, then the value of futvalue will replace the value of current value
func (cfgHook CommitConfigChangeHook) SawChange(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) (acceptchange bool) {
	log.MaestroWarnf("CommitChangeHook:SawChange: %s:%s old:%v new:%v index:%d\n", configgroup, fieldchanged, curvalue, futvalue, index)
	instance = GetInstance();
	instance.configCommit.ConfigCommitFlag = reflect.ValueOf(futvalue).Bool();
	configApplyRequestChan <- true
	return false;//return false as we would apply only those we successfully processed
}

// ChangesComplete is called when all changes for a specific configgroup tagname
// If ChangesComplete returns true, then all changes in that group will be assigned to the current struct
func (cfgHook CommitConfigChangeHook) ChangesComplete(configgroup string) (acceptallchanges bool) {
	log.MaestroInfof("CommitChangeHook:ChangesComplete: %s\n", configgroup)
	return false; //return false as we would apply only those we successfully processed
}