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

	fsmStateToMainC := make(chan elevstate.ElevState, 1024)
	confirmedCallsToElevatorC := make(chan elevsync.CommonCalls, 1024)
	callsToLightsC := make(chan elevsync.CommonCalls, 1024)

	hardWareCallToSyncC := make(chan elevio.CallEvent, 1024)
	completedCallToSyncC := make(chan elevio.CallEvent, 1024)
	localStateToSyncC := make(chan elevstate.ElevState, 1024)
	syncedVariablesToMainC := make(chan elevsync.SyncedData, 1024)
	otherDataToSyncC := make(chan elevsync.NetworkMsg, 1024)

	otherCabCallsRequestC := make(chan string, 1024)
	selfCabCallsToSyncC := make(chan []elevsync.CabCalls, 1024)
	otherCabCallsToNetworkC := make(chan elevsync.CabCalls, 1024)
	networkRequestSelfDataC := make(chan struct{}, 1024)
	selfDataToNetworkC := make(chan elevsync.NetworkMsg, 1024)
	alivePeersC := make(chan []string, 1024)

	go elevator.Elevator(fsmStateToMainC, completedCallToSyncC, confirmedCallsToElevatorC, hardwareReconnectedC)
	go lights.SetLights(callsToLightsC)

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
		localStateToSyncC,
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
	var prevSyncedVariables elevsync.SyncedData

	// For debug
	i := 0

	for {

		select {
		case state = <-fsmStateToMainC:
			if state != prevState {
				localStateToSyncC <- state
				prevState = state
			}
		case syncedVariables := <-syncedVariablesToMainC:
			if syncedVariables.Equals(prevSyncedVariables) {
				break
			}

			allStates := append(
				[]elevsync.OtherElevatorBool{
					{
						ID:           id,
						State:        state,
						CabCallsBool: syncedVariables.LocalCabCalls,
					}},
				syncedVariables.OtherElevatorBoolList...,
			)

			confirmedCallsToElevatorC <- elevsync.CommonCalls{
				HallCalls: sequenceassigner.AssignCalls(allStates, syncedVariables.SyncedHallCalls),
				CabCalls:  syncedVariables.LocalCabCalls,
			}
			callsToLightsC <- elevsync.CommonCalls{
				HallCalls: syncedVariables.SyncedHallCalls,
				CabCalls:  syncedVariables.LocalCabCalls,
			}
			prevSyncedVariables = syncedVariables

		case <-hardwareDisconnectedC:
			state.MotorStop = true
			localStateToSyncC <- state

		// Debug to monitor state and alive
		case <-time.After(3 * time.Second):
			i++
			fmt.Println("main", i, "state:", state.Floor, state.Direction, state.Behaviour)
		}
	}
}
