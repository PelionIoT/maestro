package events

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

import "sync"

type refCounter struct {
	refs int
	//	cond sync.Cond
	mutex sync.Mutex
}

func (counter *refCounter) ref() (ret int) {
	counter.mutex.Lock()
	counter.refs++
	ret = counter.refs
	counter.mutex.Unlock()
	return
}

func (counter *refCounter) unref() (ret int) {
	counter.mutex.Lock()
	counter.refs--
	ret = counter.refs
	counter.mutex.Unlock()
	return
}

func (counter *refCounter) lock() (ret int) {
	ret = counter.refs
	counter.mutex.Lock()
	return
}

func (counter *refCounter) unlock() (ret int) {
	ret = counter.refs
	counter.mutex.Unlock()
	return
}

func (counter *refCounter) refAndlock() (ret int) {
	counter.mutex.Lock()
	counter.refs++
	ret = counter.refs
	return
}

func (counter *refCounter) unrefAndlock() (ret int) {
	counter.mutex.Lock()
	counter.refs--
	ret = counter.refs
	return
}

func (counter *refCounter) refAndUnlock() (ret int) {
	counter.refs++
	ret = counter.refs
	counter.mutex.Unlock()
	return
}

func (counter *refCounter) unrefAndUnlock() (ret int) {
	counter.refs--
	ret = counter.refs
	counter.mutex.Unlock()
	return
}
