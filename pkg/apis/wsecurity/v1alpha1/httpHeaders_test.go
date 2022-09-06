package v1alpha1

import (
	"net/http"
	"testing"
)

func TestHeaders_V1(t *testing.T) {
	header := http.Header{}
	header.Add("a", "x")
	header2 := http.Header{}
	header2.Add("b", "x")
	arguments := [][]http.Header{
		{header},
		{header2},
		{header2},
		{header},
		{header},
		{header},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(HeadersProfile))
		piles = append(piles, new(HeadersPile))
		configs = append(configs, new(HeadersConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}
