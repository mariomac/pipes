# CHANGELOG

# v0.7.0
* **BREAKING CHANGE**: StartProvider, MiddleProvider and TermProvider now accept a context that is
  can be enriched by the sender nodes, and can return an error that would cause the pipe build
  to return error.
  - Consequently, `builder.Build` also requires a context to be invoked.
* Also, providers can return an error as second return value. If any provider
  returns an error, the graph Build method will also return an error.

# v0.6.0
* As **an initial, unstable API**, we allow defining multiple Start functions into a start node.
  After some evaluation, we will implement also multiple MIddle and Terminal functions that behave
  as a single node.
* High-level API nodes do not need to implement `Instancer` if you define a `nodeId` tag in the
  config struct that defines them.
* An `InstaceID == '-'` or `nodeId:"-"` will ignore this field from the graph.
* Graph configs does not need to implement `ConnectedConfig` interface if their properties define the
  `sendTo` configuration.
* Graph builder returns error if nodes remain unconnected
* Graph builder returns error if a node sends data to itself
* A node config implementing `Enabler` interface can allow users disabling nodes without requiring pointers.
* Disabled or nil nodes can forward the received data by using the `forwardTo` annotation.

## Breaking changes
* High-level graph API add contexts to the `graph.Run(context.Context)` library, and
* Start providers in High-level API must return a `StartFuncCtx[OUT]` function instead of a
  `StartFunc[OUT]`
* In version 1.0 maybe StartFunc is replaced by StartFuncCtx and we force the usage of contexts always


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
