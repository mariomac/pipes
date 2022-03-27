package stages

import (
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/mariomac/pipes/pkg/graph"
	"io"
	"io/ioutil"
)

type PipeConfig struct {
	Http            []Http    `hcl:"http,block"`
	StdOut          []Stdout  `hcl:"stdout,block"`
	Deleter         []Deleter `hcl:"deleter,block"`
	graph.Connector `hcl:"connect"`
}

func ReadConfig(in io.Reader) (PipeConfig, error) {
	src, err := ioutil.ReadAll(in)
	if err != nil {
		return PipeConfig{}, err
	}
	var pc PipeConfig
	err = hclsimple.Decode(".hcl", src, nil, &pc)
	return pc, err
}
