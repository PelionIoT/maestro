#!/usr/bin/env bash

# exit on error to make errors more visible
set -e

# Load go environment variables
. /etc/profile.d/envvars.sh

/vagrant/vagrant/components/build_edge_utils.sh
/vagrant/vagrant/components/build_edge.sh
/vagrant/vagrant/components/build_devicedb.sh
/vagrant/vagrant/components/build_maestro_shell.sh
/vagrant/vagrant/components/build_maestro.sh
