# Vagrant and maestro

This readme is dedicated to explaining the details of how maestro works with vagrant

## Background

There are 2 scripts that vagrant utilizes:
* provision.sh
* build.sh

### Provision

See: `provision.sh`

Provisioning sets up the virtual machine with the required set of package dependencies and configures networking interfaces.  Provisioning occurs during vagrant virtual machine creation and will not run again unless the original VM is deleted from disk or a user specifically restarts the provisioning phase.  The user must rerun provisioning if the VM is out of date with the provision.sh script.

The command to manually run provisioning on a VM is as follows:

```bash
vagrant reload --provision
```

#### Script details

The provision script does the following:
* Installs required apt packages
* Download and installs Go into `/opt/go`
* Creates a `envvars.sh` script which contains the Go environment variables
    * This script lives in `/etc/profile.d/envvars.sh` and can be sourced within a shell to give the user the Go environment variables and paths
* Creates a read/write maestro folder in `/var/maestro`
* Renames the network interfaces to the old Ubuntu network interface naming convention (`eth0`, `eth1`, etc)
* Creates 2 networks within the VM
    * `eth0` - This is the control network, and is managed by vagrant (NAT). SSH and test commands are sent over this interface
    * `eth1` - This is the test network. Maestro is allowed to configure this network however it wants, and vagrant/Ubuntu will not interact with this interface. This allows maestro to enable/disable DCHP on the network interface without locking out SSH functionality

### Build

See: `build.sh`

The build script downloads and compiles Maestro along with its dependent packages and associated utilities.  The script must be called manually after the user brings up the VM.

```bash
vagrant ssh -c "build_maestro"
```

#### Script details

The build script does the following:
* Sources the Go environment variables
* Builds maestro dependencies
* Builds maestro
* Creates a default `maestro.config` that is placed in the maestro home directory
* Builds maestro-shell

## Additional information
