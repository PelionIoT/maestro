#!/bin/bash -e

. /etc/profile.d/envvars.sh

sudo ip addr flush dev eth1
sudo ip addr flush dev eth2

SITE_ID=$(cat $DEVICEDB_SRC/hack/certs/site_id)
DEVICE_ID=$(cat $DEVICEDB_SRC/hack/certs/device_id)

cp ./scripts/maestro.config.template $MAESTRO_SRC/maestro.config

sed -i "s|{{DEVICE_ID}}|$DEVICE_ID|g" $MAESTRO_SRC/maestro.config

sudo maestro &>/tmp/maestro.log &

echo "Waiting 5 seconds for maestro to boot up..."
sleep 5

echo "COMPLETE. maestro running in background! Please kill with 'sudo pkill maestro' when finished..."
