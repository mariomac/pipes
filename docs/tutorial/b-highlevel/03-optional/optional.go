package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/mariomac/pipes/pkg/graph"
	"github.com/mariomac/pipes/pkg/node"
)

type InputWords string

func WordReaderProvider(input InputWords) (node.StartFunc[string], error) {
	return func(out chan<- string) {
		for _, word := range strings.Split(string(input), " ") {
			if len(word) > 0 {
				out <- word
			}
		}
	}, nil
}

type CasingType int

const None = CasingType(0)
const ToUpper = CasingType(1)
const ToLower = CasingType(2)

func (c CasingType) Enabled() bool {
	return c == ToUpper || c == ToLower
}

func CasingProvider(c CasingType) (node.MiddleFunc[string, string], error) {
	return func(in <-chan string, out chan<- string) {
		for i := range in {
			if c == ToLower {
				out <- strings.ToLower(i)
			} else {
				out <- strings.ToUpper(i)
			}
		}
	}, nil
}

type Untilder struct{}

func UntilderProvider(_ *Untilder) (node.MiddleFunc[string, string], error) {
	return func(in <-chan string, out chan<- string) {
		for tilded := range in {
			untilded := make([]byte, len(tilded))
			t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
			ln, _, _ := t.Transform(untilded, []byte(tilded), true)
			out <- string(untilded[:ln])
		}
	}, nil
}

type Aggregator struct{}

func AggregatorProvider(_ Aggregator) (node.TerminalFunc[string], error) {
	return func(in <-chan string) {
		anagrams := map[string][]string{}
		alreadyChecked := map[string]struct{}{}
		for word := range in {
			if _, ok := alreadyChecked[word]; ok {
				continue
			}
			alreadyChecked[word] = struct{}{}
			sorted := []byte(word)
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i] < sorted[j]
			})
			anagrams[string(sorted)] = append(anagrams[string(sorted)], word)
		}
		fmt.Println("list of anagrams:")
		for _, vals := range anagrams {
			if len(vals) > 1 {
				fmt.Println(strings.Join(vals, ", "))
			}
		}
	}, nil
}

type AnagramFinder struct {
	Input  InputWords `nodeId:"reader" sendTo:"caser"`
	Caser  CasingType `nodeId:"caser" forwardTo:"untild"`
	Untild *Untilder  `nodeId:"untild" forwardTo:"aggr"`
	Aggr   Aggregator `nodeId:"aggr"`
}

func main() {
	gb := graph.NewBuilder()
	graph.RegisterStart(gb, WordReaderProvider)
	graph.RegisterMiddle(gb, CasingProvider)
	graph.RegisterMiddle(gb, UntilderProvider)
	graph.RegisterTerminal(gb, AggregatorProvider)

	finder, err := gb.Build(AnagramFinder{
		Input:  `The Hôtel of Letho is the first fistr of a rank from Nark Kran on eth`,
		Caser:  ToUpper,
		Untild: &Untilder{},
	})
	if err != nil {
		panic(err)
	}
	finder.Run()
}
