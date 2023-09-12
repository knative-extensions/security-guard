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

package v1alpha1

import (
	"fmt"
	"strings"

	"github.com/zeebo/xxh3"
)

//////////////////// KeyValProfile ////////////////

const MAX_KEYS_LEARNED = 7
const MAX_KEY_LENGTH = 64

// Exposes ValueProfile interface
type KeyValProfile map[string]*SimpleValProfile

// Profile a generic map of key vals
func (profile *KeyValProfile) profileI(args ...interface{}) {
	switch v := args[0].(type) {
	case map[string]string:
		profile.ProfileMapString(v)
	case map[string][]string:
		profile.ProfileMapStringSlice(v)
	default:
		panic("Unsupported type in KeyValProfile")
	}

}

func hashIfNeeded(keyIn string) string {
	l := len(keyIn)
	if l <= MAX_KEY_LENGTH {
		return keyIn
	}
	h := xxh3.HashString(keyIn)
	keyOut := fmt.Sprintf("xxHash %d", h)
	return keyOut
}

func (profile *KeyValProfile) ProfileMapString(keyValMap map[string]string) {
	*profile = nil
	if len(keyValMap) == 0 { // no keys
		*profile = nil
		return
	}
	*profile = make(map[string]*SimpleValProfile, len(keyValMap))
	for k, v := range keyValMap {
		key := hashIfNeeded(k)
		// Profile the concatenated value
		(*profile)[key] = new(SimpleValProfile)
		(*profile)[key].Profile(v)
	}
}

func (profile *KeyValProfile) ProfileMapStringSlice(keyValMap map[string][]string) {
	*profile = nil
	if len(keyValMap) == 0 { // no keys
		*profile = nil
		return
	}
	*profile = make(map[string]*SimpleValProfile, len(keyValMap))
	for k, v := range keyValMap {
		key := hashIfNeeded(k)
		// Concatenate all strings into one value
		// Appropriate for evaluating []string where order should be also preserved
		val := strings.Join(v, " ")

		// Profile the concatenated value
		(*profile)[key] = new(SimpleValProfile)
		(*profile)[key].Profile(val)
	}
}

//////////////////// KeyValPile ////////////////

// Exposes ValuePile interface
type KeyValPile map[string]*SimpleValPile

func (pile *KeyValPile) addI(valProfile ValueProfile) {
	pile.Add(valProfile.(*KeyValProfile))
}

// profile is RO and unchanged - never uses profile internal objects
func (pile *KeyValPile) Add(profile *KeyValProfile) {
	if *pile == nil {
		*pile = make(map[string]*SimpleValPile, 16)
	}
	for key, v := range *profile {
		svp, exists := (*pile)[key]
		if !exists {
			svp = new(SimpleValPile)
			(*pile)[key] = svp
		}
		svp.Add(v)
	}
}

func (pile *KeyValPile) Clear() {
	*pile = nil
}

func (pile *KeyValPile) mergeI(otherValPile ValuePile) {
	pile.Merge(otherValPile.(*KeyValPile))
}

// otherPile is RO and unchanged - never uses otherPile internal objects
func (pile *KeyValPile) Merge(otherPile *KeyValPile) {
	if otherPile == nil {
		return
	}
	if *pile == nil {
		*pile = *otherPile
		return
	}
	for key, val := range *otherPile {
		if _, exists := (*pile)[key]; !exists {
			(*pile)[key] = new(SimpleValPile)
		}
		(*pile)[key].Merge(val)
	}
}

//////////////////// KeyValConfig ////////////////

// Exposes ValueConfig interface
type KeyValConfig struct {
	Vals          map[string]*SimpleValConfig `json:"vals"`          // Profile the value of known keys
	OtherVals     *SimpleValConfig            `json:"otherVals"`     // Profile the values of other keys
	OtherKeynames *SimpleValConfig            `json:"otherKeynames"` // Profile the keynames of other keys
	valScores     [MAX_KEYS_LEARNED]svcScore
}

type svcScore struct {
	score uint32
	key   string
}

func (config *KeyValConfig) decideI(valProfile ValueProfile) *Decision {
	return config.Decide(valProfile.(*KeyValProfile))
}

