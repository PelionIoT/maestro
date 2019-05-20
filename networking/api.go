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
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/armPelionEdge/dhcp4"
	"github.com/armPelionEdge/dhcp4client"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/debugging"
	"github.com/armPelionEdge/maestroSpecs"
	"github.com/armPelionEdge/netlink"
	"golang.org/x/sys/unix"
)

const (
	ERROR_NO_IF                   = 0x1
	ERROR_INVALID_SETTINGS        = 0x2
	ERROR_SOCKET_ERROR            = 0x3
	ERROR_NETWORK_TIMEOUT         = 0x4
	ERROR_GENERAL_INTERFACE_ERROR = 0x5
	ERROR_NO_RESPONSE_TIMEOUT     = 0x7
	ERROR_DHCP_INVALID_LEASE      = 0xF01
)

type DhcpLeaseInfo struct {
	CurrentIP     net.IP // current IP address
	CurrentMask   net.IPMask
	LeaseAcquired int64        // When we got the last packet giving us an IP - Unix epoch time
	LastAckPacket dhcp4.Packet // this is just a []byte, interpreted with dhcp package

	// if set, this is the default route which is currently setup
	defaultRoute *netlink.Route
	// true if the lease has valid DNS entries
	// hasDNS bool
	// these either come from DHCP options, or we use intelligent values for them
	// based off the lease time.
	// https://serverfault.com/questions/348908/whats-the-difference-between-rebound-and-renew-state-on-dhcp-client
	// To manage the lease extension process, two timers are set at the time that a lease is
	// allocated. The renewal timer (T1) goes off to tell the client it is time to try to renew
	// the lease with the server that initially granted it. The rebinding timer (T2) goes off
	// if the client is not successful in renewing with that server, and tells it to try any
	// server to have the lease extended. If the lease is renewed or rebound, the client
	// goes back to normal operation. If it cannot be rebound, it will expire and the client will
	// need to seek a new lease.
	renewalTime int64 // based on "T1" Option 58 - actual Unix epoch time
	rebindTime  int64 // based on "T2" Option  59 - actual Unix epoch time
	// not stored in DB
	parsedOptions dhcp4.Options // this is built from the Packet above. We just cache it after parsing
}

func nmLogErrorf(format string, a ...interface{}) {
	log.MaestroErrorf("NetworkManager: "+format, a...)
}

func nmLogWarnf(format string, a ...interface{}) {
	log.MaestroWarnf("NetworkManager: "+format, a...)
}

func nmLogSuccessf(format string, a ...interface{}) {
	log.MaestroSuccessf("NetworkManager: "+format, a...)
}

func nmLogDebugf(format string, a ...interface{}) {
	log.MaestroDebugf("NetworkManager: "+format, a...)
}

// should be called when a lease is restored from the
// database
func (this *DhcpLeaseInfo) deserialize() {
	this.parsedOptions = this.LastAckPacket.ParseOptions()
	this.parseRenewalTimes()
}

// Assumes options are parsed and LeaseAcquired is valid time in the past. We can call this on DB load, or right after getting a lease.
// Basically we use what we can work with. If a server does not send us all expected
// lease time information, we try to make use of what we have.
// Most of the time Option 51 is sent. Often 58 and 59 or not.
func (this *DhcpLeaseInfo) parseRenewalTimes() (renewnow bool) {
	renewnow = false
	var leasetimesec int64
	var t1 int64
	var t2 int64
	delta := time.Now().Unix() - this.LeaseAcquired // LeaseAcquired is in Unix Epoch seconds
	if delta < 0 {
		log.MaestroErrorf("DHCP: corruption in DB record or other error. Lease time info invalid.")
		renewnow = true
		return
	}
	if len(this.parsedOptions[dhcp4.OptionIPAddressLeaseTime]) > 0 {
		// do we have Option 51, this is the IP Address Lease time?
		origleaselength := binary.BigEndian.Uint32(this.parsedOptions[dhcp4.OptionIPAddressLeaseTime])
		log.MaestroDebugf("NetworkManager: DHCP: lease had lease time of %d seconds. Lease acquired %d", origleaselength, this.LeaseAcquired)
		leasetimesec = int64(origleaselength) - delta
		if leasetimesec < 1 {
			log.MaestroDebugf("NetworkManager: DHCP: lease is over. Need to renew.")
			// We are past the life of the lease
			renewnow = true
			return
		}
	} else {
		log.MaestroWarnf("NetworkManager: DHCP: Unusual, We did not get an Option 51 lease time.")
	}
	slice, ok := this.parsedOptions[dhcp4.OptionRenewalTimeValue]
	if ok {
		t1 = int64(binary.BigEndian.Uint32(slice)) - delta
		if t1 < 0 {
			renewnow = true
			return
		}
		if t1 > leasetimesec && leasetimesec > 0 {
			log.MaestroWarnf("NetworkManager: DHCP: Unusual, received T1 (option 58) time which is bigger than lease time. Using lease time.")
			this.renewalTime = this.LeaseAcquired + int64(leasetimesec) - 300 // 5 minutes before end
		} else {
			this.renewalTime = this.LeaseAcquired + int64(t1)
		}
	} else {
		this.renewalTime = 0
	}
	slice, ok = this.parsedOptions[dhcp4.OptionRebindingTimeValue]
	if ok {
		t2 = int64(binary.BigEndian.Uint32(slice)) - delta
		if t2 < 0 {
			renewnow = true
			return
		}
		if t2 > leasetimesec && leasetimesec > 0 {
			log.MaestroWarnf("NetworkManager: DHCP: Unusual, received T2 (option 58) time which is bigger than lease time. Using lease time.")
			this.rebindTime = this.LeaseAcquired + int64(leasetimesec) - 150 // 2.5 minutes before end
		} else {
			this.rebindTime = this.LeaseAcquired + int64(t2)
		}
	} else {
		this.rebindTime = 0
	}
	if this.rebindTime == 0 {
		this.rebindTime = this.renewalTime
	}
	if this.renewalTime == 0 {
		this.renewalTime = leasetimesec
	}
	if this.renewalTime == 0 {
		log.MaestroWarnf("NetworkManager: DHCP: Could not determin a lease time. Need new lease.")
		renewnow = true
	}
	return
}

// should be called after parsedOptions is filled
func (this *DhcpLeaseInfo) GetLeaseSubnetMask() net.IPMask {
	slice, ok := this.parsedOptions[dhcp4.OptionSubnetMask]
	if !ok {
		log.MaestroWarnf("NetworkManager: DHCP: No subnet mask was provided, using a default.")
		return this.CurrentIP.DefaultMask()
	} else {
		return net.IPv4Mask(slice[0], slice[1], slice[2], slice[3])
	}
}

// should be called after parsedOptions is filled
func (this *DhcpLeaseInfo) GetRouter() (ok bool, ret net.IP) {
	slice, ok := this.parsedOptions[dhcp4.OptionRouter]
	if !ok {
		log.MaestroWarnf("NetworkManager: DHCP: No router IP was provided from DHCP server.")
	} else {
		ret = net.IP(slice)
	}
	return
}

