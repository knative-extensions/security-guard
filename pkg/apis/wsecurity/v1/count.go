package v1

import (
	"bytes"
	"fmt"
)

//////////////////// CountProfile ////////////////

// Exposes ValueProfile interface
type CountProfile uint8

func (profile *CountProfile) DeepCopyValueProfile() ValueProfile {
	return profile
}

func (profile *CountProfile) Profile(args ...interface{}) {
	*profile = CountProfile(args[0].(uint8))
}

func (profile *CountProfile) String(depth int) string {
	return fmt.Sprintf("%d", uint8(*profile))
}

//////////////////// CountPile ////////////////

// Exposes ValuePile interface
type CountPile []uint8

func (pile *CountPile) DeepCopyValuePile() ValuePile {
	return pile
}

func (pile *CountPile) Add(valProfile ValueProfile) {
	profile := *valProfile.(*CountProfile)
	*pile = append(*pile, uint8(profile))
}

func (pile *CountPile) Clear() {
	*pile = nil
}

func (pile *CountPile) Merge(otherValPile ValuePile) {
	otherPile := otherValPile.(*CountPile)
	*pile = append(*pile, *otherPile...)
}

func (pile *CountPile) String(depth int) string {
	return fmt.Sprintf("%v", *pile)
}

//////////////////// CountConfig ////////////////
type countRange struct {
	Min uint8 `json:"min"`
	Max uint8 `json:"max"`
}

func (cRange *countRange) fuseTwoRanges(otherRange *countRange) bool {
	if cRange.Max < otherRange.Min || cRange.Min > otherRange.Max {
		// no overlap - nothing to do!
		return false
	}

	// There is overlap of some sort
	if cRange.Min > otherRange.Min {
		cRange.Min = otherRange.Min
	}
	if cRange.Max < otherRange.Max {
		cRange.Max = otherRange.Max
	}
	return true
}

// Exposes ValueConfig interface
type CountConfig []countRange

func (config *CountConfig) DeepCopyValueConfig() ValueConfig {
	return config
}

func (config *CountConfig) Decide(valProfile ValueProfile) string {
	profile := *valProfile.(*CountProfile)
	if profile == 0 {
		return ""
	}
	// v>0
	if len(*config) == 0 {
		return fmt.Sprintf("Value %d Not Allowed!", profile)
	}

	for j := 0; j < len(*config); j++ {
		if uint8(profile) < (*config)[j].Min {
			break
		}
		if uint8(profile) <= (*config)[j].Max { // found ok interval
			return ""
		}
	}
	return fmt.Sprintf("Counter out of Range: %d", profile)
}

// Learn now offers the simplest single rule support
// Future: Improve Learn
func (config *CountConfig) Learn(valPile ValuePile) {
	pile := valPile.(*CountPile)
	min := uint8(0)
	max := uint8(0)
	if len(*pile) > 0 {
		min = (*pile)[0]
		max = (*pile)[0]
	}
	for _, v := range *pile {
		if min > v {
			min = v
		}
		if max < v {
			max = v
		}
	}
	*config = append(*config, countRange{min, max})
}

// Fuse CountConfig in-place
// The implementation look to opportunistically merge new entries to existing ones
// The implementation does now squash entries even if after the Fuse they may be squashed
// This is done to achieve Fuse in-place
// Future: Improve Fuse - e.g. by keeping extra entries in Range [0,0] and reusing them
//                        instead of adding new entries
func (config *CountConfig) Fuse(otherValConfig ValueConfig) {
	var fused bool

	otherConfig := otherValConfig.(*CountConfig)
	for _, other := range *otherConfig {
		fused = false
		for idx, this := range *config {
			if fused = this.fuseTwoRanges(&other); fused {
				(*config)[idx] = this
				break
			}
		}
		if !fused {
			*config = append(*config, other)
		}
	}
}

func (config CountConfig) String(depth int) string {
	if len(config) == 0 {
		return "null"
	}
	var description bytes.Buffer
	description.WriteString(fmt.Sprintf("[{Min:%d,Max: %d", config[0].Min, config[0].Max))
	for j := 1; j < len(config); j++ {
		description.WriteString(fmt.Sprintf("}, {Min:%d,Max: %d", config[j].Min, config[j].Max))
	}
	description.WriteString("}]")
	return description.String()
}
