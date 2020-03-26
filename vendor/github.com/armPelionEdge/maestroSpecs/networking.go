package maestroSpecs

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

//    "encoding/json"

type WiFiSettings struct {
	// TBD
}

type IEEE8021x struct {
	// TBD
}

type AliasAddressV4 struct {
	// IPv4 Address
	IPv4Addr string `yaml:"ipv4_addr" json:"ipv4_addr"`
	// IPv4 Subnet Mask
	IPv4Mask string `yaml:"ipv4_mask" json:"ipv4_mask"`
	// IPv4 Broadcast Addr. If empty, set automatically
	IPv4BCast string `yaml:"ipv4_bcast" json:"ipv4_bcast"`
}

const (
	// MaxRoutePriority is largest number which can be used for RoutePriority setting
	MaxRoutePriority = 9
)

type NetIfConfigPayload struct {
	// like "eth0"
	IfName string `yaml:"if_name" json:"if_name" netgroup:"if"`
	// use either this or IfName, index is a non-zero positive integer
	// IfName is preferred and takes precedence
	IfIndex int `yaml:"if_index" json:"if_index" netgroup:"if"`
	// use DHCP?
	DhcpV4Enabled bool `yaml:"dhcpv4" json:"dhcpv4" netgroup:"ipv4"`
	// IPv4 Address
	IPv4Addr string `yaml:"ipv4_addr" json:"ipv4_addr" netgroup:"ipv4"`
	// IPv4 Subnet Mask  - of the CIDR form, such as in 192.168.1.21/24
	// then this number would be 24
	IPv4Mask int `yaml:"ipv4_mask" json:"ipv4_mask" netgroup:"ipv4"`
	// IPv4 Broadcast Addr. If empty, set automatically
	IPv4BCast string `yaml:"ipv4_bcast" json:"ipv4_bcast" netgroup:"ipv4"`
	// additional addresses to assign to the interface, if any
	AliasAddrV4 []AliasAddressV4 `yaml:"alias_ipv4" json:"alias_ipv4" netgroup:"ipv4"`
	// IPv6 Address
	IPv6Addr string `yaml:"ipv6_addr" json:"ipv6_addr"  netgroup:"ipv6"`
	// // IPv4 Subnet Mask
	// IPv6Mask string `yaml:"ipv4_mask" json:"ipv4_mask"`

	// HwAddr is a MAC address or similar address
	// MAC addresses should be formatted "aa:bb:cc:01:23:45" etc.
	// 48 or 64 bit addresses can be stated, or whatever may be supported by the hardware
	// such as a 20-octet IP for InifiniBand
	HwAddr string `yaml:"hw_addr" json:"hw_addr" netgroup:"mac"`

	// WiFi settings, if any
	WiFiSettings *WiFiSettings `yaml:"wifi" json:"wifi" netgroup:"wifi"`

	// IEEE8021x is the IEEE 802.1x auth settings
	IEEE8021x *IEEE8021x `yaml:"ieee8021x" json:"ieee8021x" netgroup:"IEEE8021x"`

	// These typically optional:
	// if not empty, this address will be deleted before setting the new address
	ReplaceAddress string `yaml:"replace_addr" json:"replace_addr" netgroup:"mac"`
	// if true, this will clear any existing addresses assigned to the interface
	// before setting up the specified addresses
	ClearAddresses bool `yaml:"clear_addresses" json:"clear_addresses" netgroup:"mac"`
	// if true, the interface should be configured, be turned off / disabled (rare)
	Down bool `yaml:"down" json:"down" group:"mac"`

	// DefaultGateway is a default route associated with this interface, which should be set if this
	// interface is up. If empty, none will be setup - unless DHCP provides one. If one is provided
	// here it will override the DHCP provided one. The Priority field
	// should help determine which route has the best metric, which will allow the kernel
	// to use the fastest route.
	DefaultGateway string `yaml:"default_gateway" json:"default_gateway"  netgroup:"gateway"`

	// NOT IMPLEMENTED
	FallbackDefaultGateway string `yaml:"fallback_default_gateway" json:"fallback_default_gateway" netgroup:"gateway"`

	// RoutePriority. Priority 0 - means the first, primary interface, 1 would mean
	// the secondary, etc. Priority is used to decide which physical interface should
	// be the default route, if such interface has a default GW
	// valid values: 0-9 (MaxRoutePriority)
	RoutePriority int `yaml:"route_priority" json:"route_priority" netgroup:"route"`

	// An Aux interface, is not considered in the line up of outbound interfaces
	// It usually routes to an internal only network
	Aux bool `yaml:"aux" json:"aux"  netgroup:"mac"`

	// NOT IMPLEMENTED
	// Override DNS
	// Use only on a secondary, backup interface. If the interface becomes active
	// then DNS should switch to this server.
	NameserverOverrides string `yaml:"nameserver_overrides" json:"nameserver_overrides" netgroup:"nameserver"`

	// NOT IMPLEMENTED
	// routes this interfaces provides
	Routes []string `yaml:"routes" json:"routes" netgroup:"route"`

	// NOT IMPLEMENTED
	// A https test server to use for this interface, to make sure
	// the interface has a route out. If empty, ignored
	TestHttpsRouteOut string `yaml:"test_https_route_out" json:"test_https_route_out" netgroup:"http"`

	// NOT IMPLEMENTED
	// A host to ICMPv4 echo, to test that this interface has
	// a route to where it should. If empty, ignored
	TestICMPv4EchoOut string `yaml:"test_echo_route_out" json:"test_echo_route_out" netgroup:"ipv4"`

	// DHCP specific options:

	// Normally DHCP services will clear all addresses
	// on the given interface before setting the interface address
	// provided by the DHCP server. This disables that behavior,
	// meaning existing addresses will remain on the interface - if they were there
	// before Maestro started.
	DhcpDisableClearAddresses bool `yaml:"dhcp_disable_clear_addresses" json:"dhcp_disable_clear_addresses" netgroup:"dhcp"`

	// DhcpStepTimeout is the maximum amount of seconds to wait in each step
	// of getting a DHCP address. If not set, a sane default will be used.
	DhcpStepTimeout int `yaml:"dhcp_step_timeout" json:"dhcp_step_timeout"  netgroup:"dhcp"`

	// For use typically in the config file, not required. For incoming API calls,
	// the default behavior is "override", b/c the API calls always modify the
	// interface's database entry.
	// existing: "override"  # this will replace any data already set in the database
	// existing: ""          # this is the default. The database will take precedence, if it has an entry for the interface
	// existing: "replace"   # this will remove the existing datavase entry entirely. And then take whatever
	//                       # the config file has to offer
	Existing string `yaml:"existing" json:"existing" netgroup:"config_netif"`
}