func (this *DhcpLeaseInfo) GetDNS() (ok bool, ret []net.IP) {
	slice, ok := this.parsedOptions[dhcp4.OptionDomainNameServer]
	if !ok {
		log.MaestroWarnf("NetworkManager: DHCP: No DNS was provided. Will use global settings or fallbacks")
	} else {
		l := len(slice)
		// ok, so each IPv4 address is 4 bytes.
		n := l / 4
		if l%4 > 0 {
			log.MaestroErrorf("DHCP: Option %d - DNS - field length was not a multiple of 4. Weird", dhcp4.OptionDomainNameServer)
		}
		if n > 0 {
			ret = make([]net.IP, n)
			for i := 0; i < n; i++ {
				ret[i] = net.IP(slice[i*4 : i*4+4])
			}
		}
	}
	return
}

func (this *DhcpLeaseInfo) GetLocalDomain() (ok bool, localdomain string) {
	slice, ok := this.parsedOptions[dhcp4.OptionDomainName]
	if ok {
		localdomain = string(slice)
	}
	return
}

// should be called after parsedOptions is filled
func (this *DhcpLeaseInfo) GetClientDomainName() (ok bool, ret []byte) {
	ret, ok = this.parsedOptions[dhcp4.OptionDomainName]
	return
}

// should be called after parsedOptions is filled
func (this *DhcpLeaseInfo) GetDHCPServer() (ok bool, ret net.IP) {
	slice, ok := this.parsedOptions[dhcp4.OptionServerIdentifier]
	if !ok {
		return
	} else {
		ret = net.IP(slice)
		return
	}
}

func (this *DhcpLeaseInfo) GetSIADDR() (ok bool, ret net.IP) {
	if len(this.LastAckPacket) > 0 {
		ret = this.LastAckPacket.SIAddr()
		ok = true
	}
	return
}

func (this *DhcpLeaseInfo) GetGIADDR() (ok bool, ret net.IP) {
	if len(this.LastAckPacket) > 0 {
		ret = this.LastAckPacket.GIAddr()
		ok = true
	}
	return
}

func (this *DhcpLeaseInfo) IsValid() bool {
	if len(this.CurrentIP) >= net.IPv4len && len(this.CurrentMask) >= net.IPv4len {
		return true
	} else {
		return false
	}
}

func (this *DhcpLeaseInfo) IsExpired() bool {
	var remainrenewal int64
	now := time.Now().Unix()

	remainrenewal = this.renewalTime - now
	if remainrenewal < 30 {
		remainrenewal = this.rebindTime - now
	}
	if remainrenewal >= 30 {
		return true;
	}

	return false
}

func (this *DhcpLeaseInfo) RemainOnLease() int64 {
	return 0
}

type NetworkAPIError struct {
	Errstring  string `json:"error"`
	Code       int    `json:"code"`
	IfName     string `json:"if_name"`
	IfIndex    int    `json:"if_index"`
	Underlying error  `json:"underlying"`
}

func (this *NetworkAPIError) Error() string {
	return fmt.Sprintf("%s code:%d", this.Errstring, this.Code)
}

type InterfaceStatus struct {
	// nil error means interface setup ok
	Err     error  `json:"err"`
	Up      bool   `json:"up"`
	IfName  string `json:"if_name"`
	IfIndex int    `json:"if_index"`
	IPV4    string `json:"ipv4"`
	IPV6    string `json:"ipv6"` // not implemented TODO
	ipv4    net.IP
	ipv6    net.IP // not implemented TODO
}

func GetInterfaceMacAddress(ifname string) (macaddr net.HardwareAddr, ifindex int, err error) {
	linklist, err := netlink.LinkList()
	if err == nil {
		for _, link := range linklist {
			if link.Attrs().Name == ifname {
				macaddr = link.Attrs().HardwareAddr
				ifindex = link.Attrs().Index
			}
		}
		if len(macaddr) < 1 {
			iferr := &NetworkAPIError{
				Code:      ERROR_NO_IF,
				Errstring: fmt.Sprintf("Can't find interface of Name %s", ifname),
				IfName:    ifname,
			}
			err = iferr
		}
	}
	return
}

// SetInterfaceMacAddress sets the mac address. ifname is preferred, if not provided (empty string) then ifindex is used
// MAC addresses should be formatted "aa:bb:cc:01:23:45" etc.
// 48 or 64 bit addresses can be stated, or whatever may be supported by the hardware
// such as a 20-octet IP for InifiniBand
func SetInterfaceMacAddress(ifname string, ifindex int, macaddr string) (err error) {
	var link netlink.Link
	var linklist []netlink.Link
	var mac net.HardwareAddr
	// ok - parse mac
	mac, err = net.ParseMAC(macaddr)
	if err == nil {
		linklist, err = netlink.LinkList()
		if err == nil {
			// first, find the link we are talking about
			// use name if one provided, otherwise use index
			if len(ifname) > 0 {
				for _, alink := range linklist {
					if alink.Attrs().Name == ifname {
						link = alink
						break
					}
				}
			} else {
				for _, alink := range linklist {
					if alink.Attrs().Index == ifindex {
						link = alink
						break
					}
				}
			}
			if link != nil {
				err = netlink.LinkSetHardwareAddr(link, mac)
			} else {
				iferr := &NetworkAPIError{
					Code:      ERROR_NO_IF,
					Errstring: fmt.Sprintf("Can't find interface of Index %d / %s", ifindex, ifname),
					IfName:    ifname,
				}
				err = iferr
			}
		}
	}
	return
}

// ReleaseFromServer will release the DHCP lease to the server for the given interface
// NOT IMPLEMENTED
func ReleaseFromServer(ifname string, leasinfo *DhcpLeaseInfo) (ok bool, err error) {
	return
}

/**
 * RenewFromServer renews from the server directy - a packet sent direct to the server, vs. a broadcast
 * @param {[type]} ifname   string                           [description]
 * @param {[type]} leasinfo *DhcpLeaseInfo                   [description]
 * @param {[type]} opts     *dhcp4client.DhcpRequestOptions) (outleaseinfo *DhcpLeaseInfo, err error [description]
 */
