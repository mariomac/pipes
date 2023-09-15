package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"strings"

	"github.com/mariomac/pipes/pkg/graph"
	"github.com/mariomac/pipes/pkg/node"
)

type LineReader struct {
	Input io.Reader
}

type WordFilter struct {
	Match string
}

type LineWriter struct {
	Output io.Writer
}

type Grepper struct {
	Reader LineReader `nodeId:"reader" sendTo:"filter"`
	Filter WordFilter `nodeId:"filter" sendTo:"writer"`
	Writer LineWriter `nodeId:"writer"`
}

func LineReaderProvider(_ context.Context, cfg LineReader) (node.StartFuncCtx[string], error) {
	return func(_ context.Context, out chan<- string) {
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

func WordFilterProvider(_ context.Context, cfg WordFilter) node.MiddleFunc[string, string] {
	// a middle and terminal node shouldn't end until its previous node ends and
	// all the input is processed
	return func(in <-chan string, out chan<- string) {
		for line := range in {
			// the input line will be only forwarded if it contains the match substring
			if strings.Contains(line, cfg.Match) {
				out <- line
			}
		}
	}
}

func LineWriterProvider(_ context.Context, cfg LineWriter) node.TerminalFunc[string] {
	return func(in <-chan string) {
		for line := range in {
			// ignore error handling for the sake of brevity
			_, _ = cfg.Output.Write(append([]byte(line), '\n'))
		}
	}
}

func main() {
	graphBuilder := graph.NewBuilder()
	graph.RegisterStart(graphBuilder, LineReaderProvider)
}
