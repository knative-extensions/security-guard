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

	pi "knative.dev/security-guard/pkg/pluginterfaces"

	utils "knative.dev/security-guard/pkg/guard-utils"
)

const plugVersion string = "0.3"
const plugName string = "guard"

const (
	syncIntervalDefault       = 10 * time.Second
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
	pi.Log.Debugf("%s: Shutdown", p.name)
	p.gateState.sync()
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
	defer func() {
		p.syncTicker.Stop()
		p.podMonitorTicker.Stop()
		p.gateState.sync()
		pi.Log.Infof("%s: Done with the following statistics: %s", plugName, p.gateState.stat.Log())
	}()

	for {
		select {
		// Always finish guard here!
		case <-ctx.Done():
			return

		// Periodically send pile and alerts and get an updated Guardian
		case <-p.syncTicker.Ch():
			p.gateState.sync()

		// Periodically profile of the pod
		case <-p.podMonitorTicker.Ch():
			p.gateState.profileAndDecidePod()
		}
	}
}
func (p *plug) preInit(c map[string]string, sid string, ns string, logger pi.Logger) {
	var ok, securedCommunication bool
	var v string
	var loadInterval, pileInterval, monitorInterval string

	// Defaults used without config when used as a qpoption
	useCm := false
	monitorPod := true
	analyzeBody := false

	guardServiceUrl := "https://guard-service.knative-serving"
	if rootCA := os.Getenv("ROOT_CA"); rootCA != "" {
		securedCommunication = true
	}

	if c != nil {
		if v, ok = c["guard-url"]; ok && v != "" {
			// use default
			guardServiceUrl = v
		}

		if v, ok = c["use-cm"]; ok && strings.EqualFold(v, "true") {
			useCm = true
		}

		if v, ok = c["monitor-pod"]; ok && !strings.EqualFold(v, "true") {
			monitorPod = false
		}

		if v, ok = c["analyze-body"]; ok && strings.EqualFold(v, "true") {
			analyzeBody = true
		}

		loadInterval = c["guardian-load-interval"]
		pileInterval = c["report-pile-interval"]
		monitorInterval = c["pod-monitor-interval"]
	}

	p.syncTicker = utils.NewTicker(utils.MinimumInterval)
	p.podMonitorTicker = utils.NewTicker(utils.MinimumInterval)

	p.syncTicker.Parse(loadInterval, syncIntervalDefault)
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

	pi.Log.Debugf("guard-gate configuration: podname=%s, sid=%s, ns=%s, useCm=%t, guardUrl=%s, p.monitorPod=%t, guardian-load-interval %v, report-pile-interval %v, pod-monitor-interval %v",
		podname, sid, ns, useCm, guardServiceUrl, monitorPod, loadInterval, pileInterval, monitorInterval)

	// serviceName should never be "ns.{namespace}" as this is a reserved name
	if strings.HasPrefix(sid, "ns.") {
		// mandatory
		panic("Ileal serviceName - ns.{Namespace} is reserved")
	}

	p.gateState = new(gateState)
	p.gateState.analyzeBody = analyzeBody
	p.gateState.init(monitorPod, guardServiceUrl, podname, sid, ns, useCm)
	pi.Log.Infof("guard-gate: Secured Communication %t, Token %t", securedCommunication, p.gateState.srv.tokenActive)
}

func (p *plug) Init(ctx context.Context, c map[string]string, sid string, ns string, logger pi.Logger) context.Context {
	p.preInit(c, sid, ns, logger)

	// cant be tested as depend on KubeMgr
	p.gateState.start()

	p.gateState.sync()
	p.gateState.profileAndDecidePod()

	//goroutine for Guard instance
	go p.guardMainEventLoop(ctx)

	return ctx
}

func init() {
	gate := &plug{version: plugVersion, name: plugName}
	pi.RegisterPlug(gate)
}
