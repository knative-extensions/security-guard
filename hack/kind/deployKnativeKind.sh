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

export KO_DOCKER_REPO=ko.local

# Knative install using quickstart
kn quickstart kind -n k8s --install-serving

#Create K8s resources CRD, ServiceAccounts etc.
kubectl apply -f https://raw.githubusercontent.com/knative-sandbox/security-guard/release-0.5/config/resources/gateAccount.yaml
kubectl apply -f https://raw.githubusercontent.com/knative-sandbox/security-guard/release-0.5/config/resources/serviceAccount.yaml
kubectl apply -f https://raw.githubusercontent.com/knative-sandbox/security-guard/release-0.5/config/resources/guardiansCrd.yaml

# Kind seem to sometime need some extra time
sleep 10

# adjust knative to use guard
kubectl apply -f https://github.com/knative-sandbox/security-guard/releases/download/v0.5.0/queue-proxy.yaml
kubectl apply -f https://github.com/knative-sandbox/security-guard/releases/download/v0.5.0/config-features.yaml

# start guard-service
kubectl apply -f https://github.com/knative-sandbox/security-guard/releases/download/v0.5.0/guard-service.yaml

# Activate internal encryption
kubectl patch configmap config-network -n knative-serving --type=merge -p '{"data": {"internal-encryption": "true"}}'

# Restart activator pod
kubectl rollout restart deployment activator -n knative-serving
