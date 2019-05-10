package utils

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
	"io"
	"io/ioutil"
)

// StringifyReaderWithLimit returns a string of the io.Reader handed to it
// but limited to max. If the string length >= max, then a "..." will
// be appended.
func StringifyReaderWithLimit(rdr io.Reader, max int) (ret string, err error) {
	var bytes []byte
	bytes, err = ioutil.ReadAll(io.LimitReader(rdr, int64(max)))
	ret = string(bytes)
	if len(ret) >= max {
		ret += "..."
	}
	return
}
