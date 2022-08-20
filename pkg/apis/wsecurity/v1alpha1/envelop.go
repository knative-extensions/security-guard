package v1alpha1

import (
	"fmt"
	"time"
)

// Envelop dataType maintains session data that is collected beyond Req, ReqBody, Resp and RespBody

//////////////////// EnvelopProfile ////////////////

// Exposes ValueProfile interface
type EnvelopProfile struct {
	ResponseTime   CountProfile `json:"responsetime"`
	CompletionTime CountProfile `json:"completiontime"`
}

func (profile *EnvelopProfile) Profile(args ...interface{}) {
	reqTime := args[0].(time.Time)
	respTime := args[1].(time.Time)
	endTime := args[2].(time.Time)

	completionTime := endTime.Sub(reqTime).Seconds()
	if completionTime > 255 {
		profile.CompletionTime = 255
	} else {
		profile.CompletionTime = CountProfile(completionTime)
	}

	responseTime := respTime.Sub(reqTime).Seconds()
	if responseTime > 255 {
		profile.ResponseTime = 255
	} else {
		profile.ResponseTime = CountProfile(responseTime)
	}
}

//////////////////// EnvelopPile ////////////////

// Exposes ValuePile interface
type EnvelopPile struct {
	ResponseTime   CountPile `json:"responsetime"`
	CompletionTime CountPile `json:"completiontime"`
}

func (pile *EnvelopPile) Add(valProfile ValueProfile) {
	profile := valProfile.(*EnvelopProfile)
	pile.CompletionTime.Add(&profile.CompletionTime)
	pile.ResponseTime.Add(&profile.ResponseTime)
}

func (pile *EnvelopPile) Clear() {
	pile.CompletionTime.Clear()
	pile.ResponseTime.Clear()
}

func (pile *EnvelopPile) Merge(otherValPile ValuePile) {
	otherPile := otherValPile.(*EnvelopPile)
	pile.CompletionTime.Merge(&otherPile.CompletionTime)
	pile.ResponseTime.Merge(&otherPile.ResponseTime)
}

//////////////////// EnvelopConfig ////////////////

// Exposes ValueConfig interface
type EnvelopConfig struct {
	ResponseTime   CountConfig `json:"responsetime"`
	CompletionTime CountConfig `json:"completiontime"`
}

func (config *EnvelopConfig) Decide(valProfile ValueProfile) string {
	profile := valProfile.(*EnvelopProfile)

	var ret string
	ret = config.ResponseTime.Decide(&profile.ResponseTime)
	if ret != "" {
		return fmt.Sprintf("ResponseTime: %s", ret)
	}
	ret = config.CompletionTime.Decide(&profile.CompletionTime)
	if ret != "" {
		return fmt.Sprintf("CompletionTime: %s", ret)
	}
	return ""
}

func (config *EnvelopConfig) Learn(valPile ValuePile) {
	pile := valPile.(*EnvelopPile)

	config.CompletionTime.Learn(&pile.CompletionTime)
	config.ResponseTime.Learn(&pile.ResponseTime)
}

func (config *EnvelopConfig) Fuse(otherValConfig ValueConfig) {
	otherConfig := otherValConfig.(*EnvelopConfig)

	config.CompletionTime.Fuse(&otherConfig.CompletionTime)
	config.ResponseTime.Fuse(&otherConfig.ResponseTime)
}
