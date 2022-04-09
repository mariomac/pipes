# PIPES: Processing In Pipeline-Embedded Stages

PIPES is a library that allows to dynamically connect multiple pipeline
stages that are communicated via channels. Each stage will run in a goroutine.
Its main use case is the creation of [Extract-Transform-Load (ETL)](https://en.wikipedia.org/wiki/Extract,_transform,_load)
components, such as data cleaners or monitoring agents.

* API doc: https://pkg.go.dev/github.com/mariomac/pipes
* Tutorial: https://github.com/mariomac/pipes/tree/main/examples/tutorial

It is the upper-upstream fork version of the [Red Hat's & IBM Gopipes library](https://pkg.go.dev/github.com/netobserv/gopipes)
and the core parts of the [Red Hat's & IBM Flowlogs pipeline](https://github.com/netobserv/flowlogs-pipeline),
where I plan to add experimental features that aren't related to any concrete product nor follow
any peer review nor company standard.

This library allows wrapping functions within Nodes of a connected. In order to pass data across
the nodes, each wrapped function must receive, as arguments, an input channel, an output channel,
or both.

It has two usable API layers: the **node** API (low-level), where you manually instantiate and wire every
node; and the **graph** API (high-level), that allows you providing a predefined set of nodes that are
automatically wired and simplifies the graph formation through configuration files.

## Node low-level API

There are three types of nodes:

* **Starting** node: each of the starting point of a graph. This is, all the nodes that bring information
  from outside the graph: e.g. because they generate them or because they acquire them from an
  external source like a Web Service. A graph must have at least one Start node. A Start node must 
  have at least one output node.
* **Middle** node: any intermediate node that receives data from another node, processes/filters it,
  and forwards the data to another node. A Middle node must have at least one output node.
* **Terminal** node: any node that receives data from another node and does not forward it to
  another node, but can process it and send the results to outside the graph
  (e.g. memory, storage, web...)

With the low-level API, you can instantiate each node and connect it manually. It is simple and
efficient for Graphs whose structure is known at code time.

For illustrative examples, you can have a look to the [basic low-level example](./examples/lowlevel-basic) and the [first chapter of the step-by-step tutorial](./examples/tutorial).


## Graph high-level API

The High-Level API is aimed for graphs whose structure might be specified at runtime
(e.g. via a configuration file that specifies which stages are run and how they are connected).

This API allows registering Node Generators and Codecs:

* A **Node Provider** is a generator function that, given a unique configuration type, returns a function
  that can go inside a Start, Middle or Terminal Node (as explained in the previous section).
* A **Codec** is a middle function (this is, it's wrapped into a middle node and receives an input 
  readable channel and an output writable channel) where input and output belong to different types.
  A codec will transform the input values to its equivalent in the output type. For example, it
  could convert JSON strings to a Go map. Codecs allows wiring nodes with different output/input
  types, and are automatically instantiated when needed.

Given a configuration that contains all the Node configuration types as fields, and a connection map,
a graph builder will accordingly instantiate all the nodes and codecs (if necessary) and wire them.

For more illustrative examples, check the [graph-autopipe example](./examples/graph-autopipe) and
the [step-by-step tutorial](./examples/tutorial).
