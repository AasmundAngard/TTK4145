package elevstate

import (
	"root/elevio"
	"root/sync"
)

type Behaviour int

const (
	Idle     Behaviour = 0
	Moving             = 1
	DoorOpen           = 2
)

type ElevState struct {
	Behaviour Behaviour
	Floor     int
	Direction Direction
}

func (e ElevState) ToCabCallEvent() sync.CallEvent {
	return sync.CallEvent{Floor: e.Floor, Button: elevio.BT_Cab}
}
func (e ElevState) ToHallCallEvent() sync.CallEvent {
	switch e.Direction {
	case Up:
		return sync.CallEvent{Floor: e.Floor, Button: elevio.BT_HallUp}
	case Down:
		return sync.CallEvent{Floor: e.Floor, Button: elevio.BT_HallDown}
	default:
		panic("Invalid Direction to ButtonEvent")
	}
}
