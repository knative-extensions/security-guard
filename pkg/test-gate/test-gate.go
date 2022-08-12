/*
Copyright 2022 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testgate

import (
	"context"
	"net/http"

	pi "knative.dev/security-guard/pkg/pluginterfaces"
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

func (p *plug) Init(ctx context.Context, config map[string]string, serviceName string, namespace string, logger pi.Logger) context.Context {

	pi.Log.Infof("Plug %s: Never use in production", p.name)
	p.answer = "CU"
	p.sender = "someone"
	if config != nil {
		if v, ok := config["sender"]; ok && v != "" {
			p.sender = v
			pi.Log.Debugf("Plug %s: found sender %s", p.name, p.sender)
		}
		if v, ok := config["response"]; ok && v != "" {
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
