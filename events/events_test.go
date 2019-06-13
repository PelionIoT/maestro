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

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/armPelionEdge/httprouter"
	"github.com/armPelionEdge/maestro/storage"
	"github.com/boltdb/bolt"
)

func TestMain(m *testing.M) {
	m.Run()
}

type testStorageManager struct {
	db     *bolt.DB
	dbname string
}

const (
	TEST_DB_NAME = "testStorageManager.db"
)

func (this *testStorageManager) start() {
	this.dbname = TEST_DB_NAME
	db, err := bolt.Open(TEST_DB_NAME, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	this.db = db
}

func (this *testStorageManager) GetDb() *bolt.DB {
	return this.db
}

func (this *testStorageManager) GetDbName() string {
	return this.dbname
}

func (this *testStorageManager) RegisterStorageUser(user storage.StorageUser) {
	if this.db != nil {
		user.StorageInit(this)
		user.StorageReady(this)
	} else {
		log.Fatalf("test DB failed.")
	}
}

func (this *testStorageManager) shutdown(user storage.StorageUser) {
	user.StorageClosed(this)
}

type nuttin struct{}
type latch chan nuttin
type dat struct {
	x int
}

type dat2 struct {
	X int `json:"x"`
}

func newLatch(buf int) latch {
	return make(chan nuttin, buf)
}

func needlatch(c latch) {
	<-c
}

func latchon(c latch) {
	c <- nuttin{}
}

const stop_value = 0xF00D

func TestFanout1(t *testing.T) {
	_, id, err := MakeEventChannel("stuff", true, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	sub, err := SubscribeToChannel("stuff", 0)
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}

	// listen := func() {
	// }

	fmt.Printf("Starting listener...\n")

	hook := newLatch(1)

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			fmt.Println("listener() at top of loop.")
			ok, subchan := sub.GetChannel()
			if ok {
				select {
				case ev := <-subchan:
					sdata, ok := ev.Data.(*dat)
					if ok {
						fmt.Printf("listener: Got event %d\n", sdata.x)
						if sdata.x == stop_value {
							break listenLoop
						} else {
							if sdata.x != x+1 {
								log.Fatalf("Did not get correct data in event.")
							}
							x = sdata.x
						}
					} else {
						log.Fatalf("Failed type assertion")
					}
					latchon(hook)
					// default:
					//     latchon(hook)
				}
			} else {
				log.Fatalf("channel should be available. wrong.")
			}
			sub.ReleaseChannel()
		}
	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat{
			x: 1,
		}
		names := []string{"stuff"}
		fmt.Println("@SubmitEvent 1")
		//        needlatch(hook)
		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: 2,
		}
		needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: stop_value,
		}
		needlatch(hook)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

	})

}

// This version lets events drop
func TestFanout2(t *testing.T) {
	_, id, err := MakeEventChannel("stuff2", true, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	sub, err := SubscribeToChannel("stuff2", 0)
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}

	// listen := func() {
	// }

	fmt.Printf("Starting listener...\n")

	hook := newLatch(1)

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			fmt.Println("listener() at top of loop.")
			ok, subchan := sub.GetChannel()
			if ok {
				select {
				case ev := <-subchan:
					sdata, ok := ev.Data.(*dat)
					if ok {
						fmt.Printf("listener: Got event %d\n", sdata.x)
						if sdata.x == stop_value {
							break listenLoop
						} else {
							if sdata.x != x+1 {
								log.Fatalf("Did not get correct data in event.")
							}
							x = sdata.x
						}
					} else {
						log.Fatalf("Failed type assertion")
					}
					latchon(hook)
					// default:
					//     latchon(hook)
				}
			} else {
				log.Fatalf("channel should be available. wrong.")
			}
			sub.ReleaseChannel()
		}
	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat{
			x: 1,
		}
		names := []string{"stuff2"}
		fmt.Println("@SubmitEvent 1")
		//        needlatch(hook)
		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: 2,
		}
		//        needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: stop_value,
		}
		needlatch(hook)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

	})

}

// This version lets events drop and has a timeout on the subscription (which should not kick in)
func TestFanout2WithTimeout(t *testing.T) {
	_, id, err := MakeEventChannel("stuff2timeout", true, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	var timeout = int64(1000000000 * 2) // 2 seconds

	sub, err := SubscribeToChannel("stuff2timeout", timeout)
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}

	// listen := func() {
	// }

	fmt.Printf("Starting listener...\n")

	hook := newLatch(1)

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			fmt.Println("listener() at top of loop.")
			ok, subchan := sub.GetChannel()
			if ok {
				select {
				case ev := <-subchan:
					sdata, ok := ev.Data.(*dat)
					if ok {
						fmt.Printf("listener: Got event %d\n", sdata.x)
						if sdata.x == stop_value {
							break listenLoop
						} else {
							if sdata.x != x+1 {
								log.Fatalf("Did not get correct data in event.")
							}
							x = sdata.x
						}
					} else {
						log.Fatalf("Failed type assertion")
					}
					latchon(hook)
					// default:
					//     latchon(hook)
				}
			} else {
				log.Fatalf("channel timedout?? should be available. wrong.")
				sub.ReleaseChannel()
				break
			}
			sub.ReleaseChannel()
		}
	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat{
			x: 1,
		}
		names := []string{"stuff2timeout"}
		fmt.Println("@SubmitEvent 1")
		//        needlatch(hook)
		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: 2,
		}
		//        needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: stop_value,
		}
		needlatch(hook)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

	})

}

