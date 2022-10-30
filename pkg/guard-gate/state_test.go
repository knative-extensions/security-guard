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
	"crypto/x509"
	"net"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
)

var gateCanceled int

func fakeGateCancel() {
	gateCanceled++
}

func fakeGateState() *gateState {
	gs := new(gateState)
	gs.init(fakeGateCancel, false, "myurl", "mysid", "myns", true)
	srv, _ := fakeClient(http.StatusOK, "Problem in request")
	gs.srv = srv
	return gs
}

func Test_gateState_loadConfig(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		gs := fakeGateState()

		gs.loadConfig()
		if gs.criteria == nil || gs.ctrl == nil {
			t.Error("nil after load")
		}
		gs.criteria.Active = false
		gs.profileAndDecidePod()
		if gateCanceled != 0 {
			t.Error("expected no cancel")
		}

		// this test only checks if panic
		// we cant be sure what the response will be as it depends on the /proc
		gs.monitorPod = true
		gs.criteria.Active = true
		gateCanceled = 0
		gs.profileAndDecidePod()

		var pp spec.PodProfile
		gs.monitorPod = false
		gs.copyPodProfile(&pp)
		gs.monitorPod = true
		gs.copyPodProfile(&pp)
		if !reflect.DeepEqual(&gs.pod, &pp) {
			t.Errorf("expected %v to be equal to %v", pp, gs.pod)
		}

		gs.stat.Init()
		gs.alert = ""
		gs.addStat("XX")
		gs.addStat("XX")
		if ret := gs.stat.Log(); ret != "map[XX:2]" {
			t.Errorf("expected stat.log to be %s received %s", "map[XX:2]", ret)
		}
		if gs.shouldBlock() != false {
			t.Error("expected false in shouldBlock")
		}
		if gs.hasAlert() != false {
			t.Error("expected false in hasAlert")
		}
		if gs.shouldLearn(true) != false {
			t.Error("expected false in shouldLearn")
		}

		// envelop
		ep := new(spec.EnvelopProfile)
		now := time.Now()
		ep.Profile(now, now, now)
		if ret := gs.decideEnvelop(ep); ret != "" {
			t.Error("expected no alert")
		}
		ep.Profile(time.Unix(1, 1), time.Unix(3, 3), time.Unix(5, 5))
		gs.decideEnvelop(ep)
		if ret := gs.decideEnvelop(ep); ret == "" {
			t.Error("expected alert")
		}

		// req
		req := new(spec.ReqProfile)
		r, _ := http.NewRequest("Get", "", nil)
		cip := net.ParseIP("1.2.3.4")
		req.Profile(r, cip)
		gs.criteria.Active = false
		if ret := gs.decideReq(req); ret != "" {
			t.Error("expected no alert")
		}
		gs.criteria.Active = true
		if ret := gs.decideReq(req); ret == "" {
			t.Error("expected alert")
		}

		// resp
		resp := new(spec.RespProfile)
		rs := &http.Response{ContentLength: 20}

		resp.Profile(rs)
		gs.criteria.Active = false
		if ret := gs.decideResp(resp); ret != "" {
			t.Error("expected no alert")
		}
		gs.criteria.Active = true
		if ret := gs.decideResp(resp); ret == "" {
			t.Error("expected alert")
		}

		// reqBody
		body := new(spec.BodyProfile)
		body.ProfileUnstructured("x")
		gs.criteria.Active = false
		if ret := gs.decideReqBody(body); ret != "" {
			t.Error("expected no alert")
		}
		if ret := gs.decideRespBody(body); ret != "" {
			t.Error("expected no alert")
		}
		gs.criteria.Active = true
		if ret := gs.decideReqBody(body); ret == "" {
			t.Error("expected alert")
		}
		if ret := gs.decideRespBody(body); ret == "" {
			t.Error("expected alert")
		}

		gs.flushPile()
		if gs.srv.pile.Count != 0 {
			t.Error("expected pile too have 1")
		}
		profile := new(spec.SessionDataProfile)
		gs.addProfile(profile)
		if gs.srv.pile.Count != 1 {
			t.Error("expected pile too have 1")
		}
		gs.flushPile()
		if gs.srv.pile.Count != 0 {
			t.Error("expected pile too have 1")
		}
	})

}

func Test_gateState_init(t *testing.T) {
	tests := []struct {
		name  string
		cert  string
		newCA bool
	}{
		{
			name: "empty",
			cert: "",
		},
		{
			name: "bad",
			cert: "xx",
		},
		{
			name:  "good",
			cert:  testCert,
			newCA: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ROOT_CA", tt.cert)
			gs := new(gateState)
			certPool, _ := x509.SystemCertPool()
			gs.init(fakeGateCancel, false, "myurl", "mysid", "myns", true)
			if !certPool.Equal(gs.certPool) && !tt.newCA {
				t.Errorf("expected no new cert to be added")
			}
			if certPool.Equal(gs.certPool) && tt.newCA {
				t.Errorf("expected new cert to be added")
			}
			os.Unsetenv("ROOT_CA")
		})
	}
}
