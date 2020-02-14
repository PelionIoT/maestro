package greasego

// see notes here on libtcmalloc issues: https://github.com/gperftools/gperftools/issues/39

/*
#cgo LDFLAGS: -L/usr/lib/x86_64-linux-gnu -L${SRCDIR}/deps/lib
#cgo LDFLAGS: -lgrease -lstdc++ -lm -ltcmalloc_minimal
#cgo CFLAGS: -I${SRCDIR}/deps/include DEBUG(-DDEBUG_BINDINGS)
#define GREASE_IS_LOCAL 1
#include <stdio.h>
#include <stdlib.h>
#include "grease_lib.h"
#include "bindings.h"
*/
import "C"

// import "C" has to be on it's own line, or go compiler freaks out
import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unsafe"
	//	"sync/atomic"
)

// no longer including these, as they are statically in libgrease: -luv -lTW
//-static-libgcc
// # cgo LDFLAGS: /usr/lib/x86_64-linux-gnu/libunwind-coredump.so.0

// I can't get this to work right now. I think when the fix for cgo command is put in it will:
// https://github.com/golang/go/issues/16651
/*
# cgo LDFLAGS: -L/usr/lib/gcc/x86_64-linux-gnu/4.8
# cgo LDFLAGS: -L/usr/lib/x86_64-linux-gnu -Wl,-Bdynamic /usr/lib/x86_64-linux-gnu/libunwind-coredump.so.0 -Wl,-Bstatic
# cgo LDFLAGS: -Wl,-whole-archive ${SRCDIR}/deps/lib/libtcmalloc_minimal.a -Wl,-no-whole-archive
# cgo LDFLAGS: /usr/lib/gcc/x86_64-linux-gnu/4.8.4/libstdc++.a ${SRCDIR}/deps/lib/libgrease.a ${SRCDIR}/deps/lib/libuv.a ${SRCDIR}/deps/lib/libTW.a ${SRCDIR}/deps/lib/libre2.a  -lstdc++ /usr/lib/x86_64-linux-gnu/libm.a
# cgo CFLAGS: -I${SRCDIR}/deps/include
# define GREASE_IS_LOCAL 1
# include "grease_lib.h"
*/

//
// Copyright (c) 2019 ARM Limited and affiliates.
//
// SPDX-License-Identifier: MIT
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//

const GREASE_LIB_OK int = 0
const GREASE_LIB_NOT_FOUND int = 0x01E00000

// This interface is for providing a special callback to called when the
// greaseLib starts
type GreaseLibStartCB func()

//type GreaseLibStartCB interface {
//	GreaseLib_start_cb()
//}

// GreaseError is used for error reporting from greaseLib, and is analagous
// to the same structure in the C library. An errornum of 0 means 'no error'
type GreaseError struct {
	Str   string
	Errno int
}

func ConvertCGreaseError(err *C.GreaseLibError) *GreaseError {
	if err == nil {
		return nil
	}
	ret := new(GreaseError)
	ret.Errno = int((*err)._errno)
	ret.Str = C.GoString(&err.errstr[0])
	return ret
}

//This generic interfaces represents places in the
// greaseLib where the C GreaseLibCallback(err,void *) is used, but no data is ever
// passed back with the void pointer
type GreaseLibCallbackNoData interface {
	greaseLibCallback(err GreaseError)
}

// Callback used for a callback which
type GreaseLibAddTargetCB func(err *GreaseError, optsId int, targId uint32)

// The GreaseLib structure represents a single instantiation of the
// library. For now, only one instantiation is supported.
type GreaseLib struct {
	_greaseLibStartCB GreaseLibStartCB
}

type GreaseLibCallbackEvent struct {
	data interface{}
	err  *GreaseError
}

// GetGreaseLibVersion returns the version of the underlying greaseLib.so
// shared library in use
func GetGreaseLibVersion() (ret string) {
	// unbelievable the amount of syntax juggling one needs to
	// simply send come char data back to Go. wtf.
	len := C.size_t(150)
	mem := C.malloc(len)
	schar := (*C.char)(unsafe.Pointer(mem))
	C.GreaseLib_getVersion(schar, C.int(len))
	ret = C.GoString(schar)
	C.free(mem)
	return
}

// The library currently only supports a single instance
// This variable tracks the singleton
var greaseInstance *GreaseLib = nil

func getGreaseLib() *GreaseLib {
	if greaseInstance == nil {
		greaseInstance = new(GreaseLib)
	}
	return greaseInstance
}

type GreaseLibTargetFileOpts struct {
	//	uint32 _enabledFlags
	//	uint32_t _enabledFlags;
	_binding        C.GreaseLibTargetFileOpts
	Mode            uint32 // permissions for file (recommend default)
	Flags           uint32 // file flags (recommend default)
	Max_files       uint32 // max # of files to maintain (rotation)
	Max_file_size   uint32 // max size for any one file
	Max_total_size  uint64 // max total size to maintain in rotation
	Rotate_on_start bool
}

