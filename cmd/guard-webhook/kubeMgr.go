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

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"time"

	v1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	knativeservicesv1 "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

const (
	cmNs       = "knative-serving"
	cmName     = "config-guard"
	saName     = "guardian-reader"
	guardLabel = "guard.security.knative.dev/guard"
)

var (
	guardListOpt = metav1.ListOptions{
		LabelSelector: "guard.security.knative.dev/guard=true",
	}
)

type KubeMgrInterface interface {
	InitConfigs()
	ReadGuardConfig() (*corev1.ConfigMap, error)
	WatchGuardConfig(set func(cm *corev1.ConfigMap))
	AddNamespace(ns string)
	WatchGuardianSA(resource string)
	WatchKnativeServices()
	CreateMutatingWebhookConfiguration([]byte) error
}

type KubeMgr struct {
	// Function for returning k8s config
	getConfigFunc func() (*rest.Config, error)

	// Kubernetes client for Config Maps
	client kubernetes.Interface

	// CRD client for Knative Services
	kServClient knativeservicesv1.ServingV1Interface

	// Namespaces
	namespaces map[string]bool
}

func NewKubeMgr() KubeMgrInterface {
	k := new(KubeMgr)
	k.getConfigFunc = rest.InClusterConfig
	k.namespaces = make(map[string]bool)
	return k
}

func (k *KubeMgr) AddNamespace(ns string) {
	if _, ok := k.namespaces[ns]; ok {
		return
	}
	k.createGuardReaderSA(ns)
	k.createGuardReaderR(ns)
	k.createGuardReaderRB(ns)
	k.namespaces[ns] = true
}

// createMutatingWebhookConfiguration - Create MutatingWebhookConfiguration to monitor knative services
// Returns error if can't create MutatingWebhookConfiguration
func (k *KubeMgr) CreateMutatingWebhookConfiguration(caBundle []byte) error {
	failurePolicy := v1.Fail
	sideEffects := v1.SideEffectClassNone
	timeout := int32(10)
	port := int32(443)
	scope := v1.NamespacedScope
	path := "/mutate"
	webhookName := "webhook.services.guard.security.knative.dev"
	configuredWebhook := &v1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
			Labels: map[string]string{
				"app.kubernetes.io/component": "webhook",
				"app.kubernetes.io/name":      "knative-serving",
				"app.kubernetes.io/version":   "devel",
			},
		},

		Webhooks: []v1.MutatingWebhook{{
			Name:                    webhookName,
			AdmissionReviewVersions: []string{"v1"},
			FailurePolicy:           &failurePolicy,
			SideEffects:             &sideEffects,
			TimeoutSeconds:          &timeout,
			Rules: []v1.RuleWithOperations{{
				Operations: []v1.OperationType{"CREATE", "UPDATE"},
				Rule: v1.Rule{
					APIGroups:   []string{"serving.knative.dev"},
					APIVersions: []string{"v1", "v1beta1", "v1alpha1"},
					Resources:   []string{"services"},
					Scope:       &scope,
				},
			}},
			ClientConfig: v1.WebhookClientConfig{
				Service: &v1.ServiceReference{
					Namespace: "knative-serving",
					Name:      "guard-webhook",
					Port:      &port,
					Path:      &path,
				},
				CABundle: caBundle,
			},
		}},
	}
	k.client.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.TODO(), webhookName, metav1.DeleteOptions{})
	_, err := k.client.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), configuredWebhook, metav1.CreateOptions{})
	if err != nil {
		// can't create MutatingWebhookConfigurations
		return fmt.Errorf("create error %w", err)
	}
	return nil
}

// readCm - Reads a Guardian ConfigMap from KubeApi
// Returns a Guardian
// Returns error if can't read a Guardian from a well structured ConfigMap
func (k *KubeMgr) ReadGuardConfig() (*corev1.ConfigMap, error) {
	cm, err := k.client.CoreV1().ConfigMaps(cmNs).Get(context.TODO(), cmName, metav1.GetOptions{})
	if err != nil {
		// can't read ConfigMap
		return nil, fmt.Errorf("configmap %s read error %w", cmName, err)
	}

	return cm, nil
}

