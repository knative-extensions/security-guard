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

set -o errexit
set -o nounset
set -o pipefail

# Create clean environment
rm -rf cert
mkdir cert && cd cert

cat  <<EOF >ca.config
[ req ]
default_bits           = 2048
default_keyfile        = rootCACert.pem
distinguished_name     = req_guard_ca
prompt                 = no

[ req_guard_ca ]
CN                     = security-guard-ca
EOF


cat <<EOF >guard.config
[req]
req_extensions = v3_req
distinguished_name = dn
prompt = no

[dn]
CN = security-guard.default

[v3_req]
subjectAltName = DNS:guard-service.default,DNS:guard-service.default.cluster.local
EOF

cat << EOF >guard.v3-ext
subjectAltName = DNS:guard-service.default,DNS:guard-service.default.cluster.local
EOF

openssl version

# Create CA certificate
# Create private key
openssl genrsa -out ca-key.pem 2048

# Create ca certificate
openssl req -x509 -new -nodes -key ca-key.pem -days 3650 -out ca-cert.pem -config ca.config

# Certificate Verbose
echo "CA Certificate"
openssl x509 -in ca-cert.pem -text
echo

# Create server csr
# Create private key
openssl genrsa -out guard-service-key.pem 2048

# Create secuity-guard certificate
openssl req -new -key guard-service-key.pem -sha256 -out guard-service.csr -config guard.config

# CSR Verbose
echo "Guard Service CSR"
openssl req -in guard-service.csr -noout -text
echo

# Create guard certificate
openssl x509 -req -sha256 -in guard-service.csr -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out guard-service-cret.pem -days 365 -extfile guard.v3-ext

# Certificate Verbose
echo "Guard Service Certificate"
openssl x509 -in guard-service-cret.pem -text
echo

# Create guard tls secret
kubectl delete secret guard-service-tls
kubectl create secret tls guard-service-tls \
  --cert=guard-service-cret.pem  \
  --key=guard-service-key.pem

# RootCA Verbose
echo "Guard Service Certificate"
kubectl get configmap guard-rootca -o yaml
echo

# Create RootCA for guard-gates
kubectl delete configmap guard-rootca
kubectl create configmap guard-rootca --from-file=ca-cert.pem

# RootCA Verbose
echo "Guard Service Certificate"
kubectl get configmap guard-rootca -o yaml
echo