// analgous to greaseLib GreaseLibTargetOpts
type GreaseLibTargetOpts struct {
	_binding       C.GreaseLibTargetOpts
	Delim          *string
	TTY            *string
	File           *string
	OptsId         int                      // filled in automatically
	TargetCB       GreaseLibTargetCB        // used if this is target is a callback
	FileOpts       *GreaseLibTargetFileOpts // nil if not used
	Format_pre     *string
	Format_time    *string
	Format_level   *string
	Format_tag     *string
	Format_origin  *string
	Format_post    *string
	Format_pre_msg *string
	Name           *string // not used by greaseLib - but used for a string reference name
	// for the target ID
	flags    uint32
	NumBanks uint32
}

const GREASE_JSON_ESCAPE_STRINGS uint32 = C.GREASE_JSON_ESCAPE_STRINGS

func TargetOptsSetFlags(opts *GreaseLibTargetOpts, flag uint32) {
	opts.flags |= flag
}

var nextOptsId uint32 = 0

//var mutexAddTargetMap = make(map[uint32]

type GreaseIdMap map[string]uint32

var DefaultLevelMap GreaseIdMap
var DefaultTagMap GreaseIdMap

// [string]:[target ID]
var TargetMap GreaseIdMap

// This interface is used for a 'callback target' in go. The
// greaseLibCallback(err,data) will be called with a 'data' string representing
// all log data since the last callback
type GreaseLibTargetCB func(err *GreaseError, data *TargetCallbackData)

// used temporarily when waiting for the callback when we call AddTarget
type addTargetCallbackData struct {
	targetName  string
	addTargetCB GreaseLibAddTargetCB
	targetCB    GreaseLibTargetCB
}

var addTargetCallbackMap map[int]*addTargetCallbackData
var addTargetCallbackMapMutex *sync.Mutex

type TargetCallbackData struct {
	buf    *C.GreaseLibBuf
	targId uint32
}

// after calling this function, the data, and any slice
// form GetBufferAsSlice() is invalid
func RetireCallbackData(data *TargetCallbackData) {
	C.GreaseLib_cleanup_GreaseLibBuf(data.buf)
}

func (data *TargetCallbackData) GetBufferAsSlice() []byte {
	// this technique avoids copying data - which would utterly suck
	// https://www.cockroachlabs.com/blog/the-cost-and-complexity-of-cgo/
	if data.buf != nil {
		_len := uint64((*data.buf).size) // get len out of C structure
		const maxLen = uint64(0x7fffffff)
		if _len > 0 {
			if _len < maxLen {
				return (*[maxLen]byte)(unsafe.Pointer(data.buf.data))[:_len:_len]
			} else {
				fmt.Printf("@GetBufferAsSlice sanity check: %+v %d %d\n", data, _len, uint64(data.buf.size))
				panic("OOPS - len is wrong")
				// fmt.Printf("OVERLOAD of zero-copy buffer - GetBufferAsSlice() %+v %d\n",data,_len)
				// return C.GoBytes(unsafe.Pointer(data.buf.data), C.int(data.buf.size))
			}
		} else {
			return []byte{}
		}
	} else {
		return []byte{}
	}
}

// used to map callback targets (targets which have or are a callback)
var targetCallbackMap map[uint32]GreaseLibTargetCB
var targetCallbackMapMutex *sync.Mutex

type GreaseLevel uint32

const GREASE_ALL_LEVELS GreaseLevel = 0xFFFFFFFF //C.GREASE_ALL_LEVELS

