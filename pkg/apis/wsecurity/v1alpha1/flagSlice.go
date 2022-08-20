package v1alpha1

import (
	"fmt"
)

//////////////////// FlagSliceProfile ////////////////

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

func (profile *FlagSliceProfile) ProfileI(args ...interface{}) {
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

func (pile *FlagSlicePile) AddI(valProfile ValueProfile) {
	pile.Add(*valProfile.(*FlagSliceProfile))
}

func (pile *FlagSlicePile) Add(profile FlagSliceProfile) {
	*pile = mergeFlagSlices(*pile, profile)
}

func (pile *FlagSlicePile) Clear() {
	*pile = nil
}

func (pile *FlagSlicePile) MergeI(otherValPile ValuePile) {
	pile.Merge(*otherValPile.(*FlagSlicePile))
}

func (pile *FlagSlicePile) Merge(otherPile FlagSlicePile) {
	*pile = mergeFlagSlices(*pile, otherPile)
}

//////////////////// FlagSliceConfig ////////////////

// Exposes ValueConfig interface
type FlagSliceConfig []uint32

func (config *FlagSliceConfig) DecideI(valProfile ValueProfile) string {
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

func (config *FlagSliceConfig) LearnI(valPile ValuePile) {
	config.Learn(*valPile.(*FlagSlicePile))
}

func (config *FlagSliceConfig) Learn(pile FlagSlicePile) {
	*config = FlagSliceConfig(pile)
}

func (config *FlagSliceConfig) FuseI(otherValConfig ValueConfig) {
	config.Fuse(*otherValConfig.(*FlagSliceConfig))
}

func (config *FlagSliceConfig) Fuse(otherConfig FlagSliceConfig) {
	*config = mergeFlagSlices(*config, otherConfig)
}
