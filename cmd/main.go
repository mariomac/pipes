package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mariomac/go-pipes/pkg/pipe"
)

func main() {
	count := 1
	p := pipe.Start(func(out chan<- int) {
		for count <= 10 {
			out <- count
			count++
		}
	})
	p.Add(func(in <-chan int, out chan<- string) {
		for n := range in {
			out <- fmt.Sprint("Received", n)
		}
	})
	p.End(func(in <-chan string) {
		for n := range in {
			fmt.Println(n)
		}
	})
	p.Run()

	time.Sleep(10 * time.Second)
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}
}
