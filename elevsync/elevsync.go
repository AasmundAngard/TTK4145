package elevsync

import (
	"fmt"
	"root/elevator"
	"root/elevio"
	"slices"
)

func SyncOld(id string,
	hardwareCallToSyncC <-chan elevio.CallEvent,
	completedCallToSyncC <-chan elevio.CallEvent,
	selfStateToSyncC <-chan elevator.ElevState,
	syncedVariablesToMainC chan<- SyncedData,
	otherDataToSyncC <-chan NetworkMsg,
	otherCabCallsRequestC <-chan string,
	otherCabCallsToNetworkC chan<- CabCalls,
	selfCabCallsToSyncC <-chan []CabCalls,
	networkRequestSelfDataC <-chan struct{},
	selfDataToNetworkC chan<- NetworkMsg,
	alivePeersC <-chan []string) {

	var localCalls Calls
	var localState elevator.ElevState
	var OtherElevatorList OtherElevatorList

	var confirmedCalls elevator.Calls
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
					localCalls.mergeCabCalls(incomingCabCallsList)
					cabCallsRestored = true
				}

				// =====================================
				// Denne funksjonen overskriver cab calls
				NetworkMsgVersion = OtherElevatorList.updateSelfInOthersAndOthersInSelf(alivePeersList, alivePeersC, otherDataToSyncC, networkRequestSelfDataC, selfDataToNetworkC, NetworkMsgVersion, id, &localCalls, &localState, otherCabCallsRequestC, otherCabCallsToNetworkC)

				localCalls.mergeHallCallsForgiving(&OtherElevatorList)
				//print("Merging calls forgivingly")
			}

			prevAlivePeers = slices.Clone(alivePeersList)

			//Edge case: Another elevator is requesting its cab calls from this elevator
		case ID := <-otherCabCallsRequestC:
			// En heis spør om sine cab calls
			//print("Request calls")
			fmt.Println("request calls:")
			// otherCabCallsToNetworkC <- OtherElevatorList.getCabCallsfromID(ID)
			for _, elev := range OtherElevatorList {
				for _, floor := range elev.Calls.CabCalls {
					fmt.Println(floor.NeedService)
				}
			}
			// Dette gir false for alle når requesten skjer

			cabBalls := OtherElevatorList.getCabCallsfromID(ID)
			for _, floor := range cabBalls {
				fmt.Println(floor.NeedService)
			}
			otherCabCallsToNetworkC <- cabBalls
			continue
		}
		// for i := range OtherElevatorList {
		// 	fmt.Println(i)
		// 	for _, floor := range OtherElevatorList[i].Calls.CabCalls {
		// 		fmt.Println(floor)
		// 	}
		// }
		confirmedCalls = localCalls.decideCommonCalls(OtherElevatorList, localState)

		syncedData.format(confirmedCalls, OtherElevatorList)

		syncedVariablesToMainC <- syncedData
	}
}
