# Guard Service

guard-service is the he Guard backend service which is used for:

1. Learning per-service micro-rules from piles of profiles sent by [guard-gate](../../pkg/guard-gate)
1. Constructing and storing per service Guardians
1. Caching Guardians and servicing [guard-gate](../../pkg/guard-gate) requests for Guardians

Guardians are based on the [guard.security.knative.dev](../../pkg/apis/guard/v1alpha1/README.md) package.

To access Guardians, guard-service uses the [guard-kubemgr](../guard-kubemgr/README.md) package.

See [Guard Architecture](/ARCHITECTURE.md) to learn about how Guard process and learn internally security data.
