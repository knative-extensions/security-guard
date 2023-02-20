# Guard Reverse Proxy Sidecar

guard-rproxy is a reverse proxy embedded with a guard-gate and packed as a container image. The container image can than be used:

1. As a sidecar while deploying a Kubernetes microservice. This is the recommended mode of operation and offers both client request monitoring and control and microservice pod monitoring and control.

1. As a standalone exposed Pod protecting an unexposed microservice. This mode is simple to try out. It offers client request monitoring and control but does not offer microservice pod monitoring and control.

## Environment Variables

| variable | meaning | default |
| -------- | ------- | --------|
| SERVICE_NAME | Unique name given to the service  - used also as the guardian name | (required) |
| NAMESPACE | namespace used | (required)  |
| SERVICE_URL | The url where the service we protect can be reached | (required) |
| USE_CRD | if true crd is used, if false configmap is used instead | false |
| MONITOR_POD | if true the pod is monitored (sidecar use case) | false |
| GUARD_URL | the url of the guard-service | "http://guard-service.knative-serving" |
| LOG_LEVEL | the log level | info |
| GUARD_PROXY_PORT | meaning | default |
| GuardProxyPort | the port exposed by the pod | 22000 |
| POD_MONITOR_INTERVAL | time interval for monitoring the pod | 5s |
| GUARDIAN_SYNC_INTERVAL | tim interval to sync with guard-service | 60s |