// test using subscription names, instead of keeping the object
func TestFanoutGetSubscription(t *testing.T) {
	_, id, err := MakeEventChannel("somename", true, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	sub, err := SubscribeToChannel("somename", 0)
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}

	subid := sub.GetID()
	sub = nil // clear out subscription from here (let's GC maybe del - but it should not)

	var ok bool
	ok, sub = GetSubscription("somename", subid)
	if !ok {
		log.Fatalf("GetSubscription() call failed!")
	} else if sub == nil {
		log.Fatalf("GetSubscription() returned a nil EventSubscription")
	}

	// listen := func() {
	// }

	fmt.Printf("Starting listener... (TestFanoutGetSubscription)\n")

	hook := newLatch(1)

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			fmt.Println("listener() at top of loop.")
			ok, subchan := sub.GetChannel()
			if ok {
				select {
				case ev := <-subchan:
					sdata, ok := ev.Data.(*dat)
					if ok {
						fmt.Printf("listener: Got event %d\n", sdata.x)
						if sdata.x == stop_value {
							break listenLoop
						} else {
							if sdata.x != x+1 {
								log.Fatalf("Did not get correct data in event.")
							}
							x = sdata.x
						}
					} else {
						log.Fatalf("Failed type assertion")
					}
					latchon(hook)
					// default:
					//     latchon(hook)
				}
			} else {
				log.Fatalf("channel should be available. wrong.")
			}
			sub.ReleaseChannel()
		}
	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat{
			x: 1,
		}
		names := []string{"somename"}
		fmt.Println("@SubmitEvent 1")
		//        needlatch(hook)
		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: 2,
		}
		needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: stop_value,
		}
		needlatch(hook)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

	})

}

// This version lets events drop and has a timeout on the subscription (which should not kick in)
func TestFanout2TimeoutDoesNotHappen(t *testing.T) {
	channame := "timeouthappens"
	_, id, err := MakeEventChannel(channame, true, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	var timeout = int64(1000000000 * 2) // 2 seconds

	sub, err := SubscribeToChannel(channame, timeout)
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}

	sub.ReleaseChannel() // releases original Subscribe
	// listen := func() {
	// }

	fmt.Printf("Starting listener...\n")

	//	hook := newLatch(1)

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			fmt.Println("listener() at top of loop. timeout happens.")
			time.Sleep(time.Duration(timeout/2) * time.Nanosecond)
			ok, subchan := sub.GetChannel()
			if ok {
				select {
				case ev := <-subchan:
					sdata, ok := ev.Data.(*dat)
					if ok {
						fmt.Printf("listener: Got event %d\n", sdata.x)
						if sdata.x == stop_value {
							sub.ReleaseChannel() // releases GetChannel() call
							break listenLoop
						} else {
							if sdata.x != x+1 {
								log.Fatalf("Did not get correct data in event.")
							}
							x = sdata.x
						}
					} else {
						log.Fatalf("Failed type assertion")
					}

					//					latchon(hook)
					// default:
					//     latchon(hook)
				}
				fmt.Printf("--past select{}\n")
				sub.ReleaseChannel() // releases GetChannel() call
				fmt.Printf("past ReleaseChannel\n")
			} else {
				log.Fatalf("channel timedout?? should be available. Something is wrong.\n")
				//				latchon(hook)
				break
			}
		}

	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat{
			x: 1,
		}
		names := []string{channame}
		fmt.Println("@SubmitEvent 1")
		//        needlatch(hook)
		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: 2,
		}
		//        needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: stop_value,
		}
		//		needlatch(hook)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

	})

}

// This version lets events drop and has a timeout on the subscription (which *should* kick in)
func TestFanout2TimeoutDoesHappen(t *testing.T) {
	channame := "timeouthappens2"
	_, id, err := MakeEventChannel(channame, true, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	var timeout = int64(time.Second * 2) // 2 seconds

	sub, err := SubscribeToChannel(channame, timeout)
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}

	sub.ReleaseChannel() // releases original Subscribe

	// listen := func() {
	// }

	fmt.Printf("Starting listener...\n")

	//	hook := newLatch(1)

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			fmt.Println("listener() at top of loop. timeout happens.")
			time.Sleep(time.Duration(timeout*2) * time.Nanosecond)
			ok, subchan := sub.GetChannel()
			if ok {
				log.Fatalf("Wrong. The channel should have timed out.")
				// all code below until end of if{} is unreachable. just leaving for
				// continuity
				select {
				case ev := <-subchan:
					sdata, ok := ev.Data.(*dat)
					if ok {
						fmt.Printf("listener: Got event %d\n", sdata.x)
						if sdata.x == stop_value {
							sub.ReleaseChannel() // releases GetChannel() call
							break listenLoop
						} else {
							if sdata.x != x+1 {
								log.Fatalf("Did not get correct data in event.")
							}
							x = sdata.x
						}
					} else {
						log.Fatalf("Failed type assertion")
					}

					//					latchon(hook)
					// default:
					//     latchon(hook)
				}
				fmt.Printf("--past select{}\n")
				sub.ReleaseChannel() // releases GetChannel() call
				fmt.Printf("past ReleaseChannel\n")
			} else {
				fmt.Printf("channel timedout?? shouldn't be available. OK. 2\n")
				//				latchon(hook)
				if !sub.IsClosed() {
					log.Fatalf("Wrong. subcription should now report closed. It timedout.")
				}
				break
			}
		}

	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat{
			x: 1,
		}
		names := []string{channame}
		fmt.Println("@SubmitEvent 1")
		//        needlatch(hook)
		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: 2,
		}
		//        needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: stop_value,
		}
		//		needlatch(hook)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

	})

}

