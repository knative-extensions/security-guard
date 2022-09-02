package v1alpha1

import "testing"

func TestFlagSlice_V1(t *testing.T) {
	arguments := [][][]uint32{
		{{1, 4, 3, 7, 43756, 22}},
		{{1, 5, 0, 7, 533, 22}},
		{{0, 0, 66, 7, 44, 22}},
		{{4, 3, 342, 7, 3533333, 22}},
		{{1, 4, 3, 7, 0, 22}},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(FlagSliceProfile))
		piles = append(piles, new(FlagSlicePile))
		configs = append(configs, new(FlagSliceConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}
