package main

import "fmt"

func main() {
	i := 1

	for i <= 5 {
		fmt.Println(i)
		i++
	}

	for j := 1; j <= 10; j++ {
		fmt.Println(j)
	}

	for i := 1; i <= 10; i++ {
		fmt.Println(i)
	}

	for i := range 8 {
		fmt.Println(i)
	}

	for i := range "Raisul" {
		fmt.Println(i)
	}

	for i, c := range "Raisul" {
		fmt.Println(i, c)
	}

	for {
		fmt.Println("Hello")
		break
	}

	for n := range 6 {
		if n%2 == 0 {
			continue
		}
		fmt.Println(n)
	}
}
