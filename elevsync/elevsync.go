package elevsync

import (
	"root/elevio"
	"root/elevstate"
)

func Sync(id string,
	hardwareCallToSyncC <-chan elevio.CallEvent,
	completedCallToSyncC <-chan elevio.CallEvent,
	localStateToSyncC <-chan elevstate.ElevState,
	syncedVariablesToMainC chan<- SyncedData,
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

	var confirmedCalls CommonCalls
	var syncedData SyncedData

	var NetworkMsgVersion int64 = 0

	var prevAlivePeers []string

	for {
		select {
		case incomingHardwareCall := <-hardwareCallToSyncC:
			localCalls.addCall(incomingHardwareCall)

		case incomingFinishedCall := <-completedCallToSyncC:
			localCalls.removeCall(incomingFinishedCall)

		case incomingLocalState := <-localStateToSyncC:
			localState = incomingLocalState

		case incomingNetworkMsg := <-otherDataToSyncC:
			OtherElevatorList.update(incomingNetworkMsg)
			localCalls.mergeHallCalls(incomingNetworkMsg.Calls)

		case <-networkRequestSelfDataC:
			selfDataToNetworkC <- NetworkMsg{Version: NetworkMsgVersion, SenderID: id, Calls: localCalls, State: localState}
			NetworkMsgVersion++
			continue

		case alivePeersList := <-alivePeersC:
			OtherElevatorList.updateAliveStatus(alivePeersList)

			if OtherElevatorList.detectReconnect(prevAlivePeers) == true {

				NetworkMsgVersion = OtherElevatorList.updateSelfInOthersAndOthersInSelf(alivePeersList, otherDataToSyncC, networkRequestSelfDataC, selfDataToNetworkC, NetworkMsgVersion, id, &localCalls, &localState)

				localCalls.mergeHallCallsForgiving(&OtherElevatorList)
				//print("Merging calls forgivingly")
			}

			copy(prevAlivePeers, alivePeersList)

			//Edge case: This elevator has requested its cab calls and receives them
		case incomingCabCallsList := <-selfCabCallsToSyncC:
			//print("Received calls")
			localCalls.mergeCabCalls(incomingCabCallsList)

			//Edge case: Another elevator is requesting its cab calls from this elevator
		case ID := <-otherCabCallsRequestC:
			//print("Request calls")
			otherCabCallsToNetworkC <- OtherElevatorList.getCabCallsfromID(ID)
			continue
		}

		confirmedCalls = localCalls.decideCommonCalls(OtherElevatorList, localState)

		syncedData.format(confirmedCalls, OtherElevatorList)

		syncedVariablesToMainC <- syncedData
	}
}
