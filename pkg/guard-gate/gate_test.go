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

package guardgate

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
	utils "knative.dev/security-guard/pkg/guard-utils"
	pi "knative.dev/security-guard/pkg/pluginterfaces"
)

type dLog struct{}

func (d dLog) Debugf(format string, args ...interface{}) {}
func (d dLog) Infof(format string, args ...interface{})  {}
func (d dLog) Warnf(format string, args ...interface{})  {}
func (d dLog) Errorf(format string, args ...interface{}) {}
func (d dLog) Sync() error                               { return nil }

var defaultLog dLog

func testInit(c map[string]string) *plug {
	p := new(plug)
	p.version = plugVersion
	p.name = plugName

	if c == nil {
		c = make(map[string]string)
		c["guard-url"] = "url"
		c["use-cm"] = "true"
		c["monitor-pod"] = "x"
	}

	pi.RegisterPlug(p)
	p.preInit(c, "svcName", "myns", defaultLog)
	p.gateState = fakeGateState()
	p.gateState.sync(true)
	return p
}

func initTickerTest() *plug {
	p := new(plug)
	p.version = plugVersion
	p.name = plugName

	c := make(map[string]string)
	c["guard-url"] = "url"
	c["use-cm"] = "true"
	c["monitor-pod"] = "x"

	pi.RegisterPlug(p)

	p.preInit(c, "svcName", "myns", defaultLog)
	p.gateState = fakeGateState()
	p.gateState.sync(true)
	p.gateState.stat.Init()
	return p
}

func cancelLater(cancel context.CancelFunc) {
	td, _ := time.ParseDuration("10ms")
	<-time.After(td)
	cancel()
}

func Test_plug_guardMainEventLoop_1(t *testing.T) {
	t.Run("syncTicker", func(t *testing.T) {
		p := initTickerTest()
		p.syncTicker = utils.NewTicker(100000)
		p.syncTicker.Parse("", 300000)
		// lets rely on timeout
		ctx, cancelFunction := context.WithCancel(context.Background())
		go cancelLater(cancelFunction)
		p.guardMainEventLoop(ctx)
		if ret := p.gateState.stat.Log(); ret != "map[]" {
			t.Errorf("expected stat %s received %s", "map[]", ret)
		}
	})
}
func Test_plug_guardMainEventLoop_2(t *testing.T) {
	t.Run("podMonitorTicker", func(t *testing.T) {
		p := initTickerTest()
		ctx, cancelFunction := context.WithCancel(context.Background())

		p.podMonitorTicker = utils.NewTicker(100000)
		p.podMonitorTicker.Parse("", 300000)
		// lets rely on timeout
		go cancelLater(cancelFunction)
		p.guardMainEventLoop(ctx)
		if ret := p.gateState.stat.Log(); ret != "map[]" {
			t.Errorf("expected stat %s received %s", "map[]", ret)
		}
	})
}

func Test_plug_guardMainEventLoop_4(t *testing.T) {
	t.Run("reportPileTicker", func(t *testing.T) {
		p := initTickerTest()
		ctx, cancelFunction := context.WithCancel(context.Background())
		cancelFunction()
		p.guardMainEventLoop(ctx)
		if ret := p.gateState.stat.Log(); ret != "map[]" {
			t.Errorf("expected stat %s received %s", "map[]", ret)
		}
	})
}

