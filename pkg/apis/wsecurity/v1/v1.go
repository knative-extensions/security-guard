package v1

// A Profile describing the Value
type ValueProfile interface {
	// Profile actual data provided in elem
	Profile(args ...interface{})

	// Return a multiline string ready for logging at indentation depth
	String(depth int) string

	// Deep Copy - required to enable Code Generation
	DeepCopyValueProfile() ValueProfile
}

// A Pile accumulating information from zero or more Values
type ValuePile interface {
	// Add one more profile to pile
	Add(profile ValueProfile)

	// Merge otherPile to this pile
	Merge(otherPile ValuePile)

	// Clear the pile from all profiles
	Clear()

	// Return a multiline string ready for logging at indentation depth
	String(depth int) string

	// Deep Copy - required to enable Code Generation
	DeepCopyValuePile() ValuePile
}

// A Config defining what Value should adhere to
type ValueConfig interface {
	// Merge otherConfig to this config
	Merge(otherConfig ValueConfig)

	// Learn config from a pile
	Learn(pile ValuePile)

	// Decide if profile meets config
	// Return empty string if profile meets config
	// Otherwise return a report of one or more issues found
	// (does not guarantee that all issues will be reported)
	Decide(profile ValueProfile) string

	// Return a multiline string ready for logging at indentation depth
	String(depth int) string

	// Deep Copy - required to enable Code Generation
	DeepCopyValueConfig() ValueConfig
}
