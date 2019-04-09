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

for pid in $(ps -ef | grep maestro | awk '{print $2}'); do
    if [ -f /proc/$pid/smaps ]; then
        echo "* Mem usage for PID $pid"     
        rss=$(awk 'BEGIN {i=0} /^Rss/ {i = i + $2} END {print i}' /proc/$pid/smaps)
        pss=$(awk 'BEGIN {i=0} /^Pss/ {i = i + $2 + 0.5} END {print i}' /proc/$pid/smaps)
        sc=$(awk 'BEGIN {i=0} /^Shared_Clean/ {i = i + $2} END {print i}' /proc/$pid/smaps)            
        sd=$(awk 'BEGIN {i=0} /^Shared_Dirty/ {i = i + $2} END {print i}' /proc/$pid/smaps)
        pc=$(awk 'BEGIN {i=0} /^Private_Clean/ {i = i + $2} END {print i}' /proc/$pid/smaps)
        pd=$(awk 'BEGIN {i=0} /^Private_Dirty/ {i = i + $2} END {print i}' /proc/$pid/smaps)
        echo "-- Rss: $rss kB" 
        echo "-- Pss: $pss kB"
        echo "Shared Clean $sc kB"
        echo "Shared Dirty $sd kB"
        echo "Private ${pd} + ${pc} kB"
    fi
done



