package v1

import (
	"reflect"
	"testing"
)

func TestCountProfile_Profile(t *testing.T) {
	type fields struct {
		val uint8
	}
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{{
		name:   "simple",
		fields: fields{val: 5},
		args: args{
			args: []interface{}{uint8(5)},
		},
	}}
	for _, tt := range tests {
		profile := &CountProfile{}
		pile := &CountPile{}
		config := &CountConfig{}
		ValueProfile_Test(t, profile, pile, config, tt.args.args...)
	}
}

func TestCountConfig_Merge(t *testing.T) {
	type args struct {
		otherValConfig ValueConfig
	}
	tests := []struct {
		name   string
		config *CountConfig
		args   args
		result *CountConfig
	}{{
		name:   "10-15, 20-25",
		config: &CountConfig{{Min: 10, Max: 15}},
		args:   args{otherValConfig: &CountConfig{{Min: 20, Max: 25}}},
		result: &CountConfig{{Min: 10, Max: 15}, {Min: 20, Max: 25}},
	}, {
		name:   "10-15, 15-25",
		config: &CountConfig{{Min: 10, Max: 15}},
		args:   args{otherValConfig: &CountConfig{{Min: 15, Max: 25}}},
		result: &CountConfig{{Min: 10, Max: 25}},
	}, {
		name:   "10-15, 12-14",
		config: &CountConfig{{Min: 10, Max: 15}},
		args:   args{otherValConfig: &CountConfig{{Min: 12, Max: 14}}},
		result: &CountConfig{{Min: 10, Max: 15}},
	}, {
		name:   "12-14, 10-15",
		config: &CountConfig{{Min: 12, Max: 14}},
		args:   args{otherValConfig: &CountConfig{{Min: 10, Max: 15}}},
		result: &CountConfig{{Min: 10, Max: 15}},
	}, {
		name:   "15-25, 10-15",
		config: &CountConfig{{Min: 15, Max: 25}},
		args:   args{otherValConfig: &CountConfig{{Min: 10, Max: 15}}},
		result: &CountConfig{{Min: 10, Max: 25}},
	}, {
		name:   "20-25, 10-15",
		config: &CountConfig{{Min: 20, Max: 25}},
		args:   args{otherValConfig: &CountConfig{{Min: 10, Max: 15}}},
		result: &CountConfig{{Min: 20, Max: 25}, {Min: 10, Max: 15}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.Merge(tt.args.otherValConfig)
			if !reflect.DeepEqual(tt.config, tt.result) {
				t.Errorf("Expected %v received %v\n", tt.result, tt.config)
			}
		})
	}
}
