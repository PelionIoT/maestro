package maestroSpecs

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
//	"errors"
)

type Error interface {
	GetHttpStatusCode() int
	GetErrorString() string
	GetDetail() string
}

type APIError struct {
	HttpStatusCode int
	ErrorString string
	Detail string
}

func (err *APIError) Error() string {
	return err.ErrorString + " -- " + err.Detail
} 

func (err *APIError) GetHttpStatusCode() int {
	return err.HttpStatusCode
}

func (err *APIError) GetErrorString() string {
	return err.ErrorString
}

func (err *APIError) GetDetail() string {
	return err.Detail
}

