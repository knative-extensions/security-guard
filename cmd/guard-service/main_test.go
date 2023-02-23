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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
	utils "knative.dev/security-guard/pkg/guard-utils"
)

func addSample(s *services) {
	profile1 := &spec.SessionDataProfile{}
	profile1.Req.Method.ProfileString("Get")
	pile1 := spec.SessionDataPile{}
	pile1.Add(profile1)
	r1 := s.get("ns", "sid1", false)
	s.mergeAndLearnAndPersistGuardian(r1, &pile1)
}

func testStatus(txt string, s *services, t *testing.T, pile uint32, guardian uint32, meregd uint, learned uint, persisted uint) {
	if s.cache["sid1.ns"].pile.Count != pile {
		t.Errorf("During %s - pile.Count Expected %d in pile have %d", txt, pile, s.cache["sid1.ns"].pile.Count)
	}
	if s.cache["sid1.ns"].guardianSpec.NumSamples != guardian {
		t.Errorf("During %s - guardianSpec.NumSamples Expected %d in guardianSpec have %d", txt, guardian, s.cache["sid1.ns"].guardianSpec.NumSamples)
	}
	if s.cache["sid1.ns"].pileMergeCounter != meregd {
		t.Errorf("During %s -pileMergeCounter Expected %d in pileMergeCounter have %d", txt, meregd, s.cache["sid1.ns"].pileMergeCounter)
	}
	if s.cache["sid1.ns"].guardianLearnCounter != learned {
		t.Errorf("During %s - guardianLearnCounter Expected %d in guardianLearnCounter have %d", txt, learned, s.cache["sid1.ns"].guardianLearnCounter)
	}
	if s.cache["sid1.ns"].guardianPersistCounter != persisted {
		t.Errorf("During %s - guardianPersistCounter Expected %d in guardianPersistCounter have %d", txt, persisted, s.cache["sid1.ns"].guardianPersistCounter)
	}
}

func Test_learner_mainEventLoop(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		quit := make(chan string, 1)

		// services
		s := new(services)
		s.cache = make(map[string]*serviceRecord, 64)
		s.namespaces = make(map[string]bool, 4)
		s.kmgr = new(fakeKmgr)

		ticker := utils.NewTicker(100000)
		ticker.Parse("", 100000)
		ticker.Start()
		kill := make(chan os.Signal, 1)
		l := &learner{
			services:        s,
			pileLearnTicker: ticker,
		}
		for i := uint(1); i <= 10; i++ {
			addSample(s)
			// 10% rule - immidiate learning
			testStatus(fmt.Sprintf("sample #%d", i), s, t, 0, uint32(i), i, i, 1)
		}

		// no longer 10% rule
		addSample(s)
		testStatus("11th sample", s, t, 1, 10, 11, 10, 1)

		// Start event loop - no longer 10% rule
		l.mainEventProcessing(quit, kill)
		testStatus("Main Loop", s, t, 1, 10, 11, 10, 1)

		// Start event loop - no longer 10% rule, but 30 seconds passed since we learned
		s.cache["sid1.ns"].pileLastLearn = time.Unix(0, 0)
		l.mainEventProcessing(quit, kill)
		testStatus("Main Loop with pileLastLearn reset", s, t, 0, 11, 11, 11, 1)

		// Start event loop - 5 min passed since we persisted
		s.cache["sid1.ns"].guardianLastPersist = time.Unix(0, 0)
		l.mainEventProcessing(quit, kill)
		// blocked by lastCreatedRecords
		testStatus("Main Loop with guardianLastPersist reset", s, t, 0, 11, 11, 11, 1)

		// Start event loop - also 5 min passed since we reviewed all records - first tick, build records
		s.lastCreatedRecords = time.Unix(0, 0)
		l.mainEventProcessing(quit, kill)
		testStatus("Main Loop with guardianLastPersist reset - first time", s, t, 0, 11, 11, 11, 1)

		// Start event loop - also 5 min passed since we reviewed all records - second tick, persist
		l.mainEventProcessing(quit, kill)
		testStatus("Main Loop with lastCreatedRecords reset - second time", s, t, 0, 11, 11, 11, 2)

		// Ask event loop to quit
		addSample(s)
		quit <- "test done"
		l.mainEventLoop(quit, kill)
		testStatus("Main Loop with quit", s, t, 0, 12, 12, 12, 3)
	})

}