func RenewFromServer(ifname string, leaseinfo *DhcpLeaseInfo, opts *dhcp4client.DhcpRequestOptions) (ackval int, outleaseinfo *DhcpLeaseInfo, err error) {
	hwaddr, ifindex, err := GetInterfaceMacAddress(ifname)
	var success bool
	var acknowledgementpacket dhcp4.Packet
	log.MaestroDebugf("NetworkManager: DHCP using interface index (renew) %d / %s has MAC %s", ifindex, ifname, hwaddr.String())
	if err != nil {
		return
	}
	// renew must have some kind of lease to look at
	if leaseinfo != nil {
		// totally new lease
		outleaseinfo = leaseinfo
		// var acknowledgementpacket dhcp4.Packet

		// c, err2 := dhcp4client.NewPacketSock(ifindex) //,dhcp4client.SetLocalAddr(net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 1068}), dhcp4client.SetRemoteAddr(net.UDPAddr{IP: net.IPv4bcast, Port: 1067}))
		// if err2 != nil {
		//     iferr := &NetworkAPIError{
		//         Code: ERROR_SOCKET_ERROR,
		//         Errstring: fmt.Sprintf("Can't make AF_PACKET socket. If: %s",ifname),
		//         IfName: ifname,
		//         Underlying: err2,
		//     }
		//     err = iferr
		//     return
		//     // test.Error("Client Connection Generation:" + err.Error())
		// }
		// defer c.Close()

		// DHCP server should have been provided in last acknowledgement packet
		serveraddr := leaseinfo.LastAckPacket.SIAddr()

		conn, err2 := dhcp4client.NewInetSock(dhcp4client.SetLocalAddr(net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 68}),
			dhcp4client.SetRemoteAddr(net.UDPAddr{IP: serveraddr, Port: 67}))

		if err2 != nil {
			log.MaestroErrorf("NetworkManager: DHCP Client Connection Generation - renew to server %s:%s", serveraddr.String(), err2.Error())
			iferr := &NetworkAPIError{
				Code:       ERROR_SOCKET_ERROR,
				Errstring:  fmt.Sprintf("Can't make Inet socket. If: %s", ifname),
				IfName:     ifname,
				Underlying: err2,
			}
			err = iferr
			return
			// test.Error("Client Connection Generation:" + err.Error())
		}

		// only use the interface we need an address on
		err2 = conn.BindToInterface(ifname)

		if err2 != nil {
			log.MaestroErrorf("NetworkManager: DHCP client - inetsock - failed to bindToInterface() [ignoring failure] %s", err2.Error())
		}

		defer conn.Close()

		dhcpclient, err2 := dhcp4client.New(dhcp4client.HardwareAddr(hwaddr), dhcp4client.Connection(conn))
		if err2 != nil {
			log.MaestroErrorf("DHCP client creation error: %v\n", err)
			err = err2
			return
		}
		defer dhcpclient.Close()

		//        success, acknowledgementpacket, err2 = dhcpclient.Request(opts)

		log.MaestroInfof("NetworkManager: DHCP Attempting to renew lease with server %s", serveraddr.String())
		ackval, acknowledgementpacket, err = dhcpclient.Renew(leaseinfo.LastAckPacket, opts)
		if err != nil {
			networkError, ok := err.(*net.OpError)
			if ok && networkError.Timeout() {
				log.MaestroErrorf("NetworkManager: Renewal Failed! Because it didn't get a response from DHCP server %s", serveraddr.String())
				iferr := &NetworkAPIError{
					Code:       ERROR_NO_RESPONSE_TIMEOUT,
					Errstring:  fmt.Sprintf("Error on DHCP socket request (timeout): %s --> %s", ifname, err.Error()),
					IfName:     ifname,
					Underlying: err,
				}
				err = iferr
				log.MaestroErrorf("NetworkManager: Error" + err.Error())
			} else {
				iferr := &NetworkAPIError{
					Code:       ERROR_SOCKET_ERROR,
					Errstring:  fmt.Sprintf("Error on DHCP socket request: %s --> %s", ifname, err.Error()),
					IfName:     ifname,
					Underlying: err,
				}
				err = iferr
			}
			return
		}

		debugging.DEBUG_OUT("DHCP (renew with server %s) Success:%v\n", serveraddr.String(), success)
		debugging.DEBUG_OUT("DHCP (renew with server %s) Packet:%v\n", serveraddr.String(), acknowledgementpacket)

	} else {
		debugging.DEBUG_OUT("RenewFromServer wasn't handed a lease.\n")
		iferr := &NetworkAPIError{
			Code:      ERROR_DHCP_INVALID_LEASE,
			Errstring: fmt.Sprintf("Error on DHCP socket renew request: %s - no lease", ifname),
			IfName:    ifname,
		}
		err = iferr
		return
	}

	if ackval == dhcp4client.Success {
		outleaseinfo.LastAckPacket = acknowledgementpacket
		outleaseinfo.parsedOptions = acknowledgementpacket.ParseOptions()
		outleaseinfo.LeaseAcquired = time.Now().Unix()
		outleaseinfo.CurrentIP = acknowledgementpacket.YIAddr()
		outleaseinfo.CurrentMask = outleaseinfo.GetLeaseSubnetMask()
		if outleaseinfo.parseRenewalTimes() {
			log.MaestroErrorf("NetworkManager: DHCP got extremely short or invalid lease time!")
			iferr := &NetworkAPIError{
				Code:      ERROR_DHCP_INVALID_LEASE,
				Errstring: fmt.Sprintf("DHCP lease too short or invalid: %s", ifname),
				IfName:    ifname,
			}
			err = iferr
		}
		//     test.Error("We didn't sucessfully get a DHCP Lease?")
		// } else {
		debugging.DEBUG_OUT("DHCP IP Received:%v - renewal time: %d rebind time: %d\n", acknowledgementpacket.YIAddr().String(), outleaseinfo.renewalTime, outleaseinfo.rebindTime)
		ok, ip := outleaseinfo.GetRouter()
		if ok {
			debugging.DEBUG_OUT("DHCP router is %s\n", ip.String())
		}
		ok, ips := outleaseinfo.GetDNS()
		if ok {
			for _, ip := range ips {
				debugging.DEBUG_OUT("DHCP: DNS nameserver %s\n", ip.String())
			}
		}
	}

	return

}

// InitRebootDhcpLease asks the DHCP server to confirm it's IP and provide new options. This is
// is used during the INIT_REBOOT state in the DHCP RFC 2131. It's typically called if the network
// is disconnected, then reconnected.
func InitRebootDhcpLease(ifname string, currentIP net.IP, opts *dhcp4client.DhcpRequestOptions) (success int, outleaseinfo *DhcpLeaseInfo, err error) {
	hwaddr, ifindex, err := GetInterfaceMacAddress(ifname)
	log.MaestroDebugf("NetworkManager: DHCP (InitRebootDhcpLease) using interface index %d / %s has MAC %s", ifindex, ifname, hwaddr.String())
	if err != nil {
		return
	}
	now := time.Now().Unix()
	outleaseinfo = new(DhcpLeaseInfo)

	var acknowledgementpacket dhcp4.Packet

	// make sure the interface is up - otherwise DHCP can't work
	link, err := GetInterfaceLink(ifname, ifindex)
	if err == nil {
		err = netlink.LinkSetUp(link)
		if err != nil {
			log.MaestroErrorf("NetworkManager: DHCP - could not bring interface up: %s", err.Error())
		}
	} else {
		log.MaestroErrorf("NetworkManager: DHCP - failure at GetInterfaceLink - unusual. %s", err.Error())
		return
	}

	c, err2 := dhcp4client.NewPacketSock(ifindex) //,dhcp4client.SetLocalAddr(net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 1068}), dhcp4client.SetRemoteAddr(net.UDPAddr{IP: net.IPv4bcast, Port: 1067}))
	if err2 != nil {
		iferr := &NetworkAPIError{
			Code:       ERROR_SOCKET_ERROR,
			Errstring:  fmt.Sprintf("Can't make AF_PACKET socket. If: %s", ifname),
			IfName:     ifname,
			Underlying: err2,
		}
		err = iferr
		return
		// test.Error("Client Connection Generation:" + err.Error())
	}
	defer c.Close()

	dhcpclient, err2 := dhcp4client.New(dhcp4client.HardwareAddr(hwaddr), dhcp4client.Connection(c), dhcp4client.AuxOpts(opts))
	if err2 != nil {
		log.MaestroErrorf("NetworkManager: DHCP client creation error: %v\n", err)
		err = err2
		return
	}
	defer dhcpclient.Close()

	success, acknowledgementpacket, err = dhcpclient.InitRebootRequest(currentIP, opts)
	if err != nil {
		networkError, ok := err.(*net.OpError)
		if ok && networkError.Timeout() {
			log.MaestroErrorf("NetworkManager: Renewal Failed! Because it didn't find the DHCP server very Strange")
			log.MaestroErrorf("NetworkManager: Error" + err.Error())
		} else {
			log.MaestroErrorf("NetworkManager: DHCP Renewal Failed. Details: %+v", err)
		}
		// test.Fatalf("Error:%v\n", err)
	}

	if success == dhcp4client.Success {
		outleaseinfo.LastAckPacket = acknowledgementpacket
		outleaseinfo.parsedOptions = acknowledgementpacket.ParseOptions()
		outleaseinfo.LeaseAcquired = now
		outleaseinfo.CurrentIP = acknowledgementpacket.YIAddr()
		outleaseinfo.CurrentMask = outleaseinfo.GetLeaseSubnetMask()
		if outleaseinfo.parseRenewalTimes() {
			log.MaestroErrorf("NetworkManager: DHCP got extremely short or invalid lease time!")
			iferr := &NetworkAPIError{
				Code:      ERROR_DHCP_INVALID_LEASE,
				Errstring: fmt.Sprintf("DHCP lease too short or invalid: %s", ifname),
				IfName:    ifname,
			}
			err = iferr
		}
		//     test.Error("We didn't sucessfully get a DHCP Lease?")
		// } else {
		debugging.DEBUG_OUT("DHCP IP Received:%v - renewal time: %d rebind time: %d\n", acknowledgementpacket.YIAddr().String(), outleaseinfo.renewalTime, outleaseinfo.rebindTime)
		ok, ip := outleaseinfo.GetRouter()
		if ok {
			debugging.DEBUG_OUT("DHCP router is %s\n", ip.String())
		}
		ok, ips := outleaseinfo.GetDNS()
		if ok {
			for _, ip := range ips {
				debugging.DEBUG_OUT("DHCP: DNS nameserver %s\n", ip.String())
			}
		}
	}

	return
}

