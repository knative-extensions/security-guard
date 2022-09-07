#!/usr/bin/env bash 

# Copyright 2017 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

# echo "Generating clientset for ${GROUPS_WITH_VERSIONS} at ${OUTPUT_PKG}/${CLIENTSET_PKG_NAME:-clientset}"
GOBIN="$(go env GOBIN)"
gobin="${GOBIN:-$(go env GOPATH)/bin}"
gosrc="${GOBIN:-$(go env GOPATH)/src}"
projfullpath="$(cd ${SCRIPT_ROOT}; pwd)"
boilerplate="${projfullpath}/scripts/boilerplate.go.txt"

proj="knative.dev/security-guard"
pkgapis="${proj}/pkg/apis/wsecurity/v1alpha1"
outpack="${proj}/pkg/generated/clientset" 

echo "Generating deepcopy funcs"
"${gobin}/deepcopy-gen" \
  --input-dirs $pkgapis \
  --go-header-file  "${boilerplate}" \
  -O zz_generated.deepcopy "$@"

echo "Generating clientset"
"${gobin}/client-gen" \
  --input-base "" \
  --input $pkgapis \
  --output-base "$gosrc" \
  --output-package "$outpack" \
  --go-header-file  "${boilerplate}" \
  --clientset-name "guardians" \
