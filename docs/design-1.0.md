# Roadmap document for graph composition API

> ⚠️ do not use this as an official API documentation. This just enumerates
> some features, and many of them aren't yet implemented or directly deprecated.

## Assumptions

Each graph must instantiate at least one start node.

Each graph must instantiate at least one terminal node.


## Explicit modes

In explicit modes, each node has an explicit ID and destination node.

### Basic

```go
type MyGraph struct {
    Start StartNode `nodeId:"start" sendTo:"mid"`
    Mid   MidNode   `nodeId:"mid" sendTo:"term"`
    Term  TermNode  `nodeId:"term"`
}
```

```mermaid
graph LR;
    start --> mid
    mid --> term
```

### Mixed with user-provided connections

```go
type MyGraph struct {
    Start StartNode `nodeId:"start" sendTo:"mid"`
    Mid   MidNode   `nodeId:"mid" sendTo:"term"`
    Term  TermNode  `nodeId:"term"`
    Connector
}
```

The user-provided `Connector` would override any explicit, annotated definition.

### Basic, sending to multiple nodes

```go
type MyGraph struct {
    Start StartNode `nodeId:"start" sendTo:"mid1,mid2"`
    Mid1  MidNode  `nodeId:"mid1" sendTo:"term"`
    Mid2  MidNode  `nodeId:"mid2" sendTo:"term"`
    Term  TermNode  `nodeId:"term"`
}
```

```mermaid
graph LR;
    start --> mid1
    start --> mid2
    mid1 --> term
    mid2 --> term
```

### Optional elements

Any sender is nillable. 
A sender would throw runtime error if all its receivers are nil, but some of them
can be nul (optional).

E.g:
```go
type MyGraph struct {
    Start StartNode `nodeId:"start" sendTo:"mid1,mid2"`
    Mid1  *MidNode  `nodeId:"mid1" sendTo:"term"`
    Mid2  *MidNode  `nodeId:"mid2" sendTo:"term"`
    Term  TermNode  `nodeId:"term"`
}
```

This config will work:
```go
MyGraph {
    Mid1: &MidNode{}
    Mid2: nil,
}
```
(Start node can send through mid1 despite mid2 is nil).

This config wil fail:
```go
MyGraph { Mid1: nil, Mid2: nil}
```

As `start` node can't send data to any of their destinations.

TODO: explain Enabler interface and how it can optionally disable non-pointer fields.

### BypassElement

The previous example would pass if use `forwardTo` instead of `sendTo`:

```go
type MyGraph struct {
    Start StartNode `nodeId:"start" sendTo:"mid"`
    Mid  *MidNode   `nodeId:"mid" forwardTo:"term"`
    Term  TermNode  `nodeId:"term"`
}
```

In that case, a disabled node would be ignored and the StartNode will directly send to
TermNode.

### Allow user configuring the node and destination IDs

```go
type StartNode struct {
    ID   string `nodeId` // marks value of this field as nodeId
    Dest string `sendTo` // marks value of this field as destination node Id
}
type MyGraph struct {
    Start StartNode // we don't need to define neither nodeId nor sendTo here
    Mid   MidNode   `nodeId:"mid" sendTo:"term"`
    Term  TermNode  `nodeId:"term"`
}
```

E.g. for a configuration like:
```go
MyGraph {
    Start: StartNode {
        ID:      "myStart",
        SendsTo: "mid",
    },
}
```

```mermaid
graph LR;
    myStart --> mid
    mid --> term
```

This might be specially helpful for a free-form, user-defined graphs:

```go
type StartNode struct {
    ID   string `nodeId`
    Dest string `sendTo`
}
type MidNode struct {
    ID   string `nodeId`
    Dest string `sendTo`
}
type MyGraph struct {
    Start StartNode
    Mids  []MidNode
    Term  TermNode `nodeId:"output"`
}
```

E.g. A config like:

```go
MyGraph {
    Start: StartNode {
        ID: "st", Dest: "mid1-1,mid1-2",
    },
    Mids: []MidNode {
        { ID: "mid1-1", Dest:"mid2-1,mid2-2" },
        { ID: "mid1-2", Dest:"mid2-2" },
        { ID: "mid2-1", Dest:"output" },
        { ID: "mid2-2", Dest:"output" },
    },
}
```

would generate:

```mermaid
graph LR;
    st --> mid1-1
    st --> mid1-2
    mid1-1 --> mid2-1
    mid1-1 --> mid2-2
    mid1-2 --> mid2-2
    mid2-1 --> output
    mid2-2 --> output
```

## Node auto-naming

### Sharing nodeId in array

**ON HOLD**: probably there is no need for this as semantically different nodes
would have different configuration definitions.

A "nodeId" tagging an array of nodes makes all the nodes to share de ID.

E.g.:

```go
type MyGraph struct {
    Start StartNode `nodeId:"start" sendTo:"mid"`
    Mids  []MidNode `nodeId:"mid" sendTo:"output"`
    Term  TermNode `nodeId:"term"`
}
```

A config like:

```go
MyGraph {
    Mids: []MidNode{ {}, {}, {} }, // three middle nodes
}
```

Would generate:

```mermaid
graph LR;
    start --> m0["mid"]
    start --> m1[mid]
    start --> m2[mid]
    m0 --> term
    m1 --> term
    m2 --> term
```

### Sequential array

**ON HOLD**: probably there is no need for this as semantically different nodes
would have different configuration definitions.

A `sequential` tag would make each node defined in an array to send data to the next element in the array.

If the tagged node is Middle or Terminal, the first element in the array would get the `nodeId`.

If the tagged node is Start or Middle, the last element of the array would forward the data to the `sendTo` node.

If we tag the previous example with `sequential` in the

```go
type MyGraph struct {
    Start StartNode `nodeId:"start" sendTo:"mid"`
    Mids  []MidNode `sequential nodeId:"mid" sendTo:"output"`
    Term  TermNode `nodeId:"term"`
}
```

```mermaid
graph LR;
    start --> m0[mid]
    m0 --> m1[mid]
    m1 --> m2[mid]
    m2 --> term
```