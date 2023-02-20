# maestro
Pelion Edge systems management daemon for Pelion Edge OS.

## Overview

Maestro is a replacement for a number of typical Linux OS system utilities and management programs, while providing cloud-connected systems management. Maestro is designed specifically for cloud-connected Linux OS embedded computers, with somewhat limited RAM and disk space, where file systems are often flash - and would prefer less writing to the FS over time.

Major tasks maestro provides:
- syslog daemon (replaces syslog-ng, syslogd, and others)
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
- management via local API

Maestro communicates to Pelion Cloud over HTTPS outbound. It stores its config locally in a private database, but can also use DeviceDB for storage of applications, network settings, configs and other data when used in conjunction with standard Pelion Cloud services.

If you are locally on a gateway / edge system using maestro, you should explore [maestro-shell](https://github.com/PelionIoT/maestro-shell) which will let you interact with maestro directly using the local API.

## Building

### Prerequisites

* Linux
* go (tested on go 1.15.1)
* libuv (on Debian distributions, run `sudo apt install libuv1-dev`)

### Build

```
mkdir -p bin
go build -o bin/maestro maestro/main.go
```

## Additional Features/Information

### Maestro Configuration

To view details on how to configure Maestro, see [Maestro Config](docs/maestro_config.md)
