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
	"sync"
	"time"

	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
	guardKubeMgr "knative.dev/security-guard/pkg/guard-kubemgr"
	pi "knative.dev/security-guard/pkg/pluginterfaces"
)

const (
	pileMergeLimit         = uint32(1000)
	numSamplesLimit        = uint32(1000000)
	pileLearnMinTime       = 30 * 1000000000     // 30sec
	guardianPersistMinTime = 5 * 60 * 1000000000 // 5min
)

// A cached record kept by guard-service for each deployed service
type serviceRecord struct {
	ns                     string               // namespace of the deployed service
	sid                    string               // name of the deployed service
	cmFlag                 bool                 // indicate if the deployed service uses a ConfigMap (or CRD)
	guardianSpec           *spec.GuardianSpec   // a copy of the cached deployed service Guardian (RO - no mutext needed)
	pile                   spec.SessionDataPile // the deployed service Pile (RW - protected with pileMutex)
	pileLastLearn          time.Time            // Last time we learned
	guardianLastPersist    time.Time            // Last time we stored the guardian
	guardianPersistCounter uint                 // Counter guardian peristed
	guardianLearnCounter   uint                 // Counter guardian learned
	pileMergeCounter       uint                 // Counter pile merged
	pileMutex              sync.Mutex           // protect access to the pile
	alerts                 uint                 // num of alerts
	deleted                bool                 // mark that record was deleted
}

// service cache maintaining a cached record per deployed service
type services struct {
	kmgr               guardKubeMgr.KubeMgrInterface // KubeMgr to access KuebApi during cache misses
	mutex              sync.Mutex                    // protect access to cache map and to namespaces map
	cache              map[string]*serviceRecord     // the cache
	namespaces         map[string]bool               // list of namespaces to watch for changes in ConfigMaps and CRDs
	records            []*serviceRecord              // list of records to periodically process learn and store during tick()
	lastCreatedRecords time.Time                     // last time we created the records

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
	return s
}

func (s *services) start() {
	// cant be tested due to KubeMgr
	s.kmgr.InitConfigs()
}

func (s *services) createRecords() {
	if time.Since(s.lastCreatedRecords) < guardianPersistMinTime {
		// no need to build the list until it is time to have a fresh look at all records
		return
	}
	s.lastCreatedRecords = time.Now()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Assign more work to be done now and in future ticks
	s.records = make([]*serviceRecord, len(s.cache))
	i := 0
	for _, r := range s.cache {
		s.records[i] = r
		i++
	}
}

func (s *services) flushTickerRecords() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Assign more work to be done now and in future ticks
	s.records = make([]*serviceRecord, len(s.cache))
	i := 0
	for _, r := range s.cache {
		s.records[i] = r
		r.pileLastLearn = time.UnixMicro(0)
		r.guardianLastPersist = time.UnixMicro(0)
		i++
	}
}

// Periodical background work to ensure:
// 1. Small unused piles are eventually learned
// 2. Learned unused guardians are eventually stored using KubeApi
// In some unrealistic case, it is possible that ~1K ticks (1000 seconds = ~20m)
// will be needed to persist all records (assuming all waiting to be persisted)

func (s *services) tick() {
	// Tick should not include any asynchronous work
	// Move all asynchronous work (e.g. KubeApi work) to go routines

	// try up to 100 records per tick to find one that can be persisted
	numRecordsToProcess := len(s.records)
	if numRecordsToProcess == 0 {
		// May loop over some ~10K service records
		s.createRecords()
		return
	}

	if numRecordsToProcess > 100 {
		numRecordsToProcess = 100
	}

	// find a record to persist
	// May loop and learn upto 100 service records + may persist 10
	i := 0              // i is the index of the record to learn
	persistCounter := 0 // number of records we persisted
	for ; i < numRecordsToProcess; i++ {
		r := s.records[i]
		if !r.deleted {
			if s.learnAndPersistGuardian(r) {
				persistCounter++
				if persistCounter > 10 {
					i++
					break
				}

			}
		}
	}

	// remove the records we processed
	s.records = s.records[i:]
}

// delete from cache
func (s *services) delete(ns string, sid string, cmFlag bool) {
	service := serviceKey(ns, sid, cmFlag)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if r, ok := s.cache[service]; ok {
		r.deleted = true
	}

	delete(s.cache, service)
	pi.Log.Debugf("deleteSession %s", service)
}

