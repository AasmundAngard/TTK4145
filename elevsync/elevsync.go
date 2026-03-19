package elevsync

// Package description:
//

import (
	"root/elevio"
	"root/elevstate"
)

func Sync(selfId string,
	hardwareCallToSyncC <-chan elevio.CallEvent,
	completedCallToSyncC <-chan elevio.CallEvent,
	selfStateToSyncC <-chan elevstate.ElevState,
	syncedSystemStatusToMainC chan<- SystemStatus, // syncedVariablesToMainC
	peerStatusUpdateToSyncC <-chan NetworkMsg, // peerStatusUpdateToSyncC				otherDataToSyncC
	peerRequestCabCallsToSyncC <-chan string, // peerRequestCabCallsToSyncC			otherCabCallsRequestC
	peerCabCallsToNetworkC chan<- CabNetworkMsg, // peerCabCallsToNetworkC		otherCabCallsToNetworkC
	selfCabCallsToSyncC <-chan []CabCalls, // selfCabCallsToSyncC				selfCabCallsToSyncC
	selfStatusRequestToSyncC <-chan struct{}, // selfStatusRequestToSyncC		networkRequestSelfDataC
	selfStatusToNetworkC chan<- NetworkMsg, // selfStatusToNetworkC				selfDataToNetworkC
	alivePeersToSyncC <-chan []string) { // alivePeersToSyncC							alivePeersC

	// local data
	var selfCalls Calls
	var selfState elevstate.ElevState
	var selfNetworkMsgVersion int64 = 1   // networkMsgVersion
	var hasRestoredCabCalls = false       // restoredCabCalls
	var peerElevatorList peerElevatorList //otherElevatorList

	// exported data
	var commonCalls ConfirmedCalls // confirmedCalls commonCalls
	var syncedSystemStatus SystemStatus

	for {
		select {
		case hardwareCall := <-hardwareCallToSyncC:
			selfCalls.addCall(hardwareCall)

		case completedCall := <-completedCallToSyncC:
			selfCalls.removeCall(completedCall)

		case newSelfState := <-selfStateToSyncC:
			selfState = newSelfState

		case <-selfStatusRequestToSyncC:
			outgoingNetworkMsg := NetworkMsg{Version: selfNetworkMsgVersion, SenderId: selfId, Calls: selfCalls, State: selfState}
			selfStatusToNetworkC <- outgoingNetworkMsg
			selfNetworkMsgVersion++
			continue

		case incomingNetworkMsg := <-peerStatusUpdateToSyncC:
			// Generell network-melding

			otherIsAlive, err := peerElevatorList.getAlive(incomingNetworkMsg.SenderId)
			switch {
			case otherIsAlive && err == nil:
				// Kjent ID, og heis i live
				// 		Streng versjon-merge hall calls
				// 		Oppdater lokal OtherElevator

				selfCalls.mergeHallCalls(incomingNetworkMsg.Calls)
				peerElevatorList.update(incomingNetworkMsg)
			case !otherIsAlive && err == nil:
				// Kjent ID, ikke i live
				// 	Anta at den bare reconnecta uten restart, fordi vi ikke fikk id først
				// 	Oppdater lokal other, reset
				// 	Marker som alive
				// 	Merge hall call forgiving

				// Kun broadcast cabcalls ved mottatt ID
				peerElevatorList.setAlive(incomingNetworkMsg.SenderId, true)
				peerElevatorList.resetVersion(incomingNetworkMsg.SenderId)
				peerElevatorList.updateWithoutVersionCheck(incomingNetworkMsg)
				peerElevatorList.setHallCalls(incomingNetworkMsg.SenderId, incomingNetworkMsg.Calls.HallCalls)
				selfCalls.mergeHallCallsForgiving(&peerElevatorList)
			case err != nil:
				// 	Ukjent ID, aldri sett før
				//  	Lag lokal OtherElevator
				// 		Marker som alive
				// 		Merge hall call forgiving
				peerElevatorList = append(peerElevatorList, peerElevator{
					Id:      incomingNetworkMsg.SenderId,
					Version: 0,
					Calls:   incomingNetworkMsg.Calls,
					State:   incomingNetworkMsg.State,
					Alive:   true,
				},
				)
				selfCalls.mergeHallCallsForgiving(&peerElevatorList)
			}

		case alivePeersList := <-alivePeersToSyncC:
			// Oppdatering fra Network når ny peer oppdages eller fjernes
			newPeers, lostPeers := peerElevatorList.findNewAndLostPeers(alivePeersList)

			for _, otherId := range newPeers {
				otherIsAlive, err := peerElevatorList.getAlive(otherId)
				switch {
				case otherIsAlive && err == nil:
					// Kjent elevator, men den er allerede registrert som alive
					// Ingen handling nødvendig
				case !otherIsAlive && err == nil:
					// Kjent elevator, og den er registrert som død
					peerElevatorList.setAlive(otherId, true)
					peerElevatorList.resetVersion(otherId)

				case err != nil:
					// Ukjent elevator
					// Ignorer, vent til networkMsg med status
					// Ingen handling nødvendig
				}
			}
			for _, otherId := range lostPeers {
				otherIsAlive, err := peerElevatorList.getAlive(otherId)
				switch {
				case otherIsAlive && err == nil:
					// Oppdater lokalt alive -> død
					peerElevatorList.setAlive(otherId, false)
				case !otherIsAlive && err == nil:
					// Allerede registrert som død, ignorer melding
					// Ingen handling nødvendig
				case err != nil:
					// Aldri sett elevator før, ignorer melding
					// Ingen handling nødvendig
				}
			}

		case peerId := <-peerRequestCabCallsToSyncC:
			// ID-melding fra en heis som etterspør egne cab calls
			peerIsAlive, err := peerElevatorList.getAlive(peerId) // Gir false selv om satt til alive
			otherCabCalls := peerElevatorList.getCabCallsFromId(peerId)
			peerCabCallsToNetworkC <- CabNetworkMsg{SenderId: selfId, RequesterId: peerId, CabCalls: otherCabCalls}
			switch {
			case peerIsAlive && err == nil:
				// Registrert som i live, trenger ikke sende noe
				break
			case !peerIsAlive && err == nil:
				// Registrert som død, reset lokalt versjonstall for å godta nye versjoner

				peerElevatorList.setAlive(peerId, true)
				peerElevatorList.resetVersion(peerId)

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
		commonCalls = selfCalls.decideCommonCalls(peerElevatorList, selfState)

		syncedSystemStatus.format(commonCalls, peerElevatorList)

		syncedSystemStatusToMainC <- syncedSystemStatus
	}
}
