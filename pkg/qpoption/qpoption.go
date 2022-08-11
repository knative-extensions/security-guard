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
	"knative.dev/serving/pkg/queue"
	"knative.dev/serving/pkg/queue/sharedmain"

	pi "knative.dev/security-guard/pkg/pluginterfaces"
)

// This is a Knative Queue Proxy Option (QPOption) package to manage the life cycle and configure
// a single security plug.
// It can be extended in the future to managing multiple security plugs by using the rtplugs package

var annotationsFilePath = queue.PodInfoVolumeMountPath + "/" + queue.PodInfoAnnotationsFilename
var qpExtensionPrefix = "qpextension.knative.dev/"

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

func (p *GateQPOption) ProcessAnnotations(annotationsPath string, qpExtPrefix string) bool {
	file, err := os.Open(annotationsPath)
	if err != nil {
		p.defaults.Logger.Debugf("File %s is can not be opened - is PodInfo mounted? os.Open Error: %s", annotationsPath, err.Error())
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
		if strings.HasPrefix(k, qpExtPrefix) && len(k) > len(qpExtPrefix) {
			v = strings.TrimSuffix(strings.TrimPrefix(v, "\""), "\"")
			k = k[len(qpExtPrefix):]
			keyParts := strings.Split(k, "-")
			if len(keyParts) < 2 {
				continue
			}
			extension := keyParts[0]
			action := keyParts[1]
			if strings.EqualFold(extension, p.securityPlug.PlugName()) {
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
		p.defaults.Logger.Infof("File %s - scanner Error %s", annotationsPath, err.Error())
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

	if pi.RoundTripPlugs == nil || len(pi.RoundTripPlugs) == 0 {
		pi.Log.Warnf("Image was created with qpoption package but without a Plug")
		return
	}
	if len(pi.RoundTripPlugs) > 1 {
		pi.Log.Warnf("Image was created with more then one plug")
		return
	}
	p.securityPlug = pi.RoundTripPlugs[0]
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

	if !p.ProcessAnnotations(annotationsFilePath, qpExtensionPrefix) || !p.activated {
		pi.Log.Debugf("%s is not activated", p.securityPlug.PlugName())
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
