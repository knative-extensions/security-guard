package v1

import "testing"

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
		/*
			t.Run(tt.name, func(t *testing.T) {
				rp := &CountProfile{
					val: tt.fields.val,
				}
				rp.Profile(tt.args.args...)
			})
		*/
	}
}
