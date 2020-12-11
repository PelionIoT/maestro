// Generated source file.
// Edit files in 'src' folder
package processes

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
	"syscall"
	"unsafe"
)

const (
	EFD_SEMAPHORE = 00000001
	EFD_CLOEXEC   = 02000000
	EFD_NONBLOCK  = 00004000
)

type WakeupFd struct {
	fd_read  int // on Linux, with eventfd, we only use this
	fd_write int
}

// Linux version:
func NewWakeupFd() (ret *WakeupFd, errno error) {
	fd, errno := eventfd(0, EFD_NONBLOCK)
	if errno == nil {
		ret = &WakeupFd{fd, 0}
	}
	return
}

func (this *WakeupFd) Wakeup(val uint64) (errno error) {
	len_int64 := int(8)
	_, _, e1 := syscall.RawSyscall(syscall.SYS_WRITE, uintptr(this.fd_read), uintptr(unsafe.Pointer(&val)), uintptr(len_int64))
	// check to make sure write was 8 bytes?
	if e1 != 0 {
		errno = e1
	}
	return
}

func (this *WakeupFd) ReadWakeup() (ret uint64, errno error) {
	var _p0 unsafe.Pointer
	_p0 = unsafe.Pointer(&ret)
	len_int64 := int(8)
	_, _, e1 := syscall.RawSyscall(syscall.SYS_READ, uintptr(this.fd_read), uintptr(_p0), uintptr(len_int64))
	//	r0
	//	n := uint64(r0)
	if e1 != 0 {
		errno = e1
	}
	return
}

// func WakeupFd(initval uint) (fd int, errno error) {
// 	fd, errno = eventfd(initval, 0)
// 	return
// }

// Wake's up a wakeup-use FD. Always write a non-zero integer
// to force the FD to wake up epoll(), select() etc.
// This 64-bit int size is to accomodate eventfd on Linux

// eventfd() is Linux specific
// C def: int eventfd(unsigned int initval, int flags);
func eventfd(initval uint, flags int) (fd int, errno error) {
	val, _, err := syscall.RawSyscall(syscall.SYS_EVENTFD2,
		uintptr(initval),
		uintptr(flags),
		0)
	if err == 0 {
		fd = int(val)
	} else {
		errno = err
	}
	return
}

// TODO: use a pipe() for other OS