// GetFreshDhcpLease requests a new lease, without using any existing lease information.
func GetFreshDhcpLease(ifname string, opts *dhcp4client.DhcpRequestOptions) (success int, outleaseinfo *DhcpLeaseInfo, err error) {
	hwaddr, ifindex, err := GetInterfaceMacAddress(ifname)
	log.MaestroDebugf("NetworkManager: DHCP (GetFreshLease) using interface index %d / %s has MAC %s", ifindex, ifname, hwaddr.String())
	if err != nil {
		return
	}
	now := time.Now().Unix()
	outleaseinfo = new(DhcpLeaseInfo)

	var acknowledgementpacket dhcp4.Packet

	// make sure the interface is up - otherwise DHCP can't work
	link, err := GetInterfaceLink(ifname, ifindex)
	if err == nil {
		err = netlink.LinkSetUp(link)
		if err != nil {
			log.MaestroErrorf("NetworkManager: DHCP - could not bring interface up: %s", err.Error())
		}
	} else {
		log.MaestroErrorf("NetworkManager: DHCP - failure at GetInterfaceLink - unusual. %s", err.Error())
		return
	}

	c, err2 := dhcp4client.NewPacketSock(ifindex) //,dhcp4client.SetLocalAddr(net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 1068}), dhcp4client.SetRemoteAddr(net.UDPAddr{IP: net.IPv4bcast, Port: 1067}))
	if err2 != nil {
		iferr := &NetworkAPIError{
			Code:       ERROR_SOCKET_ERROR,
			Errstring:  fmt.Sprintf("Can't make AF_PACKET socket. If: %s", ifname),
			IfName:     ifname,
			Underlying: err2,
		}
		err = iferr
		return
		// test.Error("Client Connection Generation:" + err.Error())
	}
	defer c.Close()

	dhcpclient, err2 := dhcp4client.New(dhcp4client.HardwareAddr(hwaddr), dhcp4client.Connection(c), dhcp4client.AuxOpts(opts))
	if err2 != nil {
		log.MaestroErrorf("NetworkManager: DHCP client creation error: %v\n", err)
		err = err2
		return
	}
	defer dhcpclient.Close()

	success, acknowledgementpacket, err2 = dhcpclient.Request(opts)

	debugging.DEBUG_OUT("DHCP Success:%v\n", success)
	debugging.DEBUG_OUT("DHCP Packet:%v\n", acknowledgementpacket)

	if err2 != nil {
		networkError, ok := err2.(*net.OpError)
		if ok && networkError.Timeout() {
			iferr := &NetworkAPIError{
				Code:       ERROR_NETWORK_TIMEOUT,
				Errstring:  fmt.Sprintf("No response from DHCP server. If: %s", ifname),
				IfName:     ifname,
				Underlying: networkError,
			}
			err = iferr
			return

			// test.Log("Test Skipping as it didn't find a DHCP Server")
			// test.SkipNow()
		} else {
			iferr := &NetworkAPIError{
				Code:       ERROR_SOCKET_ERROR,
				Errstring:  fmt.Sprintf("Error on DHCP socket request: %s --> %s", ifname, err2.Error()),
				IfName:     ifname,
				Underlying: err2,
			}
			err = iferr
			return
		}
	}

	if success == dhcp4client.Success {
		outleaseinfo.LastAckPacket = acknowledgementpacket
		outleaseinfo.parsedOptions = acknowledgementpacket.ParseOptions()
		outleaseinfo.LeaseAcquired = now
		outleaseinfo.CurrentIP = acknowledgementpacket.YIAddr()
		outleaseinfo.CurrentMask = outleaseinfo.GetLeaseSubnetMask()
		if outleaseinfo.parseRenewalTimes() {
			log.MaestroErrorf("NetworkManager: DHCP got extremely short or invalid lease time!")
			iferr := &NetworkAPIError{
				Code:      ERROR_DHCP_INVALID_LEASE,
				Errstring: fmt.Sprintf("DHCP lease too short or invalid: %s", ifname),
				IfName:    ifname,
			}
			err = iferr
		}
		//     test.Error("We didn't sucessfully get a DHCP Lease?")
		// } else {
		debugging.DEBUG_OUT("DHCP IP Received:%v - renewal time: %d rebind time: %d\n", acknowledgementpacket.YIAddr().String(), outleaseinfo.renewalTime, outleaseinfo.rebindTime)
		ok, ip := outleaseinfo.GetRouter()
		if ok {
			debugging.DEBUG_OUT("DHCP router is %s\n", ip.String())
		}
		ok, ips := outleaseinfo.GetDNS()
		if ok {
			for _, ip := range ips {
				debugging.DEBUG_OUT("DHCP: DNS nameserver %s\n", ip.String())
			}
		}
	}

	return
}

