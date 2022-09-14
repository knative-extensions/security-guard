package v1alpha1

import (
	"testing"
	"time"
)

func TestEnvelop_V1(t *testing.T) {
	interval, _ := time.ParseDuration("2h")
	arguments := [][]time.Time{
		{time.Now(), time.Now(), time.Now()},
		{time.Now(), time.Now().Add(interval), time.Now().Add(interval)},
		{time.Now(), time.Now().Add(interval), time.Now().Add(interval)},
		{time.Now(), time.Now(), time.Now()},
		{time.Now(), time.Now(), time.Now()},
		{time.Now(), time.Now(), time.Now()},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(EnvelopProfile))
		piles = append(piles, new(EnvelopPile))
		configs = append(configs, new(EnvelopConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}
