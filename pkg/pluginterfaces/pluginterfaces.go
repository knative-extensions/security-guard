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
	"fmt"
	"net/http"
	"strings"
	"sync"

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
var LogOnce Logger

// A plugin based on the newer RoundTripPlug supports offers this interface
//
// The plugin will have a function
//
//	func NewPlug()  RoundTripPlug {}
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

	lo := new(LogOnce_type)
	lo.known = make([]map[string]uint, 4)
	for level := 0; level < 4; level++ {
		lo.known[level] = make(map[string]uint)
	}
	LogOnce = lo
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

type LogOnce_type struct {
	known []map[string]uint
	mutex sync.Mutex
}

func (lo *LogOnce_type) Debugf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	if lo.once(0, str) {
		Log.Debugf(str)
	}
}

func (lo *LogOnce_type) Infof(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	if lo.once(1, str) {
		Log.Infof(str)
	}
}

func (lo *LogOnce_type) Warnf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	if lo.once(2, str) {
		Log.Warnf(str)
	}
}

func (lo *LogOnce_type) Errorf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	if lo.once(3, str) {
		Log.Errorf(str)
	}
}

func (lo *LogOnce_type) Sync() error {
	return Log.Sync()
}

func (lo *LogOnce_type) once(level int, str string) (firstTime bool) {
	lo.mutex.Lock()
	defer lo.mutex.Unlock()
	if _, ok := lo.known[level][str]; !ok {
		lo.known[level][str] = 0
		firstTime = true
	}
	lo.known[level][str]++
	return
}