// RequestOrRenewDhcpLease Gets a new DCHP address, based on an existing lease, if one exists. If one does
// not exist, it just gets a new address
// @param {[type]} ifname string [description]
func RequestOrRenewDhcpLease(ifname string, leaseinfo *DhcpLeaseInfo, opts *dhcp4client.DhcpRequestOptions) (success int, outleaseinfo *DhcpLeaseInfo, err error) {
	hwaddr, ifindex, err := GetInterfaceMacAddress(ifname)
	log.MaestroDebugf("NetworkManager: DHCP (RequestOrRenew) using interface index %d / %s has MAC %s", ifindex, ifname, hwaddr.String())
	if err != nil {
		return
	}

	if leaseinfo == nil {
		outleaseinfo = new(DhcpLeaseInfo)
	} else {
		outleaseinfo = leaseinfo
	}

	var acknowledgementpacket dhcp4.Packet
	now := time.Now().Unix()
	// make sure the interface is up - otherwise DHCP can't work
	link, err := GetInterfaceLink(ifname, ifindex)
	if err == nil {
		err = netlink.LinkSetUp(link)
		if err != nil {
			log.MaestroErrorf("NetworkManager: DHCP - could not bring interface up: %s", err.Error())
		}
	} else {
		log.MaestroErrorf("NetworkManager: DHCP - failure at GetInterfaceLink - unusual. %s", err.Error())
		return
	}

	c, err2 := dhcp4client.NewPacketSock(ifindex) //,dhcp4client.SetLocalAddr(net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 1068}), dhcp4client.SetRemoteAddr(net.UDPAddr{IP: net.IPv4bcast, Port: 1067}))
	if err2 != nil {
		iferr := &NetworkAPIError{
			Code:       ERROR_SOCKET_ERROR,
			Errstring:  fmt.Sprintf("Can't make AF_PACKET socket. If: %s", ifname),
			IfName:     ifname,
			Underlying: err2,
		}
		err = iferr
		return
		// test.Error("Client Connection Generation:" + err.Error())
	}
	defer c.Close()

	dhcpclient, err2 := dhcp4client.New(dhcp4client.HardwareAddr(hwaddr), dhcp4client.Connection(c), dhcp4client.AuxOpts(opts))
	if err2 != nil {
		log.MaestroErrorf("NetworkManager: DHCP client creation error: %v\n", err)
		err = err2
		return
	}
	defer dhcpclient.Close()

	if leaseinfo == nil {
		debugging.DEBUG_OUT("leaseinfo is nil!\n")
	}
	debugging.DEBUG_OUT("leaseinfo %+v\n", leaseinfo)

	//If the lease is not valid or expired, get a new lease
	if leaseinfo == nil || !leaseinfo.IsValid() || !leaseinfo.IsExpired() {
		// totally new lease
		success, acknowledgementpacket, err2 = dhcpclient.Request(opts)

		debugging.DEBUG_OUT("DHCP Success:%v\n", success)
		debugging.DEBUG_OUT("DHCP Packet:%v\n", acknowledgementpacket)

		if err2 != nil {
			networkError, ok := err2.(*net.OpError)
			if ok && networkError.Timeout() {
				iferr := &NetworkAPIError{
					Code:       ERROR_NETWORK_TIMEOUT,
					Errstring:  fmt.Sprintf("No response from DHCP server. If: %s", ifname),
					IfName:     ifname,
					Underlying: networkError,
				}
				err = iferr
				return

				// test.Log("Test Skipping as it didn't find a DHCP Server")
				// test.SkipNow()
			} else {
				iferr := &NetworkAPIError{
					Code:       ERROR_SOCKET_ERROR,
					Errstring:  fmt.Sprintf("Error on DHCP socket request: %s --> %s", ifname, err2.Error()),
					IfName:     ifname,
					Underlying: err2,
				}
				err = iferr
				return
			}
		}
	} else {

		log.MaestroInfof("NetworkManager: DHCP Renewing Lease (%s / %d)", ifname, ifindex)
		success, acknowledgementpacket, err = dhcpclient.Renew(leaseinfo.LastAckPacket, opts)
		if err != nil {
			networkError, ok := err.(*net.OpError)
			if ok && networkError.Timeout() {
				log.MaestroErrorf("NetworkManager: Renewal Failed! Because it didn't find the DHCP server very Strange")
				log.MaestroErrorf("NetworkManager: Error" + err.Error())
			} else {
				log.MaestroErrorf("NetworkManager: DHCP Renewal Failed. Details: %+v", err)
			}
			// test.Fatalf("Error:%v\n", err)
		}
	}

	// if !success {
	//     log.MaestroErrorf("We didn't sucessfully Renew a DHCP Lease?")
	// } else {
	//     log.MaestroSuccessf("IP Received:%v\n", acknowledgementpacket.YIAddr().String())
	// }

	if success == dhcp4client.Success {
		outleaseinfo.LastAckPacket = acknowledgementpacket
		outleaseinfo.parsedOptions = acknowledgementpacket.ParseOptions()
		outleaseinfo.LeaseAcquired = now
		outleaseinfo.CurrentIP = acknowledgementpacket.YIAddr()
		outleaseinfo.CurrentMask = outleaseinfo.GetLeaseSubnetMask()
		if outleaseinfo.parseRenewalTimes() {
			log.MaestroErrorf("NetworkManager: DHCP got extremely short or invalid lease time!")
			iferr := &NetworkAPIError{
				Code:      ERROR_DHCP_INVALID_LEASE,
				Errstring: fmt.Sprintf("DHCP lease too short or invalid: %s", ifname),
				IfName:    ifname,
			}
			err = iferr
		}
		//     test.Error("We didn't sucessfully get a DHCP Lease?")
		// } else {
		debugging.DEBUG_OUT("DHCP IP Received:%v - renewal time: %d rebind time: %d\n", acknowledgementpacket.YIAddr().String(), outleaseinfo.renewalTime, outleaseinfo.rebindTime)
		ok, ip := outleaseinfo.GetRouter()
		if ok {
			debugging.DEBUG_OUT("DHCP router is %s\n", ip.String())
		}
		ok, ips := outleaseinfo.GetDNS()
		if ok {
			for _, ip := range ips {
				debugging.DEBUG_OUT("DHCP: DNS nameserver %s\n", ip.String())
			}
		}
	}

	return
}

func GetLinkStatusByName(name string) (ret *netlink.LinkAttrs, err error) {
	linklist, err := netlink.LinkList()

	if err == nil {
		for _, link := range linklist {
			if link.Attrs().Name == name {
				ret = link.Attrs()
				break
			}
		}
		if ret == nil {
			iferr := &NetworkAPIError{
				Code:      ERROR_NO_IF,
				Errstring: fmt.Sprintf("Can't find interface of Index %s", name),
				IfName:    name,
			}
			err = iferr
		}
	}
	return
}

// Clear all address on an interface, with exception of exception list
func ClearAllAddressesByLinkName(name string, exceptions []net.IP) (err error) {

	// linklist, err := netlink.LinkList()
	// if err == nil {
	//     for _, link := range linklist {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return
	}
	debugging.DEBUG_OUT("ClearAllAddressesByLinkName link: %+v\n", link)
	if link.Attrs().Name == name {
		addrlist, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err == nil {
		AddressLoop:
			for _, addr := range addrlist {
				if exceptions != nil {
					for _, exception := range exceptions {
						// https://golang.org/ref/spec#Struct_types
						if bytes.Compare(exception, addr.IP) == 0 {
							continue AddressLoop
						}
					}
					netlink.AddrDel(link, &addr)
				} else {
					debugging.DEBUG_OUT("del addr: %+v\n", addr)
					netlink.AddrDel(link, &addr)
				}
			}
		}

	}

	// break
	// }
	// }
	return
	//    netlink.
}

