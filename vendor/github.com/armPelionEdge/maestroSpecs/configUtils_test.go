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

// Composite configurations
// used for setting up a system to a particular state

// Some test objects:
// 'magicTag' is the tag we are using to assign config group changes
// 'wizardItems' is one of a number of config groups

import (
	"fmt"
	"log"
	"testing"
)

type FriendsOfHarry struct {
	Hermione      string `magicTag:"groupDumbledore"`
	Ron           string `magicTag:"groupDumbledore"`
	Nobody        string
	DudeWithBeard string `magicTag:"groupDumbledore"`
}

type WizardFood struct {
	Item string `magicTag:"fooditems"`
	Qty  int    `magicTag:"fooditems"`
}

type WizardStuff struct {
	NumberOfPotions int `magicTag:"wizardItems"`
}

type testStruct1 struct {
	AnInt      int            `magicTag:"groupMagic"`
	AString    string         `magicTag:"groupMagic"`
	Dumbledore string         `magicTag:"groupDumbledore"`
	Harry      string         `magicTag:"groupDumbledore"`
	Friends    FriendsOfHarry `magicTag:"groupDumbledore"`
	Stuff      *WizardStuff   `magicTag:"wizardItems"`
}

type testStruct2 struct {
	AnInt      int            `magicTag:"groupMagic"`
	AString    string         `magicTag:"groupMagic"`
	Dumbledore string         `magicTag:"groupDumbledore"`
	Harry      string         `magicTag:"groupDumbledore"`
	Friends    FriendsOfHarry `magicTag:"groupDumbledore"`
	Stuff      *WizardStuff   `magicTag:"wizardItems"`
	Food       []*WizardFood
}

type testStruct3 struct {
	AnInt      int            `magicTag:"groupMagic"`
	AString    string         `magicTag:"groupMagic"`
	Dumbledore string         `magicTag:"groupDumbledore"`
	Harry      string         `magicTag:"groupDumbledore"`
	Friends    FriendsOfHarry `magicTag:"groupDumbledore"`
	Stuff      *WizardStuff   `magicTag:"wizardItems"`
	Food       []WizardFood   `magicTag:"food"`
}

type sortOfTestStruct1 struct {
	AnInt      int
	AString    string
	Dumbledore string
	Harry      string
	Friends    FriendsOfHarry
}

type almostTestStruct1 struct {
	AnInt      int            `magicTag:"groupMagic"`
	Dumbledore string         `magicTag:"groupDumbledore"`
	Harry      string         `magicTag:"groupDumbledore"`
	Friends    FriendsOfHarry `magicTag:"groupDumbledore"`
	Stuff      *WizardStuff   `magicTag:"wizardItems"`
}

func TestMain(m *testing.M) {
	m.Run()
}

type testHook struct {
	a func(g string)
	b func(g string, f string, v interface{}, v2 interface{}, index int) bool
	c func(g string) bool
}

func (hook *testHook) ChangesStart(g string) {
	if hook.a != nil {
		hook.a(g)
	}
}

func (hook *testHook) SawChange(g string, field string, curv interface{}, futv interface{}, index int) (ret bool) {
	if hook.b != nil {
		ret = hook.b(g, field, curv, futv, index)
	}
	return
}

func (hook *testHook) ChangesComplete(g string) (ret bool) {
	if hook.c != nil {
		ret = hook.c(g)
	}
	return
}

func (hook *testHook) OnChangesStart(p func(string)) {
	hook.a = p
}

func (hook *testHook) OnSawChange(p func(g string, field string, curv interface{}, futv interface{}, index int) bool) {
	hook.b = p
}

func (hook *testHook) OnChangesComplete(p func(g string) bool) {
	hook.c = p
}

type testHookItems struct {
}

