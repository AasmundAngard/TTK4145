package elevator

import (
	"root/elevio"
	"strconv"
)

type Behaviour int

const (
	Idle     Behaviour = 0
	Moving             = 1
	DoorOpen           = 2
)

type Direction int

const (
	Up   Direction = 0
	Down Direction = 1
)

type ElevState struct {
	Behaviour      Behaviour
	Floor          int
	Direction      Direction
	MotorStop      bool
	DoorObstructed bool
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

func (b Behaviour) String() string {
	switch b {
	case Idle:
		return "idle"
	case Moving:
		return "moving"
	case DoorOpen:
		return "doorOpen"
	default:
		panic(strconv.Itoa(int(b)))
	}
}

func (d Direction) Opposite() Direction {
	switch d {
	case Up:
		return Down
	case Down:
		return Up
	default:
		panic("Invalid Direction")
	}
}

func (d Direction) ToMD() elevio.MotorDirection {
	switch d {
	case Up:
		return elevio.MD_Up
	case Down:
		return elevio.MD_Down
	default:
		panic("Invalid Direction")
	}
}

func (d Direction) String() string {
	switch d {
	case Up:
		return "up"
	case Down:
		return "down"
	default:
		panic(strconv.Itoa(int(d)))
	}
}
