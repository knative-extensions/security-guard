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

package v1alpha1

import (
	"reflect"
	"testing"
)

func TestSimpleVal_V1(t *testing.T) {
	arguments := [][]map[string][]string{
		{{"a": {"abc"}}},
		{{"a": {"123abc"}, "b": {"12"}}},
		{{"a": {"abcd"}}},
		{{"ex": {"abc"}}},
		{{"dfods": {"sdf;jsdfojssdfsdfsdlfosjf2390rj09uf"}}},
		{{"a*(Y((H(H&&^%&": {"^&U%&&^GTT*YHOIJMOI"}}},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(KeyValProfile))
		piles = append(piles, new(KeyValPile))
		configs = append(configs, new(KeyValConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}

func TestKeyValConfig_Decide(t *testing.T) {
	type fields struct {
		Vals          map[string]*SimpleValConfig
		OtherVals     *SimpleValConfig
		OtherKeynames *SimpleValConfig
	}
	type args struct {
		profile *KeyValProfile
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Decision
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &KeyValConfig{
				Vals:          tt.fields.Vals,
				OtherVals:     tt.fields.OtherVals,
				OtherKeynames: tt.fields.OtherKeynames,
			}
			if got := config.Decide(tt.args.profile); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KeyValConfig.Decide() = %v, want %v", got, tt.want)
			}
		})
	}
}
