package v1

// A Profile describing the Value
type ValueProfile interface {
	// Profile the data provided in args
	Profile(args ...interface{})

	// Return a multiline string ready for logging at indentation depth
	String(depth int) string
}

// A Pile accumulating information from zero or more Values
type ValuePile interface {
	// Add one more profile to pile
	// Profile should not be used after it is added to a pile
	// Pile may absorb some or the profile internal structures
	Add(profile ValueProfile)

	// Merge otherPile to this pile
	// otherPile should not be used after it is merged to a pile
	// Pile may absorb some or the otherPile internal structures
	Merge(otherPile ValuePile)

	// Clear the pile from all profiles and free any memory held by pile
	Clear()

	// Return a multiline string ready for logging at indentation depth
	String(depth int) string
}

// A Config defining what Value should adhere to
type ValueConfig interface {
	// Learn config from a pile - destroy any prior state of Config
	// pile should not be used after it is Learned by config
	// Config may absorb some or the pile internal structures
	Learn(pile ValuePile)

	// Fuse otherConfig to this config
	// otherConfig should not be used after it is fused to a config
	// Config may absorb some or the otherConfig internal structures
	Fuse(otherConfig ValueConfig)

	// Decide if profile meets config
	// Return empty string if profile meets config
	// Otherwise return a report of one or more issues found
	// (does not guarantee that all issues will be reported)
	// Profile is unchanged and unaffected by Decide and can be used again
	Decide(profile ValueProfile) string

	// Return a multiline string ready for logging at indentation depth
	String(depth int) string
}
