//go:build !race
// +build !race

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
	"testing"
	"time"

	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
	utils "knative.dev/security-guard/pkg/guard-utils"
)

func addToPile(s *services) {
	profile1 := &spec.SessionDataProfile{}
	profile1.Req.Method.ProfileString("Get")
	pile1 := spec.SessionDataPile{}
	pile1.Add(profile1)
	r1 := s.get("ns", "sid1", false)
	s.merge(r1, &pile1)
}

func Test_learner_mainEventLoop(t *testing.T) {
	log = utils.CreateLogger("x")
	quit := make(chan string)

	// services
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = new(fakeKmgr)

	ticker := utils.NewTicker(100000)
	ticker.Parse("", 100000)
	ticker.Start()

	addToPile(s)

	t.Run("simple", func(t *testing.T) {
		l := &learner{
			services:        s,
			pileLearnTicker: ticker,
		}
		if s.cache["sid1.ns"].pile.Count != 1 {
			t.Errorf("Expected 1 in pile  have %d", s.cache["sid1.ns"].pile.Count)
		}

		// Start event loop
		go l.mainEventLoop(quit)

		<-time.After(100 * time.Millisecond)

		if s.cache["sid1.ns"].pile.Count != 0 {
			t.Errorf("Expected 0 in pile  have %d", s.cache["sid1.ns"].pile.Count)
		}
		quit <- "test done"
		// Asked event loop to quit
		<-time.After(100 * time.Millisecond)

		addToPile(s)
		if s.cache["sid1.ns"].pile.Count != 1 {
			t.Errorf("Expected 1 in pile  have %d", s.cache["sid1.ns"].pile.Count)
		}

		<-time.After(100 * time.Millisecond)

		if s.cache["sid1.ns"].pile.Count != 1 {
			t.Errorf("Expected 1 in pile  have %d", s.cache["sid1.ns"].pile.Count)
		}
	})

}