func TestBasicSubscribeClose(t *testing.T) {
	channame := "TestBasicSubscribeClose"
	fmt.Printf("Making channel...\n")
	_, id, err := MakeEventChannel(channame, true, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	//	var timeout = int64(1000000000 * 2) // 2 seconds
	fmt.Printf("Subscribing to channel...\n")
	sub, err := SubscribeToChannel(channame, 0)
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}
	fmt.Printf("Past subscribe channel\n")
	subid := sub.GetID()
	fmt.Printf("Id of slave is: %s\n", subid)
	sub.ReleaseChannel() // releases original Subscribe
	fmt.Printf("Past ReleaseChannel\n")
	sub.Close()
	fmt.Printf("Past Close\n")
	ok, subret := GetSubscription(channame, subid)
	fmt.Printf("Past GetSubscription %v %v", ok, subret)
	if ok {
		log.Fatalf("Should not be able to get subscription by ID - should be closed")
	}

}

// This version lets events drop and has a timeout on the subscription (which *should* kick in)
func TestTimeoutDoesHappenGetSub(t *testing.T) {
	channame := "timeouthappens2sub"
	fmt.Printf("Making channel...")
	_, id, err := MakeEventChannel(channame, true, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	var timeout = int64(1000000000 * 2) // 2 seconds

	fmt.Printf("Subscribing to channel...")
	sub, err := SubscribeToChannel(channame, timeout)
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}

	subid := sub.GetID()
	fmt.Printf("Id of slave is: %s\n", subid)
	sub.ReleaseChannel() // releases original Subscribe

	sub = nil
	// listen := func() {
	// }

	fmt.Printf("Starting listener...\n")

	//	hook := newLatch(1)

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			fmt.Println("listener() at top of loop. timeout happens.")
			time.Sleep(time.Duration(timeout*2) * time.Nanosecond)

			ok, sub := GetSubscription(channame, subid)

			if ok {
				log.Fatalf("Should not be able to get subscription by ID - should be timed out")
			} else {
				if sub != nil {
					log.Printf("not nil, but: %+v\n", sub)
					log.Fatalf("Should return nil if subscription gone.")
				}
				return
			}
			// just left for context:
			ok, subchan := sub.GetChannel()
			if ok {
				log.Fatalf("Wrong. The channel should have timed out.")
				// all code below until end of if{} is unreachable. just leaving for
				// continuity
				select {
				case ev := <-subchan:
					sdata, ok := ev.Data.(*dat)
					if ok {
						fmt.Printf("listener: Got event %d\n", sdata.x)
						if sdata.x == stop_value {
							sub.ReleaseChannel() // releases GetChannel() call
							break listenLoop
						} else {
							if sdata.x != x+1 {
								log.Fatalf("Did not get correct data in event.")
							}
							x = sdata.x
						}
					} else {
						log.Fatalf("Failed type assertion")
					}

					//					latchon(hook)
					// default:
					//     latchon(hook)
				}
				fmt.Printf("--past select{}\n")
				sub.ReleaseChannel() // releases GetChannel() call
				fmt.Printf("past ReleaseChannel\n")
			} else {
				fmt.Printf("channel timedout?? should be available. OK. 2\n")
				//				latchon(hook)
				if !sub.IsClosed() {
					log.Fatalf("Wrong. subcription should now report closed. It timedout.")
				}
				break
			}
		}

	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat{
			x: 1,
		}
		names := []string{channame}
		fmt.Println("@SubmitEvent 1")
		//        needlatch(hook)
		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: 2,
		}
		//        needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: stop_value,
		}
		//		needlatch(hook)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

	})

}

// This version lets events drop
func TestFanout2Multi(t *testing.T) {
	_, id, err := MakeEventChannel("stuff2", true, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	sub, err := SubscribeToChannel("stuff2", 0)
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}
	sub.ReleaseChannel()
	sub2, err := SubscribeToChannel("stuff2", 0)
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}
	sub2.ReleaseChannel()
	// listen := func() {
	// }

	fmt.Printf("Starting listener...\n")

	hook := newLatch(1)
	hook2 := newLatch(1)

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
		latchon(hook)
	listenLoop:
		for {
			fmt.Println("listener() at top of loop.")
			ok, subchan := sub.GetChannel()
			if ok {
				select {
				case ev := <-subchan:
					sdata, ok := ev.Data.(*dat)
					if ok {
						fmt.Printf("listener: Got event %d\n", sdata.x)
						if sdata.x == stop_value {
							break listenLoop
						} else {
							if sdata.x != x+1 {
								log.Fatalf("Did not get correct data in event.")
							}
							x = sdata.x
						}
					} else {
						log.Fatalf("Failed type assertion")
					}
					latchon(hook)
					// default:
					//     latchon(hook)
				}
			} else {
				log.Fatalf("channel should be available. wrong.")
			}
			sub.ReleaseChannel()
		}
	})

	t.Run("listener2", func(t *testing.T) {
		t.Parallel()
		x := 0
		latchon(hook2)
	listenLoop:
		for {
			fmt.Println("listener2() at top of loop.")
			ok, subchan := sub2.GetChannel()
			if ok {
				select {
				case ev := <-subchan:
					sdata, ok := ev.Data.(*dat)
					if ok {
						fmt.Printf("listener: Got event %d\n", sdata.x)
						if sdata.x == stop_value {
							break listenLoop
						} else {
							if sdata.x != x+1 {
								log.Fatalf("Did not get correct data in event.")
							}
							x = sdata.x
						}
					} else {
						log.Fatalf("Failed type assertion")
					}
					latchon(hook2)
					// default:
					//     latchon(hook)
				}
			} else {
				log.Fatalf("channel should be available. wrong.")
			}
			sub2.ReleaseChannel()
		}
	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat{
			x: 1,
		}
		names := []string{"stuff2"}
		fmt.Println("@SubmitEvent 1")
		needlatch(hook)
		needlatch(hook2)
		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: 2,
		}
		needlatch(hook)
		needlatch(hook2)

		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: stop_value,
		}
		needlatch(hook)
		needlatch(hook2)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

	})

}

