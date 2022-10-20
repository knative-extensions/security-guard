# Guard on Vanilla Kubernetes

Please refer to [An Opinionated Kubernetes](https://davidhadas.wordpress.com/2022/08/29/knative-an-opinionated-kubernetes/) to learn why Knative should be your preferred path for deploying web services over Kubernetes rather than deploying directly on Kubernetes.

For direct deployment on Kubernetes, Guard needs to be deployed in the service pods as a reverse proxy sidecar and the pod needs to be configured to pass ingress requests via the guard sidecar.

See [documentation and a sidecar image](./cmd/guard-rproxy/README.md).
