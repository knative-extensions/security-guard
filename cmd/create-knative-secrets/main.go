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
	"crypto/rsa"
	"crypto/x509"
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	pkglogging "knative.dev/pkg/logging"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/control-protocol/pkg/certificates"
)

const (
	caExpirationInterval = time.Hour * 24 * 365 * 10 // 10 years
	expirationInterval   = time.Hour * 24 * 30       // 30 days
	rotationThreshold    = 10 * time.Minute
)

func main() {
	var err error
	var kubeCfg *rest.Config
	var devKubeConfigStr *string

	// Try to detect in-cluster config
	if kubeCfg, err = rest.InClusterConfig(); err != nil {
		// Not running in cluster
		if home := homedir.HomeDir(); home != "" {
			devKubeConfigStr = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			devKubeConfigStr = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()

		// Use the current context in kubeconfig
		if kubeCfg, err = clientcmd.BuildConfigFromFlags("", *devKubeConfigStr); err != nil {
			fmt.Printf("No Config found to access KubeApi! err: %v\n", err)
			return
		}
	}

	// Create a secrets client
	client, err := kubernetes.NewForConfig(kubeCfg)
	if err != nil {
		fmt.Printf("Failed to configure KubeAPi using config: %v\n", err)
		return
	}

	ctx := context.Background()
	l, _ := zap.NewDevelopment()
	logger := l.Sugar()
	ctx = pkglogging.WithLogger(ctx, logger)

	secrets := client.CoreV1().Secrets("knative-serving")

	// Certificate Authority
	caSecret, err := secrets.Get(context.Background(), "serving-certs-ctrl-ca", metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		fmt.Printf("knative-serving-certs secret is missing - lets create it\n")

		s := corev1.Secret{}
		s.Name = "serving-certs-ctrl-ca"
		s.Namespace = "knative-serving"
		s.Data = map[string][]byte{}
		caSecret, err = secrets.Create(context.Background(), &s, metav1.CreateOptions{})
	}
	if err != nil {
		fmt.Printf("Error accessing serving-certs-ctrl-ca secret: %v\n", err)
		return
	}
	caCert, caPk, err := parseAndValidateSecret(caSecret, false)
	if err != nil {
		fmt.Printf("serving-certs-ctrl-ca secret is missing the required keypair - lets add it\n")

		// We need to generate a new CA cert, then shortcircuit the reconciler
		keyPair, err := certificates.CreateCACerts(ctx, caExpirationInterval)
		if err != nil {
			fmt.Printf("Cannot generate the keypair for the serving-certs-ctrl-ca secret: %v\n", err)
			return
		}
		err = commitUpdatedSecret(client, caSecret, keyPair, nil)
		if err != nil {
			fmt.Printf("Failed to commit the keypair for the serving-certs-ctrl-ca secret: %v\n", err)
			return
		}
		caCert, caPk, err = parseAndValidateSecret(caSecret, false)
		if err != nil {
			fmt.Printf("Failed while validating keypair for serving-certs-ctrl-ca : %v\n", err)
			return
		}
	}
	fmt.Printf("Done processing serving-certs-ctrl-ca secret\n")

	// Certificate Authority Public Key
	caPublicSecret, err := secrets.Get(context.Background(), "serving-certs-ctrl-ca-public", metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		fmt.Printf("serving-certs-ctrl-ca-public secret is missing - lets create it\n")

		s := corev1.Secret{}
		s.Name = "serving-certs-ctrl-ca-public"
		s.Namespace = "knative-serving"
		s.Data = map[string][]byte{}
		caPublicSecret, err = secrets.Create(context.Background(), &s, metav1.CreateOptions{})
	}
	if err != nil {
		fmt.Printf("Error accessing serving-certs-ctrl-ca-public secret: %v\n", err)
		return
	}
	caPublicBytes := caSecret.Data[certificates.SecretCertKey]
	caPublicSecret.Data = make(map[string][]byte, 2)
	caPublicSecret.Data[certificates.CaCertName] = caPublicBytes
	fmt.Printf("Done processing serving-certs-ctrl-ca-public secret\n")

	_, err = client.CoreV1().Secrets("knative-serving").Update(context.Background(), caPublicSecret, metav1.UpdateOptions{})
	if err != nil {
		fmt.Printf("Error updating serving-certs-ctrl-ca-public secret: %v\n", err)
		return
	}
	// Current Keys
	secret, err := secrets.Get(context.Background(), "knative-serving-certs", metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		fmt.Printf("knative-serving-certs secret is missing - lets create it\n")

		s := corev1.Secret{}
		s.Name = "knative-serving-certs"
		s.Namespace = "knative-serving"
		s.Data = map[string][]byte{}
		secret, err = secrets.Create(context.Background(), &s, metav1.CreateOptions{})
	}
	if err != nil {
		fmt.Printf("Error accessing knative-serving-certs secret: %v\n", err)
		return
	}

	// Reconcile the provided secret
	_, _, err = parseAndValidateSecret(secret, true)
	if err != nil {
		fmt.Printf("knative-serving-certs secret is missing the required keypair - lets add it\n")

		// Check the secret to reconcile type
		var keyPair *certificates.KeyPair

		keyPair, err = certificates.CreateDataPlaneCert(ctx, caPk, caCert, expirationInterval)
		if err != nil {
			fmt.Printf("Cannot generate the keypair for the knative-serving-certs secret: %v\n", err)
			return
		}
		err = commitUpdatedSecret(client, secret, keyPair, caSecret.Data[certificates.SecretCertKey])
		if err != nil {
			fmt.Printf("Failed to commit the keypair for the knative-serving-certs secret: %v\n", err)
			return
		}
		_, _, err = certificates.ParseCert(keyPair.CertBytes(), keyPair.PrivateKeyBytes())
		if err != nil {
			fmt.Printf("Failed while validating keypair for knative-serving-certs secret: %v\n", err)
			return
		}
	}
	fmt.Printf("Done processing knative-serving-certs secret\n")
}

func commitUpdatedSecret(client kubernetes.Interface, secret *corev1.Secret, keyPair *certificates.KeyPair, caCert []byte) error {
	secret.Data = make(map[string][]byte, 6)
	secret.Data[certificates.CertName] = keyPair.CertBytes()
	secret.Data[certificates.PrivateKeyName] = keyPair.PrivateKeyBytes()
	secret.Data[certificates.SecretCertKey] = keyPair.CertBytes()
	secret.Data[certificates.SecretPKKey] = keyPair.PrivateKeyBytes()
	if caCert != nil {
		secret.Data[certificates.SecretCaCertKey] = caCert
		secret.Data[certificates.CaCertName] = caCert
	}

	_, err := client.CoreV1().Secrets(secret.Namespace).Update(context.Background(), secret, metav1.UpdateOptions{})
	return err
}

func parseAndValidateSecret(secret *corev1.Secret, shouldContainCaCert bool) (*x509.Certificate, *rsa.PrivateKey, error) {
	certBytes, ok := secret.Data[certificates.SecretCertKey]
	if !ok {
		return nil, nil, fmt.Errorf("missing cert bytes")
	}
	pkBytes, ok := secret.Data[certificates.SecretPKKey]
	if !ok {
		return nil, nil, fmt.Errorf("missing pk bytes")
	}
	if shouldContainCaCert {
		if _, ok := secret.Data[certificates.SecretCaCertKey]; !ok {
			return nil, nil, fmt.Errorf("missing ca cert bytes")
		}
	}

	caCert, caPk, err := certificates.ParseCert(certBytes, pkBytes)
	if err != nil {
		return nil, nil, err
	}
	if err := certificates.ValidateCert(caCert, rotationThreshold); err != nil {
		return nil, nil, err
	}
	return caCert, caPk, nil
}
