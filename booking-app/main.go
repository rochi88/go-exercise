package main

import "fmt"

func main() {
	const totalSeat uint = 50
	var remainingSeat uint
	var bookings []string

	var firstName string
	var lastName string
	var email string
	var seats uint

	for {
		fmt.Printf("Please enter first name")
		fmt.Scan(&firstName)

		fmt.Printf("Please enter last name")
		fmt.Scan(&lastName)

		fmt.Printf("Please enter email")
		fmt.Scan(&email)

		fmt.Printf("Please enter required seats")
		fmt.Scan(&seats)

		bookings = append(bookings, firstName, " ", lastName)
		remainingSeat = totalSeat - seats

		fmt.Printf("%v", bookings)
		fmt.Printf("Remaining seat: %v", remainingSeat)

	}
}