func TestNofanout1(t *testing.T) {
	chan_name := "stuff3"
	_, id, err := MakeEventChannel(chan_name, false, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	sub, err := SubscribeToChannel(chan_name, 0)
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}
	sub.ReleaseChannel()

	hook := newLatch(1)

	fmt.Printf("Starting listener...\n")

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			fmt.Println("listener() at top of loop.")
			ok, subchan := sub.GetChannel()
			if ok {
				select {
				case ev := <-subchan:
					sdata, ok := ev.Data.(*dat)
					if ok {
						fmt.Printf("listener: Got event %d\n", sdata.x)
						if sdata.x == stop_value {
							break listenLoop
						} else {
							if sdata.x != x+1 {
								log.Fatalf("Did not get correct data in event.")
							}
							x = sdata.x
						}
					} else {
						log.Fatalf("Failed type assertion")
					}
					latchon(hook)
					// default:
					//     latchon(hook)
				}
			} else {
				log.Fatalf("channel should be available. wrong.")
			}
			sub.ReleaseChannel()
		}
	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat{
			x: 1,
		}
		names := []string{chan_name}
		fmt.Println("@SubmitEvent 1")
		//        needlatch(hook)
		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: 2,
		}
		needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: stop_value,
		}
		needlatch(hook)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

	})

}

// This version lets events drop
func TestNofanout2Multi(t *testing.T) {
	chan_name := "stuff4"
	_, id, err := MakeEventChannel(chan_name, false, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	hook := newLatch(1)

	sub, err := SubscribeToChannel(chan_name, 0)
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}

	sub2, err := SubscribeToChannel(chan_name, 0)
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}

	fmt.Printf("Starting listener...\n")

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			fmt.Println("listener() at top of loop.")
			ok, subchan := sub.GetChannel()
			if ok {
				select {
				case ev := <-subchan:
					sdata, ok := ev.Data.(*dat)
					if ok {
						fmt.Printf("listener: Got event %d\n", sdata.x)
						if sdata.x == stop_value {
							break listenLoop
						} else {
							if sdata.x != x+1 {
								fmt.Println("Ok - but did not get next ordered data.")
							}
							x = sdata.x
						}
					} else {
						log.Fatalf("Failed type assertion")
					}
				case <-hook:
					fmt.Println("got 'hook' break")
					break listenLoop
					//                    latchon(hook)
					// default:
					//     latchon(hook)
				}
			} else {
				log.Fatalf("channel should be available. wrong.")
			}
			sub.ReleaseChannel()
		}
	})

	t.Run("listener2", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			fmt.Println("listener2() at top of loop.")
			ok, subchan := sub2.GetChannel()
			if ok {
				select {
				case ev := <-subchan:
					sdata, ok := ev.Data.(*dat)
					if ok {
						fmt.Printf("listener: Got event %d\n", sdata.x)
						if sdata.x == stop_value {
							break listenLoop
						} else {
							if sdata.x != x+1 {
								fmt.Println("Ok - but did not get next ordered data.")
							}
							x = sdata.x
						}
					} else {
						log.Fatalf("Failed type assertion")
					}
				case <-hook:
					fmt.Println("got 'hook' break")
					break listenLoop

					//                    latchon(hook)
					// default:
					//     latchon(hook2)
				}
			} else {
				log.Fatalf("channel should be available. wrong.")
			}
			sub2.ReleaseChannel()
		}
	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat{
			x: 1,
		}
		names := []string{chan_name}
		fmt.Println("@SubmitEvent 1")
		//        needlatch(hook)
		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: 2,
		}
		//        needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: 3,
		}
		//        needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: 4,
		}
		//        needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat{
			x: stop_value,
		}
		// needlatch(hook)
		// needlatch(hook2)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}
		close(hook)

	})

}

//////////////////////////////////////////////////
// Tests for DeferredResponseManager
//////////////////////////////////////////////////

// we create some incoming type so the json encoder can decode the normally generic
// MaestroEvent API for us. This could also be done using a UnmarshalJSON()
// see: http://gregtrowbridge.com/golang-json-serialization-with-interfaces/
type dat2Incoming struct {
	X int `json:"x"`
}

type maestroEventIncoming struct {
	Data dat2Incoming `json:"data"`
}

