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
	"os"
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
	log = utils.CreateLogger("x")
	quit := make(chan string)

	// services
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	utils.MinimumInterval = 1000
	ticker := new(utils.Ticker)
	ticker.Parse("", 1000)
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

func Test_learner_baseHandler(t *testing.T) {
	log = utils.CreateLogger("x")

	tests := []struct {
		name       string
		query      url.Values
		wantNs     string
		wantSid    string
		wantCmFlag bool
		wantRecord *serviceRecord
	}{
		{
			name:  "empty",
			query: url.Values{},
		},
		{
			name:  "noNs",
			query: url.Values{"sid": []string{"x"}},
		},
		{
			name:  "noSid",
			query: url.Values{"ns": []string{"x"}},
		},
		{
			name:  "doubleSid",
			query: url.Values{"ns": []string{"x"}, "sid": []string{"x", "y"}},
		},
		{
			name:  "doubleNs",
			query: url.Values{"ns": []string{"x", "y"}, "sid": []string{"x"}},
		},
		{
			name:  "doubleCm",
			query: url.Values{"ns": []string{"x"}, "sid": []string{"x"}, "cm": []string{"x", "y"}},
		},
		{
			name:       "ok",
			query:      url.Values{"ns": []string{"x"}, "sid": []string{"x"}},
			wantNs:     "x",
			wantSid:    "x",
			wantRecord: &serviceRecord{ns: "x", sid: "x", guardianSpec: new(spec.GuardianSpec)},
		},
		{
			name:       "okWithBadCm",
			query:      url.Values{"ns": []string{"x"}, "sid": []string{"x"}, "cm": []string{"x"}},
			wantNs:     "x",
			wantSid:    "x",
			wantRecord: &serviceRecord{ns: "x", sid: "x", guardianSpec: new(spec.GuardianSpec)},
		},
		{
			name:       "okWithTrueCm",
			query:      url.Values{"ns": []string{"x"}, "sid": []string{"x"}, "cm": []string{"true"}},
			wantNs:     "x",
			wantSid:    "x",
			wantCmFlag: true,
			wantRecord: &serviceRecord{ns: "x", sid: "x", cmFlag: true, guardianSpec: new(spec.GuardianSpec)},
		},
		{
			name:       "okWithFalseCm",
			query:      url.Values{"ns": []string{"x"}, "sid": []string{"x"}, "cm": []string{"false"}},
			wantNs:     "x",
			wantSid:    "x",
			wantRecord: &serviceRecord{ns: "x", sid: "x", guardianSpec: new(spec.GuardianSpec)},
		},
		{
			name:    "bad sid",
			query:   url.Values{"ns": []string{"x"}, "sid": []string{"ns-zz"}},
			wantNs:  "x",
			wantSid: "",
		},
	}
	for _, tt := range tests {
		// services
		s := new(services)
		s.cache = make(map[string]*serviceRecord, 64)
		s.namespaces = make(map[string]bool, 4)
		s.kmgr = new(fakeKmgr)

		utils.MinimumInterval = 1000
		ticker := new(utils.Ticker)
		if tt.wantRecord != nil {
			tt.wantRecord.pile.Clear()
		}
		t.Run(tt.name, func(t *testing.T) {
			l := &learner{
				services:        s,
				pileLearnTicker: ticker,
			}
			gotNs, gotSid, gotCmFlag, gotRecord := l.baseHandler(tt.query)
			if gotNs != tt.wantNs {
				t.Errorf("learner.baseHandler() gotNs = %v, want %v", gotNs, tt.wantNs)
			}
			if gotSid != tt.wantSid {
				t.Errorf("learner.baseHandler() gotSid = %v, want %v", gotSid, tt.wantSid)
			}
			if gotCmFlag != tt.wantCmFlag {
				t.Errorf("learner.baseHandler() gotCmFlag = %v, want %v", gotCmFlag, tt.wantCmFlag)
			}
			if !reflect.DeepEqual(gotRecord, tt.wantRecord) {
				t.Errorf("learner.baseHandler() gotRecord = %v, want %v", gotRecord, tt.wantRecord)
			}
		})
	}
}

func TestFetchConfigHandler_NoQuery(t *testing.T) {
	log = utils.CreateLogger("x")
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	utils.MinimumInterval = 1000
	l, _, _, _ := _main()
	l.services = s

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/config", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.fetchConfig)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	// Check the response body is what we expect.
	buf := make([]byte, 0)
	if reflect.DeepEqual(rr.Body, buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), buf)
	}

}

func TestFetchConfigHandler_main(t *testing.T) {
	os.Unsetenv("GUARD_SERVICE_PORT")
	_, _, target, _ := _main()

	if target != ":8888" {
		t.Errorf("handler returned wrong default target code: got %s want %s", target, ":8888")
	}

	os.Setenv("GUARD_SERVICE_PORT", "9999")
	_, _, target, _ = _main()

	if target != ":9999" {
		t.Errorf("handler returned wrong default target code: got %s want %s", target, ":9999")
	}

	os.Unsetenv("GUARD_SERVICE_PORT")
}

func TestFetchConfigHandler_POST(t *testing.T) {
	log = utils.CreateLogger("x")
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	utils.MinimumInterval = 1000
	ticker := new(utils.Ticker)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("POST", "/config", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.fetchConfig)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	// Check the response body is what we expect.
	buf := make([]byte, 0)
	if reflect.DeepEqual(rr.Body, buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), buf)
	}
}

func TestFetchConfigHandler_WithQuery(t *testing.T) {
	log = utils.CreateLogger("x")
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	utils.MinimumInterval = 1000
	ticker := new(utils.Ticker)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/config?sid=x&ns=x", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.fetchConfig)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	//expected := `{"configured": null,"control":null}`
	g := new(spec.GuardianSpec)
	buf, _ := json.Marshal(g)
	if reflect.DeepEqual(rr.Body, buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), buf)
	}
}

func TestProcessPileHandler_NoQuery(t *testing.T) {
	log = utils.CreateLogger("x")
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	utils.MinimumInterval = 1000
	ticker := new(utils.Ticker)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/pile", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.processPile)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	// Check the response body is what we expect.
	buf := make([]byte, 0)
	if reflect.DeepEqual(rr.Body, buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), buf)
	}
}

func TestProcessPileHandler_WithQueryAndPile(t *testing.T) {
	log = utils.CreateLogger("x")
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	utils.MinimumInterval = 1000
	ticker := new(utils.Ticker)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}
	record := s.get("ns", "sid9", false)
	postBody, _ := json.Marshal(&record.pile)
	reqBody := bytes.NewBuffer(postBody)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/pile?sid=x&ns=x", reqBody)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.processPile)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	//expected := `{"configured": null,"control":null}`
	g := new(spec.GuardianSpec)
	buf, _ := json.Marshal(g)
	if reflect.DeepEqual(rr.Body, buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), buf)
	}
}

func TestProcessPileHandler_WithQueryAndNoPile(t *testing.T) {
	log = utils.CreateLogger("x")
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	utils.MinimumInterval = 1000
	ticker := new(utils.Ticker)
	l := &learner{
		services:        s,
		pileLearnTicker: ticker,
	}
	postBody, _ := json.Marshal("xx")
	reqBody := bytes.NewBuffer(postBody)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/pile?sid=x&ns=x", reqBody)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(l.processPile)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	// Check the response body is what we expect.
	buf := make([]byte, 0)
	if reflect.DeepEqual(rr.Body, buf) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), buf)
	}
}
