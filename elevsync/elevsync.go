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

	var NetworkMsgVersion int64 = 1

	var prevAlivePeers []string

	var cabCallsRestored = false

	print("sync init")

	for {
		select {
		// Get cab calls in init before letting other elevators overwrite it in themselves
		case incomingCabCallsList := <-selfCabCallsToSyncC:
			localCalls.mergeCabCalls(incomingCabCallsList)
			cabCallsRestored = true

			//Stop elevator from getting stuck if all other elevators are initing
		case ID := <-otherCabCallsRequestC:
			otherCabCallsToNetworkC <- OtherElevatorList.getCabCallsfromIDAndResetVersion(ID)
			continue
		}

		if cabCallsRestored == true {
			break
		}
	}

	print("sync init finished")

	for {
		select {
		case incomingHardwareCall := <-hardwareCallToSyncC:
			localCalls.addCall(incomingHardwareCall)

		case incomingFinishedCall := <-completedCallToSyncC:
			localCalls.removeCall(incomingFinishedCall)

		case localState = <-selfStateToSyncC:
			localStatePtr := &localState
			DrainChannel(selfStateToSyncC, localStatePtr)

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

				//Handles the edge case where alivePeers case get chosen before updating from each other elevators networkmsg
				//NetworkMsgVersion = OtherElevatorList.updateSelfInOthersAndOthersInSelf(alivePeersList, alivePeersC, otherDataToSyncC, networkRequestSelfDataC, selfDataToNetworkC, NetworkMsgVersion, id, &localCalls, &localState)

				//Accepts unserviced calls from other elevators and self, even if version number lower
				localCalls.mergeHallCallsForgiving(&OtherElevatorList)
			}

			prevAlivePeers = slices.Clone(alivePeersList)

			//Edge case: Another elevator is requesting its cab calls from this elevator
		case otherId := <-otherCabCallsRequestC:
			otherCabCallsToNetworkC <- OtherElevatorList.getCabCallsfromIDAndResetVersion(otherId)
			continue
		}

		confirmedCalls = localCalls.decideCommonCalls(OtherElevatorList, localState)

		syncedData.format(confirmedCalls, OtherElevatorList)

		syncedVariablesToMainC <- syncedData
	}
}
