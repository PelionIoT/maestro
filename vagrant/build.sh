#!/usr/bin/env bash

# exit on error to make errors more visible
set -e

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

# Create maestro dummy config if config does not exist
cd $MAESTRO_SRC
if [ ! -f maestro.config ]; then
    cp /vagrant/vagrant/maestro.config maestro.config
fi
