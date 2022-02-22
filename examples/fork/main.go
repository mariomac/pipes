package main

import (
	"fmt"
	"github.com/mariomac/go-pipes/pkg/pipe"
	"math"
	"math/rand"
	"time"
)

func RandomGenerator(seed int64, elements int) func(out chan<- int64) {
	rand.Seed(seed)
	return func(out chan<- int64) {
		for n := 0; n < elements; n++ {
			out <- rand.Int63n(1000)
		}
	}
}

func MaxCalculator(in <-chan int64) {
	max := int64(math.MinInt64)
	for n := range in {
		if n > max {
			max = n
		}
	}
	fmt.Println("max", max)
}

func MinCalculator(in <-chan int64) {
	min := int64(math.MaxInt64)
	for n := range in {
		if n < min {
			min = n
		}
	}
	fmt.Println("min", min)
}

func main() {
	pipeline := pipe.Start(RandomGenerator(123, 100))
	// A pipeline can be forked into two sub-pipelines
	left, right := pipeline.Fork()
	// and each pipeline branch can be built as a normal pipeline
	left.Add(MaxCalculator)
	right.Add(MinCalculator)
	// running the parent pipeline will run the two sub-pipelines
	pipeline.Run()

	time.Sleep(2 * time.Second)
}