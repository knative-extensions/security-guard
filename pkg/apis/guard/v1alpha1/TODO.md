# TODO

## Review

Items to review:

* v1alpha1_test.go:
  * "for example, `go test -run Basics ./...` will come back with "no tests to run". That's probably fine for now, but may be something we want to revisit down the line."

## Implement

Items to work on:

* v1alpha1_test.go
  * "I don't understand how these permutations were arrived at".  - clarify what each test does
  * "Also, why call `.String(9)` without testing the output? I'd expect String to not change the value; if these are simply testing for non-panic, it seems like a more focused test would make more sense." - ensure all tests are both focused and the output is tested.
* Remove Start() from plugs if not used - could be there for historical reasons

## Future

* Consider if go1.18 geneics can be used given that Kubernetes gengo does not seem to support generics code.

## Evan architecture comments

* Think about how to accumulate and possibly reduce the data in a Pile before shipping it to the guard-service, as well as doing further reducing on the guard service.
* For the model, "Incremental Combining" from Streaming Systems is a pretty good overview, IIRC (I don't have my copy at hand at the moment, working off Google Books).
* Wire encoding, you may want to investigate formats like Protobuf, CapnProto, and Thrift.

## Security of Guard

Currently Guard has no security for communication between gate and service.
This means the potential for:

* Fake pile reports, changing the learned micro-rules of others
* Fake config requests, receiving the manual and learned micro-rules of others

Discuss what is out day 1 strategy. Can we rely on network policies and have a guard-service per security domain (namespace for example)? What are the implications for downstream?
