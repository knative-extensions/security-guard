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
