package zeroconf

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// RegisterServiceEntry let's you register a service by handing in a raw ServiceEntry
// struct for better flexibility, along with string array of ips to publish and intefaces
// to run the server on. notIfaces lets you black list specific interface to not publish to
func RegisterServiceEntry(entry *ServiceEntry, ips []string, ifaces []string, notIfaces []string) (*Server, error) {
	// entry := NewServiceEntry(instance, service, domain)
	// entry.Port = port
	// entry.Text = text
	// entry.HostName = host

	if entry.Instance == "" {
		return nil, fmt.Errorf("Missing service instance name")
	}
	if entry.Service == "" {
		return nil, fmt.Errorf("Missing service name")
	}
	// if entry.HostName == "" {
	// 	return nil, fmt.Errorf("Missing host name")
	// }
	if entry.Domain == "" {
		entry.Domain = "local"
	}
	if entry.Port == 0 {
		return nil, fmt.Errorf("Missing port")
	}

	if len(ips) < 1 {
		return nil, fmt.Errorf("No IP address stated")
	}

	var err error
	if entry.HostName == "" {
		entry.HostName, err = os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("Could not determine host")
		}
	}

	if !strings.HasSuffix(trimDot(entry.HostName), entry.Domain) {
		entry.HostName = fmt.Sprintf("%s.%s.", trimDot(entry.HostName), trimDot(entry.Domain))
	}

	for _, ip := range ips {
		ipAddr := net.ParseIP(ip)
		if ipAddr == nil {
			return nil, fmt.Errorf("Failed to parse given IP: %v", ip)
		} else if ipv4 := ipAddr.To4(); ipv4 != nil {
			entry.AddrIPv4 = append(entry.AddrIPv4, ipAddr)
		} else if ipv6 := ipAddr.To16(); ipv6 != nil {
			entry.AddrIPv6 = append(entry.AddrIPv6, ipAddr)
		} else {
			return nil, fmt.Errorf("The IP is neither IPv4 nor IPv6: %#v", ipAddr)
		}
	}

	var _ifaces []net.Interface

	if len(ifaces) == 0 {
		_ifaces = listMulticastInterfaces()
	} else {
		for _, name := range ifaces {
			var iface net.Interface
			actualiface, err := net.InterfaceByName(name)
			if err == nil {
				iface = *actualiface
			} else {
				return nil, fmt.Errorf("interface not found %s", name)
			}
			_ifaces = append(_ifaces, iface)
		}
	}

	for _, name := range notIfaces {
	innerIfLoop:
		for i, ifs := range _ifaces {
			if ifs.Name == name {
				// this removes interface 'i' from the list
				// (bump the last interface off the list, and replace slot 'i'
				// then resize the slice, not have the - now duplicated - last slot)
				_ifaces[i] = _ifaces[len(_ifaces)-1]
				_ifaces[len(_ifaces)-1] = net.Interface{} // make sure GC cleanups all fields in Interface
				_ifaces = _ifaces[:len(_ifaces)-1]
				break innerIfLoop
			}
		}
	}

	logDebug("RegisterServiceEntry: publishing to interfaces: ", _ifaces)

	s, err := newServer(_ifaces)
	if err != nil {
		return nil, err
	}

	s.service = entry
	go s.mainloop()
	go s.probe()

	return s, nil
}

func dupServiceEntryNoIps(entry *ServiceEntry) (ret *ServiceEntry) {
	// ret = NewServiceEntry(entry.Instance, entry.Service, entry.Domain)
	// ret.HostName = entry.HostName
	ret = new(ServiceEntry)
	*ret = *entry
	//	(*ret).ServiceRecord = (*entry).ServiceRecord
	// ret.ServiceRecord.serviceInstanceName = entry.ServiceRecord.serviceInstanceName
	// ret.ServiceRecord.serviceName = entry.ServiceRecord.serviceName
	// ret.ServiceRecord.serviceTypeName = entry.ServiceRecord.serviceTypeName
	ret.AddrIPv4 = nil
	ret.AddrIPv6 = nil
	ret.Text = make([]string, len(entry.Text))
	copy(ret.Text, entry.Text)
	return
}

