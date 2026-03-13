package elevstate

import (
	"root/elevio"
	"strconv"
)

type Behaviour int

const (
	Idle      Behaviour = 0
	Moving              = 1
	DoorOpen            = 2
	Motorstop           = 3
)

func (b Behaviour) String() string {
	switch b {
	case Idle:
		return "idle"
	case Moving:
		return "moving"
	case DoorOpen:
		return "doorOpen"
	case Motorstop:
		return "motorstop"
	default:
		panic(strconv.Itoa(int(b)))
	}
}

type ElevState struct {
	Behaviour Behaviour
	Floor     int
	Direction Direction
}

func (e ElevState) ToCabCallEvent() elevio.CallEvent {
	return elevio.CallEvent{Floor: e.Floor, Button: elevio.BT_Cab}
}
func (e ElevState) ToHallCallEvent() elevio.CallEvent {
	switch e.Direction {
	case Up:
		return elevio.CallEvent{Floor: e.Floor, Button: elevio.BT_HallUp}
	case Down:
		return elevio.CallEvent{Floor: e.Floor, Button: elevio.BT_HallDown}
	default:
		panic("Invalid Direction to ButtonEvent")
	}
}
