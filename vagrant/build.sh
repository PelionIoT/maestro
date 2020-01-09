#!/usr/bin/env bash

# Load go environment variables
. /etc/profile.d/envvars.sh
sudo usermod -aG docker $USER

# Import GO project maestro
go get github.com/armPelionEdge/maestro || true

# Build greaselib for grease_echo utility
# cd $GOPATH/src/github.com/armPelionEdge/greasego
# sed -i -e 's/make libgrease.a-server/make all/g' build-deps.sh

# Go to newly created maestro directory
cd $MAESTRO_SRC

# Build maestro dependencies
./build-deps.sh

# Apply maestro patch
git config --global user.email "fake@email.com"
git config --global user.name "Vagrant User"
git am /tmp/0001-Fake-devicedb-running-on-local-machine.patch

# Build maestro
DEBUG=1 DEBUG2=1 ./build.sh

# Create maestro dummy config if config does not exist
[ -f maestro.config ] || \
echo 'network:
    interfaces:
        - if_name: eth1
          exists: replace
          dhcpv4: true
          hw_addr: "{{ARCH_ETHERNET_MAC}}"
config_end: true
' >> maestro.config

# Import GO project maestro-shell
go get github.com/armPelionEdge/maestro-shell || true
go install github.com/armPelionEdge/maestro-shell

# Set go get to use SSH instead of https due to the private devicedb repo
git config --global url.git@github.com:.insteadOf https://github.com/

# Import project devicedb
cd $MAESTRO_SRC/..
go get github.com/armPelionEdge/devicedb

# Create directory for devicedb certificates
mkdir -p devicedb/edgeconfig
