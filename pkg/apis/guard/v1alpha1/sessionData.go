package v1alpha1

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

//////////////////// SessionDataProfile ////////////////

// Exposes ValueProfile interface
type SessionDataProfile struct {
	Req      ReqProfile     `json:"req"`
	Resp     RespProfile    `json:"resp"`
	ReqBody  BodyProfile    `json:"reqbody"`
	RespBody BodyProfile    `json:"respbody"`
	Envelop  EnvelopProfile `json:"envelop"`
	Pod      PodProfile     `json:"pod"`
}

func (profile *SessionDataProfile) profileI(args ...interface{}) {
	req := args[0].(*http.Request)
	cip := args[1].(net.IP)
	resp := args[2].(*http.Response)
	reqData := args[3]
	respData := args[4]
	reqTime := args[5].(time.Time)
	respTime := args[6].(time.Time)
	endTime := args[7].(time.Time)
	profile.Profile(req, cip, resp, reqData, respData, reqTime, respTime, endTime)
}

func (profile *SessionDataProfile) Profile(req *http.Request, cip net.IP, resp *http.Response, reqData interface{}, respData interface{}, reqTime time.Time, respTime time.Time, endTime time.Time) {
	// never used
	profile.Req.Profile(req, cip)
	profile.Resp.Profile(resp)
	profile.ReqBody.ProfileStructured(reqData)
	profile.RespBody.ProfileStructured(respData)
	profile.Envelop.Profile(reqTime, respTime, endTime)
	profile.Pod.Profile()
}

//////////////////// SessionDataPile ////////////////

// Exposes ValuePile interface
type SessionDataPile struct {
	Count    uint32      `json:"count"`
	Req      ReqPile     `json:"req"`
	Resp     RespPile    `json:"resp"`
	ReqBody  BodyPile    `json:"reqbody"`
	RespBody BodyPile    `json:"respbody"`
	Envelop  EnvelopPile `json:"envelop"`
	Pod      PodPile     `json:"pod"`
}

func (pile *SessionDataPile) addI(valProfile ValueProfile) {
	pile.Add(valProfile.(*SessionDataProfile))
}

func (pile *SessionDataPile) Add(profile *SessionDataProfile) {
	pile.Count++
	pile.Req.Add(&profile.Req)
	pile.Resp.Add(&profile.Resp)
	pile.ReqBody.Add(&profile.ReqBody)
	pile.RespBody.Add(&profile.RespBody)
	pile.Envelop.Add(&profile.Envelop)
	pile.Pod.Add(&profile.Pod)
}

func (pile *SessionDataPile) Clear() {
	pile.Count = 0
	pile.Req.Clear()
	pile.Resp.Clear()
	pile.ReqBody.Clear()
	pile.RespBody.Clear()
	pile.Envelop.Clear()
	pile.Pod.Clear()
}

func (pile *SessionDataPile) mergeI(otherValPile ValuePile) {
	pile.Merge(otherValPile.(*SessionDataPile))
}

func (pile *SessionDataPile) Merge(otherPile *SessionDataPile) {
	pile.Count += otherPile.Count
	pile.Req.Merge(&otherPile.Req)
	pile.Resp.Merge(&otherPile.Resp)
	pile.ReqBody.Merge(&otherPile.ReqBody)
	pile.RespBody.Merge(&otherPile.RespBody)
	pile.Envelop.Merge(&otherPile.Envelop)
	pile.Pod.Merge(&otherPile.Pod)
}

//////////////////// SessionDataConfig ////////////////

// Exposes ValueConfig interface
type SessionDataConfig struct {
	Active   bool          `json:"active"`   // If not active, criteria ignored
	Req      ReqConfig     `json:"req"`      // Request criteria for blocking/allowing
	Resp     RespConfig    `json:"resp"`     // Response criteria for blocking/allowing
	ReqBody  BodyConfig    `json:"reqbody"`  // Request body criteria for blocking/allowing
	RespBody BodyConfig    `json:"respbody"` // Response body criteria for blocking/allowing
	Envelop  EnvelopConfig `json:"envelop"`  // Envelop criteria for blocking/allowing
	Pod      PodConfig     `json:"pod"`      // Pod criteria for blocking/allowing
}

func (config *SessionDataConfig) decideI(valProfile ValueProfile) string {
	return config.Decide(valProfile.(*SessionDataProfile))
}

func (config *SessionDataConfig) Decide(profile *SessionDataProfile) string {
	if ret := config.Req.Decide(&profile.Req); ret != "" {
		return fmt.Sprintf("Req: %s", ret)
	}
	if ret := config.Resp.Decide(&profile.Resp); ret != "" {
		return fmt.Sprintf("Resp: %s", ret)
	}
	if ret := config.ReqBody.Decide(&profile.ReqBody); ret != "" {
		return fmt.Sprintf("ReqBody: %s", ret)
	}
	if ret := config.RespBody.Decide(&profile.RespBody); ret != "" {
		return fmt.Sprintf("RespBody: %s", ret)
	}
	if ret := config.Envelop.Decide(&profile.Envelop); ret != "" {
		return fmt.Sprintf("Envelop: %s", ret)
	}
	if ret := config.Pod.Decide(&profile.Pod); ret != "" {
		return fmt.Sprintf("Pod: %s", ret)
	}

	return ""
}

func (config *SessionDataConfig) learnI(valPile ValuePile) {
	config.Learn(valPile.(*SessionDataPile))
}

func (config *SessionDataConfig) Learn(pile *SessionDataPile) {
	config.Req.Learn(&pile.Req)
	config.Resp.Learn(&pile.Resp)
	config.ReqBody.Learn(&pile.ReqBody)
	config.RespBody.Learn(&pile.RespBody)
	config.Envelop.Learn(&pile.Envelop)
	config.Pod.Learn(&pile.Pod)
}

func (config *SessionDataConfig) fuseI(otherValConfig ValueConfig) {
	config.Fuse(otherValConfig.(*SessionDataConfig))
}

func (config *SessionDataConfig) Fuse(otherConfig *SessionDataConfig) {
	config.Active = config.Active || otherConfig.Active
	config.Req.Fuse(&otherConfig.Req)
	config.Resp.Fuse(&otherConfig.Resp)
	config.ReqBody.Fuse(&otherConfig.ReqBody)
	config.RespBody.Fuse(&otherConfig.RespBody)
	config.Envelop.Fuse(&otherConfig.Envelop)
	config.Pod.Fuse(&otherConfig.Pod)
}