// Test the ResponseManager by creating a subchannel, then a mock server, and simulating a long poll
func TestResponseManager1(t *testing.T) {
	channame := "Monterrey"
	_, id, err := MakeEventChannel(channame, true, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	sub, err := SubscribeToChannel(channame, 0) // subscribe with no timeout
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}

	fmt.Printf("Starting listener...\n")

	// hook := newLatch(1)

	manager := NewDeferredResponseManager()

	// note - you must release the channel before handing to manager
	// if you want the manager to handle the subscription auto timingout
	sub.ReleaseChannel()
	manager.Start()

	//Wait until the manager is running before subscribing, otherwise we will hang
	for { 
		if(manager.running != true) {
			time.Sleep(1000) 
		} else {
			break
		}
	}

	subid, _, err := manager.AddEventSubscription(sub) // _ is responder

	handler := manager.MakeSubscriptionHttpHandler("test", func(w http.ResponseWriter, req *http.Request) (err error, id string) {
		parts := strings.Split(req.URL.Path, "/")
		id = parts[len(parts)-1]
		fmt.Sprintf("parse path: %+v  id=%s\n", parts, id)
		return
	}, func(w http.ResponseWriter, req *http.Request) {}, nil, 2500*time.Millisecond)

	fmt.Printf("added event subscription to ResponseManager %s\n", subid)
	///////////////////
	// setup test server
	// See: https://golang.org/src/net/http/httptest/example_test.go
	ts := httptest.NewServer(handler)

	if err != nil {
		log.Fatal(err)
	}

	//	fmt.Printf("GET>> %s", greeting)
	// Output: Hello, client
	/////////////////

	if err != nil {
		log.Fatalf("Got error from manager.AddEventSubscriotion: %+v", err)
		t.FailNow()
	}

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			// Get events through a local loopback server as a test
			inevents := make([]maestroEventIncoming, 0)
			fmt.Println("http listener() at top of loop.")
			url := fmt.Sprintf("%s/%s", ts.URL, subid)
			fmt.Printf("GET %s\n", url)
			res, err := http.Get(url)
			if err != nil {
				fmt.Printf("Error from Get() %s\n", err.Error())
				log.Fatal(err)
			}
			inbuf, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			fmt.Printf("Got %s\n", string(inbuf))
			err = json.Unmarshal(inbuf, &inevents)

			if err != nil {
				log.Fatalf("Failed to decode json. --> %v", err.Error())
			}

			fmt.Printf("inevents: %+v\n", inevents)
			for n, ev := range inevents {
				// sdata, ok2 := ev.Data.(dat2_inc)
				// if ok2 {
				fmt.Printf("listener: Got event (%d) %d\n", n, ev.Data.X)
				if ev.Data.X == stop_value {
					fmt.Printf("got last data\n")
					break listenLoop
				} else {
					if ev.Data.X != x+1 {
						log.Fatal("Did not get correct data in event.")
					}
					x = ev.Data.X
				}
				// } else {
				// 	log.Fatalf("Failed to cast .Data to test 'dat2' type")
				// }
			}

			defer ts.Close()
			// ok, subchan := sub.GetChannel()
			// if ok {
			// 	select {
			// 	case ev := <-subchan:
			// 		sdata, ok := ev.Data.(*dat)
			// 		if ok {
			// 			fmt.Printf("listener: Got event %d\n", sdata.x)
			// 			if sdata.x == stop_value {
			// 				break listenLoop
			// 			} else {
			// 				if sdata.x != x+1 {
			// 					log.Fatalf("Did not get correct data in event.")
			// 				}
			// 				x = sdata.x
			// 			}
			// 		} else {
			// 			log.Fatalf("Failed type assertion")
			// 		}
			// 		latchon(hook)
			// 		// default:
			// 		//     latchon(hook)
			// 	}
			// } else {
			// 	log.Fatalf("channel should be available. wrong.")
			// }
			// sub.ReleaseChannel()
		}
	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat2{
			X: 1,
		}
		names := []string{channame}
		fmt.Println("Waiting 1 seconds before submitting events.")
		time.Sleep(time.Duration(1) * time.Second)
		fmt.Println("@SubmitEvent 1")
		//        needlatch(hook)

		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %v", err2)
			t.FailNow()
		}
		if dropped {
			log.Fatal("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat2{
			X: 2,
		}
		//        needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatal("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat2{
			X: stop_value,
		}
		// needlatch(hook)
		time.Sleep(time.Duration(1) * time.Second)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatal("Wrong - should not have dropped")
			t.FailNow()
		}

	})

}


