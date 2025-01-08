package main

import "fmt"

func main() {
	var name string = "Raisul"
	const days int = 7
	var x, y int = 1, 2
	var z uint // Variables declared without a corresponding initialization are zero-valued.
	temp := 1

	fmt.Println("Checking the uses of variables")
	fmt.Printf("Name is %v, Days is %v, x is %v, y is %v, z is %v, temp is %v\n", name, days, x, y, z, temp)
}
