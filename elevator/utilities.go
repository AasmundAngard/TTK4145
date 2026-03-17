package elevator

import (
	"root/config"
	"root/elevio"
	"root/elevstate"
	"root/elevsync"
)

func DrainChannel[T any](variableC <-chan T, variable *T) {
drainChannel:
	for {
		select {
		case *variable = <-variableC:
		default:
			break drainChannel
		}
	}
}

func orderDone(state elevstate.ElevState, hCalls *elevsync.HallCallsBool, cCalls *elevsync.CabCallsBool, completedCallToSyncC chan<- elevio.CallEvent) {
	if cCalls[state.Floor] {
		cCalls[state.Floor] = false
		completedCallToSyncC <- state.ToCabCallEvent()
	}
	if hCalls[state.Floor][state.Direction] && !state.MotorStop && !state.DoorObstructed {
		hCalls[state.Floor][state.Direction] = false
		completedCallToSyncC <- state.ToHallCallEvent()
	}
}

func orderInDirection(direction elevstate.Direction, floor int, hallCalls elevsync.HallCallsBool, cabCalls elevsync.CabCallsBool) bool {
	switch direction {
	case elevstate.Up:
		return requestsAbove(hallCalls, cabCalls, floor)
	case elevstate.Down:
		return requestsBelow(hallCalls, cabCalls, floor)
	default:
		panic("Illegal direction")
	}
}
func requestsAbove(hallCalls elevsync.HallCallsBool, cabCalls elevsync.CabCallsBool, currentFloor int) bool {
	for f := currentFloor + 1; f < config.NumFloors; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]) {
			return true
		}
	}
	return false
}

func requestsBelow(hallCalls elevsync.HallCallsBool, cabCalls elevsync.CabCallsBool, currentFloor int) bool {
	for f := 0; f < currentFloor; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]) {
			return true
		}
	}
	return false
}
