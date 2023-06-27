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
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
)

const testCert = `
-----BEGIN CERTIFICATE-----
MIICtDCCAZwCCQDzpJfrosIDzzANBgkqhkiG9w0BAQsFADAcMRowGAYDVQQDDBFz
ZWN1cml0eS1ndWFyZC1jYTAeFw0yMjEwMjcxMzA0MzFaFw0zMjEwMjQxMzA0MzFa
MBwxGjAYBgNVBAMMEXNlY3VyaXR5LWd1YXJkLWNhMIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEAnhNCuciY7qUqzskkBkZxe9zGJRtKONVof94oAT+nzilS
BBrs3zuHcI8v3qBQk63Hdj8xGw860A1fliKkO15iaC6QCRevVCUQ+pypIgRFY4Hj
S7ryLGStLjqXvBH/zaxio5Sz4+yAxwChsnlqvyGqNUTjzxh82s1Y6wN7Vmjn2Pfe
zNP2us/QhTqenBUYEsl16wPHwa62ZB4sP78yuRWeNkot2rq9qtC1DmgZl8u9wmcF
D+IYME0Ihqqm4VhmnK9fmqt4ozuGBSL3Cs3+Xu8t3et+riAYkVKbXUQWqoKiSven
PNJI8wRj2S6gZLCS7Z7zW3nlnKI4qKQijlNvjzw3tQIDAQABMA0GCSqGSIb3DQEB
CwUAA4IBAQBbdn4zo2p3dAH2qIdaap92sgT/A7D0ciX4bworVQwCHVPKRtWZlI4x
Wrlo/+VQFJ7YBJgpqJf//kTiWJ6ZHCxETpJrJ2X+48oxB6DNnx14+ykI10LSYmiJ
2aCs1vkrgzcp0+qXTRLNQBnNnMmmghsTgxkCwRvwAn1+KupJeFj7y8Jxxbp9cWLy
CNyW8U4UpaeAqRgzAHzjyodt4S1zxxpQJ5FSaxSL05OJtDodgokImhgJAoTNJVqZ
T30ny2EMCCPdmZfEpITjZrNl2rT2GY47AYBk44LWvKRDvrkiKzcpDxVJ7ggUrWyE
W+ve1pVd/1brFQJi1dF1J+QwhjCv7K1x
-----END CERTIFICATE-----`

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
	g := new(spec.GuardianSpec)
	g.SetToMaximalAutomation()
	return g
}

func (f *fakeKmgr) Watch(ns string, cmFlag bool, set func(ns string, sid string, cmFlag bool, g *spec.GuardianSpec)) {
}

func (f *fakeKmgr) TokenData(token string, labels []string) (podname string, sid string, ns string, err error) {
	return "mypod", "mysid", "myns", nil
}

func (f *fakeKmgr) DeletePod(ns string, podname string) {
}

type fakeHttpClient struct {
	statusCode int
	json       []byte
	err        error
	fail       bool
	count      int
}

func fakeClient(statusCode int, response string) (*gateClient, *fakeHttpClient) {
	srv := NewGateClient("url", "x", "x", "x", false)
	client := &fakeHttpClient{statusCode: statusCode, json: []byte(response)}
	srv.httpClient = client
	srv.pile.Clear()
	srv.kubeMgr = &fakeKmgr{}
	return srv, client
}
func (hc *fakeHttpClient) ReadToken(audience string) bool {
	return false
}

func (hc *fakeHttpClient) Do(req *http.Request) (*http.Response, error) {
	hc.count++

	if hc.fail {
		return &http.Response{StatusCode: hc.statusCode, Body: nil}, hc.err

	}
	// create a new reader with that JSON
	r := io.NopCloser(bytes.NewReader(hc.json))
	return &http.Response{StatusCode: hc.statusCode, Body: r}, hc.err
}

func Test_guardClient_sync_pile(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		srv, client := fakeClient(http.StatusOK, "Problem in request")
		var decision *spec.Decision
		spec.DecideInner(&decision, 7, "xxx")

		srv.addAlert(decision, "Session")
		srv.addAlert(decision, "Pod")
		srv.addToPile(new(spec.SessionDataProfile))
		srv.addToPile(new(spec.SessionDataProfile))
		srv.syncWithService(0)
		if client.count != 1 {
			t.Error("Expected request")
		}
		if srv.pile.Count != 2 {
			t.Errorf("Expected 2 in pile received %d", srv.pile.Count)
		}
		if len(srv.alerts) != 2 {
			t.Errorf("Expected 2 in alert received %d", len(srv.alerts))
		}

		resp := new(spec.SyncMessageResp)
		resp.Guardian = new(spec.GuardianSpec)
		bytes, _ := json.Marshal(resp)
		client = &fakeHttpClient{statusCode: http.StatusBadRequest, json: bytes}
		srv.httpClient = client

		srv.syncWithService(0)
		if client.count != 1 {
			t.Error("Expected request")
		}
		if srv.pile.Count != 2 {
			t.Errorf("Expected 2 in pile received %d", srv.pile.Count)
		}
		if len(srv.alerts) != 2 {
			t.Errorf("Expected 2 in alert received %d", len(srv.alerts))
		}

		srv.httpClient = &fakeHttpClient{statusCode: http.StatusOK, json: bytes, err: errors.New("Wow")}
		srv.syncWithService(0)
		if client.count != 1 {
			t.Error("Expected request")
		}
		if srv.pile.Count != 2 {
			t.Errorf("Expected 2 in pile received %d", srv.pile.Count)
		}
		if len(srv.alerts) != 2 {
			t.Errorf("Expected 2 in alert received %d", len(srv.alerts))
		}

		srv.httpClient = &fakeHttpClient{fail: true}
		srv.syncWithService(0)
		if client.count != 1 {
			t.Error("Expected request")
		}
		if srv.pile.Count != 2 {
			t.Errorf("Expected 2 in pile received %d", srv.pile.Count)
		}
		if len(srv.alerts) != 2 {
			t.Errorf("Expected 2 in alert received %d", len(srv.alerts))
		}
		srv.httpClient = &fakeHttpClient{statusCode: http.StatusOK, json: bytes}
		srv.syncWithService(0)
		if client.count != 1 {
			t.Error("Expected request")
		}
		if srv.pile.Count != 0 {
			t.Errorf("Expected 0 in pile received %d", srv.pile.Count)
		}
		if srv.alerts != nil {
			t.Errorf("Expected nil in alerts received %v", srv.alerts)
		}
	})

}

func Test_guardClient_sync_loadGuardian(t *testing.T) {

	t.Run("simple", func(t *testing.T) {
		resp := new(spec.SyncMessageResp)
		resp.Guardian = new(spec.GuardianSpec)
		bytes, _ := json.Marshal(resp)
		srv, _ := fakeClient(200, string(bytes))
		g := new(spec.GuardianSpec)

		if got := srv.syncWithService(0); !reflect.DeepEqual(got, g) {
			t.Errorf("guardClient.loadGuardian() = %v, want %v", got, g)
		}
	})
}

func Test_gateClient_initHttpClient(t *testing.T) {
	t.Run("base", func(t *testing.T) {
		srv := &gateClient{
			sid:     "mysid",
			ns:      "myns",
			useCm:   false,
			kubeMgr: &fakeKmgr{},
		}
		srv.initHttpClient(x509.NewCertPool(), false)
	})
}
