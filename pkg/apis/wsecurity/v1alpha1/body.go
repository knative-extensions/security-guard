package v1alpha1

import "fmt"

//////////////////// BodyProfile ////////////////

// Exposes ValueProfile interface
type BodyProfile struct {
	Unstructured *SimpleValProfile  `json:"unstructured"`
	Structured   *StructuredProfile `json:"structured"`
}

func (profile *BodyProfile) profileI(args ...interface{}) {
	switch v := args[0].(type) {
	case string:
		profile.ProfileUnstructured(v)
	default:
		profile.ProfileStructured(v)
	}
}

func (profile *BodyProfile) ProfileUnstructured(str string) {
	profile.Structured = nil
	profile.Unstructured = new(SimpleValProfile)
	profile.Unstructured.Profile(str)
}

func (profile *BodyProfile) ProfileStructured(data interface{}) {
	profile.Unstructured = nil
	profile.Structured = new(StructuredProfile)
	profile.Structured.Profile(data)
}

/*
// Future: Missing implementation
func (profile *BodyProfile) String(depth int) string {
	return "Missing Implementation"
}
*/
//////////////////// BodyPile ////////////////

// Exposes ValuePile interface
type BodyPile struct {
	Unstructured *SimpleValPile  `json:"unstructured"`
	Structured   *StructuredPile `json:"structured"`
}

// Future: Missing implementation
func (pile *BodyPile) addI(valProfile ValueProfile) {
	pile.Add(valProfile.(*BodyProfile))
}

// Future: TBD - what to do when one is nil and the other is not
func (pile *BodyPile) Add(profile *BodyProfile) {
	if profile.Structured != nil {
		if pile.Structured == nil {
			pile.Structured = new(StructuredPile)
		}
		pile.Structured.Add(profile.Structured)
	}
	if profile.Unstructured != nil {
		if pile.Unstructured == nil {
			pile.Unstructured = new(SimpleValPile)
		}
		pile.Unstructured.Add(profile.Unstructured)
	}
}

// Future: Missing implementation
func (pile *BodyPile) Clear() {
	pile.Structured = nil
	pile.Unstructured = nil
}

// Future: Missing implementation
func (pile *BodyPile) mergeI(otherValPile ValuePile) {
	pile.Merge(otherValPile.(*BodyPile))
}

// Future: TBD - what to do when one is nil and the other is not
func (pile *BodyPile) Merge(otherPile *BodyPile) {
	if otherPile.Structured != nil {
		if pile.Structured == nil {
			pile.Structured = otherPile.Structured
		} else {
			pile.Structured.Merge(otherPile.Structured)
		}
	}
	if otherPile.Unstructured != nil {
		if pile.Unstructured == nil {
			pile.Unstructured = otherPile.Unstructured
		} else {
			pile.Unstructured.Merge(otherPile.Unstructured)
		}

	}
}

// Future: Missing implementation
//func (pile *BodyPile) String(depth int) string {
//	return "Missing Implementation"
//}

//////////////////// BodyConfig ////////////////

// Exposes ValueConfig interface
type BodyConfig struct {
	Unstructured *SimpleValConfig  `json:"unstructured"`
	Structured   *StructuredConfig `json:"structured"`
}

func (config *BodyConfig) decideI(valProfile ValueProfile) string {
	return config.Decide(valProfile.(*BodyProfile))
}

func (config *BodyConfig) Decide(profile *BodyProfile) string {
	if profile.Structured != nil {
		if config.Structured != nil {
			str := config.Structured.Decide(profile.Structured)
			if str != "" {
				return fmt.Sprintf("Body %s", str)
			}
		} else {
			return "Structured Body not allowed"
		}
	}
	if profile.Unstructured != nil {
		if config.Unstructured != nil {
			str := config.Unstructured.Decide(profile.Unstructured)
			if str != "" {
				return fmt.Sprintf("Body %s", str)
			}
		} else {
			return "Unstructured Body not allowed"
		}
	}
	return ""
}

// Future: Missing implementation
func (config *BodyConfig) learnI(valPile ValuePile) {
	config.Learn(valPile.(*BodyPile))
}

// Future: Missing implementation
func (config *BodyConfig) Learn(pile *BodyPile) {
	if pile.Structured != nil {
		if config.Structured == nil {
			config.Structured = new(StructuredConfig)
		}
		config.Structured.Learn(pile.Structured)
	}
	if pile.Unstructured != nil {
		if config.Unstructured == nil {
			config.Unstructured = new(SimpleValConfig)
		}
		config.Unstructured.Learn(pile.Unstructured)
	}
}

// Future: Missing implementation
func (config *BodyConfig) fuseI(otherValConfig ValueConfig) {
	config.Fuse(otherValConfig.(*BodyConfig))
}

// Future: Missing implementation
func (config *BodyConfig) Fuse(otherConfig *BodyConfig) {
	if otherConfig.Structured != nil {
		if config.Structured == nil {
			config.Structured = otherConfig.Structured
		} else {
			config.Structured.Fuse(otherConfig.Structured)
		}
	}
	if otherConfig.Unstructured != nil {
		if config.Unstructured == nil {
			config.Unstructured = otherConfig.Unstructured
		} else {
			config.Unstructured.Fuse(otherConfig.Unstructured)
		}
	}
}
