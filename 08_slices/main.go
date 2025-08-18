package main

import "fmt"

func main() {
	fmt.Println("=== Testing Slices ===")

	var slc = []int{4, 4, 6, 8}

	fmt.Printf("Length %d\n", len(slc))
	fmt.Printf("Capacity %d\n", cap(slc))
}
