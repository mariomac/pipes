package graph

import (
	"fmt"
	"reflect"

	"github.com/mariomac/pipes/pkg/graph/stage"
	"github.com/sirupsen/logrus"
)

// Connector key: ID of the source node. Value: array of destination node IDs.
type Connector map[string][]string

// Connections returns the map holded by the Connector
func (c Connector) Connections() map[string][]string {
	return c
}

var connectorType = reflect.TypeOf(Connector{})

type ConnectedConfig interface {
	Connections() map[string][]string // TODO: try using InstanceID instead of string
}

// TODO: make this private and invoke directly from Build (adding the config argument)
// applyConfig instantiates and configures the different pipeline stages according to the provided configuration
func (b *Builder) applyConfig(cfg ConnectedConfig) error {
	cv := reflect.ValueOf(cfg)
	if cv.Kind() == reflect.Pointer {
		if err := b.applyConfigReflect(cv.Elem()); err != nil {
			return err
		}
	} else {
		if err := b.applyConfigReflect(cv); err != nil {
			return err
		}
	}

	for src, dsts := range cfg.Connections() {
		for _, dst := range dsts {
			if err := b.connect(src, dst); err != nil {
				logrus.WithError(err).
					WithFields(logrus.Fields{"src": src, "dst": dst}).
					Fatal("can't connect stages")
			}
		}
	}
	return nil
}

func (b *Builder) applyConfigReflect(cfgValue reflect.Value) error {
	if cfgValue.Kind() != reflect.Struct {
		return fmt.Errorf("configuration should be a struct. Was: %s", cfgValue.Type())
	}
	valType := cfgValue.Type()
	for f := 0; f < valType.NumField(); f++ {
		field := valType.Field(f)
		if field.Type == connectorType {
			continue
		}
		fieldVal := cfgValue.Field(f)
		if fieldVal.Type().Kind() == reflect.Array || fieldVal.Type().Kind() == reflect.Slice {
			for nf := 0; nf < fieldVal.Len(); nf++ {
				if err := b.applyField(fieldVal.Index(nf)); err != nil {
					return err
				}
			}
		} else {
			if err := b.applyField(cfgValue.Field(f)); err != nil {
				return err
			}
		}
	}
	return nil
}

var graphInstanceType = reflect.TypeOf(stage.Instance(""))

func (b *Builder) applyField(field reflect.Value) error {
	instancer, ok := field.Interface().(stage.Instancer)
	if !ok {
		// if it does not implement the instancer interface, let's check if it can be converted
		// to the convenience stage.Instance type
		if !field.Type().ConvertibleTo(graphInstanceType) {
			return fmt.Errorf("field of type %s should provide an 'ID() InstanceID' method."+
				" Did you forgot to embed the stage.Instance field? ", field.Type())
		}
		instancer = field.Convert(graphInstanceType).Interface().(stage.Instance)
	}
	return instantiate(b, instancer.ID(), field)
}
