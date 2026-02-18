package main

import (
	"fmt"
)

type Behaviour int

const (
	idle     Behaviour = 0
	moving             = 1
	doorOpen           = 2
)

type Call struct {
	needService bool
	timeStamp   time
}

type ElevState struct {
	behaviour   Behaviour
	floor       int
	direction   MotorDirection
	cabRequests []bool
	timeStamp   time //?
}

func main() {
	fmt.Println("Hello TTK4145")
	fmt.Println(elevio.pollRate)
}