func (config *KeyValConfig) Decide(profile *KeyValProfile) *Decision {
	var current *Decision

	if profile == nil {
		return nil
	}

	// For each key-val, decide
	for k, v := range *profile {
		// Decide based on a known keys
		if config.Vals != nil && config.Vals[k] != nil {
			DecideChild(&current, config.Vals[k].Decide(v), "KnownKey %s", k)
			continue
		}
		// Decide based on unknown key...
		if config.OtherKeynames == nil || config.OtherVals == nil {
			DecideInner(&current, 1, "Key %s is not known", k)
			continue
		}
		// Cosnider the keyname
		var keyname SimpleValProfile
		keyname.Profile(k)
		DecideChild(&current, config.OtherKeynames.Decide(&keyname), "OtherKeyname %s:", k)

		// Cosnider the key value
		DecideChild(&current, config.OtherVals.Decide(v), "OtherVals %s", k)
	}
	return current
}

func (config *KeyValConfig) learnI(valPile ValuePile) {
	config.Learn(valPile.(*KeyValPile))
}

// Learn implementation currently is not optimized for a large number of keys
// pile is RO and unchanged - never uses pile internal objects
// Future: When the number of keys grow, Learn may reduce the number of known keys by
// aggregating all known keys which have common low security fingerprint into
// OtherKeynames and OtherVals
func (config *KeyValConfig) Learn(pile *KeyValPile) {
	if pile == nil || len(*pile) == 0 {
		return
	}

	// we should learn a maximum of MAX_KEYS
	// we aim to ensure that these are the MAX_KEYS with highest score
	// we use heuristics to get a rough estimate which are the the keys with highest score

	// learn known keys
	if config.Vals == nil {
		config.Vals = make(map[string]*SimpleValConfig, len(*pile))
	}
	for key, v_pile := range *pile {
		svc, ok := config.Vals[key]
		if ok && svc != nil {
			svc.Learn(v_pile)
			continue
		}

		// new key!
		svc = new(SimpleValConfig)
		svc.Learn(v_pile)
		score := svc.Score()
		//lowestScore := score
		lowestKey := key
		lowestSvc := svc
		index := config.findScoreIndex(score)
		if index >= 0 {
			// The lowest score key is at valScores[0]
			lowestKey = config.valScores[0].key
			lowestSvc = config.Vals[lowestKey]
			// Add svc, key, score to Vals and valScores
			config.setScore(score, key, index)
			config.Vals[key] = svc
		}
		// merge lowestScore and lowestKey with unknown
		if lowestKey == "" {
			// nothing to merge
			continue
		}
		if key != lowestKey {
			delete(config.Vals, lowestKey)
		}
		// add lowestSvc as OtherVals
		config.addSvcToOtherVals(lowestSvc)

		// add lowestKey to OtherKeynames
		config.addKeyToOtherKeynames(lowestKey)
	}
}

// set lowestSvc as OtherVals
func (config *KeyValConfig) addSvcToOtherVals(svc *SimpleValConfig) {
	if config.OtherVals == nil {
		config.OtherVals = svc
	} else {
		// Fuse config.OtherVals
		config.OtherVals.Fuse(svc)
	}
}

// add key to OtherKeynames
func (config *KeyValConfig) addKeyToOtherKeynames(key string) {
	svprofile := SimpleValProfile{}
	svprofile.Profile(key)
	svpile := SimpleValPile{}
	svpile.Add(&svprofile)
	if config.OtherKeynames == nil {
		config.OtherKeynames = &SimpleValConfig{}
	}
	config.OtherKeynames.Learn(&svpile)
}

func (config *KeyValConfig) findScoreIndex(score uint32) int {
	for index := 0; index < MAX_KEYS_LEARNED; index++ {
		if score <= config.valScores[index].score {
			return index - 1
		}
	}
	// score is bigger than all valScores[*].scores
	return MAX_KEYS_LEARNED - 1
}

func (config *KeyValConfig) setScore(score uint32, key string, index int) {
	// move valScores[i+1] to valScores[i] for any i smaller tha index
	for i := 0; i < index; i++ {
		config.valScores[i].score = config.valScores[i+1].score
		config.valScores[i].key = config.valScores[i+1].key
	}
	// set valScores[index] with the new value
	config.valScores[index].score = score
	config.valScores[index].key = key
}

func (config *KeyValConfig) Prepare() {
	for key, v := range config.Vals {
		if v != nil {
			v.Prepare()
			score := v.Score()
			index := config.findScoreIndex(score)
			if index >= 0 {
				// Add svc, key, score to Vals and valScores
				config.setScore(score, key, index)
			}
		}
	}

	if config.OtherKeynames != nil {
		config.OtherKeynames.Prepare()
	}
	if config.OtherVals != nil {
		config.OtherVals.Prepare()
	}
}
