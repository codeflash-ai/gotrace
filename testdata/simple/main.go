package main

import "time"

func main() {
	a()
	b()
}

func a() {
	time.Sleep(10 * time.Millisecond)
	c()
}

func b() {
	time.Sleep(5 * time.Millisecond)
}

func c() {
	time.Sleep(2 * time.Millisecond)
}
