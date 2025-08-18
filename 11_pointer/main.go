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

	val := 1

	fmt.Println("Initial: ", val)
	zeroVal(val)
	fmt.Println("Zeroval: ", val)
	zeroPVal(&val)
	fmt.Println("ZeroPVal: ", val)
	fmt.Println("Pointer address", &val)

}
