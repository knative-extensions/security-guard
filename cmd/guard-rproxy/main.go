package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"runtime/debug"

	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"

	"knative.dev/pkg/signals"
	_ "knative.dev/security-guard/pkg/guard-gate"
	utils "knative.dev/security-guard/pkg/guard-utils"
	pi "knative.dev/security-guard/pkg/pluginterfaces"
)

type config struct {
	ServiceName          string `split_words:"true" required:"true"`
	Namespace            string `split_words:"true" required:"true"`
	ServiceUrl           string `split_words:"true" required:"true"`
	UseCrd               bool   `split_words:"true" required:"false"`
	MonitorPod           bool   `split_words:"true" required:"false"`
	GuardUrl             string `split_words:"true" required:"false"`
	LogLevel             string `split_words:"true" required:"false"`
	GuardProxyPort       string `split_words:"true" required:"false"`
	PodMonitorInterval   string `split_words:"true" required:"false"`
	ReportPileInterval   string `split_words:"true" required:"false"`
	GuardianLoadInterval string `split_words:"true" required:"false"`
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

func preMain() (guardGate *GuardGate, mux *http.ServeMux, target string, plugConfig map[string]string, sid string, ns string, log *zap.SugaredLogger) {
	var env config
	if err := envconfig.Process("", &env); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process environment: %s\n", err.Error())
		return
	}

	plugConfig = make(map[string]string)
	guardGate = new(GuardGate)

	log = utils.CreateLogger(env.LogLevel)
	defer log.Sync()
	pi.Log = log

	if env.GuardUrl == "" {
		// use default
		env.GuardUrl = "http://guard-service.default"
	} else {
		plugConfig["guard-url"] = env.GuardUrl
	}

	// When using a Reverse Proxy, it has a default to not use pod monitoring
	plugConfig["monitor-pod"] = "false" // default when used as a standalone
	if env.MonitorPod {
		plugConfig["monitor-pod"] = "true"
	}

	// When using a Reverse Proxy, it has a default to work with CM
	plugConfig["use-cm"] = "true"
	if env.UseCrd {
		plugConfig["use-cm"] = "false"
	}

	plugConfig["guardian-load-interval"] = env.GuardianLoadInterval
	plugConfig["report-pile-interval"] = env.ReportPileInterval
	plugConfig["pod-monitor-interval"] = env.PodMonitorInterval

	sid = env.ServiceName
	ns = env.Namespace

	log.Infof("guard-proxy serving serviceName: %s, namespace: %s, serviceUrl: %s", sid, ns, env.ServiceUrl)
	parsedUrl, err := url.Parse(env.ServiceUrl)
	if err != nil {
		log.Errorf("Failed to parse serviceUrl: %s", err.Error())
		return
	}
	log.Infof("guard-proxy parsedUrl: %v", parsedUrl)

	proxy := httputil.NewSingleHostReverseProxy(parsedUrl)

	// Hook using RoundTripper

	securityPlug := pi.GetPlug()

	guardGate.securityPlug = securityPlug
	proxy.Transport = guardGate.Transport(proxy.Transport)

	target = ":22000"
	if env.GuardProxyPort != "" {
		target = fmt.Sprintf(":%s", env.GuardProxyPort)
	}

	mux = http.NewServeMux()
	mux.Handle("/", proxy)
	log.Infof("Starting Reverse Proxy on port %s", target)
	return
}

func main() {
	guardGate, mux, target, plugConfig, sid, ns, log := preMain()
	if mux == nil {
		os.Exit(1)
	}

	guardGate.securityPlug.Init(signals.NewContext(), plugConfig, sid, ns, log)
	defer guardGate.securityPlug.Shutdown()

	err := http.ListenAndServe(target, mux)
	log.Fatalf("Failed to open http local service: %s", err.Error())
}
