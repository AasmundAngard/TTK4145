package main

import (
	"flag"
	"fmt"
	"root/config"
	"root/elevator"
	"root/elevio"
	"root/elevstate"
	"root/elevsync"
	"root/network"
	"root/sequenceassigner"
	"strconv"
	"time"
)

func main() {

	idPtr := flag.String("id", "id", "ID of elevator, overwrite using -id=<newId>")
	portPtr := flag.Int("port", config.HardwarePortNumber, "Port of the hardware server, overwrite using -port=<newPort>")
	flag.Parse()

	id := *idPtr
	fmt.Println(id)
	port := *portPtr

	hardwareDisconnectedC := make(chan bool, 1024)
	hardwareReconnectedC := make(chan bool, 1024)
	elevio.Init("localhost:"+strconv.Itoa(port), config.NumFloors, hardwareDisconnectedC, hardwareReconnectedC)

	fsmStateC := make(chan elevstate.ElevState, 1024)
	callsToElevatorC := make(chan elevsync.CallsBool, 1024)

	hardWareCallsC := make(chan elevio.CallEvent, 1024)
	localStateC := make(chan elevstate.ElevState, 1024)
	completedCallC := make(chan elevio.CallEvent, 1024)
	otherDataToSyncC := make(chan elevsync.NetworkMsg, 1024)
	syncedVariablesC := make(chan elevsync.ConfirmedData, 1024)

	otherCabCallsRequestC := make(chan string, 1024)
	selfCabCallsToSyncC := make(chan []elevsync.CabCalls, 1024)
	otherCabCallsToNetworkC := make(chan elevsync.CabCalls, 1024)
	networkRequestSelfDataC := make(chan struct{}, 1024)
	selfDataToNetworkC := make(chan elevsync.NetworkMsg, 1024)
	alivePeersC := make(chan []string, 1024)

	go elevator.Elevator(fsmStateC, completedCallC, callsToElevatorC, hardwareReconnectedC)

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

	go elevio.PollButtons(hardWareCallsC)
	go elevsync.Sync(
		id,
		hardWareCallsC,
		localStateC,
		completedCallC,
		networkMsgC,
		syncedVariablesC,
		cabCallRequestOnInitC,
		cabCallReceiveOnInitC,
		cabCallSendOnRequestC,
		networkRequestMsgC,
		networkTransmitMsgC,
		alivePeersC,
	)

	var state elevstate.ElevState
	var prevState elevstate.ElevState

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

			allStates := append(
				[]elevsync.OtherElevatorBool{
					{
						State:        state,
						CabCallsBool: syncedVariables.LocalCabCalls,
					}},
				syncedVariables.OtherElevatorListBool...,
			)

			callsToElevatorC <- elevsync.CallsBool{
				HallCallsBool: sequenceassigner.AssignCalls(allStates, syncedVariables.SyncedHallCalls),
				CabCallsBool:  syncedVariables.LocalCabCalls,
			}

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
