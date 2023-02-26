# Guard Create Knative Secrets

This Kubernetes Job is used when deploying guard out side of Knative. When using Knative the knative secrets are deployed by the Knative Operator which installs Guard as well.When not using Knative, it is up to the admin to create the secrets used by Guard. Running `create-knative-secrets` as a Job allows a simple preparation of the required secrets.

It create secrets:

- `serving-certs-ctrl-ca` that includes the Certificate Authority Key Pair. This secret is used to create the other two secrets.
- `knative-serving-certs` that includes the Certificate Authority Key Pair and the guard service Key Pair. This secret is used by `guard-service`.

```bash
kubectl get secrets -n knative-serving

NAME                           TYPE     DATA   AGE
knative-serving-certs          Opaque   6      2m26s
serving-certs-ctrl-ca          Opaque   4      2m27s
```

The Job can for example be installed using: `ko apply -f ./config-kubernetes/deploy/create-knative-secrets.yaml`

After the creation of the secrets, the `./hack/copyCerts.sh <namespace>` can be used to copy the `knative-serving-certs` secret to other namespaces. The new seceret is named `<namespace>-serving-certs` For example:

```bash
./hack/copyCerts.sh test

Copying secert to namespace: "test"
secret/test-serving-certs created

kubectl get secrets

NAME                           TYPE     DATA   AGE
test-serving-certs             Opaque   6      3s

```
