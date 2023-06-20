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

	utils "knative.dev/security-guard/pkg/guard-utils"
)

const plugVersion string = "0.5"
const plugName string = "guard"

const (
	sessionKey = "GuardSession"
	maxBody    = int64(1048576)
)

const (
	syncIntervalDefault       = 60 * time.Second
	podMonitorIntervalDefault = 5 * time.Second
)

var errSecurity error = errors.New("security blocked by guard")

type ctxKey string

type plug struct {
	name    string
	version string

	// guard gate plug specifics
	gateState        *gateState    // maintainer of the criteria and ctrl, include pod profile, gate stats and gate level alert
	podMonitorTicker *utils.Ticker // tick to gateState.profileAndDecidePod()
	syncTicker       *utils.Ticker // tick to gateState.sync()
}

func (p *plug) Shutdown() {
	pi.Log.Infof("%s Shutdown - performing final Sync!", p.name)
	p.syncTicker.Stop()
	p.podMonitorTicker.Stop()
	if p.gateState.srv.pile.Count > 0 || len(p.gateState.srv.alerts) > 0 {
		p.gateState.sync(false, true)
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
	ctx, cancelFunction := context.WithCancel(req.Context())

	s := newSession(p.gateState, cancelFunction) // maintainer of the profile

	// Req
	s.screenEnvelop()
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

	//goroutine to accompany the request
	go s.sessionEventLoop(ctx)

	return req, nil
}

func (p *plug) ApproveResponse(req *http.Request, resp *http.Response) (*http.Response, error) {

	s := getSessionFromContext(req.Context())
	if s == nil { // This should never happen!
		pi.Log.Infof("%s ........... Blocked During Response! Missing context!", p.name)
		return nil, errors.New("missing context")
	}

	s.gotResponse = true
	s.screenResponse(resp)
	s.screenResponseBody(resp)
	s.screenEnvelop()

	if p.gateState.shouldBlock() && (s.hasAlert() || p.gateState.hasAlert()) {
		p.gateState.addStat("BlockOnResponse")
		pi.Log.Debugf("Response blocked")
		s.cancel()
		return nil, errSecurity
	}

	return resp, nil
}

func (p *plug) guardMainEventLoop(ctx context.Context) {
	p.syncTicker.Start()
	p.podMonitorTicker.Start()
	for {
		select {
		// Always finish guard here!
		case <-ctx.Done():
			pi.Log.Debugf("Terminating the guardMainEventLoop")
			p.syncTicker.Stop()
			p.podMonitorTicker.Stop()
			return
		// Periodically send pile and alerts and get an updated Guardian
		case <-p.syncTicker.Ch():
			p.gateState.syncIfNeeded()

		// Periodically profile of the pod
		case <-p.podMonitorTicker.Ch():
			p.gateState.profileAndDecidePod()
		}
	}
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

	p.syncTicker = utils.NewTicker(utils.MinimumInterval)
	p.podMonitorTicker = utils.NewTicker(utils.MinimumInterval)

	p.syncTicker.Parse(syncInterval, syncIntervalDefault)
	p.podMonitorTicker.Parse(monitorInterval, podMonitorIntervalDefault)

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

	p.gateState = NewGateState(ctx)
	p.gateState.analyzeBody = analyzeBody

	// p.gateState.init uses a `useCm` flag
	p.gateState.init(monitorPod, guardServiceUrl, podname, sid, ns, !useCrd, rootCA)

	pi.Log.Infof("guard-gate: Secured Communication %t (CERT Verification %t AUTH Token %t)", p.gateState.srv.certVerifyActive && p.gateState.srv.tokenActive, p.gateState.srv.certVerifyActive, p.gateState.srv.tokenActive)
}

func (p *plug) Init(ctx context.Context, c map[string]string, sid string, ns string, logger pi.Logger) context.Context {
	p.preInit(ctx, c, sid, ns, logger)

	// cant be tested as depend on KubeMgr
	p.gateState.start()

	p.gateState.sync(true, true)
	p.gateState.profileAndDecidePod()

	//goroutine for Guard instance
	go p.guardMainEventLoop(ctx)

	return ctx
}

func init() {
	gate := &plug{version: plugVersion, name: plugName}
	pi.RegisterPlug(gate)
}
