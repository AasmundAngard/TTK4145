package main

import (
	"flag"
	"fmt"
	"root/config"
	"root/elevator"
	"root/elevio"
	"root/elevsync"
	"root/lights"
	"root/network"
	"root/sequenceassigner"
	"strconv"
	"time"
)

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

	selfStateToMainC := make(chan elevator.ElevState, 1024)
	selfCallsToElevatorC := make(chan elevator.Calls, 1024)
	commonCallsToLightsC := make(chan elevator.Calls, 1024)

	hardWareCallToSyncC := make(chan elevio.CallEvent, 1024)
	completedCallToSyncC := make(chan elevio.CallEvent, 1024)
	selfStateToSyncC := make(chan elevator.ElevState, 1024)
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
		syncedVariablesToMainC,
		otherDataToSyncC,
		otherCabCallsRequestC,
		otherCabCallsToNetworkC,
		selfCabCallsToSyncC,
		networkRequestSelfDataC,
		selfDataToNetworkC,
		alivePeersC,
	)

	var state elevator.ElevState
	var prevState elevator.ElevState
	var syncedVariables elevsync.SyncedData
	var prevSyncedVariables elevsync.SyncedData

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
			allStates := append(
				[]elevsync.OtherElevatorBool{
					{
						ID:           selfId,
						State:        state,
						CabCallsBool: syncedVariables.LocalCabCalls,
					},
				},
				syncedVariables.OtherElevatorBoolList...,
			)

			selfCallsToElevatorC <- elevator.Calls{
				HallCalls: sequenceassigner.AssignCalls(allStates, syncedVariables.SyncedHallCalls),
				CabCalls:  syncedVariables.LocalCabCalls,
			}

			commonCallsToLightsC <- elevator.Calls{
				HallCalls: syncedVariables.SyncedHallCalls,
				CabCalls:  syncedVariables.LocalCabCalls,
			}
		case <-hardwareDisconnectedC:
			state.MotorStop = true
			selfStateToSyncC <- state
		}
	}
}
