package main

import "fmt"

func main() {
	fmt.Printf("=== Testing Arrays ===\n")

	var a [5]int
	fmt.Println("emp:", a)

	a[4] = 100
	fmt.Println("get:", a[4])
	fmt.Println("len", len(a))

}
