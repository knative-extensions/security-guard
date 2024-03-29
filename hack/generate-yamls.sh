#! /usr/bin/env bash
#
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
#
# This script builds all the YAMLs that Knative security-advisor publishes.
# It may be varied between different branches, of what it does, but the
# following usage must be observed:
#
# generate-yamls.sh  <repo-root-dir> <generated-yaml-list>
#     repo-root-dir         the root directory of the repository.
#     generated-yaml-list   an output file that will contain the list of all
#                           YAML files. The first file listed must be our
#                           manifest that contains all images to be tagged.

# Different versions of our scripts should be able to call this script with
# such assumption so that the test/publishing/tagging steps can evolve
# differently than how the YAMLs are built.

# The following environment variables affect the behavior of this script:
# * `$KO_FLAGS` Any extra flags that will be passed to ko.
# * `$YAML_OUTPUT_DIR` Where to put the generated YAML files, otherwise a
#   random temporary directory will be created. **All existing YAML files in
#   this directory will be deleted.**
# * `$KO_DOCKER_REPO` If not set, use ko.local as the registry.

set -o errexit
set -o pipefail

readonly YAML_REPO_ROOT=${1:?"First argument must be the repo root dir"}
readonly YAML_LIST_FILE=${2:?"Second argument must be the output file"}
readonly YAML_ENV_FILE=${3:-$(mktemp)}

# Set output directory
if [[ -z "${YAML_OUTPUT_DIR:-}" ]]; then
  readonly YAML_OUTPUT_DIR="$(mktemp -d)"