func TestBasicConf1(t *testing.T) {
	fmt.Printf("Test Basic1\n")

	a := NewConfigAnalyzer("magicTag")

	if a == nil {
		log.Fatal("Failed to create config analyzer object")
	}

	// ChangesStart(configgroup string)
	// // SawChange is called whenever a field changes. It will be called only once for each field which is changed.
	// // It will always be called after ChangesStart is called
	// SawChange(configgroup string, fieldchanged string, value interface{})
	// // ChangesComplete is called when all changes for a specific configgroup tagname
	// ChangesComplete(configgroup string)

	changesN := 0
	startsN := 0

	hookMagic := new(testHook)

	hookMagic.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookMagic\n", g)
		log.Fatal("Shold not be called - no change here")
	})

	hookMagic.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		fmt.Printf("SawChange( %s ) hookMagic: %s: %+v --> %+v\n", g, field, curv, futv)
		changesN++
		return true
	})

	a.AddHook("groupMagic", hookMagic)

	hookDum := new(testHook)

	hookDum.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookDum\n", g)
		startsN++
	})

	hookDum.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		fmt.Printf("SawChange( %s ) hookDum: %s: %+v --> %+v\n", g, field, curv, futv)
		changesN++
		return true
	})

	a.AddHook("groupDumbledore", hookDum)

	var s1 testStruct1
	var s2 testStruct1
	s1.AString = "funny"
	s2.AString = "funny"
	s1.AnInt = 3
	s2.AnInt = 3
	s1.Dumbledore = "yeap"
	s2.Dumbledore = "yeap"
	s1.Harry = "harry"
	s2.Harry = "notharry"

	same, noaction, err := a.CallChanges(s1, s2)

	if err != nil {
		fmt.Printf("Error from CallOnChanges: %s\n", err.Error())
		return
	} else {
		log.Fatal("CallChanges should fail. Not a pointer")
		fmt.Printf("ret %+v %+v\n", same, noaction)
	}

	if changesN != 1 {
		log.Fatal("Saw invalid amount of changes.")
	}
	if startsN != 1 {
		log.Fatal("Saw invalid amount of start changes.")
	}

	fmt.Printf("s1.Harry = %s\n", s1.Harry)

	if s1.Harry != "notharry" {
		log.Fatal("Failed to transfer value")
	}

}

// as above but with pointers to structs
func TestBasicConf1Ptr(t *testing.T) {
	fmt.Printf("Test Basic1\n")

	a := NewConfigAnalyzer("magicTag")

	if a == nil {
		log.Fatal("Failed to create config analyzer object")
	}

	// ChangesStart(configgroup string)
	// // SawChange is called whenever a field changes. It will be called only once for each field which is changed.
	// // It will always be called after ChangesStart is called
	// SawChange(configgroup string, fieldchanged string, value interface{})
	// // ChangesComplete is called when all changes for a specific configgroup tagname
	// ChangesComplete(configgroup string)

	changesN := 0
	startsN := 0

	hookMagic := new(testHook)

	hookMagic.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookMagic\n", g)
		log.Fatal("Shold not be called - no change here")
	})

	hookMagic.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookMagic: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookMagic: %s\n", g, field)
		changesN++
		return true
	})

	a.AddHook("groupMagic", hookMagic)

	hookDum := new(testHook)

	hookDum.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookDum\n", g)
		startsN++
	})

	hookDum.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookDum: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookDum: %s\n", g, field)
		changesN++
		return true
	})

	a.AddHook("groupDumbledore", hookDum)

	s1 := new(testStruct1)
	s2 := new(testStruct1)
	s1.AString = "funny"
	s2.AString = "funny"
	s1.AnInt = 3
	s2.AnInt = 3
	s1.Dumbledore = "yeap"
	s2.Dumbledore = "yeap"
	s1.Harry = "harry"
	s2.Harry = "notharry"

	same, noaction, err := a.CallChanges(s1, s2)

	if err != nil {
		log.Fatal("Error from CallOnChanges:", err.Error())
	} else {
		fmt.Printf("ret %+v %+v\n", same, noaction)
	}

	if changesN != 1 {
		log.Fatal("Saw invalid amount of changes.")
	}
	if startsN != 1 {
		log.Fatal("Saw invalid amount of start changes.")
	}

	fmt.Printf("s1.Harry = %s\n", s1.Harry)

	if s1.Harry != "notharry" {
		log.Fatal("Failed to transfer value")
	}

}

