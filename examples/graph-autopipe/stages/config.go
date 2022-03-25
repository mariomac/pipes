package stages

import (
	"io"
	"io/ioutil"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/mariomac/pipes/pkg/graph"
	"github.com/mariomac/pipes/pkg/graph/stage"
	"github.com/sirupsen/logrus"
)

type PipeConfig struct {
	Http    []Http      `hcl:"http,block"`
	StdOut  []Stdout    `hcl:"stdout,block"`
	Deleter []Deleter   `hcl:"deleter,block"`
	Connect Connections `hcl:"connect"`
}

// Connections key: name of the source node. Value: array of destination nodes.
type Connections map[string][]string

func ReadConfig(in io.Reader) (PipeConfig, error) {
	src, err := ioutil.ReadAll(in)
	if err != nil {
		return PipeConfig{}, err
	}
	var pc PipeConfig
	err = hclsimple.Decode(".hcl", src, nil, &pc)
	return pc, err
}

// ApplyConfig instantiates and configures the different pipeline stages according to the provided PipeConfig
func ApplyConfig(cfg *PipeConfig, builder *graph.Builder) {
	// TODO: find a better way to configure from HCL without having to iterate all the stage types
	for _, stg := range cfg.StdOut {
		if err := graph.NewTerminal[Stdout, string](builder, stage.Name(stg.Name), StdoutExportStage, stg); err != nil {
			logrus.WithError(err).WithField("config", stg).Fatal("can't instantiate node")
		}
	}
	for _, stg := range cfg.Http {
		if err := graph.NewStart[Http, []byte](builder, stage.Name(stg.Name), HttpIngestStage, stg); err != nil {
			logrus.WithError(err).WithField("config", stg).Fatal("can't instantiate node")
		}
	}
	for _, stg := range cfg.Deleter {
		if err := graph.NewMiddle[Deleter, map[string]any, map[string]any](builder, stage.Name(stg.Name), FieldDeleterStage, stg); err != nil {
			logrus.WithError(err).WithField("config", stg).Fatal("can't instantiate node")
		}
	}
	for src, dsts := range cfg.Connect {
		for _, dst := range dsts {
			if err := builder.Connect(stage.Name(src), stage.Name(dst)); err != nil {
				logrus.WithError(err).
					WithFields(logrus.Fields{"src": src, "dst": dst}).
					Fatal("can't connect stages")
			}
		}
	}
}
