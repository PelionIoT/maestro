#!/usr/bin/env bash

# exit on error to make errors more visible
set -e

# Load go environment variables
. /etc/profile.d/envvars.sh

# Import GO project maestro
mkdir -p $MAESTRO_SRC
git clone https://github.com/PelionIoT/maestro $MAESTRO_SRC

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

mkdir -p bin
go build -o bin/maestro maestro/main.go

# Create maestro dummy config if config does not exist
cd $MAESTRO_SRC
if [ ! -f maestro.config ]; then
    cp /vagrant/vagrant/maestro.config maestro.config
fi
