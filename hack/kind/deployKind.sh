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

CONFIG="$(mktemp)"
cat <<EOF > $CONFIG
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: k8s
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
EOF



# Create Kind cluster
kind delete cluster --name k8s
kind create cluster --config $CONFIG
kubectl cluster-info --context kind-k8s
kubectl create namespace knative-serving
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

#Create K8s resources CRD, ServiceAccounts etc.
kubectl apply -f https://raw.githubusercontent.com/knative-sandbox/security-guard/release-0.5/config/resources/gateAccount.yaml
kubectl apply -f https://raw.githubusercontent.com/knative-sandbox/security-guard/release-0.5/config/resources/serviceAccount.yaml
kubectl apply -f https://raw.githubusercontent.com/knative-sandbox/security-guard/release-0.5/config/resources/guardiansCrd.yaml

# start create-secrets
kubectl apply -f https://github.com/knative-sandbox/security-guard/releases/download/v0.5.0/create-secrets.yaml

# start guard-service
kubectl apply -f https://github.com/knative-sandbox/security-guard/releases/download/v0.5.0/guard-service.yaml

# wait for keys to be ready
kubectl wait --namespace knative-serving --for=condition=complete job/create-knative-secrets --timeout=120s

# wait for ingress to be ready
kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller  --timeout=120s

# Copying secert to namespace: \"default\"
REPLACE_NS="s/ namespace: .*/ namespace: default/"
REPLACE_NAME="s/ name: knative-serving-certs/ name: default-serving-certs/"
kubectl get secret knative-serving-certs --namespace=knative-serving -o yaml |sed "${REPLACE_NS}" |sed "${REPLACE_NAME}" |sed  "s/ selfLink: .*/ /"|sed  "s/ uid: .*/ /" |sed  "s/ resourceVersion: .*/ /" |kubectl apply -f -

#add hellowworld - protected using a guard sidecar  (the recommended pattern)
kubectl apply -f https://github.com/knative-sandbox/security-guard/releases/download/v0.5.0/secured-helloworld.yaml

#add myapp - protected using a separate guard pod (non-recommended pattern)
kubectl apply -f https://github.com/knative-sandbox/security-guard/releases/download/v0.5.0/secured-layered-myapp.yaml

#cleanup
rm $CONFIG
