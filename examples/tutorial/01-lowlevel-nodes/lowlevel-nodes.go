package main

import (
	"fmt"
	"strings"

	"github.com/mariomac/pipes/pkg/node"
)

type StartConfig struct {
	Prefix string
}

func StartProvider(cfg StartConfig) node.StartFunc[string] {
	return func(out chan<- string) {
		out <- cfg.Prefix + ", 1"
		out <- cfg.Prefix + ", 2"
		out <- cfg.Prefix + ", 3"
		// a node is ended when its internal function ends
	}
}

func MiddleProvider() node.MiddleFunc[string, string] {
	return func(in <-chan string, out chan<- string) {
		// a middle and terminal node shouldn't end until its previous node ends and
		// all the input is processed
		for i := range in {
			out <- strings.ToUpper(i)
		}
	}
}

func TerminalProvider() node.TerminalFunc[string] {
	return func(in <-chan string) {
		for i := range in {
			fmt.Println(i)
		}
	}
}

func main() {
	// Instantiation
	start1 := node.AsStart(StartProvider(StartConfig{Prefix: "Hello"}))
	start2 := node.AsStart(StartProvider(StartConfig{Prefix: "Hi"}))
	middle := node.AsMiddle(MiddleProvider())
	terminal := node.AsTerminal(TerminalProvider())

	// Connection
	start1.SendsTo(middle)
	start2.SendsTo(middle)
	middle.SendsTo(terminal)

	// All the start nodes must start
	start1.Start()
	start2.Start()

	// We should wait for all the terminal nodes
	<-terminal.Done()
}
