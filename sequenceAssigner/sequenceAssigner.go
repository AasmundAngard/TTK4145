package sequenceAssigner

import (
	"root/elevio"
)

type ElevBehaviour int

const (
	Idle ElevBehaviour = iota
	DoorOpen
	Moving
)

type DirnBehaviourPair struct {
	dirn 		elevio.MotorDirection
	behaviour	ElevBehaviour
}

func requestsAbove(requests []Call, currentFloor int) bool {
	for 
}

// Returns next direction and behaviour based on call-requests and current direction and floor
func assign(requests []Call, dir elevio.MotorDirection, currentFloor int) {
	switch dir {
	case elevio.MD_Up:

	default:
		return
	}
}
