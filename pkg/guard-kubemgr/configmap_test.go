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
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	guardfake "knative.dev/security-guard/pkg/client/clientset/versioned/fake"
)

func TestKubeMgr_GetConfig(t *testing.T) {
	config := map[string]string{"a": "b"}

	type fields struct {
		kclientset *k8sfake.Clientset
	}
	tests := []struct {
		name       string
		fields     fields
		config     map[string]string
		wantConfig map[string]string
		wantErr    bool
	}{
		{
			name: "missing",
			fields: fields{
				kclientset: k8sfake.NewSimpleClientset(&v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "xx-mysid",
						Namespace:   "myns",
						Annotations: map[string]string{},
					}})},
			wantConfig: config,
			wantErr:    true,
		},
		{
			name: "malformed",
			fields: fields{
				kclientset: k8sfake.NewSimpleClientset(&v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "mysid",
						Namespace:   "myns",
						Annotations: map[string]string{},
					}})},
			wantConfig: config,
			wantErr:    false,
		},
		{
			name: "empty",
			fields: fields{
				kclientset: k8sfake.NewSimpleClientset(&v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "mysid",
						Namespace:   "myns",
						Annotations: map[string]string{},
					},
					Data: map[string]string{},
				})},
			wantConfig: config,
			wantErr:    false,
		},
		{
			name: "ok",
			fields: fields{
				kclientset: k8sfake.NewSimpleClientset(&v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "mysid",
						Namespace:   "myns",
						Annotations: map[string]string{},
					},
					Data: map[string]string{"a": "c", "x": "y"},
				})},
			wantConfig: map[string]string{"a": "c", "x": "y"},
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := new(KubeMgr)
			k.getConfigFunc = fakeGetInclusterConfig
			k.cmClient = tt.fields.kclientset
			k.crdClient = guardfake.NewSimpleClientset().GuardV1alpha1()
			k.getConfigs()
			myconfig := make(map[string]string)
			for k, v := range config {
				myconfig[k] = v
			}
			if err := k.GetConfig("myns", "mysid", myconfig); (err != nil) != tt.wantErr {
				t.Errorf("KubeMgr.GetConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(myconfig, tt.wantConfig) {
				t.Errorf("KubeMgr.ReadCm() = %v, want %v", myconfig, tt.wantConfig)
			}
		})
	}
}
