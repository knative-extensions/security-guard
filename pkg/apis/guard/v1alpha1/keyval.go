package v1alpha1

import (
	"fmt"
	"strings"
)

//////////////////// KeyValProfile ////////////////

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

func (profile *KeyValProfile) ProfileMapString(keyValMap map[string]string) {
	*profile = nil
	if len(keyValMap) == 0 { // no keys
		*profile = nil
		return
	}
	*profile = make(map[string]*SimpleValProfile, len(keyValMap))
	for k, v := range keyValMap {
		// Profile the concatenated value
		(*profile)[k] = new(SimpleValProfile)
		(*profile)[k].Profile(v)
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
		// Concatenate all strings into one value
		// Appropriate for evaluating []string where order should be also preserved
		val := strings.Join(v, " ")

		// Profile the concatenated value
		(*profile)[k] = new(SimpleValProfile)
		(*profile)[k].Profile(val)
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
}

func (config *KeyValConfig) decideI(valProfile ValueProfile) string {
	return config.Decide(valProfile.(*KeyValProfile))
}

func (config *KeyValConfig) Decide(profile *KeyValProfile) string {
	if profile == nil {
		return ""
	}

	// For each key-val, decide
	for k, v := range *profile {
		// Decide based on a known keys
		if config.Vals != nil && config.Vals[k] != nil {
			if ret := config.Vals[k].Decide(v); ret != "" {
				return fmt.Sprintf("Known Key %s: %s", k, ret)
			}
			continue
		}
		// Decide based on unknown key...
		if config.OtherKeynames == nil || config.OtherVals == nil {
			return fmt.Sprintf("Key %s is not known", k)
		}
		// Cosnider the keyname
		var keyname SimpleValProfile
		keyname.Profile(k)
		if ret := config.OtherKeynames.Decide(&keyname); ret != "" {
			return fmt.Sprintf("Other keyname %s: %s", k, ret)
		}
		// Cosnider the key value
		if ret := config.OtherVals.Decide(v); ret != "" {
			return fmt.Sprintf("Other keyname %s: %s", k, ret)
		}
		continue
	}
	return ""
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
	config.OtherVals = nil
	config.OtherKeynames = nil

	if pile == nil {
		config.Vals = nil
		return
	}

	// learn known keys
	config.Vals = make(map[string]*SimpleValConfig, len(*pile))
	for k, v := range *pile {
		svc := new(SimpleValConfig)
		svc.Learn(v)
		config.Vals[k] = svc
	}
}

func (config *KeyValConfig) fuseI(otherValConfig ValueConfig) {
	config.Fuse(otherValConfig.(*KeyValConfig))
}

// otherConfig is RO and unchanged - never uses otherConfig internal objects
func (config *KeyValConfig) Fuse(otherConfig *KeyValConfig) {
	if otherConfig == nil {
		return
	}
	if config.Vals == nil {
		config.Vals = make(map[string]*SimpleValConfig, len(otherConfig.Vals))
	}
	// fuse known keys
	for k, v := range otherConfig.Vals {
		svc, exists := config.Vals[k]
		if exists {
			svc.Fuse(v)
		} else {
			svc := new(SimpleValConfig)
			svc.Fuse(v)
			config.Vals[k] = svc
		}
	}

	// fuse keynames of unknown keys
	if otherConfig.OtherKeynames != nil {
		if config.OtherKeynames == nil {
			config.OtherKeynames = new(SimpleValConfig)
		}
		config.OtherKeynames.Fuse(otherConfig.OtherKeynames)
	}

	// fuse key values of unknown keys
	if otherConfig.OtherVals != nil {
		if config.OtherVals == nil {
			config.OtherVals = new(SimpleValConfig)
		}
		config.OtherVals.Fuse(otherConfig.OtherVals)
	}
}
