package main

import (
	"fmt"
)

type Call struct {
	needService bool
	timeStamp   time
}

func main() {
	fmt.Println("Hello TTK4145")
	fmt.Println(elevio.pollRate)
}
