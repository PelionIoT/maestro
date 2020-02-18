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
git checkout $(cat /tmp/githash.txt)

# Build maestro dependencies
./build-deps.sh

# Apply maestro patch
git am /tmp/0001-PATCH-Fake-devicedb-running-on-local-machine.patch

# Build maestro
DEBUG=1 DEBUG2=1 ./build.sh

# Import GO project maestro-shell
go get github.com/armPelionEdge/maestro-shell || true
go install github.com/armPelionEdge/maestro-shell

# Generate identity certificates to use to connect to Pelion
cd $MAESTRO_SRC/..
git clone git@github.com:armPelionEdge/edge-utils.git
cd edge-utils/debug_scripts/get_new_gw_identity/developer_gateway_identity
chmod +x generate_self_signed_certs.sh
# TODO swap out ACCOUNTID and DEVICEID with real IDs, not fake ones
OU=ACCOUNTID internalid=DEVICEID ./generate_self_signed_certs.sh $MAESTRO_CERTS

# Import project devicedb
cd $MAESTRO_SRC/..
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
if [ ! -f maestro.config ]; then
    SYMPHONY_HOST="gateways.mbedcloudstaging.net"
    SYMPHONY_CLIENT_CRT=$(cat $MAESTRO_CERTS/device_cert.pem)
    SYMPHONY_CLIENT_KEY=$(cat $MAESTRO_CERTS/device_private_key.pem)

    mv /tmp/maestro.config maestro.config
    sed -i "s|{{DEVICE_ID}}|$DEVICE_ID|g" maestro.config
    sed -i "s|{{DEVICEDB_SRC}}|$DEVICEDB_SRC|g" maestro.config

    yq w -i maestro.config symphony.host -- "$SYMPHONY_HOST"
    yq w -i maestro.config symphony.client_key -- "$SYMPHONY_CLIENT_KEY"
    yq w -i maestro.config symphony.client_cert -- "$SYMPHONY_CLIENT_CRT"
fi