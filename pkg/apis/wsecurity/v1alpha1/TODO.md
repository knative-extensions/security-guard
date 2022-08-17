# TODO

## Review

Items to review:

* Regarding consumed objects - choose between:
  * decide to DeepCopy always and not affect other items
  * decide to Clear other items always
  * decide to Clear other items only if we consume them such that we never reach silent data corruption
  * decide to make only some DataTypes exposed (e.g. avoid exposing low-level data types such as count and do Clear() only on exposed DataTypes
mark the other data type as "consumed" (not sure if this will make more sense than simply clearing them)
* Decide - "Would it be better to return an error instead of a string", "Take a look at the pattern here for errors... you can define an accumulator for errors like so: [knative/pkg/blob/main/apis/field_error.go](https://github.com/knative/pkg/blob/main/apis/field_error.go#L196") - discuss the options and what should be done.
* v1alpha1_test.go:

  * "for example, `go test -run Basics ./...` will come back with "no tests to run". That's probably fine for now, but may be something we want to revisit down the line."

## Implement

Items to work on:

* v1alpha1_test.go
  * "I don't understand how these permutations were arrived at".  - clarify what each test does
  * "Also, why call `.String(9)` without testing the output? I'd expect String to not change the value; if these are simply testing for non-panic, it seems like a more focused test would make more sense." - ensure all tests are both focused and the output is tested.
* Add architecture document

## Future

* Consider if go1.18 geneics can be used given that Kubernetes gengo does not seem to support generics code.
