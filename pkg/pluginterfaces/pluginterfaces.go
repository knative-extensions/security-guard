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

package pluginterfaces

import (
	"context"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// Any logger of this interface can be used by rtplugs and all connected plugs
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Sync() error
}

// The logger for the rtplugs and all connected plugs
var Log Logger

// A plugin based on the newer RoundTripPlug supports offers this interface
type RoundTripPlug interface {
	Init(ctx context.Context, config map[string]string, serviceName string, namespace string, logger Logger) context.Context
	Shutdown()
	PlugName() string
	PlugVersion() string
	ApproveRequest(*http.Request) (*http.Request, error)
	ApproveResponse(*http.Request, *http.Response) (*http.Response, error)
}

func init() {
	logger, _ := zap.NewDevelopment()
	Log = logger.Sugar()
}

var roundTripPlugs []RoundTripPlug

// GetPlugByName() is called by implementatuions supporting multiple plugs
func GetPlugByName(name string) RoundTripPlug {
	if len(roundTripPlugs) == 0 {
		Log.Warnf("Image was created with qpoption package but without plugs")
		return nil
	}
	for _, p := range roundTripPlugs {
		if strings.EqualFold(name, p.PlugName()) {
			return p
		}
	}
	return nil
}

// GetPlug() is called by implementations supporting a single plug
func GetPlug() RoundTripPlug {
	if len(roundTripPlugs) == 0 {
		Log.Warnf("Image was created with qpoption package but without a plug")
		return nil
	}
	if len(roundTripPlugs) > 1 {
		Log.Warnf("Image was created with multiple plugs")
		return nil
	}
	return roundTripPlugs[0]
}

// RegisterPlug() is called from init() function of plugs
func RegisterPlug(p RoundTripPlug) {
	roundTripPlugs = append(roundTripPlugs, p)
}
