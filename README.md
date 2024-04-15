# PIPES: Processing In Pipeline-Embedded Stages

PIPES is a library that allows to dynamically connect multiple pipeline
stages that are communicated via channels. Each stage will run in a goroutine.
Its main use case is the creation of [Extract-Transform-Load (ETL)](https://en.wikipedia.org/wiki/Extract,_transform,_load)
components, such as data cleaners or monitoring agents.

* API doc: https://pkg.go.dev/github.com/mariomac/pipes
* Tutorial: https://github.com/mariomac/pipes/tree/main/tutorial

It is a fork version of the [Red Hat's & IBM Gopipes library](https://pkg.go.dev/github.com/netobserv/gopipes),
but this library is not related to any concrete product.

This library allows wrapping functions within Nodes of a connected Graph. In order to pass data across
the nodes, each wrapped function must receive, as arguments, an input channel, an output channel,
or both.

There are three types of nodes:

* **Start** node: each of the starting point of a graph. This is, all the nodes that bring information
  from outside the graph: e.g. because they generate them or because they acquire them from an
  external source like a Web Service. A graph must have at least one Start node. A Start node must
  have at least one output node.
* **Middle** node: any intermediate node that receives data from another node, processes/filters it,
  and forwards the data to another node. A Middle node must have at least one output node.
* **Final** node: any node that receives data from another node and does not forward it to
  another node, but can process it and send the results to outside the graph
  (e.g. memory, storage, web...)

Nodes are instantiated, assigned and connected via an API formed by:

* A **NodesMap** interface whose implementing objects defines the variables pointing to the nodes
  and how they are interconnected via its `Connect()` method.
* A **pipeline Builder** that receives how to instantiate each node and where in the **NodesMap**
  object to store it.
* A **pipeline Runner** that is created from the builder, and manages the execution lifecycle of all
  the nodes.
