package templates

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
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/PelionIoT/mustache"
)

const ARCH_PREFIX = "ARCH_"

type TemplateVarDictionary struct {
	Map map[string]string
}

func NewTemplateVarDictionary() (ret *TemplateVarDictionary) {
	ret = new(TemplateVarDictionary)
	ret.Map = make(map[string]string)
	return
}

// AddTaggedStructArch adds a structure of tagged items in a struct. Only string fields are added,
// and only public fields. An example would be:
// type Example struct {
//     Stuff string `dict:"STUFF"`
// }
// In this case the tag "ARCH_STUFF" would be added, with whatever value it's assigned.
func (dict *TemplateVarDictionary) AddTaggedStructArch(strct interface{}) (numadded int, err error) {
	if strct == nil {
		err = errors.New("nil value passed")
		return
	}
	kind := reflect.ValueOf(strct).Kind()
	var reflectType reflect.Type
	var reflectValue reflect.Value
	if kind == reflect.Ptr {
		reflectType = reflect.TypeOf(strct).Elem()
		//		DEBUG_OUT2("ONE.1 (ptr)\n")
		reflectValue = reflect.ValueOf(strct).Elem()
		//		DEBUG_OUT2("TWO (ptr)\n")
	} else {
		reflectType = reflect.TypeOf(strct)
		//		DEBUG_OUT2("ONE.1\n")
		reflectValue = reflect.ValueOf(strct)
		//		DEBUG_OUT2("TWO\n")
	}

	if reflectType.Kind() != reflect.Struct {
		err = errors.New("invalid type passed")
		return
	}
	//	assignToStruct := reflect.ValueOf(opts).Elem()
	for i := 0; i < reflectType.NumField(); i++ {
		field := reflectType.Field(i)
		fieldval := reflectValue.FieldByName(field.Name)
		fieldType := field.Type
		if alias, ok := field.Tag.Lookup("dict"); ok {
			if fieldType.Kind() == reflect.String {
				if len(fieldval.String()) > 0 {
					dict.AddArch(alias, fieldval.String())
					numadded++
				}
			}
		}
	}

	return
}

// AddTaggedStruct adds a structure of tagged items in a struct. Only string fields are added,
// and only public fields. An example would be:
// type Example struct {
//     Stuff string `dict:"STUFF"`
// }
// In this case the tag "STUFF" would be added, with whatever value it's assigned.
func (dict *TemplateVarDictionary) AddTaggedStruct(strct interface{}) (numadded int, err error) {
	if strct == nil {
		err = errors.New("nil value passed")
		return
	}
	kind := reflect.ValueOf(strct).Kind()
	var reflectType reflect.Type
	var reflectValue reflect.Value
	if kind == reflect.Ptr {
		reflectType = reflect.TypeOf(strct).Elem()
		//		DEBUG_OUT2("ONE.1 (ptr)\n")
		reflectValue = reflect.ValueOf(strct).Elem()
		//		DEBUG_OUT2("TWO (ptr)\n")
	} else {
		reflectType = reflect.TypeOf(strct)
		//		DEBUG_OUT2("ONE.1\n")
		reflectValue = reflect.ValueOf(strct)
		//		DEBUG_OUT2("TWO\n")
	}

	if reflectType.Kind() != reflect.Struct {
		err = errors.New("invalid type passed")
		return
	}
	//	assignToStruct := reflect.ValueOf(opts).Elem()
	for i := 0; i < reflectType.NumField(); i++ {
		field := reflectType.Field(i)
		fieldval := reflectValue.FieldByName(field.Name)
		fieldType := field.Type
		if alias, ok := field.Tag.Lookup("dict"); ok {
			if fieldType.Kind() == reflect.String {
				if len(fieldval.String()) > 0 {
					dict.Add(alias, fieldval.String())
					numadded++
				}
			}
		}
	}

	return
}

func (dict *TemplateVarDictionary) Add(key string, value string) {
	dict.Map[key] = value
}

func (dict *TemplateVarDictionary) AddArch(key string, value string) {
	dict.Map[ARCH_PREFIX+key] = value
}

func (dict *TemplateVarDictionary) Del(key string) {
	delete(dict.Map, key)
}

func (this *TemplateVarDictionary) Get(key string) (ret string, ok bool) {
	ret, ok = this.Map[key]
	if ok {
		ret = mustache.Render(ret, this.Map)
	}
	return
}

func (this *TemplateVarDictionary) Render(input string) (output string) {
	output = mustache.Render(input, this.Map)
	// do it again, to handle meta vars which hand meta-vars inside them
	output = mustache.Render(output, this.Map)
	output = mustache.Render(output, this.Map)
	return
}

func (this *TemplateVarDictionary) DumpAsString() (output string) {
	var b strings.Builder
	for k, v := range this.Map {
		fmt.Fprintf(&b, "[%s]=%s\n", k, v)
	}
	output = b.String()
	return
}

const (
	TEMPLATEERROR_UNKNOWN     = iota
	TEMPLATEERROR_BAD_FILE    = iota
	TEMPLATEERROR_PERMISSIONS = iota
	TEMPLATEERROR_NO_TEMPLATE = iota
)

var job_error_map = map[int]string{
	0: "unknown",
	TEMPLATEERROR_BAD_FILE:    "TEMPLATEERROR_BAD_FILE",
	TEMPLATEERROR_PERMISSIONS: "TEMPLATEERROR_PERMISSIONS",
	TEMPLATEERROR_NO_TEMPLATE: "TEMPLATEERROR_NO_TEMPLATE",
}

type TemplateError struct {
	TemplateName string
	Code         int
	ErrString    string
}

func (this *TemplateError) Error() string {
	s := job_error_map[this.Code]
	return "JobError: " + this.TemplateName + ":" + s + " (" + strconv.Itoa(this.Code) + ") " + this.ErrString
}
