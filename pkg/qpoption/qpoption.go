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
	"knative.dev/serving/pkg/queue/sharedmain"

	pi "knative.dev/security-guard/pkg/pluginterfaces"
)

// This is a Knative Queue Proxy Option (QPOption) package to manage the life cycle and configure
// a single security plug.
// It can be extended in the future to managing multiple security plugs by using the rtplugs package

var annotationsFilePath = sharedmain.PodInfoAnnotationsPath
var qpOptionPrefix = "qpoption.knative.dev/"

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
		if r := recover(); r != nil {
			pi.Log.Warnf("Recovered from panic during RoundTrip! Recover: %v\n", r)
			pi.Log.Debugf("Stacktrace from panic: \n %s\n" + string(debug.Stack()))
			err = errors.New("panic during RoundTrip")
			resp = nil
		}
	}()

	if req, err = p.securityPlug.ApproveRequest(req); err != nil {
		pi.Log.Debugf("%s: returning error %v", p.securityPlug.PlugName(), err)
		resp = nil
		return
	}

	if resp, err = p.nextRoundTripper.RoundTrip(req); err == nil {
		resp, err = p.securityPlug.ApproveResponse(req, resp)
	}
	return
}

func (p *GateQPOption) ProcessAnnotations() bool {
	file, err := os.Open(annotationsFilePath)
	if err != nil {
		p.defaults.Logger.Debugf("File %s cannot be opened - is PodInfo mounted? os.Open Error: %s", annotationsFilePath, err.Error())
		return false
	}
	defer file.Close()
	p.config = make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txt := scanner.Text()
		txt = strings.ToLower(txt)

		// Annotation structure:
		// 		either: <qpOptionPrefix><extension>-activate=s<val>
		// 		or:     <qpOptionPrefix><extension>-config-<key>=<val>
		parts := strings.Split(txt, "=")

		k := parts[0] // <qpOptionPrefix><extension>-*
		v := parts[1] // <val>
		if strings.HasPrefix(k, qpOptionPrefix) && len(k) > len(qpOptionPrefix) {
			k = k[len(qpOptionPrefix):]

			// k structure: <extenion>-activate or <extension>-config-<key>
			keyParts := strings.Split(k, "-")
			if len(keyParts) < 2 {
				continue
			}
			extension := keyParts[0] // <extension>
			action := keyParts[1]    // activate or config
			if strings.EqualFold(extension, p.securityPlug.PlugName()) {
				// remove double quates if exists
				v = strings.TrimSuffix(strings.TrimPrefix(v, "\""), "\"")
				switch action {
				case "activate":
					if strings.EqualFold(v, "enable") {
						p.activated = true
					}
				case "config":
					if len(keyParts) == 3 {
						extensionKey := keyParts[2]
						p.config[extensionKey] = v
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		p.defaults.Logger.Infof("File %s - scanner Error %s", annotationsFilePath, err.Error())
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

	if p.securityPlug = pi.GetPlug(); p.securityPlug == nil {
		return
	}

	p.defaults = defaults
	namespace := defaults.Env.ServingNamespace
	serviceName := defaults.Env.ServingService
	if serviceName == "" {
		serviceName = defaults.Env.ServingConfiguration
	}

	if defaults.Logger == nil {
		logger, _ := zap.NewProduction()
		defaults.Logger = logger.Sugar()
		defaults.Logger.Warnf("Received a nil logger\n")
	}
	pi.Log = defaults.Logger

	// build p.config

	if !p.ProcessAnnotations() || !p.activated {
		pi.Log.Debugf("No plug was activated")
		return
	}

	pi.Log.Debugf("Activating %s version %s with config %v in pod %s namespace %s", p.securityPlug.PlugName(), p.securityPlug.PlugVersion(), p.config, serviceName, namespace)

	// setup context
	if defaults.Ctx == nil {
		pi.Log.Warnf("Received a nil context\n")
		defaults.Ctx = context.Background()
	}
	defaults.Ctx = p.securityPlug.Init(defaults.Ctx, p.config, serviceName, namespace, defaults.Logger)

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
