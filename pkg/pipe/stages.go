package pipe

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"
)

type Record map[string]string

func Ingest() <-chan string {
	out := make(chan string)
	go func() {
		cnt := 1
		for {
			out <- fmt.Sprintf(`{"Foo":"%d","Bar":"%d"}`, cnt, rand.Intn(1000))
			cnt++
			time.Sleep(time.Second)
		}
	}()
	return out
}

func JSONToRecord(in <-chan string) <-chan Record {
	out := make(chan Record)
	go func() {
		for ir := range in {
			or := Record{}
			if err := json.Unmarshal([]byte(ir), &or); err != nil {
				log.Println("error transforming:", err.Error())
			} else {
				out <- or
			}
		}
	}()
	return out
}

func Appender(in <-chan Record, key, val string) <-chan Record {
	out := make(chan Record)
	go func() {
		for ir := range in {
			ir[key] = val
			out <- ir
		}
	}()
	return out
}

func RecordToLine(in <-chan Record) <-chan string {
	out := make(chan string)
	go func() {
		for ir := range in {
			out <- fmt.Sprintf("%+v", ir)
		}
	}()
	return out
}

func Print(in <-chan string) {
	go func() {
		for r := range in {
			log.Println(r)
		}
	}()
}
