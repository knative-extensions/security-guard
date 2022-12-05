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
	"net/http"
	"testing"
)

func TestResp_V1(t *testing.T) {
	resp := &http.Response{Header: http.Header{"a": {"x"}}}
	resp2 := &http.Response{Header: http.Header{"b": {"x"}}}
	arguments := [][]*http.Response{
		{resp},
		{resp2},
		{resp2},
		{resp},
		{resp},
		{resp},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(RespProfile))
		piles = append(piles, new(RespPile))
		configs = append(configs, new(RespConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}
