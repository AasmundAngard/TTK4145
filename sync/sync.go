package sync

import (
	"root/config"
)

// Channel overview
// hardwareCalls: 	Sync <- HW
// finishedCalls: 	Sync <- Main
// syncedData: 		Sync -> Main

type Call struct {
	NeedService bool
	TimeStamp   int64
}

type HallCalls [config.NumFloors][2]Call
type CabCalls [config.NumFloors]Call
type Calls struct {
	HallCalls HallCalls
	CabCalls  CabCalls
}

type HallCallsBool [config.NumFloors][2]bool
type CabCallsBool [config.NumFloors]bool
type CallsBool struct {
	HallBoolCalls HallCallsBool
	CabBoolCalls  [config.NumElevators]CabCallsBool
}

func Sync(hardwareCalls chan CallsType, finishedCalls chan CallsType, syncedData chan BoolCallsType) {
	var calls CallsType
	var boolCalls BoolCallsType

	for {
		select {
		case incomingHardwareCalls := <-hardwareCalls:
			calls = updateCallData(calls, incomingHardwareCalls)
		case incomingFinishedCalls := <-finishedCalls:
			calls = updateCallData(calls, incomingFinishedCalls)
		}

		boolCalls.HallBoolCalls = hallCallsToBoolCalls(calls.HallCalls)
		boolCalls.CabBoolCalls = cabCallsToBoolCalls(calls.CabCalls)

		syncedData <- boolCalls
	}
}

func updateCallData(current CallsType, incoming CallsType) CallsType {
	for floor := 0; floor < config.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {
			if incoming.HallCalls[floor][btn].TimeStamp.After(current.HallCalls[floor][btn].TimeStamp) {
				current.HallCalls[floor][btn] = incoming.HallCalls[floor][btn]
			}
		}
		if incoming.CabCalls[floor].TimeStamp.After(current.CabCalls[floor].TimeStamp) {
			current.CabCalls[floor] = incoming.CabCalls[floor]
		}
	}
	return current
}

func cabCallsToBoolCalls(c CabCallsType) CabBoolCallsType {
	var b CabBoolCallsType

	for i, e := range c {
		b[i] = e.NeedService
	}

	return b
}

func hallCallsToBoolCalls(c HallCallsType) HallBoolCallsType {
	var b HallBoolCallsType

	for i, e := range c {
		b[i][0] = e[0].NeedService
		b[i][1] = e[1].NeedService
	}

	return b
}
