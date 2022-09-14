package v1alpha1

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http/httptest"
	"testing"
)

func TestReq_V1(t *testing.T) {
	body := map[string][]string{
		"abc": {"ccc", "dddd"},
		"www": {"aaa", "bbb"},
	}

	jsonBytes1, _ := json.Marshal(body)
	jsonBytes2, _ := json.Marshal(body)
	req := httptest.NewRequest("GET", "/", bytes.NewReader(jsonBytes1))
	req2 := httptest.NewRequest("POST", "/eee/ddd/f.html", bytes.NewReader(jsonBytes2))

	cip := net.IPv4(1, 2, 3, 5)
	cip2 := net.IPv4(1, 22, 3, 5)
	arguments := [][]interface{}{
		{req, cip},
		{req2, cip2},
		{req2, cip},
		{req, cip},
		{req, cip},
		{req, cip},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(ReqProfile))
		piles = append(piles, new(ReqPile))
		configs = append(configs, new(ReqConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}
