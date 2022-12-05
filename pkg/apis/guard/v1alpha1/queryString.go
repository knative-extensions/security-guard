package v1alpha1

import (
	"net/url"
)

//////////////////// QueryProfile ////////////////

// Exposes ValueProfile interface
type QueryProfile struct {
	Kv KeyValProfile `json:"kv"`
}

func (profile *QueryProfile) profileI(args ...interface{}) {
	profile.Profile(args[0].(url.Values))
}

func (profile *QueryProfile) Profile(values url.Values) {
	profile.Kv.ProfileMapStringSlice(values)
}

//////////////////// QueryPile ////////////////

// Exposes ValuePile interface
type QueryPile struct {
	Kv *KeyValPile `json:"kv"`
}

func (pile *QueryPile) addI(valProfile ValueProfile) {
	pile.Add(valProfile.(*QueryProfile))
}

func (pile *QueryPile) Add(profile *QueryProfile) {
	if pile.Kv == nil {
		pile.Kv = new(KeyValPile)
	}
	pile.Kv.Add(&profile.Kv)
}

func (pile *QueryPile) mergeI(otherValPile ValuePile) {
	pile.Merge(otherValPile.(*QueryPile))
}

func (pile *QueryPile) Merge(otherPile *QueryPile) {
	if pile.Kv == nil {
		pile.Kv = new(KeyValPile)
	}
	pile.Kv.Merge(otherPile.Kv)
}

func (pile *QueryPile) Clear() {
	pile.Kv = new(KeyValPile)
	if pile.Kv != nil {
		pile.Kv.Clear()
	}
}

//////////////////// QueryConfig ////////////////

// Exposes ValueConfig interface
type QueryConfig struct {
	Kv KeyValConfig `json:"kv"`
}

func (config *QueryConfig) decideI(valProfile ValueProfile) *Decision {
	return config.Decide(valProfile.(*QueryProfile))
}

func (config *QueryConfig) Decide(profile *QueryProfile) *Decision {
	var current *Decision
	DecideChild(&current, config.Kv.Decide(&profile.Kv), "KeyVal")
	return current
}

func (config *QueryConfig) learnI(valPile ValuePile) {
	config.Learn(valPile.(*QueryPile))
}

func (config *QueryConfig) Learn(pile *QueryPile) {
	config.Kv.Learn(pile.Kv)
}

func (config *QueryConfig) fuseI(otherValConfig ValueConfig) {
	config.Fuse(otherValConfig.(*QueryConfig))
}

func (config *QueryConfig) Fuse(otherConfig *QueryConfig) {
	config.Kv.Fuse(&otherConfig.Kv)
}
