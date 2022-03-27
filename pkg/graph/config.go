package graph

import (
	"fmt"
	"reflect"

	"github.com/mariomac/pipes/pkg/graph/stage"
	"github.com/sirupsen/logrus"
)

const connectionsFieldName = "Connector"

// Connections key: ID of the source node. Value: array of destination node IDs.
type Connector map[string][]string

type ConnectionProvider interface {
	Connections() map[string][]string
}

func (c Connector) Connections() map[string][]string {
	return c
}

// ApplyConfig instantiates and configures the different pipeline stages according to the provided PipeConfig
func ApplyConfig[T ConnectionProvider](cfg T, builder *Builder) error {
	cv := reflect.ValueOf(cfg)
	if cv.Kind() == reflect.Pointer {
		if err := applyConfig(builder, cv.Elem()); err != nil {
			return err
		}
	} else {
		if err := applyConfig(builder, cv); err != nil {
			return err
		}
	}

	for src, dsts := range cfg.Connections() {
		for _, dst := range dsts {
			if err := builder.Connect(stage.InstanceID(src), stage.InstanceID(dst)); err != nil {
				logrus.WithError(err).
					WithFields(logrus.Fields{"src": src, "dst": dst}).
					Fatal("can't connect stages")
			}
		}
	}
	return nil
}

func applyConfig(b *Builder, cfgValue reflect.Value) error {
	if cfgValue.Kind() != reflect.Struct {
		return fmt.Errorf("configuration should be a struct. Was: %s", cfgValue.Type())
	}
	valType := cfgValue.Type()
	for f := 0; f < valType.NumField(); f++ {
		field := valType.Field(f)
		if field.Name == connectionsFieldName {
			continue
		}
		fieldVal := cfgValue.Field(f)
		if fieldVal.Type().Kind() == reflect.Array || fieldVal.Type().Kind() == reflect.Slice {
			for nf := 0; nf < fieldVal.Len(); nf++ {
				if err := applyField(b, fieldVal.Index(nf)); err != nil {
					return err
				}
			}
		} else {
			if err := applyField(b, cfgValue.Field(f)); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyField(b *Builder, field reflect.Value) error {
	instancer, ok := field.Interface().(stage.InstanceIDProvider)
	if !ok {
		return fmt.Errorf("field of type %s should provide an 'ID() InstanceID' method."+
			" Did you forgot to embed the stage.Instance field? ", field.Type())
	}
	return instantiate(b, instancer.ID(), field)
}
