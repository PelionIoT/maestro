package maestroSpecs

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

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

// Composite configurations
// used for setting up a system to a particular state

// ConfigChangeHook should be implemented if change should happen
// for a given config struct. A single ConfigChangeHook instance is assigned
// to a config group.
// For clarity, the order of execution for the ConfigChangeHook interface is:
// [Analysis Run]  --> No changes (no action)
//   ||
//   ||  See one or more changes
//   \/
// (once) ChangesStart() [ called only once, for a given analysis ]
//   ||
//   ||
//   \/
// (one or more calls) SawChange() [ called once for every field which changed ]
//   ||
//   ||
//   \/
// (once) ChangesComplete()  [ called once ]
type ConfigChangeHook interface {
	// ChangesStart is called before reporting any changes via multiple calls to SawChange. It will only be called
	// if there is at least one change to report
	ChangesStart(configgroup string)
	// SawChange is called whenever a field changes. It will be called only once for each field which is changed.
	// It will always be called after ChangesStart is called
	// If SawChange return true, then the value of futvalue will replace the value of current value
	// Index is valid if the value is >=0, in that case it indicates the index of the item in a slice
	SawChange(configgroup string, fieldchanged string, futvalue interface{}, curvalue interface{}, index int) (acceptchange bool)
	// ChangesComplete is called when all changes for a specific configgroup tagname
	// If ChangesComplete returns true, then all changes in that group will be assigned to the current struct
	ChangesComplete(configgroup string) (acceptallchanges bool)
}

type ConfigAnalyzer struct {
	// thread safe map
	configMap sync.Map // maps config group to ConfigChangeHook
	configTag string
}

// NewConfigAnalyzer returns a new, empty ConfigAnalyzer object
// which will operate off the struct tag of 'configtag'
func NewConfigAnalyzer(configtag string) (ret *ConfigAnalyzer) {
	ret = new(ConfigAnalyzer)
	ret.configTag = configtag
	return
}

// AddHook associated a config group label with a ConfigChangeHook
func (a *ConfigAnalyzer) AddHook(configgroup string, hook ConfigChangeHook) {
	a.configMap.Store(configgroup, hook)
}

// some helper functions

func findField(identifier string) (found bool, ret reflect.StructField) {
	return
}

func getTagKeyValue(field reflect.StructField, keyname string) (found bool, ret string) {
	return
}

type changes struct {
	fieldnames []string // the list of all fields which changed
	futvals    []reflect.Value
	curvals    []reflect.Value
	//	curValues   []reflect.Value
	configgroup string // which config group this is
	index	[]int; // this is valid if the value is >=0, in that case it indicates the index of the item in a slice
}

func (a *ConfigAnalyzer) callGroupChanges(c *changes) {
	hooki, ok := a.configMap.Load(c.configgroup)

	if ok {
		hook, ok2 := hooki.(ConfigChangeHook)
		if ok2 {
			hook.ChangesStart(c.configgroup)
			for n, fieldname := range c.fieldnames {
				
				takeit := hook.SawChange(c.configgroup, fieldname, c.futvals[n].Interface(), c.curvals[n].Interface(), c.index[n])
				if takeit {
					c.curvals[n].Set(c.futvals[n])
				}
			}
			takeall := hook.ChangesComplete(c.configgroup)
			if takeall {
				for n := range c.curvals {
					c.curvals[n].Set(c.futvals[n])
				}
			}
		}
	}
}

