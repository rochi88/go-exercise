package main

import "fmt"

func main() {
	var name string = "Raisul"
	const days int = 7
	var x, y int = 1, 2
	var z uint // Variables declared without a corresponding initialization are zero-valued.
	temp := 1

	fmt.Println("Checking the uses of variables")
	fmt.Printf("Name is %v,\nDays is %v,\nx is %v,\ny is %v,\nz is %v,\ntemp is %v\n", name, days, x, y, z, temp)
}
