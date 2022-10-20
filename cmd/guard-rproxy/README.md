# Guard Reverse Proxy Sidecar

guard-rproxy is a reverse proxy embedded with a guard-gate and packed as a container image. The container image can than be used:

1. As a sidecar while deploying a Kubernetes microservice. This is the recommended mode of operation and offers both client request monitoring and control and microservice pod monitoring and control.

1. As a standalone exposed Pod in from of a protected unexposed microservice. This mode is simple to try out. It offers client request monitoring and control but does not offer microservice pod monitoring and control.

## Installing Security-Guard

Security-Guard includes automation for auto-learning a per service Guardian.
Auto-learning requires you to deploy a `guard-service` on your kubernetes cluster.
`guard-service` should be installed in any namespace where you deploy knative services that require Security-Guard protection.

### Install from source

1. Clone the Security-Guard repository using `git clone git@github.com:knative-sandbox/security-guard.git`
1. Do `cd security-guard`
1. Run `ko apply -Rf ./config/resources/`
1. Run `ko apply -Rf ./config/deploy/guard-service.yaml`

### Install from released images and yamls

    Use released images to update your system to enable Security-Guard:

1. Add the necessary Security-Guard resources to your cluster using:

        kubectl apply -f https://raw.githubusercontent.com/knative-sandbox/security-guard/release-0.1/config/resources/gateAccount.yaml
        kubectl apply -f https://raw.githubusercontent.com/knative-sandbox/security-guard/release-0.1/config/resources/serviceAccount.yaml
        kubectl apply -f https://raw.githubusercontent.com/knative-sandbox/security-guard/release-0.1/config/resources/guardiansCrd.yaml

1. Deploy `guard-service` on your system to enable automated learning of micro-rules. In the current version, it is recommended to deploy `guard-service` in any namespace where knative services are deployed.

An easy way to do that is using:

    kubectl apply -f https://github.com/knative-sandbox/security-guard/releases/download/v0.1.2/guard-service.yaml

## Deploying a pod with a Security-Guard sidecar

Use the following example yaml to deploy an example helloworld container with a guard sidecar:

    kubectl apply -f https://github.com/knative-sandbox/security-guard/releases/download/v0.1.2/secured-helloworld.yaml

Security alerts can be seen in the  `guard-rproxy` container of the `secured-helloworld` pod using:

    kubectl logs deployment/secured-helloworld guard-rproxy -f

## Deploying a separate Security-Guard Pod

Use the following example yaml to deploy one example `myapp` pod that include a container running helloworld, (this pod is not exposed outside of the cluster) and one `myapp-guard` pod that include a guard container to expose the myapp service outside the cluster while performing security-behavior monitoring and control on client requests:

    kubectl apply -f https://github.com/knative-sandbox/security-guard/releases/download/v0.1.2/secured-layered-myapp.yaml

Security alerts can be seen in the `myapp-guard` pod using:

    kubectl logs deployment/myapp-guard -f
