package elevsync

import (
	"root/elevio"
	"root/elevstate"
)

func Sync(id string,
	hardwareCallToSyncC <-chan elevio.CallEvent,
	completedCallToSyncC <-chan elevio.CallEvent,
	selfStateToSyncC <-chan elevstate.ElevState,
	syncedVariablesToMainC chan<- SyncedData,
	otherDataToSyncC <-chan NetworkMsg,
	otherCabCallsRequestC <-chan string,
	otherCabCallsToNetworkC chan<- CabNetworkMsg,
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

	var cabCallsRestored = false

	for {
		select {
		case incomingHardwareCall := <-hardwareCallToSyncC:
			localCalls.addCall(incomingHardwareCall)

		case incomingFinishedCall := <-completedCallToSyncC:
			localCalls.removeCall(incomingFinishedCall)

		case incomingLocalState := <-selfStateToSyncC:
			localState = incomingLocalState

		case <-networkRequestSelfDataC:
			// Network ber sync om nyeste locale state
			selfDataToNetworkC <- NetworkMsg{Version: NetworkMsgVersion, SenderID: id, Calls: localCalls, State: localState}
			NetworkMsgVersion++
			continue

		case incomingNetworkMsg := <-otherDataToSyncC:
			// Generell network-melding

			otherIsAlive, err := OtherElevatorList.getAlive(incomingNetworkMsg.SenderID)
			switch {
			case otherIsAlive && err == nil:
				// Kjent ID, og heis i live
				// 		Streng versjon-merge hall calls
				// 		Oppdater lokal OtherElevator

				localCalls.mergeHallCalls(incomingNetworkMsg.Calls)
				OtherElevatorList.update(incomingNetworkMsg)
			case !otherIsAlive && err == nil:
				// Kjent ID, ikke i live
				// 	Anta at den bare reconnecta uten restart, fordi vi ikke fikk id først
				// 	Oppdater lokal other, reset
				// 	Marker som alive
				// 	Merge hall call forgiving

				// Kun broadcast cabcalls ved mottatt ID
				OtherElevatorList.setAlive(incomingNetworkMsg.SenderID, true)
				OtherElevatorList.resetVersion(incomingNetworkMsg.SenderID)
				OtherElevatorList.updateWithoutVersionCheck(incomingNetworkMsg)
				OtherElevatorList.setHallCalls(incomingNetworkMsg.SenderID, incomingNetworkMsg.Calls.HallCalls)
				localCalls.mergeHallCallsForgiving(&OtherElevatorList)
			case err != nil:
				// 	Ukjent ID, aldri sett før
				//  	Lag lokal OtherElevator
				// 		Marker som alive
				// 		Merge hall call forgiving
				OtherElevatorList = append(OtherElevatorList, OtherElevator{
					ID:      incomingNetworkMsg.SenderID,
					Version: 0,
					Calls:   incomingNetworkMsg.Calls,
					State:   incomingNetworkMsg.State,
					Alive:   true,
				},
				)
				localCalls.mergeHallCallsForgiving(&OtherElevatorList)
			}

		case alivePeersList := <-alivePeersC:
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
					OtherElevatorList.setAlive(otherId, true)
					OtherElevatorList.resetVersion(otherId)

				case err != nil:
					// Ukjent elevator
					// Ignorer, vent til networkMsg med status
					break
				}
			}
			for _, otherId := range lostPeers {
				otherIsAlive, err := OtherElevatorList.getAlive(otherId)
				switch {
				case otherIsAlive && err == nil:
					// Oppdater lokalt alive -> død
					OtherElevatorList.setAlive(otherId, false)
				case !otherIsAlive && err == nil:
					// Allerede registrert som død, ignorer melding
					break
				case err != nil:
					// Aldri sett elevator før, ignorer melding
					break
				}
			}

		case ID := <-otherCabCallsRequestC:
			// ID-melding fra en heis som etterspør egne cab calls
			peerIsAlive, err := OtherElevatorList.getAlive(ID) // Gir false selv om satt til alive
			otherCabCalls := OtherElevatorList.getCabCallsfromID(ID)
			otherCabCallsToNetworkC <- CabNetworkMsg{SenderID: id, RequesterID: ID, CabCalls: otherCabCalls}
			switch {
			case peerIsAlive && err == nil:
				// Registrert som i live, trenger ikke sende noe
				break
			case !peerIsAlive && err == nil:
				// Registrert som død, reset lokalt versjonstall for å godta nye versjoner

				OtherElevatorList.setAlive(ID, true)
				OtherElevatorList.resetVersion(ID)

			case err != nil:
				// Aldri sett ID før, vi har ikke dens cab calls, ignorer request
				break
			}

		case incomingCabCallsList := <-selfCabCallsToSyncC:
			// Mottar egne cab calls
			if !cabCallsRestored {
				localCalls.mergeCabCalls(incomingCabCallsList)
			}
			cabCallsRestored = true

		}
		confirmedCalls = localCalls.decideCommonCalls(OtherElevatorList, localState)

		syncedData.format(confirmedCalls, OtherElevatorList)

		syncedVariablesToMainC <- syncedData
	}
}
