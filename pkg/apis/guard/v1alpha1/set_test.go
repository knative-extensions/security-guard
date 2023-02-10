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

import (
	"encoding/json"
	"testing"
)

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

func TestSet_Duplicates(t *testing.T) {
	var profile1 SetProfile
	var profile2 SetProfile
	var pile1 SetPile
	var pile2 SetPile
	var config SetConfig
	var config_load SetConfig

	profile1.ProfileString("X")
	profile1.ProfileString("X")
	profile1.ProfileString("X")
	profile1.ProfileString("X")
	profile2.ProfileString("X")
	profile2.ProfileString("X")
	pile1.Add(&profile1)
	pile2.Add(&profile2)
	pile2.Merge(&pile1)
	if len(pile2.List) != 1 {
		t.Errorf("SetPile.Merge() expected 1 items found %d", len(pile2.List))
	}
	config.Learn(&pile2)
	if len(config.List) != 1 {
		t.Errorf("SetPile.Learn() expected 1 item found %d", len(config.List))
	}
	config.Learn(&pile2)
	if len(config.List) != 1 {
		t.Errorf("SetPile.Learn() expected 1 item found %d", len(config.List))
	}
	j, _ := json.Marshal(config)
	json.Unmarshal(j, &config_load)
	config_load.Prepare()
	if len(config_load.List) != 1 {
		t.Errorf("SetPile.Learn() expected 1 item found %d", len(config_load.List))
	}
	config_load.Learn(&pile2)
	if len(config_load.List) != 1 {
		t.Errorf("SetPile.Learn() expected 1 item found %d", len(config_load.List))
	}
	j, _ = json.Marshal(config)
	json.Unmarshal(j, &config_load)
	config_load.Prepare()
	if len(config_load.List) != 1 {
		t.Errorf("SetPile.Learn() expected 1 item found %d", len(config_load.List))
	}
	config_load.Learn(&pile2)
	if len(config_load.List) != 1 {
		t.Errorf("SetPile.Learn() expected 1 item found %d", len(config_load.List))
	}
}
