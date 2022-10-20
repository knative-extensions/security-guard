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

package main

import (
	"os"
	"testing"

	_ "knative.dev/security-guard/pkg/guard-gate"
)

func Test_preMain(t *testing.T) {
	tests := []struct {
		name   string
		target string
		mux    bool
		env    map[string]string
	}{
		{
			name: "missing env",
		},
		{
			name: "missing service_url",
			env:  map[string]string{"SERVICE_NAME": "sid", "NAMESPACE": "ns"},
		},
		{
			name: "missing service_sid",
			env:  map[string]string{"SERVICE_URL": "http://127.0.0.1:80", "NAMESPACE": "ns"},
		},
		{
			name: "missing service_ns",
			env:  map[string]string{"SERVICE_NAME": "sid", "SERVICE_URL": "http://127.0.0.1:80"},
		},
		{
			name:   "envok",
			env:    map[string]string{"SERVICE_NAME": "sid", "NAMESPACE": "ns", "SERVICE_URL": "http://127.0.0.1:80"},
			mux:    true,
			target: ":22000",
		},
		{
			name: "fullenv",
			env: map[string]string{
				"SERVICE_NAME":     "sid",
				"NAMESPACE":        "ns",
				"SERVICE_URL":      "http://127.0.0.1:81",
				"GUARD_URL":        "http://127.0.0.1:82",
				"MONITOR_POD":      "true",
				"USE_CRD":          "true",
				"GUARD_PROXY_PORT": "8888",
			},
			mux:    true,
			target: ":8888",
		},
		{
			name: "wrongenv",
			env: map[string]string{
				"SERVICE_NAME":     "sid",
				"NAMESPACE":        "ns",
				"SERVICE_URL":      "http://user:abc{DEf1=ghi@example.com:5432",
				"GUARD_URL":        "http://user:abc{DEf1=ghi@example.com:5432",
				"MONITOR_POD":      "true",
				"USE_CRD":          "true",
				"GUARD_PROXY_PORT": "88881",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				os.Setenv(k, v)
			}
			//guardGate, mux, target, plugConfig, sid, ns, log := preMain()
			_, mux, target, _, _, _, _ := preMain()
			if (mux != nil) != tt.mux {
				t.Errorf("preMain() mux expected %t, received %t", tt.mux, mux != nil)
			}
			if target != tt.target {
				t.Errorf("preMain() target expected %v, received %v", tt.target, target)
			}
			for k := range tt.env {
				os.Unsetenv(k)
			}
		})
	}
}