func init() {
	addTargetCallbackMap = map[int]*addTargetCallbackData{}
	addTargetCallbackMapMutex = new(sync.Mutex)
	targetCallbackMap = map[uint32]GreaseLibTargetCB{}
	targetCallbackMapMutex = new(sync.Mutex)

	TargetMap = GreaseIdMap{
		"default": C.GREASE_DEFAULT_TARGET_ID, // default target ID is always 0
	}

	// defined in grease_client.h
	DefaultLevelMap = GreaseIdMap{
		"log":     C.GREASE_LEVEL_LOG,
		"error":   C.GREASE_LEVEL_ERROR,
		"warn":    C.GREASE_LEVEL_WARN,
		"debug":   C.GREASE_LEVEL_DEBUG,
		"debug2":  C.GREASE_LEVEL_DEBUG2,
		"debug3":  C.GREASE_LEVEL_DEBUG3,
		"user1":   C.GREASE_LEVEL_USER1,
		"user2":   C.GREASE_LEVEL_USER2,
		"success": C.GREASE_LEVEL_SUCCESS,
		"info":    C.GREASE_LEVEL_INFO,
		"trace":   C.GREASE_LEVEL_TRACE,
	}

	// defined in grease_common_tags.h
	DefaultTagMap = GreaseIdMap{
		"stdout": C.GREASE_TAG_STDOUT,
		"stderr": C.GREASE_TAG_STDERR,
		"syslog": C.GREASE_TAG_SYSLOG,
		"kernel": C.GREASE_TAG_KERNEL,
		// deviceJS specific tags - defined in grease_lib.cc
		"console": C.GREASE_CONSOLE_TAG,
		"native":  C.GREASE_NATIVE_TAG,
		// grease_echo
		"grease-echo": C.GREASE_ECHO_TAG,
		// syslog common facility names:
		// these are defined in grease_lib.cc, and are only for server use. Client should not use
		// these tag unless they are logging through syslog() libc calls
		"sys-auth":     uint32(C.GREASE_RESERVED_TAGS_SYS_AUTH),
		"sys-authpriv": uint32(C.GREASE_RESERVED_TAGS_SYS_AUTHPRIV),
		"sys-cron":     uint32(C.GREASE_RESERVED_TAGS_SYS_CRON),
		"sys-daemon":   uint32(C.GREASE_RESERVED_TAGS_SYS_DAEMON),
		"sys-ftp":      uint32(C.GREASE_RESERVED_TAGS_SYS_FTP),
		"sys-kern":     uint32(C.GREASE_RESERVED_TAGS_SYS_KERN),
		"sys-lpr":      uint32(C.GREASE_RESERVED_TAGS_SYS_LPR),
		"sys-mail":     uint32(C.GREASE_RESERVED_TAGS_SYS_MAIL),
		"sys-mark":     uint32(C.GREASE_RESERVED_TAGS_SYS_MARK),
		"sys-security": uint32(C.GREASE_RESERVED_TAGS_SYS_SECURITY),
		"sys-syslog":   uint32(C.GREASE_RESERVED_TAGS_SYS_SYSLOG),
		"sys-user":     uint32(C.GREASE_RESERVED_TAGS_SYS_USER),
		"sys-uucp":     uint32(C.GREASE_RESERVED_TAGS_SYS_UUCP),
		"sys-local0":   uint32(C.GREASE_RESERVED_TAGS_SYS_LOCAL0),
		"sys-local1":   uint32(C.GREASE_RESERVED_TAGS_SYS_LOCAL1),
		"sys-local2":   uint32(C.GREASE_RESERVED_TAGS_SYS_LOCAL2),
		"sys-local3":   uint32(C.GREASE_RESERVED_TAGS_SYS_LOCAL3),
		"sys-local4":   uint32(C.GREASE_RESERVED_TAGS_SYS_LOCAL4),
		"sys-local5":   uint32(C.GREASE_RESERVED_TAGS_SYS_LOCAL5),
		"sys-local6":   uint32(C.GREASE_RESERVED_TAGS_SYS_LOCAL6),
		"sys-local7":   uint32(C.GREASE_RESERVED_TAGS_SYS_LOCAL7),
	}

	// this setups the default redirect callback to always be the above one.
	// So the bindings here will handle the close whenever a redirected stderr/stdout closes
	// if no specific callback was handed in at AddFDForStdout() / Stderr()
	C.GreaseLib_addDefaultRedirectorClosedCB(C.GreaseLibProcessClosedRedirect(C.greasego_childClosedFDCallback))

}

//func init() {
//
//}

//export do_startGreaseLib_cb
func do_startGreaseLib_cb() {
	_instance := getGreaseLib()
	if _instance._greaseLibStartCB != nil {
		_instance._greaseLibStartCB()
	}
}

// Start the library. The the cb.GreaseLib_start_cb() callback will be called
// upon start.
func StartGreaseLib(cb GreaseLibStartCB) {
	_instance := getGreaseLib()
	_instance._greaseLibStartCB = cb
	DEBUG(fmt.Printf("calling GreaseLib_start()\n"))
	C.GreaseLib_start((C.GreaseLibCallback)(unsafe.Pointer(C.greasego_startGreaseLibCB)))
}

// looks for a field in a struct with 'tag' and returns that
// field's reflect.Type
func findTypeByTag(tag string, in interface{}) reflect.Type {
	typ := reflect.TypeOf(in)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		found := field.Tag.Get("greaseType")
		//		fmt.Println("found greaseType tag of",found)
		if len(found) > 0 && strings.Compare(found, tag) == 0 {
			DEBUG(fmt.Println("Found template type of", field.Type, " - tag:", tag))
			return field.Type
		}
	}
	return nil
}

// return value = 0 means the target name doesn't exist
// return value > 0 is the ID of the target
func GetTargetId(name string) uint32 {
	return TargetMap[name]
}

func SetSelfOriginLabel(label string) {
	s := C.CString(label)
	C.greasego_setSelfOriginLabel(s)
	C.free(unsafe.Pointer(s))
}

func AddLevelLabel(val uint32, label string) {
	cstr := C.CString(label)
	clen := C.strlen(cstr)
	C.GreaseLib_addLevelLabel(C.uint32_t(val), cstr, C.int(clen))
	C.free(unsafe.Pointer(cstr))
}

func AddTagLabel(val uint32, label string) {
	cstr := C.CString(label)
	clen := C.strlen(cstr)
	C.GreaseLib_addTagLabel(C.uint32_t(val), cstr, C.int(clen))
	C.free(unsafe.Pointer(cstr))
}

