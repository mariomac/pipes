package main

import (
	"flag"
	"log"
	"os"

	"github.com/mariomac/pipes/examples/graph-autopipe/stages"
	"github.com/mariomac/pipes/pkg/graph"
)

var graphFile = flag.String("graph", "", "HCL graph file")

func main() {
	flag.Parse()
	if graphFile == nil || *graphFile == "" {
		flag.PrintDefaults()
		os.Exit(-1)
	}
	builder := graph.NewBuilder()

	// register the pipeline stage types that the user could want to instantiate and wire in the configuration
	graph.RegisterStart(builder, stages.HttpIngestStage, stages.HttpIngestProvider)
	graph.RegisterMiddle(builder, stages.FieldDeleterStage, stages.FieldDeleterTransformProvider)
	graph.RegisterExport(builder, stages.StdoutExportStage, stages.StdOutExportProvider)

	// register codecs for automatic transformation between incompatible stages
	graph.RegisterCodec(builder, stages.BytesToStringCodec)
	graph.RegisterCodec(builder, stages.JSONBytesToMapCodec)
	graph.RegisterCodec(builder, stages.MapToStringCodec)

	// Parse config and build graph from it
	grp, err := os.Open(*graphFile)
	if err != nil {
		log.Printf("can't load configuration: %v", err)
	}
	cfg, err := stages.ReadConfig(grp)
	if err != nil {
		log.Printf("can't instantiate configuration: %v", err)
	}

	stages.ApplyConfig(&cfg, builder)

	// build and run the graph
	b := builder.Build()
	b.Run()
}