func TestDeeperConf1Ptr(t *testing.T) {
	fmt.Printf("Test Basic1\n")

	a := NewConfigAnalyzer("magicTag")

	if a == nil {
		log.Fatal("Failed to create config analyzer object")
	}

	// ChangesStart(configgroup string)
	// // SawChange is called whenever a field changes. It will be called only once for each field which is changed.
	// // It will always be called after ChangesStart is called
	// SawChange(configgroup string, fieldchanged string, value interface{})
	// // ChangesComplete is called when all changes for a specific configgroup tagname
	// ChangesComplete(configgroup string)

	changesN := 0
	startsN := 0

	hookMagic := new(testHook)

	hookMagic.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookMagic\n", g)
		//		log.Fatal("Shold not be called - no change here")
		startsN++
	})

	hookMagic.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookMagic: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookMagic: %s\n", g, field)
		changesN++
		return true
	})

	a.AddHook("groupMagic", hookMagic)

	hookDum := new(testHook)

	hookDum.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookDum\n", g)
		startsN++
	})

	hookDum.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookDum: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookDum: %s\n", g, field)
		changesN++
		return true
	})

	a.AddHook("groupDumbledore", hookDum)

	s1 := new(testStruct1)
	s2 := new(testStruct1)
	s1.AString = "funny"
	s2.AString = "funny"
	s1.AnInt = 3
	s2.AnInt = 5
	s1.Dumbledore = "yeap"
	s2.Dumbledore = "yeap"
	s1.Harry = "harry"
	s2.Harry = "notharry"

	s1.Friends.Hermione = "hermi"
	s2.Friends.Hermione = "hermy"

	same, noaction, err := a.CallChanges(s1, s2)

	if err != nil {
		log.Fatal("Error from CallOnChanges:", err.Error())
	} else {
		fmt.Printf("ret %+v %+v\n", same, noaction)
	}

	fmt.Printf("s1: %+v\n", s1)

	if changesN != 3 {
		log.Fatal("Saw invalid amount of changes.")
	}
	if startsN != 2 {
		log.Fatal("Saw invalid amount of start changes.")
	}

	fmt.Printf("s1.Harry = %s\n", s1.Harry)

	if s1.Harry != "notharry" {
		log.Fatal("Failed to transfer value")
	}

}

func TestDeeperConf2Ptr(t *testing.T) {
	fmt.Printf("Test Basic1\n")

	a := NewConfigAnalyzer("magicTag")

	if a == nil {
		log.Fatal("Failed to create config analyzer object")
	}

	// ChangesStart(configgroup string)
	// // SawChange is called whenever a field changes. It will be called only once for each field which is changed.
	// // It will always be called after ChangesStart is called
	// SawChange(configgroup string, fieldchanged string, value interface{})
	// // ChangesComplete is called when all changes for a specific configgroup tagname
	// ChangesComplete(configgroup string)

	changesN := 0
	startsN := 0

	hookMagic := new(testHook)

	hookMagic.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookMagic\n", g)
		//		log.Fatal("Shold not be called - no change here")
		startsN++
	})

	hookMagic.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookMagic: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookMagic: %s\n", g, field)
		changesN++
		return true
	})

	a.AddHook("groupMagic", hookMagic)

	hookStuff := new(testHook)

	hookStuff.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookStuff\n", g)
		//		log.Fatal("Shold not be called - no change here")
		startsN++
	})

	hookStuff.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookMagic: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookStuff: %s\n", g, field)
		changesN++
		return true
	})
	a.AddHook("wizardItems", hookStuff)

	hookDum := new(testHook)

	hookDum.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookDum\n", g)
		startsN++
	})

	hookDum.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookDum: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookDum: %s\n", g, field)
		changesN++
		return true
	})

	a.AddHook("groupDumbledore", hookDum)

	s1 := new(testStruct1)
	s2 := new(testStruct1)
	s1.AString = "funny"
	s2.AString = "funny"
	s1.AnInt = 3
	s2.AnInt = 5
	s1.Dumbledore = "yeap"
	s2.Dumbledore = "yeap"
	s1.Harry = "harry"
	s2.Harry = "notharry"

	s1.Friends.Hermione = "hermi"
	s2.Friends.Hermione = "hermy"
	s1.Stuff = new(WizardStuff)
	s2.Stuff = new(WizardStuff)

	s1.Friends.Nobody = "what"
	s2.Friends.Nobody = "nuttn"

	s1.Stuff.NumberOfPotions = 3
	s2.Stuff.NumberOfPotions = 7

	same, noaction, err := a.CallChanges(s1, s2)

	if err != nil {
		log.Fatal("Error from CallOnChanges:", err.Error())
	} else {
		fmt.Printf("ret %+v %+v\n", same, noaction)
	}

	fmt.Printf("s1: %+v\n", s1)
	fmt.Printf("s1.Stuff: %+v\n", s1.Stuff)

	fmt.Printf("startsN = %d  changesN = %d\n", startsN, changesN)

	if changesN != 4 {
		log.Fatal("Saw invalid amount of changes.")
	}
	if startsN != 3 {
		log.Fatal("Saw invalid amount of start changes.")
	}

	if s1.Friends.Nobody != "what" {
		log.Fatal("No tagged field should not change.")
	}

	fmt.Printf("s1.Harry = %s\n", s1.Harry)

	if s1.Harry != "notharry" {
		log.Fatal("Failed to transfer value")
	}

}

