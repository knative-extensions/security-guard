# Guard Kube Manager

This package stores and controls Guardians via KubeApi

Guardians which are composed of a set of micro-rules are used to control a [guard-gate](../guard-gate/README.md).

Guardians are stored either as a CRD or in a ConfigMap depending on the system used. Guardians are based on the [guard.security.knative.dev](../apis/guard/v1alpha1/README.md) package.

This package exports methods for reading and setting Guardians using either a ConfigMap or a Guardian CRD.
