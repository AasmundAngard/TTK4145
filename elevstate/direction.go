package elevstate // Foreløpig

import (
	"root/elevio"
)

type Direction int

const (
	Up   Direction = 0
	Down           = 1
)

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

func (d Direction) ToBtnType() elevio.ButtonType {
	switch d {
	case Up:
		return elevio.BT_HallUp
	case Down:
		return elevio.BT_HallDown
	default:
		return 0
	}
}

func (d Direction) String() string {
	switch d {
	case Up:
		return "up"
	case Down:
		return "down"
	default:
		panic(d)
	}
}
