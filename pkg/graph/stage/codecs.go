package stage

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

func BytesToStringCodec(in <-chan []byte, out chan<- string) {
	for i := range in {
		out <- string(i)
	}
}

func JSONBytesToMapCodec(in <-chan []byte, out chan<- map[string]interface{}) {
	log := logrus.WithField("codec", "JSONBytesToMapCodec")
	for i := range in {
		var m map[string]interface{}
		if err := json.Unmarshal(i, &m); err != nil {
			log.WithError(err).WithField("json", string(i)).Warn("skipping record")
			continue
		}
		out <- m
	}
}

func MapToStringCodec(in <-chan map[string]interface{}, out chan<- string) {
	log := logrus.WithField("codec", "MapToStringCodec")
	for i := range in {
		bytes, err := json.Marshal(i)
		if err != nil {
			log.WithError(err).WithField("map", i).Warn("skipping record")
			continue
		}
		out <- string(bytes)
	}
}
