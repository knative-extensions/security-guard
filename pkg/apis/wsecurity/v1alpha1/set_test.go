package v1alpha1

import "testing"

func TestSet_STRING(t *testing.T) {
	arguments := [][]string{
		{"ABC"},
		{"CDE"},
		{"123"},
		{""},
		{"FKJSDNFKJSHDFKJSDFKJSDKJ"},
		{"$$"},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(SetProfile))
		piles = append(piles, new(SetPile))
		configs = append(configs, new(SetConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}

func TestSet_STRINGSLICE(t *testing.T) {
	arguments := [][][]string{
		{{"ABC"}},
		{{"CDE", "XXY"}},
		{{"123"}},
		{{""}},
		{{"FKJSDNFKJS", "HDFK", "JSDFKJSDKJ"}},
		{{"$$"}},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(SetProfile))
		piles = append(piles, new(SetPile))
		configs = append(configs, new(SetConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}
