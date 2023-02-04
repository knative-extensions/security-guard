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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"sync"

	"net/http"
	"os"
	"path"
	"time"

	"knative.dev/control-protocol/pkg/certificates"
	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
	guardKubeMgr "knative.dev/security-guard/pkg/guard-kubemgr"
	pi "knative.dev/security-guard/pkg/pluginterfaces"
)

const (
	MAX_ALERTS = 1000
	PILE_LIMIT = 1000
)

type httpClientInterface interface {
	ReadToken(audience string) bool
	Do(req *http.Request) (*http.Response, error)
}

type httpClient struct {
	client           http.Client
	token            string
	tokenRefreshTime time.Time
	missingToken     bool
}

func (hc *httpClient) Do(req *http.Request) (*http.Response, error) {
	if !hc.missingToken {
		// add authorization header - optional for this revision of guard
		req.Header.Add("Authorization", "Bearer "+hc.token)
	}

	return hc.client.Do(req)
}

func (hc *httpClient) ReadToken(audience string) (tokenActive bool) {
	// If not yet tokenRefreshTime, skip reading
	now := time.Now()
	if hc.tokenRefreshTime.After(now) {
		return
	}
	// refresh in 100 minuets
	hc.tokenRefreshTime = now.Add(100 * time.Minute)

	// TODO: replace  "/var/run/secrets/tokens" with sharedMain.QPOptionTokenDirPath once merged.
	b, err := os.ReadFile(path.Join("/var/run/secrets/tokens", audience))

	if err != nil {
		pi.Log.Debugf("Token %s is missing - working without token", audience)
		hc.missingToken = true
		return
	}
	hc.token = string(b)
	hc.missingToken = false
	tokenActive = true

	pi.Log.Debugf("Refreshing client token - next refresh at %s", hc.tokenRefreshTime.String())
	return
}

type gateClient struct {
	guardServiceUrl string
	podname         string
	sid             string
	ns              string
	useCm           bool
	httpClient      httpClientInterface
	pile            spec.SessionDataPile
	pileMutex       sync.Mutex
	alerts          []spec.Alert
	kubeMgr         guardKubeMgr.KubeMgrInterface
}

func NewGateClient(guardServiceUrl string, podname string, sid string, ns string, useCm bool) *gateClient {
	srv := new(gateClient)
	srv.kubeMgr = guardKubeMgr.NewKubeMgr()
	srv.guardServiceUrl = guardServiceUrl
	srv.podname = podname
	srv.sid = sid
	srv.ns = ns
	srv.useCm = useCm

	srv.pile.Clear()

	return srv
}

func (srv *gateClient) initKubeMgr() {
	// initializtion that cant be tested due to use of KubeAMgr
	srv.kubeMgr.InitConfigs()
}

func (srv *gateClient) initHttpClient(certPool *x509.CertPool) (tokenActive bool) {
	client := new(httpClient)
	client.client.Transport = &http.Transport{
		MaxConnsPerHost:     0,
		MaxIdleConns:        0,
		MaxIdleConnsPerHost: 0,
		TLSClientConfig: &tls.Config{
			ServerName: certificates.FakeDnsName,
			RootCAs:    certPool,
		},
	}
	srv.httpClient = client
	tokenActive = srv.httpClient.ReadToken(guardKubeMgr.ServiceAudience)
	return
}

func (srv *gateClient) syncWithService() *spec.GuardianSpec {
	var syncReq spec.SyncMessageReq
	var syncResp spec.SyncMessageResp

	srv.httpClient.ReadToken(guardKubeMgr.ServiceAudience)

	syncReq.Alerts = srv.alerts
	syncReq.Pile = &srv.pile

	// protect pile internals read/write
	srv.pileMutex.Lock()
	postBody, marshalErr := json.Marshal(syncReq)
	// We clear pile event if we dont know if the pile is sent to the service - to avoid accumulating forever
	srv.pileMutex.Unlock()
	// Must unlock srv.pileMutex before http.NewRequest

	if marshalErr != nil {
		// should never happen
		pi.Log.Infof("Error during marshal: %v", marshalErr)
		return nil
	}
	reqBody := bytes.NewBuffer(postBody)
	req, err := http.NewRequest(http.MethodPost, srv.guardServiceUrl+"/sync", reqBody)
	if err != nil {
		pi.Log.Infof("Http.NewRequest error %v", err)
		return nil
	}
	query := req.URL.Query()
	query.Add("sid", srv.sid)
	query.Add("ns", srv.ns)
	query.Add("pod", srv.podname)
	if srv.useCm {
		query.Add("cm", "true")
	}
	req.URL.RawQuery = query.Encode()
	pi.Log.Debugf("Sync with guard-service!")

	res, postErr := srv.httpClient.Do(req)
	if postErr != nil {
		pi.Log.Infof("httpClient.Do error %v", postErr)
		return nil
	}
	if res.StatusCode != http.StatusOK {
		pi.Log.Infof("guard-service did not respond with 200 OK")
		return nil
	}

	if res.Body == nil {
		pi.Log.Infof("guard-service did not accept sync - no response given")
		return nil
	}
	defer res.Body.Close()
	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		pi.Log.Infof("Response error %v", readErr)
		return nil
	}
	if len(body) == 0 {
		pi.Log.Infof("guard-service did not accept sync - response is empty")
		return nil
	}

	jsonErr := json.Unmarshal(body, &syncResp)
	if jsonErr != nil {
		pi.Log.Infof("GuardianSpec: unmarshel error %v", jsonErr)
		return nil
	}
	pi.Log.Debugf("loadGuardianFromService: accepted guardian from guard-service")

	// We clear only when we know pile and alerts were delivered
	srv.alerts = nil
	srv.pile.Clear()
	return syncResp.Guardian
}

func (srv *gateClient) addAlert(decision *spec.Decision, level string) {
	srv.alerts = spec.AddAlert(srv.alerts, decision, level)

	if numAlerts := len(srv.alerts); numAlerts > MAX_ALERTS {
		srv.alerts = srv.alerts[:numAlerts-1]
	}
}

func (srv *gateClient) addToPile(profile *spec.SessionDataProfile) uint32 {
	if srv.pile.Count < 2*PILE_LIMIT {
		// protect pile internals read/write
		srv.pileMutex.Lock()
		srv.pile.Add(profile)
		srv.pileMutex.Unlock()
		// Must unlock srv.pileMutex before srv.reportPile
	}
	pi.Log.Debugf("Learn - add to pile! pileCount %d", srv.pile.Count)
	return srv.pile.Count
}

func (srv *gateClient) syncWithServiceAndKubeApi() *spec.GuardianSpec {
	wsGate := srv.syncWithService()
	if wsGate == nil {
		// never return nil!
		wsGate = srv.kubeMgr.GetGuardian(srv.ns, srv.sid, srv.useCm, true)
	}
	return wsGate
}
