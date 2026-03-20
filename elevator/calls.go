package elevator

import (
	"root/config"
	"root/elevio"
)

type HallCallsBool [config.NumFloors][2]bool
type CabCallsBool [config.NumFloors]bool
type Calls struct {
	HallCalls HallCallsBool
	CabCalls  CabCallsBool
}

func clearCall(state ElevState, hallCalls *HallCallsBool, cabCalls *CabCallsBool, completedCallToSyncC chan<- elevio.CallEvent) {
	if cabCalls[state.Floor] {
		cabCalls[state.Floor] = false
		completedCallToSyncC <- state.ToCabCallEvent()
	}
	if hallCalls[state.Floor][state.Direction] && !state.MotorStop && !state.DoorObstructed {
		hallCalls[state.Floor][state.Direction] = false
		completedCallToSyncC <- state.ToHallCallEvent()
	}
}

func orderInDirection(direction Direction, floor int, hallCalls HallCallsBool, cabCalls CabCallsBool) bool {
	switch direction {
	case Up:
		return requestsAbove(hallCalls, cabCalls, floor)
	case Down:
		return requestsBelow(hallCalls, cabCalls, floor)
	default:
		panic("Illegal direction")
	}
}
func requestsAbove(hallCalls HallCallsBool, cabCalls CabCallsBool, currentFloor int) bool {
	for f := currentFloor + 1; f < config.NumFloors; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]) {
			return true
		}
	}
	return false
}

func requestsBelow(hallCalls HallCallsBool, cabCalls CabCallsBool, currentFloor int) bool {
	for f := 0; f < currentFloor; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]) {
			return true
		}
	}
	return false
}
