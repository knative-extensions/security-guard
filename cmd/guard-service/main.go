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
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
	utils "knative.dev/security-guard/pkg/guard-utils"

	pi "knative.dev/security-guard/pkg/pluginterfaces"
)

type config struct {
	GuardServiceLogLevel string   `split_words:"true" required:"false"`
	GuardServiceInterval string   `split_words:"true" required:"false"`
	GuardServiceAuth     string   `split_words:"true" required:"false"`
	GuardServiceLabels   []string `split_words:"true" required:"false"`
	GuardServiceTls      string   `split_words:"true" required:"false"`
}

type learner struct {
	services        *services
	pileLearnTicker *utils.Ticker
	env             config
	srv             *http.Server
}

func (l *learner) authenticate(req *http.Request) (podname string, sid string, ns string, err error) {
	token := req.Header.Get("Authorization")
	if !strings.HasPrefix(token, "Bearer ") {
		err = fmt.Errorf("missing token")
		return
	}
	token = token[7:]
	podname, sid, ns, err = l.services.kmgr.TokenData(token, l.env.GuardServiceLabels)
	if err != nil {
		err = fmt.Errorf("cant verify token %w", err)
		return
	}
	if sid == "ns-"+ns {
		err = fmt.Errorf("token of a service with illegal name %s", sid)
		return
	}
	return
}

// queryDataNoAuth handle queryString when NoAuth is used
func (l *learner) queryDataNoAuth(query url.Values) (cmFlag bool, pod string, sid string, ns string, err error) {
	// first we do the same as with Auth
	cmFlag, err = l.queryDataAuth(query)
	if err != nil {
		return
	}

	// now get the remaining parameters for the NoAuth case
	sidSlice := query["sid"]
	nsSlice := query["ns"]
	podSlice := query["pod"]

	if len(sidSlice) != 1 || len(nsSlice) != 1 || len(podSlice) != 1 {
		err = fmt.Errorf("query should have a single value for pod, sid and ns")
		return
	}

	// extract and sanitize pod, sid and ns
	pod = utils.Sanitize(podSlice[0])
	sid = utils.Sanitize(sidSlice[0])
	ns = utils.Sanitize(nsSlice[0])

	if sid == "ns-"+ns {
		err = fmt.Errorf("query sid of a service with illegal name that starts with ns-")
		return
	}

	if len(pod) < 1 {
		err = fmt.Errorf("query missing pod")
		return
	}

	if len(sid) < 1 {
		err = fmt.Errorf("query missing sid")
		return
	}

	if len(ns) < 1 {
		err = fmt.Errorf("query missing ns")
		return
	}

	return
}

// queryDataAuth handle queryString when Auth is used
func (l *learner) queryDataAuth(query url.Values) (cmFlag bool, err error) {
	cmFlagSlice := query["cm"]

	if len(cmFlagSlice) > 1 {
		err = fmt.Errorf("query has more then one cmflag value")
		return
	}

	// extract and sanitize cmFlag
	if len(cmFlagSlice) > 0 {
		cmFlag = (cmFlagSlice[0] == "true")
	}

	return
}

