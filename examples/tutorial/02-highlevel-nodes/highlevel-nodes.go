package main

import (
	"fmt"
	"strings"

	"github.com/mariomac/pipes/pkg/graph"
	"github.com/mariomac/pipes/pkg/graph/stage"
	"github.com/mariomac/pipes/pkg/node"
)

type StartConfig struct {
	stage.Instance // need to be added in high-level
	Prefix         string
}

// in high-level, even if we don't want to pass configuration, we should
// create it with at least the instance ID
type MiddleConfig struct {
	stage.Instance
}
type TerminalConfig struct {
	stage.Instance
}

type Config struct {
	graph.Connector // required in high-level api to specify how nodes are connected
	Starts          []StartConfig
	Middle          MiddleConfig
	Terminal        TerminalConfig
}

func StartProvider(cfg StartConfig) node.StartFunc[string] {
	return func(out chan<- string) {
		out <- cfg.Prefix + ", 1"
		out <- cfg.Prefix + ", 2"
		out <- cfg.Prefix + ", 3"
		// a node is ended when its internal function ends
	}
}

func MiddleProvider(_ MiddleConfig) node.MiddleFunc[string, string] {
	return func(in <-chan string, out chan<- string) {
		// a middle and terminal node shouldn't end until its previous node ends and
		// all the input is processed
		for i := range in {
			out <- strings.ToUpper(i)
		}
	}
}

func TerminalProvider(_ TerminalConfig) node.TerminalFunc[string] {
	return func(in <-chan string) {
		for i := range in {
			fmt.Println(i)
		}
	}
}

func main() {
	builder := graph.NewBuilder()

	graph.RegisterStart(builder, StartProvider)
	graph.RegisterMiddle(builder, MiddleProvider)
	graph.RegisterTerminal(builder, TerminalProvider)

	grp, err := builder.Build(Config{
		Starts: []StartConfig{
			{Instance: "helloer", Prefix: "Hello"},
			{Instance: "hier", Prefix: "Hi"},
		},
		Middle:   MiddleConfig{"uppercaser"},
		Terminal: TerminalConfig{"printer"},
		Connector: graph.Connector{
			"helloer":    []string{"uppercaser"},
			"hier":       []string{"uppercaser"},
			"uppercaser": []string{"printer"},
		},
	})
	if err != nil {
		panic(err)
	}
	// graph.Run it's blocking and won't continue until the graph stopped processing
	grp.Run()
}