func (k *KubeMgr) getConfigs() *rest.Config {
	var err error
	var kubeCfg *rest.Config
	var devKubeConfigStr *string

	// Try to detect in-cluster config
	if kubeCfg, err = k.getConfigFunc(); err == nil {
		return kubeCfg
	}

	// Not running in cluster
	if home := homedir.HomeDir(); home != "" {
		devKubeConfigStr = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		devKubeConfigStr = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// Use the current context in kubeconfig
	if kubeCfg, err = clientcmd.BuildConfigFromFlags("", *devKubeConfigStr); err != nil {
		panic(fmt.Sprintf("No Config found! err %s", err.Error()))
	}
	return kubeCfg
}

// Initialize the Kubernetes client and CRD client to communicate with the KubeApi
func (k *KubeMgr) InitConfigs() {
	var err error

	kubeCfg := k.getConfigs()

	// Create a cm client
	k.client, err = kubernetes.NewForConfig(kubeCfg)
	if err != nil {
		panic(err.Error())
	}
	// Create a knative client
	k.kServClient, err = knativeservicesv1.NewForConfig(kubeCfg)
	if err != nil {
		panic(err.Error())
	}

}

func (k *KubeMgr) createGuardReaderSA(ns string) {
	Log.Infof("Namespace %s - testing ServiceAccount!", ns)
	// ServiceAccount
	newServiceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:   saName,
			Labels: map[string]string{guardLabel: "true"},
		},
	}
	currentServiceAccount, err := k.client.CoreV1().ServiceAccounts(ns).Get(context.Background(), saName, metav1.GetOptions{})
	if err != nil {
		Log.Infof("ServiceAccount Get: %s", err.Error())
		currentServiceAccount = nil
	}
	if currentServiceAccount == nil {
		Log.Infof("ServiceAccount Create")
		_, err = k.client.CoreV1().ServiceAccounts(ns).Create(context.Background(), newServiceAccount, metav1.CreateOptions{})
		if err != nil {
			Log.Infof("ServiceAccount Create: %s", err.Error())
		}
	} else if !equality.Semantic.DeepEqual(currentServiceAccount.Labels, newServiceAccount.Labels) {
		Log.Infof("ServiceAccount Update")
		_, err = k.client.CoreV1().ServiceAccounts(ns).Update(context.Background(), newServiceAccount, metav1.UpdateOptions{})
		if err != nil {
			Log.Infof("ServiceAccount Update: %s", err.Error())
		}
	}
}

func (k *KubeMgr) createGuardReaderR(ns string) {
	Log.Infof("Namespace %s - testing Role!", ns)
	// Role
	newRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:   saName,
			Labels: map[string]string{guardLabel: "true"},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"guard.security.knative.dev"},
				Resources: []string{"guardians"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
	currentRole, err := k.client.RbacV1().Roles(ns).Get(context.Background(), saName, metav1.GetOptions{})
	if err != nil {
		Log.Infof("Role Get: %s", err.Error())
		currentRole = nil
	}
	if currentRole == nil {
		Log.Infof("Role Create")
		_, err = k.client.RbacV1().Roles(ns).Create(context.Background(), newRole, metav1.CreateOptions{})
		if err != nil {
			Log.Infof("Role Create: %s", err.Error())
		}
	} else if !equality.Semantic.DeepEqual(currentRole.Rules, newRole.Rules) || !equality.Semantic.DeepEqual(currentRole.Labels, newRole.Labels) {
		Log.Infof("Role Update")
		_, err = k.client.RbacV1().Roles(ns).Update(context.Background(), newRole, metav1.UpdateOptions{})
		if err != nil {
			Log.Infof("Role Update: %s", err.Error())
		}
	}
}
func (k *KubeMgr) createGuardReaderRB(ns string) {
	Log.Infof("Namespace %s - testing RoleBinding!", ns)
	// RoleBinding
	newRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   saName,
			Labels: map[string]string{guardLabel: "true"},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     saName,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: "",
				Kind:     "ServiceAccount",
				Name:     saName,
			},
		},
	}

	currentRoleBinding, err := k.client.RbacV1().RoleBindings(ns).Get(context.Background(), saName, metav1.GetOptions{})
	if err != nil {
		Log.Infof("RoleBinding Get: %s", err.Error())
		currentRoleBinding = nil
	}

	if currentRoleBinding == nil {
		Log.Infof("RoleBinding Create")
		_, err = k.client.RbacV1().RoleBindings(ns).Create(context.Background(), newRoleBinding, metav1.CreateOptions{})
		if err != nil {
			Log.Infof("RoleBinding Create: %s", err.Error())
		}
	} else if !equality.Semantic.DeepEqual(currentRoleBinding.Subjects, newRoleBinding.Subjects) ||
		!equality.Semantic.DeepEqual(currentRoleBinding.RoleRef, newRoleBinding.RoleRef) ||
		!equality.Semantic.DeepEqual(currentRoleBinding.Labels, newRoleBinding.Labels) {
		Log.Infof("RoleBinding Update")
		_, err = k.client.RbacV1().RoleBindings(ns).Update(context.Background(), newRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			Log.Infof("RoleBinding Update: %s", err.Error())
		}
	}
}

// Watch never returns - use with a goroutine
// Watch for changes in ConfigMap
func (k *KubeMgr) WatchGuardConfig(set func(cm *corev1.ConfigMap)) {
	for {
		k.watchGuardConfigOnce(set)
		timeout, _ := time.ParseDuration("100s")
		time.Sleep(timeout)
	}
}

