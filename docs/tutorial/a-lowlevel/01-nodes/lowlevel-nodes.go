package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"strings"

	"github.com/mariomac/pipes/pkg/node"
)

// LineReader returns a node.StartFunc[string] that will read each text line
// from the provided io.Reader and will forward it to the output channel
// of the node.
func LineReader(input io.Reader) node.StartFunc[string] {
	return func(out chan<- string) {
		scan := bufio.NewScanner(input)
		for scan.Scan() {
			out <- scan.Text()
		}
		if err := scan.Err(); err != nil {
			log.Println("error scanning", err)
		}
		// when the start function ends, the output channel will
		// be automatically closed
	}
}

// WordFilter returns a node.MiddleFunc[string, string] that will read
// each text line from the input channel and will forward to the output
// channel the lines that contain the match argument as a substring.
func WordFilter(match string) node.MiddleFunc[string, string] {
	return func(in <-chan string, out chan<- string) {
		for line := range in {
			// the input line will be only forwarded if it contains the match substring
			if strings.Contains(line, match) {
				out <- line
			}
		}
	}
}

// LineWriter returns a node.TerminalFunc[string] that reads all the lines from
// the input channel and forwards them to the provided output io.Writer.
func LineWriter(output io.Writer) node.TerminalFunc[string] {
	return func(in <-chan string) {
		for line := range in {
			// ignore error handling for the sake of brevity
			_, _ = output.Write(append([]byte(line), '\n'))
		}
	}
}

func main() {
	// file mock, can be any implementer or io.Reader
	inputText := strings.NewReader(
		"hello my friend\n" +
			"how are you?\n" +
			"hello again\n" +
			"but bye")

	// Instantiation
	start := node.AsStart(LineReader(inputText))
	middle := node.AsMiddle(WordFilter("hello"))
	terminal := node.AsTerminal(LineWriter(os.Stdout))

	// Connection
	start.SendTo(middle)
	middle.SendTo(terminal)

	// All the start nodes must start.
	// This graph only has one start node
	start.Start()

	// We should wait for all the terminal nodes
	<-terminal.Done()
}
