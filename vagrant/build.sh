#!/usr/bin/env bash

# Load go environment variables
. /etc/profile.d/envvars.sh

# Setup default email and name for github. Needed to apply patches
git config --global user.email "fake@email.com"
git config --global user.name "Vagrant User"

# Set go get to use SSH instead of https due to the private devicedb repo
git config --global url.git@github.com:.insteadOf https://github.com/

# Import GO project maestro
go get github.com/armPelionEdge/maestro || true

# Go to newly created maestro directory
cd $MAESTRO_SRC

# Build maestro dependencies
./build-deps.sh

# Apply maestro patch
git am /tmp/0001-Fake-devicedb-running-on-local-machine.patch

# Build maestro
DEBUG=1 DEBUG2=1 ./build.sh

# Import GO project maestro-shell
go get github.com/armPelionEdge/maestro-shell || true
go install github.com/armPelionEdge/maestro-shell

# Import project devicedb
go get github.com/armPelionEdge/devicedb || true

# Generate devicedb certs and identity file
cd $DEVICEDB_SRC/hack
mkdir -p certs
./generate-certs-and-identity.sh certs

# Startup devicedb cloud in background
cp /tmp/docker-compose.yaml $DEVICEDB_SRC/docker-compose.yaml
docker-compose up -d

# Add device and site information to the cloud
./compose-cloud-add-device.sh certs

# Restart devicedb edge
sudo systemctl restart devicedb_edge

cd $MAESTRO_SRC

# Read Device ID from certs folder
DEVICE_ID=$(cat $DEVICEDB_SRC/hack/certs/device_id)

# Set a host for the device-id that devicedb uses
if grep -Fxq "unconfigured-devicedb-host" /etc/hosts
then
    sudo sed -i "s/unconfigured-devicedb-host/$DEVICE_ID/g" /etc/hosts || true
else
    echo "127.0.0.1       $DEVICE_ID" | sudo tee -a /etc/hosts
fi

# Create maestro dummy config if config does not exist
[ -f maestro.config ] || \
echo "network:
    interfaces:
        - if_name: eth1
          exists: replace
          dhcpv4: true
          hw_addr: \"{{ARCH_ETHERNET_MAC}}\"
devicedb_conn_config:
    devicedb_uri: \"https://$DEVICE_ID:9090\"
    devicedb_prefix: \"vagrant\"
    devicedb_bucket: \"lww\"
    relay_id: \"$DEVICE_ID\"
    ca_chain: \"$DEVICEDB_SRC/hack/certs/myCA.pem\"
config_end: true
" >> maestro.config
