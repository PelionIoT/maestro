#!/usr/bin/env bash

# exit on error to make errors more visible
set -e

# Load go environment variables
. /etc/profile.d/envvars.sh

# Generate identity certificates to use to connect to Pelion
if [ ! -d ${PELION_EDGE_SRC} ]; then
	mkdir -p ${PELION_EDGE_SRC}
fi
cd ${PELION_EDGE_SRC}

if [ ! -d edge-utils ]; then
    git clone git@github.com:armPelionEdge/edge-utils.git
    cd edge-utils/debug_scripts/get_new_gw_identity/developer_gateway_identity
    chmod +x generate_self_signed_certs.sh
    # TODO swap out ACCOUNTID and DEVICEID with real IDs, not fake ones
    OU=ACCOUNTID internalid=DEVICEID ./generate_self_signed_certs.sh $MAESTRO_CERTS
fi