func AddOriginLabel(val uint32, label string) {
	cstr := C.CString(label)
	clen := C.strlen(cstr)
	C.GreaseLib_addOriginLabel(C.uint32_t(val), cstr, C.int(clen))
	C.free(unsafe.Pointer(cstr))
}

// Assigns values to a struct based on StructTags of `greaseAssign` and `greaseType`
// Not that with string, this always assumes the structure it will fill will have a *string, not a string
func AssignFromStruct(opts interface{}, obj interface{}) { //, typ reflect.Type) {
	// recommended reading if not familiar with reflect: https://blog.golang.org/laws-of-reflection
	// first deal with all string fields
	typ := reflect.TypeOf(obj)
	assignToStruct := reflect.ValueOf(opts).Elem()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldval := reflect.ValueOf(obj).FieldByName(field.Name)
		fieldType := field.Type
		if alias, ok := field.Tag.Lookup("greaseAssign"); ok {
			assignToField := assignToStruct.FieldByName(alias)
			if fieldType.Kind() == reflect.String {
				//			if alias, ok := field.Tag.Lookup("greaseAssign"); ok {
				if alias != "" {
					//				fmt.Println("alias: ",alias)
					//				if(reflect.ValueOf(obj).FieldByName(alias).IsValid() ) {
					if fieldval.IsValid() {
						val := reflect.New(fieldType)
						val.Elem().Set(fieldval)
						if len(fieldval.String()) > 0 {
							DEBUG(fmt.Println("Will assign:", fieldval.String()))
							if reflect.ValueOf(opts).Elem().FieldByName(alias).CanSet() {
								DEBUG(fmt.Printf("Set string Ptr value to <%s>\n", val.Elem().String()))
								reflect.ValueOf(opts).Elem().FieldByName(alias).Set(val)
							} else {
								fmt.Println("ERROR: No valid field of name:", alias)
							}
						} else {
							//						fmt.Println("Skipping field",alias,"b/c was blank")
						}

					} else {
						//						fmt.Println("Skipping field",alias," not valid")
					}

				} else {
					//				fmt.Println("(blank)")
				}
				//			} else {
				//			fmt.Println("(not specified)")
				//			}
			} else {

				DEBUG(fmt.Println("here1"))

				switch field.Type.Kind() {
				case reflect.Bool:
					fallthrough
				case reflect.Int:
					fallthrough
				case reflect.Int16:
					fallthrough
				case reflect.Int32:
					fallthrough
				case reflect.Int64:
					fallthrough
				case reflect.Int8:
					fallthrough
				case reflect.Uint:
					fallthrough
				case reflect.Uint16:
					fallthrough
				case reflect.Uint32:
					fallthrough
				case reflect.Uint64:
					fallthrough
				case reflect.Uint8:

					//						if alias, ok := field.Tag.Lookup("greaseAssign"); ok {
					if alias != "" {
						//				fmt.Println("alias: ",alias)
						//				if(reflect.ValueOf(obj).FieldByName(alias).IsValid() ) {
						if fieldval.IsValid() {
							//							val := reflect.New(fieldType)
							//							val.Elem().Set(fieldval)
							if len(fieldval.String()) > 0 { // if it's a string, make sure it's not empty
								DEBUG(fmt.Println("Will assign:", fieldval.String(), "to", alias))
								if assignToField.IsValid() {
									DEBUG(fmt.Println("valid field"))
								}
								if assignToField.CanSet() {
									DEBUG(fmt.Printf("Set value to <%s>\n", fieldval.String()))
									assignToField.Set(fieldval)
								} else {
									fmt.Println("ERROR: No valid field of name:", alias)
								}
							} else {
								//						fmt.Println("Skipping field",alias,"b/c was blank")
							}

						} else {
							//						fmt.Println("Skipping field",alias," not valid")
						}

					} else {
						//				fmt.Println("(blank)")
					}
					//						}
				case reflect.Ptr:
					DEBUG(fmt.Println("PTR found - in reflection"))

					if fieldval.IsValid() {
						fieldval = fieldval.Elem()
						fieldType = fieldval.Elem().Type()
					}
					fallthrough

				case reflect.Struct:
					//						fmt.Println("@struct")
					strct_name := alias
					//						if strct_name, ok := field.Tag.Lookup("greaseAssign"); ok {
					//							fmt.Println("@struct w/ greaseAssign")

					if strct_name != "" {
						if strct_name != "" {
							strct_orig := reflect.ValueOf(obj).FieldByName(field.Name)
							if strct_orig.IsValid() {
								DEBUG(fmt.Println("@struct - recurse and New (", field.Name, " - ", strct_name, ")"))
								if reflect.ValueOf(opts).Elem().FieldByName(strct_name).IsValid() {
									if reflect.ValueOf(opts).Elem().FieldByName(strct_name).IsNil() {
										//										fmt.Println("but field is nil")
										typ := findTypeByTag(strct_name, obj) // I could not figure out a way to do this with pure reflection, so resorted to this
										if typ != nil {
											val := reflect.New(typ)
											reflect.ValueOf(opts).Elem().FieldByName(strct_name).Set(val) // assign newly created sub options (inner struct)
											//											inner_opts := val //reflect.ValueOf(opts).Elem().FieldByName(strct_name).Addr() // reflect.ValueOf(opts).Elem().FieldByName(strct_name).Elem()
											AssignFromStruct(val.Interface(), strct_orig.Interface()) //, strct_orig.Type())
										} else {
											fmt.Println("ERROR: could not find a template field for such greaseType label")
										}
										//										val := reflect.New(reflect.Indirect(reflect.ValueOf(opts).Elem().FieldByName(strct_name)).Type())
										//										reflect.ValueOf(opts).Elem().FieldByName(strct_name).Set(val)	// assign newly created sub options (inner struct)
									} else {
										inner_opts := reflect.ValueOf(opts).Elem().FieldByName(strct_name) // reflect.ValueOf(opts).Elem().FieldByName(strct_name).Elem()
										AssignFromStruct(inner_opts, strct_orig.Interface())               //, strct_orig.Type())
									}
								} else {
									DEBUG(fmt.Println("inner - not valid"))
								}
							}

						}

					} // end if strct_name
					//						}

				}
			}
		}

	}
	DEBUG(fmt.Println("exit assign"))
}

