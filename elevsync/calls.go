package elevsync

import (
	"root/config"
	"root/elevio"
	"root/elevstate"
	"strconv"
)

type Call struct {
	NeedService bool
	TimeStamp   int64
}

const (
	ServicedCall   bool = false
	UnservicedCall      = true
)

type HallCalls [config.NumFloors][2]Call
type CabCalls [config.NumFloors]Call
type Calls struct {
	HallCalls HallCalls
	CabCalls  CabCalls
}

func (c HallCalls) toBool() HallCallsBool {
	var b HallCallsBool

	for i, e := range c {
		b[i][0] = e[0].NeedService
		b[i][1] = e[1].NeedService
	}

	return b
}

func (c CabCalls) toBool() CabCallsBool {
	var b CabCallsBool

	for i, e := range c {
		b[i] = e.NeedService
	}

	return b
}

func newCabCalls() CabCalls {
	var cabCalls CabCalls
	for floor := 0; floor < config.NumFloors; floor++ {
		cabCalls[floor].NeedService = false
		cabCalls[floor].TimeStamp = 0
	}
	return cabCalls
}

type HallCallsBool [config.NumFloors][2]bool
type CabCallsBool [config.NumFloors]bool
type CallsBool struct {
	HallCallsBool HallCallsBool
	CabCallsBool  CabCallsBool
}

func (h HallCallsBool) HasCalls() bool {
	for _, floor := range h {
		if floor[0] == true || floor[1] == true {
			return true
		}
	}
	return false
}
func (h CabCallsBool) HasCalls() bool {
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
			if incoming.HallCalls[floor][btn].TimeStamp > self.HallCalls[floor][btn].TimeStamp {
				self.HallCalls[floor][btn] = incoming.HallCalls[floor][btn]
			}
		}
	}
}

func (self *Calls) mergeCabCalls(incomingCabCallsLists []CabCalls) {
	mergedCabCalls := newCabCalls()

	for _, cabCalls := range incomingCabCallsLists {
		for floor := 0; floor < config.NumFloors; floor++ {
			if cabCalls[floor].TimeStamp > mergedCabCalls[floor].TimeStamp {
				mergedCabCalls[floor] = cabCalls[floor]
			}
		}
	}

	for floor := 0; floor < config.NumFloors; floor++ {
		mergedCabCalls[floor].NeedService = mergedCabCalls[floor].NeedService || self.CabCalls[floor].NeedService
		mergedCabCalls[floor].TimeStamp++
	}

	self.CabCalls = mergedCabCalls
}

func (self Calls) decideCommonCalls(otherElevatorList OtherElevatorList, localState elevstate.ElevState) CallsBool {
	var confirmedCalls CallsBool
	confirmedCalls.HallCallsBool = self.HallCalls.toBool()
	confirmedCalls.CabCallsBool = self.CabCalls.toBool()

	for floor := 0; floor < config.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {
			if self.HallCalls[floor][btn].NeedService == false || localState.MotorStop == true || localState.DoorObstructed == true {
				continue
			}

			confirmed := true
			for _, otherElevator := range otherElevatorList {
				if otherElevator.Alive == false || otherElevator.State.MotorStop == true || otherElevator.State.DoorObstructed == true {
					continue
				}

				if otherElevator.Calls.HallCalls[floor][btn].NeedService == false || otherElevator.Calls.HallCalls[floor][btn].TimeStamp != self.HallCalls[floor][btn].TimeStamp {
					// If the other elevator does not have the same call or has a different timestamp, we do not confirm it
					confirmed = false
					confirmedCalls.HallCallsBool[floor][btn] = false
					break
				}
			}

			if confirmed {
				confirmedCalls.HallCallsBool[floor][btn] = true
			}
		}
	}

	return confirmedCalls
}

func (self *Calls) addCall(incoming elevio.CallEvent) {
	floor := incoming.Floor
	btn := incoming.Button
	switch btn {
	case elevio.BT_HallUp, elevio.BT_HallDown:
		if self.HallCalls[floor][btn].NeedService != UnservicedCall {
			self.HallCalls[floor][btn].NeedService = UnservicedCall
			self.HallCalls[floor][btn].TimeStamp++
		}
	case elevio.BT_Cab:
		if self.CabCalls[floor].NeedService != UnservicedCall {
			self.CabCalls[floor].NeedService = UnservicedCall
			self.CabCalls[floor].TimeStamp++
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
			self.HallCalls[floor][btn].NeedService = ServicedCall
			self.HallCalls[floor][btn].TimeStamp++
		}
	case elevio.BT_Cab:
		if self.CabCalls[floor].NeedService != ServicedCall {
			self.CabCalls[floor].NeedService = ServicedCall
			self.CabCalls[floor].TimeStamp++
		}
	default:
		panic("Invalid ButtonType " + strconv.Itoa(int(btn)))
	}
}
