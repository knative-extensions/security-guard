package v1alpha1

import (
	"testing"
)

func TestCount_V1(t *testing.T) {
	arguments := [][]uint8{
		{4},
		{5},
		{1},
		{254},
		{255},
		{0},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(CountProfile))
		piles = append(piles, new(CountPile))
		configs = append(configs, new(CountConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}

	ValueTests_All(t, profiles, piles, configs, args...)
}
