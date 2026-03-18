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

func main() {

	idPtr := flag.String("id", "defaultID", "ID of elevator, overwrite using -id=<newId>")
	portPtr := flag.Int("port", config.HardwarePortNumber, "Port of the hardware server, overwrite using -port=<newPort>")
	flag.Parse()

	id := *idPtr
	fmt.Println(id)
	port := *portPtr

	hardwareDisconnectedC := make(chan bool, 1024)
	hardwareReconnectedC := make(chan bool, 1024)
	elevio.Init("localhost:"+strconv.Itoa(port), config.NumFloors, hardwareDisconnectedC, hardwareReconnectedC)

	selfStateToMainC := make(chan elevstate.ElevState, 1024)
	selfCallsToElevatorC := make(chan elevsync.CommonCalls, 1024)
	commonCallsToLightsC := make(chan elevsync.CommonCalls, 1024)

	hardWareCallToSyncC := make(chan elevio.CallEvent, 1024)
	completedCallToSyncC := make(chan elevio.CallEvent, 1024)
	selfStateToSyncC := make(chan elevstate.ElevState, 1024)
	syncedVariablesToMainC := make(chan elevsync.SyncedData, 1024)
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
		id,
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
		id,
		hardWareCallToSyncC,
		completedCallToSyncC,
		selfStateToSyncC,
		syncedVariablesToMainC,
		otherDataToSyncC,
		otherCabCallsRequestC,
		otherCabCallsToNetworkC,
		selfCabCallsToSyncC,
		networkRequestSelfDataC,
		selfDataToNetworkC,
		alivePeersC,
	)

	var state elevstate.ElevState
	var prevState elevstate.ElevState
	var syncedVariables elevsync.SyncedData
	var prevSyncedVariables elevsync.SyncedData

	syncTicker := time.NewTicker(time.Hour)
	// syncTicker := time.NewTicker(config.SyncTimeout)

	for {

		select {
		case state = <-selfStateToMainC:
			if state != prevState {
				selfStateToSyncC <- state
				prevState = state
			}
		case syncedVariables = <-syncedVariablesToMainC:
			if syncedVariables.Equals(prevSyncedVariables) {
				break
			}
			prevSyncedVariables = syncedVariables
			updateElevator(id, state, syncedVariables, selfCallsToElevatorC, commonCallsToLightsC)
		case <-syncTicker.C:
			updateElevator(id, state, syncedVariables, selfCallsToElevatorC, commonCallsToLightsC)
		case <-hardwareDisconnectedC:
			state.MotorStop = true
			selfStateToSyncC <- state
		}
	}
}

func updateElevator(
	id string,
	state elevstate.ElevState,
	synced elevsync.SyncedData,
	selfCallsToElevatorC chan<- elevsync.CommonCalls,
	commonCallsToLightsC chan<- elevsync.CommonCalls,
) {

	allStates := append(
		[]elevsync.OtherElevatorBool{
			{
				ID:           id,
				State:        state,
				CabCallsBool: synced.LocalCabCalls,
			},
		},
		synced.OtherElevatorBoolList...,
	)

	selfCallsToElevatorC <- elevsync.CommonCalls{
		HallCalls: sequenceassigner.AssignCalls(allStates, synced.SyncedHallCalls),
		CabCalls:  synced.LocalCabCalls,
	}

	commonCallsToLightsC <- elevsync.CommonCalls{
		HallCalls: synced.SyncedHallCalls,
		CabCalls:  synced.LocalCabCalls,
	}
}
