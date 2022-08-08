package testgate

import (
	"context"
	"net/http"

	pi "github.com/IBM/go-security-plugs/pluginterfaces"
)

const version string = "0.0.7"
const name string = "testgate"

type plug struct {
	name    string
	version string

	sender string
	answer string

	// Add here any other state the extension needs
}

func (p *plug) PlugName() string {
	return p.name
}

func (p *plug) PlugVersion() string {
	return p.version
}

func (p *plug) ApproveRequest(req *http.Request) (*http.Request, error) {
	if _, ok := req.Header["X-Testgate-Hi"]; ok {
		pi.Log.Infof("Plug %s: hehe, %s noticed me!", p.name, p.sender)
	}
	return req, nil
}

func (p *plug) ApproveResponse(req *http.Request, resp *http.Response) (*http.Response, error) {
	if _, ok := req.Header["X-Testgate-Hi"]; ok {
		resp.Header.Add("X-Testgate-Bye", p.answer)
	}
	return resp, nil
}

func (p *plug) Shutdown() {
	pi.Log.Infof("Plug %s: Shutdown", p.name)
}

func (p *plug) Start(ctx context.Context) context.Context {
	return ctx
}

func (p *plug) Init(ctx context.Context, c map[string]string, serviceName string, namespace string, logger pi.Logger) context.Context {

	pi.Log.Infof("Plug %s: Never use in production", p.name)
	p.answer = "CU"
	p.sender = "someone"
	if c != nil {
		if v, ok := c["sender"]; ok && v != "" {
			p.sender = v
			pi.Log.Debugf("Plug %s: found sender %s", p.name, p.sender)
		}
		if v, ok := c["response"]; ok && v != "" {
			p.answer = v
			pi.Log.Debugf("Plug %s: found answer %s", p.name, p.answer)
		}
	}
	return ctx
}

func init() {
	p := new(plug)
	p.version = version
	p.name = name
	pi.RegisterPlug(p)
}
