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

Install Vagrant plugins:

```bash
vagrant plugin install vagrant-reload
```

## Building

Start up the virtual machine:

```bash
vagrant up
```

## Running

```bash
vagrant ssh # Log in to VM
sudo su - # Switch to root user
cd $GOPATH/src/github.com/armPelionEdge/maestro # Go to maestro home dir

maestro # Run maestro binary
```

## Testing

### Unit tests

Example of running a networking test:

```bash
vagrant ssh # Log in to VM
sudo su - # Switch to root user
cd $GOPATH/src/github.com/armPelionEdge/maestro # Go to maestro home dir

cd networking # Open networking tests
go test -v -run DhcpRequest # Run DhcpRequest test
```

### System Tests

<To be continued>