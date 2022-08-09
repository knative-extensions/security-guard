/*
Copyright 2018 The Knative Authors

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

package qpoption

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"go.uber.org/zap"
	"knative.dev/serving/pkg/queue"
	"knative.dev/serving/pkg/queue/sharedmain"

	pi "knative.dev/security-guard/pkg/pluginterfaces"
)

// This is a Knative Queue Proxy Option (QPOption) package to manage the life cycle and configrue
// a single security plug.
// It can be extended in the future to managing multiple securiity plugs by using the rtplugs package

var annotationsFilePath = queue.PodInfoVolumeMountPath + "/" + queue.PodInfoAnnotationsFilename
var qpextentionPreifx = "qpextention.knative.dev/"

type GateQPOption struct {
	config           map[string]string
	activated        bool
	defaults         *sharedmain.Defaults
	securityPlug     pi.RoundTripPlug
	nextRoundTripper http.RoundTripper // the next roundtripper
}

func NewGateQPOption() *GateQPOption {
	return new(GateQPOption)
}

func (p *GateQPOption) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			pi.Log.Warnf("Recovered from panic during RoundTrip! Recover: %v\n", recovered)
			pi.Log.Debugf("Stacktrace from panic: \n %s\n" + string(debug.Stack()))
			err = errors.New("paniced during RoundTrip")
			resp = nil
		}
	}()

	if req, err = p.securityPlug.ApproveRequest(req); err == nil {
		pi.Log.Infof("p %v", p)
		pi.Log.Infof("p.nextRoundTripper %v", p.nextRoundTripper)
		if resp, err = p.nextRoundTripper.RoundTrip(req); err == nil {
			resp, err = p.securityPlug.ApproveResponse(req, resp)
		}
	}
	if err != nil {
		pi.Log.Debugf("%s: returning error %v", p.securityPlug.PlugName(), err)
		resp = nil
	}
	return
}

func (p *GateQPOption) ProcessAnnotations(annotationsPath string, qpextentionPreifx string) bool {
	file, err := os.Open(annotationsPath)
	if err != nil {
		p.defaults.Logger.Debugf("Cant find %s. Apperently podInfo is not mounted. os.Open Error %s", annotationsPath, err.Error())
		return false
	}
	defer file.Close()
	p.config = make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txt := scanner.Text()
		txt = strings.ToLower(txt)
		parts := strings.Split(txt, "=")

		k := parts[0]
		v := parts[1]
		if strings.HasPrefix(k, qpextentionPreifx) && len(k) > len(qpextentionPreifx) {
			v = strings.TrimSuffix(strings.TrimPrefix(v, "\""), "\"")
			k = k[len(qpextentionPreifx):]
			keyparts := strings.Split(k, "-")
			if len(keyparts) < 2 {
				continue
			}
			extension := keyparts[0]
			action := keyparts[1]
			if strings.EqualFold(extension, p.securityPlug.PlugName()) {
				switch action {
				case "activate":
					if strings.EqualFold(v, "enable") {
						p.activated = true
					}
				case "config":
					if len(keyparts) == 3 {
						extensionKey := keyparts[2]
						p.config[extensionKey] = v
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		p.defaults.Logger.Infof("Scanner Error %s", err.Error())
		return false
	}
	return true
}

func (p *GateQPOption) Setup(defaults *sharedmain.Defaults) {
	// Never panic the caller app from here
	defer func() {
		if r := recover(); r != nil {
			pi.Log.Warnf("Recovered from panic during Setup()! Recover: %v", r)
		}
	}()

	if (pi.RoundTripPlugs == nil) || len(pi.RoundTripPlugs) != 1 {
		pi.Log.Infof("Option without a Plug, Or with more than one Plug - Skip Option")
		return
	}
	p.securityPlug = pi.RoundTripPlugs[0]
	p.defaults = defaults
	namespace := defaults.Env.ServingNamespace
	servicName := defaults.Env.ServingService
	if servicName == "" {
		servicName = defaults.Env.ServingConfiguration
	}

	if defaults.Logger == nil {
		logger, _ := zap.NewProduction()
		defaults.Logger = logger.Sugar()
		defaults.Logger.Warnf("Received a nil logger\n")
	}
	pi.Log = defaults.Logger

	// build p.config

	if !p.ProcessAnnotations(annotationsFilePath, qpextentionPreifx) || !p.activated {
		pi.Log.Debugf("%s is not activated", p.securityPlug.PlugName())
		return
	}

	pi.Log.Debugf("Activating %s version %s with config %v in pod %s namespace %s", p.securityPlug.PlugName(), p.securityPlug.PlugVersion(), p.config, servicName, namespace)

	// setup context
	if defaults.Ctx == nil {
		pi.Log.Warnf("Received a nil context\n")
		defaults.Ctx = context.Background()
	}
	defaults.Ctx = p.securityPlug.Init(defaults.Ctx, p.config, servicName, namespace, defaults.Logger)

	// setup transport
	if defaults.Transport == nil {
		pi.Log.Warnf("Received a nil transport\n")
		defaults.Transport = http.DefaultTransport
	}
	p.nextRoundTripper = defaults.Transport
	defaults.Transport = p
}

func (p *GateQPOption) Shutdown() {
	defer func() {
		if r := recover(); r != nil {
			pi.Log.Warnf("Recovered from panic during Shutdown! Recover: %v", r)
		}
		pi.Log.Sync()
	}()
	p.securityPlug.Shutdown()
}
