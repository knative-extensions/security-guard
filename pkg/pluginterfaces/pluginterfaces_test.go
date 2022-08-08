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

package pluginterfaces

import (
	"context"
	"net/http"
	"testing"
)

type plug struct {
}

func (p *plug) Init(ctx context.Context, c map[string]string, serviceName string, namespace string, logger Logger) context.Context {
	return ctx
}

func (p *plug) Shutdown() {}
func (p *plug) PlugName() string {
	return ""
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

func TestRegisterPlug(t *testing.T) {
	type args struct {
		p RoundTripPlug
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "simple", args: args{&plug{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterPlug(tt.args.p)
			if len(RoundTripPlugs) != 1 {
				t.Errorf("RegisterPlug error = wromg list length")
			}
		})
	}
}
