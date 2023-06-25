/*
Copyright 2022 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dselect

import (
	"context"
	"sync"
	"testing"
	"time"
)

var myTestMutex sync.Mutex

func myFunc(myVal *int64) func(val int64) {
	return func(val int64) {
		myTestMutex.Lock()
		defer myTestMutex.Unlock()
		*myVal = val
	}
}

func TestNewDSelect(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		var ticks, ticks1, ticks2, complete1, complete2 int64
		setTick := myFunc(&ticks)
		setTick1 := myFunc(&ticks1)
		setTick2 := myFunc(&ticks2)
		setComplete1 := myFunc(&complete1)
		setComplete2 := myFunc(&complete2)
		ctxMain, cancelMain := context.WithCancel(context.Background())
		ctx1, cancel1 := context.WithCancel(context.Background())
		ctx2 := context.Background()
		ds := NewDSelect(ctxMain, setTick)
		ds.Add(ctx1, setTick1, setComplete1, 2)
		ds.Add(ctx2, setTick2, setComplete2, 2)
		myTestMutex.Lock()
		if ticks != 0 || ticks1 != 0 || ticks2 != 0 {
			t.Errorf("ticks should be zero  - got instead %d %d %d", ticks, ticks1, ticks2)
		}
		if complete1 != 0 || complete2 != 0 {
			t.Errorf("complete should be zero  - got instead %d %d", complete1, complete2)
		}
		myTestMutex.Unlock()
		time.Sleep(time.Second + 100*time.Millisecond)
		myTestMutex.Lock()
		if ticks == 0 || ticks1 != ticks || ticks2 != ticks {
			t.Errorf("ticks1 and ticks2 should be zero, but not main tick  - got instead %d %d %d", ticks, ticks1, ticks2)
		}
		if complete1 != 0 || complete2 != 0 {
			t.Errorf("complete should be zero  - got instead %d %d", complete1, complete2)
		}
		myTestMutex.Unlock()
		time.Sleep(time.Second)
		myTestMutex.Lock()
		if ticks == 0 || ticks1 == ticks || ticks2 != ticks1 {
			t.Errorf("ticks1 and ticks 2 should be equal and not zero - got instead %d %d %d", ticks, ticks1, ticks2)
		}
		if complete1 != 0 || complete2 != 0 {
			t.Errorf("ticks1 and ticks 2 should be equal and not zero - got instead %d %d %d", ticks, ticks1, ticks2)
		}
		myTestMutex.Unlock()
		cancel1()
		time.Sleep(time.Second)
		myTestMutex.Lock()
		if ticks == 0 || ticks1 == 0 || ticks2 != ticks1 {
			t.Errorf("ticks1 and ticks 2 should be equal and not zero - got instead %d %d %d", ticks, ticks1, ticks2)
		}
		if complete1 == 0 || complete2 != 0 {
			t.Errorf("complete2 should, but not complete1  - got instead %d %d", complete1, complete2)
		}
		myTestMutex.Unlock()
		time.Sleep(time.Second)
		myTestMutex.Lock()
		if ticks == 0 || ticks1 == 0 || ticks2 == ticks1 {
			t.Errorf("ticks should be different  - got instead %d %d %d", ticks, ticks1, ticks2)
		}
		if complete1 == 0 || complete2 != 0 {
			t.Errorf("complete2 should, but not complete1  - got instead %d %d", complete1, complete2)
		}
		myTestMutex.Unlock()
		cancelMain()
		time.Sleep(time.Second)
		myTestMutex.Lock()
		if ticks == 0 || ticks1 == 0 || ticks2 == ticks1 {
			t.Errorf("ticks should be zero  - got instead %d %d %d", ticks, ticks1, ticks2)
		}
		if complete1 == 0 || complete2 == 0 {
			t.Errorf("complete should not be zero  - got instead %d %d", complete1, complete2)
		}
		myTestMutex.Unlock()
	})

}
