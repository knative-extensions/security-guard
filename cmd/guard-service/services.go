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
	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
	guardKubeMgr "knative.dev/security-guard/pkg/guard-kubemgr"
)

var maxPileCount = uint32(1000)

// A cached record kept by guard-service for each deployed service
type serviceRecord struct {
	// namespace of the deployed service
	ns string
	// name of the deployed service
	sid string
	// indicate if the deployed service uses a ConfigMap (or CRD)
	cmFlag bool
	// a copy of the cached deployed service Guardian
	guardianSpec *spec.GuardianSpec
	// the deployed service Pile and a counter indicating the number of profiles piled
	pile spec.SessionDataPile
}

// service cache maintaining a cached record per deployed service
type services struct {
	// KubeMgr to access KuebApi during cache misses
	kmgr guardKubeMgr.KubeMgrInterface
	// the cache
	cache map[string]*serviceRecord
	// list of namespaces to watch for changes in ConfigMaps and CRDs
	namespaces map[string]bool
	// list of cache keys to periodically process during a tick()
	tickerKeys []string
}

// determine the cacheKey from its components
func serviceKey(ns string, sid string, cmFlag bool) string {
	service := sid + "." + ns
	if cmFlag {
		service += ".cm"
	}
	return service
}

func newServices() *services {
	s := new(services)
	s.cache = make(map[string]*serviceRecord, 64)
	s.namespaces = make(map[string]bool, 4)
	s.kmgr = guardKubeMgr.NewKubeMgr()
	s.kmgr.InitConfigs()
	return s
}

// periodical background work to ensure small piles eventually are stored using KubeApi
// uses a single KubeMgr.Set() per tick(), a tick is 5 seconds by default
// For 1000 deployed active but slow services, it may take an hour and a half to store the config in KubeAPI()
func (s *services) tick() {
	if len(s.cache) == 0 {
		return
	}
	if len(s.tickerKeys) == 0 {
		s.tickerKeys = make([]string, len(s.cache))
		i := 0
		for k := range s.cache {
			s.tickerKeys[i] = k
			i++
		}
	}
	i := 0
	maxIterations := len(s.tickerKeys)
	if maxIterations > 100 {
		maxIterations = 100
	}
	for ; i < maxIterations; i++ {
		// try to learnPile the first of maxIterations records
		if record, exists := s.cache[s.tickerKeys[i]]; exists {
			if s.learnPile(record) {
				// learnPile one record
				i++
				break
			}
		}
	}
	// remove the keys we processed from the key slice
	s.tickerKeys = s.tickerKeys[i:]
}

// delete from cache
func (s *services) delete(ns string, sid string, cmFlag bool) {
	service := serviceKey(ns, sid, cmFlag)
	delete(s.cache, service)
	log.Debugf("deleteSession %s", service)
}

// get from cache or from KubeApi (or get a default Guardian)
func (s *services) get(ns string, sid string, cmFlag bool) *serviceRecord {
	service := serviceKey(ns, sid, cmFlag)
	// try to get from cache
	record := s.cache[service]
	if record == nil {
		// not cached, get from kubeApi or create a default and add to cache
		s.set(ns, sid, cmFlag, s.kmgr.GetGuardian(ns, sid, cmFlag, true))
		record = s.cache[service]
	}
	// record is never nil
	return record
}

// set to cache
// caller ensures that guardianSpec is never nil
func (s *services) set(ns string, sid string, cmFlag bool, guardianSpec *spec.GuardianSpec) {
	service := serviceKey(ns, sid, cmFlag)

	record, exists := s.cache[service]
	if !exists {
		record = new(serviceRecord)
		record.pile.Clear()
		record.ns = ns
		record.sid = sid
		record.cmFlag = cmFlag
		s.cache[service] = record
	}
	record.guardianSpec = guardianSpec
	if _, ok := s.namespaces[ns]; !ok {
		s.namespaces[ns] = true
		go s.kmgr.Watch(ns, cmFlag, s.update)
	}
	log.Debugf("cache record for %s.%s", ns, sid)
}

// update cache
// delete if guardianSpec is nil, set otherwise
func (s *services) update(ns string, sid string, cmFlag bool, guardianSpec *spec.GuardianSpec) {
	if guardianSpec == nil {
		s.delete(ns, sid, cmFlag)
	} else {
		s.set(ns, sid, cmFlag, guardianSpec)
	}
}

// update the record pile by merging a new pile
func (s *services) merge(record *serviceRecord, pile *spec.SessionDataPile) {
	record.pile.Merge(pile)
	if record.pile.Count > maxPileCount {
		s.learnPile(record)
	}
}

// update the record guardianSpec by learning a new config and fusing with the record existing config
// update KubeAPI as well.
// return true if we try to learn and access kubeApi, false if count is zero and we have nothing to do
func (s *services) learnPile(record *serviceRecord) bool {
	if record.pile.Count == 0 {
		return false
	}
	config := new(spec.SessionDataConfig)

	// TBD move to periodical under Ticker
	config.Learn(&record.pile)
	record.pile.Clear()
	if record.guardianSpec.Learned != nil {
		config.Fuse(record.guardianSpec.Learned)
	}

	// update the cached record
	record.guardianSpec.Learned = config
	record.guardianSpec.Learned.Active = true

	// update the kubeApi record
	if err := s.kmgr.Set(record.ns, record.sid, record.cmFlag, record.guardianSpec); err != nil {
		log.Infof("Failed to update KubeApi with new config %s.%s: %v", record.ns, record.sid, err)
		return true
	}
	log.Debugf("Update KubeApi with new config %s.%s", record.ns, record.sid)
	return true
}
