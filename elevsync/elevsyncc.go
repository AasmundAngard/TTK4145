package elevsync

import (
	"root/elevio"
	"root/elevstate"
)

func Sync(selfId string,
	hardwareCallToSyncC <-chan elevio.CallEvent,
	completedCallToSyncC <-chan elevio.CallEvent,
	selfStateToSyncC <-chan elevstate.ElevState,
	syncedSystemStatusToMainC chan<- SyncedData, // syncedVariablesToMainC
	peerStatusUpdateToSyncC <-chan NetworkMsg, // peerStatusUpdateToSyncC				otherDataToSyncC
	peerRequestCabCallsToSyncC <-chan string, // peerRequestCabCallsToSyncC			otherCabCallsRequestC
	peerCabCallsToNetworkC chan<- CabNetworkMsg, // peerCabCallsToNetworkC		otherCabCallsToNetworkC
	selfCabCallsToSyncC <-chan []CabCalls, // selfCabCallsToSyncC				selfCabCallsToSyncC
	selfStatusRequestToSyncC <-chan struct{}, // selfStatusRequestToSyncC		networkRequestSelfDataC
	selfStatusToNetworkC chan<- NetworkMsg, // selfStatusToNetworkC				selfDataToNetworkC
	alivePeersToSyncC <-chan []string) { // alivePeersToSyncC							alivePeersC

	var selfCalls Calls
	var selfState elevstate.ElevState
	var otherElevatorList OtherElevatorList //OtherElevatorList

	var commonCalls ConfirmedCalls
	var syncedSystemStatus SystemStatus // syncedData

	var selfNetworkMsgVersion int64 = 1

	var hasRestoredCabCalls = false

	for {
		select {
		case hardwareCall := <-hardwareCallToSyncC:
			selfCalls.addCall(hardwareCall)

		case completedCall := <-completedCallToSyncC:
			selfCalls.removeCall(completedCall)

		case newSelfState := <-selfStateToSyncC:
			selfState = newSelfState

		case <-selfStatusRequestToSyncC:
			selfStatusToNetworkC <- NetworkMsg{Version: selfNetworkMsgVersion, SenderID: selfId, Calls: selfCalls, State: selfState}
			selfNetworkMsgVersion++
			continue

		case incomingNetworkMsg := <-peerStatusUpdateToSyncC:
			// Generell network-melding

			otherIsAlive, err := otherElevatorList.getAlive(incomingNetworkMsg.SenderID)
			switch {
			case otherIsAlive && err == nil:
				// Kjent ID, og heis i live
				// 		Streng versjon-merge hall calls
				// 		Oppdater lokal OtherElevator

				selfCalls.mergeHallCalls(incomingNetworkMsg.Calls)
				otherElevatorList.update(incomingNetworkMsg)
			case !otherIsAlive && err == nil:
				// Kjent ID, ikke i live
				// 	Anta at den bare reconnecta uten restart, fordi vi ikke fikk id først
				// 	Oppdater lokal other, reset
				// 	Marker som alive
				// 	Merge hall call forgiving

				// Kun broadcast cabcalls ved mottatt ID
				otherElevatorList.setAlive(incomingNetworkMsg.SenderID, true)
				otherElevatorList.resetVersion(incomingNetworkMsg.SenderID)
				otherElevatorList.updateWithoutVersionCheck(incomingNetworkMsg)
				otherElevatorList.setHallCalls(incomingNetworkMsg.SenderID, incomingNetworkMsg.Calls.HallCalls)
				selfCalls.mergeHallCallsForgiving(&otherElevatorList)
			case err != nil:
				// 	Ukjent ID, aldri sett før
				//  	Lag lokal OtherElevator
				// 		Marker som alive
				// 		Merge hall call forgiving
				otherElevatorList = append(otherElevatorList, OtherElevator{
					ID:      incomingNetworkMsg.SenderID,
					Version: 0,
					Calls:   incomingNetworkMsg.Calls,
					State:   incomingNetworkMsg.State,
					Alive:   true,
				},
				)
				selfCalls.mergeHallCallsForgiving(&otherElevatorList)
			}

		case alivePeersList := <-alivePeersToSyncC:
			// Oppdatering fra Network når ny peer oppdages eller fjernes
			newPeers, lostPeers := OtherElevatorList.findNewAndLostPeers(alivePeersList)

			for _, otherId := range newPeers {
				otherIsAlive, err := OtherElevatorList.getAlive(otherId)
				switch {
				case otherIsAlive && err == nil:
					// Kjent elevator, men den er allerede registrert som alive
					break
				case !otherIsAlive && err == nil:
					// Kjent elevator, og den er registrert som død
					otherElevatorList.setAlive(otherId, true)
					otherElevatorList.resetVersion(otherId)

				case err != nil:
					// Ukjent elevator
					// Ignorer, vent til networkMsg med status
					break
				}
			}
			for _, otherId := range lostPeers {
				otherIsAlive, err := otherElevatorList.getAlive(otherId)
				switch {
				case otherIsAlive && err == nil:
					// Oppdater lokalt alive -> død
					otherElevatorList.setAlive(otherId, false)
				case !otherIsAlive && err == nil:
					// Allerede registrert som død, ignorer melding
					break
				case err != nil:
					// Aldri sett elevator før, ignorer melding
					break
				}
			}

		case peerId := <-peerRequestCabCallsToSyncC:
			// ID-melding fra en heis som etterspør egne cab calls
			peerIsAlive, err := otherElevatorList.getAlive(peerId) // Gir false selv om satt til alive
			otherCabCalls := otherElevatorList.getCabCallsfromID(peerId)
			peerCabCallsToNetworkC <- CabNetworkMsg{SenderID: selfId, RequesterID: peerId, CabCalls: otherCabCalls}
			switch {
			case peerIsAlive && err == nil:
				// Registrert som i live, trenger ikke sende noe
				break
			case !peerIsAlive && err == nil:
				// Registrert som død, reset lokalt versjonstall for å godta nye versjoner

				otherElevatorList.setAlive(peerId, true)
				otherElevatorList.resetVersion(peerId)

			case err != nil:
				// Aldri sett ID før, vi har ikke dens cab calls, ignorer request
				break
			}

		case incomingCabCallsList := <-selfCabCallsToSyncC:
			// Mottar egne cab calls
			if !hasRestoredCabCalls {
				selfCalls.mergeCabCalls(incomingCabCallsList)
			}
			hasRestoredCabCalls = true

		}
		commonCalls = selfCalls.decideCommonCalls(otherElevatorList, selfState)

		syncedSystemStatus.format(commonCalls, otherElevatorList)

		syncedSystemStatusToMainC <- syncedSystemStatus
	}
}
