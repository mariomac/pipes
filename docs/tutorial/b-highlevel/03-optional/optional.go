package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/mariomac/pipes/pkg/graph"

	"github.com/mariomac/pipes/pkg/node"
)

// InputWords is a string containing multiple words
type InputWords string

// WordReaderProvider provides a Start function that splits the words
// in the input string and submits them individually to the output channel.
func WordReaderProvider(input InputWords) (node.StartFunc[string], error) {
	return func(out chan<- string) {
		notWords := regexp.MustCompile("[^a-zA-Z\u00C0-\u017F]")
		for _, word := range notWords.Split(string(input), -1) {
			if len(word) > 0 {
				out <- word
			}
		}
	}, nil
}

// CasingType indicates a transformation to do to a word: none, convert to upper case
// or convert to lower case.
type CasingType int

const None = CasingType(0)
const ToUpper = CasingType(1)
const ToLower = CasingType(2)

// Enabled returns true if the node that is associated to that CasingType is
// enabled (e.g. because it needs to transform each word), and false if the
// node is disabled (won't be started to save computing resources).
func (c CasingType) Enabled() bool {
	return c == ToUpper || c == ToLower
}

// CasingProvider returns a middle function that transforms and submits each received
// word according to the provided CasingType
func CasingProvider(c CasingType) (node.MidFunc[string, string], error) {
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

// Untilder is an empty type, only used to define that node in the
// Graph struct.
type Untilder struct{}

// UntilderProvider returns a middle func that replaces vowels and consonants with
// a tilde by their "non-tilded" similar. For example it would replace each ñ by n
// or each ô by o.
func UntilderProvider(_ *Untilder) (node.MidFunc[string, string], error) {
	return func(in <-chan string, out chan<- string) {
		for tilded := range in {
			untilded := make([]byte, len(tilded))
			t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
			ln, _, _ := t.Transform(untilded, []byte(tilded), true)
			out <- string(untilded[:ln])
		}
	}, nil
}

// Aggregator is an empty type only used to define the associated node in the Graph.
type Aggregator struct{}

// AggregatorProvider reads all the words from the input channel and, after the channel
// is closed and all the words are read, it shows the anagrams.
func AggregatorProvider(_ Aggregator) (node.EndFunc[string], error) {
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
	Input  InputWords `sendTo:"Untild"`
	Untild *Untilder  `forwardTo:"Caser"`
	Caser  CasingType `forwardTo:"Aggr"`
	Aggr   Aggregator
}

func main() {
	gb := graph.NewBuilder()
	graph.RegisterStart(gb, WordReaderProvider)
	graph.RegisterMiddle(gb, CasingProvider)
	graph.RegisterMiddle(gb, UntilderProvider)
	graph.RegisterTerminal(gb, AggregatorProvider)

	finder, err := gb.Build(AnagramFinder{
		Input:  `The hôtel of letho is the first fistr of a rank from Nark Kran on eth!`,
		Caser:  ToUpper,
		Untild: &Untilder{},
		// Please notice that we don't really need to instantiate the Aggregator field,
		// as its empty value is already an Aggregator{} instance
	})
	if err != nil {
		panic(err)
	}
	finder.Run()
}
