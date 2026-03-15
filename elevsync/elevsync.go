package elevsync

import (
	"root/elevio"
	"root/elevstate"
)

func Sync(id string,
	hardwareCallC <-chan elevio.CallEvent,
	completedCallC <-chan elevio.CallEvent,
	localStateC <-chan elevstate.ElevState,
	syncedVariablesC chan<- ConfirmedData,

	otherDataToSyncC <-chan NetworkMsg,
	otherCabCallsRequestC <-chan string,
	otherCabCallsToNetworkC chan<- CabCalls,
	selfCabCallsToSyncC <-chan []CabCalls,
	networkRequestSelfDataC <-chan struct{},
	selfDataToNetworkC chan<- NetworkMsg,
	alivePeersC <-chan []string) {

	var localCalls Calls
	var localState elevstate.ElevState
	var OtherElevatorList OtherElevatorList

	var confirmedCalls CallsBool
	var confirmedData ConfirmedData

	var NetworkMsgTimestamp int64 = 0

	for {
		select {
		case incomingHardwareCall := <-hardwareCallC:
			localCalls.addCall(incomingHardwareCall)

		case incomingFinishedCall := <-completedCallC:
			localCalls.removeCall(incomingFinishedCall)

		case incomingLocalState := <-localStateC:
			localState = incomingLocalState

		case incomingNetworkMsg := <-otherDataToSyncC:
			OtherElevatorList.update(incomingNetworkMsg)
			localCalls.mergeHallCalls(incomingNetworkMsg.Calls)

		case <-networkRequestSelfDataC:
			selfDataToNetworkC <- NetworkMsg{TimeStamp: NetworkMsgTimestamp, SenderID: id, Calls: localCalls, State: localState}
			NetworkMsgTimestamp++
			continue

		case alivePeersList := <-alivePeersC:
			OtherElevatorList.updateAliveStatus(alivePeersList)

			//Edge case: This elevator has requested its cab calls and receives them
		case incomingCabCallsList := <-selfCabCallsToSyncC:
			localCalls.mergeCabCalls(incomingCabCallsList)

			//Edge case: Another elevator is requesting its cab calls from this elevator
		case ID := <-otherCabCallsRequestC:
			otherCabCallsToNetworkC <- OtherElevatorList.getCabCallsfromID(ID)
			continue
		}

		confirmedCalls = localCalls.decideCommonCalls(OtherElevatorList, localState)

		confirmedData.format(confirmedCalls, OtherElevatorList)

		syncedVariablesC <- confirmedData
	}
}