fi
rm -fr ${YAML_OUTPUT_DIR}/*.yaml

# Generated Knative component YAML files
readonly SECURED_HELLO_YAML=${YAML_OUTPUT_DIR}/secured-helloworld.yaml
readonly SECURED_LAYERED_MYAPP_YAML=${YAML_OUTPUT_DIR}/secured-layered-myapp.yaml
readonly TESTSRV_YAML=${YAML_OUTPUT_DIR}/testsrv.yaml
readonly CREATE_SECRETS_YAML=${YAML_OUTPUT_DIR}/create-secrets.yaml
readonly CONFIG_FEATURES_YAML=${YAML_OUTPUT_DIR}/config-features.yaml
readonly GUARD_SERVICE_YAML=${YAML_OUTPUT_DIR}/guard-service.yaml
readonly QUEUE_PROXY_YAML=${YAML_OUTPUT_DIR}/queue-proxy.yaml
readonly GATE_ACCOUNT_YAML=${YAML_OUTPUT_DIR}/gate-account.yaml
readonly SERVICE_ACCOUNT_YAML=${YAML_OUTPUT_DIR}/service-account.yaml
readonly GUARDIAN_CRD_YAML=${YAML_OUTPUT_DIR}/guardian-crd.yaml
readonly DEPLOY_KIND=${YAML_OUTPUT_DIR}/deploy-kind.sh
readonly DEPLOY_KNATIVE_KIND=${YAML_OUTPUT_DIR}/deploy-knative-kind.sh

# Flags for all ko commands
KO_YAML_FLAGS="-P"
KO_FLAGS="${KO_FLAGS:-}"
[[ "${KO_DOCKER_REPO}" != gcr.io/* ]] && KO_YAML_FLAGS=""

if [[ "${KO_FLAGS}" != *"--platform"* ]]; then
  KO_YAML_FLAGS="${KO_YAML_FLAGS} --platform=all"
fi

readonly KO_YAML_FLAGS="${KO_YAML_FLAGS} ${KO_FLAGS}"

if [[ -n "${TAG:-}" ]]; then
  LABEL_YAML_CMD=(sed -e "s|serving.knative.dev/release: devel|serving.knative.dev/release: \"${TAG}\"|" -e "s|app.kubernetes.io/version: devel|app.kubernetes.io/version: \"${TAG:1}\"|")
else
  LABEL_YAML_CMD=(cat)
fi

: ${KO_DOCKER_REPO:="ko.local"}
export KO_DOCKER_REPO

cd "${YAML_REPO_ROOT}"

echo "Building Knative Secuity-Guard"
echo KO_YAML_FLAGS: ${KO_YAML_FLAGS}
ko resolve ${KO_YAML_FLAGS} -f config-kubernetes/deploy/secured-helloworld.yaml | "${LABEL_YAML_CMD[@]}" > "${SECURED_HELLO_YAML}"
ko resolve ${KO_YAML_FLAGS} -f config-kubernetes/deploy/secured-layered-myapp.yaml | "${LABEL_YAML_CMD[@]}" > "${SECURED_LAYERED_MYAPP_YAML}"
ko resolve ${KO_YAML_FLAGS} -f test/e2e/services/httptest/deploy.yaml | "${LABEL_YAML_CMD[@]}" > "${TESTSRV_YAML}"
ko resolve ${KO_YAML_FLAGS} -f config-kubernetes/deploy/create-secrets.yaml | "${LABEL_YAML_CMD[@]}" > "${CREATE_SECRETS_YAML}"
ko resolve ${KO_YAML_FLAGS} -f config/deploy/config-features.yaml | "${LABEL_YAML_CMD[@]}" > "${CONFIG_FEATURES_YAML}"
ko resolve ${KO_YAML_FLAGS} -f config/deploy/guard-service.yaml | "${LABEL_YAML_CMD[@]}" > "${GUARD_SERVICE_YAML}"
ko resolve ${KO_YAML_FLAGS} -f config/deploy/queue-proxy.yaml | "${LABEL_YAML_CMD[@]}" > "${QUEUE_PROXY_YAML}"
ko resolve ${KO_YAML_FLAGS} -f config/resources/gateAccount.yaml | "${LABEL_YAML_CMD[@]}" > "${GATE_ACCOUNT_YAML}"
ko resolve ${KO_YAML_FLAGS} -f config/resources/serviceAccount.yaml | "${LABEL_YAML_CMD[@]}" > "${SERVICE_ACCOUNT_YAML}"
ko resolve ${KO_YAML_FLAGS} -f config/resources/guardiansCrd.yaml | "${LABEL_YAML_CMD[@]}" > "${GUARDIAN_CRD_YAML}"
cp hack/kind/deployKind.sh "${DEPLOY_KIND}"
cp hack/kind/deployKnativeKind.sh "${DEPLOY_KNATIVE_KIND}"
echo "All manifests generated"

# List generated YAML files

cat << EOF > ${YAML_LIST_FILE}
${SECURED_HELLO_YAML}
${SECURED_LAYERED_MYAPP_YAML}
${TESTSRV_YAML}
${CREATE_SECRETS_YAML}
${CONFIG_FEATURES_YAML}
${GUARD_SERVICE_YAML}
${QUEUE_PROXY_YAML}
${GATE_ACCOUNT_YAML}
${SERVICE_ACCOUNT_YAML}
${GUARDIAN_CRD_YAML}
${DEPLOY_KIND}
${DEPLOY_KNATIVE_KIND}
EOF

cat << EOF > "${YAML_ENV_FILE}"
export SECURED_HELLO_YAML=${SECURED_HELLO_YAML}
export SECURED_LAYERED_MYAPP_YAML=${SECURED_LAYERED_MYAPP_YAML}
export TESTSRV_YAML=${TESTSRV_YAML}
export CREATE_SECRETS_YAML=${CREATE_SECRETS_YAML}
export CONFIG_FEATURES_YAML=${CONFIG_FEATURES_YAML}
export GUARD_SERVICE_YAML=${GUARD_SERVICE_YAML}
export QUEUE_PROXY_YAML=${QUEUE_PROXY_YAML}
export GATE_ACCOUNT_YAML=${GATE_ACCOUNT_YAML}
export SERVICE_ACCOUNT_YAML=${SERVICE_ACCOUNT_YAML}
export GUARDIAN_CRD_YAML=${GUARDIAN_CRD_YAML}
export DEPLOY_KIND=${DEPLOY_KIND}
export DEPLOY_KNATIVE_KIND=${DEPLOY_KNATIVE_KIND}
EOF
