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
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/PelionIoT/maestro/log"
	"github.com/PelionIoT/maestroSpecs"
	"github.com/PelionIoT/netlink"
)

const (
	ifup   = iota
	ifdown = iota
)

// routingTable helps manage having multiple interfaces with multiple
//

type routePrio struct {
	route  *netlink.Route
	prio   int // lower is "better" priority
	status int // up or down
	active bool
	// ifname       string
	// ifs          map[string]*netlink.Route
	// lock         sync.Mutex
}

type routingTable struct {
	// defaultRoutesMap keeps track of any interface which had a default route
	// and allows the network manager to pick the next highest priority
	// name to *defaultRoutePrio
	defaultRoutes sync.Map
	// protects variable below
	lock sync.Mutex
	// which interface currently has the default route
	primaryDefaultRouteIf string
	defaultRoute          *netlink.Route
	oldDefaultRoute       *netlink.Route
}

func (table *routingTable) addDefaultRouteForInterface(ifname string, prio int, route *netlink.Route, status int) (err error) {
	if route != nil {
		if prio > maestroSpecs.MaxRoutePriority {
			err = errors.New("invalid priority")
			return
		}
		table.defaultRoutes.Store(ifname, &routePrio{
			prio:   prio,
			route:  route,
			status: status,
			active: false,
		})
	} else {
		log.MaestroError("NetworkManager: an attempt to play a nil route in table occurred")
	}
	return
}

func (table *routingTable) removeDefaultRouteForInterface(ifname string) {
	table.defaultRoutes.Delete(ifname)
}

func (table *routingTable) markIfAsDown(ifname string) (err error) {
	rP, ok := table.defaultRoutes.Load(ifname)
	if ok {
		r, ok := rP.(*routePrio)
		if ok {
			r.status = ifdown
			r.active = false
		} else {
			err = errors.New("internal error")
		}
	} else {
		err = errors.New("no interface")
	}
	return
}

func (table *routingTable) markIfAsUp(ifname string) (err error) {
	rP, ok := table.defaultRoutes.Load(ifname)
	if ok {
		r, ok := rP.(*routePrio)
		if ok {
			r.status = ifup
		} else {
			err = errors.New("internal error")
		}
	} else {
		err = errors.New("no interface")
	}
	return
}

func (table *routingTable) isIfUp(ifname string) (up bool, err error) {
	rP, ok := table.defaultRoutes.Load(ifname)
	if ok {
		r, ok := rP.(*routePrio)
		if ok {
			if r.status == ifup {
				up = true
			}
		} else {
			err = errors.New("internal error")
		}
	} else {
		err = errors.New("no interface")
	}
	return
}

// walks all routes. Returns the route with the lowest prio value which is not down.
// changed returns true if there is something actionable
func (table *routingTable) findPreferredRoute() (ok bool, ifname string, route *netlink.Route) {
	bestPrio := maestroSpecs.MaxRoutePriority

	var rPrio, r *routePrio
	var name string
	var ok2, ok3 bool
	// walk through all value in map
	// mark all as inactive
	// find the route with the lowest prio (best) which is still active
	check := func(key, val interface{}) (ret bool) {
		name, ok3 = key.(string)
		r, ok2 = val.(*routePrio)
		if ok3 && ok2 {
			if r.status == ifup && r.prio <= bestPrio {
				rPrio = r
				bestPrio = r.prio
			}
			// reset all routes to not active
			r.active = false
			// at the very least, we found 1 route, even if it's down
			ok = true
		}
		return true
	}

	table.defaultRoutes.Range(check)
	if ok {
		if rPrio == nil {
			table.lock.Lock()
			table.primaryDefaultRouteIf = ""
			table.oldDefaultRoute = table.defaultRoute
			table.defaultRoute = nil
			table.lock.Unlock()
		} else {
			rPrio.active = true
			ifname = name
			route = rPrio.route
			table.lock.Lock()
			table.primaryDefaultRouteIf = name
			table.oldDefaultRoute = table.defaultRoute
			table.defaultRoute = rPrio.route
			table.lock.Unlock()
		}
	}
	return
}

func (table *routingTable) setPreferredRoute(setDefaultRoute bool) (err error) {
	table.lock.Lock()
	if setDefaultRoute {
		// was there an old route, remove it.
		if table.oldDefaultRoute != nil { //} && (table.oldDefaultRoute != table.defaultRoute) {
			table.lock.Unlock()
			netlink.RouteDel(table.oldDefaultRoute)
			table.lock.Lock()
		}
		// update the route if it changed
		//	if table.oldDefaultRoute != table.defaultRoute {
		route := table.defaultRoute
		table.lock.Unlock()
		if route != nil {
			err = netlink.RouteAdd(route)
			if err != nil && err.Error() == "file exists" {
				// get the link index for this interface
				// _, ifindex, err2 := GetInterfaceIndexAndName(table.primaryDefaultRouteIf, 0)

				// if err2 != nil {
				// 	return
				// }

				existing := &netlink.Route{
					Dst: &net.IPNet{
						IP:   net.IPv4(0, 0, 0, 0),
						Mask: net.IPv4Mask(0, 0, 0, 0),
					},
					// LinkIndex: ifindex,
				}

				err2 := netlink.RouteDel(existing)
				if err2 != nil {
					err = fmt.Errorf("Could not delete default route: %s", err2.Error())
					return
				}

				err = netlink.RouteAdd(route)
			}
		}
	} else {
		table.defaultRoute = &netlink.Route{
			Dst: &net.IPNet{
				IP:   net.IPv4(0, 0, 0, 0),
				Mask: net.IPv4Mask(0, 0, 0, 0),
			},
			// LinkIndex: ifindex,
		}
		table.lock.Unlock()
	}
	// } else {
	// 	table.lock.Unlock()
	// }
	return
}

func (table *routingTable) getCurrentPreferredRoute() (ifname string, route *netlink.Route) {
	ifname = table.primaryDefaultRouteIf
	route = table.defaultRoute
	return
}
