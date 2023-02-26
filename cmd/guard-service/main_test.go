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
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
	utils "knative.dev/security-guard/pkg/guard-utils"
)

func addToPile(s *services) {
	profile1 := &spec.SessionDataProfile{}
	profile1.Req.Method.ProfileString("Get")
	pile1 := spec.SessionDataPile{}
	pile1.Add(profile1)
	r1 := s.get("ns", "sid1", false)
	s.merge(r1, &pile1)
}

func Test_learner_mainEventLoop(t *testing.T) {
	quit := make(chan string)

	// services
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	ticker := utils.NewTicker(100000)
	ticker.Parse("", 100000)
	ticker.Start()

	addToPile(s)

	t.Run("simple", func(t *testing.T) {
		l := &learner{
			services:        s,
			pileLearnTicker: ticker,
		}
		if s.cache["sid1.ns"].pile.Count != 1 {
			t.Errorf("Expected 1 in pile  have %d", s.cache["sid1.ns"].pile.Count)
		}

		// Start event loop
		go l.mainEventLoop(quit)

		<-time.After(100 * time.Millisecond)

		if s.cache["sid1.ns"].pile.Count != 0 {
			t.Errorf("Expected 0 in pile  have %d", s.cache["sid1.ns"].pile.Count)
		}
		quit <- "test done"
		// Asked event loop to quit
		<-time.After(100 * time.Millisecond)

		addToPile(s)
		if s.cache["sid1.ns"].pile.Count != 1 {
			t.Errorf("Expected 1 in pile  have %d", s.cache["sid1.ns"].pile.Count)
		}

		<-time.After(100 * time.Millisecond)

		if s.cache["sid1.ns"].pile.Count != 1 {
			t.Errorf("Expected 1 in pile  have %d", s.cache["sid1.ns"].pile.Count)
		}
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
			l.env.GuardServiceAuth = "false"
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
			l.env.GuardServiceAuth = "anything"
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
	l.env.GuardServiceAuth = "anything"

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
	l.env.GuardServiceAuth = "false"
	l.env.GuardServiceTls = "false"
	l.env.GuardServiceInterval = "30s"
	l.env.GuardServiceLabels = []string{"aaa", "bbb"}

	srv, _ := l.init(100000)

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
	l.env.GuardServiceAuth = "anything"
	l.env.GuardServiceTls = "anything"
	l.env.GuardServiceInterval = "asdkasg"
	l.env.GuardServiceLabels = []string{}

	srv, _ := l.init(100000)

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
	l.env.GuardServiceAuth = "anything"

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
	l.env.GuardServiceAuth = "anything"

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
	l.env.GuardServiceAuth = "anything"

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
	l.env.GuardServiceAuth = "anything"

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
	//expected := `null`
	var syncResp spec.SyncMessageResp
	syncResp.Guardian = new(spec.GuardianSpec)
	syncResp.Guardian.SetToMaximalAutomation()
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
	l.env.GuardServiceAuth = "anything"

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
	l.env.GuardServiceAuth = "anything"

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
	//expected := `null`
	var syncResp spec.SyncMessageResp
	syncResp.Guardian = new(spec.GuardianSpec)
	syncResp.Guardian.SetToMaximalAutomation()
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
	l.env.GuardServiceAuth = "false"

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
	//expected := `null`
	var syncResp spec.SyncMessageResp
	syncResp.Guardian = new(spec.GuardianSpec)
	syncResp.Guardian.SetToMaximalAutomation()
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
	l.env.GuardServiceAuth = "false"

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
			l.env.GuardServiceAuth = "anything"

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
