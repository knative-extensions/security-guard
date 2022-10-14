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

	"github.com/golang-jwt/jwt/v4"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *KubeMgr) validateToken(token string) (err error) {
	tr := authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token:     token,
			Audiences: []string{"guard-service"},
		},
	}
	tokenReview, err := k.cmClient.AuthenticationV1().TokenReviews().Create(context.TODO(), &tr, metav1.CreateOptions{})
	if err != nil {
		err = fmt.Errorf("tokenreviews failed %w", err)
		return
	}
	if !tokenReview.Status.Authenticated {
		err = fmt.Errorf("not Authenticated")
		return
	}
	return
}
func (k *KubeMgr) parseJwt(token string) (podname string, ns string, err error) {
	var ok bool

	jwtToken, _, jwtTokenErr := new(jwt.Parser).ParseUnverified(token, jwt.MapClaims{})
	if jwtTokenErr != nil {
		err = fmt.Errorf("failed to parse jwtToken %w", jwtTokenErr)
		return
	}
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		err = fmt.Errorf("claims.(jwt.MapClaims) failed")
		return
	}
	data, ok := claims["kubernetes.io"].(map[string]interface{})
	if !ok {
		err = fmt.Errorf("claims[kubernetes.io].(map[string]interface{}) failed")
		return
	}
	ns, ok = data["namespace"].(string)
	if !ok {
		err = fmt.Errorf("claims[kubernetes.io][namespace].(string) failed")
		return
	}
	if ns == "" {
		err = fmt.Errorf("claims[kubernetes.io][namespace] empty")
		return
	}
	podData, ok := data["pod"].(map[string]interface{})
	if !ok {
		err = fmt.Errorf("claims[kubernetes.io][pod].(map[string]interface{} failed")
		return
	}
	podname, ok = podData["name"].(string)
	if !ok {
		err = fmt.Errorf("claims[kubernetes.io][pod][name].(string) failed")
		return

	}
	if podname == "" {
		err = fmt.Errorf("claims[kubernetes.io][pod][name] empty")
		return
	}
	return
}

func (k *KubeMgr) getPodData(podname string, ns string) (sid string, err error) {
	var ok bool

	pod, err := k.cmClient.CoreV1().Pods(ns).Get(context.TODO(), podname, metav1.GetOptions{})
	if err != nil {
		err = fmt.Errorf("cant get pod data from kubeAPI %w", err)
		return
	}
	if sid, ok = pod.Labels["serving.knative.dev/service"]; !ok {
		if sid, ok = pod.Labels["app"]; !ok {
			sid = podname
		}
	}
	return
}

func (k *KubeMgr) TokenData(token string) (sid string, ns string, err error) {
	var podname string

	// stage 1 - first validate the jwt using kubeAPI
	err = k.validateToken(token)
	if err != nil {
		return
	}

	// stage 2 - parse the now validated jwt and obtain namespace and podname
	podname, ns, err = k.parseJwt(token)
	if err != nil {
		return
	}

	// stage 3 - get pod data from kubeAPI - extract the service/app name
	sid, err = k.getPodData(podname, ns)
	return
}
