package main

import (
	"fmt"
	"math"
)

const c string = "Raisul"

func main() {
	fmt.Println(c)

	const n = 5000
	const d = 3e20 / n
	fmt.Println(d)
	fmt.Println(int64(d))

	fmt.Println(math.Sin(n))
}
