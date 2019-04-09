package maestro

// Copyright (c) 2018, Arm Limited and affiliates.
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
DEBUG(	"runtime")
DEBUG( 	"fmt")
)

func DumpMemStats() {
    DEBUG(var stats runtime.MemStats)
    DEBUG(runtime.ReadMemStats(&stats))
    DEBUG_OUT("------ReadMemStats------\n")
    //DEBUG_OUT("%v+\n",stats)
    DEBUG_OUT("  ReadMemStats Alloc      %d\n",stats.Alloc)
    DEBUG_OUT("  ReadMemStats TotalAlloc %d\n",stats.TotalAlloc)
    DEBUG_OUT("  ReadMemStats Sys        %d\n",stats.Sys)
    DEBUG_OUT("  ReadMemStats Mallocs    %d\n",stats.Mallocs)
    DEBUG_OUT("  ReadMemStats HeapAlloc  %d\n",stats.HeapAlloc)
    DEBUG_OUT("  ")
	DEBUG_OUT("------ReadMemStats done------\n")            
}

/**
 * There appears to be a golang runtime bug in ReadMemStats() - make sure you dont call this 
 * in production:
 *  
 *runtime stack:
panic(0x702f40, 0xc420010140)
    /opt/go/src/runtime/panic.go:389 +0x6d2
runtime.ReadMemStats.func1()
    /opt/go/src/runtime/mstats.go:185 +0x2a

goroutine 7 [running]:
runtime.systemstack_switch()
    /opt/go/src/runtime/asm_amd64.s:252 fp=0xc4201b67a0 sp=0xc4201b6798
runtime.ReadMemStats(0xc4201b6810)
    /opt/go/src/runtime/mstats.go:186 +0x63 fp=0xc4201b67d0 sp=0xc4201b67a0
github.com/armPelionEdge/maestro.DumpMemStats()
    /home/ed/work/gostuff/src/github.com/armPelionEdge/maestro/debug.go:12 +0x67 fp=0xc4201b7f00 sp=0xc4201b67d0
github.com/armPelionEdge/maestro.(*Client).startTicker.func1(0xc42005a9c0)
    /home/ed/work/gostuff/src/github.com/armPelionEdge/maestro/httpSymphonyClient.go:235 +0x14d fp=0xc4201b7f98 sp=0xc4201b7f00
runtime.goexit()
    /opt/go/src/runtime/asm_amd64.s:2086 +0x1 fp=0xc4201b7fa0 sp=0xc4201b7f98
created by github.com/armPelionEdge/maestro.(*Client).startTicker
    /home/ed/work/gostuff/src/github.com/armPelionEdge/maestro/httpSymphonyClient.go:238 +0x68


 * 
 */