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


# Set the ROOT_CA and token audiences

echo "Copy the certificate to file"
kubectl get secret -n knative-serving knative-serving-certs -o json| jq -r '.data."ca-cert.pem"' | base64 -d >  queue-sidecar-rootca

echo "Create a temporary config-deployment configmap with the certificate"
kubectl create cm config-deployment --from-file queue-sidecar-rootca -o json --dry-run=client> temp.json

echo "Get the current config-deployment configmap"
kubectl get cm config-deployment -n knative-serving -o json | jq 'del(.data, .binaryData | ."queue-sidecar-token-audiences", ."queue-sidecar-rootca" )' > config-deployment.json

echo "Add queue-sidecar-token-audiences"
jq '.data |= . + { "queue-sidecar-token-audiences": "guard-service"}'  config-deployment.json > audiences.json

echo "Join the two config-deployment configmaps into one"
jq -s '.[0] * .[1]' audiences.json temp.json > joined.json

echo "Apply the joined config-deployment configmap"
kubectl apply -f joined.json -n knative-serving

echo "cleanup"
rm joined.json audiences.json queue-sidecar-rootca temp.json config-deployment.json

echo "Results:"
kubectl get cm config-deployment -n knative-serving -o json|jq '.data'
