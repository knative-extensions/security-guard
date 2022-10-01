/*
Copyright 2018 The Knative Authors

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

package main

import (
	"os"

	_ "knative.dev/security-guard/pkg/guard-gate"
	"knative.dev/security-guard/pkg/qpoption"
	"knative.dev/serving/pkg/queue/sharedmain"
)

// Knative Serving Queue Proxy with support for a guard-gate QPOption
func main() {
	qOpt := qpoption.NewGateQPOption()
	defer qOpt.Shutdown()

	if sharedmain.Main(qOpt.Setup) != nil {
		qOpt.Shutdown()
		os.Exit(1)
	}
}
