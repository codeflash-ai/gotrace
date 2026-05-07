package main

import (
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		worker("alpha")
	}()

	go func() {
		defer wg.Done()
		worker("beta")
	}()

	wg.Wait()
}

func worker(name string) {
	time.Sleep(10 * time.Millisecond)
	process(name)
}

func process(name string) {
	time.Sleep(5 * time.Millisecond)
}
