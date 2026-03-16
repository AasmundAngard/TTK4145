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

	fsmStateC := make(chan elevstate.ElevState, 1024)
	callsToElevatorC := make(chan elevsync.CommonCalls, 1024)

	callsToLightsC := make(chan elevsync.CommonCalls, 1024)

	hardWareCallC := make(chan elevio.CallEvent, 1024)
	completedCallC := make(chan elevio.CallEvent, 1024)
	localStateC := make(chan elevstate.ElevState, 1024)
	syncedVariablesC := make(chan elevsync.SyncedData, 1024)
	otherDataToSyncC := make(chan elevsync.NetworkMsg, 1024)

	otherCabCallsRequestC := make(chan string, 1024)
	selfCabCallsToSyncC := make(chan []elevsync.CabCalls, 1024)
	otherCabCallsToNetworkC := make(chan elevsync.CabCalls, 1024)
	networkRequestSelfDataC := make(chan struct{}, 1024)
	selfDataToNetworkC := make(chan elevsync.NetworkMsg, 1024)
	alivePeersC := make(chan []string, 1024)

	go elevator.Elevator(fsmStateC, completedCallC, callsToElevatorC, hardwareReconnectedC)
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

	go elevio.PollButtons(hardWareCallC)
	go elevsync.Sync(
		id,
		hardWareCallC,
		completedCallC,
		localStateC,
		syncedVariablesC,
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
		case state = <-fsmStateC:
			if state != prevState {
				localStateC <- state
				prevState = state
			}
		case syncedVariables := <-syncedVariablesC:
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

			callsToElevatorC <- elevsync.CommonCalls{
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
			localStateC <- state

		// Debug to monitor state and alive
		case <-time.After(3 * time.Second):
			i++
			fmt.Println("main", i, "state:", state.Floor, state.Direction, state.Behaviour)
		}
	}
}
