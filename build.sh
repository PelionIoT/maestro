#!/bin/bash

# Copyright (c) 2018, Arm Limited and affiliates.
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eEufo pipefail

on_err() { printf "%s: '%s' failed with '%s'\n" "${1}:${2}" "${3}" "$?" 1>&2; }
trap 'on_err "${BASH_SOURCE}" "${LINENO}" "${BASH_COMMAND}"' ERR


SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  SELF="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done

THIS_DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

pushd $THIS_DIR

if [ \( -n "$DEBUG" \) -o \( -n "$DEBUG2" \) ]; then
  echo "DEBUG ON"
  GOTAGS="-tags debug"
	make native.a-debug
else
	make native.a
fi

popd

# let's get the current commit, and make sure Version() has this.
COMMIT=`git rev-parse --short=7 HEAD`
DATE=`date`
sed -i -e "s/COMMIT_NUMBER/${COMMIT}/g" maestroutils/status.go 
sed -i -e "s/BUILD_DATE/${DATE}/g" maestroutils/status.go 

# highlight errors: https://serverfault.com/questions/59262/bash-print-stderr-in-red-color
# color()(set -o pipefail;"$@" 2>&1>&3|sed $'s,.*,\e[31m&\e[m,'>&2)3>&1

if [ "$1" != "preprocess_only" ]; then
	pushd $GOPATH/bin
  echo $PWD
	if [ ! -z "$TIGHT" ]; then
	    go build $GOTAGS -ldflags="-s -w" "$@" github.com/armPelionEdge/maestro/maestro 
	else
	    go build $GOTAGS "$@" github.com/armPelionEdge/maestro/maestro 
	fi
	popd
fi
