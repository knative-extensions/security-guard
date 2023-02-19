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

if [ -z "$1" ]
  then
    echo "Usage: "
    echo "        $0 <namespace-to-copy-secret-to> "
    exit
fi

echo Copying secert to namespace: \"$1\"
REPLACE="s/namespace: .*/namespace: ${1}/"
kubectl get secret serving-certs-ctrl-ca-public  --namespace=knative-serving -o yaml | sed "${REPLACE}" |sed  "s/selfLink: .*/ /"|sed  "s/uid: .*/ /" |sed  "s/resourceVersion: .*/ /" | kubectl apply -f -
