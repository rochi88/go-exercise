package closures

import "fmt"

func main() {
	fmt.Println("=== Testing Closures ===")

	// Defining a closure that captures a variable from its surrounding scope
	closureFunc := func(x int) int {
		return x * x
	}

	// Using the closure
	num := 5
	result := closureFunc(num)
	fmt.Printf("The square of %d is %d\n", num, result)

	// Another example with a counter closure
	counter := func() func() int {
		count := 0
		return func() int {
			count++
			return count
		}
	}()

	fmt.Println("Counter:", counter()) // 1
	fmt.Println("Counter:", counter()) // 2
	fmt.Println("Counter:", counter()) // 3

	fmt.Println("=== End Testing Closures ===")
}
