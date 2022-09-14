package v1alpha1

import (
	"net/http"
	"testing"
)

func TestResp_V1(t *testing.T) {
	resp := &http.Response{Header: http.Header{"a": {"x"}}}
	resp2 := &http.Response{Header: http.Header{"b": {"x"}}}
	arguments := [][]*http.Response{
		{resp},
		{resp2},
		{resp2},
		{resp},
		{resp},
		{resp},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(RespProfile))
		piles = append(piles, new(RespPile))
		configs = append(configs, new(RespConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}
