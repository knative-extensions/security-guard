# TestGate

This plug includes a minimal RoundTripper gate that can be used for **E2E testing**.

The plug accepts two configuration parameters:

* SENDER - a string indicating a sender name (default is "someone")
* RESPONSE - a string indicating the response to send (default is "CU")

If the gate sees a request header of `x-testgate-hi`

1. The gate will log: `hehe, <SENDER> noticed me!`
2. The gate will add the following response header `x-testgate-bye: <RESPONSE>`

This plug is not meant to be used in production.

## Using Plugs

Using this plug depends on the system where your service runs. When using Knative, use [qpoption](../qpoption) to attache a plug to Queue Proxy.
