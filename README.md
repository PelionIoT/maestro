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

### Interacting

If you are locally on a gateway / edge system using maestro, you should explore [maestro-shell](https://github.com/armPelionEdge/maestro-shell) which will let you interact with maestro directly using the local API.

### Building

#### prerequisites

Build pre-requisites:

* golang
* python
* gcc

Maestro has a simple internal preprocessor `build.sh` script which use `m4` - make sure `m4` is installed on your build system (normally included with gcc)

You will need gcc for compiling maestro's dependencies.

*On Ubuntu*

`sudo apt-get install build-essential python`  should take care of all this.

If you go not have a Go build environment up, you will need to install [golang](https://golang.org/dl/). Just expand the tar ball and put in in /opt, with `sudo mv go /opt`

You also will need your golang env vars setup correctly. Here is a script you can just run, assuming your Go workspace is at `$HOME/work/gostuff`:

*setup-go.sh*
```
#!/bin/bash

export GIT_TERMINAL_PROMPT=1
export GOROOT=/opt/go
export GOPATH="$HOME/work/gostuff"
export GOBIN="$HOME/work/gostuff/bin"
export PATH="$PATH:$GOROOT/bin:$GOBIN"
```

then just run: `source ./setup-go.sh` 

Run your build commands from this sub-shell.

##### On Arch

 * install go:
  `sudo pacman -S golang`

 * Add generated executables into your path (so that you can do stuff like `maestro` on the command line)
   ```
   export PATH="$PATH:$HOME/go/bin"
   ```


#### build instructions

`go get github.com/armPelionEdge/maestro`

Enter github credentials as needed. Now, where `$GOPATH` is your go workspace folder, as in the sub-shell above...

```
cd $GOPATH/src/github.com/armPelionEdge/maestro
```

Run a script to build dependencies in the `/vendor` folder:
```
./build-deps.sh
```

This will take a bit. You're building a bunch of libs used by maestro, including some native code.

Next build maestro:

```
cd $GOPATH/src/github.com/armPelionEdge/maestro
DEBUG=1 DEBUG2=1 ./build.sh
```

Eliminate `DEBUG` and `DEBUG2` env var to generate less debug output.

#### Running tests

Different subsystems have different test suites. Running these may require root privleges, for instance networking. You also need to run the pre-processor if making changes.

Example, from `$GOPATH/src/github.com/armPelionEdge/maestro`:

```
DEBUG=1 DEBUG2=1 ./build.sh && cd networking && sudo -E \
  LD_LIBRARY_PATH=../vendor/github.com/armPelionEdge/greasego/deps/lib go test -v -run DhcpRequest
```

Notes:

* The dependencies of greasego (a native library extension to go) are in
  `$GOPATH/src/github.com/armPelionEdge/maestro/vendor/github.com/armPelionEdge/greasego/deps/lib`,
  which requires us to add that directory to the `LD_LIBRARY_PATH` environment variable so that
  go can find these libraries. A relative path is used in the example to reduce overall command length.
* The build command is not strictly required to run tests, when you have already built. It's good practice
  to include this in the command so that you don't test a stale build.

#### Running `maestro`

Maestro requires a config in it's working directory called `maestro.config`, so create it:

`touch maestro.config`

Build and run `maestro`:
```
DEBUG=1 DEBUG2=1 ./build.sh && LD_LIBRARY_PATH=vendor/github.com/armPelionEdge/greasego/deps/lib maestro
```

If you get a `maestro: command not found` error, check that your `$GOBIN` is part of your `$PATH`

#### Other examples

The Docker build file, for `djs-soft-relay` shows build instruction also, using this exact above method.
https://github.com/armPelionEdge/cloud-installer/blob/master/djs-soft-relay/build-wwcontainer.sh#L155

#### TroubleShooting

1. `[ERROR] Failed to create scratch path directory: mkdir /tmp/maestro/images: permission denied`
   `[ERROR] Failed to create scratch path directory: mkdir /var/maestro/images: permission denied`

Maestro execution on local host creates directories at `/tmp/maestro/images` and `/var/maestro/images` path. Verify that the user running `maestro` has the proper permissions to access the respective directories.