// Test the ResponseManager by creating a subchannel, then a mock server, and simulating a long poll
// In this case, the client, after getting events, takes 5 seconds to ask for more, and the
// long response handler times out before this.
func TestResponseManagerClientWaits5Seconds(t *testing.T) {
	fmt.Printf("-----------------TestResponseManagerClientTimeout------------------\n")
	channame := "Guadalajara"
	_, id, err := MakeEventChannel(channame, true, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	sub, err := SubscribeToChannel(channame, 0) // subscribe with no timeout
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}

	fmt.Printf("Starting listener...\n")

	// hook := newLatch(1)

	manager := NewDeferredResponseManager()
	manager.Start()

	//Wait until the manager is running before subscribing, otherwise we will hang
	for { 
		if(manager.running != true) {
			time.Sleep(1000) 
		} else {
			break
		}
	}

	// note - you must release the channel before handing to manager
	// if you want the manager to handle the subscription auto timingout
	sub.ReleaseChannel()

	subid, _, err := manager.AddEventSubscription(sub) // _ is responder

	handler := manager.MakeSubscriptionHttpHandler("test", func(w http.ResponseWriter, req *http.Request) (err error, id string) {
		parts := strings.Split(req.URL.Path, "/")
		id = parts[len(parts)-1]
		fmt.Sprintf("parse path: %+v  id=%s\n", parts, id)
		return
		// timeout in 2.5 seconds -->
	}, func(w http.ResponseWriter, req *http.Request) {}, nil, 2500*time.Millisecond)

	fmt.Printf("added event subscription to ResponseManager %s\n", subid)
	///////////////////
	// setup test server
	// See: https://golang.org/src/net/http/httptest/example_test.go
	ts := httptest.NewServer(handler)

	if err != nil {
		log.Fatal(err)
	}

	//	fmt.Printf("GET>> %s", greeting)
	// Output: Hello, client
	/////////////////

	if err != nil {
		log.Fatalf("Got error from manager.AddEventSubscriotion: %+v", err)
		t.FailNow()
	}

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			fmt.Println("Client at top of loop.")
			// Get events through a local loopback server as a test
			inevents := make([]maestroEventIncoming, 0)
			fmt.Println("http listener() at top of loop.")
			url := fmt.Sprintf("%s/%s", ts.URL, subid)
			fmt.Printf("GET %s\n", url)
			res, err := http.Get(url)
			if err != nil {
				fmt.Printf("Error from Get() %s\n", err.Error())
				log.Fatal(err)
			}
			inbuf, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			fmt.Printf("Got <%s>\n", string(inbuf))
			err = json.Unmarshal(inbuf, &inevents)

			if res.StatusCode != 200 {
				if res.StatusCode == http.StatusNoContent {
					if x > 1 {
						fmt.Println("Got StatusNoContent")
						break
					} else {
						log.Fatalf("Got StatusNoContent but did not get the first 2 data events")
						break
					}
				} else {
					log.Fatalf("Got weird StatusCode back %d", res.StatusCode)
				}
			}

			if err != nil {
				log.Fatalf("Failed to decode json. --> %v", err.Error())
			}

			fmt.Printf("inevents: %+v\n", inevents)
			for n, ev := range inevents {
				// sdata, ok2 := ev.Data.(dat2_inc)
				// if ok2 {
				fmt.Printf("listener: Got event (%d) %d\n", n, ev.Data.X)
				if ev.Data.X == stop_value {
					//					fmt.Printf("got last data\n")
					log.Fatalf("got last data - should not have\n")
					break listenLoop
				} else {
					if ev.Data.X != x+1 {
						log.Fatalf("Did not get correct data in event.")
					}
					x = ev.Data.X
				}
				// } else {
				// 	log.Fatalf("Failed to cast .Data to test 'dat2' type")
				// }
			}

			defer ts.Close()
			// waiting this long will cause us to miss the event last event.
			// now we should get a NoContent
			fmt.Println("Client waiting 5 seconds...")
			time.Sleep(time.Duration(5) * time.Second)

		}
	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat2{
			X: 1,
		}
		names := []string{channame}
		fmt.Println("Waiting 1 seconds before submitting events.")
		time.Sleep(time.Duration(1) * time.Second)
		fmt.Println("@SubmitEvent 1")
		//        needlatch(hook)

		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %v", err2)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat2{
			X: 2,
		}
		//        needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat2{
			X: stop_value,
		}
		// needlatch(hook)
		time.Sleep(time.Duration(1) * time.Second)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

	})

}

// in this scenario, the Client waits so long that the response manager shutsdown the
// the event subscription
func TestResponseManagerTimeout1(t *testing.T) {
	channame := "Tequila"
	_, id, err := MakeEventChannel(channame, true, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	sub, err := SubscribeToChannel(channame, int64(time.Millisecond*2500)) // subscribe with no timeout
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}

	fmt.Printf("Starting listener...\n")

	// hook := newLatch(1)

	manager := NewDeferredResponseManager()
	manager.Start()

	//Wait until the manager is running before subscribing, otherwise we will hang
	for { 
		if(manager.running != true) {
			time.Sleep(1000) 
		} else {
			break
		}
	}

	// note - you must release the channel before handing to manager
	// if you want the manager to handle the subscription auto timingout
	sub.ReleaseChannel()

	subid, _, err := manager.AddEventSubscription(sub) // _ is responder

	handler := manager.MakeSubscriptionHttpHandler("test", func(w http.ResponseWriter, req *http.Request) (err error, id string) {
		parts := strings.Split(req.URL.Path, "/")
		id = parts[len(parts)-1]
		fmt.Sprintf("parse path: %+v  id=%s\n", parts, id)
		return
	}, func(w http.ResponseWriter, req *http.Request) {}, nil, 2500*time.Millisecond)

	fmt.Printf("added event subscription to ResponseManager %s\n", subid)
	///////////////////
	// setup test server
	// See: https://golang.org/src/net/http/httptest/example_test.go
	ts := httptest.NewServer(handler)

	if err != nil {
		log.Fatal(err)
	}

	//	fmt.Printf("GET>> %s", greeting)
	// Output: Hello, client
	/////////////////

	if err != nil {
		log.Fatalf("Got error from manager.AddEventSubscriotion: %+v", err)
		t.FailNow()
	}

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			// Get events through a local loopback server as a test
			inevents := make([]maestroEventIncoming, 0)
			fmt.Println("http listener() at top of loop.")
			url := fmt.Sprintf("%s/%s", ts.URL, subid)
			fmt.Printf("GET %s\n", url)
			res, err := http.Get(url)
			if err != nil {
				fmt.Printf("Error from Get() %s\n", err.Error())
				log.Fatal(err)
			}
			inbuf, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			fmt.Printf("Got %s\n", string(inbuf))
			err = json.Unmarshal(inbuf, &inevents)

			if res.StatusCode != 200 {
				if res.StatusCode == http.StatusNoContent {
					if x > 1 {
						log.Fatal("Got StatusNoContent - should get 404")
						//						fmt.Println("Got StatusNoContent")
						break
					} else {
						log.Fatal("Got StatusNoContent but did not get the first 2 data events")
						break
					}
				} else if res.StatusCode == http.StatusNotFound {
					if x > 1 {
						fmt.Println("Got StatusNotFound - correct")
						break
					} else {
						log.Fatal("Got StatusNotFound but did not get the first 2 data events")
						break
					}
				} else {
					log.Fatalf("Got weird StatusCode back: %v", res.StatusCode)
				}
			}

			if err != nil {
				log.Fatalf("Failed to decode json. --> %v", err.Error())
			}

			fmt.Printf("inevents: %+v\n", inevents)
			for n, ev := range inevents {
				// sdata, ok2 := ev.Data.(dat2_inc)
				// if ok2 {
				fmt.Printf("listener: Got event (%d) %d\n", n, ev.Data.X)
				if ev.Data.X == stop_value {
					fmt.Printf("got last data\n")
					break listenLoop
				} else {
					if ev.Data.X != x+1 {
						log.Fatal("Did not get correct data in event.")
					}
					x = ev.Data.X
				}
				// } else {
				// 	log.Fatalf("Failed to cast .Data to test 'dat2' type")
				// }
			}

			defer ts.Close()
			fmt.Println("Client / listener - waiting 10 seconds.")
			time.Sleep(time.Second * 10)

		}
	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat2{
			X: 1,
		}
		names := []string{channame}
		fmt.Println("Waiting 1 seconds before submitting events.")
		time.Sleep(time.Duration(1) * time.Second)
		fmt.Println("@SubmitEvent 1")
		//        needlatch(hook)

		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %v", err2)
			t.FailNow()
		}
		if dropped {
			log.Fatal("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat2{
			X: 2,
		}
		//        needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatal("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat2{
			X: stop_value,
		}
		// needlatch(hook)
		time.Sleep(time.Duration(1) * time.Second)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

	})

}

