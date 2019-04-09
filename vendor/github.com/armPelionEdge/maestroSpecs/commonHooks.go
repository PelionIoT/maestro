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

// These interfaces are for building a hook library for maestro, which 
// allows the use of custom plugins for specific hardware

// For any Hook library to work, this interface must be implemented:
type Hook interface {
    // Impl. should setup the given hook library, returning an err (error) if
    // the setup failed to complete. A nil err indicates the Hook installed correctly.
    // The id string whould be instance unique to Maestro
    // The library should use log Logger to output any info or errors
    MaestroInitHook(log Logger) (err error, id string)

    // called when maestro shuts down
    // The signal type for shutdown is passed, or zero if none.
    MaestroStopHook(sig int)
}

type CommonHooks interface {
    // called when Maestro has started all services
    MaestroReady() 

    // called at a periodic interval. Useful for doing something periodically
    // such as hitting a watchdog. 'counter' is continually incremented
    // See maestro/src/maestroConfig for details 
    MaestroPeriodicOk(counter int)
}

type JobHooks interface {

    MaestroJobStarted(name string)

    MaestroJobStopped(name string)

    MaestroJobFailed(name string)

    // called if a Job's max restarts are reached, or other reason
    // the Job can no longer ever be restarted
    MaestroJobTotalFailure(name string)
}


type NetworkHooks interface {
        
    // called if the default route is setup (typicall meaning the network or Internet is reachable)
    // and DNS is setup with at least a single resolver.
    MaestroNetworkReady()

    // called when Maestro has positive connectivity to it's management server
    MaestroCloudManagerReady()

    // called whent the conditions of MaestroNetworkReady or no longer true
    MaestroNetworkDown()

    // called when an Interface is setup
    MaestroInterfaceSetup(name string, ifindex int)

    // called when an interface is shutdown
    MaestroInterfaceShutdown(name string, ifindex int)

}