// Watch for changes in ConfigMap
func (k *KubeMgr) watchGuardConfigOnce(set func(cm *corev1.ConfigMap)) (e error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			Log.Errorf("recovered from panic while watching cm and crd for ns %s! recover: %v", cmNs, recovered)
		}
	}()
	watcherCm, err := k.client.CoreV1().ConfigMaps(cmNs).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("watch cm ns %s err %v", cmNs, err)
	}
	chCm := watcherCm.ResultChan()
	for {
		select {
		case event, ok := <-chCm:
			if !ok {
				// the channel got closed, so we need to restart
				return fmt.Errorf("watch cm ns %s kubernetes hung up on us, restarting event watcher", cmNs)
			}

			// handle the cm event
			switch event.Type {
			case watch.Deleted:
				fallthrough
			case watch.Modified:
				fallthrough
			case watch.Added:
				cm, ok := event.Object.(*corev1.ConfigMap)
				if !ok {
					return fmt.Errorf("watch cm ns %s kubernetes cant convert to type configmap", cmNs)
				}
				if cm.ObjectMeta.Name != cmName || cm.ObjectMeta.Namespace != cmNs {
					// skip...
					continue
				}
				if event.Type == watch.Deleted {
					set(nil)
					continue
				}
				set(cm)
			case watch.Error:
				s := event.Object.(*metav1.Status)
				Log.Infof("Error during watch CM: \n\tListMeta %v\n\tTypeMeta %v\n", s.ListMeta, s.TypeMeta)
			}
		case <-time.After(10 * time.Minute):
			// deal with the issue where we get no events
			return fmt.Errorf("watch cm %s ns %s timeout, restarting event watcher", cmName, cmNs)
		}
	}
}

// Watch never returns - use with a goroutine
// Watch for changes in ConfigMap
func (k *KubeMgr) WatchGuardianSA(resource string) {
	for {
		k.watchOnce(resource)
		timeout, _ := time.ParseDuration("100s")
		time.Sleep(timeout)
	}
}

// Watch for changes in ServiceAccount
func (k *KubeMgr) watchOnce(resource string) (e error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			Log.Errorf("recovered from panic while watching %s! recover: %v", resource, recovered)
		}
	}()
	var watcher watch.Interface
	var err error
	switch resource {
	case "ServiceAccounts":
		watcher, err = k.client.CoreV1().ServiceAccounts("").Watch(context.TODO(), guardListOpt)
	case "Roles":
		watcher, err = k.client.RbacV1().Roles("").Watch(context.TODO(), guardListOpt)
	case "RoleBindings":
		watcher, err = k.client.RbacV1().RoleBindings("").Watch(context.TODO(), guardListOpt)
	}
	if err != nil {
		return fmt.Errorf("watch %s err %v", resource, err)
	}
	ch := watcher.ResultChan()
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				// the channel got closed, so we need to restart
				return fmt.Errorf("watch  %s kubernetes hung up on us, restarting event watcher", resource)
			}
			// handle the cm event
			switch event.Type {
			case watch.Deleted:
				fallthrough
			case watch.Modified:
				fallthrough
			case watch.Added:
				switch resource {
				case "ServiceAccounts":
					if o, ok := event.Object.(*corev1.ServiceAccount); ok {
						k.createGuardReaderSA(o.ObjectMeta.Namespace)
					}
				case "Roles":
					if o, ok := event.Object.(*rbacv1.Role); ok {
						k.createGuardReaderR(o.ObjectMeta.Namespace)
					}
				case "RoleBindings":
					if o, ok := event.Object.(*rbacv1.RoleBinding); ok {
						k.createGuardReaderRB(o.ObjectMeta.Namespace)
					}
				}
			case watch.Error:
				s := event.Object.(*metav1.Status)
				Log.Infof("Error during watch %s: \n\tListMeta %v\n\tTypeMeta %v\n", resource, s.ListMeta, s.TypeMeta)
			}
		case <-time.After(10 * time.Minute):
			// deal with the issue where we get no events
			return fmt.Errorf("watch %s timeout, restarting event watcher", resource)
		}
	}
}

// Watch never returns - use with a goroutine
// Watch for changes in ConfigMap
func (k *KubeMgr) WatchKnativeServices() {
	for {
		k.watchKnativeServicesOnce()
		timeout, _ := time.ParseDuration("100s")
		time.Sleep(timeout)
	}
}

// Watch for changes in Knative Services
func (k *KubeMgr) watchKnativeServicesOnce() (e error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			Log.Errorf("Panic while watching knative services: %v", recovered)
		}
	}()
	var watcher watch.Interface
	var err error
	watcher, err = k.kServClient.Services("").Watch(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return fmt.Errorf("watch knative services err %v", err)
	}
	ch := watcher.ResultChan()
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				// the channel got closed, so we need to restart
				return fmt.Errorf("watch knative services kubernetes hung up on us, restarting event watcher")
			}
			// handle the cm event
			switch event.Type {
			case watch.Modified, watch.Added:
				if config.ServiceAccountEnable {
					if o, ok := event.Object.(*servingv1.Service); ok {
						k.AddNamespace(o.ObjectMeta.Namespace)
					}
				}
			case watch.Error:
				s := event.Object.(*metav1.Status)
				Log.Infof("Error during watch knative services: \n\tListMeta %v\n\tTypeMeta %v\n", s.ListMeta, s.TypeMeta)
			}
		case <-time.After(10 * time.Minute):
			// deal with the issue where we get no events
			return fmt.Errorf("watch knative services timeout, restarting event watcher")
		}
	}
}
