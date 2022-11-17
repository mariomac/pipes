# CHANGELOG

# v0.NEXT
* High-level API nodes do not need to implement `Instancer` if you define a `nodeId` tag in the
  config struct that defines them.

## Breaking changes
* High-level graph API add contexts to the `graph.Run(context.Context)` library, and
* Start providers in High-level API must return a `StartFuncCtx[OUT]` function instead of a
  `StartFunc[OUT]`
* In version 1.0 maybe StartFunc is replaced by StartFuncCtx and we force the usage of contexts always

## TO DO
* Allow also an `instanceId` field inside the struct so any string or stringer can be marked
as InstanceId without needing to provide an `ID` implementor.
* Allow that all nodes sharing a nodeId can treated as a group for sending/receiving in the connector info

# v0.5.0
* Context propagation. Added: `StartFuncCtx[OUT]` type, `AsStartCtx` function and `StartCtx` method.

# v0.4.2
* High-Level API: when the builder can't connect nodes with different types, it returns
  an error instead of panicking.
# v0.4.1
* High-Level API: Instancer constraint in Providers' CFG

# v0.4.0

* Autopiping high-level graph API
* Breaking changes:
    - `node.Init` renamed to `node.Start`

# v0.3.0

* Inter-node communication input channels are now unbuffered by default. To make them buffered,
  you can append the `node.ChannelBufferLen` function to the `AsMiddle` and `AsTerminal` functions.

# v0.2.0

* Ported to Go 1.18 and generics for faster and safer execution.

# v0.1.1

* Added InType and OutType inspection functions to the Nodes

# v0.1.0

* Initial import from github.com/mariomac/go-pipes
