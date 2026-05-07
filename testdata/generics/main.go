package main

import "fmt"

func main() {
	result := Map([]int{1, 2, 3}, double)
	fmt.Println(result)
}

func Map[T any, U any](input []T, fn func(T) U) []U {
	out := make([]U, len(input))
	for i, v := range input {
		out[i] = fn(v)
	}
	return out
}

func double(x int) int {
	return x * 2
}