func (l *learner) baseHandler(w http.ResponseWriter, req *http.Request) (record *serviceRecord, podname string, err error) {
	var sid, ns string
	var cmFlag bool

	if l.env.GuardServiceAuth != "false" {
		cmFlag, err = l.queryDataAuth(req.URL.Query())
		if err != nil {
			pi.Log.Infof("queryData failed with %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		podname, sid, ns, err = l.authenticate(req)
		if err != nil {
			pi.Log.Infof("authenticate failed with %v", err)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
	} else {
		cmFlag, podname, sid, ns, err = l.queryDataNoAuth(req.URL.Query())
		if err != nil {
			pi.Log.Infof("queryData failed with %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// get session record, create one if does not exist
	record = l.services.get(ns, sid, cmFlag)
	if record == nil {
		// should never happen
		err = fmt.Errorf("no record created")
		pi.Log.Infof("internal error %v for request ns %s, sid %s, pod %s, cmFlag %t", err, ns, sid, podname, cmFlag)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

func (l *learner) processSync(w http.ResponseWriter, req *http.Request) {
	var syncReq spec.SyncMessageReq
	var syncResp spec.SyncMessageResp

	record, podname, err := l.baseHandler(w, req)
	if err != nil {
		return
	}
	if req.Method != "POST" || req.URL.Path != "/sync" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	if req.ContentLength == 0 || req.Body == nil {
		http.Error(w, "400 not found.", http.StatusBadRequest)
		return
	}

	err = json.NewDecoder(req.Body).Decode(&syncReq)
	if err != nil {
		pi.Log.Infof("processSync error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	record.recordMutex.Lock()
	if syncReq.IamCompromised {
		pi.Log.Infof("Gate %s ns %s reported Pod is Compromised!!", podname, record.ns)
		l.services.deletePod(record, podname)
	}

	// merge if needed, learn if needed and persist if needed
	l.services.mergeAndLearnAndPersistGuardian(record, syncReq.Pile)
	if syncReq.Pile != nil {
		pi.Log.Debugf("Sync %s.%s pod %s Pile %d Alerts %d Compromised %t => mergeCounter %d learnCounter %d persistCounter %d", record.ns, record.sid, podname, syncReq.Pile.Count, len(syncReq.Alerts), syncReq.IamCompromised, record.pileMergeCounter, record.guardianLearnCounter, record.guardianPersistCounter)
	}

	if syncReq.Alerts != nil {
		pi.Log.Infof("Pod %s ns %s sent %d Alerts", podname, record.ns, len(syncReq.Alerts))
		for _, alert := range syncReq.Alerts {
			record.alerts++
			time := time.Unix(alert.Time, 0)
			pi.Log.Debugf("---- %d alerts since %02d:%02d:%02d %s -> %s", alert.Count, time.Hour(), time.Minute(), time.Second(), alert.Level, alert.Decision.String(""))
		}
	}
	syncResp.Guardian = record.guardianSpec
	record.recordMutex.Unlock()

	buf, err := json.Marshal(syncResp)
	if err != nil {
		// should never happen
		pi.Log.Infof("Servicing processSync error while JSON Marshal %v", err)
		http.Error(w, "Failed to marshal data", http.StatusInternalServerError)
		return
	}
	w.Write(buf)
}

func (l *learner) mainEventProcessing(quit <-chan bool, kill <-chan os.Signal) bool {
	select {
	case <-l.pileLearnTicker.Ch():
		// no reenterncy! No need to protect tick() only data with mutex
		l.services.tick()
	case reason := <-kill:
		pi.Log.Infof("mainEventLoop received kill signal: %s", reason.String())
		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
		l.srv.Shutdown(shutdownCtx)
		defer shutdownRelease()
	case <-quit:
		return false
	}
	return true
}

func (l *learner) mainEventLoop(quit <-chan bool, flushed chan<- bool, kill <-chan os.Signal) {
	// main loop
	alive := true
	for alive {
		alive = l.mainEventProcessing(quit, kill)
	}

	// main loop ended, persisting remaining services records
	pi.Log.Infof("Persist all changed records")
	l.services.flushTickerRecords() // mark all for immediate learn and persist
	for len(l.services.records) > 0 {
		pi.Log.Infof("mainEventLoop completion - persisting %d remaining services records", len(l.services.records))
		<-l.pileLearnTicker.Ch()
		l.services.tick()
	}
	pi.Log.Infof("mainEventLoop done")
	flushed <- true
}

// initialization of the lerner + prepare the web service
func (l *learner) init() (srv *http.Server, quit chan bool, flushed chan bool) {
	utils.CreateLogger(l.env.GuardServiceLogLevel)

	l.pileLearnTicker = utils.NewTicker(time.Second)
	l.pileLearnTicker.Start()

	l.services = newServices()

	mux := http.NewServeMux()
	mux.HandleFunc("/sync", l.processSync)

	target := ":8888"

	srv = &http.Server{
		Addr:    target,
		Handler: mux,
	}

	quit = make(chan bool)
	flushed = make(chan bool)

	pi.Log.Infof("Starting guard-service on %s", target)
	if l.env.GuardServiceAuth != "false" {
		l.env.GuardServiceAuth = "true"
		pi.Log.Infof("Token turned on - clients identity is confirmed")
	} else {
		pi.Log.Infof("Token turned off - clients identity is not confirmed")
	}
	if l.env.GuardServiceTls != "false" {
		pi.Log.Infof("TLS turned on")
	} else {
		pi.Log.Infof("TLS turned off")
	}
	return
}

func main() {
	var err error
	kill := make(chan os.Signal, 1)
	// catch SIGETRM or SIGINTERRUPT
	signal.Notify(kill, syscall.SIGTERM, syscall.SIGINT)

	l := new(learner)
	// affected by env
	if err := envconfig.Process("", &l.env); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process environment: %s\n", err.Error())
		os.Exit(1)
	}

	// move all initialization which can be tested using unit tests to init
	srv, quit, flushed := l.init()

	l.srv = srv

	// cant be tested due to KubeMgr
	l.services.start()
	// start a mainLoop
	go l.mainEventLoop(quit, flushed, kill)

	// affected by file system
	// starts a web service
	if l.env.GuardServiceTls != "false" {
		srv.TLSConfig = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
		}
		_, err = os.Stat("/secrets/public-cert.pem")
		if err == nil {
			err = srv.ListenAndServeTLS("/secrets/public-cert.pem", "/secrets/private-key.pem")
		} else {
			if os.IsNotExist(err) {
				// Since the secret keys should be at some point renamed, if we are here lets try the new names
				err = srv.ListenAndServeTLS("/secrets/tls.crt", "/secrets/tls.key")
			}
		}
	} else {
		err = srv.ListenAndServe()
	}
	if errors.Is(err, http.ErrServerClosed) {
		pi.Log.Infof("Http service orderly shutdown")
	} else {
		pi.Log.Infof("Http service error while starting server: %v", err)
	}
	quit <- true
	<-flushed
	pi.Log.Infof("guard-service done")
}