func convertOptsToCGreaseLib(opts *GreaseLibTargetOpts) {
	//	C.GreaseLib_init_GreaseLibTargetOpts(&opts._binding) // init it to the library defaults
	if opts.Delim != nil {
		opts._binding.delim = C.CString(*opts.Delim)
		opts._binding.len_delim = C.int(len(*opts.Delim))
	}
	if opts.TTY != nil {
		opts._binding.tty = C.CString(*opts.TTY)
	}
	if opts.File != nil {
		opts._binding.file = C.CString(*opts.File)
	}

	if opts.Format_pre != nil {
		opts._binding.format_pre = C.CString(*opts.Format_pre)
		opts._binding.format_pre_len = C.int(len(*opts.Format_pre))
	}
	if opts.Format_time != nil {
		opts._binding.format_time = C.CString(*opts.Format_time)
		opts._binding.format_time_len = C.int(len(*opts.Format_time))
	}
	if opts.Format_level != nil {
		opts._binding.format_level = C.CString(*opts.Format_level)
		opts._binding.format_level_len = C.int(len(*opts.Format_level))
	}
	if opts.Format_tag != nil {
		opts._binding.format_tag = C.CString(*opts.Format_tag)
		opts._binding.format_tag_len = C.int(len(*opts.Format_tag))
	}
	if opts.Format_origin != nil {
		opts._binding.format_origin = C.CString(*opts.Format_origin)
		opts._binding.format_origin_len = C.int(len(*opts.Format_origin))
	}
	if opts.Format_post != nil {
		opts._binding.format_post = C.CString(*opts.Format_post)
		opts._binding.format_post_len = C.int(len(*opts.Format_post))
	}
	if opts.Format_pre_msg != nil {
		opts._binding.format_pre_msg = C.CString(*opts.Format_pre_msg)
		opts._binding.format_pre_msg_len = C.int(len(*opts.Format_pre_msg))
	}
	if opts.NumBanks > 0 {
		opts._binding.num_banks = C.uint32_t(opts.NumBanks)
	}
	if opts.FileOpts != nil {
		opts._binding.fileOpts = C.GreaseLib_new_GreaseLibTargetFileOpts()
		if opts.FileOpts.Max_file_size > 0 {
			opts._binding.fileOpts.max_file_size = C.uint32_t(opts.FileOpts.Max_file_size)
			C.GreaseLib_set_flag_GreaseLibTargetFileOpts(opts._binding.fileOpts, C.GREASE_LIB_SET_FILEOPTS_ROTATE)
			C.GreaseLib_set_flag_GreaseLibTargetFileOpts(opts._binding.fileOpts, C.GREASE_LIB_SET_FILEOPTS_MAXFILESIZE)
		}
		if opts.FileOpts.Max_files > 0 {
			opts._binding.fileOpts.max_files = C.uint32_t(opts.FileOpts.Max_files)
			C.GreaseLib_set_flag_GreaseLibTargetFileOpts(opts._binding.fileOpts, C.GREASE_LIB_SET_FILEOPTS_ROTATE)
			C.GreaseLib_set_flag_GreaseLibTargetFileOpts(opts._binding.fileOpts, C.GREASE_LIB_SET_FILEOPTS_MAXFILES)
		}
		if opts.FileOpts.Max_total_size > 0 {
			opts._binding.fileOpts.max_total_size = C.uint64_t(opts.FileOpts.Max_total_size)
			C.GreaseLib_set_flag_GreaseLibTargetFileOpts(opts._binding.fileOpts, C.GREASE_LIB_SET_FILEOPTS_MAXTOTALSIZE)
		}
		if opts.FileOpts.Rotate_on_start {
			C.GreaseLib_set_flag_GreaseLibTargetFileOpts(opts._binding.fileOpts, C.GREASE_LIB_SET_FILEOPTS_ROTATEONSTART)
		}
		if opts.FileOpts.Flags > 0 {
			opts._binding.fileOpts.flags = C.uint32_t(opts.FileOpts.Flags)
		}
		if opts.FileOpts.Mode > 0 {
			opts._binding.fileOpts.mode = C.uint32_t(opts.FileOpts.Mode)
		}
	}
	C.GreaseLib_set_flag_GreaseLibTargetOpts(&opts._binding, C.uint32_t(opts.flags))
}

