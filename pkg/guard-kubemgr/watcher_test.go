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
	"fmt"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stest "k8s.io/client-go/testing"
	spec "knative.dev/security-guard/pkg/apis/guard/v1alpha1"
	guardfake "knative.dev/security-guard/pkg/client/clientset/versioned/fake"
	guardv1alpha1 "knative.dev/security-guard/pkg/client/clientset/versioned/typed/guard/v1alpha1"

	"k8s.io/client-go/rest"
)

func TestKubeMgr_CM_WatchOnce(t *testing.T) {
	type fields struct {
		getConfigFunc func() (*rest.Config, error)
		cmClient      *k8sfake.Clientset
		cmWatcher     *watch.FakeWatcher
		crdClient     guardv1alpha1.GuardV1alpha1Interface
		crdWatcher    *watch.FakeWatcher
	}
	type args struct {
		ns     string
		cmFlag bool
		set    func(ns string, sid string, cmFlag bool, g *spec.GuardianSpec)
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				cmClient:   k8sfake.NewSimpleClientset(),
				cmWatcher:  watch.NewFake(),
				crdClient:  guardfake.NewSimpleClientset().GuardV1alpha1(),
				crdWatcher: watch.NewFake(),
			},
			args: args{
				ns:     "ns",
				cmFlag: true,
				set:    func(ns string, sid string, cmFlag bool, g *spec.GuardianSpec) {},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields.cmClient.PrependWatchReactor("configmaps", k8stest.DefaultWatchReactor(tt.fields.cmWatcher, nil))

			go func() {
				defer tt.fields.cmWatcher.Stop()
				for i := 0; i < 3; i++ {
					fmt.Printf("i=%d\n", i)
					time.Sleep(30 * time.Millisecond)
					tt.fields.cmWatcher.Delete(&v1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "guardian.sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Data: map[string]string{"Guardian": "{\"control\": {}}"},
					})
					time.Sleep(30 * time.Millisecond)
					tt.fields.cmWatcher.Delete(&v1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "XXguardian.sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Data: map[string]string{"Guardian": "{\"control\": {}}"},
					})
					time.Sleep(30 * time.Millisecond)
					tt.fields.cmWatcher.Modify(&v1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "XXguardian.sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Data: map[string]string{"Guardian": "{\"control\": {}}"},
					})
					time.Sleep(300 * time.Millisecond)
					tt.fields.cmWatcher.Add(&v1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "guardian.sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Data: map[string]string{"Guardian": "{\"control\": {}}"},
					})
					time.Sleep(30 * time.Millisecond)
					tt.fields.cmWatcher.Add(&v1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "guardian.sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Data: map[string]string{"Guardian": "aaa"},
					})
					time.Sleep(30 * time.Millisecond)
					tt.fields.cmWatcher.Add(&v1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "guardian.sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Data: map[string]string{"Guardian": ""},
					})
					time.Sleep(30 * time.Millisecond)
					tt.fields.cmWatcher.Add(&v1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "guardian.sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Data: map[string]string{"xxx": "aaa"},
					})
					time.Sleep(30 * time.Millisecond)
					tt.fields.cmWatcher.Add(&v1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "guardian.sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
					})
				}
			}()

			k := &KubeMgr{
				getConfigFunc: tt.fields.getConfigFunc,
				cmClient:      tt.fields.cmClient,
				crdClient:     tt.fields.crdClient,
			}
			k.WatchOnce(tt.args.ns, tt.args.cmFlag, tt.args.set)
		})
	}
}

func TestKubeMgr_CRD_WatchOnce(t *testing.T) {
	type fields struct {
		getConfigFunc func() (*rest.Config, error)
		cmClient      *k8sfake.Clientset
		cmWatcher     *watch.FakeWatcher
		crdClient     *guardfake.Clientset
		crdWatcher    *watch.FakeWatcher
	}
	type args struct {
		ns     string
		cmFlag bool
		set    func(ns string, sid string, cmFlag bool, g *spec.GuardianSpec)
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				cmClient:   k8sfake.NewSimpleClientset(),
				cmWatcher:  watch.NewFake(),
				crdClient:  guardfake.NewSimpleClientset(),
				crdWatcher: watch.NewFake(),
			},
			args: args{
				ns:     "ns",
				cmFlag: false,
				set:    func(ns string, sid string, cmFlag bool, g *spec.GuardianSpec) {},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields.crdClient.PrependWatchReactor("guardians", k8stest.DefaultWatchReactor(tt.fields.crdWatcher, nil))

			go func() {
				defer tt.fields.crdWatcher.Stop()
				for i := 0; i < 3; i++ {
					fmt.Printf("i=%d\n", i)
					time.Sleep(30 * time.Millisecond)

					tt.fields.crdWatcher.Delete(&spec.Guardian{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Spec: &spec.GuardianSpec{
							Control: &spec.Ctrl{},
						},
					})
					time.Sleep(30 * time.Millisecond)
					tt.fields.crdWatcher.Delete(&spec.Guardian{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Spec: &spec.GuardianSpec{
							Control: &spec.Ctrl{},
						},
					})
					time.Sleep(30 * time.Millisecond)
					tt.fields.crdWatcher.Modify(&spec.Guardian{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Spec: &spec.GuardianSpec{
							Control: &spec.Ctrl{},
						},
					})

					time.Sleep(30 * time.Millisecond)
					tt.fields.crdWatcher.Add(&spec.Guardian{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Spec: &spec.GuardianSpec{
							Control: &spec.Ctrl{},
						},
					})

					time.Sleep(300 * time.Millisecond)
					tt.fields.crdWatcher.Add(&spec.Guardian{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Spec: &spec.GuardianSpec{
							Control: &spec.Ctrl{},
						},
					})
					time.Sleep(30 * time.Millisecond)
					tt.fields.crdWatcher.Add(&spec.Guardian{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Spec: &spec.GuardianSpec{
							Control: &spec.Ctrl{},
						},
					})
					time.Sleep(30 * time.Millisecond)
					tt.fields.crdWatcher.Add(&spec.Guardian{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Spec: &spec.GuardianSpec{
							Control: &spec.Ctrl{},
						},
					})
					time.Sleep(30 * time.Millisecond)
					tt.fields.crdWatcher.Add(&spec.Guardian{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Spec: &spec.GuardianSpec{
							Control: &spec.Ctrl{},
						},
					})
					time.Sleep(30 * time.Millisecond)
					tt.fields.crdWatcher.Add(&spec.Guardian{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "sid",
							Namespace:   "ns",
							Annotations: map[string]string{},
						},
						Spec: &spec.GuardianSpec{
							Control: &spec.Ctrl{},
						},
					})

				}
			}()

			k := &KubeMgr{
				getConfigFunc: tt.fields.getConfigFunc,
				cmClient:      tt.fields.cmClient,
				crdClient:     tt.fields.crdClient.GuardV1alpha1(),
			}
			k.WatchOnce(tt.args.ns, tt.args.cmFlag, tt.args.set)
		})
	}
}
