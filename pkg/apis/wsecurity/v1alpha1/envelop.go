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

func (profile *EnvelopProfile) profileI(args ...interface{}) {
	profile.Profile(args[0].(time.Time), args[1].(time.Time), args[2].(time.Time))
}

func (profile *EnvelopProfile) Profile(reqTime time.Time, respTime time.Time, endTime time.Time) {

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

func (pile *EnvelopPile) addI(valProfile ValueProfile) {
	pile.Add(valProfile.(*EnvelopProfile))
}

func (pile *EnvelopPile) Add(profile *EnvelopProfile) {
	pile.CompletionTime.Add(profile.CompletionTime)
	pile.ResponseTime.Add(profile.ResponseTime)
}

func (pile *EnvelopPile) Clear() {
	pile.CompletionTime.Clear()
	pile.ResponseTime.Clear()
}

func (pile *EnvelopPile) mergeI(otherValPile ValuePile) {
	pile.Merge(otherValPile.(*EnvelopPile))
}

func (pile *EnvelopPile) Merge(otherPile *EnvelopPile) {
	pile.CompletionTime.Merge(otherPile.CompletionTime)
	pile.ResponseTime.Merge(otherPile.ResponseTime)
}

//////////////////// EnvelopConfig ////////////////

// Exposes ValueConfig interface
type EnvelopConfig struct {
	ResponseTime   CountConfig `json:"responsetime"`
	CompletionTime CountConfig `json:"completiontime"`
}

func (config *EnvelopConfig) decideI(valProfile ValueProfile) string {
	return config.Decide(valProfile.(*EnvelopProfile))
}

func (config *EnvelopConfig) Decide(profile *EnvelopProfile) string {
	var ret string
	ret = config.ResponseTime.Decide(profile.ResponseTime)
	if ret != "" {
		return fmt.Sprintf("ResponseTime: %s", ret)
	}
	ret = config.CompletionTime.Decide(profile.CompletionTime)
	if ret != "" {
		return fmt.Sprintf("CompletionTime: %s", ret)
	}
	return ""
}

func (config *EnvelopConfig) learnI(valPile ValuePile) {
	config.Learn(valPile.(*EnvelopPile))
}

func (config *EnvelopConfig) Learn(pile *EnvelopPile) {
	config.CompletionTime.Learn(pile.CompletionTime)
	config.ResponseTime.Learn(pile.ResponseTime)
}

func (config *EnvelopConfig) fuseI(otherValConfig ValueConfig) {
	config.Fuse(otherValConfig.(*EnvelopConfig))
}

func (config *EnvelopConfig) Fuse(otherConfig *EnvelopConfig) {
	config.CompletionTime.Fuse(otherConfig.CompletionTime)
	config.ResponseTime.Fuse(otherConfig.ResponseTime)
}
