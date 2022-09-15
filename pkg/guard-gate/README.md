# Guard Gate

guard-gate is a go package that can be attached to a go reverse proxy in order to extract profiles about requests, responses and the pod it is running at.

Each profile is compared to the criteria defined in the Guardian allowing the guard-gate to alert about misbehavior or block misbehavers.

Additionally, profiles are piled together and sent to the [guard-service](../../cmd/guard-service/README.md) to enable it to learn new Guardians.

To access Guardians, guard-gate uses either the [guard-service](../../cmd/guard-service/README.md) or the [guard-kubemgr](../guard-kubemgr/README.md) package.

See [Guard Architecture](/ARCHITECTURE.md) to learn about how Guard process and learn internally security data.
