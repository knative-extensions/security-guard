/*
Copyright 2022 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import "testing"

func TestStructuredProfiler_V1(t *testing.T) {
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
		profiles = append(profiles, new(StructuredProfile))
		piles = append(piles, new(StructuredPile))
		configs = append(configs, new(StructuredConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}
