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
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionData(t *testing.T) {
	body := map[string][]string{
		"abc": {"ccc", "dddd"},
		"www": {"aaa", "bbb"},
	}

	jsonBytes1, _ := json.Marshal(body)
	jsonBytes2, _ := json.Marshal(body)
	req := httptest.NewRequest("GET", "/", bytes.NewReader(jsonBytes1))
	req2 := httptest.NewRequest("POST", "/eee/ddd/f.html", bytes.NewReader(jsonBytes2))
	resp := &http.Response{Header: http.Header{"a": {"x"}}}
	resp2 := &http.Response{Header: http.Header{"b": {"x"}}}

	cip := net.IPv4(1, 2, 3, 5)
	cip2 := net.IPv4(1, 22, 3, 5)
	arguments := [][]interface{}{
		{req, cip, resp, nil, nil, time.Now(), time.Now(), time.Now()},
		{req2, cip2, resp2, nil, nil, time.Now(), time.Now(), time.Now()},
		{req2, cip, resp2, nil, nil, time.Now(), time.Now(), time.Now()},
		{req, cip, resp2, nil, nil, time.Now(), time.Now(), time.Now()},
		{req, cip, resp2, nil, nil, time.Now(), time.Now(), time.Now()},
		{req, cip, resp2, nil, nil, time.Now(), time.Now(), time.Now()},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(SessionDataProfile))
		piles = append(piles, new(SessionDataPile))
		configs = append(configs, new(SessionDataConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}

func TestPile_Json(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		pile := &SessionDataPile{}
		var bytes []byte
		var err error
		if bytes, err = json.Marshal(pile); err != nil {
			t.Errorf("json.Marshal Error %v", err.Error())
		}
		if err = json.Unmarshal(bytes, &pile); err != nil {
			t.Errorf("json.Unmarshal Error %v", err.Error())
		}

	})

	t.Run("Full", func(t *testing.T) {
		sp := new(SessionDataProfile)
		pile := &SessionDataPile{}
		pile.Add(sp)
		{
			manifestJson, err := json.MarshalIndent(pile, "", "  ")
			if err != nil {
				t.Errorf("json.Marshal Error %v", err.Error())
			}
			fmt.Println(string(manifestJson))

			pile2 := &SessionDataPile{}
			err = json.Unmarshal(manifestJson, &pile2)
			if err != nil {
				t.Errorf("json.Unmarshal Error %v", err.Error())
			}
		}
	})
}
