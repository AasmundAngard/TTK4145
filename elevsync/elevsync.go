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
	CabCallsBool  CabCallsBool
}

const (
	ServicedCall   bool = false
	UnservicedCall      = true
)

type NetworkReceiveMsg struct {
	SenderID  int
	TimeStamp int64
	Calls     Calls
	State     State
}
type NetworkTransmitMsg struct {
	Calls Calls
	State State
}

type OtherElevator struct {
	ID        int
	TimeStamp int64
	Calls     Calls
	State     State
	Active    bool
}
type OtherElevatorList []OtherElevator
type OtherElevatorBool struct {
	//ID		   	 int
	State        State
	CabCallsBool CabCallsBool
}

type SyncedData struct {
	CallsBool             CallsBool
	OtherElevatorListBool []OtherElevatorBool
}

const ElevatorID int = 0
const tolerance int = 10000000 // 10 ms in nanoseconds

func Sync(hardwareCalls <-chan elevio.CallEvent, localState <-chan State, finishedCalls <-chan elevio.CallEvent, networkMsg <-chan NetworkReceiveMsg, syncedData chan<- SyncedData) {
	var localCalls Calls
	var OtherElevatorList OtherElevatorList

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
			localCalls.update(incomingHardwareCall, UnservicedCall)

		case incomingFinishedCall := <-finishedCalls:
			localCalls.update(incomingFinishedCall, ServicedCall)

		case incomingNetworkMsg := <-networkMsg:
			OtherElevatorList.update(incomingNetworkMsg)
			localCalls.mergeHallCalls(incomingNetworkMsg.Calls)
		}

		confirmedCalls = localCalls.confirm(otherElevatorList)

		syncedDataToSend.CallsBool.HallCallsBool = confirmedCalls.HallCalls.toBool()
		syncedDataToSend.CallsBool.CabCallsBool = confirmedCalls.CabCalls.toBool()
		syncedDataToSend.OtherElevatorListBool = OtherElevatorList.toBool()

		syncedData <- syncedDataToSend
	}
}

func (OtherElevatorList OtherElevatorList) update(incomingNetworkMsg NetworkReceiveMsg) {
	elevatorFound := false

	for i, otherElevator := range OtherElevatorList {
		if otherElevator.ID == incomingNetworkMsg.SenderID {
			if otherElevatorList[i].TimeStamp < incomingNetworkMsg.TimeStamp {
				OtherElevatorList[i].State = incomingNetworkMsg.State
				OtherElevatorList[i].Calls = incomingNetworkMsg.Calls
			}
			elevatorFound = true
			break
		}
	}

	if !elevatorFound {
		OtherElevatorList = append(OtherElevatorList, OtherElevator{ID: incomingNetworkMsg.SenderID, State: incomingNetworkMsg.State, CabCalls: incomingNetworkMsg.Calls.CabCalls})
	}

	return
}

func (OtherElevatorList OtherElevatorList) toBool() []OtherElevatorBool {
	var OtherElevatorListBool []OtherElevatorBool

	for _, otherElevator := range OtherElevatorList {
		OtherElevatorListBool = append(OtherElevatorListBool, OtherElevatorBool{ID: otherElevator.ID, State: otherElevator.State, CabCallsBool: otherElevator.CabCalls.toBool()})
	}

	return OtherElevatorListBool
}

func (current Calls) confirm(otherElevatorList OtherElevatorList) Calls {
	var confirmedCalls Calls

	for floor := 0; floor < config.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {
			for 
}

func (current Calls) update(incoming elevio.CallEvent, callstate bool) {
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

	return
}

func (current Calls) mergeHallCalls(incoming Calls) Calls {
	for floor := 0; floor < config.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {
			if incoming.HallCalls[floor][btn].TimeStamp > current.HallCalls[floor][btn].TimeStamp {
				current.HallCalls[floor][btn] = incoming.HallCalls[floor][btn]
			}
		}
	}
	return
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