func IsIPv4AddressSet(ifname string, thisaddr *netlink.Addr) (response bool, err error) {
	response = false
	var linklist []netlink.Link
	linklist, err = netlink.LinkList()
	if err == nil && thisaddr != nil {
		for _, link := range linklist {
			if link.Attrs().Name == ifname {
				var addrlist []netlink.Addr
				addrlist, err = netlink.AddrList(link, netlink.FAMILY_V4)
				if err == nil {
					for _, addr := range addrlist {
						debugging.DEBUG_OUT("Check %+v %+v\n", addr, *thisaddr)
						if addr.Equal(*thisaddr) {
							response = true
							return
						}
						// if exceptions != nil {
						//     for _, exception := range exceptions {
						//         // https://golang.org/ref/spec#Struct_types
						//         if bytes.Compare(exception, addr.IP) == 0 {
						//             continue AddressLoop
						//         }
						//     }
						//     netlink.AddrDel(link,&addr)
						// } else {
						//     debugging.DEBUG_OUT("del addr: %+v\n",addr)
						//     netlink.AddrDel(link,&addr)
						// }
					}
				}

			}
			break
		}
	}
	return

}

// GetInterfaceIndexAndName lets you pass in either an ifname string, with the network interface name, or
// a valid index. The function will return both the index and name, or error
// if such inteface does not exist. Prefers looking up by name first, if len(ifname) < 0
// it will look up by index
func GetInterfaceIndexAndName(ifname string, ifindex int) (retifname string, retifindex int, err error) {
	linklist, err := netlink.LinkList()
	if err == nil {
		if len(ifname) > 0 {
			for _, link := range linklist {
				if link.Attrs().Name == ifname {
					retifname = ifname
					retifindex = link.Attrs().Index
					return
				}
			}
			iferr := &NetworkAPIError{
				Code:      ERROR_NO_IF,
				Errstring: fmt.Sprintf("Can't find interface of Name %s", ifname),
				IfName:    ifname,
			}
			err = iferr
		} else {
			for _, link := range linklist {
				if link.Attrs().Index == ifindex {
					retifname = link.Attrs().Name
					retifindex = ifindex
					return
				}
			}
			iferr := &NetworkAPIError{
				Code:      ERROR_NO_IF,
				Errstring: fmt.Sprintf("Can't find interface of Index %d", ifindex),
				IfIndex:   ifindex,
			}
			err = iferr
		}
	}
	return
}

// GetInterfaceLink lets you pass in either an ifname string, with the network interface name, or
// a valid index. The function will return a netlink.Link, or error
// if such inteface does not exist. Prefers looking up by name first, if len(ifname) < 0
// it will look up by index
func GetInterfaceLink(ifname string, ifindex int) (ret netlink.Link, err error) {
	linklist, err := netlink.LinkList()
	if err == nil {
		if len(ifname) > 0 {
			for _, link := range linklist {
				if link.Attrs().Name == ifname {
					ret = link
					return
				}
			}
			iferr := &NetworkAPIError{
				Code:      ERROR_NO_IF,
				Errstring: fmt.Sprintf("Can't find interface of Name %s", ifname),
				IfName:    ifname,
			}
			err = iferr
		} else {
			for _, link := range linklist {
				if link.Attrs().Index == ifindex {
					ret = link
					return
				}
			}
			iferr := &NetworkAPIError{
				Code:      ERROR_NO_IF,
				Errstring: fmt.Sprintf("Can't find interface of Index %d", ifindex),
				IfIndex:   ifindex,
			}
			err = iferr
		}
	}
	return
}

// RemoveDefaultRoute removes the default route from the default routing table
// To understand "default table" - read more here:
// https://www.thomas-krenn.com/en/wiki/Two_Default_Gateways_on_One_System
func RemoveDefaultRoute() (err error) {
	route := &netlink.Route{
		Dst: nil,
		Gw:  nil,
	}
	err = netlink.RouteDel(route)
	return
}

// setupDefaultRouteInPrimaryTable sets a default route into the NetworkManager's internal route priority table
// based on the ifconfig information and/or the DhcpLeaseInfo
// If the ifconfig has a provided default gateway, this will be used, otherwise the DhcpLeaseInfo's
// default gateway will be provided. If neither have a setting, then no route will be set, 'routeset' will
// be fales and gw will return the zero value. leaseinfo may be nil, in which case it is ignored.
// The route priority table will set one active default route, based on interface status and priority.
func setupDefaultRouteInPrimaryTable(inst *networkManagerInstance, ifconfig *maestroSpecs.NetIfConfigPayload, leaseinfo *DhcpLeaseInfo) (routeset bool, gw string, err error) {
	if leaseinfo != nil && leaseinfo.defaultRoute != nil {
		inst.primaryTable.removeDefaultRouteForInterface(ifconfig.IfName)
	}
	// do we have a statically set default gateway? if so use it
	if ifconfig != nil && len(ifconfig.DefaultGateway) > 0 {
		ip := net.ParseIP(ifconfig.DefaultGateway)
		if ip != nil {

			// get the link index for this interface
			ifname, ifindex, err2 := GetInterfaceIndexAndName(ifconfig.IfName, ifconfig.IfIndex)

			if err != nil {
				err = err2
				return
			}

			route := &netlink.Route{
				Dst:       nil,
				Gw:        ip,
				LinkIndex: ifindex,
			}
			err = inst.primaryTable.addDefaultRouteForInterface(ifname, ifconfig.RoutePriority, route, ifup)
			if err == nil {
				routeset = true
				gw = ip.String()
			}
		} else {
			err = &NetworkAPIError{
				Code:      ERROR_INVALID_SETTINGS,
				Errstring: fmt.Sprintf("Not a valid IP for GW address"),
				IfIndex:   ifconfig.IfIndex,
				IfName:    ifconfig.IfName,
			}
			return
		}
		// parse IP
		// set route
	} else {
		prio := ifconfig.RoutePriority
		if leaseinfo != nil {
			ok, ip := leaseinfo.GetRouter()
			if !ok {
				// ok, we got no router setting from DHCP. However, we are going to make an assumption
				// this is a semi-broken DHCP server setup, and that the router is the DHCP server.
				// Since this is an assumption, we will make this the lowest priority choice in the
				// primaryTable.
				prio = maestroSpecs.MaxRoutePriority // so make the prio - now - the "worst" priority
				log.MaestroWarnf("NetworkManager: DHCP server had no router option. Looking for workarounds...")
				// So will mark DHCP server itself as router, but with weaker prio.

				ok, ip = leaseinfo.GetDHCPServer()
				if ok {
					log.MaestroWarnf("NetworkManager: Workaround - will put DHCP Server addr %s as router, but at lowest prio.", ip.String())
				} else {
					log.MaestroWarnf("NetworkManager: Workaround - no DHCP Server option. OK, GIADDR?")
					ok, ip = leaseinfo.GetGIADDR()
					if ok && !ip.Equal(net.IPv4(0, 0, 0, 0)) {
						log.MaestroWarnf("NetworkManager: Workaround - will put GIADDR field %s as router, but at lowest prio.", ip.String())
					} else {
						log.MaestroWarnf("NetworkManager: Workaround - GIADDR is empty. OK, SIADDR?")
						ok, ip = leaseinfo.GetSIADDR()
						if ok && !ip.Equal(net.IPv4(0, 0, 0, 0)) {
							log.MaestroWarnf("NetworkManager: Workaround - will put SIADDR field %s as router, but at lowest prio.", ip.String())
						} else {
							ok = false
							log.MaestroErrorf("NetworkManager: DHCP server is broken. No workaround available. No route to Internet.")
							err = &NetworkAPIError{
								Code:       ERROR_DHCP_INVALID_LEASE,
								Errstring:  fmt.Sprintf("DHCP: can't determine router: %s", ifconfig.IfName),
								IfName:     ifconfig.IfName,
								Underlying: nil,
							}
						}
					}
				}
			}
			if ok {
				route := &netlink.Route{
					Dst: nil,
					Gw:  ip,
				}
				err = inst.primaryTable.addDefaultRouteForInterface(ifconfig.IfName, prio, route, ifup)
				if err == nil {
					leaseinfo.defaultRoute = route
					routeset = true
					gw = ip.String()
				}
			}
		}
	}
	return
}

