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

package guardkubemgr

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pi "knative.dev/security-guard/pkg/pluginterfaces"
)

// GetConfig - Reads a Configmap and adds its value to the curent config
// Returns error if can't read the Configmap
func (k *KubeMgr) GetConfig(ns string, cmName string, config map[string]string) error {
	cm, err := k.cmClient.CoreV1().ConfigMaps(ns).Get(context.TODO(), cmName, metav1.GetOptions{})
	if err != nil {
		// can't read ConfigMap
		return fmt.Errorf("configmap %s read error %w", cmName, err)
	}

	for k, v := range cm.Data {
		pi.Log.Infof("configmap %s: %s", k, v)
		config[k] = v
	}
	return nil
}
