package v1alpha1

import (
	"net/http"
)

//////////////////// HeadersProfile ////////////////

// Exposes ValueProfile interface
type HeadersProfile struct {
	Kv KeyValProfile `json:"kv"`
}

func (profile *HeadersProfile) profileI(args ...interface{}) {
	profile.Profile(args[0].(http.Header))
}

func (profile *HeadersProfile) Profile(headers http.Header) {
	profile.Kv.ProfileMapStringSlice(headers)
}

//////////////////// HeadersPile ////////////////

// Exposes ValuePile interface
type HeadersPile struct {
	Kv *KeyValPile `json:"kv"`
}

func (pile *HeadersPile) addI(valProfile ValueProfile) {
	pile.Add(valProfile.(*HeadersProfile))
}

func (pile *HeadersPile) Add(profile *HeadersProfile) {
	if pile.Kv == nil {
		pile.Kv = new(KeyValPile)
	}
	pile.Kv.Add(&profile.Kv)
}

func (pile *HeadersPile) mergeI(otherValPile ValuePile) {
	pile.Merge(otherValPile.(*HeadersPile))
}

func (pile *HeadersPile) Merge(otherPile *HeadersPile) {
	if pile.Kv == nil {
		pile.Kv = new(KeyValPile)
	}
	pile.Kv.Merge(otherPile.Kv)
}

func (pile *HeadersPile) Clear() {
	pile.Kv = new(KeyValPile)
	if pile.Kv != nil {
		pile.Kv.Clear()
	}
}

//////////////////// HeadersConfig ////////////////

// Exposes ValueConfig interface
type HeadersConfig struct {
	Kv KeyValConfig `json:"kv"`
}

func (config *HeadersConfig) decideI(valProfile ValueProfile) *Decision {
	return config.Decide(valProfile.(*HeadersProfile))
}

func (config *HeadersConfig) Decide(profile *HeadersProfile) *Decision {
	var current *Decision
	DecideChild(&current, config.Kv.Decide(&profile.Kv), "KeyVal")
	return current
}

func (config *HeadersConfig) learnI(valPile ValuePile) {
	config.Learn(valPile.(*HeadersPile))
}

func (config *HeadersConfig) Learn(pile *HeadersPile) {
	config.Kv.Learn(pile.Kv)
}

func (config *HeadersConfig) fuseI(otherValConfig ValueConfig) {
	config.Fuse(otherValConfig.(*HeadersConfig))
}

func (config *HeadersConfig) Fuse(otherConfig *HeadersConfig) {
	config.Kv.Fuse(&otherConfig.Kv)
}
