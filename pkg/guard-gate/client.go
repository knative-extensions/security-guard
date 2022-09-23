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
	"io/ioutil"
	"net/http"

	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
	guardKubeMgr "knative.dev/security-guard/pkg/guard-kubemgr"
	pi "knative.dev/security-guard/pkg/pluginterfaces"
)

type httpClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

type httpClient struct {
	client *http.Client
}

func (hc *httpClient) Do(req *http.Request) (*http.Response, error) {
	return hc.client.Do(req)
}

type gateClient struct {
	guardServiceUrl string
	sid             string
	ns              string
	useCm           bool
	httpClient      httpClientInterface
	pile            spec.SessionDataPile
	kubeMgr         guardKubeMgr.KubeMgrInterface
}

func NewGateClient(guardServiceUrl string, sid string, ns string, useCm bool) *gateClient {
	srv := new(gateClient)
	srv.guardServiceUrl = guardServiceUrl
	srv.sid = sid
	srv.ns = ns
	srv.useCm = useCm
	srv.httpClient = new(httpClient)
	srv.clearPile()
	srv.kubeMgr = guardKubeMgr.NewKubeMgr()
	return srv
}

func (srv *gateClient) start() {
	// initializtion that cant be tested due to use of KubeAMgr
	srv.kubeMgr.InitConfigs()
}

func (srv *gateClient) reportPile() {
	if srv.pile.Count == 0 {
		pi.Log.Debugf("No pile to report to guard-service!")
		return
	}
	defer srv.clearPile()

	pi.Log.Infof("Reporting a pile with pileCount %d records to guard-service", srv.pile.Count)

	postBody, marshalErr := json.Marshal(srv.pile)

	if marshalErr != nil {
		// should never happen
		pi.Log.Warnf("Error during marshal: %v", marshalErr)
		return
	}
	reqBody := bytes.NewBuffer(postBody)
	req, err := http.NewRequest(http.MethodPost, srv.guardServiceUrl+"/pile", reqBody)
	if err != nil {
		pi.Log.Warnf("Http.NewRequest error %v", err)
		return
	}
	query := req.URL.Query()
	query.Add("sid", srv.sid)
	query.Add("ns", srv.ns)
	if srv.useCm {
		query.Add("cm", "true")
	}
	req.URL.RawQuery = query.Encode()

	res, postErr := srv.httpClient.Do(req)
	if postErr != nil {
		pi.Log.Warnf("httpClient.Do error %v", postErr)
		return
	}
	if res.Body != nil {
		defer res.Body.Close()
		body, readErr := ioutil.ReadAll(res.Body)
		if readErr != nil {
			pi.Log.Infof("Response error %v", readErr)
			return
		}
		if len(body) != 0 {
			pi.Log.Infof("guard-service response is %s", string(body))
		}
	}
}

func (srv *gateClient) addToPile(profile *spec.SessionDataProfile) {
	srv.pile.Add(profile)
	pi.Log.Debugf("Learn - add to pile! pileCount %d", srv.pile.Count)
}

func (srv *gateClient) clearPile() {
	srv.pile.Clear()
}

func (srv *gateClient) loadGuardian() *spec.GuardianSpec {
	wsGate := srv.loadGuardianFromService()
	if wsGate == nil {
		// never return nil!
		wsGate = srv.kubeMgr.GetGuardian(srv.ns, srv.sid, srv.useCm, true)
	}
	return wsGate
}

func (srv *gateClient) loadGuardianFromService() *spec.GuardianSpec {
	req, err := http.NewRequest(http.MethodGet, srv.guardServiceUrl+"/config", nil)
	if err != nil {
		pi.Log.Warnf("loadGuardianFromService Http.NewRequest error %v", err)
		return nil
	}
	query := req.URL.Query()
	query.Add("sid", srv.sid)
	query.Add("ns", srv.ns)
	if srv.useCm {
		query.Add("cm", "true")
	}
	req.URL.RawQuery = query.Encode()

	res, err := srv.httpClient.Do(req)
	if err != nil {
		pi.Log.Warnf("loadGuardianFromService httpClient.Do error %v", err)
		return nil
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		pi.Log.Infof("loadGuardianFromService Response error %v", err)
		return nil
	}
	if len(body) == 0 {
		pi.Log.Infof("loadGuardianFromService Response empty")
		return nil
	}
	pi.Log.Debugf("loadGuardianFromService: accepted guardian from guard-service")

	g := new(spec.GuardianSpec)
	jsonErr := json.Unmarshal(body, g)
	if jsonErr != nil {
		pi.Log.Infof("loadGuardianFromService GuardianSpec: unmarshel error %v", jsonErr)
		return nil
	}
	return g
}
