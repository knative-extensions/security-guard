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

package guardgate

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"

	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
)

type fakeKmgr struct{}

func (f *fakeKmgr) InitConfigs() {}

func (f *fakeKmgr) Read(ns string, sid string, isCm bool) (*spec.GuardianSpec, error) {
	return new(spec.GuardianSpec), nil
}

func (f *fakeKmgr) Create(ns string, sid string, isCm bool, guardianSpec *spec.GuardianSpec) error {
	return nil
}

func (f *fakeKmgr) Set(ns string, sid string, isCm bool, guardianSpec *spec.GuardianSpec) error {
	return nil
}

func (f *fakeKmgr) GetGuardian(ns string, sid string, cm bool, autoActivate bool) *spec.GuardianSpec {
	return new(spec.GuardianSpec)
}

func (f *fakeKmgr) Watch(ns string, cmFlag bool, set func(ns string, sid string, cmFlag bool, g *spec.GuardianSpec)) {
}

func (f *fakeKmgr) TokenData(token string) (sid string, ns string, err error) {
	return "mysid", "myns", nil
}

func (f *fakeKmgr) GetConfig(ns string, cmName string, config map[string]string) error {
	return nil
}

type fakeHttpClient struct {
	statusCode int
	json       []byte
	err        error
	fail       bool
	count      int
}

func fakeClient(statusCode int, response string) (*gateClient, *fakeHttpClient) {
	srv := NewGateClient("url", "x", "x", false)
	client := &fakeHttpClient{statusCode: statusCode, json: []byte(response)}
	srv.httpClient = client
	srv.clearPile()
	srv.kubeMgr = &fakeKmgr{}
	return srv, client
}
func (hc *fakeHttpClient) ReadToken(audience string) {

}

func (hc *fakeHttpClient) Do(req *http.Request) (*http.Response, error) {
	hc.count++

	if hc.fail {
		return &http.Response{StatusCode: hc.statusCode, Body: nil}, hc.err

	}
	// create a new reader with that JSON
	r := ioutil.NopCloser(bytes.NewReader(hc.json))
	return &http.Response{StatusCode: hc.statusCode, Body: r}, hc.err
}

func Test_guardClient_reportPile(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		srv, client := fakeClient(http.StatusOK, "Problem in request")

		srv.reportPile()
		if client.count != 0 {
			t.Error("Expected no request")
		}

		srv.addToPile(new(spec.SessionDataProfile))
		srv.addToPile(new(spec.SessionDataProfile))
		srv.reportPile()
		if client.count != 1 {
			t.Error("Expected no request")
		}

		srv.addToPile(new(spec.SessionDataProfile))
		srv.addToPile(new(spec.SessionDataProfile))

		client = &fakeHttpClient{statusCode: http.StatusBadRequest, json: []byte("Problem in request")}
		srv.httpClient = client
		srv.reportPile()
		if client.count != 1 {
			t.Error("Expected request")
		}

		srv.pile.Count = 1
		srv.httpClient = &fakeHttpClient{statusCode: http.StatusBadRequest, json: []byte("Problem in request"), err: errors.New("Wow")}
		srv.reportPile()
		if client.count != 1 {
			t.Error("Expected request")
		}
		srv.pile.Count = 1
		client = &fakeHttpClient{fail: true}
		srv.httpClient = client
		srv.reportPile()
		if client.count != 1 {
			t.Error("Expected request")
		}
	})

}

func Test_guardClient_loadGuardian(t *testing.T) {

	t.Run("simple", func(t *testing.T) {
		srv, _ := fakeClient(0, "")
		g := new(spec.GuardianSpec)

		if got := srv.loadGuardian(); !reflect.DeepEqual(got, g) {
			t.Errorf("guardClient.loadGuardian() = %v, want %v", got, g)
		}

		j, _ := json.Marshal(new(spec.GuardianSpec))
		srv.httpClient = &fakeHttpClient{statusCode: http.StatusOK, json: j}
		srv.clearPile()
		srv.kubeMgr = &fakeKmgr{}
		g = new(spec.GuardianSpec)

		if got := srv.loadGuardian(); !reflect.DeepEqual(got, g) {
			t.Errorf("guardClient.loadGuardian() = %v, want %v", got, g)
		}
	})
}