// get from cache or from KubeApi (or get a default Guardian)
// if new namespace, start watching this namespace for changes in guardians
func (s *services) get(ns string, sid string, cmFlag bool) *serviceRecord {
	var knownNamespace bool = true

	service := serviceKey(ns, sid, cmFlag)

	s.mutex.Lock()
	// check if known Namespace
	_, knownNamespace = s.namespaces[ns]
	if !knownNamespace {
		s.namespaces[ns] = true
	}
	// try to get from cache
	record := s.cache[service]
	s.mutex.Unlock()
	// Must unlock s.mutex before s.kmgr.Watch, s.kmgr.GetGuardian, s.set

	// watch any unknown namespace
	if !knownNamespace {
		go s.kmgr.Watch(ns, cmFlag, s.update)
	}

	if record == nil {
		// not cached, get from kubeApi or create a default and add to cache
		record = s.set(ns, sid, cmFlag, s.kmgr.GetGuardian(ns, sid, cmFlag, true))
	}
	// record is never nil here

	return record
}

// set to cache
// caller ensures that guardianSpec is never nil
func (s *services) set(ns string, sid string, cmFlag bool, guardianSpec *spec.GuardianSpec) *serviceRecord {
	// we have  a new guardianSpec from update() or from get()
	if guardianSpec.Learned != nil {
		guardianSpec.Learned.Prepare()
	}

	service := serviceKey(ns, sid, cmFlag)

	s.mutex.Lock()
	defer s.mutex.Unlock()
	record, exists := s.cache[service]
	if !exists {
		record = new(serviceRecord)
		record.pile.Clear()
		record.pileLastLearn = time.Now()
		record.guardianLastPersist = time.Now()
		record.ns = ns
		record.sid = sid
		record.cmFlag = cmFlag
		s.cache[service] = record
	}

	record.guardianSpec = guardianSpec
	pi.Log.Debugf("cache record for %s.%s", ns, sid)
	return record
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
func (s *services) mergeAndLearnAndPersistGuardian(record *serviceRecord, pile *spec.SessionDataPile) {
	if pile != nil && pile.Count > 0 {
		// Must unlock pileMutex before s.learnPile
		record.pileMutex.Lock()
		record.pile.Merge(pile)
		record.pileMutex.Unlock()
		record.pileMergeCounter++
	}

	s.learnAndPersistGuardian(record)
}

// Update the guardian using the pile
// Persist guardian using KubeAPI
// Return true if persisted, false if not
func (s *services) learnAndPersistGuardian(record *serviceRecord) bool {
	var shouldPersist bool
	var shouldLearn bool

	if record.guardianSpec.Learned == nil {
		// Our first guardian
		record.guardianSpec.Learned = new(spec.SessionDataConfig)
		shouldPersist = true
		shouldLearn = true
	} else {
		// we already have a critiria - do we need to learn again?
		if record.pile.Count >= pileMergeLimit || time.Since(record.pileLastLearn) >= pileLearnMinTime {
			shouldLearn = true
		}

		// we already have a critiria - do we need to persist?
		if time.Since(record.guardianLastPersist) > guardianPersistMinTime {
			shouldPersist = true
		}
	}

	if shouldLearn && record.pile.Count > 0 {
		// ok, lets learn
		record.pileMutex.Lock()
		record.guardianSpec.Learned.Learn(&record.pile)
		record.guardianSpec.NumSamples += record.pile.Count
		if record.guardianSpec.NumSamples > numSamplesLimit {
			record.guardianSpec.NumSamples = numSamplesLimit
		}
		record.pileLastLearn = time.Now()
		record.pile.Clear()
		record.pileMutex.Unlock()
		record.guardianLearnCounter++
		// Must unlock record.pileMutex before s.persist
	}

	if shouldPersist {
		// update the kubeApi record
		record.guardianLastPersist = time.Now()
		record.guardianPersistCounter++
		go s.persistGuardian(record)
	}
	return shouldPersist
}

func (s *services) persistGuardian(record *serviceRecord) {
	if err := s.kmgr.Set(record.ns, record.sid, record.cmFlag, record.guardianSpec); err != nil {
		pi.Log.Infof("Failed to update KubeApi with new config %s.%s: %v", record.ns, record.sid, err)
	} else {
		pi.Log.Debugf("Update KubeApi with new config %s.%s", record.ns, record.sid)
	}
}

func (s *services) deletePod(record *serviceRecord, podname string) {
	s.kmgr.DeletePod(record.ns, podname)
}
