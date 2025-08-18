package main

import "fmt"

func main() {
	fmt.Println("=== Testing Maps ===")

	var m = make(map[int]string)

	m[1] = "Cow"
	m[2] = "Goat"

	fmt.Println(m)
}