// RegisterServiceEntryEachInterfaceIP publishes the service entry to each interface separately, using each interfaces
// assigned IP address. If ifaces is empty, it will go through all network interfaces except loopback.
func RegisterServiceEntryEachInterfaceIP(entry *ServiceEntry, ifaces []string, notIfaces []string) (servers []*Server, err error) {
	// entry := NewServiceEntry(instance, service, domain)
	// entry.Port = port
	// entry.Text = text
	// entry.HostName = host

	if entry.Instance == "" {
		return nil, fmt.Errorf("Missing service instance name")
	}
	if entry.Service == "" {
		return nil, fmt.Errorf("Missing service name")
	}
	// if entry.HostName == "" {
	// 	return nil, fmt.Errorf("Missing host name")
	// }
	// fmt.Printf("mdns Domain:[%s] %d\n", entry.Domain, len(entry.Domain))
	if entry.Domain == "" {
		entry.Domain = "local"
	}
	logDebug("Domain:[%s] %d\n", entry.Domain, len(entry.Domain))
	if entry.Port == 0 {
		return nil, fmt.Errorf("Missing port")
	}

	if entry.HostName == "" {
		entry.HostName, err = os.Hostname()
		if err != nil {
			err = fmt.Errorf("Could not determine host")
			return
		}
	}

	if !strings.HasSuffix(trimDot(entry.HostName), entry.Domain) {
		entry.HostName = fmt.Sprintf("%s.%s.", trimDot(entry.HostName), trimDot(entry.Domain))
	}

	// for _, ip := range ips {
	// 	ipAddr := net.ParseIP(ip)
	// 	if ipAddr == nil {
	// 		return nil, fmt.Errorf("Failed to parse given IP: %v", ip)
	// 	} else if ipv4 := ipAddr.To4(); ipv4 != nil {
	// 		entry.AddrIPv4 = append(entry.AddrIPv4, ipAddr)
	// 	} else if ipv6 := ipAddr.To16(); ipv6 != nil {
	// 		entry.AddrIPv6 = append(entry.AddrIPv6, ipAddr)
	// 	} else {
	// 		return nil, fmt.Errorf("The IP is neither IPv4 nor IPv6: %#v", ipAddr)
	// 	}
	// }

	var _ifaces []net.Interface

	if len(ifaces) == 0 {
		_ifaces = listMulticastInterfaces()
		logDebug("found multicast Interfaces ", _ifaces)
	} else {
		for _, name := range ifaces {
			var iface net.Interface
			actualiface, err2 := net.InterfaceByName(name)
			if err2 == nil {
				iface = *actualiface
			} else {
				err = fmt.Errorf("interface not found %s", name)
				return
			}
			_ifaces = append(_ifaces, iface)
		}
	}

	for _, name := range notIfaces {
	innerIfLoop:
		for i, ifs := range _ifaces {
			if ifs.Name == name {
				// this removes interface 'i' from the list
				// (bump the last interface off the list, and replace slot 'i'
				// then resize the slice, not have the - now duplicated - last slot)
				_ifaces[i] = _ifaces[len(_ifaces)-1]
				_ifaces[len(_ifaces)-1] = net.Interface{} // make sure GC cleanups all fields in Interface
				_ifaces = _ifaces[:len(_ifaces)-1]
				break innerIfLoop
			}
		}
	}

	//	logDebug("mdns orgin:%+v\n", entry)
	// loop through and publish each inteface / IP combination
	for _, interf := range _ifaces {
		// get address(es) for each interface
		addrs, err2 := interf.Addrs()
		if err2 != nil {
			err = err2
			return
		}
		// create a unique entry for each interface
		newentry := dupServiceEntryNoIps(entry)
		//		logDebug("mdns duped entry: %+v\n", newentry)
		for _, addr := range addrs {
			switch ip := addr.(type) {
			case *net.IPNet:
				if ip.IP.DefaultMask() != nil {
					if ipv4 := ip.IP.To4(); ipv4 != nil {
						newentry.AddrIPv4 = append(newentry.AddrIPv4, ip.IP)
					} else if ipv6 := ip.IP.To16(); ipv6 != nil {
						newentry.AddrIPv6 = append(newentry.AddrIPv6, ip.IP)
					}
					// else {
					// 	return nil, fmt.Errorf("The IP is neither IPv4 nor IPv6: %#v", ip.IP)
					// }
					//					newentry.AddrIPv4
					//					return nil, (ip.IP)
				}
			}
		}
		logDebug("Serving out interface ", interf.Name)
		server, err2 := newServer([]net.Interface{interf})
		if err2 == nil {
			servers = append(servers, server)
		} else {
			err = err2
			return
		}
	}

	for _, server := range servers {
		server.service = entry
		go server.mainloop()
		go server.probe()
	}

	// s, err := newServer(_ifaces)
	// if err != nil {
	// 	return nil, err
	// }

	// s.service = entry
	// go s.mainloop()
	// go s.probe()

	return

}
