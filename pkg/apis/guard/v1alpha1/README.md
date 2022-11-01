# Security Data Package

This package serves as the beating heart of Guard.

It defines data structures that meet the [v1alpha1 interface](v1alpha1.go).

## The guard/v1alpha1 API

The [v1alpha1 interface](v1alpha1.go) includes three interfaces:

- Profile: describing a sample of a data type
- Pile: accumulating multiple samples of a data type to enable learning.
- Config: the rules describing what is expected from the data type.

## Core Activity

Per Sample:

1. Profile.Profile(...sample...) - Create a profile from the sample
1. Config.Decide(profile)  - Decide if it conforms to the config rules
1. Pile.Add(profile)  - Add it to a pile.

Periodically:

1. Pile.Merge(someOtherPile) - Merge someOtherPile to Pile.
1. Config.Learn(pile) - Learn a new config rules based on a pile.
1. Config.Fuse(someOtherConfig) - Fuse configs to form a new config from an old one.

Note:

- Profiles, Piles and Configs are build to be transportable across a distributed system.

## Distributed System

Guard supports working in a distributed system by allowing many instances to collect samples and take decisions.

The instances each collect piles and send them to a central service that merge the piles and learn a new config based on the enw piles and the old config.

The config is then sent back to the instances and is kept in a persistent store.
