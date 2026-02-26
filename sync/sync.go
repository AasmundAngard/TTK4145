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

type HallCalls [config.NumFloors][2]Call
type CabCalls [config.NumFloors]Call
type Calls struct {
	HallCalls HallCalls
	CabCalls  CabCalls
}

type HallCallsBool [config.NumFloors][2]bool
type CabCallsBool [config.NumFloors]bool
type CallsBool struct {
	HallCallsBool HallCallsBool
	CabCallsBool  [config.NumElevators]CabCallsBool
}

type CallEvent struct {
	Floor  int
	Button ButtonType
	TimeStamp int64
}

const (
	ServicedCall bool = false
	UnservicedCall      = true
)

type networkMsg struct {
	SenderID int
	TimeStamp int64
	Calls    Calls
	state    State
}

type otherElevators []struct {
	State State
	CabCallsBool CabCallsBool
}


type syncedData struct {
	callsBool CallsBool
	otherElevators otherElevators

}

const ElevatorID int = 0
const tolerance int = 100000000 // 100 ms in nanoseconds

func Sync(hardwareCalls chan CallEvent, finishedCalls chan CallEvent, networkMsg chan networkMsg, syncedData chan syncedData) {
	var calls Calls
	var callsBool CallsBool
	var otherElevators otherElevators

	for {
		select {
		case incomingHardwareCalls := <-hardwareCalls:
			calls = updateCall(calls, incomingHardwareCalls, UnservicedCall)
			
		case incomingFinishedCalls := <-finishedCalls:
			calls = updateCall(calls, incomingFinishedCalls, ServicedCall)

		case incomingNetworkMsg := <-networkMsg:
			calls = mergeCalls(calls, incomingNetworkMsg.Calls)


		}

		callsBool.HallCallsBool = hallCallsToBools(calls.HallCalls)
		callsBool.CabCallsBool = cabCallsToBools(calls.CabCalls)

		syncedData <- callsBool
	}
}

func updateCall(current Calls, incoming CallEvent, callstate bool) Calls {
	floor := incoming.Floor
	btn := incoming.Button
	elevator := elevatorID

	if btn == elevio.BT_HallUp || btn == elevio.BT_HallDown {
		if incoming.TimeStamp > current.HallCalls[floor][btn].TimeStamp {
			current.HallCalls[floor][btn].NeedService = callstate
			current.HallCalls[floor][btn].TimeStamp = incoming.TimeStamp
		}
	}
	else if btn == elevio.BT_Cab {
		if incoming.TimeStamp > current.CabCalls[ElevatorID][floor].TimeStamp {
			current.CabCalls[ElevatorID][floor].NeedService = callstate
			current.CabCalls[ElevatorID][floor].TimeStamp = incoming.TimeStamp
		}
	}

	return current
}

func mergeCalls(current Calls, incoming Calls) Calls {
	for floor := 0; floor < config.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {
			if incoming.HallCalls[floor][btn].TimeStamp > current.HallCalls[floor][btn].TimeStamp {
				current.HallCalls[floor][btn] = incoming.HallCalls[floor][btn]
			}
		}
	}
}


func cabCallsToBools(c CabCalls) CabCallsBool {
	var b CabCallsBool

	for i, e := range c {
		b[i] = e.NeedService
	}

	return b
}

func hallCallsToBools(c HallCalls) HallCallsBool {
	var b HallCallsBool

	for i, e := range c {
		b[i][0] = e[0].NeedService
		b[i][1] = e[1].NeedService
	}

	return b
}