func TestNonCongruentStructs(t *testing.T) {
	fmt.Printf("TestNonCongruentStructs\n")

	a := NewConfigAnalyzer("magicTag")

	if a == nil {
		log.Fatal("Failed to create config analyzer object")
	}

	// ChangesStart(configgroup string)
	// // SawChange is called whenever a field changes. It will be called only once for each field which is changed.
	// // It will always be called after ChangesStart is called
	// SawChange(configgroup string, fieldchanged string, value interface{})
	// // ChangesComplete is called when all changes for a specific configgroup tagname
	// ChangesComplete(configgroup string)

	changesN := 0
	startsN := 0

	hookMagic := new(testHook)

	hookMagic.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookMagic\n", g)
		log.Fatal("Shold not be called - no change here")
	})

	hookMagic.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookMagic: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookMagic: %s\n", g, field)
		changesN++
		return true
	})

	a.AddHook("groupMagic", hookMagic)

	hookDum := new(testHook)

	hookDum.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookDum\n", g)
		startsN++
	})

	hookDum.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookDum: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookDum: %s\n", g, field)
		changesN++
		return true
	})

	a.AddHook("groupDumbledore", hookDum)

	s1 := new(testStruct1)
	s2 := new(sortOfTestStruct1)
	s1.AString = "funny"
	s2.AString = "funny"
	s1.AnInt = 3
	s2.AnInt = 3
	s1.Dumbledore = "yeap"
	s2.Dumbledore = "yeap"
	s1.Harry = "harry"
	s2.Harry = "notharry"

	same, noaction, err := a.CallChanges(s1, s2)

	if err != nil {
		log.Fatal("Error from CallOnChanges:", err.Error())
	} else {
		fmt.Printf("ret %+v %+v\n", same, noaction)
	}

	if changesN != 1 {
		log.Fatal("Saw invalid amount of changes.")
	}
	if startsN != 1 {
		log.Fatal("Saw invalid amount of start changes.")
	}

	fmt.Printf("s1.Harry = %s\n", s1.Harry)

	if s1.Harry != "notharry" {
		log.Fatal("Failed to transfer value")
	}

}

func TestFillInNilStruct(t *testing.T) {
	fmt.Printf("TestFillInNilStruct\n")

	a := NewConfigAnalyzer("magicTag")

	if a == nil {
		log.Fatal("Failed to create config analyzer object")
	}

	// ChangesStart(configgroup string)
	// // SawChange is called whenever a field changes. It will be called only once for each field which is changed.
	// // It will always be called after ChangesStart is called
	// SawChange(configgroup string, fieldchanged string, value interface{})
	// // ChangesComplete is called when all changes for a specific configgroup tagname
	// ChangesComplete(configgroup string)

	changesN := 0
	startsN := 0

	hookMagic := new(testHook)

	hookMagic.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookMagic\n", g)
		log.Fatal("Shold not be called - no change here")
	})

	hookMagic.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookMagic: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookMagic: %s\n", g, field)
		changesN++
		return true
	})

	a.AddHook("groupMagic", hookMagic)

	hookDum := new(testHook)

	hookDum.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookDum\n", g)
		startsN++
	})

	hookDum.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookDum: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookDum: %s\n", g, field)
		changesN++
		return true
	})

	a.AddHook("groupDumbledore", hookDum)

	hookWiz := new(testHook)

	hookWiz.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookWiz\n", g)
		startsN++
	})

	hookWiz.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookDum: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookWiz: %s\n", g, field)
		v, ok := futv.(int)
		if ok {
			if v != 7 {
				log.Fatal("Wrong value for future value: ", v)
			}
		} else {
			log.Fatal("cast failed in OnSawChange for WizardStuff")
		}
		changesN++
		return true
	})

	a.AddHook("wizardItems", hookWiz)

	s1 := new(testStruct1)
	s2 := new(testStruct1)
	s1.AString = "funny"
	s2.AString = "funny"
	s1.AnInt = 3
	s2.AnInt = 3
	s1.Dumbledore = "yeap"
	s2.Dumbledore = "yeap"
	s1.Harry = "harry"
	s2.Harry = "notharry"

	//	s1.Stuff = new(WizardStuff)
	s2.Stuff = new(WizardStuff)

	//	s1.Stuff.NumberOfPotions = 3
	s2.Stuff.NumberOfPotions = 7

	same, noaction, err := a.CallChanges(s1, s2)

	if err != nil {
		log.Fatal("Error from CallOnChanges:", err.Error())
	} else {
		fmt.Printf("ret %+v %+v\n", same, noaction)
	}

	if changesN != 2 {
		log.Fatal("Saw invalid amount of changes.")
	}
	if startsN != 2 {
		log.Fatal("Saw invalid amount of start changes.")
	}

	fmt.Printf("s1.Harry = %s\n", s1.Harry)

	if s1.Stuff != nil {
		if s1.Stuff.NumberOfPotions != 7 {
			log.Fatal("Failed to transfer value")
		}
	} else {
		log.Fatal("Failed to create Stuff")
	}

	if s1.Harry != "notharry" {
		log.Fatal("Failed to transfer value")
	}

}

