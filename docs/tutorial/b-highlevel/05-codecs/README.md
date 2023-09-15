# Tutorial 03: Coder/Decoders (Codecs)

Using the **low-level Node API**, let's create a start node that generates integers and a terminal node that
accepts strings:

```go
start := node.AsStart(func(out chan<- int) {
    out <- 123
})
end := node.AsTerminal(func(in <-chan string) {
    for i := range in {
        fmt.Println(i)
    }
})
```

Since the output channel of the start node has a different type than the
input channel of the terminal node, you can't connect them. If you use the node
API to connect nodes from different types, you would get a compiler error thanks
to the type safety provided by Go 1.18's generics:

```go
start.SendsTo(end)
// compiler error: cannot use end (variable of type *node.Terminal[string])
// as type node.Receiver[int] in argument to start.SendsTo:
// *node.Terminal[string] does not implement node.Receiver[int]
// (wrong type for joiner method)
```

If you wanted to connect both nodes, the compile-time type safety would force you
to create a middle node that converts integers into strings, and put it between
the start and the end node:

```go
middle := node.AsMiddle(func(in <-chan int, out chan<- string) {
    for i := range in {
        out <- strconv.Itoa(i)
    }
})
start.SendsTo(middle)
middle.SendsTo(end)
```

For the scenario of the **high-level Graph API**, the graph is not built at compile time
but at runtime, letting the user to specify which and how nodes are connected. That would
allow them connecting nodes with incompatible output/input types. You can provide
extra middle nodes acting as type translators, and let the user manually connect them.
However, you should document some implementation details and permeate them to the user,
who should define extra nodes and connections.

To avoid this extra complexity, the Graph API allows defining **Codecs**, which are a
middle function that can translate different types of data and are instantiated,
transparently to the user, when the user connects two nodes with different types.

For example, let's define a random number generator and a string printer as graph stage providers:

```go
type GeneratorConfig struct {
	stage.Instance
	Repeat     int
	Seed       int64
	LowerBound int
	UpperBound int
}

func Generator(cfg GeneratorConfig) node.StartFuncCtx[int] {
	return func(_ context.Background, out chan<- int) {
		rand.Seed(cfg.Seed)
		for n := 0; n < cfg.Repeat; n++ {
			out <- cfg.LowerBound + rand.Intn(cfg.UpperBound-cfg.LowerBound)
		}
	}
}

type PrinterConfig struct {
	stage.Instance
}

func Printer(_ PrinterConfig) node.TerminalFunc[string] {
	return func(in <-chan string) {
		for i := range in {
			fmt.Println("received: ", i)
		}
	}
}
```

Then, let's create a connected configuration that would allow the user
connecting both nodes if we populated a `Config` value from e.g. a YAML
file:

```go
type Config struct {
	graph.Connector
	Generator GeneratorConfig
	Printer   PrinterConfig
}
```

As explained in the [previous tutorial](../02-highlevel-nodes/), we need
to create a graph builder, register the providers and build the
graph from a configuration file (which, in this example, is hardcoded
for clarity):

```go
gb := graph.NewBuilder()
graph.RegisterStart(gb, Generator)
graph.RegisterTerminal(gb, Printer)

grp, err := gb.Build(context.Background(), Config{
    Generator: GeneratorConfig{
        Instance:   "generator",
        LowerBound: -10,
        UpperBound: 10,
        Seed:       time.Now().UnixNano(),
        Repeat:     5,
    },
    Printer: PrinterConfig{"printer"},
    Connector: graph.Connector{
        "generator": []string{"printer"},
    },
})
```

This code will compile without any error, but at runtime, the node builder will complain
because it can't connect an integer generator to a string printer:

```
panic: can't connect "generator" and "printer" stages because
there isn't registered any int -> string codec
```

This error can be unavoidable if nodes outputs/inputs have different semantics and
can't be translated. But in some cases (e.g. converting a JSON text to a Go `map`),
you could define a *Codec*, which is any function that fulfills the `node.MiddleFunc`
signature.

To connect the nodes of the tutorial example, we need to create a `MiddleFunc` that
converts any `int` to a `string`:

```go
func IntStringCodec(in <-chan int, out chan<- string) {
	for i := range in {
		out <- strconv.Itoa(i)
	}
}
```

Then, in the `main` code, registering the codec before invoking the `Build` method:

```go
gb := graph.NewBuilder()
graph.RegisterStart(gb, Generator)
graph.RegisterTerminal(gb, Printer)

graph.RegisterCodec(gb, IntStringCodec)
```

Now, the `Build` method will succeed and running the graph (`grp.Run(ctx)` in
the [example code](codecs.go)) should print something similar to:

```
received:  -5
received:  7
received:  3
received:  -8
received:  -10
```