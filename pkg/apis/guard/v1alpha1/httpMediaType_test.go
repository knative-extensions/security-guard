package v1alpha1

import (
	"net/http"
	"testing"
)

func TestMediaType(t *testing.T) {
	header := http.Header{}
	header.Add("a", "x")
	header2 := http.Header{}
	header2.Add("b", "x")
	arguments := [][]string{
		{"text/html;charset=utf-8"},
		{"text/html;charset=utf-8;charset=utf-1;serkpodks485re&*^&^%&&%*"},
		{"text;charset=utf-8"},
		{"text/plain;charset=utf-8"},
		{"text/html;charset=utf-82"},
		{"text/html"},
		{"text"},
		{"375^^&87878"},
		{""},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(MediaTypeProfile))
		piles = append(piles, new(MediaTypePile))
		configs = append(configs, new(MediaTypeConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}
