package elevsync

import (
	"root/elevio"
	"root/elevstate"
	"slices"
)

func Sync(id string,
	hardwareCallToSyncC <-chan elevio.CallEvent,
	completedCallToSyncC <-chan elevio.CallEvent,
	selfStateToSyncC <-chan elevstate.ElevState,
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

	var cabCallsRestored = false

	for {
		select {
		case incomingHardwareCall := <-hardwareCallToSyncC:
			localCalls.addCall(incomingHardwareCall)

		case incomingFinishedCall := <-completedCallToSyncC:
			localCalls.removeCall(incomingFinishedCall)

		case incomingLocalState := <-selfStateToSyncC:
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
				if cabCallsRestored == false {
					incomingCabCallsList := <-selfCabCallsToSyncC
<<<<<<< HEAD
					//print("Received cabc|alls")
=======
>>>>>>> ae6303bcf974f8c56bcf36b734af8962499ca805
					localCalls.mergeCabCalls(incomingCabCallsList)
					cabCallsRestored = true
				}

				NetworkMsgVersion = OtherElevatorList.updateSelfInOthersAndOthersInSelf(alivePeersList, alivePeersC, otherDataToSyncC, networkRequestSelfDataC, selfDataToNetworkC, NetworkMsgVersion, id, &localCalls, &localState)

				localCalls.mergeHallCallsForgiving(&OtherElevatorList)
				//print("Merging calls forgivingly")
			}

<<<<<<< HEAD
			copy(prevAlivePeers, alivePeersList)
=======
			prevAlivePeers = slices.Clone(alivePeersList)
>>>>>>> ae6303bcf974f8c56bcf36b734af8962499ca805

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
