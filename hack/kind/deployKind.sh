#!/usr/bin/env bash

# Copyright 2022 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Create Kind cluster
kind delete cluster --name k8s
kind create cluster --config ./hack/kind/kind-config.yaml
kubectl cluster-info --context kind-k8s
kubectl create namespace knative-serving
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

#Create K8s resources CRD, ServiceAccounts etc.
kubectl apply -Rf ./config/resources/
export KO_DOCKER_REPO=ko.local

# create and load create-knative-secrets image
CS_IMAGE=`ko build ko://knative.dev/security-guard/cmd/create-knative-secrets -B  `
kind load docker-image $CS_IMAGE --name k8s

# create and load guard-rproxy image
GR_IMAGE=`ko build ko://knative.dev/security-guard/cmd/guard-rproxy -B  `
kind load docker-image $GR_IMAGE --name k8s
docker tag $GR_IMAGE guard-rproxy
kind load docker-image guard-rproxy --name k8s

# create and load guard-service image
GS_IMAGE=`ko build ko://knative.dev/security-guard/cmd/guard-service -B  `
kind load docker-image $GS_IMAGE --name k8s

# start create-secrets
ko apply -f ./config-kubernetes/deploy/create-secrets.yaml -B

# start guard-service
ko apply -f ./config/deploy/guard-service.yaml -B

# wait for keys to be ready
kubectl wait --namespace knative-serving --for=condition=complete job/create-knative-secrets --timeout=120s

# copy the secret with the ca key to the default namespace
./hack/copyCerts.sh default

# wait for ingress to be ready
kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller  --timeout=120s

#add hellowworld - protected using a guard sidecar  (the recommended pattern)
ko apply -f ./config-kubernetes/deploy/secured-helloworld.yaml -B

#add myapp - protected using a separate guard pod (non-recommended pattern)
ko apply -f ./config-kubernetes/deploy/secured-layered-myapp.yaml -B
