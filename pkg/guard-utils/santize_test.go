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

import "testing"

func Test_sanitize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "letter",
			in:   "a",
			want: "a",
		},
		{
			name: "letters",
			in:   "abc",
			want: "abc",
		},
		{
			name: "1a",
			in:   "1a",
			want: "",
		},
		{
			name: "a1",
			in:   "a1",
			want: "",
		},
		{
			name: "1",
			in:   "1",
			want: "",
		},
		{
			name: "-a",
			in:   "-a",
			want: "",
		},
		{
			name: "a-",
			in:   "a-",
			want: "",
		},
		{
			name: "a-",
			in:   "abc.d",
			want: "",
		},
		{
			name: "a1-a",
			in:   "a1-a",
			want: "a1-a",
		},
		{
			name: "long",
			in:   "abcdefghi1abcdefghi2abcdefghi3abcdefghi4abcdefghi5abcdefghi6a",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sanitize(tt.in); got != tt.want {
				t.Errorf("sanitize() = %v, want %v", got, tt.want)
			}
		})
	}
}
