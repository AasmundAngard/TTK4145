package main

import (
	"flag"
	"fmt"
	"root/config"
	"root/elevator"
	"root/elevio"
	"root/elevstate"
	"root/elevsync"
	"root/lights"
	"root/network"
	"root/sequenceassigner"
	"strconv"
	"time"
)

// main initializes the system and the main goroutines of the elevator.
//
// main later acts as a coordinator between the goroutines Elevator and Sync, by
// 	- Forwarding state from Elevator to Sync
//  - Notifying Sync about hardware disconnects, by modifying the elevator state
// 	- Receiving common hall calls, local cab calls and state of all peers from Sync
// 	- Forwarding assigned calls to Elevator
//
// main also controls the elevator lights through the Lights goroutine.

func main() {
	time.Sleep(config.StartupWait)

	selfIdPtr := flag.String("id", "defaultID", "ID of elevator, overwrite using -id=<newId>")
	portPtr := flag.Int("port", config.HardwarePortNumber, "Port of the hardware server, overwrite using -port=<newPort>")
	flag.Parse()

	selfId := *selfIdPtr
	fmt.Println(selfId)
	port := *portPtr

	hardwareDisconnectedC := make(chan bool, 1024)
	hardwareReconnectedC := make(chan bool, 1024)
	elevio.Init("localhost:"+strconv.Itoa(port), config.NumFloors, hardwareDisconnectedC, hardwareReconnectedC)

	selfStateToMainC := make(chan elevstate.ElevState, 1024)
	selfCallsToElevatorC := make(chan elevsync.ConfirmedCalls, 1024)
	commonCallsToLightsC := make(chan elevsync.ConfirmedCalls, 1024)

	hardWareCallToSyncC := make(chan elevio.CallEvent, 1024)
	completedCallToSyncC := make(chan elevio.CallEvent, 1024)
	selfStateToSyncC := make(chan elevstate.ElevState, 1024)
	syncedSystemStatusToMainC := make(chan elevsync.SystemStatus, 1024)

	otherDataToSyncC := make(chan elevsync.NetworkMsg, 1024)

	otherCabCallsRequestC := make(chan string, 1024)
	selfCabCallsToSyncC := make(chan []elevsync.CabCalls, 1024)
	otherCabCallsToNetworkC := make(chan elevsync.CabNetworkMsg, 1024)
	networkRequestSelfDataC := make(chan struct{}, 1024)
	selfDataToNetworkC := make(chan elevsync.NetworkMsg, 1024)
	alivePeersC := make(chan []string, 1024)

	go elevator.Elevator(selfStateToMainC, completedCallToSyncC, selfCallsToElevatorC, hardwareReconnectedC)
	go lights.Lights(commonCallsToLightsC)

	go network.Network(
		selfId,
		networkRequestSelfDataC,
		selfDataToNetworkC,
		otherDataToSyncC,
		alivePeersC,
		otherCabCallsRequestC,
		otherCabCallsToNetworkC,
		selfCabCallsToSyncC,
	)

	go elevio.PollButtons(hardWareCallToSyncC)
	go elevsync.Sync(
		selfId,
		hardWareCallToSyncC,
		completedCallToSyncC,
		selfStateToSyncC,
		syncedSystemStatusToMainC,
		otherDataToSyncC,
		otherCabCallsRequestC,
		otherCabCallsToNetworkC,
		selfCabCallsToSyncC,
		networkRequestSelfDataC,
		selfDataToNetworkC,
		alivePeersC,
	)

	reSyncTicker := time.NewTicker(config.ReSyncMain)

	var state elevstate.ElevState
	var prevState elevstate.ElevState
	var syncedSystemStatus elevsync.SystemStatus
	var prevSyncedSystemStatus elevsync.SystemStatus

	for {

		select {
		case state = <-selfStateToMainC:
			if state != prevState {
				selfStateToSyncC <- state
				prevState = state
			}

		case syncedSystemStatus = <-syncedSystemStatusToMainC:
			if syncedSystemStatus.Equals(prevSyncedSystemStatus) {
				break
			}
			updateElevator(selfId, state, syncedSystemStatus, selfCallsToElevatorC, commonCallsToLightsC)
			prevSyncedSystemStatus = syncedSystemStatus
		case <-reSyncTicker.C:
			updateElevator(selfId, state, syncedSystemStatus, selfCallsToElevatorC, commonCallsToLightsC)
		case <-hardwareDisconnectedC:
			state.MotorStop = true
			selfStateToSyncC <- state
		}
	}
}

func updateElevator(
	selfId string,
	state elevstate.ElevState,
	syncedSystemStatus elevsync.SystemStatus,
	selfCallsToElevatorC chan<- elevsync.ConfirmedCalls,
	commonCallsToLightsC chan<- elevsync.ConfirmedCalls,
) {

	allStates := append(
		[]elevsync.ConfirmedPeerElevator{
			{
				Id:       selfId,
				State:    state,
				CabCalls: syncedSystemStatus.SelfCabCalls,
			},
		},
		syncedSystemStatus.PeerElevators...,
	)

	selfCallsToElevatorC <- elevsync.ConfirmedCalls{
		HallCalls: sequenceassigner.AssignCalls(allStates, syncedSystemStatus.CommonHallCalls),
		CabCalls:  syncedSystemStatus.SelfCabCalls,
	}

	commonCallsToLightsC <- elevsync.ConfirmedCalls{
		HallCalls: syncedSystemStatus.CommonHallCalls,
		CabCalls:  syncedSystemStatus.SelfCabCalls,
	}
}
