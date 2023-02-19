# Guard Create Knative Secrets

This Kubernetes Job is used when deploying guard out side of Knative. When using Knative the knative secrets are deployed by the Knative Operator which installs Guard as well.When not using Knative, it is up to the admin to create the secrets used by Guard. Running `create-knative-secrets` as a Job allows a simple preperation of the required secrets.

It create to secrets:

- `serving-certs-ctrl-ca` that includes the Certificate Authority Key Pair. This secret is used to create the other two secrets.
- `serving-certs-ctrl-ca-public` that includes the Certificate Authority Public Key. This secret is used by `guard-rproxy`.
- `knative-serving-certs` that includes the Certificate Authority Key Pair and the guard service Key Pair. This secret is used by `guard-service`

```bash
kubectl get secrets -n knative-serving

NAME                           TYPE     DATA   AGE
knative-serving-certs          Opaque   6      2m26s
serving-certs-ctrl-ca          Opaque   4      2m27s
serving-certs-ctrl-ca-public   Opaque   1      2m26s
```

The Job can for example be installed using: `ko apply -f ./config-production/deploy/create-knative-secrets.yaml`

After the creation of the secrets, the `./hack/copyPublicCaKey.sh <namespace>` can be used to copy the `serving-certs-ctrl-ca-public` secret to other namespaces. For example:

```bash
./hack/copyPublicCaKey.sh default

Copying secert to namespace: "default"
secret/serving-certs-ctrl-ca-public created

kubectl get secrets

NAME                           TYPE     DATA   AGE
serving-certs-ctrl-ca-public   Opaque   1      3s

```
