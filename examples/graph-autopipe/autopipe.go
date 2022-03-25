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

	// register codecs for automatic transformation between incompatible stages
	graph.RegisterCodec(builder, stages.BytesToStringCodec)
	graph.RegisterCodec(builder, stages.JSONBytesToMapCodec)
	graph.RegisterCodec(builder, stages.MapToStringCodec)

	// register the pipeline stages that are actually doing something
	graph.RegisterIngest(builder, stages.HttpIngestProvider)
	graph.RegisterTransform(builder, stages.FieldDeleterTransformProvider)
	graph.RegisterExport(builder, stages.StdOutExportProvider)

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
