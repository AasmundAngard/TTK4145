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
	// Init Hardware
	// Init sync -> Sync starts reading from the network and hardware
	// Sync gives info to main about floor sensors

	// Sync should not broadcast before main says so? Maybe uninitialized tag?

	// If between floors -> floor sensor registers no floors, go down until
	// sync registers a floor and says so to main

	// Read ID, num elevators and num floors from config file

	// Read common hall calls and cab calls from other elevators' broadcasts

	// Update own state with new info in sync

	// Main starts receiving common hall calls,
	// Main enters main loop

	// Main only receives common hall calls from sync
	// Light up hall buttons based on hall calls
	// Light up cab call buttons based on cab calls

	// Main calls sequenceAsigner, assigning current calls and finding direction

	// Main decides direction based on current state (doorOpen and result from sequenceAssigner)

	// If stopButton -> stop
	// If call at current floor -> stop
	// If shouldMove -> Start actuators
	// If shouldOpenDoor -> start door closing countdown
	// if doorOpen and doorCountdown over -> try closing door

}
