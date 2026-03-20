package elevator

import (
	"root/config"
	"root/elevio"
)

type HallCalls [config.NumFloors][2]bool
type CabCalls [config.NumFloors]bool
type Calls struct {
	HallCalls HallCalls
	CabCalls  CabCalls
}

func clearCall(state ElevState, hallCalls *HallCalls, cabCalls *CabCalls, completedCallToSyncC chan<- elevio.CallEvent) {
	if cabCalls[state.Floor] {
		cabCalls[state.Floor] = false
		completedCallToSyncC <- state.ToCabCallEvent()
	}
	if hallCalls[state.Floor][state.Direction] && !state.MotorStop && !state.DoorObstructed {
		hallCalls[state.Floor][state.Direction] = false
		completedCallToSyncC <- state.ToHallCallEvent()
	}
}

func orderInDirection(direction Direction, floor int, hallCalls HallCalls, cabCalls CabCalls) bool {
	switch direction {
	case Up:
		return requestsAbove(hallCalls, cabCalls, floor)
	case Down:
		return requestsBelow(hallCalls, cabCalls, floor)
	default:
		panic("Illegal direction")
	}
}
func requestsAbove(hallCalls HallCalls, cabCalls CabCalls, currentFloor int) bool {
	for f := currentFloor + 1; f < config.NumFloors; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]) {
			return true
		}
	}
	return false
}

func requestsBelow(hallCalls HallCalls, cabCalls CabCalls, currentFloor int) bool {
	for f := 0; f < currentFloor; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]) {
			return true
		}
	}
	return false
}
