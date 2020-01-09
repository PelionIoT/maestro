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

## Developing Maestro

Due to the large list of system services provided by Maestro, the recommended way to develop and test Maestro is in isolation using a virtual machine.  Running Maestro on your local dev box, while possible, would create a lot of headaches and running Maestro in Docker is not possible.  See the 'Why Vagrant?' section for additional info.

### Pre-Requisites

Install the following tools:

* [vagrant](https://www.vagrantup.com/) (version 2.2.6)
* [virtualbox](https://www.virtualbox.org/) (version 6.0)

Install Vagrant plugins. `vagrant-reload` is a plugin that allows vagrant to reboot a VM during provisioning. This is needed to reboot the vagrant VM while setting up required network interfaces.

```bash
vagrant plugin install vagrant-reload
```

### Why Vagrant?

Maestro uses the `vagrant` for the following reasons:
1. Reliability - Vagrant guarantees that every user that tries to build maestro gets the same environment and toolset.
2. Network control - Unlike docker, vagrant allows us to specify network interfaces. Specifically, we setup a control and test network, which is explained in the `vagrant` folder README
3. System testing - Once again a limitation to docker, vagrant allows us to start and stop the maestro daemon and configure the system for complete system testing. Additionally, we can bring in other daemons into the system over time for additional testing (ex. devicedb, devicejs)

For more information as to how vagrant sets up the maestro environment, please read the README in the `vagrant` folder

### Building

Start up the virtual machine. The very first time `vagrant up` is run, the VM will go through a provisioning phase.

```bash
git clone git@github.com:armPelionEdge/maestro.git
cd maestro
vagrant up
```

Then build maestro and its dependencies. You can run this whenever you desire as long as the VM is online

```bash
vagrant ssh -c "build_maestro"
```

### Running

```bash
vagrant ssh -c "sudo maestro"
```

### Testing

#### Unit tests

Example of running a networking test:

```bash
vagrant ssh # Log in to VM
cd $MAESTRO_SRC # Go to maestro home dir

cd networking # Open networking tests
go test -v -run DhcpRequest # Run DhcpRequest test
```

#### System Tests

On the host machine, run the following commands:
Note: Make sure you had built using `vagrant up` and `vagrant ssh -c "build_maestro"` before running tests.

```bash
cd tests # Go to SysTests folder
npm i # Only run once to download dependencies
npm test # Run mocha test suite
```

## Additional Features/Information

### DeviceDB

In order to test additional maestro functionality, the vagrant VM has the ability to startup and run devicedb, another Arm Pelion Edge gateway feature.

Before going further, please make sure you have built maestro using the instructions above. If you built when in a SSH shell, please restart your shell.

To start devicedb, run the following command from your host machine:

```bash
vagrant ssh -c "devicedb_server"
```
