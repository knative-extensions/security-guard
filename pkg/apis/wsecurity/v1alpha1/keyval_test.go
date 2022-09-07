package v1alpha1

import (
	"testing"
)

func TestSimpleVal_V1(t *testing.T) {
	arguments := [][]map[string][]string{
		{{"a": {"abc"}}},
		{{"a": {"123abc"}, "b": {"12"}}},
		{{"a": {"abcd"}}},
		{{"ex": {"abc"}}},
		{{"dfods": {"sdf;jsdfojssdfsdfsdlfosjf2390rj09uf"}}},
		{{"a*(Y((H(H&&^%&": {"^&U%&&^GTT*YHOIJMOI"}}},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(KeyValProfile))
		piles = append(piles, new(KeyValPile))
		configs = append(configs, new(KeyValConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}
