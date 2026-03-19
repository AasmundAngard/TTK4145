package elevsync

import (
	"root/config"
	"root/elevio"
	"root/elevstate"
	"strconv"
)

type call struct {
	NeedService bool
	Version     int64
}

const (
	ServicedCall   bool = false
	UnservicedCall      = true
)

type hallCalls [config.NumFloors][2]call
type CabCalls [config.NumFloors]call
type Calls struct {
	HallCalls hallCalls
	CabCalls  CabCalls
}

func (c hallCalls) confirm() ConfirmedHallCalls {
	var b ConfirmedHallCalls

	for i, e := range c {
		b[i][0] = e[0].NeedService
		b[i][1] = e[1].NeedService
	}

	return b
}

func (c CabCalls) confirm() ConfirmedCabCalls {
	var b ConfirmedCabCalls

	for i, e := range c {
		b[i] = e.NeedService
	}

	return b
}

func newCabCalls() CabCalls {
	var cabCalls CabCalls
	for floor := 0; floor < config.NumFloors; floor++ {
		cabCalls[floor].NeedService = false
		cabCalls[floor].Version = 0
	}
	return cabCalls
}

type ConfirmedHallCalls [config.NumFloors][2]bool
type ConfirmedCabCalls [config.NumFloors]bool
type ConfirmedCalls struct {
	HallCalls ConfirmedHallCalls
	CabCalls  ConfirmedCabCalls
}

func (h ConfirmedHallCalls) HasCalls() bool {
	for _, floor := range h {
		if floor[0] == true || floor[1] == true {
			return true
		}
	}
	return false
}
func (h ConfirmedCabCalls) HasCalls() bool {
	for _, floor := range h {
		if floor == true {
			return true
		}
	}
	return false
}

func (self *Calls) mergeHallCalls(incoming Calls) {
	for floor := 0; floor < config.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {
			if incoming.HallCalls[floor][btn].Version > self.HallCalls[floor][btn].Version {
				(*self).HallCalls[floor][btn] = incoming.HallCalls[floor][btn]
			}
		}
	}
}

func (self *Calls) mergeCabCalls(cabCallsLists []CabCalls) {
	mergedCabCalls := newCabCalls()

	for _, cabCalls := range cabCallsLists {
		for floor := 0; floor < config.NumFloors; floor++ {
			if cabCalls[floor].Version > mergedCabCalls[floor].Version {
				mergedCabCalls[floor] = cabCalls[floor]
			}
		}
	}

	for floor := 0; floor < config.NumFloors; floor++ {
		mergedCabCalls[floor].NeedService = mergedCabCalls[floor].NeedService || self.CabCalls[floor].NeedService
	}

	(*self).CabCalls = mergedCabCalls
}

func (self Calls) decideCommonCalls(peerElevators peerElevatorList, selfState elevstate.ElevState) ConfirmedCalls {
	var commonCalls ConfirmedCalls
	commonCalls.HallCalls = self.HallCalls.confirm()
	commonCalls.CabCalls = self.CabCalls.confirm()

	for floor := 0; floor < config.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {
			if !self.HallCalls[floor][btn].NeedService || selfState.MotorStop || selfState.DoorObstructed {
				continue
			}

			confirmed := true
			for _, peerElevator := range peerElevators {
				if !peerElevator.Alive || peerElevator.State.MotorStop || peerElevator.State.DoorObstructed {
					continue
				}

				if !peerElevator.Calls.HallCalls[floor][btn].NeedService || peerElevator.Calls.HallCalls[floor][btn].Version != self.HallCalls[floor][btn].Version {
					confirmed = false
					commonCalls.HallCalls[floor][btn] = false
					break
				}
			}

			if confirmed {
				commonCalls.HallCalls[floor][btn] = true
			}
		}
	}

	return commonCalls
}

func (self *Calls) addCall(incoming elevio.CallEvent) {
	floor := incoming.Floor
	btn := incoming.Button
	switch btn {
	case elevio.BT_HallUp, elevio.BT_HallDown:
		if self.HallCalls[floor][btn].NeedService != UnservicedCall {
			(*self).HallCalls[floor][btn].NeedService = UnservicedCall
			(*self).HallCalls[floor][btn].Version++
		}
	case elevio.BT_Cab:
		if self.CabCalls[floor].NeedService != UnservicedCall {
			(*self).CabCalls[floor].NeedService = UnservicedCall
			(*self).CabCalls[floor].Version++
		}
	default:
		panic("Invalid ButtonType " + strconv.Itoa(int(btn)))
	}
}

func (self *Calls) removeCall(incoming elevio.CallEvent) {
	floor := incoming.Floor
	btn := incoming.Button
	switch btn {
	case elevio.BT_HallUp, elevio.BT_HallDown:
		if self.HallCalls[floor][btn].NeedService != ServicedCall {
			(*self).HallCalls[floor][btn].NeedService = ServicedCall
			(*self).HallCalls[floor][btn].Version++
		}
	case elevio.BT_Cab:
		if self.CabCalls[floor].NeedService != ServicedCall {
			(*self).CabCalls[floor].NeedService = ServicedCall
			(*self).CabCalls[floor].Version++
		}
	default:
		panic("Invalid ButtonType " + strconv.Itoa(int(btn)))
	}
}