func TestTakeAllChanges(t *testing.T) {
	fmt.Printf("TestTakeAllChanges\n")

	a := NewConfigAnalyzer("magicTag")

	if a == nil {
		log.Fatal("Failed to create config analyzer object")
	}

	// ChangesStart(configgroup string)
	// // SawChange is called whenever a field changes. It will be called only once for each field which is changed.
	// // It will always be called after ChangesStart is called
	// SawChange(configgroup string, fieldchanged string, value interface{})
	// // ChangesComplete is called when all changes for a specific configgroup tagname
	// ChangesComplete(configgroup string)

	changesN := 0
	startsN := 0

	hookMagic := new(testHook)

	hookMagic.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookMagic\n", g)
		log.Fatal("Shold not be called - no change here")
	})

	hookMagic.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookMagic: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookMagic: %s\n", g, field)
		changesN++
		return true
	})

	a.AddHook("groupMagic", hookMagic)

	hookDum := new(testHook)

	hookDum.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookDum\n", g)
		startsN++
	})

	hookDum.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookDum: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookDum: %s\n", g, field)
		changesN++
		return false
	})

	hookDum.OnChangesComplete(func(s string) bool {
		fmt.Printf("ChangesComplete( %s ) hookDum", s)
		return true
	})

	a.AddHook("groupDumbledore", hookDum)

	hookWiz := new(testHook)

	hookWiz.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookWiz\n", g)
		startsN++
	})

	hookWiz.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookDum: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookWiz: %s\n", g, field)
		v, ok := futv.(int)
		if ok {
			if v != 7 {
				log.Fatal("Wrong value for future value: ", v)
			}
		} else {
			log.Fatal("cast failed in OnSawChange for WizardStuff")
		}
		changesN++
		return false
	})

	hookWiz.OnChangesComplete(func(s string) bool {
		fmt.Printf("ChangesComplete( %s )", s)
		return true
	})

	a.AddHook("wizardItems", hookWiz)

	s1 := new(testStruct1)
	s2 := new(testStruct1)
	s1.AString = "funny"
	s2.AString = "funny"
	s1.AnInt = 3
	s2.AnInt = 3
	s1.Dumbledore = "yeap"
	s2.Dumbledore = "yeap"
	s1.Harry = "harry"
	s2.Harry = "notharry"

	//	s1.Stuff = new(WizardStuff)
	s2.Stuff = new(WizardStuff)

	//	s1.Stuff.NumberOfPotions = 3
	s2.Stuff.NumberOfPotions = 7

	same, noaction, err := a.CallChanges(s1, s2)

	if err != nil {
		log.Fatal("Error from CallOnChanges:", err.Error())
	} else {
		fmt.Printf("ret %+v %+v\n", same, noaction)
	}

	if changesN != 2 {
		log.Fatal("Saw invalid amount of changes.")
	}
	if startsN != 2 {
		log.Fatal("Saw invalid amount of start changes.")
	}

	fmt.Printf("s1.Harry = %s\n", s1.Harry)

	if s1.Stuff != nil {
		if s1.Stuff.NumberOfPotions != 7 {
			log.Fatal("Failed to transfer value: NumberOfPotions")
		}
	} else {
		log.Fatal("Failed to create Stuff")
	}

	if s1.Harry != "notharry" {
		log.Fatal("Failed to transfer value: Harry")
	}

}

