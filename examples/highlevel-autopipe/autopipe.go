package main

import (
	"flag"
	"log"
	"os"

	"github.com/mariomac/pipes/pkg/config"
	"github.com/mariomac/pipes/pkg/graph"
	"github.com/mariomac/pipes/pkg/stage"
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
	builder.RegisterCodec(stage.BytesToStringCodec)
	builder.RegisterCodec(stage.JSONBytesToMapCodec)
	builder.RegisterCodec(stage.MapToStringCodec)

	// register the pipeline stages that are actually doing something
	builder.RegisterIngest(stage.HttpIngestProvider)
	builder.RegisterTransform(stage.FieldDeleterTransformProvider)
	builder.RegisterExport(stage.StdOutExportProvider)

	// Parse config and build graph from it
	grp, err := os.Open(*graphFile)
	if err != nil {
		log.Printf("can't load configuration: %v", err)
	}
	cfg, err := config.ReadConfig(grp)
	if err != nil {
		log.Printf("can't instantiate configuration: %v", err)
	}
	config.ApplyConfig(&cfg, builder)

	// build and run the graph
	b := builder.Build()
	b.Run()
}
