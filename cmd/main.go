package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mariomac/go-pipes/pkg/pipe"
	"gopkg.in/yaml.v2"
)

const pipelineConfig = `
invocations:
  - name: ingest
  - fork:
      right:
        - name: print 
      left:
        - name: json2record
        - name: appender
          args:
            - "direction"
            - "left"
        - name: record2line
        - name: print
`

func main() {
	definition := pipe.PipelineDefinition{}
	exitOnErr(yaml.Unmarshal([]byte(pipelineConfig), &definition))
	exitOnErr(pipe.Run(definition))
	<-context.TODO().Done()
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}
}
