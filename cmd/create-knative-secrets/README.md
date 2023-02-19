# Guard Create Knative Secrets

This Kubernetes Job is used when deploying guard out side of Knative. When using Knative the knative secrets are deployed using the Knative Operator which installs Guars as well.

For example it can be installed using: `ko apply -f ./config-production/deploy/create-knative-secrets.yaml`
