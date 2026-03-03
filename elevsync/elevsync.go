package elevsync

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

const (
	ServicedCall   bool = false
	UnservicedCall      = true
)

type NetworkMsg struct {
	SenderID  int
	TimeStamp int64
	Calls     Calls
	State     State
}

type OtherElevator struct {
	State        State
	CabCallsBool CabCallsBool
}

type syncOtherElevator struct {
	State        State
	CabCalls     CabCalls

}

type SyncedData struct {
	CallsBool      CallsBool
	OtherElevators []OtherElevator
}

type globalCalls struct {
	SenderID int
	Calls    Calls
}

const ElevatorID int = 0
const tolerance int = 10000000 // 10 ms in nanoseconds

func Sync(hardwareCalls <-chan elevio.CallEvent, localState <-chan State, finishedCalls <-chan elevio.CallEvent, networkMsg <-chan NetworkMsg, syncedData chan<- SyncedData) {
	var localCalls Calls
	var globalCallslist []globalCalls

	var confirmedCalls Calls

	// store hwcall i localcalls
	// send localcalls til network
	// oppdater localcalls til synccalls når networkmsg kommer inn
	// send syncedcalls til main

	// hvordan skal sync vite når en localcall er synced?
	// for hver hallcall i localcalls, sjekk om den er i de

	var syncedDataToSend SyncedData
	

	for {
		select {
		case incomingHardwareCall := <-hardwareCalls:
			localCalls = localCalls.updateCall(incomingHardwareCall, UnservicedCall)
			
		case incomingFinishedCall := <-finishedCalls:
			localCalls = localCalls.updateCall(incomingFinishedCall, ServicedCall)

		case incomingNetworkMsg := <-networkMsg:
			localCalls = localCalls.mergeCalls(incomingNetworkMsg.Calls)
		}

		syncedDataToSend.CallsBool.HallCallsBool = localCalls.HallCalls.toBool()
		syncedDataToSend.CallsBool.CabCallsBool = localCalls.CabCalls.toBool()

		syncedData <- syncedDataToSend
	}
}

func (current Calls) updateCall(incoming elevio.CallEvent, callstate bool) Calls {
	floor := incoming.Floor
	btn := incoming.Button
	elevator := elevatorID

	if btn == elevio.BT_HallUp || btn == elevio.BT_HallDown {
		if incoming.TimeStamp > current.HallCalls[floor][btn].TimeStamp {
			current.HallCalls[floor][btn].NeedService = callstate
			current.HallCalls[floor][btn].TimeStamp = incoming.TimeStamp
		}
	} else if btn == elevio.BT_Cab {
		if incoming.TimeStamp > current.CabCalls[ElevatorID][floor].TimeStamp {
			current.CabCalls[ElevatorID][floor].NeedService = callstate
			current.CabCalls[ElevatorID][floor].TimeStamp = incoming.TimeStamp
		}
	} else {
		panic("Invalid ButtonType")
	}

	return current
}

func (current Calls) mergeCalls(incoming Calls) Calls {
	for floor := 0; floor < config.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {
			if incoming.HallCalls[floor][btn].TimeStamp > current.HallCalls[floor][btn].TimeStamp {
				current.HallCalls[floor][btn] = incoming.HallCalls[floor][btn]
			}
		}
	}
}

func (c CabCalls) toBool() CabCallsBool {
	var b CabCallsBool

	for i, e := range c {
		b[i] = e.NeedService
	}

	return b
}

func (c HallCalls) toBool() HallCallsBool {
	var b HallCallsBool

	for i, e := range c {
		b[i][0] = e[0].NeedService
		b[i][1] = e[1].NeedService
	}

	return b
}
