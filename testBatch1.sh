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

EXECPATH="$1"
ARG1="$EXECPATH"
ARG2="$2"
ARG3="$3"
ARG4="$4"
ARG5="$5"
ARG6="$6"
ARG7="$7"

PAYLOAD="{\"task_id\":\"aaaa\",\"ops\":[{\"type\":\"image\",\
\"op\":\"add\",\
\"image\":{\"job\":\"specialsauce\",\"version\":\"1.0.0\",\
\"size\":100011111111111111,\
\"url\":\"http://127.0.0.1:8080/stuff.zip\",\
\"checksum\":{\"type\":\"md5sum\",\"value\":\"asdsadsa\"},\"image_type\":\"devicejs_tarball\"}\
}]}"

#\"Daemonize\":true}"

echo curl -i \
    --unix-socket /tmp/maestroapi.sock http://localhost/ops \
    -H "Host: 127.0.0.1" \
    -H "Accept: application/json" \
    -X POST -d $PAYLOAD
#    -X POST -d '{"Path":"${EXECPATH}","Arguments":["$ARG1","$ARG2","$ARG3","$ARG4","$ARG5","$ARG6","$ARG7"]}'

curl -i \
    --unix-socket /tmp/maestroapi.sock http://localhost/ops \
    -H "Host: 127.0.0.1" \
    -H "Accept: application/json" \
    -X POST -d $PAYLOAD

#    -X POST -d '{"Path":"${EXECPATH}","Arguments":["$ARG1","$ARG2","$ARG3","$ARG4","$ARG5","$ARG6","$ARG7"]}'

#    -H "Accept: application/json" \
#    -H "X-HTTP-Method-Override: PUT" \
#     -X POST -d "Path":"$PATH","Arguments":"Tip 3","targetModule":"Target 3","configurationGroup":null,"name":"Configuration Deneme 3","description":null,"identity":"Configuration Deneme 3","version":0,"systemId":3,"active":true


# Notes:

# the http unix-socket is requires root normally, for security reasons

# you will need curl 7.40 or later, and libcurl 7.40 or later.
# if you install the newer one on Ubuntu 14.04, use the LD_PRELOAD trick like such
# sudo LD_PRELOAD="/usr/lib/libcurl.so.4.4.0" ./startProcessTest.sh
