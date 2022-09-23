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
	"fmt"

	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
	utils "knative.dev/security-guard/pkg/guard-utils"
	pi "knative.dev/security-guard/pkg/pluginterfaces"
)

func logAlert(alert string) {
	pi.Log.Warnf("SECURITY ALERT! %s", alert)
}

type gateState struct {
	cancelFunc context.CancelFunc      // cancel the entire reverse proxy
	ctrl       *spec.Ctrl              // gate Ctrl
	criteria   *spec.SessionDataConfig // gate Criteria
	stat       utils.Stat              // gate stats
	alert      string                  // gate alert
	monitorPod bool                    // should gate profile the pod?
	pod        spec.PodProfile         // pod profile
	srv        *gateClient             // maintainer of the pile, include client to the guard-service & kubeApi
}

func (gs *gateState) init(cancelFunc context.CancelFunc, monitorPod bool, guardServiceUrl string, sid string, ns string, useCm bool) {
	gs.stat.Init()
	gs.monitorPod = monitorPod
	gs.cancelFunc = cancelFunc
	gs.srv = NewGateClient(guardServiceUrl, sid, ns, useCm)
}

func (gs gateState) start() {
	// initializtion that cant be tested due to use of KubeAMgr
	gs.srv.start()
}

// loadConfig is called periodically to load updated configuration from a Guardian
func (gs *gateState) loadConfig() {
	// loadGuardian never returns nil!
	g := gs.srv.loadGuardian()

	if gs.ctrl = g.Control; gs.ctrl == nil {
		gs.ctrl = new(spec.Ctrl)
	}

	if gs.ctrl.Auto {
		gs.criteria = g.Learned
	} else {
		gs.criteria = g.Configured
	}
	if gs.criteria == nil {
		gs.criteria = new(spec.SessionDataConfig)
	}
}

// flushPile is called periodically to send the pile to the guard-service
func (gs *gateState) flushPile() {
	gs.srv.reportPile()
}

// addProfile is called every time we have a new profile ready to be added to a pile
func (gs *gateState) addProfile(profile *spec.SessionDataProfile) {
	gs.srv.addToPile(profile)
}

// Methods to profile and decide base don pod data.
// Enables the POD profile to be copied to the sessio  profile for reporting to pile.

// profileAndDecidePod is called periodically to profile the pod and decide if to raise an alert
func (gs *gateState) profileAndDecidePod() {
	if !gs.monitorPod {
		return
	}
	//First profile
	gs.pod.Profile()

	// Now decide
	// Current behaviour is latching the alert forever
	// This makes sense from security standpoint as the pod is now considered contaminated
	// If we are blocking, this means we will simply keep blocking all requests forever
	// Therefore we terminate the reverse proxy
	// Future - add more controls to decide what to do in this situation
	if gs.criteria.Active {
		decision := gs.criteria.Pod.Decide(&gs.pod)
		if decision != "" {
			gs.addStat("PodAlert")
			gs.alert = fmt.Sprintf("Pod: %s", decision)
			logAlert(gs.alert)
			// terminate the reverse proxy
			gs.cancelFunc()
		}
	}
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
func (gs *gateState) decideReq(rp *spec.ReqProfile) string {
	if gs.criteria.Active {
		if decision := gs.criteria.Req.Decide(rp); decision != "" {
			gs.addStat("ReqAlert")
			return fmt.Sprintf("HttpRequest: %s", decision)
		}
	}
	return ""
}

// returns the alert text if needed
func (gs *gateState) decideResp(rp *spec.RespProfile) string {
	if gs.criteria.Active {
		if decision := gs.criteria.Resp.Decide(rp); decision != "" {
			gs.addStat("RespAlert")
			return fmt.Sprintf("HttpResponse: %s", decision)
		}
	}
	return ""
}

// returns the alert text if needed
func (gs *gateState) decideReqBody(bp *spec.BodyProfile) string {
	if gs.criteria.Active {
		if decision := gs.criteria.ReqBody.Decide(bp); decision != "" {
			gs.addStat("ReqBodyAlert")
			return fmt.Sprintf("HttpRequestBody: %s", decision)
		}
	}
	return ""
}

// returns the alert text if needed
func (gs *gateState) decideRespBody(bp *spec.BodyProfile) string {
	if gs.criteria.Active {
		if decision := gs.criteria.RespBody.Decide(bp); decision != "" {
			gs.addStat("RespBodyAlert")
			return fmt.Sprintf("HttpResponseBody: %s", decision)
		}
	}
	return ""
}

// returns the alert text if needed
func (gs *gateState) decideEnvelop(ep *spec.EnvelopProfile) string {
	if gs.criteria.Active {
		if decision := gs.criteria.Envelop.Decide(ep); decision != "" {
			gs.addStat("EnvelopAlert")
			return fmt.Sprintf("Envelop: %s", decision)
		}
	}
	return ""
}

// generic methods:

// advance statistics counter
func (gs *gateState) addStat(key string) {
	gs.stat.Add(key)
}

// are we blocking request on alerts?
func (gs *gateState) shouldBlock() bool {
	return gs.ctrl.Block
}

// do we have a gate level alert?
func (gs *gateState) hasAlert() bool {
	return gs.alert != ""
}

// should we be learning?
func (gs *gateState) shouldLearn(sessionAlert bool) bool {
	// dio we have an alert?
	alert := (gs.alert != "") || !sessionAlert
	return gs.ctrl.Learn && (!alert || gs.ctrl.Force)
}
