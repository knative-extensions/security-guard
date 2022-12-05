package v1alpha1

import (
	"reflect"
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

func TestKeyValConfig_Decide(t *testing.T) {
	type fields struct {
		Vals          map[string]*SimpleValConfig
		OtherVals     *SimpleValConfig
		OtherKeynames *SimpleValConfig
	}
	type args struct {
		profile *KeyValProfile
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Decision
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &KeyValConfig{
				Vals:          tt.fields.Vals,
				OtherVals:     tt.fields.OtherVals,
				OtherKeynames: tt.fields.OtherKeynames,
			}
			if got := config.Decide(tt.args.profile); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KeyValConfig.Decide() = %v, want %v", got, tt.want)
			}
		})
	}
}
