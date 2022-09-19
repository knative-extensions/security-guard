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
	"fmt"
	"net/http"
	"os"

	"go.uber.org/zap"
	utils "knative.dev/security-guard/pkg/guard-utils"

	"github.com/kelseyhightower/envconfig"
)

var log *zap.SugaredLogger

type config struct {
	GuardServiceLogLevel string `split_words:"true" required:"false"`
	GuardServicePort     string `split_words:"true" required:"false"`
}

type learner struct {
	services        *services
	pileLearnTicker utils.Ticker
}

func (l *learner) fetchConfig(w http.ResponseWriter, req *http.Request) {
}

func (l *learner) processPile(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte{})
}

func (l *learner) mainEventLoop(quit chan int) {

	for {
		select {
		case <-l.pileLearnTicker.Ch():
			l.services.tick()
		case <-quit:
			log.Info("mainEventLoop was asked to quit!")
			return
		}
	}
}

// Set network policies to ensure that only pods in your trust domain can use the service!
func main() {
	var env config
	if err := envconfig.Process("", &env); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process environment: %s\n", err.Error())
		os.Exit(1)
	}

	l := new(learner)
	l.services = newServices()
	log = utils.CreateLogger(env.GuardServiceLogLevel)
	l.pileLearnTicker.Start()

	http.HandleFunc("/config", l.fetchConfig)
	http.HandleFunc("/pile", l.processPile)

	target := ":8888"
	if env.GuardServicePort != "" {
		target = fmt.Sprintf(":%s", env.GuardServicePort)
	}

	// start a mainLoop
	quit := make(chan int)
	go l.mainEventLoop(quit)

	log.Infof("Starting guard-learner on %s", target)
	err := http.ListenAndServe(target, nil)
	log.Infof("Failed to start %v", err)
	quit <- 0
}
