# Guard Service

guard-service is the he Guard backend service which is used for:

1. Learning per-service micro-rules from piles of profiles sent by [guard-gate](../../pkg/guard-gate)
1. Constructing and storing per service Guardians
1. Caching Guardians and servicing [guard-gate](../../pkg/guard-gate) requests for Guardians

Guardians are based on the [guard.security.knative.dev](../../pkg/apis/guard/v1alpha1/README.md) package.

To access Guardians, guard-service uses the [guard-kubemgr](../guard-kubemgr/README.md) package.

See [Guard Architecture](/ARCHITECTURE.md) to learn about how Guard process and learn internally security data.

## Security

To secure the current version of guard-service, the guard-service must be deployed on the same trust domain as the set of services it supports. One possible configuration is to deploy a security-guard in the same namespace as the deployed services and ensure network policy prohibits any external communication to/from the guard-service.

Do not open guard-service to the Internet, allow only local trusted services to communicate with the guard-service.

Always review the set of micro-rules produced by guard-service before moving to a production environment and if you decide to use guard-service in a production environment, it is safer to use manual microrules and treat the microrules produced by guard-service as a recommendation for human review.
