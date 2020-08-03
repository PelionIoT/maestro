#!/usr/bin/env bash

# exit on error to make errors more visible
set -e

# Load go environment variables
. /etc/profile.d/envvars.sh

# Import project devicedb
if [ ! -d ${PELION_EDGE_SRC} ]; then
	mkdir -p ${PELION_EDGE_SRC}
fi
cd ${PELION_EDGE_SRC}

if [ ! -d mbed-edge ]; then
	git clone git@github.com:ARMmbed/mbed-edge.git
fi
cd mbed-edge

git submodule update --init --recursive
mbed deploy --protocol=ssh

cp cmake/toolchains/mcc-linux-x86.cmake config/toolchain.cmake
cp /vagrant/vagrant/files/target.cmake config/target.cmake

sed -i 's!/dev/random!/dev/urandom!' lib/mbed-cloud-client/mbed-client-pal/Source/Port/Reference-Impl/OS_Specific/Linux/Board_Specific/TARGET_x86_x64/pal_plat_x86_x64.c

cp /vagrant/mbed_cloud_dev_credentials.c ./config/.

mkdir -p ./build
cd ./build
cmake -DCMAKE_BUILD_TYPE=Release -DTRACE_LEVEL=WARN -DFIRMWARE_UPDATE=OFF -DDEVELOPER_MODE=ON -DFACTORY_MODE=OFF -DBYOC_MODE=OFF -DTARGET_CONFIG_ROOT=$PELION_EDGE_SRC/mbed-edge/config -DMBED_CLOUD_DEV_UPDATE_ID=ON ..
make

sudo cp bin/edge-core /usr/sbin/edge-core