// type Nameserver struct {
//     IpAddr string `yaml:"if_index" json:"if_index"`
// }

type NetworkConfigPayload struct {
	// If the flag is "disable : false" then maestro will do networking.  If the flag is "disable : true" then maestro will not do networking.
	Disable bool `yaml:"disable" json:"disable" netgroup:"config_network"`

	// an array of network interfaces to configure
	Interfaces []*NetIfConfigPayload `yaml:"interfaces" json:"interfaces" netgroup:"if"`

	// DontOverrideDefaultRoute when true means maestro will not
	// replace the default route, if one exists, with a setting from the interface
	// whether via DHCP or static. If a route is not in place, or the route was set by
	// maestro, it will replace this route.
	DontOverrideDefaultRoute bool `yaml:"dont_override_default_route" json:"dont_override_default_route" netgroup:"route"`

	Nameservers []string `yaml:"nameservers" json:"nameservers" netgroup:"nameserver"`

	// this tells network subsystem to ignore DHCP offers which have
	// DNS server settings. Whatever the DHCP server says in regards to
	// DNS will be ignored
	DnsIgnoreDhcp bool `yaml:"dns_ignore_dhcp" json:"dns_ignore_dhcp" netgroup:"dns"`

	// AltResolvConf, if populated with a string, will cause the network subsystem to
	// not write /etc/resolv.conf, and instead write what would go to /etc/resolv.conf to
	// an alternate file.
	AltResolvConf string `yaml:"alt_resolv_conf" json:"alt_resolve_conf" netgroup:"nameserver"`

	// if any of the below are true, then Nameserver shoul be set to
	// 127.0.0.1

	// DnsRunLocalCaching if true, a local caching nameserver will be used.
	DnsRunLocalCaching bool `yaml:"dns_local_caching" json:"dns_local_caching" netgroup:"dns"`

	// True if the gateway should forward all DNS requests to a
	// specific gateway only
	DnsForwardTo string `yaml:"default_gateway" json:"default_gateway" netgroup:"dns"`

	// True if you want the gateway to do lookup via DNS root servers
	// i.e. the gateway runs it's own recursive, lookup name server
	DnsRunRootLookup bool `yaml:"dns_root_lookup" json:"dns_root_lookup" netgroup:"dns"`

	// A string, which is what the content of the /etc/hosts file should
	// be
	DnsHostsData string `yaml:"dns_hosts_data" json:"dns_hosts_data" netgroup:"dns"`

	// Fallback Gateways. (NOT IMPLEMENTED)
	// These will be enabled if a primary outbound interface fails,
	// and a secondary outbound interface exists
	FallbackNameservers string `yaml:"fallback_nameservers" json:"fallback_nameservers" netgroup:"nameserver"`

	// For use typically in the config file, not required. For incoming API calls,
	// the default behavior is "override", b/c the API calls always modify the
	// interface's database entry.
	// existing: "override"  # this will replace any data already set in the database
	// existing: ""          # this is the default. The database will take precedence, if it has an entry for the interface
	// existing: "replace"   # this will remove the existing database entry entirely. And then take whatever
	//                       # the config file has to offer
	Existing string `yaml:"existing" json:"existing" netgroup:"config_network"`
}

