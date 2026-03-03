package main

import (
	"flag"
	"fmt"
	"root/config"
	"root/elevio"
	"root/elevstate"
	"root/elevsync"
	"root/lights"
	"root/sequenceassigner"
	"strconv"
)

func NextState(hCalls elevsync.HallCallsBool, cCalls elevsync.CabCallsBool, state elevstate.ElevState) elevstate.ElevState {
	return elevstate.ElevState{Behaviour: elevstate.Moving, Floor: 0, Direction: elevstate.Up}
}

func main() {

	idPtr := flag.Int("id", 0, "ID of elevator, overwrite using -id=<newId>")
	portPtr := flag.Int("fork", config.HardwarePortNumber, "Port of the hardware server, overwrite using -port=<newPort>")
	flag.Parse()

	id := *idPtr
	fmt.Println(id)
	port := *portPtr

	elevio.Init("localhost:"+strconv.Itoa(port), config.NumFloors) // Dette er til den lokale heisserveren man kan kjøre (alt. hardware)

	stopButtonC := make(chan bool, 16)
	floorSensorC := make(chan int, 1)
	openDoorC := make(chan bool, 1)
	doorClosedC := make(chan bool, 1)
	doorObstructedC := make(chan bool, 1)

	hardWareCallsC := make(chan elevio.CallEvent, 16)
	localStateC := make(chan elevstate.ElevState, 16)
	completedCallC := make(chan elevio.CallEvent, 16)
	networkMsgC := make(chan elevsync.NetworkMsg, 16)
	syncedVariablesC := make(chan elevsync.SyncedData, 16)

	go elevio.PollStopButton(stopButtonC)
	go elevio.PollFloorSensor(floorSensorC)
	go elevio.PollButtons(hardWareCallsC)
	go Door(openDoorC, doorClosedC, doorObstructedC)
	go elevsync.Sync(hardWareCallsC, localStateC, completedCallC, networkMsgC, syncedVariablesC)
	// Sync should not broadcast before main says so? Maybe uninitialized tag?

	// If between floors -> floor sensor registers no floors, go down until

	var state elevstate.ElevState
	state.Behaviour = elevstate.Moving
	state.Direction = elevstate.Up
	var syncedVariables elevsync.SyncedData
	var hCalls elevsync.HallCallsBool
	var cCalls elevsync.CabCallsBool

	for {

		select {
		case newFloor := <-floorSensorC:
			state.Floor = newFloor
			elevio.SetFloorIndicator(state.Floor)
			switch state.Behaviour {
			case elevstate.Moving:
				nextState := sequenceassigner.NextState(hCalls, cCalls, state)
				switch nextState.Behaviour {
				case elevstate.DoorOpen:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorC <- true
					state.Direction = nextState.Direction
					if cCalls[state.Floor] {
						cCalls[state.Floor] = false
						completedCallC <- state.ToCabCallEvent()
					}
					state.Behaviour = elevstate.DoorOpen
				case elevstate.Moving:
					state.Direction = nextState.Direction
					elevio.SetMotorDirection(state.Direction.ToMD())
					state.Behaviour = elevstate.Moving
				case elevstate.Idle:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = elevstate.Idle
				default:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = elevstate.Idle
				}
				// switch {

				// case cCalls[state.Floor] && hCalls[state.Floor][state.Direction]:
				// 	elevio.SetMotorDirection(elevio.MD_Stop)
				// 	openDoorC <- true
				// 	state.Behaviour = DoorOpen

				// 	completedCallC <- state.toCabButtonEvent()
				// 	cCalls[state.Floor] = false

				// case cCalls[state.Floor] && hCalls[state.Floor][state.Direction.Opposite()]:
				// 	elevio.SetMotorDirection(elevio.MD_Stop)
				// 	openDoorC <- true
				// 	state.Direction = state.Direction.Opposite()
				// 	state.Behaviour = DoorOpen

				// 	completedCallC <- state.toCabButtonEvent()
				// 	cCalls[state.Floor] = false

				// case cCalls[state.Floor]:
				// 	elevio.SetMotorDirection(elevio.MD_Stop)
				// 	openDoorC <- true
				// 	state.Behaviour = DoorOpen

				// 	completedCallC <- state.toCabButtonEvent()
				// 	cCalls[state.Floor] = false

				// case hCalls[state.Floor][state.Direction]:
				// 	elevio.SetMotorDirection(elevio.MD_Stop)
				// 	openDoorC <- true
				// 	state.Behaviour = DoorOpen

				// case hCalls[state.Floor][state.Direction.Opposite()]:
				// 	elevio.SetMotorDirection(elevio.MD_Stop)
				// 	openDoorC <- true
				// 	state.Behaviour = DoorOpen
				// 	state.Direction = state.Direction.Opposite()

				// default:
				// 	if nextDirection == state.Direction.toMD() {
				// 		elevio.SetMotorDirection(state.Direction.toMD())

				// 	} else if nextDirection == state.Direction.Opposite().toMD() {
				// 		elevio.SetMotorDirection(state.Direction.Opposite().toMD())

				// 	} else {
				// 		elevio.SetMotorDirection(elevio.MD_Stop)
				// 		openDoorC <- true
				// 		state.Behaviour = DoorOpen
				// 	}

				// }
			default:
				panic("Impossible state")
			}

		case <-doorClosedC:
			switch state.Behaviour {
			case elevstate.DoorOpen:
				nextState := sequenceassigner.NextState(hCalls, cCalls, state)
				switch nextState.Behaviour {
				case elevstate.Moving:
					elevio.SetMotorDirection(state.Direction.ToMD())

					if hCalls[state.Floor][state.Direction] {
						completedCallC <- state.ToHallCallEvent()
						hCalls[state.Floor][state.Direction] = false
					}
					state.Behaviour = elevstate.Moving
				case elevstate.DoorOpen:
					openDoorC <- true
					state.Direction = state.Direction.Opposite()
					completedCallC <- state.ToHallCallEvent()
					hCalls[state.Floor][state.Direction] = false
					state.Behaviour = elevstate.DoorOpen
				case elevstate.Idle:
					state.Behaviour = elevstate.Idle
				default:
					state.Behaviour = elevstate.Idle
				}
			default:
				panic("Door closed in impossible state")
			}
		case syncedVariables = <-syncedVariablesC:
			fmt.Println("Received to main")

		drainChannel:
			for {
				select {
				case syncedVariables = <-syncedVariablesC:
				default:
					break drainChannel
				}
			}
			lights.SetLights(syncedVariables.CallsBool)

			cCalls = syncedVariables.CallsBool.CabCallsBool[0]
			fmt.Println("length cab calls: ", len(cCalls))

			var allElevStates [config.NumElevators]elevstate.ElevState
			allElevStates[0] = state
			for index, item := range syncedVariables.OtherElevators {
				allElevStates[index+1] = item.State

			}
			hCalls = sequenceassigner.AssignCalls(allElevStates, syncedVariables.CallsBool)
			fmt.Println("length hall calls: ", len(hCalls))
			for index, _ := range hCalls {
				fmt.Println(index, ":", hCalls[index][0], ",", hCalls[index][1])
			}

		case <-stopButtonC:
			elevio.SetMotorDirection(elevio.MD_Stop)
			state.Behaviour = elevstate.Idle
		}
		lights.SetLights(syncedVariables.CallsBool)
		localStateC <- state
	}

}
