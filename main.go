package main

import (
	"flag"
	"fmt"
	"root/config"
	"root/elevator"
	"root/elevio"
	"root/elevstate"
	"root/elevsync"
	"root/sequenceassigner"
	"strconv"
	"time"
)

func main() {

	idPtr := flag.Int("id", 0, "ID of elevator, overwrite using -id=<newId>")
	portPtr := flag.Int("fork", config.HardwarePortNumber, "Port of the hardware server, overwrite using -port=<newPort>")
	flag.Parse()

	id := *idPtr
	fmt.Println(id)
	port := *portPtr

	hardwareDisconnectedC := make(chan bool, 16)
	hardwareReconnectedC := make(chan bool, 16)
	elevio.Init("localhost:"+strconv.Itoa(port), config.NumFloors, hardwareDisconnectedC, hardwareReconnectedC)

	fsmStateC := make(chan elevstate.ElevState, 16)
	callsToElevatorC := make(chan elevsync.CallsBool, 16)

	hardWareCallsC := make(chan elevio.CallEvent, 16)
	localStateC := make(chan elevstate.ElevState, 16)
	completedCallC := make(chan elevio.CallEvent, 16)
	networkMsgC := make(chan elevsync.NetworkReceiveMsg, 16)
	syncedVariablesC := make(chan elevsync.SyncedData, 16)

	// For network -> sync
	cabCallRequestOnInitC := make(chan string, 16)
	cabCallReceiveOnInitC := make(chan elevsync.CabCallsList, 16)
	cabCallSendOnRequestC := make(chan elevsync.CabCalls, 16)

	go elevator.Elevator(fsmStateC, completedCallC, callsToElevatorC, hardwareReconnectedC)

	go elevio.PollButtons(hardWareCallsC)
	go elevsync.Sync(
		hardWareCallsC,
		localStateC,
		completedCallC,
		networkMsgC,
		syncedVariablesC,
		cabCallRequestOnInitC,
		cabCallReceiveOnInitC,
		cabCallSendOnRequestC,
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
			state.Behaviour = elevstate.Motorstop
			localStateC <- state

		// Debug to monitor state and alive
		case <-time.After(3 * time.Second):
			i++
			fmt.Println("main", i, "state:", state.Floor, state.Direction, state.Behaviour)
		}
	}
}
