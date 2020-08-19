#!/bin/bash

# Copyright (c) 2019, Arm Limited and affiliates.
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
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done

THIS_DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

pushd "${THIS_DIR}"

# we use go mod download instead of git-clone so that we obtain the version
# in go.mod
go mod download
GREASEGO_SRC=$(go list -f {{.Dir}} -m github.com/armPelionEdge/greasego)
cp -r ${GREASEGO_SRC} ./greasego
cd greasego
make

popd

