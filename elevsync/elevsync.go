package elevsync

import (
	"root/elevio"
	"root/elevstate"
)

// Channel overview
// hardwareCalls: 	Sync <- HW
// finishedCalls: 	Sync <- Main
// syncedData: 		Sync -> Main

func Sync(
	hardwareCalls <-chan elevio.CallEvent,
	localStateCh <-chan elevstate.ElevState,
	finishedCalls <-chan elevio.CallEvent,
	networkReceiveMsg <-chan NetworkReceiveMsg,
	syncedData chan<- SyncedData,
	cabCallsRequest <-chan string,
	cabCallsReceive <-chan CabCallsList,
	cabCallsSend chan<- CabCalls,
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
			localCalls.addCall(incomingHardwareCall)

		case incomingFinishedCall := <-finishedCalls:
			localCalls.removeCall(incomingFinishedCall)

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
