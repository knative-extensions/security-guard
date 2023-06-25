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
	"errors"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	_ "net/http/pprof"

	pi "knative.dev/security-guard/pkg/pluginterfaces"
)

const plugVersion string = "0.5"
const plugName string = "guard"

const (
	sessionKey = "GuardSession"
	maxBody    = int64(1048576)
)

var errSecurity error = errors.New("security blocked by guard")

type ctxKey string

type plug struct {
	name    string
	version string

	// guard gate plug specifics
	gateState *gateState // maintainer of the criteria and ctrl, include pod profile, gate stats and gate level alert
}

func (p *plug) Shutdown() {
	pi.Log.Infof("%s Shutdown - performing final Sync!", p.name)
	ticks := p.gateState.getTicks()
	if p.gateState.srv.pile.Count > 0 || len(p.gateState.srv.alerts) > 0 {
		p.gateState.sync(false, true, ticks)
	}
	pi.Log.Infof("%s - Done with the following statistics: %s", p.name, p.gateState.stat.Log())
	pi.Log.Sync()
}

func (p *plug) PlugName() string {
	return p.name
}

func (p *plug) PlugVersion() string {
	return p.version
}

func (p *plug) ApproveRequest(req *http.Request) (*http.Request, error) {
	ticks := p.gateState.getTicks()
	ctx, cancelFunction := context.WithCancel(req.Context())

	s := newSession(p.gateState, cancelFunction, ticks) // maintainer of the profile

	// Req
	s.screenEnvelop(ticks)
	s.screenRequest(req)
	s.screenRequestBody(req)

	if p.gateState.shouldBlock() {
		// Should we alert?
		if s.gateState.hasAlert() {
			p.gateState.addStat("BlockOnPod")
			pi.Log.Debugf("Request blocked")
			cancelFunction()
			return nil, errSecurity
		}
		if s.hasAlert() {
			s.logAlert()
			p.gateState.addStat("BlockOnRequest")
			pi.Log.Debugf("Request blocked")
			cancelFunction()
			return nil, errSecurity
		}
	}

	// Request not blocked
	ctx = s.addSessionToContext(ctx)
	ctx = context.WithValue(ctx, ctxKey("GuardSession"), s)

	req = req.WithContext(ctx)

	p.gateState.Add(ctx, s)

	return req, nil
}

func (p *plug) ApproveResponse(req *http.Request, resp *http.Response) (*http.Response, error) {

	s := getSessionFromContext(req.Context())
	if s == nil { // This should never happen!
		pi.Log.Infof("%s ........... Blocked During Response! Missing context!", p.name)
		return nil, errors.New("missing context")
	}
	ticks := p.gateState.getTicks()
	s.gotResponse = true
	s.screenResponse(resp, ticks)
	s.screenResponseBody(resp)
	s.screenEnvelop(ticks)

	if p.gateState.shouldBlock() && (s.hasAlert() || p.gateState.hasAlert()) {
		p.gateState.addStat("BlockOnResponse")
		pi.Log.Debugf("Response blocked")
		s.cancel()
		return nil, errSecurity
	}

	return resp, nil
}

func (p *plug) preInit(ctx context.Context, c map[string]string, sid string, ns string, logger pi.Logger) {
	var ok bool
	var v string
	var syncInterval, monitorInterval string
	var rootCA string

	// Defaults used without config when used as a qpoption
	useCrd := true
	monitorPod := true
	analyzeBody := true

	guardServiceUrl := "https://guard-service.knative-serving"

	if c != nil {
		rootCA = c["rootca"]

		if v, ok = c["guard-url"]; ok && v != "" {
			// use default
			guardServiceUrl = v
		}

		if v, ok = c["use-crd"]; ok && strings.EqualFold(v, "false") {
			useCrd = false
		}

		if v, ok = c["monitor-pod"]; ok && !strings.EqualFold(v, "true") {
			monitorPod = false
		}

		if v, ok = c["analyze-body"]; ok && strings.EqualFold(v, "true") {
			analyzeBody = true
		}

		syncInterval = c["guardian-sync-interval"]
		monitorInterval = c["pod-monitor-interval"]
	} else {
		pi.Log.Infof("guard-gate missing configuration")
	}

	podname := "unknown"
	if v, ok = c["podname"]; ok {
		podname = v
	} else {
		data, err := os.ReadFile("/etc/hostname")
		if err == nil {
			str := regexp.MustCompile(`[^a-zA-Z0-9\-]+`).ReplaceAllString(string(data), "")
			podname = str
		}
	}

	pi.Log.Infof("guard-gate configuration: podname=%s, sid=%s, ns=%s, useCrd=%t, guardUrl=%s, p.monitorPod=%t, guardian-sync-interval %v,  pod-monitor-interval %v",
		podname, sid, ns, useCrd, guardServiceUrl, monitorPod, syncInterval, monitorInterval)

	// serviceName should never be "ns.{namespace}" as this is a reserved name
	if strings.HasPrefix(sid, "ns.") {
		// mandatory
		panic("Ileal serviceName - ns.{Namespace} is reserved")
	}

	var syncServiceSecs, podMonitorSecs int64
	d, err := time.ParseDuration(syncInterval)
	if err == nil {
		syncServiceSecs = int64(d.Seconds())
	} else {
		syncServiceSecs = 60
		if syncInterval != "" {
			pi.Log.Errorf("interval illegal value %s - using default value instead (err: %v)", syncInterval, err)
		}
	}

	d, err = time.ParseDuration(monitorInterval)
	if err == nil {
		podMonitorSecs = int64(d.Seconds())
	} else {
		podMonitorSecs = 5
		if monitorInterval == "" {
			pi.Log.Errorf("interval illegal value %s - using default value instead (err: %v)", monitorInterval, err)
		}
	}

	p.gateState = NewGateState(ctx, syncServiceSecs, podMonitorSecs, monitorPod, guardServiceUrl, podname, sid, ns, !useCrd, rootCA)
	p.gateState.analyzeBody = analyzeBody

	pi.Log.Infof("guard-gate: Secured Communication %t (CERT Verification %t AUTH Token %t)", p.gateState.srv.certVerifyActive && p.gateState.srv.tokenActive, p.gateState.srv.certVerifyActive, p.gateState.srv.tokenActive)
}

func (p *plug) Init(ctx context.Context, c map[string]string, sid string, ns string, logger pi.Logger) context.Context {
	p.preInit(ctx, c, sid, ns, logger)

	// cant be tested as depend on KubeMgr
	p.gateState.start()

	ticks := time.Now().Unix()
	p.gateState.sync(true, true, ticks)
	p.gateState.profileAndDecidePod(ticks)

	return ctx
}

func init() {
	gate := &plug{version: plugVersion, name: plugName}
	pi.RegisterPlug(gate)
}
