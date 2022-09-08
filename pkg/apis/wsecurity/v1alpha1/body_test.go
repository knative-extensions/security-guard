package v1alpha1

import "testing"

func TestBody_Structured(t *testing.T) {
	arguments := [][]interface{}{
		// array of string
		{[]string{"aaa", "ok"}},
		// array of booleans
		{[]bool{true, false}},
		// array of floats
		{[]float64{1, 2, 5.5}},
		// array of maps
		{[]map[string]string{{"a": "aaa", "b": "ok"}, {"c": "a"}, {}, {"a": "x", "c": "d"}}},
		// array of array
		{[][]string{{"aaa", "ok"}, {"a"}, {}, {"x", "d"}}},

		// map of staff
		{map[string]interface{}{"a": "123abc", "b": float64(12), "c": float64(2.3), "d": true, "e": []float64{1, 2, 43}}},
		// Map of floats
		{map[string][]float64{"a": {float64(1), float64(2), float64(5.5)}}},
		// Map of strings
		{map[string][]string{"ex": {"abc"}}},
		// Map of arrays
		{map[string][]string{"a": {"abc"}}},
		// Map of Maps
		{map[string]map[string]float64{"a": {"a": 1, "b": 2}, "x": {"a": 1, "b": 2}}},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(BodyProfile))
		piles = append(piles, new(BodyPile))
		configs = append(configs, new(BodyConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}

	ValueTests_All(t, profiles, piles, configs, args...)
}

func TestBody_Unstructured(t *testing.T) {
	arguments := [][]string{
		{"licenseID=string&content=string&/paramsXML=string"},
		{"<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<string xmlns=\"http://clearforest.com/\">string</string>"},
		{"[{\"userId\": 1,\"id\": 1,\"title\": \"example title\",\"completed\": false},{\"userId\": 1,\"id\": 2,\"title\": \"another example title\",\"completed\": true},]"},
		{"376512377457"},
		{"***&*&^&^%^&(*&"},
		{"רגע עם דודלי"},
		{"Yada Yada Yada"},
		{"192.168.72.13"},
		{""},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(BodyProfile))
		piles = append(piles, new(BodyPile))
		configs = append(configs, new(BodyConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}

	ValueTests_All(t, profiles, piles, configs, args...)
}