func Test_plug_Initialize(t *testing.T) {

	tests := []struct {
		name            string
		c               map[string]string
		monitorPod      bool
		guardServiceUrl string
		useCm           bool
	}{
		// TODO: Add test cases.
		{
			name: "default",
			c: map[string]string{
				"guard-url":   "url",
				"use-cm":      "x",
				"monitor-pod": "x",
			},
			monitorPod:      false,
			guardServiceUrl: "url",
			useCm:           false,
		}, {
			name: "alternative",
			c: map[string]string{
				"guard-url":   "url1",
				"use-cm":      "true",
				"monitor-pod": "true",
			},
			monitorPod:      true,
			guardServiceUrl: "url1",
			useCm:           true,
		}, {
			name:            "no c",
			c:               nil,
			monitorPod:      true,
			guardServiceUrl: "http://guard-service.knative-serving",
			useCm:           false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := new(plug)
			p.version = plugVersion
			p.name = plugName
			p.podMonitorTicker = utils.NewTicker(utils.MinimumInterval)
			p.syncTicker = utils.NewTicker(utils.MinimumInterval)

			pi.RegisterPlug(p)
			p.preInit(tt.c, "svcName", "myns", defaultLog)

			if tt.monitorPod != p.gateState.monitorPod {
				t.Errorf("extected monitorPod %t got %t", tt.monitorPod, p.gateState.monitorPod)
			}
			if tt.guardServiceUrl != p.gateState.srv.guardServiceUrl {
				t.Errorf("extected guardServiceUrl %s got %s", tt.guardServiceUrl, p.gateState.srv.guardServiceUrl)
			}
			if tt.useCm != p.gateState.srv.useCm {
				t.Errorf("extected useCm %t got %t", tt.useCm, p.gateState.srv.useCm)
			}
		})

	}
}

func Test_plug_initPanic(t *testing.T) {
	t.Run("panic on sid", func(t *testing.T) {
		defer func() { _ = recover() }()
		p := new(plug)
		p.preInit(nil, "ns.svcName", "myns", defaultLog)
		t.Error("extected to panic")
	})
}

func Test_plug_Shutdown(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
		{""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testInit(nil)
			p.Shutdown()
		})
	}
}

func Test_plug_PlugName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// TODO: Add test cases.
		{"", plugName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testInit(nil)
			if got := p.PlugName(); got != tt.want {
				t.Errorf("plug.PlugName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_plug_PlugVersion(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// TODO: Add test cases.
		{"", plugVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testInit(nil)
			if got := p.PlugVersion(); got != tt.want {
				t.Errorf("plug.PlugVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_plug_ApproveResponse(t *testing.T) {
	t.Run("", func(t *testing.T) {
		p := testInit(nil)

		req := httptest.NewRequest("GET", "/some/path", nil)
		respRecorder := httptest.NewRecorder()
		fmt.Fprintf(respRecorder, "Hi there!")
		resp := respRecorder.Result()
		resp.Request = req
		resp.Header.Set("name", "val")

		_, err1 := p.ApproveResponse(req, resp)
		if err1 == nil {
			t.Errorf("ApproveResponse expected error ! ")
		}

		ctx, cancelFunction := context.WithCancel(req.Context())
		s := newSession(p.gateState, cancelFunction) // maintainer of the profile
		ctx = s.addSessionToContext(ctx)
		ctx = context.WithValue(ctx, ctxKey("GuardSession"), s)

		req1 := req.WithContext(ctx)

		resp.Request = req1

		_, err2 := p.ApproveResponse(req1, resp)
		if err2 != nil {
			t.Errorf("ApproveResponse error %v! ", err2)
		}

		p.gateState.decision = &spec.Decision{}
		p.gateState.ctrl.Block = true

		_, err3 := p.ApproveResponse(req1, resp)

		if err3 == nil {
			t.Errorf("ApproveRequest returned error = %v", err1)
		}

	})

}

func Test_plug_ApproveRequest(t *testing.T) {
	t.Run("", func(t *testing.T) {
		p := testInit(nil)
		req := httptest.NewRequest("GET", "/some/path", nil)
		req.Header.Set("name", "value")

		req1, err1 := p.ApproveRequest(req)

		if err1 != nil {
			t.Errorf("ApproveRequest returned error = %v", err1)
		}
		if req1 == nil {
			t.Errorf("ApproveRequest did not return a req ")
		}
		_, cancelFunction1 := context.WithCancel(req1.Context())
		cancelFunction1()

		p.gateState.decision = &spec.Decision{}
		p.gateState.ctrl.Block = true

		req2, err2 := p.ApproveRequest(req)

		if err2 == nil {
			t.Errorf("ApproveRequest returned error = %v", err1)
		}
		if req2 != nil {
			t.Errorf("ApproveRequest did not return a req ")
		}
		_, cancelFunction2 := context.WithCancel(req1.Context())
		cancelFunction2()
	})

}