const OP_TYPE_NET_INTERFACE = "net_interface"

// manipulate the IP v4/v6 address on an interface
const OP_ADD_ADDRESS = "add_addr"
const OP_REMOVE_ADDRESS = "rm_addr"
const OP_UPDATE_ADDRESS = "update_addr"

// take interface up / down
const OP_INTERFACE_STATE_CHANGE = "interface_state_change"

// something other than an interface change
const OP_TYPE_NET_CONFIG = "net_config"

// change pf DNS, default GW, or other other
const OP_UPDATE_CONFIG = "update_config"

// force the interface to release its DHCP address
const OP_RELEASE_DHCP = "release_dhcp"

// OP_RENEW_DHCP ask the interface to renew its address (or get a new address if it already released)
const OP_RENEW_DHCP = "renew_dhcp"

// NetInterfaceOperation is a "Operation"
type NetInterfaceOperation interface {
	GetType() string // operation type
	GetOp() string
	GetParams() []string
	GetTaskId() string

	GetIfConfig() *NetIfConfigPayload
}

type NetInterfaceOpPayload struct {
	Type   string   `yaml:"type" json:"type"` // should always be 'job'
	Op     string   `yaml:"op" json:"op"`
	Params []string `yaml:"params" json:"params"`
	TaskId string   `yaml:"task_id" json:"task_id"`

	IfConfig *NetIfConfigPayload `yaml:"ifconfig" json:"ifconfig"`
}

func NewNetInterfaceOpPayload() *NetInterfaceOpPayload {
	return &NetInterfaceOpPayload{Type: OP_TYPE_NET_INTERFACE}
}
func (this *NetInterfaceOpPayload) GetType() string {
	return this.Type
}
func (this *NetInterfaceOpPayload) GetOp() string {
	return this.Op
}
func (this *NetInterfaceOpPayload) GetParams() []string {
	return this.Params
}
func (this *NetInterfaceOpPayload) GetTaskId() string {
	return this.TaskId
}
func (this *NetInterfaceOpPayload) GetIfConfig() *NetIfConfigPayload {
	return this.IfConfig
}

// NetInterfaceOperation is a "Operation"
type NetConfigOperation interface {
	GetType() string // operation type
	GetOp() string
	GetParams() []string
	GetTaskId() string

	GetNetConfig() *NetworkConfigPayload
}

type NetConfigOpPayload struct {
	Type      string                `yaml:"type" json:"type"` // should always be 'job'
	Op        string                `yaml:"op" json:"op"`
	Params    []string              `yaml:"params" json:"params"`
	TaskId    string                `yaml:"task_id" json:"task_id"`
	NetConfig *NetworkConfigPayload `yaml:"ifconfig" json:"ifconfig"`
}

func NewNetConfigOpPayload() *NetConfigOpPayload {
	return &NetConfigOpPayload{Type: OP_TYPE_NET_INTERFACE}
}
func (this *NetConfigOpPayload) GetType() string {
	return this.Type
}
func (this *NetConfigOpPayload) GetOp() string {
	return this.Op
}
func (this *NetConfigOpPayload) GetParams() []string {
	return this.Params
}
func (this *NetConfigOpPayload) GetTaskId() string {
	return this.TaskId
}
func (this *NetConfigOpPayload) GetNetConfig() *NetworkConfigPayload {
	return this.NetConfig
}
