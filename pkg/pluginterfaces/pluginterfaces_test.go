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

package pluginterfaces

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

type plug struct{}

func (p *plug) Init(ctx context.Context, c map[string]string, serviceName string, namespace string, logger Logger) context.Context {
	return ctx
}

func (p *plug) Shutdown() {}
func (p *plug) PlugName() string {
	return "plug42"
}
func (p *plug) PlugVersion() string {
	return ""
}
func (p *plug) ApproveRequest(*http.Request) (*http.Request, error) {
	return nil, nil
}
func (p *plug) ApproveResponse(*http.Request, *http.Response) (*http.Response, error) {
	return nil, nil
}

func TestGetPlug(t *testing.T) {
	t.Run("no plug", func(t *testing.T) {
		if got := GetPlug(); got != nil {
			t.Errorf("GetPlug() = %v, want %v", got, nil)
		}
		if got := GetPlugByName("plug42"); got != nil {
			t.Errorf("GetPlug() = %v, want %v", got, nil)
		}

		RegisterPlug(&plug{})
		if got := GetPlug(); !reflect.DeepEqual(got, &plug{}) {
			t.Errorf("GetPlug() = %v, want %v", got, &plug{})
		}
		if got := GetPlugByName("plug41"); got != nil {
			t.Errorf("GetPlug() = %v, want %v", got, nil)
		}
		if got := GetPlugByName("plug42"); !reflect.DeepEqual(got, &plug{}) {
			t.Errorf("GetPlug() = %v, want %v", got, &plug{})
		}

		RegisterPlug(&plug{})
		if got := GetPlug(); got != nil {
			t.Errorf("GetPlug() = %v, want %v", got, nil)
		}
		if got := GetPlugByName("plug41"); got != nil {
			t.Errorf("GetPlug() = %v, want %v", got, nil)
		}
		if got := GetPlugByName("plug42"); !reflect.DeepEqual(got, &plug{}) {
			t.Errorf("GetPlug() = %v, want %v", got, &plug{})
		}
	})
}

func TestGetPlugByName(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want RoundTripPlug
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPlugByName(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPlugByName() = %v, want %v", got, tt.want)
			}
		})
	}
}

type MyLogger struct {
	debug int
	info  int
	warn  int
	err   int
	sync  int
}

func (m *MyLogger) Debugf(format string, args ...interface{}) {
	m.debug++
}
func (m *MyLogger) Infof(format string, args ...interface{}) {
	m.info++
}
func (m *MyLogger) Warnf(format string, args ...interface{}) {
	m.warn++
}
func (m *MyLogger) Errorf(format string, args ...interface{}) {
	m.err++
}
func (m *MyLogger) Sync() error {
	m.sync++
	return nil
}

func Test_LOGONCE(t *testing.T) {
	var mylogger MyLogger
	Log = &mylogger

	t.Run("simple", func(t *testing.T) {
		if mylogger.debug != 0 {
			t.Errorf("mylogger.debug expected 0 got %d", mylogger.debug)
		}
		LogOnce.Debugf("xxx")
		if mylogger.debug != 1 {
			t.Errorf("mylogger.debug expected 1 got %d", mylogger.debug)
		}
		LogOnce.Debugf("yyy")
		if mylogger.debug != 2 {
			t.Errorf("mylogger.debug expected 2 got %d", mylogger.debug)
		}
		LogOnce.Debugf("yyy")
		if mylogger.debug != 2 {
			t.Errorf("mylogger.debug expected 2 got %d", mylogger.debug)
		}

		if mylogger.info != 0 {
			t.Errorf("mylogger.info expected 0 got %d", mylogger.info)
		}
		LogOnce.Infof("xxx")
		if mylogger.info != 1 {
			t.Errorf("mylogger.info expected 1 got %d", mylogger.info)
		}
		LogOnce.Infof("yyy")
		if mylogger.info != 2 {
			t.Errorf("mylogger.info expected 2 got %d", mylogger.info)
		}
		LogOnce.Infof("yyy")
		if mylogger.info != 2 {
			t.Errorf("mylogger.info expected 2 got %d", mylogger.info)
		}

		if mylogger.warn != 0 {
			t.Errorf("mylogger.warn expected 0 got %d", mylogger.warn)
		}
		LogOnce.Warnf("xxx")
		if mylogger.warn != 1 {
			t.Errorf("mylogger.warn expected 1 got %d", mylogger.warn)
		}
		LogOnce.Warnf("yyy")
		if mylogger.warn != 2 {
			t.Errorf("mylogger.warn expected 2 got %d", mylogger.warn)
		}
		LogOnce.Warnf("yyy")
		if mylogger.warn != 2 {
			t.Errorf("mylogger.warn expected 2 got %d", mylogger.warn)
		}

		if mylogger.err != 0 {
			t.Errorf("mylogger.err expected 0 got %d", mylogger.err)
		}
		LogOnce.Errorf("xxx")
		if mylogger.err != 1 {
			t.Errorf("mylogger.err expected 1 got %d", mylogger.err)
		}
		LogOnce.Errorf("yyy")
		if mylogger.err != 2 {
			t.Errorf("mylogger.err expected 2 got %d", mylogger.err)
		}
		LogOnce.Errorf("yyy")
		if mylogger.err != 2 {
			t.Errorf("mylogger.err expected 2 got %d", mylogger.err)
		}

		if mylogger.sync != 0 {
			t.Errorf("mylogger.sync expected 0 got %d", mylogger.sync)
		}
		LogOnce.Sync()
		if mylogger.sync != 1 {
			t.Errorf("mylogger.sync expected 1 got %d", mylogger.sync)
		}

	})

}
