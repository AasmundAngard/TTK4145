package sync

import (
	"root/config"
	"root/elevio"
)

// Channel overview
// hardwareCalls: 	Sync <- HW
// finishedCalls: 	Sync <- Main
// syncedData: 		Sync -> Main

type Call struct {
	NeedService bool
	TimeStamp   int64
}

type HallCallsType [config.NumFloors][2]Call
type CabCallsType [config.NumFloors]Call
type CallsType struct {
	HallCalls HallCallsType
	CabCalls  CabCallsType
}

type HallBoolCallsType [config.NumFloors][2]bool
type CabBoolCallsType [config.NumFloors]bool
type BoolCallsType struct {
	HallBoolCalls HallBoolCallsType
	CabBoolCalls  CabBoolCallsType
}

type CallEvent struct {
	Floor  int
	Button ButtonType
	TimeStamp int64
}

func Sync(hardwareCalls chan CallEvent, finishedCalls chan CallEvent, syncedData chan BoolCallsType) {
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

func updateCallData(current CallsType, incoming CallEvent) CallsType {
	floor := incoming.Floor
	btn := incoming.Button

	if btn == elevio.BT_HallUp || btn == elevio.BT_HallDown {
		if incoming.HallCalls[floor][btn].TimeStamp.After(current.HallCalls[floor][btn].TimeStamp) {
		current.HallCalls[floor][btn] = incoming.HallCalls[floor][btn]
		}
	}
	else if btn == elevio.BT_Cab {
		if incoming.CabCalls[floor].TimeStamp.After(current.CabCalls[floor].TimeStamp) {
			current.CabCalls[floor] = incoming.CabCalls[floor]
		}
	}

	return current


	//for floor := 0; floor < config.NumFloors; floor++ {
	//	for btn := 0; btn < 2; btn++ {
	//		if incoming.HallCalls[floor][btn].TimeStamp.After(current.HallCalls[floor][btn].TimeStamp) {
	//			current.HallCalls[floor][btn] = incoming.HallCalls[floor][btn]
	//		}
	//	}
	//	if incoming.CabCalls[floor].TimeStamp.After(current.CabCalls[floor].TimeStamp) {
	//		current.CabCalls[floor] = incoming.CabCalls[floor]
	//	}
	//}
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
