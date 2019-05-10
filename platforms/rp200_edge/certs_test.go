package rp200_edge

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
	"fmt"
	"log"
	"testing"

	"github.com/armPelionEdge/maestro/testtools"
)

var logger *testtools.NonProductionPrefixedLogger

func TestMain(m *testing.M) {
	logger = testtools.NewNonProductionPrefixedLogger("certs_test: ")
	m.Run()
}

func TestGenerateDeviceKey(t *testing.T) {
	key, cert, err := GeneratePlatformDeviceKeyNCert("012345678901234567890123456789ab", "012345678901234567890123456789ab", logger)
	if err == nil {
		fmt.Printf("Key:\n%s\n^^^^^^^^^^\n", key)
		fmt.Printf("Cert:\n%s\n^^^^^^^^^^\n", cert)
	} else {
		log.Fatal("Error from GeneratePlatformDeviceKeyNCert:", err)
	}

}
