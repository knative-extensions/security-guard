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

package main

// Uncomment when running in a development environment out side of the cluster
// import _ "k8s.io/client-go/plugin/pkg/client/auth"
/*
import (
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func init() {
	setTestEnvironment()
}

func setTestEnvironment() {
	os.Setenv("CONTAINER_CONCURRENCY", "1")
	os.Setenv("QUEUE_SERVING_PORT", "8765")
	os.Setenv("QUEUE_SERVING_TLS_PORT", "8764")
	os.Setenv("USER_PORT", "8877")
	os.Setenv("REVISION_TIMEOUT_SECONDS", "1")
	os.Setenv("SERVING_LOGGING_CONFIG", "debug")
	os.Setenv("SERVING_LOGGING_LEVEL", "debug")
	os.Setenv("SERVING_NAMESPACE", "myNs")
	os.Setenv("SERVING_CONFIGURATION", "myservice")
	os.Setenv("SERVING_REVISION", "myservice")
	os.Setenv("SERVING_POD", "myservice-pod")
	os.Setenv("SERVING_POD_IP", "1.2.3.4")
}
*/
