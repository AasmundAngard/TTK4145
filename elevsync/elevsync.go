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
type CabCallsList []CabCalls
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
	Alive    bool
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

func Sync(hardwareCalls <-chan elevio.CallEvent, localState <-chan State, finishedCalls <-chan elevio.CallEvent, networkMsg <-chan NetworkReceiveMsg, syncedData chan<- SyncedData, cabCallsRequest <-chan string, cabCallsReceive <-chan CabCallsList, cabCallsSend chan<- CabCalls) {
	var localCalls Calls
	var OtherElevatorList OtherElevatorList

	var confirmedCalls CallsBool
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

		case incomingCabCallsList := <- cabCallsReceive:
			localCalls.mergeCabCalls(incomingCabCallsList)

		case ID := <- cabCallsRequest:
			cabCallsSend <- otherElevatorList.getCabCallsfromID(ID)
			continue
		}

		confirmedCalls = localCalls.decideCommonCalls(otherElevatorList)

		syncedDataToSend.format(confirmedCalls, OtherElevatorList)

		syncedData <- syncedDataToSend
	}
}

func (syncedData SyncedData) format(confirmedCalls CabCalls, OtherElevatorList OtherElevatorList) {
	syncedData.CallsBool = confirmedCalls
	syncedData.OtherElevatorListBool = OtherElevatorList.toBool()

	return
}

func (otherElevatorList OtherElevatorList) getCabCallsfromID(ID int) CabCalls {} {
	var cabCalls CabCalls

	for _, otherElevator := range otherElevatorList {
		if otherElevator.ID == ID {
			return otherElevator.CabCalls
		}
	}
	panic("ID not found in OtherElevatorList when fetching cabcalls")
	return
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

func (current Calls) decideCommonCalls(otherElevatorList OtherElevatorList) {
	var confirmedCalls CallsBool
	confirmedCalls.HallCallsBool = current.HallCalls.toBool()
	confirmedCalls.CabCallsBool = current.CabCalls.toBool()

	for floor := 0; floor < config.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {
			if current.HallCalls[floor][btn] == false {
				continue
			}

			confirmed := true
			for _, otherElevator := range otherElevatorList {
				if otherElevator.Calls.HallCalls[floor][btn].NeedService == false || otherElevator.Calls.HallCalls[floor][btn].TimeStamp != current.HallCalls[floor][btn].TimeStamp {
					confirmed = false
					confirmedCalls.HallCallsBool[floor][btn] = false
					break
				}
			}

			if confirmed{
				confirmedCalls.HallCallsBool[floor][btn] = true
			}
		}
	}

	return confirmedCalls
}

func (current Calls) update(incoming elevio.CallEvent, callstate bool) {
	floor := incoming.Floor
	btn := incoming.Button
	elevator := elevatorID
l    
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

func (current Calls) mergeHallCalls(incoming Calls) {
	for floor := 0; floor < config.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {
			if incoming.HallCalls[floor][btn].TimeStamp > current.HallCalls[floor][btn].TimeStamp {
				current.HallCalls[floor][btn] = incoming.HallCalls[floor][btn]
			}
		}
	}
	return
}

func (localCalls Calls) mergeCabCalls(incomingCabCallsLists CabCallsList) { 
	var mergedCabCalls CabCalls{incomingCabCallsLists[0]}

	for _, cabCalls := range incomingCabCallsLists[1:] {
		for floor := 0; floor < config.NumFloors; floor++ {
			if cabCalls[floor].TimeStamp > mergedCabCalls[floor].TimeStamp {
				mergedCabCalls[floor] = cabCalls[floor]
			}
		}
	}
	
	for floor := 0; floor < config.NumFloors; floor++ {
		mergedCabCalls[floor].NeedService = mergedCabCalls[floor].NeedService || localCalls.CabCalls[floor].NeedService
		mergedCabCalls[floor].TimeStamp++
	}

	localCalls.CabCalls = mergedCabCalls
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