//export do_addTargetCB
func do_addTargetCB(err *C.GreaseLibError, info *C.GreaseLibStartedTargetInfo) {
	// TODO: convert to GreaseLibTargetOpts or number, fire correct callback
	optsid := int(0)
	//	fmt.Println("HERE1111 do_addTargetCB")
	if info != nil {
		//	fmt.Printf("opts -----------> %+v\n", *info)
		goerr := ConvertCGreaseError(err)
		if goerr != nil {
			fmt.Printf("Error on Callback: %d\n", goerr.Errno)
		}

		optsid = int((*info).optsId)
		//		fmt.Printf("HERE2222 do_addTargetCB %d\n",optsid)
		addTargetCallbackMapMutex.Lock()
		data := addTargetCallbackMap[optsid]
		addTargetCallbackMapMutex.Unlock()

		// save the TargetInfo into the TargetMap
		TargetMap[data.targetName] = uint32(info.targId)

		if data.targetCB != nil { // assign the target callback, if one is provided
			targetCallbackMapMutex.Lock()
			targetCallbackMap[uint32(info.targId)] = data.targetCB
			targetCallbackMapMutex.Unlock()
		}

		if data.addTargetCB != nil { // call the AddTarget callback
			// (the callback for the operation of adding a Target)
			data.addTargetCB(goerr, optsid, uint32(info.targId))
			addTargetCallbackMapMutex.Lock()
			delete(addTargetCallbackMap, optsid)
			addTargetCallbackMapMutex.Unlock()
		} else {
			//			fmt.Printf("NO CALLBACK FOUND. optsid: %d\n",optsid)
		}
	}
}

//export do_modifyDefaultTargetCB
func do_modifyDefaultTargetCB(err *C.GreaseLibError, info *C.GreaseLibStartedTargetInfo) {
	// TODO: convert to GreaseLibTargetOpts or number, fire correct callback
	optsid := int(0)
	//	fmt.Println("HERE1111 do_addTargetCB")
	if info != nil {
		//	fmt.Printf("opts -----------> %+v\n", *info)
		goerr := ConvertCGreaseError(err)
		if goerr != nil {
			fmt.Printf("Error on Callback: %d\n", goerr.Errno)
		}
		optsid = int((*info).optsId)
		//		fmt.Printf("HERE2222 do_addTargetCB %d\n",optsid)

		addTargetCallbackMapMutex.Lock()
		data := addTargetCallbackMap[optsid]
		addTargetCallbackMapMutex.Unlock()

		if data.targetCB != nil { // assign the target callback, if one is provided
			targetCallbackMapMutex.Lock()
			targetCallbackMap[uint32(info.targId)] = data.targetCB
			targetCallbackMapMutex.Unlock()
		}

		if data.addTargetCB != nil { // call the AddTarget callback
			// (the callback for the operation of adding a Target)
			data.addTargetCB(goerr, optsid, uint32(info.targId))
			addTargetCallbackMapMutex.Lock()
			delete(addTargetCallbackMap, optsid)
			addTargetCallbackMapMutex.Unlock()
		} else {
			//			fmt.Printf("NO CALLBACK FOUND. optsid: %d\n",optsid)
		}
	}
}

func NewGreaseLibTargetOpts() *GreaseLibTargetOpts {
	ret := new(GreaseLibTargetOpts)
	C.GreaseLib_init_GreaseLibTargetOpts(&ret._binding) // init it to the library defaults
	return ret
}

//export do_commonTargetCB
func do_commonTargetCB(err *C.GreaseLibError, d *C.GreaseLibBuf, targetId C.uint32_t) {
	// this gets called by C.greasego_commonTargetCallback
	targetCallbackMapMutex.Lock()
	cb := targetCallbackMap[uint32(targetId)]
	targetCallbackMapMutex.Unlock()
	if cb != nil {
		data := new(TargetCallbackData)
		data.buf = d
		data.targId = uint32(targetId)
		var goerr *GreaseError
		if err != nil {
			goerr = ConvertCGreaseError(err)
		}
		cb(goerr, data)
	}
}

