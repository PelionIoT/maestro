#!/usr/bin/env bash

# exit on error to make errors more visible
set -e

# Load go environment variables
. /etc/profile.d/envvars.sh

# Import GO project maestro-shell
go get github.com/armPelionEdge/maestro-shell || true

cd ${MAESTRO_SHELL_SRC}
./build-deps.sh
go build
go install github.com/armPelionEdge/maestro-shell
