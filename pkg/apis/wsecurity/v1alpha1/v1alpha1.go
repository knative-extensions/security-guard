package v1alpha1

// A Profile describing the Value
type ValueProfile interface {
	// ProfileI the data provided in args
	ProfileI(args ...interface{})
}

// A Pile accumulating information from zero or more Values
type ValuePile interface {
	// AddI one more profile to pile
	// Profile should not be used after it is added to a pile
	// Pile may absorb some or the profile internal structures
	AddI(profile ValueProfile)

	// MergeI otherPile to this pile
	// otherPile should not be used after it is merged to a pile
	// Pile may absorb some or the otherPile internal structures
	MergeI(otherPile ValuePile)

	// Clear the pile from all profiles and free any memory held by pile
	Clear()
}

// A Config defining what Value should adhere to
type ValueConfig interface {
	// LearnI config from a pile - destroy any prior state of Config
	// pile should not be used after it is Learned by config
	// Config may absorb some or the pile internal structures
	LearnI(pile ValuePile)

	// FuseI otherConfig to this config
	// otherConfig should not be used after it is fused to a config
	// Config may absorb some or the otherConfig internal structures
	FuseI(otherConfig ValueConfig)

	// DecideI if profile meets config
	// Return empty string if profile meets config
	// Otherwise return a report of one or more issues found
	// (does not guarantee that all issues will be reported)
	// Profile is unchanged and unaffected by DecideI and can be used again
	DecideI(profile ValueProfile) string
}
