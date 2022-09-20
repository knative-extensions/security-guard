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

	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
	utils "knative.dev/security-guard/pkg/guard-utils"
)

type fakeKmgr struct{}

func (f *fakeKmgr) InitConfigs() {}

func (f *fakeKmgr) Read(ns string, sid string, isCm bool) (*spec.GuardianSpec, error) {
	return new(spec.GuardianSpec), nil
}

func (f *fakeKmgr) Create(ns string, sid string, isCm bool, guardianSpec *spec.GuardianSpec) error {
	return nil
}

func (f *fakeKmgr) Set(ns string, sid string, isCm bool, guardianSpec *spec.GuardianSpec) error {
	return nil
}

func (f *fakeKmgr) GetGuardian(ns string, sid string, cm bool, autoActivate bool) *spec.GuardianSpec {
	return new(spec.GuardianSpec)
}

func (f *fakeKmgr) Watch(ns string, cmFlag bool, set func(ns string, sid string, cmFlag bool, g *spec.GuardianSpec)) {
}

func Test_serviceKey(t *testing.T) {
	type args struct {
		ns     string
		sid    string
		cmFlag bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "sid.ns with crd",
			args: args{ns: "ns", sid: "sid", cmFlag: false},
			want: "sid.ns",
		},
		{
			name: "sid.ns with cm",
			args: args{ns: "ns", sid: "sid", cmFlag: true},
			want: "sid.ns.cm",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := serviceKey(tt.args.ns, tt.args.sid, tt.args.cmFlag); got != tt.want {
				t.Errorf("serviceKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_services_tick(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "simple",
		},
	}
	log = utils.CreateLogger("debug")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := new(services)
			s.cache = make(map[string]*serviceRecord, 64)
			s.namespaces = make(map[string]bool, 4)
			s.kmgr = new(fakeKmgr)

			s.tick()

			r9 := s.get("ns", "sid9", false)
			if r9 == nil {
				t.Errorf("Expected record received nil")
			}
			s.set("ns", "sid1", false, new(spec.GuardianSpec))
			if len(s.cache) != 2 {
				t.Errorf("Expected 2 in cache, have %d", len(s.cache))
			}
			if s.cache["sid1.ns"] == nil {
				t.Errorf("Expected s.cache['sid1.ns'] to not be nil - %v", s.cache)
			}
			if s.cache["sid1.ns"].pile.Count != 0 {
				t.Errorf("Expected pileCount of 0 in cache, have %d", s.cache["sid1.ns"].pile.Count)
			}
			if s.cache["sid1.ns"].cmFlag != false {
				t.Errorf("Expected false cmFLag")
			}
			if s.cache["sid1.ns"].ns != "ns" {
				t.Errorf("Expected ns to be 'ns'")
			}
			if s.cache["sid1.ns"].sid != "sid1" {
				t.Errorf("Expected sid to be 'sid1'")
			}
			r1 := s.get("ns", "sid1", false)
			if r1 == nil {
				t.Errorf("Expected record received nil")
			}
			profile1 := &spec.SessionDataProfile{}
			profile1.Req.Method.ProfileString("Get")
			pile1 := spec.SessionDataPile{}
			pile1.Add(profile1)

			// add three more
			profile1 = &spec.SessionDataProfile{}
			profile1.Req.Method.ProfileString("Get")
			pile1.Add(profile1)
			profile1 = &spec.SessionDataProfile{}
			profile1.Req.Method.ProfileString("Get")
			pile1.Add(profile1)
			profile1 = &spec.SessionDataProfile{}
			profile1.Req.Method.ProfileString("Get")
			pile1.Add(profile1)
			if pile1.Count != 4 {
				t.Errorf("Expected len(pile1.Req.Method.List) of 4 in cache, have %d", pile1.Count)
			}
			s.merge(r1, &pile1)
			if s.cache["sid1.ns"].pile.Count != 4 {
				t.Errorf("Expected pileCount of 4 in cache, have %d", s.cache["sid1.ns"].pile.Count)
			}
			s.set("ns", "sid2", true, new(spec.GuardianSpec))
			s.update("ns", "sid3", false, new(spec.GuardianSpec))
			s.update("ns", "sid4", false, new(spec.GuardianSpec))
			if len(s.cache) != 5 {
				t.Errorf("Expected 5 in cache, have %d", len(s.cache))
			}
			s.update("ns", "sid4", false, nil)
			s.update("ns", "sid5", false, nil)

			r2 := s.get("ns", "sid2", true)
			r3 := s.get("ns", "sid3", false)
			profile2 := &spec.SessionDataProfile{}
			profile2.Req.Method.ProfileString("Get")
			profile3 := &spec.SessionDataProfile{}
			profile3.Req.Method.ProfileString("Get")

			pile2 := spec.SessionDataPile{}
			pile2.Add(profile2)
			pile2.Add(&spec.SessionDataProfile{})
			s.merge(r2, &pile2)
			pile3 := spec.SessionDataPile{}
			pile3.Add(profile3)
			pile3.Add(&spec.SessionDataProfile{})
			s.merge(r3, &pile3)

			s.tick()
			s.tick()

			profile11 := &spec.SessionDataProfile{}
			profile11.Req.Method.ProfileString("Get")
			pile11 := spec.SessionDataPile{}
			pile11.Add(profile11)
			s.merge(r1, &pile11)
			s.tick()
			s.tick()
			if len(s.cache) != 4 {
				t.Errorf("Expected 4 in cache, have %d", len(s.cache))
			}
		})
	}
}