func Test_NOTLS_learner_baseHandler(t *testing.T) {
	tests := []struct {
		name       string
		query      url.Values
		wantErr    bool
		wantSid    string
		wantNs     string
		wantPod    string
		wantCmFlag bool
	}{
		{
			name:    "doubleCm",
			query:   url.Values{"ns": []string{"myns"}, "pod": []string{"mypod"}, "sid": []string{"x"}, "cm": []string{"x", "y"}},
			wantErr: true,
		},
		{
			name:    "doubleNs",
			query:   url.Values{"ns": []string{"myns", "anotherns"}, "pod": []string{"mypod"}, "sid": []string{"x"}},
			wantErr: true,
		},
		{
			name:    "emptyNS",
			query:   url.Values{"ns": []string{""}, "pod": []string{"mypod"}, "sid": []string{"x"}},
			wantErr: true,
		},
		{
			name:    "emptySid",
			query:   url.Values{"ns": []string{"myns"}, "pod": []string{"mypod"}, "sid": []string{""}},
			wantErr: true,
		},
		{
			name:    "emptyPod",
			query:   url.Values{"ns": []string{"myns"}, "pod": []string{""}, "sid": []string{"x"}},
			wantErr: true,
		},
		{
			name:    "badSid",
			query:   url.Values{"ns": []string{"myns"}, "pod": []string{"mypod"}, "sid": []string{"ns-myns"}},
			wantErr: true,
		},
		{
			name:    "ok",
			query:   url.Values{"ns": []string{"myns"}, "pod": []string{"mypod"}, "sid": []string{"x"}},
			wantPod: "mypod",
			wantSid: "x",
			wantNs:  "myns",
		},
		{
			name:    "okWithBadCm",
			query:   url.Values{"ns": []string{"myns"}, "pod": []string{"mypod"}, "sid": []string{"x"}, "cm": []string{"x"}},
			wantPod: "mypod",
			wantSid: "x",
			wantNs:  "myns",
		},
		{
			name:       "okWithTrueCm",
			query:      url.Values{"ns": []string{"myns"}, "pod": []string{"mypod"}, "sid": []string{"x"}, "cm": []string{"true"}},
			wantPod:    "mypod",
			wantSid:    "x",
			wantNs:     "myns",
			wantCmFlag: true,
		},
		{
			name:    "okWithFalseCm",
			query:   url.Values{"ns": []string{"myns"}, "pod": []string{"mypod"}, "sid": []string{"x"}, "cm": []string{"false"}},
			wantPod: "mypod",
			wantSid: "x",
			wantNs:  "myns",
		},
	}
	for _, tt := range tests {
		// services
		s := new(services)
		s.cache = make(map[string]*serviceRecord, 64)
		s.namespaces = make(map[string]bool, 4)
		s.kmgr = new(fakeKmgr)

		ticker := utils.NewTicker(100000)
		t.Run(tt.name, func(t *testing.T) {
			l := &learner{
				services:        s,
				pileLearnTicker: ticker,
			}
			l.env.GuardServiceAuth = false
			gotCmFlag, gotPod, gotSid, gotNs, gotErr := l.queryDataNoAuth(tt.query)
			if tt.wantErr == (gotErr == nil) {
				t.Errorf("learner.queryData() gotErr = %v, want %v", gotErr, tt.wantErr)
			}
			if tt.wantErr || (gotErr != nil) {
				return
			}
			if tt.wantCmFlag != gotCmFlag {
				t.Errorf("learner.queryData() wantCmFlag = %v, and gotCmFlag %v", tt.wantCmFlag, gotCmFlag)
			}
			if tt.wantSid != gotSid {
				t.Errorf("learner.queryData() wantSid = %v, and gotSid %v", tt.wantSid, gotSid)
			}
			if tt.wantNs != gotNs {
				t.Errorf("learner.queryData() wantNs = %v, and gotNs %v", tt.wantNs, gotNs)
			}
			if tt.wantPod != gotPod {
				t.Errorf("learner.queryData() wantPod = %v, and gotPod %v", tt.wantPod, gotPod)
			}
		})
	}
}

