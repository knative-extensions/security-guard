package testgate

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	pi "github.com/IBM/go-security-plugs/pluginterfaces"
)

type dLog struct {
}

func (d dLog) Debugf(format string, args ...interface{}) {}
func (d dLog) Infof(format string, args ...interface{})  {}
func (d dLog) Warnf(format string, args ...interface{})  {}
func (d dLog) Errorf(format string, args ...interface{}) {}
func (d dLog) Sync() error                               { return nil }

var defaultLog dLog

func testinit() *plug {
	p := new(plug)
	p.version = version
	p.name = name
	c := make(map[string]string)
	c["sender"] = "Sender"
	c["response"] = "response"

	pi.RegisterPlug(p)
	p.Init(context.Background(), c, "svcName", "myns", defaultLog)
	return p
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func Test_plug_Initialize(t *testing.T) {
	type args struct {
		l pi.Logger
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
		{"Log args", args{defaultLog}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = testinit()
		})
	}
}

func Test_plug_Shutdown(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
		{""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testinit()
			p.Shutdown()
		})
	}
}

func Test_plug_PlugName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// TODO: Add test cases.
		{"", "testgate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testinit()
			if got := p.PlugName(); got != tt.want {
				t.Errorf("plug.PlugName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_plug_PlugVersion(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// TODO: Add test cases.
		{"", version},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testinit()
			if got := p.PlugVersion(); got != tt.want {
				t.Errorf("plug.PlugVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_plug_ApproveResponse(t *testing.T) {
	t.Run("", func(t *testing.T) {
		p := testinit()

		req := httptest.NewRequest("GET", "/some/path", nil)
		respRecorder := httptest.NewRecorder()
		fmt.Fprintf(respRecorder, "Hi there!")
		resp := respRecorder.Result()
		resp.Request = req
		resp.Header.Set("name", "val")

		_, err1 := p.ApproveResponse(req, resp)
		if err1 != nil {
			t.Errorf("ApproveResponse error %v! ", err1)
		}
		if resp.Header.Get("X-Testgate-Bye") != "" {
			t.Errorf("ApproveResponse did not said Bye! ")
		}

		req.Header.Set("X-Testgate-Hi", "value")
		_, err2 := p.ApproveResponse(req, resp)
		if err2 != nil {
			t.Errorf("ApproveResponse error %v! ", err2)
		}
		if resp.Header.Get("X-Testgate-Bye") == "" {
			t.Errorf("ApproveResponse did not say Bye! ")
		}
	})

}

func Test_plug_ApproveRequest(t *testing.T) {
	t.Run("", func(t *testing.T) {
		p := testinit()
		req := httptest.NewRequest("GET", "/some/path", nil)
		req.Header.Set("name", "value")

		req1, err1 := p.ApproveRequest(req)

		if err1 != nil {
			t.Errorf("ApproveRequest returned error = %v", err1)
		}
		if req1 == nil {
			t.Errorf("ApproveRequest did not return a req ")
		}

		req.Header.Set("X-Testgate-Hi", "value")

		req2, err2 := p.ApproveRequest(req)

		if err2 != nil {
			t.Errorf("ApproveRequest returned error = %v", err1)
		}
		if req2 == nil {
			t.Errorf("ApproveRequest did not return a req ")
		}

	})

}

func Test_plug_Start(t *testing.T) {
	t.Run("Start", func(t *testing.T) {
		p := testinit()
		ctx := context.Background()
		if got := p.Start(ctx); !reflect.DeepEqual(got, ctx) {
			t.Errorf("plug.Start() = %v, want %v", got, ctx)
		}
	})

}
