package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mariomac/pipes/pipe"
)

type FileLine struct {
	FileName string
	Line     string
}

type MiniGrepNodes struct {
	fileFinder  pipe.Start[*os.File]
	fileScanner pipe.Middle[*os.File, FileLine]
	matchFilter pipe.Middle[FileLine, FileLine]
	printer     pipe.Final[FileLine]
}

func (n *MiniGrepNodes) Connect() {
	n.fileFinder.SendTo(n.fileScanner)
	n.fileScanner.SendTo(n.matchFilter)
	n.matchFilter.SendTo(n.printer)
}

func finderPtr(f *MiniGrepNodes) *pipe.Start[*os.File]             { return &f.fileFinder }
func scannerPtr(f *MiniGrepNodes) *pipe.Middle[*os.File, FileLine] { return &f.fileScanner }
func matcherPtr(f *MiniGrepNodes) *pipe.Middle[FileLine, FileLine] { return &f.matchFilter }
func printerPtr(f *MiniGrepNodes) *pipe.Final[FileLine]            { return &f.printer }

func FileFinder(files []string) pipe.StartFunc[*os.File] {
	return func(out chan<- *os.File) {
		// if no file patterns are provided, minigrep filters standard input
		if len(files) == 0 {
			out <- os.Stdin
		}
		for _, fname := range files {
			if handler, err := os.Open(fname); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", fname, err.Error())
			} else {
				out <- handler
			}
		}
	}
}

func FileScanner(in <-chan *os.File, out chan<- FileLine) {
	for f := range in {
		fileName := f.Name()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			out <- FileLine{
				FileName: fileName,
				Line:     scanner.Text(),
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %s\n", f.Name(), err.Error())
		}
	}
}

func MatchFilterProvider(regex string) pipe.MiddleProvider[FileLine, FileLine] {
	return func() (pipe.MiddleFunc[FileLine, FileLine], error) {
		return func(in <-chan FileLine, out chan<- FileLine) {
			for line := range in {
				if strings.Contains(line.Line, regex) {
					out <- line
				}
			}
		}, nil
	}
}

func Printer(in <-chan FileLine) {
	for l := range in {
		fmt.Printf("%s:%s\n", l.FileName, l.Line)
	}
}

func main() {
	gb := pipe.NewBuilder(&MiniGrepNodes{})
	pipe.AddStart(gb, finderPtr, FileFinder([]string{"minigrep.go", "../../pipe/node.go"}))
	pipe.AddMiddle(gb, scannerPtr, FileScanner)
	pipe.AddMiddleProvider(gb, matcherPtr, MatchFilterProvider("string"))
	pipe.AddFinal(gb, printerPtr, Printer)

	runner, err := gb.Build()
	if err != nil {
		log.Fatal("minigrep:", err.Error())
	}
	runner.Start()

	<-runner.Done()
}
