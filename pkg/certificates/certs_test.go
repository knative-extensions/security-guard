/*
Copyright 2021 The Knative Authors

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

package certificates

import (
	"testing"
	"time"
)

func TestCreateCACerts(t *testing.T) {

	t.Run("base", func(t *testing.T) {
		caKp, err := CreateCACerts(time.Hour)
		if err != nil {
			t.Errorf("CreateCACerts() error = %v", err)
			return
		}

		caCertPem := caKp.Cert()

		if caCertPem == nil {
			t.Errorf("Empty Cert")
			return
		}
		if caCertPem.Type != "CERTIFICATE" {
			t.Errorf("Cert wrong type %s", caCertPem.Type)
			return
		}

		caPkPem := caKp.PrivateKey()
		if caPkPem == nil {
			t.Errorf("Empty PrivateKey")
			return
		}
		if caPkPem.Type != "RSA PRIVATE KEY" {
			t.Errorf("PrivateKey wrong type %s", caPkPem.Type)
			return
		}

		caCert1, caPk1, err := caKp.Parse()
		if err != nil {
			t.Errorf("Error in caKp.Parse %v", err)
			return
		}

		caCert, caPk, err := ParseCert(caKp.CertBytes(), caKp.PrivateKeyBytes())
		if err != nil {
			t.Errorf("Error in ParseCert %v", err)
			return
		}

		if !caCert1.Equal(caCert) {
			t.Errorf("non matching caCerts")
			return
		}
		if !caPk1.Equal(caPk) {
			t.Errorf("non matching caPk")
			return
		}

		if err := CheckExpiry(caCert, 1); err != nil {
			t.Errorf("CheckExpiry error %v", err)
			return
		}

		sans := []string{"guard-webhook.knative-serving.svc"}
		kp, err := CreateCert(caPk, caCert, 1, sans...)
		if err != nil {
			t.Errorf("CreateCert error %v", err)
			return
		}
		certPem := kp.Cert()

		if certPem == nil {
			t.Errorf("Empty Cert")
			return
		}
		if certPem.Type != "CERTIFICATE" {
			t.Errorf("Cert wrong type %s", certPem.Type)
			return
		}

		pkPem := kp.PrivateKey()
		if pkPem == nil {
			t.Errorf("Empty PrivateKey")
			return
		}
		if pkPem.Type != "RSA PRIVATE KEY" {
			t.Errorf("PrivateKey wrong type %s", pkPem.Type)
			return
		}
	})

}
