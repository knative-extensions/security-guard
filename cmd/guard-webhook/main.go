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
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"encoding/json"

	"go.uber.org/zap"
	admission "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"knative.dev/security-guard/pkg/certificates"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecFactory  = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecFactory.UniversalDeserializer()
	Log           *zap.SugaredLogger
	config        conf
	kubeMgr       KubeMgrInterface
	jsonPatchType = admission.PatchTypeJSONPatch
)

const MAX_MUTATE_BODY = 1000000

// add kind AdmissionReview in scheme
func init() {
	logger, _ := zap.NewDevelopment()
	Log = logger.Sugar()
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admission.AddToScheme(runtimeScheme)
	_ = appsv1.AddToScheme(runtimeScheme)
	_ = rbacv1.AddToScheme(runtimeScheme)
	_ = servingv1.AddToScheme(runtimeScheme)
}

func serveMutate(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, MAX_MUTATE_BODY)
		if data, err := io.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		Log.Error("contentType=%s, expect application/json", contentType)
		return
	}

	//Log.Info("handling request: %s", string(body))
	var requestObject, responseObj runtime.Object
	var gvk *schema.GroupVersionKind
	var err error
	if requestObject, gvk, err = deserializer.Decode(body, nil, nil); err != nil {
		msg := fmt.Sprintf("Request could not be decoded: %v", err)
		Log.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return

	}

	requestedAdmissionReview, ok := requestObject.(*admission.AdmissionReview)
	if !ok {
		Log.Error("Expected v1.AdmissionReview but got: %T", requestObject)
		return
	}

	responseAdmissionReview := &admission.AdmissionReview{}
	responseAdmissionReview.SetGroupVersionKind(*gvk)
	responseAdmissionReview.Response = respond(*requestedAdmissionReview)
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
	responseObj = responseAdmissionReview

	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		Log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(respBytes); err != nil {
		Log.Error(err)
	}
}

// Mutate and respond to incoming Knative Services
func respond(ar admission.AdmissionReview) (response *admission.AdmissionResponse) {
	defer func() { // recovers panic
		if e := recover(); e != nil {
			Log.Error("Panic during webhook: ", e)
			response = &admission.AdmissionResponse{Allowed: true}
			return
		}
	}()

	// Check first if operation is not allowed
	var obj runtime.Object
	var gvk *schema.GroupVersionKind
	var err error
	if obj, gvk, err = deserializer.Decode(ar.Request.Object.Raw, nil, nil); err != nil {
		panic(fmt.Sprintf("Failed deserialize %v  - error %s ", ar.Request.Resource, err.Error()))
	}

	// From here on - operation is allowed
	if gvk.Kind != "Service" {
		panic(fmt.Sprintf("Unexpected resource %v", ar.Request.Resource))
	}
	service := obj.(*servingv1.Service)
	jpatch := &jPatch{}

	// Configure guardian-reader serviceaccount, role and rolebinding

	if config.ServiceAccountEnable {
		if service.Spec.Template.Spec.ServiceAccountName == "" {
			Log.Info("Service - serviceAccountName added to service!")
			jpatch.jOps = append(jpatch.jOps, jOp{
				op:   "add",
				path: "/spec/template/spec/serviceaccountname",
				val:  "guardian-reader",
			})
		} else {
			if config.ServiceAccountForce {
				Log.Info("Service - serviceAccountName replaced in service!")
				jpatch.jOps = append(jpatch.jOps, jOp{
					op:   "replace",
					path: "/spec/template/spec/serviceaccountname",
					val:  "guardian-reader",
				})
			}
		}
	}

	// Configure Guard to be active
	if config.GuardEnable {
		if _, ok := service.Spec.Template.Annotations["qpoption.knative.dev/guard-activate"]; ok {
			Log.Info("Service - Guard already configured!")
			if config.GuardForce {
				Log.Info("Service - Guard forced!")
				jpatch.jOps = append(jpatch.jOps, jOp{
					op:   "replace",
					path: "/spec/template/metadata/annotations",
					val:  map[string]interface{}{"qpoption.knative.dev/guard-activate": "enable"},
				})
			}
		} else {
			Log.Info("Service - Guard not configured - now enabled!")
			jpatch.jOps = append(jpatch.jOps, jOp{
				op:   "add",
				path: "/spec/template/metadata/annotations",
				val:  map[string]interface{}{"qpoption.knative.dev/guard-activate": "enable"},
			})
		}
	}

	jpatch.build()
	if jpatch.empty {
		response = &admission.AdmissionResponse{Allowed: true}
	} else {
		response = &admission.AdmissionResponse{Allowed: true, PatchType: &jsonPatchType, Patch: jpatch.bytes}
	}
	return
}

