package elevsync

import (
	"root/config"
	"root/elevio"
	"root/elevstate"
	"strconv"
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

func (current *Calls) mergeHallCalls(incoming Calls) {
	for floor := 0; floor < config.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {
			if incoming.HallCalls[floor][btn].TimeStamp > current.HallCalls[floor][btn].TimeStamp {
				current.HallCalls[floor][btn] = incoming.HallCalls[floor][btn]
			}
		}
	}
}

func (localCalls *Calls) mergeCabCalls(incomingCabCallsLists CabCallsList) {
	mergedCabCalls := newCabCalls()

	for _, cabCalls := range incomingCabCallsLists {
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

func (current Calls) decideCommonCalls(otherElevatorList OtherElevatorList) CallsBool {
	var confirmedCalls CallsBool
	confirmedCalls.HallCallsBool = current.HallCalls.toBool()
	confirmedCalls.CabCallsBool = current.CabCalls.toBool()

	for floor := 0; floor < config.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {
			if current.HallCalls[floor][btn].NeedService == false {
				continue
			}

			confirmed := true
			for _, otherElevator := range otherElevatorList {
				if otherElevator.Alive == false || otherElevator.State.MotorStop == true || otherElevator.State.DoorObstruction == true {
					continue
				}

				if otherElevator.Calls.HallCalls[floor][btn].NeedService == false || otherElevator.Calls.HallCalls[floor][btn].TimeStamp != current.HallCalls[floor][btn].TimeStamp {
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

func (current *Calls) update(incoming elevio.CallEvent, callstate bool) {
	floor := incoming.Floor
	btn := incoming.Button
	switch btn {
	case elevio.BT_HallUp, elevio.BT_HallDown:
		if current.HallCalls[floor][btn].NeedService != callstate {
			current.HallCalls[floor][btn].NeedService = callstate
			current.HallCalls[floor][btn].TimeStamp++
		}
	case elevio.BT_Cab:
		if current.CabCalls[floor].NeedService != callstate {
			current.CabCalls[floor].NeedService = callstate
			current.CabCalls[floor].TimeStamp++
		}
	default:
		panic("Invalid ButtonType " + strconv.Itoa(int(btn)))
	}

	// if btn == elevio.BT_HallUp || btn == elevio.BT_HallDown {
	// 	if current.HallCalls[floor][btn].NeedService != callstate {
	// 		current.HallCalls[floor][btn].NeedService = callstate
	// 		current.HallCalls[floor][btn].TimeStamp++
	// 	}

	// } else if btn == elevio.BT_Cab {
	// 	if current.CabCalls[floor].NeedService != callstate {
	// 		current.CabCalls[floor].NeedService = callstate
	// 		current.CabCalls[floor].TimeStamp++
	// 	}

	// } else {
	// 	panic("Invalid ButtonType " + strconv.Itoa(int(btn)))
	// }

}

const (
	ServicedCall   bool = false
	UnservicedCall      = true
)

type NetworkReceiveMsg struct {
	TimeStamp int64
	SenderID  string
	Calls     Calls
	State     elevstate.ElevState
}
type NetworkTransmitMsg struct {
	Calls Calls
	State elevstate.ElevState
}

type OtherElevator struct {
	ID        string
	TimeStamp int64
	Calls     Calls
	State     elevstate.ElevState
	Alive     bool
}
type OtherElevatorList []OtherElevator
type OtherElevatorBool struct {
	//ID		   	 int
	State        elevstate.ElevState
	CabCallsBool CabCallsBool
}

func (otherElevatorList OtherElevatorList) getCabCallsfromID(ID string) CabCalls {
	cabCalls := newCabCalls()

	for _, otherElevator := range otherElevatorList {
		if otherElevator.ID == ID {
			return otherElevator.Calls.CabCalls
		}
	}
	return cabCalls
}

func (OtherElevatorList *OtherElevatorList) update(incomingNetworkMsg NetworkReceiveMsg) {
	elevatorFound := false

	for i, otherElevator := range *OtherElevatorList {
		if otherElevator.ID == incomingNetworkMsg.SenderID {
			if otherElevator.TimeStamp < incomingNetworkMsg.TimeStamp {
				(*OtherElevatorList)[i].State = incomingNetworkMsg.State
				(*OtherElevatorList)[i].Calls = incomingNetworkMsg.Calls
				(*OtherElevatorList)[i].TimeStamp = incomingNetworkMsg.TimeStamp
			}
			elevatorFound = true
			break
		}
	}

	if !elevatorFound {
		*OtherElevatorList = append(*OtherElevatorList, OtherElevator{ID: incomingNetworkMsg.SenderID, TimeStamp: incomingNetworkMsg.TimeStamp, State: incomingNetworkMsg.State, Calls: incomingNetworkMsg.Calls})
	}

}

func (OtherElevatorList *OtherElevatorList) updateAliveStatus(alivePeersList []string) {

	for i, otherElevator := range *OtherElevatorList {
		alive := false
		for _, alivePeer := range alivePeersList {
			if otherElevator.ID == alivePeer {
				alive = true
				break
			}
			// Sus, should reset timestamp when dead, but not disconnect????
			(*OtherElevatorList)[i].TimeStamp = 0
		}
		(*OtherElevatorList)[i].Alive = alive
	}
}

func (OtherElevatorList OtherElevatorList) workingElevsOnlyToBool() []OtherElevatorBool {
	var OtherElevatorListBool []OtherElevatorBool

	for _, otherElevator := range OtherElevatorList {
		if otherElevator.Alive == true {
			OtherElevatorListBool = append(OtherElevatorListBool, OtherElevatorBool{State: otherElevator.State, CabCallsBool: otherElevator.Calls.CabCalls.toBool()})
		}
	}

	return OtherElevatorListBool
}

type SyncedData struct {
	LocalCabCalls         CabCallsBool
	SyncedHallCalls       HallCallsBool
	OtherElevatorListBool []OtherElevatorBool
}

func (syncedData *SyncedData) format(confirmedCalls CallsBool, OtherElevatorList OtherElevatorList) {
	syncedData.LocalCabCalls = confirmedCalls.CabCallsBool
	syncedData.SyncedHallCalls = confirmedCalls.HallCallsBool
	syncedData.OtherElevatorListBool = OtherElevatorList.workingElevsOnlyToBool()
}

func Sync(hardwareCalls <-chan elevio.CallEvent,
	localStateCh <-chan elevstate.ElevState,
	finishedCalls <-chan elevio.CallEvent,
	networkReceiveMsg <-chan NetworkReceiveMsg,
	syncedData chan<- SyncedData,
	cabCallsRequest <-chan string,
	cabCallsReceive <-chan CabCallsList,
	cabCallsSend chan<- CabCalls,
	hardwareDisconnectedC <-chan bool,
	networkRequestMsg <-chan struct{},
	networkTransmitMsgCh chan<- NetworkTransmitMsg,
	alivePeers <-chan []string) {

	var localCalls Calls
	var localState elevstate.ElevState
	var OtherElevatorList OtherElevatorList

	var confirmedCalls CallsBool
	var syncedDataToSend SyncedData

	for {
		select {
		case incomingHardwareCall := <-hardwareCalls:
			localCalls.update(incomingHardwareCall, UnservicedCall)

		case incomingFinishedCall := <-finishedCalls:
			localCalls.update(incomingFinishedCall, ServicedCall)

		case incomingLocalState := <-localStateCh:
			localState = incomingLocalState

		case incomingNetworkMsg := <-networkReceiveMsg:
			OtherElevatorList.update(incomingNetworkMsg)
			localCalls.mergeHallCalls(incomingNetworkMsg.Calls)

		case <-networkRequestMsg:
			networkTransmitMsgCh <- NetworkTransmitMsg{Calls: localCalls, State: localState}
			continue

		case alivePeersList := <-alivePeers:
			OtherElevatorList.updateAliveStatus(alivePeersList)

			//Edge case: This elevator has requested its cab calls and receives them
		case incomingCabCallsList := <-cabCallsReceive:
			localCalls.mergeCabCalls(incomingCabCallsList)

			//Edge case: Another elevator is requesting its cab calls from this elevator
		case ID := <-cabCallsRequest:
			cabCallsSend <- OtherElevatorList.getCabCallsfromID(ID)
			continue

		}

		confirmedCalls = localCalls.decideCommonCalls(OtherElevatorList)

		syncedDataToSend.format(confirmedCalls, OtherElevatorList)

		syncedData <- syncedDataToSend
	}
}
