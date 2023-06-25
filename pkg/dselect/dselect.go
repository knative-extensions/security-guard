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
	"reflect"
	"time"
)

type dSelectSrv struct {
	ctx      context.Context
	complete func(int64)
	tick     func(int64)
	tickSecs int64
	lastTick int64
}

type DSelect struct {
	ctx       context.Context
	addSelect chan *dSelectSrv
	setTick   func(int64)
}

func NewDSelect(ctx context.Context, setTick func(int64)) *DSelect {
	ds := new(DSelect)
	ds.ctx = ctx
	ds.setTick = setTick
	ds.addSelect = make(chan *dSelectSrv)
	go ds.selects()
	return ds
}

func (ds *DSelect) Add(ctx context.Context, tick func(int64), complete func(int64), tickSecs int64) {
	ds.addSelect <- &dSelectSrv{
		ctx:      ctx,
		tick:     tick,
		complete: complete,
		tickSecs: tickSecs,
	}
}

func (ds *DSelect) selects() {
	var selectCases []reflect.SelectCase
	var dsSrvices []*dSelectSrv
	ticker := time.NewTicker(time.Second)
	ticks := time.Now().Unix()

	// selectCase 1 (chosen index=0)
	selectCases = append(selectCases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ds.ctx.Done()),
	})
	// selectCase 2 (chosen index=1)
	selectCases = append(selectCases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ds.addSelect),
	})
	// selectCase 3 (chosen index=2)
	selectCases = append(selectCases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ticker.C),
	})

	for {
		chosen, recv, _ := reflect.Select(selectCases)
		switch chosen {
		case 0: // ds.ctx.Done()
			ticker.Stop()
			for _, dsSrv := range dsSrvices {
				dsSrv.complete(ticks)
			}
			return
		case 1: // ds.addSelect
			dselectSrv := recv.Interface().(*dSelectSrv)
			dsSrvices = append(dsSrvices, dselectSrv)
			selectCases = append(selectCases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(dselectSrv.ctx.Done()),
			})
		case 2: // 1 second ticker
			ticks = recv.Interface().(time.Time).Unix()
			ds.setTick(ticks)
			numServices := len(dsSrvices)
			for index := 0; index < numServices; index++ {
				dsSrv := dsSrvices[index]
				if ticks > dsSrv.lastTick+dsSrv.tickSecs {
					dsSrv.tick(ticks)
					dsSrv.lastTick = ticks
				}
			}
		default: // dsSrv[index].ctx.Done()
			index := chosen - 3
			dsSrv := dsSrvices[index]
			dsSrv.complete(ticks)
			dsSrvices = append(dsSrvices[:index], dsSrvices[index+1:]...)
			selectCases = append(selectCases[:chosen], selectCases[chosen+1:]...)
		}
	}
}
