/*
Copyright 2018 The Knative Authors

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

package qpoption

import (
	"context"
	"errors"
	"net/http"
	"os"
	"reflect"
	"testing"

	pi "knative.dev/security-guard/pkg/pluginterfaces"
	"go.uber.org/zap"
	"knative.dev/serving/pkg/queue/sharedmain"
)

var myQpextentionPreifx = "my.prefix/"
var myAnnotationsPath = "/tmp/annotations"
var myPlugName = "myplug"
var sugar *zap.SugaredLogger
var defaults *sharedmain.Defaults
var defaults2 *sharedmain.Defaults
var returnedError error

type myplug struct {
	name    string
	version string
}

func (p *myplug) PlugName() string {
	return p.name
}

func (p *myplug) PlugVersion() string {
	return p.version
}

func (p *myplug) ApproveRequest(req *http.Request) (*http.Request, error) {
	return req, nil
}

func (p *myplug) ApproveResponse(req *http.Request, resp *http.Response) (*http.Response, error) {
	return resp, returnedError
}

func (p *myplug) Shutdown() {
	pi.Log.Infof("Plug %s: Shutdown", p.name)
}

func (p *myplug) Start(ctx context.Context) context.Context {
	return ctx
}

func (p *myplug) Init(ctx context.Context, c map[string]string, serviceName string, namespace string, logger pi.Logger) context.Context {
	return ctx
}

func initGate() *GateQPOption {
	n := NewGateQPOption()
	n.defaults = defaults
	n.securityPlug = &myplug{name: myPlugName, version: "myver"}
	return n
}

func clearAnnotations() {
	os.Remove(myAnnotationsPath)
}
func addConfigAnnotations(a map[string]string) (string, string) {
	file, err := os.Create(myAnnotationsPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var buf string
	for k, v := range a {
		buf = buf + myQpextentionPreifx + myPlugName + "-config-" + k + "=" + v + "\n"
	}
	buf = buf + myQpextentionPreifx + "config-k=\n"
	buf = buf + myQpextentionPreifx + "config-=v\n"
	buf = buf + myQpextentionPreifx + myPlugName + "con-k=v\n"
	buf = buf + myQpextentionPreifx + "config=enable\n"
	buf = buf + "boom/config=enable\n"
	buf = buf + "config=enable\n"
	file.WriteString(buf)
	return myAnnotationsPath, myQpextentionPreifx
}

func addActivateAnnotations(a map[string]string) (string, string) {
	file, err := os.Create(myAnnotationsPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var buf string
	for k, v := range a {
		buf = buf + myQpextentionPreifx + k + "-activate=" + v + "\n"
	}
	buf = buf + myQpextentionPreifx + "activate=enable\n"
	buf = buf + "boom/activate=enable\n"
	buf = buf + "activate=enable\n"

	file.WriteString(buf)
	return myAnnotationsPath, myQpextentionPreifx
}

func TestNewGateQPOption(t *testing.T) {
	t.Run("TestNewGateQPOption", func(t *testing.T) {
		want := new(GateQPOption)
		if got := NewGateQPOption(); !reflect.DeepEqual(got, want) {
			t.Errorf("NewGateQPOption() = %v, want %v", got, want)
		}
	})
}

func TestGateQPOption_RoundTrip(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		p := initGate()
		req := new(http.Request)
		addActivateAnnotations(map[string]string{myPlugName: "enable"})
		annotationsFilePath = myAnnotationsPath
		qpextentionPreifx = myQpextentionPreifx
		p.Setup(defaults)
		clearAnnotations()
		gotResp, err := p.RoundTrip(req)
		if err != nil {
			t.Errorf("GateQPOption.RoundTrip() error = %v", err)
			return
		}
		if gotResp == nil {
			t.Errorf("GateQPOption.RoundTrip() gotResp is nil")
		}
	})
	t.Run("RoundTrip", func(t *testing.T) {
		p := initGate()
		req := new(http.Request)
		_, err := p.RoundTrip(req)
		if err == nil {
			t.Errorf("GateQPOption.RoundTrip() error was expected")
			return
		}
	})
	t.Run("RoundTrip", func(t *testing.T) {
		p := initGate()
		req := new(http.Request)
		addActivateAnnotations(map[string]string{myPlugName: "enable"})
		annotationsFilePath = myAnnotationsPath
		qpextentionPreifx = myQpextentionPreifx
		p.Setup(defaults)
		clearAnnotations()
		myerr := errors.New("bad")
		returnedError = myerr
		_, err := p.RoundTrip(req)
		returnedError = nil
		if err != myerr {
			t.Errorf("GateQPOption.RoundTrip() wrong error was returned")
			return
		}

	})
}

func TestGateQPOption_ProcessConfigAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]string
		wantResp bool
	}{
		{name: "empty", wantResp: true, config: map[string]string{}},
		{name: "few", wantResp: true, config: map[string]string{"abckey": "abc val", "key": "val", "key123": "val123"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := initGate()
			myAnnotations, myQpextentionPreifx := addConfigAnnotations(tt.config)
			gotResp := p.ProcessAnnotations(myAnnotations, myQpextentionPreifx)
			clearAnnotations()
			if gotResp != tt.wantResp {
				t.Errorf("GateQPOption.ProcessAnnotations() gotResp = %v, wantResp %v", gotResp, tt.wantResp)
				return
			}
			if !reflect.DeepEqual(p.config, tt.config) {
				t.Errorf("GateQPOption.ProcessAnnotations() = %v, want %v", p.config, tt.config)
			}
		})
	}
}

func TestGateQPOption_ProcessActivateAnnotations(t *testing.T) {
	tests := []struct {
		name      string
		config    map[string]string
		activated bool
		wantResp  bool
	}{
		{name: "empty", wantResp: true, activated: false, config: map[string]string{}},
		{name: "other", wantResp: true, activated: false, config: map[string]string{"abckey": "enable"}},
		{name: "true", wantResp: true, activated: true, config: map[string]string{myPlugName: "enable"}},
		{name: "false", wantResp: true, activated: false, config: map[string]string{myPlugName: "bla"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := initGate()
			myAnnotations, myQpextentionPreifx := addActivateAnnotations(tt.config)
			gotResp := p.ProcessAnnotations(myAnnotations, myQpextentionPreifx)
			clearAnnotations()
			if gotResp != tt.wantResp {
				t.Errorf("GateQPOption.ProcessAnnotations() gotResp = %v, wantResp %v", gotResp, tt.wantResp)
				return
			}
			if !reflect.DeepEqual(p.activated, tt.activated) {
				t.Errorf("GateQPOption.ProcessAnnotations() activated = %v, want %v", p.activated, tt.activated)
			}
		})
	}
}

func TestGateQPOption_ProcessNoAnnotations(t *testing.T) {

	t.Run("No annotations", func(t *testing.T) {
		p := initGate()
		clearAnnotations()
		gotResp := p.ProcessAnnotations(myAnnotationsPath, myQpextentionPreifx)
		if gotResp != false {
			t.Errorf("GateQPOption.ProcessAnnotations() gotResp = %v, wantResp %v", gotResp, false)
			return
		}
		var config map[string]string
		if !reflect.DeepEqual(p.config, config) {
			t.Errorf("GateQPOption.ProcessAnnotations() = %v, want %v", p.config, config)
		}
	})
}

func TestGateQPOption_Setup(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		p := initGate()
		addActivateAnnotations(map[string]string{myPlugName: "enable"})
		annotationsFilePath = myAnnotationsPath
		qpextentionPreifx = myQpextentionPreifx
		p.Setup(defaults)
		clearAnnotations()
		if !reflect.DeepEqual(p.activated, true) {
			t.Errorf("GateQPOption.Setup() = %v, want %v", p.activated, true)
		}
		if pi.Log == nil {
			t.Errorf("GateQPOption.Setup() pi.Log is nil")
		}
		if p.securityPlug == nil {
			t.Errorf("GateQPOption.Setup() p.securityPlug  is nil")
		}
	})
	t.Run("missing", func(t *testing.T) {
		p := initGate()
		addActivateAnnotations(map[string]string{myPlugName: "enable"})
		annotationsFilePath = myAnnotationsPath
		qpextentionPreifx = myQpextentionPreifx
		p.Setup(defaults2)
		clearAnnotations()
		if !reflect.DeepEqual(p.activated, true) {
			t.Errorf("GateQPOption.Setup() = %v, want %v", p.activated, true)
		}
		if pi.Log == nil {
			t.Errorf("GateQPOption.Setup() pi.Log is nil")
		}
		if p.securityPlug == nil {
			t.Errorf("GateQPOption.Setup() p.securityPlug  is nil")
		}
	})
	t.Run("no annotations", func(t *testing.T) {
		p := initGate()
		clearAnnotations()
		p.Setup(defaults2)
		if !reflect.DeepEqual(p.activated, false) {
			t.Errorf("GateQPOption.Setup() = %v, want %v", p.activated, false)
		}
		if pi.Log == nil {
			t.Errorf("GateQPOption.Setup() pi.Log is nil")
		}
		if p.securityPlug == nil {
			t.Errorf("GateQPOption.Setup() p.securityPlug  is nil")
		}
	})

}

func TestGateQPOption_Shutdown(t *testing.T) {

	t.Run("sutdown", func(t *testing.T) {
		p := initGate()
		p.Shutdown()
	})

}

type myDefaultTransport struct {
	//nextRoundTripper http.RoundTripper // the next roundtripper
}

func (p *myDefaultTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	return new(http.Response), nil
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	//shutdown()
	os.Exit(code)
}

func setup() {
	logger, _ := zap.NewDevelopment()
	sugar = logger.Sugar()
	defaults = new(sharedmain.Defaults)
	defaults.Logger = sugar
	defaults.Ctx = context.Background()
	defaults.Env.ServingConfiguration = "ServingConfiguration"
	defaults.Env.ServingNamespace = "ServingNamespace"
	defaults.Env.ServingPod = "ServingPod"
	defaults.Env.ServingPodIP = "ServingPodIP"
	defaults.Env.ServingRevision = "ServingRevision"
	defaults.Env.ServingService = "ServingService"
	defaults.Transport = new(myDefaultTransport)

	defaults2 = new(sharedmain.Defaults)
	defaults2.Env.ServingConfiguration = "ServingConfiguration"
	defaults2.Env.ServingNamespace = "ServingNamespace"
	defaults2.Env.ServingPod = "ServingPod"
	defaults2.Env.ServingPodIP = "ServingPodIP"
	defaults2.Env.ServingRevision = "ServingRevision"
	defaults2.Env.ServingService = ""
	plug := new(myplug)
	plug.name = myPlugName
	plug.version = "myver"
	pi.RegisterPlug(plug)

}
