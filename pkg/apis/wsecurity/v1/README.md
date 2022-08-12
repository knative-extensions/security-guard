# Security Data Package

This package serves as the beating hurt of Guard. 

It defines data structures that meet the [v1 interface](v1.go).

The [v1 interface](v1.go) includes three interfaces:

- Profile: describing a sample of a data type
- Pile: accumulating multiple samples of a data type to enable learning.
- Config: the rules describing what is expected from the data type.