func AddTarget(opts *GreaseLibTargetOpts, cb GreaseLibAddTargetCB) {
	convertOptsToCGreaseLib(opts)
	optid := int(opts._binding.optsId)
	// optid := atomic.AddUint32(&nextOptsId, 1)
	// opts._binding.optsId = C.int(optid) // that ID needs to be unique amongst threads

	dat := new(addTargetCallbackData)

	if opts.Name != nil {
		dat.targetName = *opts.Name
	} else if opts.File != nil {
		dat.targetName = *opts.File
	} else if opts.TTY != nil {
		dat.targetName = *opts.TTY
	} else {
		DEBUG(fmt.Println("WARN: Missing target name, so creating a default name"))
		dat.targetName = fmt.Sprintf("UnnamedTarget%d", optid)
	}
	DEBUG(fmt.Println("Adding target name: ", dat.targetName))

	if opts.TargetCB != nil {
		dat.targetCB = opts.TargetCB
		// assign the common callback
		opts._binding.targetCB = C.GreaseLibTargetCallback(C.greasego_commonTargetCB)
	}

	if cb != nil {
		dat.addTargetCB = cb
	}
	addTargetCallbackMapMutex.Lock()
	addTargetCallbackMap[optid] = dat
	addTargetCallbackMapMutex.Unlock()
	C.greasego_wrapper_addTarget(&(opts._binding)) // use the wrapper func
}

func ModifyDefaultTarget(opts *GreaseLibTargetOpts) int {
	convertOptsToCGreaseLib(opts)
	return int(C.GreaseLib_modifyDefaultTarget(&opts._binding)) // use the wrapper func
}

const GREASE_LIB_SET_FILEOPTS_MODE uint32 = 0x10000000
const GREASE_LIB_SET_FILEOPTS_FLAGS uint32 = 0x20000000
const GREASE_LIB_SET_FILEOPTS_MAXFILES uint32 = 0x40000000
const GREASE_LIB_SET_FILEOPTS_MAXFILESIZE uint32 = 0x80000000
const GREASE_LIB_SET_FILEOPTS_MAXTOTALSIZE uint32 = 0x01000000
const GREASE_LIB_SET_FILEOPTS_ROTATEONSTART uint32 = 0x02000000 // set if you want files to rotate on start
const GREASE_LIB_SET_FILEOPTS_ROTATE uint32 = 0x04000000        // set if you want files to rotate, if not set all other rotate options are skipped

func SetFileOpts(opts GreaseLibTargetOpts, flag uint32, val uint32) {
	if opts._binding.fileOpts == nil { // it's a C pointer there, but apparently this works
		opts._binding.fileOpts = C.GreaseLib_new_GreaseLibTargetFileOpts()
	}
	C.GreaseLib_set_flag_GreaseLibTargetFileOpts(opts._binding.fileOpts, C.uint32_t(val))
}

func SetupStandardLevels() int {
	return int(C.GreaseLib_setupStandardLevels())
}
func SetupStandardTags() int {
	return int(C.GreaseLib_setupStandardTags())
}

type GreaseLibFilter struct {
	_binding C.GreaseLibFilter
	_isInit  bool

	Origin uint32
	Tag    uint32
	Target uint32
	Mask   uint32

	//	_enabledFlags uint32
	//	origin uint32
	//	tag uint32
	//	target uint32
	//	mask uint32
	//	id uint32
	Format_pre          *string
	Format_post         *string
	Format_post_pre_msg *string
}

const GREASE_LIB_SET_FILTER_ORIGIN uint32 = 0x1
const GREASE_LIB_SET_FILTER_TAG uint32 = 0x2
const GREASE_LIB_SET_FILTER_TARGET uint32 = 0x4
const GREASE_LIB_SET_FILTER_MASK uint32 = 0x8

func NewGreaseLibFilter() *GreaseLibFilter {
	ret := new(GreaseLibFilter)
	C.GreaseLib_init_GreaseLibFilter(&ret._binding) // init it to the library defaults

	ret.Mask = uint32(ret._binding.mask)
	ret.Origin = uint32(ret._binding.origin)
	ret.Tag = uint32(ret._binding.tag)
	ret.Target = uint32(ret._binding.target)

	ret._isInit = true
	return ret
}

func SetFilterValue(filter *GreaseLibFilter, flag uint32, val uint32) {
	// GreaseLib_setvalue_GreaseLibFilter(GreaseLibFilter *opts,uint32_t flag,uint32_t val)
	C.GreaseLib_setvalue_GreaseLibFilter(&(filter._binding), C.uint32_t(flag), C.uint32_t(val))
}

func convertFilterToCGreaseLib(opts *GreaseLibFilter) {
	if !opts._isInit {
		C.GreaseLib_init_GreaseLibFilter(&opts._binding) // init it to the library defaults
	}
	if opts.Format_pre != nil {
		opts._binding.format_pre = C.CString(*opts.Format_pre)
		opts._binding.format_pre_len = C.int(len(*opts.Format_pre))
	}
	if opts.Format_post != nil {
		opts._binding.format_post = C.CString(*opts.Format_post)
		opts._binding.format_post_len = C.int(len(*opts.Format_post))
	}
	if opts.Format_post_pre_msg != nil {
		opts._binding.format_post = C.CString(*opts.Format_post_pre_msg)
		opts._binding.format_post_pre_msg_len = C.int(len(*opts.Format_post_pre_msg))
	}

}

