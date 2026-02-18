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
	direction   elevio.MotorDirection
}

type CallList struct {
	hallCalls	[]Call // 2n elementer to Call-objekt for hver etasje, en for opp og en for ned: 
	// [Call_etg1_opp, Call_etg1_ned, Call_etg2_opp, Call_etg2_ned, ..., Call_etgn_opp, Call_etgn_ned]
	cabCalls	[]Call // n elementer, ett Call-objekt for hver etasje
}


func main() {
	fmt.Println("Hello TTK4145")
	fmt.Println(elevio.pollRate)
}