// SetupDefaultRouteFromLease sets a default route based on the ifconfig information and/or the DhcpLeaseInfo
// If the ifconfig has a provided default gateway, this will be used, otherwise the DhcpLeaseInfo's
// default gateway will be provided. If neither have a setting, then no route will be set, 'routeset' will
// be fales and gw will return the zero value. leaseinfo may be nil, in which case it is ignored.
func SetupDefaultRouteFromLease(ifconfig *maestroSpecs.NetIfConfigPayload, leaseinfo *DhcpLeaseInfo) (routeset bool, gw string, err error) {
	if leaseinfo != nil && leaseinfo.defaultRoute != nil {
		err2 := netlink.RouteDel(leaseinfo.defaultRoute)
		if err2 != nil {
			log.MaestroErrorf("NetworkManager: Could not remove old DHCP lease route: %s", err2.Error())
		} else {
			leaseinfo.defaultRoute = nil
		}
	}
	if ifconfig != nil && len(ifconfig.DefaultGateway) > 0 {
		ip := net.ParseIP(ifconfig.DefaultGateway)
		if ip != nil {
			route := netlink.Route{
				Dst: nil,
				Gw:  ip,
			}
			err = netlink.RouteAdd(&route)
			if err == nil {
				routeset = true
				gw = ip.String()
			}
		} else {
			err = &NetworkAPIError{
				Code:      ERROR_INVALID_SETTINGS,
				Errstring: fmt.Sprintf("Not a valid IP for GW address"),
				IfIndex:   ifconfig.IfIndex,
				IfName:    ifconfig.IfName,
			}
			return
		}
		// parse IP
		// set route
	} else {
		if leaseinfo != nil {
			ok, ip := leaseinfo.GetRouter()
			if ok {
				route := &netlink.Route{
					Dst: nil,
					Gw:  ip,
				}
				err = netlink.RouteAdd(route)
				if err == nil {
					leaseinfo.defaultRoute = route
					routeset = true
					gw = ip.String()
				}
			}
		}
	}
	return
}

// AppendDNSResolver will find the resolver info, and append it to the dnsFileBuffer bytes.Buffer. If successful,
// dnsset will return true, primaryDNS will return the primary
// DNS server to use, and err will be nil. ifconfig data will take precedence
// over the DhcpLeaseInfo data. If ifconfig has no DNS information, then the leaseinfo data will be used, if available.
func AppendDNSResolver(ifconfig *maestroSpecs.NetIfConfigPayload, leaseinfo *DhcpLeaseInfo, dnsFileBuffer *dnsBuf, commentonly bool) (dnsset bool, primaryDNS string, err error) {
	// if ifconfig != nil {
	// actually ifconfig does not carry DNS info right now - TODO
	// }
	if leaseinfo != nil {
		ok, ips := leaseinfo.GetDNS()
		if ok {
			dnsFileBuffer.lock.Lock()
			dnsFileBuffer.buf.WriteString(fmt.Sprintf("# via interface %s (maestro DHCP)\n", ifconfig.IfName))
			if commentonly {
				dnsFileBuffer.buf.WriteString("#     DNS from DHCP is disabled by 'dns_ignore_dhcp: true'\n")
			}
			for _, ip := range ips {
				dnsset = true
				if len(primaryDNS) < 1 {
					primaryDNS = ip.String()
				}
				if commentonly {
					dnsFileBuffer.buf.WriteString("# ")
				}
				dnsFileBuffer.buf.WriteString(fmt.Sprintf("nameserver %s\n", ip.String()))
			}
			debugging.DEBUG_OUT("AppendDNSResolve() output %s\n", dnsFileBuffer.buf.String())
			dnsFileBuffer.lock.Unlock()
		}
		ok, localdomain := leaseinfo.GetLocalDomain()
		if ok {
			dnsFileBuffer.lock.Lock()
			if commentonly {
				dnsFileBuffer.buf.WriteString("# ")
			}
			dnsFileBuffer.buf.WriteString(fmt.Sprintf("search %s\n", localdomain))
			dnsFileBuffer.lock.Unlock()
		}
	}
	return
}

func SetupInterfaceFromLease(ifconfig *maestroSpecs.NetIfConfigPayload, leaseinfo *DhcpLeaseInfo) (result *InterfaceStatus, err error) {
	debugging.DEBUG_OUT("SetupInterfaceFromLease() -->\n")
	ifname, ifindex, err := GetInterfaceIndexAndName(ifconfig.IfName, ifconfig.IfIndex)
	if err != nil {
		return
	}
	if leaseinfo == nil {
		iferr := &NetworkAPIError{
			Code:      ERROR_INVALID_SETTINGS,
			Errstring: fmt.Sprintf("Interface %s lease was nil", ifname),
			IfName:    ifname,
		}
		err = iferr
		return
	}

	// first, let's setup the IP address
	ip := &net.IPNet{IP: leaseinfo.CurrentIP, Mask: leaseinfo.CurrentMask}
	linkaddr := &netlink.Addr{IPNet: ip}
	isset, err := IsIPv4AddressSet(ifname, linkaddr)
	if err != nil {
		debugging.DEBUG_OUT("IPv4 Addr: %s already set\n", linkaddr.String())
		return
	}
	result = new(InterfaceStatus)
	// if the IP address is already set, we will leave it alone

	link, err2 := netlink.LinkByName(ifname)
	if err2 == nil {

		if !ifconfig.DhcpDisableClearAddresses {
			ClearAllAddressesByLinkName(ifname, nil)
		}

		if !isset {
			err2 = netlink.AddrReplace(link, linkaddr)
			if err2 != nil {
				iferr := &NetworkAPIError{
					Code:       ERROR_GENERAL_INTERFACE_ERROR,
					Errstring:  fmt.Sprintf("Failed to set interface %s address", ifname),
					IfName:     ifname,
					Underlying: err2,
				}
				err = iferr
				return
			}
		}
	} else {
		iferr := &NetworkAPIError{
			Code:       ERROR_NO_IF,
			Errstring:  fmt.Sprintf("LinkByName failed: %s", ifname),
			IfName:     ifname,
			Underlying: err2,
		}
		err = iferr
		return
	}
	if (link.Attrs().Flags & net.FlagUp) > 0 {
		result.Up = true
	}
	debugging.DEBUG_OUT("SetupInterfaceFromLease() IP addr set.\n")
	result.IfName = ifname
	result.IfIndex = ifindex
	result.ipv4 = leaseinfo.CurrentIP
	result.IPV4 = leaseinfo.CurrentIP.String()

	return
}

