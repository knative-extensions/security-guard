package v1

import (
	"reflect"
	"testing"
)

func ValueProfile_Test(t *testing.T, profile ValueProfile, pile ValuePile, config ValueConfig, args ...interface{}) {
	t.Run("Profile", func(t *testing.T) {

		// Test ProfileValue
		profile.Profile(args...)

		if str := profile.String(3); str == "" {
			t.Errorf("profile.String() no string provided")
		}

		if v := profile.DeepCopyValueProfile(); !reflect.DeepEqual(v, profile) {
			t.Errorf("DeepCopyValueProfile() = %v, want %v", v, profile)
		}

		// Test PileValue
		pile.Add(profile)
		pile.Clear()
		pile.Add(profile)
		pile.Add(profile)
		pile.Merge(pile)
		pile.Merge(pile)
		if str := pile.String(3); str == "" {
			t.Errorf("pile.String()  - no string provided")
		}
		if v := pile.DeepCopyValuePile(); !reflect.DeepEqual(v, pile) {
			t.Errorf("DeepCopyValuePile() = %v, want %v", v, pile)
		}

		// Test ConfigValue
		config.Learn(pile)
		config.Learn(pile)
		config.Merge(config)
		config.Merge(config)

		if str := config.Decide(profile); str != "" {
			t.Errorf("config.Decide(profile) wrong decission")
		}

		if str := config.String(3); str == "" {
			t.Errorf("config.String() no string provided")
		}

		if v := config.DeepCopyValueConfig(); !reflect.DeepEqual(v, config) {
			t.Errorf("DeepCopyValueConfig() = %v, want %v", v, config)
		}
	})
}
