package elevator

import (
	"root/config"
	"root/elevio"
	"root/elevstate"
	"root/elevsync"
)

func clearCall(state elevstate.ElevState, hallCalls *elevsync.ConfirmedHallCalls, cabCalls *elevsync.ConfirmedCabCalls, completedCallToSyncC chan<- elevio.CallEvent) {
	if cabCalls[state.Floor] {
		cabCalls[state.Floor] = false
		completedCallToSyncC <- state.ToCabCallEvent()
	}
	if hallCalls[state.Floor][state.Direction] && !state.MotorStop && !state.DoorObstructed {
		hallCalls[state.Floor][state.Direction] = false
		completedCallToSyncC <- state.ToHallCallEvent()
	}
}

func callInDirection(direction elevstate.Direction, floor int, hallCalls elevsync.ConfirmedHallCalls, cabCalls elevsync.ConfirmedCabCalls) bool {
	switch direction {
	case elevstate.Up:
		return callsAbove(hallCalls, cabCalls, floor)
	case elevstate.Down:
		return callsBelow(hallCalls, cabCalls, floor)
	default:
		panic("Illegal direction")
	}
}
func callsAbove(hallCalls elevsync.ConfirmedHallCalls, cabCalls elevsync.ConfirmedCabCalls, currentFloor int) bool {
	for f := currentFloor + 1; f < config.NumFloors; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]) {
			return true
		}
	}
	return false
}

func callsBelow(hallCalls elevsync.ConfirmedHallCalls, cabCalls elevsync.ConfirmedCabCalls, currentFloor int) bool {
	for f := 0; f < currentFloor; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]) {
			return true
		}
	}
	return false
}
