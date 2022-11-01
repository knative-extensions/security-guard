package v1alpha1

import (
	"fmt"
)

//////////////////// FlagSliceProfile ////////////////

// extra is RO and no internal objects are used
func mergeFlagSlices(base []uint32, extra []uint32) []uint32 {
	if missing := len(extra) - len(base); missing > 0 {
		// Dynamically allocate as many blockElements as needed
		base = append(base, make([]uint32, missing)...)
	}

	for i, v := range extra {
		base[i] = base[i] | v
	}
	return base
}

// Exposes ValueProfile interface
type FlagSliceProfile []uint32

func (profile *FlagSliceProfile) profileI(args ...interface{}) {
	profile.Profile(args[0].([]uint32))
}

func (profile *FlagSliceProfile) Profile(vals []uint32) {
	if len(vals) > 0 {
		*profile = vals
	}
}

//////////////////// FlagSlicePile ////////////////

// Exposes ValuePile interface
type FlagSlicePile []uint32

func (pile *FlagSlicePile) addI(valProfile ValueProfile) {
	pile.Add(*valProfile.(*FlagSliceProfile))
}

// profile is RO and unchanged - never uses profile internal objects
func (pile *FlagSlicePile) Add(profile FlagSliceProfile) {
	*pile = mergeFlagSlices(*pile, profile)
}

func (pile *FlagSlicePile) Clear() {
	*pile = nil
}

func (pile *FlagSlicePile) mergeI(otherValPile ValuePile) {
	pile.Merge(*otherValPile.(*FlagSlicePile))
}

// otherPile is RO and unchanged - never uses otherPile internal objects
func (pile *FlagSlicePile) Merge(otherPile FlagSlicePile) {
	*pile = mergeFlagSlices(*pile, otherPile)
}

//////////////////// FlagSliceConfig ////////////////

// Exposes ValueConfig interface
type FlagSliceConfig []uint32

func (config *FlagSliceConfig) decideI(valProfile ValueProfile) string {
	return config.Decide(*valProfile.(*FlagSliceProfile))
}

func (config *FlagSliceConfig) Decide(profile FlagSliceProfile) string {
	for i, v := range profile {
		if v == 0 {
			continue
		}
		if i < len(*config) && (v & ^(*config)[i]) == 0 {
			continue
		}
		return fmt.Sprintf("Unexpected Flags in FlagSlice %x on Element %d", v, i)
	}
	return ""
}

func (config *FlagSliceConfig) learnI(valPile ValuePile) {
	config.Learn(*valPile.(*FlagSlicePile))
}

// otherPile is RO and unchanged - never uses otherPile internal objects
func (config *FlagSliceConfig) Learn(pile FlagSlicePile) {
	*config = make(FlagSliceConfig, len(pile))
	copy(*config, pile)
}

func (config *FlagSliceConfig) fuseI(otherValConfig ValueConfig) {
	config.Fuse(*otherValConfig.(*FlagSliceConfig))
}

// otherConfig is RO and unchanged - never uses otherConfig internal objects
func (config *FlagSliceConfig) Fuse(otherConfig FlagSliceConfig) {
	*config = mergeFlagSlices(*config, otherConfig)
}
