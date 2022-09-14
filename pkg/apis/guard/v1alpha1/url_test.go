package v1alpha1

import "testing"

func TestUrl_V1(t *testing.T) {
	arguments := [][]string{
		{"abc/def/file.html"},
		{"CDE"},
		{"abc/234/fil^e.ht%ml"},
		{""},
		{"FKJSDNFKJSHDFKJSDFKJSDKJ"},
		{"$$"},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(UrlProfile))
		piles = append(piles, new(UrlPile))
		configs = append(configs, new(UrlConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}
