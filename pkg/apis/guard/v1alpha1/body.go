package v1alpha1

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

//////////////////// BodyPile ////////////////

// Exposes ValuePile interface
type BodyPile struct {
	Unstructured *SimpleValPile  `json:"unstructured"`
	Structured   *StructuredPile `json:"structured"`
}

func (pile *BodyPile) addI(valProfile ValueProfile) {
	pile.Add(valProfile.(*BodyProfile))
}

// profile is RO and unchanged - never uses profile internal objects
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

func (pile *BodyPile) Clear() {
	pile.Structured = nil
	pile.Unstructured = nil
}

func (pile *BodyPile) mergeI(otherValPile ValuePile) {
	pile.Merge(otherValPile.(*BodyPile))
}

// otherPile is RO and unchanged - never uses otherPile internal objects
func (pile *BodyPile) Merge(otherPile *BodyPile) {
	if otherPile.Structured != nil {
		if pile.Structured == nil {
			pile.Structured = new(StructuredPile)
		}
		pile.Structured.Merge(otherPile.Structured)
	}
	if otherPile.Unstructured != nil {
		if pile.Unstructured == nil {
			pile.Unstructured = new(SimpleValPile)
		}
		pile.Unstructured.Merge(otherPile.Unstructured)
	}
}

//////////////////// BodyConfig ////////////////

// Exposes ValueConfig interface
type BodyConfig struct {
	Unstructured *SimpleValConfig  `json:"unstructured"`
	Structured   *StructuredConfig `json:"structured"`
}

func (config *BodyConfig) decideI(valProfile ValueProfile) *Decision {
	return config.Decide(valProfile.(*BodyProfile))
}

func (config *BodyConfig) Decide(profile *BodyProfile) *Decision {
	var current *Decision

	if profile.Structured != nil {
		if config.Structured != nil {
			DecideChild(&current, config.Structured.Decide(profile.Structured), "Body")
		} else {
			DecideInner(&current, 1, "Structured Body not allowed")
		}
	}
	if profile.Unstructured != nil {
		if config.Unstructured != nil {
			DecideChild(&current, config.Unstructured.Decide(profile.Unstructured), "Body")
		} else {
			DecideInner(&current, 1, "Unstructured Body not allowed")
		}
	}
	return current
}

func (config *BodyConfig) learnI(valPile ValuePile) {
	config.Learn(valPile.(*BodyPile))
}

// pile is RO and unchanged - never uses pile internal objects
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

func (config *BodyConfig) fuseI(otherValConfig ValueConfig) {
	config.Fuse(otherValConfig.(*BodyConfig))
}

// otherConfig is RO and unchanged - never uses otherConfig internal objects
func (config *BodyConfig) Fuse(otherConfig *BodyConfig) {
	if otherConfig.Structured != nil {
		if config.Structured == nil {
			config.Structured = new(StructuredConfig)
		}
		config.Structured.Fuse(otherConfig.Structured)
	}
	if otherConfig.Unstructured != nil {
		if config.Unstructured == nil {
			config.Unstructured = new(SimpleValConfig)
		}
		config.Unstructured.Fuse(otherConfig.Unstructured)
	}
}
