package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/mariomac/pipes/examples/graph-autopipe/stages"
	"github.com/mariomac/pipes/pkg/graph"
)

var graphFile = flag.String("graph", "", "HCL graph file")

func BuildGraph(graphFile string) graph.Graph {
	builder := graph.NewBuilder()

	// register the pipeline stage types that the user could want to instantiate and wire in the configuration
	graph.RegisterStart(builder, stages.HttpIngestProvider)
	graph.RegisterMiddle(builder, stages.FieldDeleterTransformProvider)
	graph.RegisterTerminal(builder, stages.StdOutExportProvider)

	// register codecs for automatic transformation between incompatible stages
	graph.RegisterCodec(builder, stages.BytesToStringCodec)
	graph.RegisterCodec(builder, stages.JSONBytesToMapCodec)
	graph.RegisterCodec(builder, stages.MapToStringCodec)

	// Parse config and build graph from it
	grp, err := os.Open(graphFile)
	if err != nil {
		log.Printf("can't load configuration: %v", err)
		panic(err)
	}
	cfg, err := stages.ReadConfig(grp)
	if err != nil {
		log.Printf("can't instantiate configuration: %v", err)
		panic(err)
	}

	if g, err := builder.Build(context.TODO(), cfg); err != nil {
		panic(err)
	} else {
		return g
	}
}

func main() {
	flag.Parse()
	if graphFile == nil || *graphFile == "" {
		flag.PrintDefaults()
		os.Exit(-1)
	}
	p := BuildGraph(*graphFile)
	p.Run(context.Background())
}
