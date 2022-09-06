package v1alpha1

import (
	"fmt"
	"mime"
)

//////////////////// MediaTypeProfile ////////////////

// Exposes ValueProfile interface
// TypeToken include rfc7231 type "/" subtype
type MediaTypeProfile struct {
	TypeTokens SetProfile    `json:"type"`   // "text/html"
	Params     KeyValProfile `json:"params"` // {"charset": "utf-8"}
}

func (profile *MediaTypeProfile) profileI(args ...interface{}) {
	profile.Profile(args[0].(string))
}

func (profile *MediaTypeProfile) Profile(str string) {
	if mediaType, params, err := mime.ParseMediaType(str); err == nil && mediaType != "" {
		profile.TypeTokens.ProfileString(mediaType)
		profile.Params.ProfileMapString(params)
		return
	}
	// For clients that fail to send media type
	profile.TypeTokens.ProfileString("none")
	profile.Params.ProfileMapString(nil)
}

//////////////////// MediaTypePile ////////////////

// Exposes ValuePile interface
type MediaTypePile struct {
	TypeTokens SetPile    `json:"type"`
	Params     KeyValPile `json:"params"`
}

func (pile *MediaTypePile) addI(valProfile ValueProfile) {
	pile.Add(valProfile.(*MediaTypeProfile))
}

func (pile *MediaTypePile) Add(profile *MediaTypeProfile) {
	pile.TypeTokens.Add(&profile.TypeTokens)
	pile.Params.Add(&profile.Params)
}

func (pile *MediaTypePile) mergeI(otherValPile ValuePile) {
	pile.Merge(otherValPile.(*MediaTypePile))
}

func (pile *MediaTypePile) Merge(otherPile *MediaTypePile) {
	pile.TypeTokens.Merge(&otherPile.TypeTokens)
	pile.Params.Merge(&otherPile.Params)
}

func (pile *MediaTypePile) Clear() {

	pile.TypeTokens.Clear()
	pile.Params.Clear()
}

//////////////////// MediaTypeConfig ////////////////

// Exposes ValueConfig interface
type MediaTypeConfig struct {
	TypeTokens SetConfig    `json:"type"`
	Params     KeyValConfig `json:"params"`
}

func (config *MediaTypeConfig) decideI(valProfile ValueProfile) string {
	return config.Decide(valProfile.(*MediaTypeProfile))
}

func (config *MediaTypeConfig) Decide(profile *MediaTypeProfile) string {
	if str := config.TypeTokens.Decide(&profile.TypeTokens); str != "" {
		return fmt.Sprintf("Type: %s", str)
	}
	if str := config.Params.Decide(&profile.Params); str != "" {
		return fmt.Sprintf("Params: %s", str)
	}
	return ""
}

func (config *MediaTypeConfig) learnI(valPile ValuePile) {
	config.Learn(valPile.(*MediaTypePile))
}

func (config *MediaTypeConfig) Learn(pile *MediaTypePile) {
	config.TypeTokens.Learn(&pile.TypeTokens)
	config.Params.Learn(&pile.Params)
}

func (config *MediaTypeConfig) fuseI(otherValConfig ValueConfig) {
	config.Fuse(otherValConfig.(*MediaTypeConfig))
}

func (config *MediaTypeConfig) Fuse(otherConfig *MediaTypeConfig) {
	config.TypeTokens.Fuse(&otherConfig.TypeTokens)
	config.Params.Fuse(&otherConfig.Params)
}