func Test_TLS_learner_baseHandler(t *testing.T) {
	tests := []struct {
		name       string
		query      url.Values
		wantErr    bool
		wantCmFlag bool
	}{
		{
			name:    "doubleCm",
			query:   url.Values{"ns": []string{"myns"}, "pod": []string{"mypod"}, "sid": []string{"x"}, "cm": []string{"x", "y"}},
			wantErr: true,
		},
		{
			name:  "ok",
			query: url.Values{"ns": []string{"myns"}, "pod": []string{"mypod"}, "sid": []string{"xxx"}},
		},
		{
			name:  "okWithBadCm",
			query: url.Values{"ns": []string{"myns"}, "pod": []string{"mypod"}, "sid": []string{"x"}, "cm": []string{"x"}},
		},
		{
			name:       "okWithTrueCm",
			query:      url.Values{"ns": []string{"myns"}, "pod": []string{"mypod"}, "sid": []string{"x"}, "cm": []string{"true"}},
			wantCmFlag: true,
		},
		{
			name:  "okWithFalseCm",
			query: url.Values{"ns": []string{"myns"}, "pod": []string{"mypod"}, "sid": []string{"x"}, "cm": []string{"false"}},
		},
	}
	for _, tt := range tests {
		// services
		s := new(services)
		s.cache = make(map[string]*serviceRecord, 64)
		s.namespaces = make(map[string]bool, 4)
		s.kmgr = new(fakeKmgr)

		ticker := utils.NewTicker(100000)
		t.Run(tt.name, func(t *testing.T) {
			l := &learner{
				services:        s,
				pileLearnTicker: ticker,
			}
			l.env.GuardServiceAuth = true
			gotCmFlag, gotErr := l.queryDataAuth(tt.query)
			if tt.wantErr == (gotErr == nil) {
				t.Errorf("learner.queryData() gotErr = %v, want %v", gotErr, tt.wantErr)
			}
			if tt.wantErr || (gotErr != nil) {
				return
			}
			if tt.wantCmFlag != gotCmFlag {
				t.Errorf("learner.queryData() wantCmFlag = %v, and gotCmFlag %v", tt.wantCmFlag, gotCmFlag)
			}
		})
	}
}

func TestTLS_SyncHandler_MissingToken(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/sync", nil)
	if err != nil {
		t.Fatal(err)
	}
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	ticker := utils.NewTicker(100000)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}
	l.env.GuardServiceAuth = true

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.processSync)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	status := rr.Code
	if status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusUnauthorized)
	}

	// Check the response body is what we expect.
	buf := []byte{}
	if reflect.DeepEqual(rr.Body, buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), buf)
	}

}

func TestNOTLS_SyncHandler_main(t *testing.T) {
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	ticker := utils.NewTicker(100000)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}
	l.env.GuardServiceAuth = false
	l.env.GuardServiceTls = false
	l.env.GuardServiceLabels = []string{"aaa", "bbb"}

	srv, _ := l.init()

	if srv.Addr != ":8888" {
		t.Errorf("handler returned wrong default target code: got %s want %s", srv.Addr, ":8888")
	}
}

func TestTLS_SyncHandler_main(t *testing.T) {
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	ticker := utils.NewTicker(100000)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}
	l.env.GuardServiceAuth = true
	l.env.GuardServiceTls = true
	l.env.GuardServiceLabels = []string{}

	srv, _ := l.init()

	if srv.Addr != ":8888" {
		t.Errorf("handler returned wrong default target code: got %s want %s", srv.Addr, ":555")
	}
}

func TestTLS_SyncHandler_NotPOST(t *testing.T) {
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	ticker := utils.NewTicker(100000)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}
	l.env.GuardServiceAuth = true

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/sync", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer abc")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.processSync)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}

	// Check the response body is what we expect.
	buf := []byte{}
	if reflect.DeepEqual(rr.Body, buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), buf)
	}
}

