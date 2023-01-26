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
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"time"

	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
	utils "knative.dev/security-guard/pkg/guard-utils"
	pi "knative.dev/security-guard/pkg/pluginterfaces"

	"knative.dev/security-guard/pkg/iodup"
)

const sessionKey = "GuardSession"

const (
	other_type      = 0
	json_type       = 1
	multipart_type  = 2
	urlencoded_type = 3
)

var maxBody int64 = int64(1048576)

type session struct {
	sessionTicker *utils.Ticker
	gotResponse   bool
	alert         string                  // session alert
	reqTime       time.Time               // time when session was started
	respTime      time.Time               // time when session response came
	cancelFunc    context.CancelFunc      // cancel the session
	profile       spec.SessionDataProfile // maintainer of the session profile
	gateState     *gateState              // maintainer of the criteria and ctrl, include pod profile, gate stats and gate level alert
}

func newSession(state *gateState, cancel context.CancelFunc) *session {
	s := new(session)
	s.reqTime = time.Now()
	s.respTime = s.reqTime // indicates that we do not know the response time
	s.gateState = state
	s.cancelFunc = cancel
	s.sessionTicker = utils.NewTicker(utils.MinimumInterval)
	if err := s.sessionTicker.Parse("", podMonitorIntervalDefault); err != nil {
		pi.Log.Debugf("Error on Ticker Parse: %v", err)
	}
	state.addStat("Total")
	return s
}

func getSessionFromContext(ctx context.Context) *session {
	defer func() {
		// This should never happen!
		if r := recover(); r != nil {
			pi.Log.Warnf("getSessionFromContext Recovered %s", r)
		}
	}()

	s, sExists := ctx.Value(ctxKey(sessionKey)).(*session)
	if !sExists {
		// This should never happen!
		return nil
	}
	return s
}

func (s *session) addSessionToContext(ctxIn context.Context) context.Context {
	return context.WithValue(ctxIn, ctxKey(sessionKey), s)
}

func (s *session) hasAlert() bool {
	return s.alert != ""
}

func (s *session) cancel() {
	s.cancelFunc()
}

func (s *session) sessionEventLoop(ctx context.Context) {
	s.sessionTicker.Start()

	defer func() {
		s.sessionTicker.Stop()

		// Should we learn?
		if s.gateState.shouldLearn(s.hasAlert()) && s.gotResponse {
			s.gateState.addProfile(&s.profile)
		}

		// Should we alert?
		if s.gateState.hasAlert() {
			s.gateState.logAlert()
			s.gateState.addStat("BlockOnPod")
			return
		}
		if s.hasAlert() {
			logAlert(s.alert)
			s.gateState.addStat("SessionLevelAlert")
			return
		}
		// no alert
		if !s.gotResponse {
			pi.Log.Debugf("No Alert but completed before receiving a response!")
			s.gateState.addStat("NoResponse")
			return
		}
		if s.gateState.criteria == nil {
			pi.Log.Debugf("No Alert since no criteria")
			s.gateState.addStat("NoAlertNoCriteria")
			return
		}
		if !s.gateState.criteria.Active {
			pi.Log.Debugf("No Alert since criteria is not active")
			s.gateState.addStat("NoAlertCriteriaNotActive")
			return
		}
		pi.Log.Debugf("No Alert!")
		s.gateState.addStat("NoAlert")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.sessionTicker.Ch():
			s.screenEnvelop()
			s.screenPod()
			if s.gateState.shouldBlock() && (s.hasAlert() || s.gateState.hasAlert()) {
				pi.Log.Debugf("Request processing canceled during sessionTicker")
				s.cancel()
				return
			}
			pi.Log.Debugf("Session Tick")
		}
	}
}