func addDefaultRoutesToPrimaryTable(inst *networkManagerInstance, ifs []*maestroSpecs.NetIfConfigPayload) (results []InterfaceStatus, err error) {
	linklist, err := netlink.LinkList()
	if err == nil {
		for _, link := range linklist {
			debugging.DEBUG_OUT("link: %+v\n", link)
			link = link
		}
	}

	results = make([]InterfaceStatus, len(ifs))
	links := make([]netlink.Link, len(ifs))
	for n, configif := range ifs {
		results[n].IfName = configif.IfName
		results[n].IfIndex = configif.IfIndex
		// no ifname, ok let's find it via Index
		if len(configif.IfName) < 1 {
			for _, link := range linklist {
				if link.Attrs().Index == configif.IfIndex {
					ifs[n].IfName = link.Attrs().Name
					results[n].IfName = ifs[n].IfName
					if (link.Attrs().Flags & net.FlagUp) > 0 {
						results[n].Up = true
					}
					links[n] = link
				}
			}
			if len(configif.IfName) < 1 {
				iferr := &NetworkAPIError{
					Code:      ERROR_NO_IF,
					Errstring: fmt.Sprintf("Can't find interface of Index %d", configif.IfIndex),
					IfIndex:   configif.IfIndex,
				}
				results[n].Err = iferr
			}
		} else {
			// find the Index, in case we need it
			found := false
			for _, link := range linklist {
				if link.Attrs().Name == configif.IfName {
					ifs[n].IfIndex = link.Attrs().Index
					results[n].IfIndex = ifs[n].IfIndex
					if (link.Attrs().Flags & net.FlagUp) > 0 {
						results[n].Up = true
					}
					found = true
					links[n] = link
					break
				}
			}
			if !found {
				iferr := &NetworkAPIError{
					Code:      ERROR_NO_IF,
					Errstring: fmt.Sprintf("Can't find interface of Name %s", configif.IfName),
					IfName:    configif.IfName,
				}
				results[n].Err = iferr
			}
		}

		if len(configif.DefaultGateway) > 0 {
			ip := net.ParseIP(configif.DefaultGateway)
			if ip != nil {
				route := &netlink.Route{
					Dst: nil,
					Gw:  ip,
				}
				inst.primaryTable.addDefaultRouteForInterface(configif.IfName, configif.RoutePriority, route, ifup)
			} else {
				iferr := &NetworkAPIError{
					Code:      ERROR_INVALID_SETTINGS,
					Errstring: fmt.Sprintf("Interface Name %s has invalid DefaultGateway IP address", configif.IfName),
					IfName:    configif.IfName,
				}
				results[n].Err = iferr
			}
		}
	}

	// for _, ifconf := range ifs {
	// // setup a static IP config
	// }
	return
}

// SetupStaticInterfaces sets up static interfaces
func SetupStaticInterfaces(ifs []*maestroSpecs.NetIfConfigPayload) (results []InterfaceStatus, err error) {
	debugging.DEBUG_OUT("SetupStaticInterfaces() -->\n")
	// find out what we already have...
	addrlist, err := netlink.AddrList(nil, netlink.FAMILY_V4)
	if err == nil {
		for _, addr := range addrlist {
			debugging.DEBUG_OUT("addr: %+v\n", addr)
			addr = addr // dummy code for compiler annoying errors when debug is off
		}
	}

	linklist, err := netlink.LinkList()
	if err == nil {
		for _, link := range linklist {
			debugging.DEBUG_OUT("link: %+v\n", link)
			link = link
		}
	}

	results = make([]InterfaceStatus, len(ifs))
	links := make([]netlink.Link, len(ifs))
	// find the interface name
	for n, configif := range ifs {
		results[n].IfName = configif.IfName
		results[n].IfIndex = configif.IfIndex
		// no ifname, ok let's find it via Index
		if len(configif.IfName) < 1 {
			for _, link := range linklist {
				if link.Attrs().Index == configif.IfIndex {
					ifs[n].IfName = link.Attrs().Name
					results[n].IfName = ifs[n].IfName
					if (link.Attrs().Flags & net.FlagUp) > 0 {
						results[n].Up = true
					}
					links[n] = link
				}
			}
			if len(configif.IfName) < 1 {
				iferr := &NetworkAPIError{
					Code:      ERROR_NO_IF,
					Errstring: fmt.Sprintf("Can't find interface of Index %d", configif.IfIndex),
					IfIndex:   configif.IfIndex,
				}
				results[n].Err = iferr
			}
		} else {
			// find the Index, in case we need it
			found := false
			for _, link := range linklist {
				if link.Attrs().Name == configif.IfName {
					ifs[n].IfIndex = link.Attrs().Index
					results[n].IfIndex = ifs[n].IfIndex
					if (link.Attrs().Flags & net.FlagUp) > 0 {
						results[n].Up = true
					}
					found = true
					links[n] = link
					break
				}
			}
			if !found {
				iferr := &NetworkAPIError{
					Code:      ERROR_NO_IF,
					Errstring: fmt.Sprintf("Can't find interface of Name %s", configif.IfName),
					IfName:    configif.IfName,
				}
				results[n].Err = iferr
			}
		}
		// if no errors, then let's try to configure this thing
		if results[n].Err == nil {
			if configif.ClearAddresses {
				ClearAllAddressesByLinkName(ifs[n].IfName, nil)
			}
			if !configif.DhcpV4Enabled {
				ip := net.ParseIP(configif.IPv4Addr)
				if ip != nil && (configif.IPv4Mask >= 0 && configif.IPv4Mask < 33) {
					ipnetaddr := &net.IPNet{IP: ip, Mask: net.CIDRMask(configif.IPv4Mask, 32)}
					addr := &netlink.Addr{
						IPNet: ipnetaddr,
						Label: configif.IfName,
						Scope: unix.RT_SCOPE_UNIVERSE,
						Flags: unix.IFA_F_PERMANENT, // | unix.IFA_F_OPTIMISTIC, // Optimisitc doc here (all I can find):  https://lwn.net/Articles/218597/
					}
					err2 := netlink.AddrReplace(links[n], addr)
					if err2 != nil {
						log.MaestroErrorf("NetworkManager: Failed to setup interface %s - %s", configif.IfName, err2.Error())
						results[n].Err = err2
					} else {
						results[n].ipv4 = ip
						results[n].IPV4 = ip.String()
					}
				} else {
					iferr := &NetworkAPIError{
						Code:      ERROR_INVALID_SETTINGS,
						Errstring: fmt.Sprintf("Interface Name %s has invalid IPv4Addr / IPv4Mask", configif.IfName),
						IfName:    configif.IfName,
					}
					results[n].Err = iferr
				}
			}
		}
	}

	// for _, ifconf := range ifs {
	// // setup a static IP config
	// }
	debugging.DEBUG_OUT("<-- returning SetupInterfaces()\n")
	return
}
