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
ROOTCA="$(mktemp)"
FILENAME=`basename $ROOTCA`
kubectl get secret -n knative-serving knative-serving-certs -o json| jq -r '.data."ca-cert.pem"' | base64 -d >  $ROOTCA

echo "Create a temporary config-deployment configmap with the certificate"
CERT=`kubectl create cm config-deployment --from-file $ROOTCA -o json --dry-run=client |jq .data.\"$FILENAME\"`

echo "Get the current config-deployment configmap"
CURRENT="$(mktemp)"
kubectl get cm config-deployment -n knative-serving -o json | jq 'del(.data, .binaryData | ."queue-sidecar-token-audiences", ."queue-sidecar-rootca" )' > $CURRENT

echo "Add queue-sidecar-token-audiences"
AUDIENCES="$(mktemp)"
jq '.data |= . + { "queue-sidecar-token-audiences": "guard-service"}' $CURRENT > $AUDIENCES

echo "Join the two config-deployment configmaps into one"
MERGED="$(mktemp)"
jq  --arg cert "${CERT}" '.data  |= . + { "queue-sidecar-rootca": $cert}' $AUDIENCES > $MERGED

echo "Apply the joined config-deployment configmap"
kubectl apply -f $MERGED -n knative-serving

echo "cleanup"
rm $MERGED $AUDIENCES $ROOTCA $CURRENT

echo "Results:"
kubectl get cm config-deployment -n knative-serving -o json|jq '.data'
