package stages

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mariomac/pipes/pkg/graph/stage"
	"github.com/mariomac/pipes/pkg/node"
	"github.com/sirupsen/logrus"
)

const defaultPort = 8080

type Http struct {
	Name string `hcl:",label"`
	Port int    `hcl:"port,optional"`
}

const HttpIngestStage = stage.Type("http")

// HttpIngestProvider listens for HTTP connections and forwards them. The instantiator
// needs to receive a stage.Http instance.
func HttpIngestProvider(c Http) node.StartFunc[[]byte] {
	port := c.Port
	if port == 0 {
		port = defaultPort
	}
	log := logrus.WithField("component", "HttpIngest")
	return func(out chan<- []byte) {
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
	}
}

type Stdout struct {
	Name    string `hcl:",label"`
	Prepend string `hcl:"prepend,optional"`
}

const StdoutExportStage = stage.Type("stdout")

// StdOutExportProvider receives any message and prints it, prepending a given message
func StdOutExportProvider(c Stdout) node.TerminalFunc[string] {
	return func(in <-chan string) {
		for s := range in {
			fmt.Println(c.Prepend + s)
		}
	}
}

type Deleter struct {
	Name   string   `hcl:",label"`
	Fields []string `hcl:"fields"`
}

const FieldDeleterStage = stage.Type("deleter")

// FieldDeleterTransformProvider receives a map and removes the configured fields from it
func FieldDeleterTransformProvider(c Deleter) node.MiddleFunc[map[string]any, map[string]any] {
	toDelete := map[string]struct{}{}
	for _, f := range c.Fields {
		toDelete[fmt.Sprint(f)] = struct{}{}
	}
	return func(in <-chan map[string]interface{}, out chan<- map[string]interface{}) {
		for m := range in {
			for td := range toDelete {
				delete(m, td)
			}
			out <- m
		}
	}
}
