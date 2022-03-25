package stage

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mariomac/pipes/pkg/node"
	"github.com/sirupsen/logrus"
)

type Type string
type Name string

// A provider wraps an instantiation function that, given a configuration argument, returns a
// node with a processing function.

type StartProvider[CFG, O any] struct {
	StageType    Type
	Instantiator func(CFG) *node.Start[O]
}

type MiddleProvider[CFG, I, O any] struct {
	StageType    Type
	Instantiator func(CFG) *node.Middle[I, O]
}

type TerminalProvider[CFG, I any] struct {
	StageType    Type
	Instantiator func(CFG) *node.Terminal[I]
}

const defaultPort = 8080

type Http struct {
	Name string `hcl:",label"`
	Port int    `hcl:"port,optional"`
}

// HttpIngestProvider listens for HTTP connections and forwards them. The instantiator
// needs to receive a stage.Http instance.
var HttpIngestProvider = StartProvider[Http, []byte]{
	StageType: "http",
	Instantiator: func(c Http) *node.Start[[]byte] {
		port := c.Port
		if port == 0 {
			port = defaultPort
		}
		log := logrus.WithField("component", "HttpIngest")
		return node.AsStart(func(out chan<- []byte) {
			err := http.ListenAndServe(fmt.Sprintf(":%d", port),
				http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					if request.Method != http.MethodPost {
						writer.WriteHeader(http.StatusBadRequest)
						return
					}
					body, err := ioutil.ReadAll(request.Body)
					if err != nil {
						log.WithError(err).Warn("failed request")
						writer.WriteHeader(http.StatusBadRequest)
						writer.Write([]byte(err.Error()))
						return
					}
					out <- body
				}))
			log.WithError(err).Warn("HTTP server ended")
		})
	},
}

type Stdout struct {
	Name    string `hcl:",label"`
	Prepend string `hcl:"prepend,optional"`
}

// StdOutExportProvider receives any message and prints it, prepending a given message
var StdOutExportProvider = TerminalProvider[Stdout, string]{
	StageType: "stdout",
	Instantiator: func(c Stdout) *node.Terminal[string] {
		return node.AsTerminal(func(in <-chan string) {
			for s := range in {
				fmt.Println(c.Prepend + s)
			}
		})
	},
}

type Deleter struct {
	Name   string   `hcl:",label"`
	Fields []string `hcl:"fields"`
}

// FieldDeleterTransformProvider receives a map and removes the configured fields from it
var FieldDeleterTransformProvider = MiddleProvider[Deleter, map[string]any, map[string]any]{
	StageType: "deleter",
	Instantiator: func(c Deleter) *node.Middle[map[string]any, map[string]any] {
		toDelete := map[string]struct{}{}
		for _, f := range c.Fields {
			toDelete[fmt.Sprint(f)] = struct{}{}
		}
		return node.AsMiddle(func(in <-chan map[string]interface{}, out chan<- map[string]interface{}) {
			for m := range in {
				for td := range toDelete {
					delete(m, td)
				}
				out <- m
			}
		})
	},
}