func (a *ConfigAnalyzer) DiffChanges(current interface{}, future interface{}) (identical bool, noaction bool, allchanges map[string]*changes, err error) {
	noaction = true
	identical = true
	allchanges = make(map[string]*changes)
	var compareStruct func(prefix string, cur interface{}, fut interface{}, index int) (errinner error)
	compareStruct = func(prefix string, cur interface{}, fut interface{}, index int) (errinner error) {
		// loop through - if see struct, call compare struct again, with 'prefix' as the field name of the struct

		// first ensure its a struct or pointer to struct, and dereference if needed
		var currType reflect.Type
		var currValue reflect.Value
		//		var futType reflect.Type
		var futValue reflect.Value
		kind := reflect.ValueOf(cur).Kind()
		if kind == reflect.Ptr {
			currType = reflect.TypeOf(cur).Elem()
			currValue = reflect.ValueOf(cur).Elem()
		} else if kind == reflect.Struct {
			if len(prefix) < 1 {
				// we can't set the values of the struct if it's passed by value. Must have pointer
				errinner = fmt.Errorf("need ptr to struct not struct: @ %s", prefix)
				return
			}

			// XXX Work around, make a pointer to the value
			// XXX reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()
			// no - we fixed this below with .Addr().Interface()
			currType = reflect.TypeOf(cur)
			//			currValue = reflect.ValueOf(cur)
			//			currValue = reflect.NewAt(currType, unsafe.Pointer(reflect.ValueOf(cur).UnsafeAddr())).Elem()
			//			currValue = reflect.NewAt(currType, unsafe.Pointer(&cur)).Elem()
			currValue = reflect.ValueOf(cur)
		}

		kind = currValue.Kind()
		if kind != reflect.Struct {
			errinner = errors.New(fmt.Sprintf("invalid type for current val, expected Struct, actual: %v", kind))
			return
		}
		kind = reflect.ValueOf(fut).Kind()
		if kind == reflect.Ptr {
			futValue = reflect.ValueOf(fut).Elem()
		} else if kind == reflect.Struct {
			futValue = reflect.ValueOf(fut)
		}
		kind = futValue.Kind()
		if kind != reflect.Struct {
			errinner = errors.New("invalid type for future val")
			return
		}

		//fmt.Printf("Vals: \n\tfut %v kind: %s \n\tcurr %v kind: %s\n", futValue, reflect.ValueOf(futValue).Kind(), currValue, reflect.ValueOf(currValue).Kind())
		// using the current struct, walk through each field
		//		assignToStruct := reflect.ValueOf(opts).Elem()
		for i := 0; i < currType.NumField(); i++ {
			field := currType.Field(i)
			fieldval := currValue.FieldByName(field.Name)
			futval := futValue.FieldByName(field.Name)
			if !futval.IsValid() {
				// so the future struct does not even have this field
				identical = false
				continue
			}
			k := fieldval.Kind()
			kf := futval.Kind()

			// deref pointers
			if k == reflect.Ptr {
				if fieldval.IsNil() {
					// can't fill in a struct which is nil
					// if we have a nil PTR to a struct, let's create the struct
					//					t := reflect.TypeOf().Elem()
					//					t := reflect.TypeOf(fieldval.Elem().Interface())
					t := field.Type
					newstruct := reflect.New(t.Elem())
					if newstruct.IsValid() && fieldval.CanSet() {
						fieldval.Set(newstruct.Convert(t))
					} else {
						fmt.Printf("Error - could not create / assign valid stuct\n")
						continue
					}
				}
				fieldval = reflect.ValueOf(fieldval.Interface()).Elem()
				k = fieldval.Kind()
			}
			if kf == reflect.Ptr {
				futval = reflect.ValueOf(futval.Interface()).Elem()
				kf = futval.Kind()
			}

			pre := prefix
			if len(pre) > 0 {
				pre += "."
			}

			if k != kf {
				identical = false
				continue
			}
			if k == reflect.Struct {
				// it is crtical to pass in as an Interface which is an Address
				e := compareStruct(pre+field.Name, fieldval.Addr().Interface(), futval.Addr().Interface(), index)
				if e != nil {
					errinner = e
					return
				}
				continue
			}
			// For slices we treat the slice as a single value,
			// unless the slice is a slice of struct or ptr to struct
			if k == reflect.Slice {
				changed := false
				curLen := fieldval.Len()
				N := futval.Len()
				if curLen != N {
					changed = true
				}
				expand := false
				for i := 0; i < N; i++ {
					indexval := futval.Index(i)
					var curindexval reflect.Value
					if i < curLen { // if current still has elements...
						curindexval = fieldval.Index(i)
					} else {
						changed = true
						expand = true
					}
					typ := indexval.Kind()
					if typ == reflect.Ptr {
						typ2 := indexval.Elem().Kind()
						if typ2 == reflect.Struct {
							if expand {
								curindexval = reflect.New(indexval.Elem().Type())
								fieldval.Set(reflect.Append(fieldval, curindexval))
							}
							err2 := compareStruct(fmt.Sprintf("%s[%d]", pre+field.Name, i), curindexval.Interface(), indexval.Interface(), i)
							if err2 != nil {
								fmt.Printf("Error on compareStruct: %s\n", err2.Error())
							}
							continue
						}
					} else if typ == reflect.Struct {
						if expand {
							// we can't expand a slice of static structs can we?
							//The new one version has a struct but the old version doesnt, so try to create a dummy struct for comparison
							curindexval = reflect.New(indexval.Type())
						}
						err2 := compareStruct(fmt.Sprintf("%s[%d]", pre+field.Name, i), curindexval.Interface(), indexval.Addr().Interface(), i)
						if err2 != nil {
							fmt.Printf("Error on compareStruct: %s\n", err2.Error())
						}
						continue
					}
					// its a slice of primitive types or strings
					// if the future slice is larger than the current, its changed
					if expand {
						changed = true
						break
					} else {
						if indexval.Interface() != curindexval.Interface() {
							changed = true
							break
						}
					}
					//					break
					//					fmt.Println(s.Index(i))
				}
				if changed {
					// entire slice is changed
					// different values - add to list
					group, ok := field.Tag.Lookup(a.configTag)
					if ok {
						c, ok2 := allchanges[group]
						if !ok2 {
							c = new(changes)
							allchanges[group] = c
							c.configgroup = group
						}
						c.fieldnames = append(c.fieldnames, pre+field.Name)
						c.futvals = append(c.futvals, futval)
						c.curvals = append(c.curvals, fieldval)
						c.index = append(c.index, index)
					}
				}
			} else {
				if !futval.IsValid() {
					fmt.Printf("futval is not valid!\n")
				}
				if futval.Interface() != fieldval.Interface() {
					identical = false
					noaction = false
					// different values - add to list
					group, ok := field.Tag.Lookup(a.configTag)
					if ok {
						c, ok2 := allchanges[group]
						if !ok2 {
							c = new(changes)
							allchanges[group] = c
							c.configgroup = group
						}
						c.fieldnames = append(c.fieldnames, pre+field.Name)
						c.futvals = append(c.futvals, futval)
						c.curvals = append(c.curvals, fieldval)
						c.index = append(c.index, index)
					}
				}
			}
		}

		return
	}

	//	kind := reflect.ValueOf(current).Kind()

	err = compareStruct("", current, future, -1)
	if err != nil {
		return
	}

	return
}

