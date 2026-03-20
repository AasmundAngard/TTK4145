package elevsync

// Package elevsync synchronizes elevator call data and peer states across the
// distributed elevator system.
//
// It collects local inputs (hardware calls, completed calls, local elevator
// state, restored cab calls, and alive-peer lists) and incoming network status
// messages from peers. It merges and version-checks hall and cab calls,
// maintains a peer list with liveness and versioning, performs forgiving merges
// on reconnects, and decides which calls are confirmed common calls.
//
// Inputs:
//   <- hardwareCallToSyncC        : local button press events
//   <- completedCallToSyncC       : reports of serviced calls
//   <- selfStateToSyncC           : local elevator state updates
//   <- peerStatusUpdateToSyncC    : incoming NetworkMsg from peers
//   <- peerRequestCabCallsToSyncC : peer requests for cab calls
//   <- selfCabCallsToSyncC        : restored local cab calls (on startup)
//   <- alivePeersToSyncC          : list of currently alive peer IDs
//   <- selfStatusRequestToSyncC   : request to publish local NetworkMsg
//
// Outputs:
//   -> syncedSystemStatusToMainC  : aggregated SystemStatus for main
//   -> selfStatusToNetworkC       : outgoing NetworkMsg (on request)
//   -> peerCabCallsToNetworkC     : responses to peers requesting cab calls
//
// Responsibilities:
//   - Merge incoming call data with local calls using versioning rules.
//   - Handle peer discovery, lost peers, and reconnects (forgiving merges).
//   - Produce confirmed common calls and a snapshot of peer states.
//   - Maintain and bump local outbound NetworkMsg version when requested.

import (
	"root/elevator"
	"root/elevio"
)

func Sync(selfId string,
	hardwareCallToSyncC <-chan elevio.CallEvent,
	completedCallToSyncC <-chan elevio.CallEvent,

	selfStateToSyncC <-chan elevator.ElevState,
	syncedSystemStatusToMainC chan<- SystemStatus,
	peerStatusUpdateToSyncC <-chan NetworkMsg,
	peerRequestCabCallsToSyncC <-chan string,
	peerCabCallsToNetworkC chan<- CabNetworkMsg,

	selfCabCallsToSyncC <-chan []CabCalls,
	selfStatusRequestToSyncC <-chan struct{},
	selfStatusToNetworkC chan<- NetworkMsg,
	alivePeersToSyncC <-chan []string) {

	// local data
	var selfCalls Calls
	var selfState elevator.ElevState
	var selfNetworkMsgVersion int64 = 1
	var hasRestoredCabCalls = false
	var peerElevatorList peerElevatorList

	// exported data
	var commonCalls ConfirmedCalls
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
			peerIsAlive, err := peerElevatorList.getAlive(incomingNetworkMsg.SenderId)

			switch {
			case peerIsAlive && err == nil:
				selfCalls.mergeHallCalls(incomingNetworkMsg.Calls)
				peerElevatorList.update(incomingNetworkMsg)

			case !peerIsAlive && err == nil:
				peerElevatorList.setAlive(incomingNetworkMsg.SenderId, true)
				peerElevatorList.resetVersion(incomingNetworkMsg.SenderId) // Discards the old version which is outdated since the peer restarted
				peerElevatorList.updateWithoutVersionCheck(incomingNetworkMsg)
				peerElevatorList.setHallCalls(incomingNetworkMsg.SenderId, incomingNetworkMsg.Calls.HallCalls)
				selfCalls.mergeHallCallsForgiving(&peerElevatorList) // Cannot guarantee data by version, so merge forgivingly

			case err != nil: // New peer discovered through incoming status update
				peerElevatorList = append(peerElevatorList, peerElevator{
					Id:      incomingNetworkMsg.SenderId,
					Version: 0,
					Calls:   incomingNetworkMsg.Calls,
					State:   incomingNetworkMsg.State,
					Alive:   true,
				},
				)
				selfCalls.mergeHallCallsForgiving(&peerElevatorList) // Cannot guarantee data by version, so merge forgivingly
			}

		case alivePeersList := <-alivePeersToSyncC:
			newPeers, lostPeers := peerElevatorList.findNewAndLostPeers(alivePeersList)

			for _, peerId := range newPeers {
				peerIsAlive, err := peerElevatorList.getAlive(peerId)

				if !peerIsAlive && err == nil {
					peerElevatorList.setAlive(peerId, true)
					peerElevatorList.resetVersion(peerId)
				}
			}

			for _, peerId := range lostPeers {
				peerIsAlive, err := peerElevatorList.getAlive(peerId)

				if peerIsAlive && err == nil {
					peerElevatorList.setAlive(peerId, false)
				}
			}

		case peerId := <-peerRequestCabCallsToSyncC: // Peer is requesting its cab calls
			peerIsAlive, err := peerElevatorList.getAlive(peerId)
			otherCabCalls := peerElevatorList.getCabCallsFromId(peerId)
			peerCabCallsToNetworkC <- CabNetworkMsg{SenderId: selfId, RequesterId: peerId, CabCalls: otherCabCalls}

			if !peerIsAlive && err == nil {
				peerElevatorList.setAlive(peerId, true)
				peerElevatorList.resetVersion(peerId)
			}

		case incomingCabCallsList := <-selfCabCallsToSyncC: // Self is receiving its cab calls
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
