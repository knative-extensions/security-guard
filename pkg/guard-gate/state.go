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
	"crypto/x509"
	"sync"
	"time"

	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
	"knative.dev/security-guard/pkg/dselect"
	utils "knative.dev/security-guard/pkg/guard-utils"
	"knative.dev/security-guard/pkg/iodup"
	pi "knative.dev/security-guard/pkg/pluginterfaces"
)

func logAlert(alert string) {
	pi.Log.Infof("SECURITY ALERT! %s", alert)
}

type gateState struct {
	analyzeBody     bool
	ctrl            *spec.Ctrl              // gate Ctrl
	criteria        *spec.SessionDataConfig // gate Criteria
	numSamples      uint32                  // number of samples used to create the Guardian
	stat            utils.Stat              // gate stats
	alert           string                  // gate alert
	decision        *spec.Decision          // gate alert decision
	monitorPod      bool                    // should gate profile the pod?
	pod             spec.PodProfile         // pod profile
	srv             *gateClient             // maintainer of the pile, include client to the guard-service & kubeApi
	certPool        *x509.CertPool          // rootCAs
	prevAlert       string                  // previous gate alert
	skippedSyncs    uint32                  // how many times we skipped sync?
	lastSync        int64                   // last time we synced
	iodups          *iodup.IoDups
	ticks           int64
	ticksMutex      sync.Mutex
	dselect         *dselect.DSelect
	syncServiceSecs int64
	podMonitorSecs  int64
}

func NewGateState(ctx context.Context, syncServiceSecs int64, podMonitorSecs int64) *gateState {
	gs := new(gateState)
	gs.ticks = time.Now().Unix()
	gs.syncServiceSecs = syncServiceSecs
	gs.podMonitorSecs = podMonitorSecs
	gs.dselect = dselect.NewDSelect(ctx, gs.setTicks)
	return gs
}

func (gs *gateState) getTicks() int64 {
	gs.ticksMutex.Lock()
	defer gs.ticksMutex.Unlock()
	return gs.ticks
}

func (gs *gateState) setTicks(ticks int64) {
	gs.ticksMutex.Lock()
	gs.ticks = ticks
	gs.ticksMutex.Unlock()

	// Periodically sync to guard-service
	if ticks%gs.syncServiceSecs == 0 {
		gs.syncIfNeeded(ticks)
	}

	// Periodically profile of the pod
	if ticks%gs.podMonitorSecs == 1 {
		gs.profileAndDecidePod(ticks)
	}
}

func (gs *gateState) Add(ctx context.Context, s *session) {
	gs.dselect.Add(ctx, s.tick, s.complete, 5)
}

func (gs *gateState) init(monitorPod bool, guardServiceUrl string, podname string, sid string, ns string, useCm bool, rootCA string) {
	var err error
	var skipVerify bool

	gs.iodups = iodup.NewIoDups(2, 128, 8192)
	gs.stat.Init()
	gs.monitorPod = monitorPod
	gs.srv = NewGateClient(guardServiceUrl, podname, sid, ns, useCm)

	gs.certPool, err = x509.SystemCertPool()
	if err != nil {
		gs.certPool = x509.NewCertPool()
	}

	if rootCA != "" {
		if ok := gs.certPool.AppendCertsFromPEM([]byte(rootCA)); ok {

			pi.Log.Infof("TLS: Success adding ROOT_CA")
		} else {
			pi.Log.Infof("TLS: Failed to AppendCertsFromPEM from ROOT_CA, ROOT_CA is: %s", rootCA)
			pi.Log.Warnf("Insecure Communication, Working without TLS ROOT_CA!!!")
			skipVerify = true
		}
	} else {
		pi.Log.Infof("ROOT_CA is empty!")
		pi.Log.Warnf("Insecure Communication, Working without TLS ROOT_CA!!!")
		skipVerify = true
	}

	gs.srv.initHttpClient(gs.certPool, skipVerify)
}

func (gs *gateState) start() {
	// Skip during simulations
	if len(gs.srv.ns) > 0 {
		// initializtion that cant be tested due to use of KubeAMgr
		gs.srv.initKubeMgr()
	}
}

// sync is called periodically to send pile and alerts and to load from the updated Guardian
func (gs *gateState) sync(shouldLoad bool, forceSync bool, ticks int64) {
	if !forceSync && (ticks-gs.lastSync) < MIN_TIME_BETWEEN_SYNCS {
		return
	}

	gs.skippedSyncs = 0
	// send pile and alerts and get Guardian - never returns nil!
	g := gs.srv.syncWithServiceAndKubeApi(ticks, shouldLoad)
	if !shouldLoad {
		return
	}

	// load guardian
	gs.numSamples = g.NumSamples

	// Set the correct Control
	if gs.ctrl = g.Control; gs.ctrl == nil {
		pi.Log.Infof("Loading Guardian  - without Control")
		gs.ctrl = new(spec.Ctrl)
	}

	// Set the correct criteria
	var criteria *spec.SessionDataConfig
	if gs.ctrl.Auto {
		criteria = g.Learned
	} else {
		criteria = g.Configured
	}
	if criteria == nil {
		pi.Log.Infof("Loading Guardian  - without criteria")
		criteria = new(spec.SessionDataConfig)
	}
	criteria.Prepare()
	gs.criteria = criteria
	pi.Log.Debugf("Loading Guardian  - Active %t Auto %t Block %t", gs.criteria.Active, gs.ctrl.Auto, gs.ctrl.Block)
}