func (s *session) screenResponseBody(resp *http.Response) {
	if !s.gateState.analyzeBody || resp.Body == nil {
		return
	}

	if resp.ContentLength > maxBody {
		// we perform response body analysis only for body smaller than 1MB
		s.profile.RespBody.ProfileFaults("TooLargeBody")
		return
	}

	if resp.ContentLength <= 0 {
		// we perform response body analysis only when we know in advance its size
		s.profile.RespBody.ProfileFaults("UnknownSizeBody")
		return
	}

	body_type := other_type

	// TBD - validate content-type params returned by ParseMediaType!
	ctype, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		ctype = "application/octet-stream"
	} else {
		switch ctype {
		case "application/json":
			body_type = json_type
		default:
			body_type = other_type
		}
	}

	dup := iodup.New(resp.Body, 2, 128, 8192)
	resp.Body = dup.Output[0]

	switch body_type {
	case json_type:
		var structuredData interface{}
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&structuredData)
		if err != nil {
			pi.Log.Debugf("Failed while decoding body json! %v", err)
			s.profile.RespBody.ProfileFaults("FailedJsonDecode")
		} else {
			s.profile.RespBody.ProfileStructured(structuredData)
		}
	default:
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			pi.Log.Debugf("Failed while analyzing unstructured data %v", err)
			s.profile.RespBody.ProfileFaults("FailedUnstructured")
		} else {
			s.profile.RespBody.ProfileUnstructured(string(bytes))
		}
	}
	resp.Body = dup.Output[1]
	s.alert += s.gateState.decideRespBody(&s.profile.RespBody)
}

func (s *session) screenRequestBody(req *http.Request) {
	if !s.gateState.analyzeBody || req.Body == nil {
		return
	}

	if req.ContentLength > maxBody {
		// we perform request body analysis only for body smaller than 1MB
		s.profile.ReqBody.ProfileFaults("TooLargeBody")
		return
	}

	if req.ContentLength <= 0 {
		// we perform request body analysis only when we know in advance its size
		s.profile.ReqBody.ProfileFaults("UnknownSizeBody")
		return
	}

	body_type := other_type

	// TBD - validate content-type params returned by ParseMediaType!
	ctype, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil {
		ctype = "application/octet-stream"
	} else {
		switch ctype {
		case "application/json":
			body_type = json_type
		case "multipart/form-data":
			body_type = multipart_type
		case "application/x-www-form-urlencoded":
			body_type = urlencoded_type
		default:
			body_type = other_type
		}
	}

	dup := iodup.New(req.Body, 2, 128, 8192)
	req.Body = dup.Output[0]

	switch body_type {
	case json_type:
		var structuredData interface{}
		dec := json.NewDecoder(req.Body)
		err = dec.Decode(&structuredData)
		if err != nil {
			pi.Log.Debugf("Failed while decoding body json! %v", err)
			s.profile.ReqBody.ProfileFaults("FailedJsonDecode")
		} else {
			s.profile.ReqBody.ProfileStructured(structuredData)
		}
	case multipart_type:
		if err := req.ParseMultipartForm(maxBody); err != nil {
			pi.Log.Debugf("Failed while ParseMultipartForm! %v", err)
			s.profile.ReqBody.ProfileFaults("FailedMultipart")
		} else {
			s.profile.ReqBody.ProfileStructured(req.PostForm)
		}

	case urlencoded_type:
		if err := req.ParseForm(); err != nil {
			pi.Log.Debugf("Failed while ParseForm! %v", err)
			s.profile.ReqBody.ProfileFaults("FailedUrlencoded")
		} else {
			s.profile.ReqBody.ProfileStructured(req.PostForm)
		}
	default:
		bytes, err := io.ReadAll(req.Body)
		if err != nil {
			pi.Log.Debugf("Failed while analyzing unstructured data %v", err)
			s.profile.ReqBody.ProfileFaults("FailedUnstructured")
		} else {
			s.profile.ReqBody.ProfileUnstructured(string(bytes))
		}
	}
	req.Body = dup.Output[1]

	s.alert += s.gateState.decideReqBody(&s.profile.ReqBody)
}

func (s *session) screenEnvelop() {
	now := time.Now()
	respTime := s.respTime
	if !s.respTime.After(s.reqTime) {
		// we do not know the response time, lets assume it is now
		respTime = now
	}
	s.profile.Envelop.Profile(s.reqTime, respTime, now)

	s.alert += s.gateState.decideEnvelop(&s.profile.Envelop)
}

func (s *session) screenPod() {
	s.gateState.copyPodProfile(&s.profile.Pod)
}

func (s *session) screenRequest(req *http.Request) {
	// Request client and server identities
	cip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		s.alert += fmt.Sprintf("illegal req.RemoteAddr %s", err.Error())
		s.gateState.addStat("ReqCipFault")
	}

	ip := net.ParseIP(cip)
	s.profile.Req.Profile(req, ip)
	//s.profile.ReqBody.Profile(reqData)

	s.alert += s.gateState.decideReq(&s.profile.Req)
}

func (s *session) screenResponse(resp *http.Response) {
	s.respTime = time.Now()
	s.profile.Resp.Profile(resp)

	s.alert += s.gateState.decideResp(&s.profile.Resp)
}
