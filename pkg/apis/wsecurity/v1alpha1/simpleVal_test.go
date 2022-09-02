package v1alpha1

import (
	"testing"
)

const (
	hebrew1    = "A23/*מכבי*/"
	hebrew2    = "A23/*מכבימכבי*/"
	chineese   = "A23/*世界世界*/"
	loremIpsum = `What is Lorem Ipsum?
				Lorem Ipsum is simply xxxxx text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard xxxxx text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially unchanged. It was xxx in the 1960s with the release of xxx sheets containing Lorem Ipsum passages, and more recently with desktop publishing software like xxx PageMaker including versions of Lorem Ipsum.

				Why do we use it?
				`
)

func TestSimpleVal_All(t *testing.T) {
	arguments := [][]string{
		{"ABC"},
		{hebrew1},
		{hebrew2},
		{""},
		{chineese},
		{loremIpsum},
		{"$$"},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(SimpleValProfile))
		piles = append(piles, new(SimpleValPile))
		configs = append(configs, new(SimpleValConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}
