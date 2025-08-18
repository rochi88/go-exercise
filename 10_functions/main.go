package main

import "fmt"

// Variadic Functions
func sum(nums ...int) {
	total := 0
	for _, val := range nums {
		total += val
	}

	fmt.Println(total)
}

func main() {
	fmt.Println("=== Testing Functions ===")

	nums := []int{3, 4, 8}

	// Calling Variadic Functions
	sum(nums...)
}
