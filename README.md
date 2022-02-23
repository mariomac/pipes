# Dynamic graph architecture for asynchronous parallel processing in Go

API doc: https://pkg.go.dev/github.com/mariomac/go-pipes

Go-pipes is a library that allows to dynamically connect multiple pipeline
stages that are communicated via channels. Each stage will run in a goroutine.

This library allows wrapping functions within Nodes of a graph. In order to pass data across
the nodes, each wrapped function must receive, as arguments, an input channel, an output channel,
or both.

There are three types of nodes:

* **Init** node: each of the starting point of a graph. This is, all the nodes that bring information
  from outside the graph: e.g. because they generate them or because they acquire them from an
  external source like a Web Service. A graph must have at least one Init node. An Init node must 
  have at least one output node.
* **Inner** node: any intermediate node that receives data from another node, processes/filters it,
  and forwards the data to another node. An Inner node must have at least one output node.
* **Terminal** node: any node that receives data from another node and does not forward it to
  another node, but can process it and send the results to outside the graph
  (e.g. memory, storage, web...)

## Example pipeline

The following pipeline has two Init nodes that send the data to two destination Inner
nodes (`odds` and `evens`). From there, the data follows their own branches until they
are eventually joined in the `printer` Terminal node.

Check the complete examples in the [examples/](./examples) folder).

```go
func main() {
	// Defining init, inner and terminal nodes that wrap some functions
	start1 := node.AsInit(StartCounter)
	start2 := node.AsInit(StartRandoms)
	odds := node.AsInner(OddFilter)
	evens := node.AsInner(EvenFilter)
	oddsMsg := node.AsInner(Messager("odd number"))
	evensMsg := node.AsInner(Messager("even number"))
	printer := node.AsTerminal(Printer)
	
	// Connecting nodes like:
	//
    // start1----\ /---start2
    //   |        X      |
    //  evens<---/ \-->odds
    //   |              |
    //  evensMsg      oddsMsg
    //        \       /
    //         printer

	start1.SendsTo(evens, odds)
	start2.SendsTo(evens, odds)
	odds.SendsTo(oddsMsg)
	evens.SendsTo(evensMsg)
	oddsMsg.SendsTo(printer)
	evensMsg.SendsTo(printer)

	start1.Start()
	start2.Start()

	time.Sleep(2 * time.Second)
}
```

Output:

```
even number: 2
odd number: 847
odd number: 59
odd number: 81
odd number: 81
even number: 0
odd number: 3
odd number: 1
odd number: 887
even number: 4
```