func TestFillInWithSlice(t *testing.T) {
	fmt.Printf("TestFillInWithArray\n")

	a := NewConfigAnalyzer("magicTag")

	if a == nil {
		log.Fatal("Failed to create config analyzer object")
	}

	// ChangesStart(configgroup string)
	// // SawChange is called whenever a field changes. It will be called only once for each field which is changed.
	// // It will always be called after ChangesStart is called
	// SawChange(configgroup string, fieldchanged string, value interface{})
	// // ChangesComplete is called when all changes for a specific configgroup tagname
	// ChangesComplete(configgroup string)

	changesN := 0
	startsN := 0
	indexFailed := false

	hookMagic := new(testHook)

	hookMagic.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookMagic\n", g)
		log.Fatal("Shold not be called - no change here")
	})

	hookMagic.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookMagic: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookMagic: %s\n", g, field)
		changesN++
		return true
	})

	a.AddHook("groupMagic", hookMagic)

	hookDum := new(testHook)

	hookDum.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookDum\n", g)
		startsN++
	})

	hookDum.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookDum: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookDum: %s\n", g, field)
		changesN++
		return true
	})

	a.AddHook("groupDumbledore", hookDum)

	hookWiz := new(testHook)

	hookWiz.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookWiz\n", g)
		startsN++
	})

	hookWiz.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookDum: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookWiz: %s\n", g, field)
		v, ok := futv.(int)
		if ok {
			if v != 7 {
				log.Fatal("Wrong value for future value: ", v)
			}
		} else {
			log.Fatal("cast failed in OnSawChange for WizardStuff")
		}
		changesN++
		return true
	})

	a.AddHook("wizardItems", hookWiz)

	hookFood := new(testHook)

	hookFood.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookFood\n", g)
		//		startsN++
	})

	hookFood.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookDum: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookFood: %s\n", g, field)
		//		changesN++
		return true
	})

	a.AddHook("food", hookFood)

	hookFoodItem := new(testHook)

	hookFoodItem.OnChangesStart(func(g string) {
		fmt.Printf("ChangesStart( %s ) hookFoodItem\n", g)
		//		startsN++
	})

	hookFoodItem.OnSawChange(func(g string, field string, futv interface{}, curv interface{}, index int) bool {
		//		fmt.Printf("SawChange( %s ) hookDum: %s: %+v --> %+v\n", g, field, curv, futv)
		fmt.Printf("SawChange( %s ) hookFoodItem: %s futval:%s index:%d\n", g, field, futv, index)
		//		changesN++
		if(futv == "apples") {
			if(index != 0) {
				indexFailed = true;
			}
		}
		if(futv == "oranges") {
			if(index != 1) {
				indexFailed = true;
			}
		}
		return true
	})

	a.AddHook("fooditems", hookFoodItem)

	s1 := new(testStruct2)
	s2 := new(testStruct2)
	s1.AString = "funny"
	s2.AString = "funny"
	s1.AnInt = 3
	s2.AnInt = 3
	s1.Dumbledore = "yeap"
	s2.Dumbledore = "yeap"
	s1.Harry = "harry"
	s2.Harry = "notharry"

	food := new(WizardFood)
	food.Item = "apples"
	food.Qty = 2

	s2.Food = append(s2.Food, food)
	food = new(WizardFood)
	food.Item = "oranges"
	food.Qty = 3
	s2.Food = append(s2.Food, food)

	//	s1.Stuff = new(WizardStuff)
	s2.Stuff = new(WizardStuff)

	//	s1.Stuff.NumberOfPotions = 3
	s2.Stuff.NumberOfPotions = 7

	same, noaction, err := a.CallChanges(s1, s2)

	if indexFailed {
		log.Fatal("Error from OnSawChange, index value mismatch")
	} else {
		fmt.Printf("indexFailed %+v\n", indexFailed)
	}

	if err != nil {
		log.Fatal("Error from CallOnChanges:", err.Error())
	} else {
		fmt.Printf("ret %+v %+v\n", same, noaction)
	}

	if changesN != 2 {
		log.Fatal("Saw invalid amount of changes.")
	}
	if startsN != 2 {
		log.Fatal("Saw invalid amount of start changes.")
	}

	fmt.Printf("s1.Harry = %s\n", s1.Harry)

	if s1.Stuff != nil {
		if s1.Stuff.NumberOfPotions != 7 {
			log.Fatal("Failed to transfer value")
		}
	} else {
		log.Fatal("Failed to create Stuff")
	}

	if s1.Harry != "notharry" {
		log.Fatal("Failed to transfer value")
	}

	if len(s1.Food) < 2 {
		log.Fatal("Invalid amount of food items")
	}

	fmt.Printf("s1.Food = %+v, %+v", s1.Food[0], s1.Food[1])
}