func AddFilter(opts *GreaseLibFilter) int {
	convertFilterToCGreaseLib(opts)
	return int(C.GreaseLib_addFilter(&opts._binding))
}
func DisableFilter(opts *GreaseLibFilter) int {
	return int(C.GreaseLib_disableFilter(&opts._binding))
}
func EnableFilter(opts *GreaseLibFilter) int {
	return int(C.GreaseLib_enableFilter(&opts._binding))
}
func ModifyFilter(opts *GreaseLibFilter) int {
	return int(C.GreaseLib_modifyFilter(&opts._binding))
}
func DeleteFilter(opts *GreaseLibFilter) int {
	return int(C.GreaseLib_deleteFilter(&opts._binding))
}

const GREASE_LIB_SINK_UNIXDGRAM uint32 = 0x1
const GREASE_LIB_SINK_PIPE uint32 = 0x2
const GREASE_LIB_SINK_SYSLOGDGRAM uint32 = 0x3
const GREASE_LIB_SINK_KLOG uint32 = 0x4
const GREASE_LIB_SINK_KLOG2 uint32 = 0x5

type GreaseLibSink struct {
	_binding C.GreaseLibSink
	id       uint32
}

func NewGreaseLibSink(sinkType uint32, path *string) *GreaseLibSink {
	sink := new(GreaseLibSink)
	var temppath *C.char
	if path != nil {
		temppath = C.CString(*path)
	}
	C.GreaseLib_init_GreaseLibSink(&sink._binding, C.uint32_t(sinkType), temppath)
	return sink
}

func AddSink(sink *GreaseLibSink) int {
	ret := int(C.GreaseLib_addSink(&sink._binding))
	if ret == GREASE_LIB_OK {
		sink.id = uint32(sink._binding.id)
	}
	return ret
}

type GreaseLibProcessClosedRedirectCallback func(err *GreaseError, stream_type int, pid int)

var closedRedirectCB GreaseLibProcessClosedRedirectCallback

func AssignChildClosedFDCallback(cb GreaseLibProcessClosedRedirectCallback) {
	closedRedirectCB = cb
}

//export do_childClosedFDCallback
func do_childClosedFDCallback(err *C.GreaseLibError, stream_type C.int, fd C.int) {
	goerr := ConvertCGreaseError(err)
	if closedRedirectCB != nil {
		closedRedirectCB(goerr, int(stream_type), int(fd))
	}
}

func AddFDForStdout(fd int, originId uint32) {
	C.GreaseLib_addFDForStdout(C.int(fd), C.uint32_t(originId), C.GreaseLibProcessClosedRedirect(C.greasego_childClosedFDCallback))
}

func AddFDForStderr(fd int, originId uint32) {
	C.GreaseLib_addFDForStderr(C.int(fd), C.uint32_t(originId), C.GreaseLibProcessClosedRedirect(C.greasego_childClosedFDCallback))
}

func RemoveFDForStderr(fd int) {
	C.GreaseLib_removeFDForStderr(C.int(fd))
}
func RemoveFDForStdout(fd int) {
	C.GreaseLib_removeFDForStdout(C.int(fd))
}

func SetInternalTagName(name string) {

}

func SetInternalLogOrigin(originid uint32, name string) {
	//	internal_origin = C.uint32_t(originid)

}

func LogError(a ...interface{}) {
	out := fmt.Sprint(a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_self_meta_error, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogErrorf(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_self_meta_error, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogWarning(a ...interface{}) {
	out := fmt.Sprint(a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_self_meta_warning, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogWarningf(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_self_meta_warning, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogInfo(a ...interface{}) {
	out := fmt.Sprint(a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_self_meta_info, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogInfof(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_self_meta_info, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogDebug(a ...interface{}) {
	out := fmt.Sprint(a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_self_meta_debug, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogDebugf(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_self_meta_debug, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogSuccess(a ...interface{}) {
	out := fmt.Sprint(a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_self_meta_success, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogSuccessf(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_self_meta_success, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogError_noOrigin(a ...interface{}) {
	out := fmt.Sprint(a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_meta_error, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogErrorf_noOrigin(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_meta_error, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogWarning_noOrigin(a ...interface{}) {
	out := fmt.Sprint(a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_meta_warning, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogWarningf_noOrigin(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_meta_warning, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogInfo_noOrigin(a ...interface{}) {
	out := fmt.Sprint(a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_meta_info, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogInfof_noOrigin(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_meta_info, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogDebug_noOrigin(a ...interface{}) {
	out := fmt.Sprint(a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_meta_debug, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogDebugf_noOrigin(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_meta_debug, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogSuccess_noOrigin(a ...interface{}) {
	out := fmt.Sprint(a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_meta_success, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func LogSuccessf_noOrigin(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	cstr := C.CString(out)
	C.GreaseLib_logCharBuffer(&C.go_meta_success, cstr, C.int(len(out)))
	C.free(unsafe.Pointer(cstr))
}

func GetUnusedTagId() (goret uint32) {
	var ret C.uint32_t
	C.GreaseLib_getUnusedTagId(&ret)
	goret = uint32(ret)
	return
}

func GetUnusedOriginId() (goret uint32) {
	var ret C.uint32_t
	C.GreaseLib_getUnusedOriginId(&ret)
	goret = uint32(ret)
	return
}
