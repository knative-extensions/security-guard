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
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/emicklei/go-restful"
)

var sessionCanceled int

func fakeSessionCancel() {
	sessionCanceled++
}

func Test_SessionInContext(t *testing.T) {
	ctx1 := context.Background()
	gs := fakeGateState()
	gs.sync(true, false, gs.getTicks())

	t.Run("simple", func(t *testing.T) {
		s1 := newSession(gs, nil, gs.getTicks())
		s1.cancelFunc = fakeSessionCancel
		ctx2 := s1.addSessionToContext(ctx1)
		s2 := getSessionFromContext(ctx2)

		if !reflect.DeepEqual(s1, s2) {
			t.Errorf("received a different session %v, want %v", s2, s1)
		}
		if s2.hasAlert() {
			t.Errorf("expected no alert")
		}

		sessionCanceled = 0
		if s2.cancel(); sessionCanceled != 1 {
			t.Errorf("expected canceled")
		}

		bodyReader := strings.NewReader(`{"Username": "12124", "Password": "testinasg", "Channel": "M"}`)

		req, _ := http.NewRequest("POST", "/config", bodyReader)
		req.Header.Set("Content-Type", restful.MIME_JSON)
		s2.screenRequest(req)
		if !s2.hasAlert() {
			t.Errorf("expected alert")
		}
		s2.decision = nil
		req.RemoteAddr = "1.2.3.4:80"
		s2.screenRequest(req)
		if s2.hasAlert() {
			t.Errorf("expected no alert")
		}
		s2.gateState.criteria.Active = true
		s2.screenRequest(req)
		if !s2.hasAlert() {
			t.Errorf("expected alert")
		}
		s2.decision = nil
		s2.gateState.criteria.Active = false

		td, _ := time.ParseDuration("1s")
		time.Sleep(td)
		s2.screenEnvelop(s2.gateState.getTicks())
		if s2.hasAlert() {
			t.Errorf("expected no alert")
		}
		s2.gateState.criteria.Active = true
		s2.screenEnvelop(s2.gateState.getTicks())
		if !s2.hasAlert() {
			t.Errorf("expected alert")
		}
		s2.decision = nil
		s2.gateState.criteria.Active = false

		resp := &http.Response{ContentLength: 20}
		s2.screenResponse(resp, s2.gateState.getTicks())
		if s2.hasAlert() {
			t.Errorf("expected no alert")
		}
		s2.gateState.criteria.Active = true
		s2.screenResponse(resp, s2.gateState.getTicks())
		if !s2.hasAlert() {
			t.Errorf("expected alert")
		}
		s2.decision = nil
		s2.gateState.criteria.Active = false

		s2.screenPod()
		if s2.hasAlert() {
			t.Errorf("expected no alert")
		}

	})

}
