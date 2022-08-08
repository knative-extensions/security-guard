// Commmon interfaces used by github.com/IBM/go-security-plugs/rtplugs and the plgins it loads
package pluginterfaces

import (
	"context"
	"net/http"

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
//
// The plugin will have a function
//		func NewPlug()  RoundTripPlug {}
//
type RoundTripPlug interface {
	Init(ctx context.Context, c map[string]string, serviceName string, namespace string, logger Logger) context.Context
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

var RoundTripPlugs []RoundTripPlug

// RegisterPlug() is called from init() function of plugs
func RegisterPlug(p RoundTripPlug) {
	RoundTripPlugs = append(RoundTripPlugs, p)
}
