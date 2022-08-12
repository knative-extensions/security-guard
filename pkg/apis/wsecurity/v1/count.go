package v1

import (
	"bytes"
	"fmt"
	"strings"
)

//////////////////// CountProfile ////////////////

// Exposes ValueProfile interface
type CountProfile struct {
	val uint8
}

func (profile *CountProfile) DeepCopyValueProfile() ValueProfile {
	return profile
}

func (profile *CountProfile) Profile(args ...interface{}) {
	profile.val = args[0].(uint8)
}

func (profile *CountProfile) String(depth int) string {
	var description bytes.Buffer
	shift := strings.Repeat("  ", depth)
	description.WriteString("{\n")
	description.WriteString(shift)
	description.WriteString(fmt.Sprintf("  Val: %d", profile.val))
	description.WriteString(shift)
	description.WriteString("}\n")
	return description.String()
}

//////////////////// CountPile ////////////////

// Exposes ValuePile interface
type CountPile struct {
	vals []uint8
}

func (pile *CountPile) DeepCopyValuePile() ValuePile {
	return pile
}

func (pile *CountPile) Add(valProfile ValueProfile) {
	profile := valProfile.(*CountProfile)
	pile.vals = append(pile.vals, profile.val)
}

func (pile *CountPile) Clear() {
	pile = nil
}

func (pile *CountPile) Merge(otherValPile ValuePile) {
	otherPile := otherValPile.(*CountPile)
	pile.vals = append(pile.vals, otherPile.vals...)
}

func (pile *CountPile) String(depth int) string {
	var description bytes.Buffer
	shift := strings.Repeat("  ", depth)
	description.WriteString("{\n")
	description.WriteString(shift)
	description.WriteString(fmt.Sprintf("  Vals: %v", *pile))
	description.WriteString(shift)
	description.WriteString("}\n")
	return description.String()
}

//////////////////// CountConfig ////////////////

type countMinMax struct {
	Min uint8 `json:"min"`
	Max uint8 `json:"max"`
}

func (minMax *countMinMax) countMerge(otherMinMax *countMinMax) {
	if minMax.Min > otherMinMax.Min {
		minMax.Min = otherMinMax.Min
	}
	if minMax.Max < otherMinMax.Max {
		minMax.Max = otherMinMax.Max
	}
}

// Exposes ValueConfig interface
type CountConfig []countMinMax

func (config *CountConfig) DeepCopyValueConfig() ValueConfig {
	return config
}

func (config *CountConfig) Decide(valProfile ValueProfile) string {
	profile := (*valProfile.(*CountProfile))
	if profile.val == 0 {
		return ""
	}
	// v>0
	if len(*config) == 0 {
		return fmt.Sprintf("Value %d Not Allowed!", profile.val)
	}

	for j := 0; j < len(*config); j++ {
		if profile.val < (*config)[j].Min {
			break
		}
		if profile.val <= (*config)[j].Max { // found ok interval
			return ""
		}
	}
	return fmt.Sprintf("Counter out of Range: %d", profile.val)
}

func (config *CountConfig) Learn(valPile ValuePile) {
	pile := valPile.(*CountPile)

	min := uint8(0)
	max := uint8(0)
	if len(pile.vals) >= 0 {
		min = pile.vals[0]
		max = pile.vals[0]
	}
	for _, v := range pile.vals {
		if min > v {
			min = v
		}
		if max < v {
			max = v
		}
	}
	*config = append(*config, countMinMax{min, max})
}

func (config *CountConfig) Merge(otherValConfig ValueConfig) {
	var found bool
	otherConfig := otherValConfig.(*CountConfig)
	for _, other := range *otherConfig {
		found = false
		for idx, this := range *config {
			if this.Min < other.Min {
				// Does "this" include "other"?
				if this.Max > other.Max {
					// "this" include "other"!
					found = true
					break
				}
				// this.Max < other.Max  - "this" does not include "other"!
				if this.Max >= other.Min {
					// "this" overlap with "other" - lets merge them!
					this.countMerge(&other)
					(*config)[idx] = this
					found = true
					break
				}
				// this.Max < other.Min - no overlap - nothing to do!
				continue
			}
			// this.Min >= other.Min
			if this.Min > other.Max {
				// no overlap - nothing to do!
				continue
			}
			// Does "other" include "this"?
			if this.Max <= other.Max {
				// "other" include "this"!
				// swap them
				this.Min = other.Min
				this.Max = other.Max
				(*config)[idx] = this
				found = true
				break
			}
			// "this" overlap with "other" - lets merge them!
			this.countMerge(&other)
			(*config)[idx] = this
			found = true
			break
		}
		if !found {
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
