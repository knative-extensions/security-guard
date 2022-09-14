package v1alpha1

import "testing"

func TestAsciiFlags_V1(t *testing.T) {
	arguments := [][]uint32{
		{4},
		{5},
		{2},
		{43756},
		{0},
		{254},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(AsciiFlagsProfile))
		piles = append(piles, new(AsciiFlagsPile))
		configs = append(configs, new(AsciiFlagsConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}
