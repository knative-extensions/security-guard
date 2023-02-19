# Guard on Vanilla Kubernetes

Please refer to [An Opinionated Kubernetes](https://davidhadas.wordpress.com/2022/08/29/knative-an-opinionated-kubernetes/) to learn why in many cases, Knative should be your preferred path for deploying web services over Kubernetes rather than deploying directly on vanilla Kubernetes.

For direct deployment of guard on Kubernetes, [guard-gate](cmd/guard-service) needs to be deployed in the service pods as a reverse proxy sidecar and the pod needs to be configured to pass ingress requests via the guard sidecar. See [documentation of guard-rproxy - the guard sidecar](./cmd/guard-rproxy/README.md).

For non-production, development only environments see:

- [guard-service deployment](./config-dev-only/deploy/guard-service.yaml)
- [secure-helloworld example app deployment](./config-dev-only/deploy/secured-helloworld.yaml)
- If you wish to use guard not in a sidecar but in its own pod, see [secure-layered-myapp example app deployment](./config-dev-only/deploy/secure-layered-myapp.yaml)

A recommended first step is deploying production guard and the examples using Kind using the [installation script](./hack/kind/deployProduction.sh) The installation scripts shows a working example of using guard in a development environment.

For production environments see:

- [guard-service deployment](./config-production/deploy/guard-service.yaml)
- [secure-helloworld example app deployment](./config-production/deploy/secured-helloworld.yaml)
- If you wish to use guard not in a sidecar but in its own pod, see [secure-layered-myapp example app deployment](./config-production/deploy/secure-layered-myapp.yaml)

When using in production, it is necessary to ensure that:

 - [guard-service](cmd/guard-service) is installed in namespace `knative-serving` [guard-rproxy](./cmd/guard-rproxy/README.md) will by default look for guard-service in this namespace. If you are using a different namespace, you need to setup [guard-rproxy](./cmd/guard-rproxy/README.md) to look for [guard-service](cmd/guard-service) elsewhere.
 - [guard-service](cmd/guard-service) namespace includes a secret that contains its certificate authority key pair and service key pair. You can run [create-knative-secrets](./cmd/create-knative-secrets/README.md) as a Job when deploying [guard-service](cmd/guard-service) to create the secret for you.
- [guard-rproxy](./cmd/guard-rproxy/README.md) runs as a sidecar as part of your service pods. Therefroe it may run in any namespace. In order to be able to securely communicate with [guard-service](cmd/guard-service), [guard-rproxy](./cmd/guard-rproxy/README.md) need to access then `serving-certs-ctrl-ca` secret that contains its certificate authority public key. You can run [create-knative-secrets](./cmd/create-knative-secrets/README.md) as a Job when deploying [guard-service](cmd/guard-service) to create the `serving-certs-ctrl-ca` secret in the `knative-serving` namespace. Once created, you may use copy the secret to any namespace where you deploy services that need to be protected using the [guard-rproxy](./cmd/guard-rproxy/README.md). You may also use [the `copyPublicCaKey` script] (./hack/copyPublicCaKey.sh) to ease your work.

A recommended first step is deploying production guard and the examples using Kind using the [installation script](./hack/kind/deployProduction.sh) The installartion scripts shows a working example of using guard in production.
