package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"strings"

	"github.com/mariomac/pipes/pkg/graph"
	"github.com/mariomac/pipes/pkg/node"
)

// LineReader node configuration
type LineReader struct {
	Input io.Reader
}

// WordFilter node configuration
type WordFilter struct {
	Match string
}

// LineWriter node configuration
type LineWriter struct {
	Output io.Writer
}

// Grepper defines the nodes of a processing graph
// and how they are connected each other
type Grepper struct {
	Reader LineReader `sendTo:"Filter"`
	Filter WordFilter `sendTo:"Writer"`
	Writer LineWriter
}

func LineReaderProvider(cfg LineReader) (node.StartFunc[string], error) {
	return func(out chan<- string) {
		scan := bufio.NewScanner(cfg.Input)
		for scan.Scan() {
			out <- scan.Text()
		}
		if err := scan.Err(); err != nil {
			log.Println("error scanning", err)
		}
		// when the start function ends, the output channel will
		// be automatically closed
	}, nil
}

func WordFilterProvider(cfg WordFilter) (node.MidFunc[string, string], error) {
	// a middle and terminal node shouldn't end until its previous node ends and
	// all the input is processed
	return func(in <-chan string, out chan<- string) {
		for line := range in {
			// the input line will be only forwarded if it contains the match substring
			if strings.Contains(line, cfg.Match) {
				out <- line
			}
		}
	}, nil
}

func LineWriterProvider(cfg LineWriter) (node.EndFunc[string], error) {
	return func(in <-chan string) {
		for line := range in {
			// ignore error handling for the sake of brevity
			_, _ = cfg.Output.Write(append([]byte(line), '\n'))
		}
	}, nil
}

func main() {
	// Create Graph builder and register all the node types
	graphBuilder := graph.NewBuilder()
	graph.RegisterStart(graphBuilder, LineReaderProvider)
	graph.RegisterMiddle(graphBuilder, WordFilterProvider)
	graph.RegisterTerminal(graphBuilder, LineWriterProvider)

	// Build graph from a given configuration, and run it
	input := strings.NewReader("hello, my friend\n" +
		"how are you?\n" +
		"I said hello but\n" +
		"I need to say goodbye")

	grepper, err := graphBuilder.Build(Grepper{
		Reader: LineReader{Input: input},
		Filter: WordFilter{Match: "hello"},
		Writer: LineWriter{Output: os.Stdout},
	})
	if err != nil {
		log.Panic("building graph", err)
	}
	grepper.Run()
}
