#!/usr/bin/env bash

# exit on error to make errors more visible
set -e

# Load go environment variables
. /etc/profile.d/envvars.sh

# Setup default email and name for github. Needed to apply patches
git config --global user.email "fake@email.com"
git config --global user.name "Vagrant User"

# Set go get to use SSH instead of https due to the private devicedb repo
git config --global url.git@github.com:.insteadOf https://github.com/

# Import GO project maestro
go get github.com/armPelionEdge/maestro || true

# Go to newly created maestro directory to check out the
# proper version
cd $MAESTRO_SRC
# Add the locally synced maestro folder as a remote.  This folder
# is synced from the host system into the VM by Vagrant.
MAESTRO_SYNC_SRC=/vagrant
git remote show vagranthost &>/dev/null || git remote add vagranthost ${MAESTRO_SYNC_SRC}
git fetch vagranthost
git reset --hard $(git -C ${MAESTRO_SYNC_SRC} rev-parse HEAD)

# apply any uncommitted changes from the synced folder
if ! git -C ${MAESTRO_SYNC_SRC} diff --quiet; then
    git reset --hard HEAD
    git -C ${MAESTRO_SYNC_SRC} diff | git apply
fi

# Build maestro dependencies
./build-deps.sh

# Build maestro
DEBUG=1 DEBUG2=1 ./build.sh

# Import GO project maestro-shell
go get github.com/armPelionEdge/maestro-shell || true
cd $MAESTRO_SRC/../maestro-shell
./build-deps.sh
go build
go install github.com/armPelionEdge/maestro-shell

# Generate identity certificates to use to connect to Pelion
cd $MAESTRO_SRC/..
if [ ! -d edge-utils ]; then
    git clone git@github.com:armPelionEdge/edge-utils.git
    cd edge-utils/debug_scripts/get_new_gw_identity/developer_gateway_identity
    chmod +x generate_self_signed_certs.sh
    # TODO swap out ACCOUNTID and DEVICEID with real IDs, not fake ones
    OU=ACCOUNTID internalid=DEVICEID ./generate_self_signed_certs.sh $MAESTRO_CERTS
fi

# Import project devicedb
go get github.com/armPelionEdge/devicedb || true

# Startup devicedb cloud in background
cd ${DEVICEDB_SRC}
cp ${MAESTRO_SRC}/vagrant/docker-compose.yaml ./docker-compose.yaml
docker-compose up -d

# Generate devicedb certs and identity file
cd ${DEVICEDB_SRC}/hack
if [ ! -d certs ]; then
    mkdir -p certs
    ./generate-certs-and-identity.sh certs
fi

# Add device and site information to the cloud
./compose-cloud-add-device.sh certs

# Restart devicedb edge
sudo systemctl restart devicedb_edge

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
cd $MAESTRO_SRC
if [ ! -f maestro.config ]; then
    SYMPHONY_HOST="gateways.mbedcloudstaging.net"
    SYMPHONY_CLIENT_CRT=$(cat $MAESTRO_CERTS/device_cert.pem)
    SYMPHONY_CLIENT_KEY=$(cat $MAESTRO_CERTS/device_private_key.pem)

    cp /vagrant/vagrant/maestro.config maestro.config
    sed -i "s|{{DEVICE_ID}}|$DEVICE_ID|g" maestro.config
    sed -i "s|{{DEVICEDB_SRC}}|$DEVICEDB_SRC|g" maestro.config

    yq w -i maestro.config symphony.host -- "$SYMPHONY_HOST"
    yq w -i maestro.config symphony.client_key -- "$SYMPHONY_CLIENT_KEY"
    yq w -i maestro.config symphony.client_cert -- "$SYMPHONY_CLIENT_CRT"
fi
