# Queue Proxy Option

This package includes glue code needed to attach a security plug such as:

- [guard-gate](../guard-gate)
- [test-gate](../test-gate)

as an option (extension) to Knative queue.

The package reads the service annotations from annotations file in the podInfo volume mounted by Queue Proxy. The annotations indicate if the security plug need to be activated and provide config parameters to the security plug.

The package then interact with the [pluginterfaces](../pluginterfaces) package and the respective security gate to ensure they are properly initialized and may start serving the requests, responses and global queue proxy context.

## Using Plugs

This package enables using security plugs with Queue Proxy by following these steps:

1. Replace cmd/queue/main.go of [serving](https://github.com/knative/serving) with the code as described below.
1. Create a new Queue Proxy Image
1. Store the new Queue Proxy Image in an image repository
1. Configure your cluster to use the new Queue Proxy Image

In order to activate [guard-gate](../guard-gate) replace cmd/queue/main.go of [serving](https://github.com/knative/serving) with the following code:

```go
package main

import "os"

import (
    "knative.dev/serving/pkg/queue/sharedmain"
    "github.com/knative-sandbox/security-guard/pkg/qpoption"
    _ "github.com/knative-sandbox/security-guard/pkg/guard-gate"
)

func main() {
    qOpt := qpoption.NewGateQPOption()
    defer qOpt.Shutdown()
    
    if sharedmain.Main(qOpt.Setup) != nil {
      qOpt.Shutdown()
      os.Exit(1)
    }
} 

```

In order to activate [test-gate](../test-gate) replace cmd/queue/main.go of [serving](https://github.com/knative/serving) with the following code:

```go
package main

import "os"

import (
    "knative.dev/serving/pkg/queue/sharedmain"
    "github.com/knative-sandbox/security-guard/pkg/qpoption"
    _ "github.com/knative-sandbox/security-guard/pkg/test-gate"
)

func main() {
    qOpt := qpoption.NewGateQPOption()
    defer qOpt.Shutdown()
    
    if sharedmain.Main(qOpt.Setup) != nil {
      qOpt.Shutdown()
      os.Exit(1)
    }
} 

```
