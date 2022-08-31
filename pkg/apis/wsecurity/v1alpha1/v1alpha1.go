package v1alpha1

// A Profile describing the Value
type ValueProfile interface {
	// profileI the data provided in args
	profileI(args ...interface{})
}

// A Pile accumulating information from zero or more Values
type ValuePile interface {
	// addI one more profile to pile
	// Profile should not be used after it is added to a pile
	// Pile may absorb some or the profile internal structures
	addI(profile ValueProfile)

	// mergeI otherPile to this pile
	// otherPile should not be used after it is merged to a pile
	// Pile may absorb some or the otherPile internal structures
	mergeI(otherPile ValuePile)

	// Clear the pile from all profiles and free any memory held by pile
	Clear()
}

// A Config defining what Value should adhere to
type ValueConfig interface {
	// learnI config from a pile - destroy any prior state of Config
	// pile should not be used after it is Learned by config
	// Config may absorb some or the pile internal structures
	learnI(pile ValuePile)

	// fuseI otherConfig to this config
	// otherConfig should not be used after it is fused to a config
	// Config may absorb some or the otherConfig internal structures
	fuseI(otherConfig ValueConfig)

	// decideI if profile meets config
	// Return empty string if profile meets config
	// Otherwise return a report of one or more issues found
	// (does not guarantee that all issues will be reported)
	// Profile is unchanged and unaffected by decideI and can be used again
	decideI(profile ValueProfile) string
}