// Test the ResponseManager by creating a subchannel, then a mock server, and simulating a long poll
// In this case, the client, after getting events, takes 5 seconds to ask for more, and the
// long response handler times out before this.
func TestResponseManagerRouterClientWaits5Seconds(t *testing.T) {
	fmt.Printf("-----------------TestResponseManagerClientTimeout------------------\n")
	channame := "Hermosillo"
	_, id, err := MakeEventChannel(channame, true, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	sub, err := SubscribeToChannel(channame, 0) // subscribe with no timeout
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}

	fmt.Printf("Starting listener...\n")

	// hook := newLatch(1)

	manager := NewDeferredResponseManager()
	manager.Start()

	//Wait until the manager is running before subscribing, otherwise we will hang
	for { 
		if(manager.running != true) {
			time.Sleep(1000) 
		} else {
			break
		}
	}

	// note - you must release the channel before handing to manager
	// if you want the manager to handle the subscription auto timingout
	sub.ReleaseChannel()

	subid, _, err := manager.AddEventSubscription(sub) // _ is responder

	router := httprouter.New()

	handler := manager.MakeSubscriptionHttprouterHandler("test", func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (err error, id string) {
		// parts := strings.Split(req.URL.Path, "/")
		// id = parts[len(parts)-1]
		id = ps.ByName("id")
		fmt.Sprintf("parse path: id=%s\n", id)
		return
		// timeout in 2.5 seconds -->
	}, func(w http.ResponseWriter, req *http.Request) {}, nil, 2500*time.Millisecond)

	fmt.Printf("added event subscription to ResponseManager %s\n", subid)

	router.GET("/events/:id", handler)
	///////////////////
	// setup test server
	// See: https://golang.org/src/net/http/httptest/example_test.go
	ts := httptest.NewServer(router)

	if err != nil {
		log.Fatal(err)
	}

	//	fmt.Printf("GET>> %s", greeting)
	// Output: Hello, client
	/////////////////

	if err != nil {
		log.Fatalf("Got error from manager.AddEventSubscriotion: %+v", err)
		t.FailNow()
	}

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			fmt.Println("Client at top of loop.")
			// Get events through a local loopback server as a test
			inevents := make([]maestroEventIncoming, 0)
			fmt.Println("http listener() at top of loop.")
			url := fmt.Sprintf("%s/events/%s", ts.URL, subid)
			fmt.Printf("GET %s\n", url)
			res, err := http.Get(url)
			if err != nil {
				fmt.Printf("Error from Get() %s\n", err.Error())
				log.Fatal(err)
			}
			inbuf, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			fmt.Printf("Got <%s>\n", string(inbuf))
			err = json.Unmarshal(inbuf, &inevents)

			if res.StatusCode != 200 {
				if res.StatusCode == http.StatusNoContent {
					if x > 1 {
						fmt.Println("Got StatusNoContent")
						break
					} else {
						log.Fatalf("Got StatusNoContent but did not get the first 2 data events")
						break
					}
				} else {
					log.Fatalf("Got weird StatusCode back %d", res.StatusCode)
				}
			}

			if err != nil {
				log.Fatalf("Failed to decode json. --> %v", err.Error())
			}

			fmt.Printf("inevents: %+v\n", inevents)
			for n, ev := range inevents {
				// sdata, ok2 := ev.Data.(dat2_inc)
				// if ok2 {
				fmt.Printf("listener: Got event (%d) %d\n", n, ev.Data.X)
				if ev.Data.X == stop_value {
					//					fmt.Printf("got last data\n")
					log.Fatal("got last data - should not have\n")
					break listenLoop
				} else {
					if ev.Data.X != x+1 {
						log.Fatalf("Did not get correct data in event.")
					}
					x = ev.Data.X
				}
				// } else {
				// 	log.Fatalf("Failed to cast .Data to test 'dat2' type")
				// }
			}

			defer ts.Close()
			// waiting this long will cause us to miss the event last event.
			// now we should get a NoContent
			fmt.Println("Client waiting 5 seconds...")
			time.Sleep(time.Duration(5) * time.Second)

		}
	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat2{
			X: 1,
		}
		names := []string{channame}
		fmt.Println("Waiting 1 seconds before submitting events.")
		time.Sleep(time.Duration(1) * time.Second)
		fmt.Println("@SubmitEvent 1")
		//        needlatch(hook)

		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %v", err2)
			t.FailNow()
		}
		if dropped {
			log.Fatal("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat2{
			X: 2,
		}
		//        needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat2{
			X: stop_value,
		}
		// needlatch(hook)
		time.Sleep(time.Duration(1) * time.Second)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

	})

}

