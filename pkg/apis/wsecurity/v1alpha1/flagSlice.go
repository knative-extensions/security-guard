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

func (profile *FlagSliceProfile) Profile(args ...interface{}) {
	if vals := args[0].([]uint32); len(vals) > 0 {
		*profile = vals
	}
}

//////////////////// FlagSlicePile ////////////////

// Exposes ValuePile interface
type FlagSlicePile []uint32

func (pile *FlagSlicePile) Add(valProfile ValueProfile) {
	profile := valProfile.(*FlagSliceProfile)
	*pile = mergeFlagSlices(*pile, *profile)
}

func (pile *FlagSlicePile) Clear() {
	*pile = nil
}

func (pile *FlagSlicePile) Merge(otherValPile ValuePile) {
	otherPile := otherValPile.(*FlagSlicePile)
	*pile = mergeFlagSlices(*pile, *otherPile)
}

//////////////////// FlagSliceConfig ////////////////

// Exposes ValueConfig interface
type FlagSliceConfig []uint32

func (config *FlagSliceConfig) Decide(valProfile ValueProfile) string {
	profile := (*valProfile.(*FlagSliceProfile))

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

func (config *FlagSliceConfig) Learn(valPile ValuePile) {
	pile := valPile.(*FlagSlicePile)
	*config = FlagSliceConfig(*pile)
}

func (config *FlagSliceConfig) Fuse(otherValConfig ValueConfig) {
	otherConfig := otherValConfig.(*FlagSliceConfig)
	*config = mergeFlagSlices(*config, *otherConfig)
}
