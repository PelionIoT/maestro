#!/usr/bin/env bash

# Load go environment variables
. /etc/profile.d/envvars.sh

# Import GO project maestro
go get github.com/armPelionEdge/maestro || true

# Build greaselib for grease_echo utility
# cd $GOPATH/src/github.com/armPelionEdge/greasego
# sed -i -e 's/make libgrease.a-server/make all/g' build-deps.sh

# Go to newly created maestro directory
cd $MAESTRO_SRC

# Build maestro dependencies
./build-deps.sh

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