func TestTLS_badUrl(t *testing.T) {
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	ticker := utils.NewTicker(100000)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}
	l.env.GuardServiceAuth = true

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("POST", "/xxx", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer abc")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.processSync)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}

	// Check the response body is what we expect.
	buf := []byte{}
	if reflect.DeepEqual(rr.Body, buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), buf)
	}
}

func TestTLS_SyncHandler_NoReqBody(t *testing.T) {
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	ticker := utils.NewTicker(100000)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}
	l.env.GuardServiceAuth = true

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("POST", "/sync", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer abc")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.processSync)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	// Check the response body is what we expect.
	buf := []byte{}
	if reflect.DeepEqual(rr.Body, buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), buf)
	}
}

func TestTLS_SyncHandler_EmptyPileAndAlert(t *testing.T) {
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	ticker := utils.NewTicker(100000)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}
	l.env.GuardServiceAuth = true

	record := s.get("ns", "sid9", false)
	postBody, _ := json.Marshal(&record.pile)
	reqBody := bytes.NewBuffer(postBody)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("POST", "/sync", reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer abc")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.processSync)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	var syncResp spec.SyncMessageResp
	syncResp.Guardian = new(spec.GuardianSpec)
	syncResp.Guardian.SetToMaximalAutomation()
	syncResp.Guardian.Learned = &spec.SessionDataConfig{}
	buf, _ := json.Marshal(syncResp)
	if !bytes.Equal(rr.Body.Bytes(), buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(buf))
	}
}

func TestTLS_SyncHandler_WithBadReq(t *testing.T) {
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	ticker := utils.NewTicker(100000)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}
	l.env.GuardServiceAuth = true

	postBody, _ := json.Marshal("xx")
	reqBody := bytes.NewBuffer(postBody)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("POST", "/sync", reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer abc")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.processSync)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	// Check the response body is what we expect.
	buf := []byte{}
	if reflect.DeepEqual(rr.Body, buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), buf)
	}
}

func TestTLS_SyncHandler_WithGoodReq(t *testing.T) {
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	ticker := utils.NewTicker(100000)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}
	l.env.GuardServiceAuth = true

	var syncReq spec.SyncMessageReq
	var decision *spec.Decision
	spec.DecideInner(&decision, 7, "problemFound")
	syncReq.Alerts = spec.AddAlert(syncReq.Alerts, decision, "Session")

	profile := &spec.SessionDataProfile{}
	profile.Req.Method.ProfileString("Get")
	pile := spec.SessionDataPile{}
	pile.Add(profile)
	syncReq.Pile = &pile

	postBody, _ := json.Marshal(syncReq)
	reqBody := bytes.NewBuffer(postBody)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("POST", "/sync", reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer abc")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.processSync)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	var syncResp spec.SyncMessageResp
	syncResp.Guardian = new(spec.GuardianSpec)
	syncResp.Guardian.NumSamples = 1
	syncResp.Guardian.SetToMaximalAutomation()
	syncResp.Guardian.Learned = &spec.SessionDataConfig{Active: true}
	syncResp.Guardian.Learned.Req.Method.List = []string{"Get"}
	syncResp.Guardian.Learned.Req.ContentLength = append(syncResp.Guardian.Learned.Req.ContentLength, spec.CountRange{Min: 0, Max: 0})
	syncResp.Guardian.Learned.Req.Url.Segments = append(syncResp.Guardian.Learned.Req.Url.Segments, spec.CountRange{Min: 0, Max: 0})
	syncResp.Guardian.Learned.Resp.ContentLength = append(syncResp.Guardian.Learned.Resp.ContentLength, spec.CountRange{Min: 0, Max: 0})
	buf, _ := json.Marshal(syncResp)
	if !bytes.Equal(rr.Body.Bytes(), buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(buf))
	}
}
func TestNOTLS_SyncHandler_WithGoodReq(t *testing.T) {
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	ticker := utils.NewTicker(100000)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}
	l.env.GuardServiceAuth = false

	var syncReq spec.SyncMessageReq
	var decision *spec.Decision
	spec.DecideInner(&decision, 7, "problemFound")
	syncReq.Alerts = spec.AddAlert(syncReq.Alerts, decision, "Session")

	profile := &spec.SessionDataProfile{}
	profile.Req.Method.ProfileString("Get")
	pile := spec.SessionDataPile{}
	pile.Add(profile)
	syncReq.Pile = &pile

	postBody, _ := json.Marshal(syncReq)
	reqBody := bytes.NewBuffer(postBody)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("POST", "/sync?sid=x&ns=x&pod=x", reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer abc")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.processSync)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	var syncResp spec.SyncMessageResp
	syncResp.Guardian = new(spec.GuardianSpec)
	syncResp.Guardian.NumSamples = 1
	syncResp.Guardian.SetToMaximalAutomation()
	syncResp.Guardian.Learned = &spec.SessionDataConfig{Active: true}
	syncResp.Guardian.Learned.Req.Method.List = []string{"Get"}
	syncResp.Guardian.Learned.Req.ContentLength = append(syncResp.Guardian.Learned.Req.ContentLength, spec.CountRange{Min: 0, Max: 0})
	syncResp.Guardian.Learned.Req.Url.Segments = append(syncResp.Guardian.Learned.Req.Url.Segments, spec.CountRange{Min: 0, Max: 0})
	syncResp.Guardian.Learned.Resp.ContentLength = append(syncResp.Guardian.Learned.Resp.ContentLength, spec.CountRange{Min: 0, Max: 0})

	buf, _ := json.Marshal(syncResp)
	if !bytes.Equal(rr.Body.Bytes(), buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(buf))
	}
}

