# TODO - Future

* Consider if go1.18 generics can be used given that Kubernetes gengo does not seem to support generics code.
* Think about how to accumulate and possibly reduce the data in a Pile before shipping it to the guard-service, as well as doing further reducing on the guard service.
* For the model, "Incremental Combining" from Streaming Systems is a pretty good overview, IIRC (I don't have my copy at hand at the moment, working off Google Books).
* Wire encoding, investigate formats like Protobuf, CapnProto, and Thrift.
