package main

import "fmt"

func zeroVal(ival int) {
	ival = 0
}

func zeroPVal(iptr *int) {
	*iptr = 0
}

func main() {
	fmt.Println("=== Testing Pointer ===")

	// Demonstrating pointer usage
	val := 1
	i, j := 45, 89
	fmt.Println(&i, &j)

	fmt.Println("Initial: ", val)
	zeroVal(val)
	fmt.Println("Zeroval: ", val)
	zeroPVal(&val)
	fmt.Println("ZeroPVal: ", val)
	fmt.Println("Pointer address", &val)

	// Exploring pointer types and nil pointers
	var p *int
	fmt.Println("Pointer ", &p)

	fmt.Println("=== End Testing Pointer ===")
}
