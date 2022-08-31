package v1alpha1

import (
	"fmt"
)

//////////////////// SetProfile ////////////////

// Exposes ValueProfile interface
// A Slice of tokens
type SetProfile []string

func (profile *SetProfile) profileI(args ...interface{}) {
	switch v := args[0].(type) {
	case string:
		profile.ProfileString(v)
	case []string:
		profile.ProfileStringSlice(v)
	default:
		panic("Unsupported type in SetProfile")
	}
}

func (profile *SetProfile) ProfileString(str string) {
	*profile = []string{str}
}

func (profile *SetProfile) ProfileStringSlice(strSlice []string) {
	*profile = make(SetProfile, len(strSlice))
	copy(*profile, strSlice)
}

//////////////////// SetPile ////////////////

// Exposes ValuePile interface
// During json.Marshal(), SetPile exposes only the List
// After json.Unmarshal(), the map will be nil even when the List is not empty
// If the map is nil, it should be populated from the information in List
// If the map is populated it is always kept in-sync with the information in List
type SetPile struct {
	List []string `json:"set"`
	m    map[string]bool
}

func (pile *SetPile) addI(valProfile ValueProfile) {
	pile.Add(valProfile.(*SetProfile))
}

func (pile *SetPile) Add(profile *SetProfile) {
	if *profile == nil {
		return
	}
	if pile.m == nil {
		pile.m = make(map[string]bool, len(pile.List)+len(*profile))
		// Populate the map from the information in List
		for _, v := range pile.List {
			pile.m[v] = true
		}
	}
	for _, v := range *profile {
		if !pile.m[v] {
			pile.m[v] = true
			pile.List = append(pile.List, v)
		}
	}
}

func (pile *SetPile) Clear() {
	pile.m = nil
	pile.List = nil
}

func (pile *SetPile) mergeI(otherValPile ValuePile) {
	pile.Merge(otherValPile.(*SetPile))
}

func (pile *SetPile) Merge(otherPile *SetPile) {
	if pile.List == nil {
		pile.List = otherPile.List
		pile.m = otherPile.m
		return
	}

	if pile.m == nil {
		pile.m = make(map[string]bool, len(pile.List)+len(otherPile.List))
		// Populate the map from the information in List
		for _, v := range pile.List {
			pile.m[v] = true
		}
	}
	for _, v := range otherPile.List {
		if !pile.m[v] {
			pile.m[v] = true
			pile.List = append(pile.List, v)
		}
	}
}

//////////////////// SetConfig ////////////////

// Exposes ValueConfig interface
// During json.Marshal(), SetConfig exposes only the List
// After json.Unmarshal(), the map will be nil even when the List is not empty
// If the map is nil, it should be populated from the information in List
// If the map is populated it is always kept in-sync with the information in List
type SetConfig struct {
	List []string `json:"set"`
	m    map[string]bool
}

func (config *SetConfig) decideI(valProfile ValueProfile) string {
	return config.Decide(valProfile.(*SetProfile))
}

func (config *SetConfig) Decide(profile *SetProfile) string {
	if *profile == nil {
		return ""
	}

	if config.m == nil {
		config.m = make(map[string]bool, len(config.List))
		// Populate the map from the information in List
		for _, v := range config.List {
			config.m[v] = true
		}
	}
	for _, v := range *profile {
		if !config.m[v] {
			return fmt.Sprintf("Unexpected key %s in Set   ", v)
		}
	}

	return ""
}

func (config *SetConfig) learnI(valPile ValuePile) {
	config.Learn(valPile.(*SetPile))
}

func (config *SetConfig) Learn(pile *SetPile) {
	config.List = pile.List
	config.m = pile.m
}

func (config *SetConfig) fuseI(otherValConfig ValueConfig) {
	config.Fuse(otherValConfig.(*SetConfig))
}

func (config *SetConfig) Fuse(otherConfig *SetConfig) {
	if config.List == nil {
		config.List = otherConfig.List
		config.m = otherConfig.m
		return
	}
	if config.m == nil {
		config.m = make(map[string]bool, len(config.List)+len(otherConfig.List))
		// Populate the map from the information in List
		for _, v := range config.List {
			config.m[v] = true
		}
	}
	for _, v := range otherConfig.List {
		if !config.m[v] {
			config.m[v] = true
			config.List = append(config.List, v)
		}
	}
}
