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

// NOTE: these should be defined in newer go, in
const (
	IFA_F_DADFAILED   = 0x8
	IFA_F_DEPRECATED  = 0x20
	IFA_F_HOMEADDRESS = 0x10
	IFA_F_NODAD       = 0x2
	IFA_F_OPTIMISTIC  = 0x4
	IFA_F_PERMANENT   = 0x80
	IFA_F_SECONDARY   = 0x1
	IFA_F_TEMPORARY   = 0x1
	IFA_F_TENTATIVE   = 0x40
	IFA_MAX           = 0x7
	IFF_ALLMULTI      = 0x200
	IFF_AUTOMEDIA     = 0x4000
	IFF_BROADCAST     = 0x2
	IFF_DEBUG         = 0x4
	IFF_DYNAMIC       = 0x8000
	IFF_LOOPBACK      = 0x8
	IFF_MASTER        = 0x400
	IFF_MULTICAST     = 0x1000
	IFF_NOARP         = 0x80
	IFF_NOTRAILERS    = 0x20
	IFF_NO_PI         = 0x1000
	IFF_ONE_QUEUE     = 0x2000
	IFF_POINTOPOINT   = 0x10
	IFF_PORTSEL       = 0x2000
	IFF_PROMISC       = 0x100
	IFF_RUNNING       = 0x40
	IFF_SLAVE         = 0x800
	IFF_TAP           = 0x2
	IFF_TUN           = 0x1
	IFF_TUN_EXCL      = 0x8000
	IFF_UP            = 0x1
	IFF_VNET_HDR      = 0x4000

	dnsTemplateFile = `#   this is a dynamically generated resolv.conf generated by maestro
#     ** making changes to this file may fail - as maestro will rewrite this file on lease renewals or config changes **
#     generated on: %s (%d)
`
	resolvConfPath = "/etc/resolv.conf"
	// rw-r--r--
	dnsFilePerms = 0644
	// the default local domain provided to /etc/resolv.conf
	defaultLocalDomain = "localdomain"

	defaultDhcpStepTimeout = 15 // 15 seconds
)
