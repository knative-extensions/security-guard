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

package guardutils

import (
	"fmt"
	"testing"
	"time"
)

func TestTicker(t *testing.T) {
	type args struct {
		default_interval time.Duration
		intervalStr      string
		start            bool
		prestop          bool
		stop             bool
		ok               bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "no start",
			args: args{},
		}, {
			name: "start",
			args: args{
				default_interval: time.Duration(0),
				intervalStr:      "",
				start:            true,
				prestop:          false,
				ok:               true,
			},
		}, {
			name: "default start",
			args: args{
				default_interval: time.Duration(3),
				intervalStr:      "",
				start:            true,
				prestop:          false,
				ok:               true,
			},
		}, {
			name: "parse start",
			args: args{
				default_interval: time.Duration(0),
				intervalStr:      "4ns",
				start:            true,
				prestop:          false,
				ok:               true,
			},
		}, {
			name: "ilegal-parse start",
			args: args{
				default_interval: time.Duration(0),
				intervalStr:      "fsdkjf",
				start:            true,
				prestop:          false,
				ok:               true,
			},
		}, {
			name: "pretop start",
			args: args{
				default_interval: time.Duration(0),
				intervalStr:      "",
				start:            true,
				prestop:          true,
				ok:               true,
			},
		}, {
			name: "prestop default start",
			args: args{
				default_interval: time.Duration(3),
				intervalStr:      "",
				start:            true,
				prestop:          true,
				ok:               true,
			},
		}, {
			name: "prestop parse start",
			args: args{
				default_interval: time.Duration(0),
				intervalStr:      "4ns",
				start:            true,
				prestop:          true,
				ok:               true,
			},
		}, {
			name: "start stop",
			args: args{
				start: true,
				stop:  true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MinimumInterval = 100000
			fmt.Println("Test", tt.name)
			var ticker Ticker
			if tt.args.start {
				ticker.Start()
			}
			if tt.args.prestop {
				ticker.Stop()
			}
			if tt.args.default_interval != time.Duration(0) || tt.args.intervalStr != "" {
				ticker.Parse(tt.args.intervalStr, tt.args.default_interval)
			}
			if tt.args.start {
				ticker.Start()
			}
			if tt.args.stop {
				ticker.Stop()
			}
			testticker := time.NewTicker(1000)
			select {
			case <-ticker.Ch():
				if !tt.args.ok {
					t.Error("Expected a Timeout!")
				}
			case <-testticker.C:
				if tt.args.ok {
					t.Error("Timeout when expected a ticker")
				}
			}
		})
	}
}