func TestResponseManagerRouterTimeout1(t *testing.T) {
	channame := "Tijuana"
	_, id, err := MakeEventChannel(channame, true, false)
	if err != nil {
		log.Fatalf("Error making event channel: %+v", err)
		t.FailNow()
	} else {
		fmt.Printf("master channel created: %s\n", id)
	}

	sub, err := SubscribeToChannel(channame, int64(time.Millisecond*2500)) // subscribe with no timeout
	if err != nil {
		log.Fatalf("Subscribe to channel error: %+v", err)
		t.FailNow()
	}

	fmt.Printf("Starting listener...\n")

	// hook := newLatch(1)

	manager := NewDeferredResponseManager()
	manager.Start()

	//Wait until the manager is running before subscribing, otherwise we will hang
	for { 
		if(manager.running != true) {
			time.Sleep(1000) 
		} else {
			break
		}
	}
    
	// note - you must release the channel before handing to manager
	// if you want the manager to handle the subscription auto timingout
	sub.ReleaseChannel()

	subid, _, err := manager.AddEventSubscription(sub) // _ is responder

	router := httprouter.New()
	
	handler := manager.MakeSubscriptionHttprouterHandler("test", func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (err error, id string) {
		// parts := strings.Split(req.URL.Path, "/")
		// id = parts[len(parts)-1]
		id = ps.ByName("id")
		fmt.Sprintf("parse path: id=%s\n", id)
		return
		// timeout in 2.5 seconds -->
	}, func(w http.ResponseWriter, req *http.Request) {}, nil, 2500*time.Millisecond)

	fmt.Printf("added event subscription to ResponseManager %s\n", subid)

	router.GET("/events/:id", handler)

	ts := httptest.NewServer(router)

	if err != nil {
		log.Fatal(err)
	}

	//	fmt.Printf("GET>> %s", greeting)
	// Output: Hello, client
	/////////////////

	if err != nil {
		log.Fatalf("Got error from manager.AddEventSubscriotion: %+v", err)
		t.FailNow()
	}

	t.Run("listener", func(t *testing.T) {
		t.Parallel()
		x := 0
	listenLoop:
		for {
			// Get events through a local loopback server as a test
			inevents := make([]maestroEventIncoming, 0)
			fmt.Println("http listener() at top of loop.")
			url := fmt.Sprintf("%s/events/%s", ts.URL, subid)
			fmt.Printf("GET %s\n", url)
			res, err := http.Get(url)
			if err != nil {
				fmt.Printf("Error from Get() %s\n", err.Error())
				log.Fatal(err)
			}
			inbuf, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			fmt.Printf("Got %s\n", string(inbuf))
			err = json.Unmarshal(inbuf, &inevents)

			if res.StatusCode != 200 {
				if res.StatusCode == http.StatusNoContent {
					if x > 1 {
						log.Fatalf("Got StatusNoContent - should get 404")
						//						fmt.Println("Got StatusNoContent")
						break
					} else {
						log.Fatalf("Got StatusNoContent but did not get the first 2 data events")
						break
					}
				} else if res.StatusCode == http.StatusNotFound {
					if x > 1 {
						fmt.Println("Got StatusNotFound - correct")
						break
					} else {
						log.Fatal("Got StatusNotFound but did not get the first 2 data events")
						break
					}
				} else {
					log.Fatalf("Got weird StatusCode back: %v", res.StatusCode)
				}
			}

			if err != nil {
				log.Fatalf("Failed to decode json. --> %v", err.Error())
			}

			fmt.Printf("inevents: %+v\n", inevents)
			for n, ev := range inevents {
				// sdata, ok2 := ev.Data.(dat2_inc)
				// if ok2 {
				fmt.Printf("listener: Got event (%d) %d\n", n, ev.Data.X)
				if ev.Data.X == stop_value {
					fmt.Printf("got last data\n")
					break listenLoop
				} else {
					if ev.Data.X != x+1 {
						log.Fatal("Did not get correct data in event.")
					}
					x = ev.Data.X
				}
				// } else {
				// 	log.Fatal("Failed to cast .Data to test 'dat2' type")
				// }
			}

			defer ts.Close()
			fmt.Println("Client / listener - waiting 10 seconds.")
			time.Sleep(time.Second * 10)

		}
	})

	t.Run("emitter", func(t *testing.T) {
		t.Parallel()
		d := &dat2{
			X: 1,
		}
		names := []string{channame}
		fmt.Println("Waiting 1 seconds before submitting events.")
		time.Sleep(time.Duration(1) * time.Second)
		fmt.Println("@SubmitEvent 1")
		//        needlatch(hook)

		dropped, err2 := SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %v", err2)
			t.FailNow()
		}
		if dropped {
			log.Fatal("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat2{
			X: 2,
		}
		//        needlatch(hook)
		fmt.Println("@SubmitEvent 2")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

		d = &dat2{
			X: stop_value,
		}
		// needlatch(hook)
		time.Sleep(time.Duration(1) * time.Second)
		fmt.Println("@SubmitEvent end")
		dropped, err2 = SubmitEvent(names, d)
		if err2 != nil {
			log.Fatalf("Error on SubmitEvent channel: %+v", err)
			t.FailNow()
		}
		if dropped {
			log.Fatalf("Wrong - should not have dropped")
			t.FailNow()
		}

	})

}

// To run test:
// sub out for your directories
// sudo GOROOT=/opt/go PATH="$PATH:/opt/go/bin" GOPATH=/home/ed/work/gostuff LD_LIBRARY_PATH=../../greasego/deps/lib /opt/go/bin/go test
