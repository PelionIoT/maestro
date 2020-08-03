#!/usr/bin/env bash

# exit on error to make errors more visible
set -e

# Load go environment variables
. /etc/profile.d/envvars.sh

# Import project devicedb
go get github.com/armPelionEdge/devicedb || true

# Startup devicedb cloud in background
cd ${DEVICEDB_SRC}
cp /vagrant/vagrant/docker-compose.yaml ./docker-compose.yaml
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
