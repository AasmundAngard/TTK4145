package elevsync

import (
	"fmt"
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

	var NetworkMsgVersion int64 = 0

	// var prevAlivePeers []string

	var cabCallsRestored = false
	// i := 0

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
			//
			// Ueffektivt? Kan ikke spørre hvert 10. millisekund
			selfDataToNetworkC <- NetworkMsg{Version: NetworkMsgVersion, SenderID: id, Calls: localCalls, State: localState}
			NetworkMsgVersion++
			continue

		case incomingNetworkMsg := <-otherDataToSyncC:
			// Generell network-melding

			// Returnerer true,nil for alive, false,nil for tidligere frakoblet
			// false,error for aldri sett før
			otherIsAlive, err := OtherElevatorList.getAlive(incomingNetworkMsg.SenderID)
			switch {
			case otherIsAlive && err == nil:
				// Fant heis, og heis i live
				localCalls.mergeHallCalls(incomingNetworkMsg.Calls)
				OtherElevatorList.update(incomingNetworkMsg)
			// 		Streng versjon-merge hall calls
			// 		Oppdater lokal OtherElevator
			case !otherIsAlive && err == nil:
				// Fant heis, ikke i live
				// Anta at den bare reconnecta uten restart, fordi vi ikke fikk id først

				// Kun resett ved mottatt ID
				// otherCabCalls := OtherElevatorList.getCabCallsfromID(incomingNetworkMsg.SenderID)
				// fmt.Println("network msg")
				// fmt.Println(otherCabCalls)
				// otherCabCallsToNetworkC <- otherCabCalls // Be network broadcaste cabcalls med id
				OtherElevatorList.setAlive(incomingNetworkMsg.SenderID, true)
				OtherElevatorList.updateWithoutVersionCheck(incomingNetworkMsg)
				OtherElevatorList.setHallCalls(incomingNetworkMsg.SenderID, incomingNetworkMsg.Calls.HallCalls)
				localCalls.mergeHallCallsForgiving(&OtherElevatorList)

				// Hent cab calls, send til network
				// Oppdater lokal other
				// Marker som alive
				// Merge hall call forgiving
			case err != nil:
				// 	Ukjent ID, aldri sett før
				OtherElevatorList = append(OtherElevatorList, OtherElevator{
					ID:      incomingNetworkMsg.SenderID,
					Version: 0,
					Calls:   incomingNetworkMsg.Calls,
					State:   incomingNetworkMsg.State,
					Alive:   true,
				},
				)
				localCalls.mergeHallCallsForgiving(&OtherElevatorList)
				//  	Lag lokal OtherElevator
				// 		Marker som alive
				// 		Merge hall call forgiving
			}

			// Sjekk: dersom markert som ikke alive:
			//		Hent cab calls og begynn å broadcaste
			//		Marker som alive
			// 		Merge hall call forgiving
			//  	Oppdater lokal OtherElevator
			// Sjekk: dersom aldri sett før
			// 		Marker som alive
			// 		Merge hall call forgiving
			//  	Oppdater lokal OtherElevator
			// Ellers: (i live):
			// 		Streng versjon-merge hall calls
			// 		Oppdater lokal OtherElevator
			// 		localCalls.mergeHallCalls(incomingNetworkMsg.Calls)
			// 		OtherElevatorList.update(incomingNetworkMsg)

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
					// otherCabCalls := OtherElevatorList.getCabCallsfromID(otherId)
					// Broadcast funnet cabcalls \/
					// Kun broadcast ved mottatt ID
					// otherCabCallsToNetworkC <- otherCabCalls
					OtherElevatorList.setAlive(otherId, true)
				case err != nil:
					// Aldri sett elevator
					// Ignorer, vent til networkMsg med status fra other!
					break
				}
			}
			for _, otherId := range lostPeers {
				otherIsAlive, err := OtherElevatorList.getAlive(otherId)
				switch {
				case otherIsAlive && err == nil:
					// Oppdater lokalt
					OtherElevatorList.setAlive(otherId, false)
				case !otherIsAlive && err == nil:
					// Allerede registrert som død, ignorer melding
					break
				case err != nil:
					// Aldri sett elevator før, ignorer melding
					break
				}
			}
			// Sjekk: dersom markert som ikke alive, nå alive:
			// 		Hent dens cab calls
			// 		Send til network-loop
			// 			Netork starter en ny thread som broadcaster disse cab calls og ID en stund
			// 		Marker som alive
			// Sjekk: dersom aldri sett før:
			// 		Oppdater alive list
			// Sjekk: dersom Heis fjernet:
			// 		Oppdater OtherElevatorList med ikke alive
			// 		returnedElevators := OtherElevatorList.updateAliveStatus(alivePeersList)
			//
			// Alltid:
			// 		MergeHallCallsForgiving
		case ID := <-otherCabCallsRequestC:
			// ID-melding fra en heis som etterspør egne cab calls
			peerIsAlive, err := OtherElevatorList.getAlive(ID) // Gir false selv om satt til alive
			switch {
			case peerIsAlive && err == nil:
				// Registrert som i live, trenger ikke sende noe
				break
			case !peerIsAlive && err == nil:
				// Registrert som død, hjelp med gjennoppretting
				otherCabCalls := OtherElevatorList.getCabCallsfromID(ID)
				otherCabCallsToNetworkC <- CabNetworkMsg{SenderID: id, RequesterID: id, CabCalls: otherCabCalls}
				OtherElevatorList.setAlive(ID, true)
			case err != nil:
				// Aldri sett ID før, vi har ikke dens cab calls, ignorer request
				break
			}
			// Sjekk: dersom markert som alive:
			// 		Ignorer
			// Sjekk: dersom markert som ikke alive, men kjent:
			// 		Hent dens cab calls
			// 		Send til network-loop
			// 			Netork starter en ny thread som broadcaster disse cab calls og ID en stund
			// 		Marker som alive
			// Sjekk: dersom aldri sett før:
			// 		Ignorer, vent på state message
		case incomingCabCallsList := <-selfCabCallsToSyncC:
			fmt.Println("received own cab calls")
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