func (gs *gateState) syncIfNeeded(ticks int64) {
	// if we have 10% new samples or more (otherwise wait till we get to 1000 samples)
	if gs.numSamples < gs.srv.pile.Count*10 || len(gs.srv.alerts) > 0 {
		gs.sync(true, false, ticks)
		return
	}

	// if we skipped 4 times, then this time we will sync
	// 5 min for the default 1 min syncInterval
	gs.skippedSyncs++
	if gs.skippedSyncs >= 5 {
		gs.sync(true, false, ticks)
	}
}

// addProfile is called every time we have a new profile ready to be added to a pile
func (gs *gateState) addProfile(profile *spec.SessionDataProfile, ticks int64) {
	if gs.srv.addToPile(profile) >= PILE_LIMIT {
		gs.sync(true, false, ticks)
	}
}

// addAlert is called every time we have a new alert
func (gs *gateState) addAlert(decision *spec.Decision, level string) {
	if gs.srv.addAlert(decision, level) >= ALERTS_LIMIT {
		gs.sync(true, false, gs.getTicks())
	}
}

// Methods to profile and decide based on pod data.
// Enables the POD profile to be copied to the sessio  profile for reporting to pile.

// profileAndDecidePod is called periodically to profile the pod and decide if to raise an alert
func (gs *gateState) profileAndDecidePod(ticks int64) {
	if !gs.monitorPod {
		return
	}
	// First profile
	gs.pod.Profile()

	// Now decide
	// Current behavior is latching the alert forever
	// This makes sense from security standpoint as the pod is now considered contaminated
	// If we are blocking, this means we will simply keep blocking all requests forever
	// Therefore we terminate the reverse proxy
	// Future - add more controls to decide what to do in this situation
	if gs.criteria != nil && gs.criteria.Active {
		spec.DecideChild(&gs.decision, gs.criteria.Pod.Decide(&gs.pod), "Pod")
		if gs.decision != nil {
			gs.logAlert()
			if gs.shouldBlock() {
				gs.srv.signalCompromised(ticks)
				// Terminate the reverse proxy since all requests will block from now on
				pi.Log.Infof("Terminating")
				gs.addStat("BlockOnPod")
			}
		}
	}
}

func (gs *gateState) logAlert() {
	if gs.decision == nil {
		return
	}
	gs.alert = gs.decision.String("Gate ->")
	if gs.prevAlert == gs.alert {
		return
	}
	gs.prevAlert = gs.alert
	logAlert(gs.alert)
	gs.addStat("GateLevelAlert")
	gs.addAlert(gs.decision, "Gate")
}

// if pod is monitored, copy its profile to the session profile
func (gs *gateState) copyPodProfile(pp *spec.PodProfile) {
	if !gs.monitorPod {
		return
	}
	gs.pod.DeepCopyInto(pp)
}

// The following decide...() methods allow sessions to provide parts of the profile as they are being formed
// the profiles are being decided and alerts is set accordingly.

// returns the alert text if needed
func (gs *gateState) decideReq(decision **spec.Decision, rp *spec.ReqProfile) {
	if gs.criteria != nil && gs.criteria.Active {
		spec.DecideChild(decision, gs.criteria.Req.Decide(rp), "HttpRequest")
		if *decision != nil {
			gs.addStat("ReqAlert")
		}
	}
}

// returns the alert text if needed
func (gs *gateState) decideResp(decision **spec.Decision, rp *spec.RespProfile) {
	if gs.criteria != nil && gs.criteria.Active {
		spec.DecideChild(decision, gs.criteria.Resp.Decide(rp), "HttpResponse")
		if *decision != nil {
			gs.addStat("RespAlert")
		}
	}
}

// returns the alert text if needed
func (gs *gateState) decideReqBody(decision **spec.Decision, bp *spec.BodyProfile) {
	if gs.criteria != nil && gs.criteria.Active {
		spec.DecideChild(decision, gs.criteria.ReqBody.Decide(bp), "HttpRequestBody")
		if *decision != nil {
			gs.addStat("ReqBodyAlert")
		}
	}
}

// returns the alert text if needed
func (gs *gateState) decideRespBody(decision **spec.Decision, bp *spec.BodyProfile) {
	if gs.criteria != nil && gs.criteria.Active {
		spec.DecideChild(decision, gs.criteria.RespBody.Decide(bp), "HttpResponseBody")
		if *decision != nil {
			gs.addStat("RespBodyAlert")
		}
	}
}

// returns the alert text if needed
func (gs *gateState) decideEnvelop(decision **spec.Decision, ep *spec.EnvelopProfile) {
	if gs.criteria != nil && gs.criteria.Active {
		spec.DecideChild(decision, gs.criteria.Envelop.Decide(ep), "Envelop")
		if *decision != nil {
			gs.addStat("EnvelopAlert")
		}
	}
}

// generic methods:

// advance statistics counter
func (gs *gateState) addStat(key string) {
	gs.stat.Add(key)
}

// are we blocking request on alerts?
func (gs *gateState) shouldBlock() bool {
	return (gs.ctrl != nil) && gs.ctrl.Block
}

// do we have a gate level alert?
func (gs *gateState) hasAlert() bool {
	return gs.decision != nil
}

// should we be learning?
func (gs *gateState) shouldLearn(sessionAlert bool) bool {
	// did we have an alert?
	alert := (gs.alert != "") || !sessionAlert
	return (gs.ctrl != nil) && gs.ctrl.Learn && (!alert || gs.ctrl.Force)
}
