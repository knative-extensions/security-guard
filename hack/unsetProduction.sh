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


# Unset the ROOT_CA
echo "Remove ROOT_CA from config-deployment configmap"
kubectl patch cm config-deployment -n knative-serving -p '{"data":{"queue-sidecar-rootca": ""}}'

echo "Results:"
kubectl get cm config-deployment -n knative-serving -o json|jq '.data'