// CallChanges does a comparison between current and future items. Current and future should be the same
// type and be a struct. This struct should use struct tags on specific fields, of the format: CONFIGTAG:"SOMEID"
// where CONFIGTAG is a specific label used throughout the whole struct, to identify a field which belongs to a config group
// of SOMEID, and referenced with a ConfigChangeHook type handed to ConfigAnalyzer by the AddHook function. The overall
// structs being compared may have multiple SOMEIDs but should always use the same CONFIGTAG value.
// The struct tags themselves may be combined with other key/values used for json or YAML encoding or anything else, space separated, which is
// compatible with the reflect.Lookup function (typical convention for multiple values in a struct tag)
// Given that this labeling via struct tags is done, then CallChanges will compare 'current' to 'future' by value, for each
// field in the struct. It will then, after a full comparison is done, call the ConfigChangeHook's methods. See ConfigChangeHook for
// how these methods are called.
//
// NOTE: Technically current and future can have different types, but must have fields with the same names to be compared. The function
// will only look at field names which are in 'current' and which are public, and which have identical types.
func (a *ConfigAnalyzer) CallChanges(current interface{}, future interface{}) (identical bool, noaction bool, err error) {
	
	identical, noaction, allchanges, err := a.DiffChanges(current, future) 

	// walk through changes, calling the callbacks as needed
	if !noaction {
		for _, c := range allchanges {
			a.callGroupChanges(c)
		}
	}

	return

}