type conf struct {
	GuardEnable          bool
	GuardForce           bool
	ServiceAccountEnable bool
	ServiceAccountForce  bool
}

func (c *conf) updateCm(cm *corev1.ConfigMap) {
	c.GuardEnable = true
	c.GuardForce = false
	c.ServiceAccountEnable = true
	c.ServiceAccountForce = true
	if cm == nil {
		Log.Info("ConfigMap missing - configure config-guard")
		return
	}
	if val, ok := cm.Data["ServiceAccount"]; ok {
		val = strings.ToLower(val)
		if val == "force" {
			c.ServiceAccountForce = true
		}
		if val == "disable" {
			c.ServiceAccountEnable = false
		}
	}
	if val, ok := cm.Data["Guard"]; ok {
		val = strings.ToLower(val)
		if val == "force" {
			c.GuardForce = true
		}
		if val == "disable" {
			c.GuardEnable = false
		}
	}
	Log.Info("ConfigMap loaded")
}

func main() {
	kubeMgr = NewKubeMgr()
	kubeMgr.InitConfigs()
	caExpirationInterval := time.Hour * 24 * 365 * 10 // 10 years
	expirationInterval := time.Hour * 24 * 30         // 30 days
	caKeyPair, err := certificates.CreateCACerts(caExpirationInterval)
	if err != nil {
		Log.Fatal("webhook  certificates.CreateCACerts failed", err)
	}
	sans := []string{"guard-webhook.knative-serving.svc"}
	caCert, caPk, err := certificates.ParseCert(caKeyPair.CertBytes(), caKeyPair.PrivateKeyBytes())
	if err != nil {
		Log.Fatal("webhook  certificates.ParseCert failed", err)
	}
	keyPair, err := certificates.CreateCert(caPk, caCert, expirationInterval, sans...)
	if err != nil {
		Log.Fatal("webhook  certificates.CreateCert failed", admission.ErrIntOverflowGenerated)
	}
	serverCert, err := tls.X509KeyPair(keyPair.CertBytes(), keyPair.PrivateKeyBytes())
	if err != nil {
		Log.Fatal("webhook  tls.X509KeyPair failed", err)
	}
	err = kubeMgr.CreateMutatingWebhookConfiguration(caKeyPair.CertBytes())
	if err != nil {
		Log.Fatal("CreateMutatingWebhookConfiguration ", err)
	}
	cm, err := kubeMgr.ReadGuardConfig()
	if err != nil {
		Log.Error(err)
	}
	config.updateCm(cm)
	go kubeMgr.WatchGuardConfig(config.updateCm)
	go kubeMgr.WatchGuardianSA("ServiceAccounts")
	go kubeMgr.WatchGuardianSA("Roles")
	go kubeMgr.WatchGuardianSA("RoleBindings")
	go kubeMgr.WatchKnativeServices()

	Log.Info("Server started ...")
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", serveMutate)
	server := &http.Server{
		Handler:           mux,
		Addr:              ":8443",
		ReadHeaderTimeout: 2 * time.Second,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{serverCert},
		},
	}
	err = server.ListenAndServeTLS("", "")
	if err != nil {
		Log.Fatal("ListenAndServeTLS", err)
	}
	//http.ListenAndServeTLS(":8443", "./secrets/tls.crt", "./secrets/tls.key", nil)
	//http.ListenAndServeTLS(":8443", "../../secrets/tls.crt", "../../secrets/tls.key", nil)

	Log.Fatal("webhook server exited")
}
