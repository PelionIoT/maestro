# maestro
Pelion Edge systems management daemon for Pelion Edge OS. 

## Overview

Maestro is a replacement for a number of typical Linux OS system utilities and management programs, while providing cloud-connected systems management. Maestro is designed specifically for cloud-connected Linux OS embedded computers, with somewhat limited RAM and disk space, where file systems are often flash - and would prefer less writing to the FS over time.

Major tasks maestro provides:
- syslog daemon (replaces syslog-ng, syslogd, and others)
- more advanced logging via the [grease-log-client](https://github.com/armPelionEdge/grease-log-client) library
- to-the-cloud logging
- periodic system stats to cloud
- config management for apps / container (config file templating, config API)
- process creation & control
- container management - starting / stoppping & installation
- network setup (DHCP, static IP settings, with more to come)
- critical systems control (reboot, remote command execution, etc.)
- watchdog support
- time sync
- initial provisioning of system

Maestro can communicate in two ways:
- locally with other process over its local API
- a 'phone-home' style communication with its 'mothership' - this is WigWag's DCS

Advantages:
- less memory footprint
- less installation on disk
- cloud connectivity
- managment via local API

Maestro communicates to Pelion Cloud over https outbouund. It stores its config locally in a private database, but can also use DeviceDB for storage of applications, network settings, configs and other data when used in conjuction with standard Pelion Cloud services.

If you are locally on a gateway / edge system using maestro, you should explore [maestro-shell](https://github.com/armPelionEdge/maestro-shell) which will let you interact with maestro directly using the local API.

## Pre-Requisites

Install the following tools:

* [vagrant](https://www.vagrantup.com/) (version 2.2.6)
* [virtualbox](https://www.virtualbox.org/) (version 6.0)

Install Vagrant plugins. `vagrant-reload` is a plugin that allows vagrant to reboot a VM during provisioning. This is needed to reboot the vagrant VM while setting up required network interfaces.

```bash
vagrant plugin install vagrant-reload
```

### Why Vagrant?

Maestro uses the `vagrant` for the following reasons:
1. Reliability - Vagrant guarentees that every user that tries to build maestro gets the same environment and toolset.
2. Network control - Unlike docker, vagrant allows us to specify network interfaces. Specifically, we setup a control and test network, which is explained in the `vagrant` folder README
3. System testing - Once again a limitation to docker, vagrant allows us to start and stop the maestro daemon and configure the system for complete system testing. Additionally, we can bring in other daemons into the system over time for additional testing (ex. devicedb, devicejs)

For more information as to how vagrant sets up the maestro environment, please read the README in the `vagrant` folder

## Building

Start up the virtual machine:

```bash
git clone git@github.com:armPelionEdge/maestro.git
cd maestro
vagrant up
```

## Running

```bash
vagrant ssh -c "sudo maestro"
```

## Testing

### Unit tests

Example of running a networking test:

```bash
vagrant ssh # Log in to VM
cd $MAESTRO_SRC # Go to maestro home dir

cd networking # Open networking tests
go test -v -run DhcpRequest # Run DhcpRequest test
```

### System Tests

On the host machine, run the following commands:
Note: Make sure you had built using `vagrant up` before running tests.

```bash
cd tests # Go to SysTests folder
npm i # Only run once to download dependencies
npm test # Run mocha test suite
```