func TestNOTLS_SyncHandler_WithBadQuery(t *testing.T) {
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	ticker := utils.NewTicker(100000)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}
	l.env.GuardServiceAuth = false

	var syncReq spec.SyncMessageReq
	var decision *spec.Decision
	spec.DecideInner(&decision, 7, "problemFound")
	syncReq.Alerts = spec.AddAlert(syncReq.Alerts, decision, "Session")

	profile := &spec.SessionDataProfile{}
	profile.Req.Method.ProfileString("Get")
	pile := spec.SessionDataPile{}
	pile.Add(profile)
	syncReq.Pile = &pile

	postBody, _ := json.Marshal(syncReq)
	reqBody := bytes.NewBuffer(postBody)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("POST", "/sync?sid=x&ns=x&pod=", reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer abc")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.processSync)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	// Check the response body is what we expect.
	buf := []byte{}
	if reflect.DeepEqual(rr.Body, buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), buf)
	}
}

func Test_learner_authenticate(t *testing.T) {

	tests := []struct {
		name          string
		authorization string
		wantPodname   string
		wantSid       string
		wantNs        string
		wantErr       bool
	}{
		{
			name:          "simple",
			authorization: "Bearer abc",
			wantPodname:   "mypod",
			wantSid:       "mysid",
			wantNs:        "myns",
			wantErr:       false,
		},
		{
			name:          "no Bearer",
			authorization: "abc",
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/sync", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Add("Authorization", tt.authorization)

			s := new(services)
			s.cache = make(map[string]*serviceRecord, 64)
			s.namespaces = make(map[string]bool, 4)
			s.kmgr = new(fakeKmgr)

			ticker := utils.NewTicker(100000)
			l := &learner{
				services:        s,
				pileLearnTicker: ticker,
			}
			l.env.GuardServiceAuth = true

			gotPodname, gotSid, gotNs, err := l.authenticate(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("learner.authenticate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPodname != tt.wantPodname {
				t.Errorf("learner.authenticate() gotPodname = %v, want %v", gotPodname, tt.wantPodname)
			}
			if gotSid != tt.wantSid {
				t.Errorf("learner.authenticate() gotSid = %v, want %v", gotSid, tt.wantSid)
			}
			if gotNs != tt.wantNs {
				t.Errorf("learner.authenticate() gotNs = %v, want %v", gotNs, tt.wantNs)
			}
		})
	}
}
