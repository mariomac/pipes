# Dynamic asynchronous graph processing

API doc: https://pkg.go.dev/github.com/mariomac/go-pipes

Go-pipes is a library that allows to dynamically connect multiple pipeline
stages that are communicated via channels. Each stage will run in a goroutine.

Example pipeline (check the basic example in the [examples/](./examples) folder):

```go
func main() {
	start1 := node.AsInit(StartCounter)
	start2 := node.AsInit(StartRandoms)
	odds := node.AsInner(OddFilter)
	evens := node.AsInner(EvenFilter)
	oddsMsg := node.AsInner(Messager("odd number"))
	evensMsg := node.AsInner(Messager("even number"))
	printer := node.AsTerminal(Printer)

	/*
			       start1----\ /---start2
			          |       X      |
			        evens<---/ \-->odds
			          |              |
			        evensMsg      oddsMsg
		                   \ 	  /
			               printer
	*/
	start1.SendsTo(evens, odds)
	start2.SendsTo(evens, odds)
	odds.SendsTo(oddsMsg)
	evens.SendsTo(evensMsg)
	oddsMsg.SendsTo(printer)
	evensMsg.SendsTo(printer)

	start1.Start()
	start2.Start()

	time.Sleep(2 * time.Second)
}```

