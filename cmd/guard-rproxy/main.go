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

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"runtime/debug"
	"time"

	"github.com/kelseyhightower/envconfig"

	"knative.dev/networking/pkg/certificates"
	"knative.dev/pkg/signals"
	_ "knative.dev/security-guard/pkg/guard-gate"
	utils "knative.dev/security-guard/pkg/guard-utils"
	pi "knative.dev/security-guard/pkg/pluginterfaces"
	"knative.dev/serving/pkg/queue"
)

type config struct {
	ServiceName          string `split_words:"true" required:"true"`
	Namespace            string `split_words:"true" required:"true"`
	ProtectedService     string `split_words:"true" required:"false"`
	GuardServiceUrl      string `split_words:"true" required:"false"`
	Port                 string `split_words:"true" required:"false"`
	UseCrd               string `split_words:"true" required:"false"`
	MonitorPod           string `split_words:"true" required:"false"`
	AnalyzeBody          string `split_words:"true" required:"false"`
	PodMonitorInterval   string `split_words:"true" required:"false"`
	GuardianSyncInterval string `split_words:"true" required:"false"`
	LogLevel             string `split_words:"true" required:"false"`
}

type GuardGate struct {
	nextRoundTripper http.RoundTripper // the next roundtripper
	securityPlug     pi.RoundTripPlug
}

func (p *GuardGate) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			pi.Log.Warnf("Recovered from panic during RoundTrip! Recover: %v\n", recovered)
			pi.Log.Debugf("Stacktrace from panic: \n %s\n" + string(debug.Stack()))
			err = errors.New("paniced during RoundTrip")
			resp = nil
		}
	}()
	req.Host = "" // req.URL.Host

	if req, err = p.securityPlug.ApproveRequest(req); err == nil {
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

func (p *GuardGate) Transport(t http.RoundTripper) http.RoundTripper {
	if t == nil {
		t = http.DefaultTransport
	}
	p.nextRoundTripper = t
	return p
}

func preMain(env *config) (guardGate *GuardGate, mux *http.ServeMux, target string, plugConfig map[string]string, sid string, ns string) {
	plugConfig = make(map[string]string)
	guardGate = new(GuardGate)

	utils.CreateLogger(env.LogLevel)

	var err error
	var buf []byte
	buf, err = os.ReadFile(path.Join(queue.CertDirectory, certificates.CaCertName))
	if err != nil {
		pi.Log.Debugf("ROOT_CA (%s) is missing - Insecure communication, working without TLS RootCA!", path.Join(queue.CertDirectory, certificates.CaCertName))
	}
	if err == nil {
		plugConfig["rootca"] = string(buf)
	}

	if env.GuardServiceUrl != "" {
		plugConfig["guard-url"] = env.GuardServiceUrl
	}

	plugConfig["monitor-pod"] = "true" // default when used as a standalone
	if env.MonitorPod == "false" {
		plugConfig["monitor-pod"] = "false"
	}

	plugConfig["analyze-body"] = "true" // default when used as a standalone
	if env.AnalyzeBody == "false" {
		plugConfig["analyze-body"] = "false"
	}

	plugConfig["use-crd"] = "true" // default when used as a standalone
	if env.UseCrd == "false" {
		plugConfig["use-crd"] = "false"
	}

	plugConfig["guardian-sync-interval"] = env.GuardianSyncInterval
	plugConfig["pod-monitor-interval"] = env.PodMonitorInterval

	sid = env.ServiceName
	ns = env.Namespace

	if len(ns) == 0 || len(sid) == 0 || sid == "ns-"+ns {
		pi.Log.Errorf("Failed illegal sid or ns")
		return
	}

	var protectedPort int
	var protectedUrl string

	// we support env.ProtectedService of three types:
	// - <empty string> - signifying to use the default url http://127.0.0.1:8080
	// - :<port number> - default url with modified port http://127.0.0.1:<port>
	// - Any legal url
	if env.ProtectedService == "" {
		protectedUrl = "http://127.0.0.1:8080"
	} else {
		if n, err := fmt.Sscanf(env.ProtectedService, ":%d", &protectedPort); err == nil && n == 1 && protectedPort > 0 && protectedPort < 0x10000 {
			// note we ignored all runes after the integer, they will also not effect us later
			protectedUrl = fmt.Sprintf("http://127.0.0.1:%d", protectedPort)
		} else {
			protectedUrl = env.ProtectedService
		}
	}

	// now we should have a url in protectedUrl, we parse it to ensure it is really a url
	parsedUrl, err := url.Parse(protectedUrl)
	if err != nil {
		pi.Log.Errorf("Failed to parse serviceUrl: %s", err.Error())
		return
	}
	if parsedUrl.Scheme != "http" {
		pi.Log.Errorf("Failed to parse serviceUrl - Use only urls that start with 'http://'")
		return
	}

	pi.Log.Infof("guard-proxy serving serviceName: %s, namespace: %s, serviceUrl: %s", sid, ns, parsedUrl.String())
	proxy := httputil.NewSingleHostReverseProxy(parsedUrl)

	// Hook using RoundTripper

	securityPlug := pi.GetPlug()

	guardGate.securityPlug = securityPlug
	proxy.Transport = guardGate.Transport(proxy.Transport)

	target = ":22000"
	if env.Port != "" {
		target = fmt.Sprintf(":%s", env.Port)
	}

	mux = http.NewServeMux()
	mux.Handle("/", proxy)
	pi.Log.Infof("Starting Reverse Proxy on port %s", target)
	return
}

func main() {
	var env config
	if err := envconfig.Process("", &env); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process environment: %s\n", err.Error())
		return
	}

	guardGate, mux, target, plugConfig, sid, ns := preMain(&env)
	if mux == nil {
		os.Exit(1)
	}
	defer utils.SyncLogger()

	signalCtx := signals.NewContext()
	guardGate.securityPlug.Init(signalCtx, plugConfig, sid, ns, pi.Log)
	srv := &http.Server{
		Addr:    target,
		Handler: mux,
	}
	go func(srv *http.Server) {
		err := srv.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			pi.Log.Infof("Http service failed to start %v\n", err)
		} else {
			pi.Log.Infof("Http services stoped!\n")
		}
	}(srv)

	// wait to die
	<-signalCtx.Done()

	pi.Log.Infof("Terminating guard-rproxy")

	// Shutdown the http services
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()
	srv.Shutdown(shutdownCtx)

	// Shutdown guard (including a final sync)
	guardGate.securityPlug.Shutdown()
}
