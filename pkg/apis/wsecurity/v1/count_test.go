package v1

import (
	"testing"
)

func TestCount_V1(t *testing.T) {
	arguments := [][]uint8{
		{4},
		{5},
		{1},
		{254},
		{255},
		{0},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(CountProfile))
		piles = append(piles, new(CountPile))
		configs = append(configs, new(CountConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}

	ValueTests_All(t, profiles, piles, configs, args...)
}

/*
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
		var profile CountProfile
		var pile CountPile
		var config CountConfig
		ValueProfile_Test(t, &profile, &pile, &config, tt.args.args...)
	}
}

func TestCountConfig_Fuse(t *testing.T) {
	type args struct {
		otherValConfig CountConfig
	}
	tests := []struct {
		name   string
		config *CountConfig
		args   args
		result *CountConfig
	}{{
		name:   "10-15, add 20-25",
		config: &CountConfig{{Min: 10, Max: 15}},
		args:   args{otherValConfig: CountConfig{{Min: 20, Max: 25}}},
		result: &CountConfig{{Min: 10, Max: 15}, {Min: 20, Max: 25}},
	}, {
		name:   "20-25 add 10-15",
		config: &CountConfig{{Min: 20, Max: 25}},
		args:   args{otherValConfig: CountConfig{{Min: 10, Max: 15}}},
		result: &CountConfig{{Min: 20, Max: 25}, {Min: 10, Max: 15}},
	}, {
		name:   "15-25, add 10-15",
		config: &CountConfig{{Min: 15, Max: 25}},
		args:   args{otherValConfig: CountConfig{{Min: 10, Max: 15}}},
		result: &CountConfig{{Min: 10, Max: 25}},
	}, {
		name:   "10-15, add 15-25",
		config: &CountConfig{{Min: 10, Max: 15}},
		args:   args{otherValConfig: CountConfig{{Min: 15, Max: 25}}},
		result: &CountConfig{{Min: 10, Max: 25}},
	}, {
		name:   "10-15, add 12-14",
		config: &CountConfig{{Min: 10, Max: 15}},
		args:   args{otherValConfig: CountConfig{{Min: 12, Max: 14}}},
		result: &CountConfig{{Min: 10, Max: 15}},
	}, {
		name:   "12-14, add 10-15",
		config: &CountConfig{{Min: 12, Max: 14}},
		args:   args{otherValConfig: CountConfig{{Min: 10, Max: 15}}},
		result: &CountConfig{{Min: 10, Max: 15}},

		//  Test Case that fail due to current Fuse limitation
		//	}, {
		//	name:   "20-25,10-15,30-35 add 15-20",
		//	config: &CountConfig{{Min: 20, Max: 25}, {Min: 10, Max: 15}, {Min: 30, Max: 35}},
		//	args:   args{otherValConfig: &CountConfig{{Min: 15, Max: 20}}},
		//	result: &CountConfig{{Min: 15, Max: 25}, {Min: 30, Max: 35}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.Fuse(&tt.args.otherValConfig)
			if !reflect.DeepEqual(tt.config, tt.result) {
				t.Errorf("Expected %v received %v\n", tt.result, tt.config)
			}
		})
	}
}

func TestCountConfig_Minimal(t *testing.T) {
	t.Run("minimal", func(t *testing.T) {
		var profile CountProfile
		var pile CountPile
		var config CountConfig
		profile.Profile(uint8(4))
		pile.Add(&profile)
		if len(pile) != 1 || pile[0] != 4 {
			t.Errorf("Expected [4] received %v\n", config)
		}
		config.Learn(&pile)
		if config[0].Max != 4 || config[0].Min != 4 || len(config) != 1 {
			t.Errorf("Expected [{4-4}] received %v\n", config)
		}

	})
	t.Run("repeated profile", func(t *testing.T) {
		var profile CountProfile
		var pile CountPile
		var config CountConfig
		profile.Profile(uint8(9))
		pile.Add(&profile)
		pile.Add(&profile)
		pile.Add(&profile)
		if len(pile) != 3 || pile[0] != 9 {
			t.Errorf("Expected [9] received %v\n", config)
		}
		config.Learn(&pile)
		if config[0].Max != 9 || config[0].Min != 9 || len(config) != 1 {
			t.Errorf("Expected [{9-9}] received %v\n", config)
		}
	})
	t.Run("different profile", func(t *testing.T) {
		var profile1 CountProfile
		var profile2 CountProfile
		var pile CountPile
		var config CountConfig
		profile1.Profile(uint8(4))
		profile2.Profile(uint8(7))
		pile.Add(&profile1)
		pile.Add(&profile2)
		if len(pile) != 2 || pile[0] != 4 || pile[1] != 7 {
			t.Errorf("Expected [4,7] received %v\n", pile)
		}
		config.Learn(&pile)
		if config[0].Max != 7 || config[0].Min != 4 || len(config) != 1 {
			t.Errorf("Expected [{4-7}] received %v\n", config)
		}
	})
	t.Run("merge piles", func(t *testing.T) {
		var profile1 CountProfile
		var profile2 CountProfile
		var profile3 CountProfile
		var profile4 CountProfile
		var pile1 CountPile
		var pile2 CountPile
		var config CountConfig
		profile1.Profile(uint8(4))
		profile2.Profile(uint8(7))
		profile3.Profile(uint8(14))
		profile4.Profile(uint8(17))
		pile1.Add(&profile1)
		pile1.Add(&profile2)
		pile2.Add(&profile3)
		pile2.Add(&profile4)
		pile1.Merge(&pile2)
		if len(pile1) != 4 {
			t.Errorf("Expected [4,7,14,17] received %v\n", pile2)
		}
		config.Learn(&pile1)
		if len(config) != 1 || config[0].Max != 17 || config[0].Min != 4 {
			t.Errorf("Expected [{4-17}] received %v\n", config)
		}
	})
	t.Run("fuse configs", func(t *testing.T) {
		var profile1 CountProfile
		var profile2 CountProfile
		var profile3 CountProfile
		var profile4 CountProfile
		var pile1 CountPile
		var pile2 CountPile
		var config1 CountConfig
		var config2 CountConfig
		profile1.Profile(uint8(4))
		profile2.Profile(uint8(7))
		profile3.Profile(uint8(14))
		profile4.Profile(uint8(17))
		pile1.Add(&profile1)
		pile1.Add(&profile2)
		pile2.Add(&profile3)
		pile2.Add(&profile4)
		config1.Learn(&pile1)
		config2.Learn(&pile2)
		config1.Fuse(&config2)
		if len(config1) != 2 || config1[0].Max != 7 || config1[0].Min != 4 || config1[1].Max != 17 || config1[1].Min != 14 {
			t.Errorf("Expected [{4-7},{14-17}] received %v\n", config1)
		}
	})

}
*/
